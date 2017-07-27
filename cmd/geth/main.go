// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// geth is the official command-line client for Ethereum.
package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/urfave/cli.v1"

	"github.com/ethereumproject/ethash"
	"github.com/ethereumproject/go-ethereum/console"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/metrics"
	"github.com/ethereumproject/go-ethereum/node"
	"time"
	"github.com/ethereumproject/go-ethereum/eth/downloader"
	"math/big"
)

// Version is the application revision identifier. It can be set with the linker
// as in: go build -ldflags "-X main.Version="`git describe --tags`
var Version = "source"

func makeCLIApp() (app *cli.App) {
	app = cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Version = Version
	app.Usage = "the go-ethereum command line interface"
	app.Action = geth
	app.HideVersion = true // we have a command to print the version

	app.Commands = []cli.Command{
		importCommand,
		exportCommand,
		dumpChainConfigCommand,
		upgradedbCommand,
		removedbCommand,
		dumpCommand,
		rollbackCommand,
		monitorCommand,
		accountCommand,
		walletCommand,
		consoleCommand,
		attachCommand,
		javascriptCommand,
		statusCommand,
		{
			Action:  makedag,
			Name:    "make-dag",
			Aliases: []string{"makedag"},
			Usage:   "Generate ethash dag (for testing)",
			Description: `
The makedag command generates an ethash DAG in /tmp/dag.

This command exists to support the system testing project.
Regular users do not need to execute it.
`,
		},
		{
			Action:  gpuinfo,
			Name:    "gpu-info",
			Aliases: []string{"gpuinfo"},
			Usage:   "GPU info",
			Description: `
Prints OpenCL device info for all found GPUs.
`,
		},
		{
			Action:  gpubench,
			Name:    "gpu-bench",
			Aliases: []string{"gpubench"},
			Usage:   "Benchmark GPU",
			Description: `
Runs quick benchmark on first GPU found.
`,
		},
		{
			Action: version,
			Name:   "version",
			Usage:  "Print ethereum version numbers",
			Description: `
The output of this command is supposed to be machine-readable.
`,
		},
	}

	app.Flags = []cli.Flag{
		NodeNameFlag,
		UnlockedAccountFlag,
		PasswordFileFlag,
		AccountsIndexFlag,
		BootnodesFlag,
		DataDirFlag,
		DocRootFlag,
		KeyStoreDirFlag,
		ChainIdentityFlag,
		BlockchainVersionFlag,
		FastSyncFlag,
		CacheFlag,
		LightKDFFlag,
		JSpathFlag,
		ListenPortFlag,
		MaxPeersFlag,
		MaxPendingPeersFlag,
		EtherbaseFlag,
		GasPriceFlag,
		MinerThreadsFlag,
		MiningEnabledFlag,
		MiningGPUFlag,
		AutoDAGFlag,
		TargetGasLimitFlag,
		NATFlag,
		NatspecEnabledFlag,
		NoDiscoverFlag,
		NodeKeyFileFlag,
		NodeKeyHexFlag,
		RPCEnabledFlag,
		RPCListenAddrFlag,
		RPCPortFlag,
		RPCApiFlag,
		WSEnabledFlag,
		WSListenAddrFlag,
		WSPortFlag,
		WSApiFlag,
		WSAllowedOriginsFlag,
		IPCDisabledFlag,
		IPCApiFlag,
		IPCPathFlag,
		ExecFlag,
		PreloadJSFlag,
		WhisperEnabledFlag,
		DevModeFlag,
		TestNetFlag,
		NetworkIdFlag,
		RPCCORSDomainFlag,
		VerbosityFlag,
		VModuleFlag,
		LogDirFlag,
		LogPaceFlag,
		BacktraceAtFlag,
		MetricsFlag,
		FakePoWFlag,
		SolcPathFlag,
		GpoMinGasPriceFlag,
		GpoMaxGasPriceFlag,
		GpoFullBlockRatioFlag,
		GpobaseStepDownFlag,
		GpobaseStepUpFlag,
		GpobaseCorrectionFactorFlag,
		ExtraDataFlag,
		Unused1,
	}

	app.Before = func(ctx *cli.Context) error {

		// It's a patch.
		// Don't know why urfave/cli isn't catching the unknown command on its own.
		if ctx.Args().Present() {
			commandExists := false
			for _, cmd := range app.Commands {
				if cmd.HasName(ctx.Args().First()) {
					commandExists = true
				}
			}
			if !commandExists {
				if e := cli.ShowCommandHelp(ctx, ctx.Args().First()); e != nil {
					return e
				}
			}
		}

		runtime.GOMAXPROCS(runtime.NumCPU())

		glog.CopyStandardLogTo("INFO")

		if ctx.GlobalIsSet(aliasableName(LogDirFlag.Name, ctx)) {
			if p := ctx.GlobalString(aliasableName(LogDirFlag.Name, ctx)); p != "" {
				if e := os.MkdirAll(p, os.ModePerm); e != nil {
					return e
				}
				glog.SetLogDir(p)
			}
		} else {
			glog.SetToStderr(true) // I don't know why...
		}

		if s := ctx.String("metrics"); s != "" {
			go metrics.Collect(s)
		}

		// This should be the only place where reporting is enabled
		// because it is not intended to run while testing.
		// In addition to this check, bad block reports are sent only
		// for chains with the main network genesis block and network id 1.
		eth.EnableBadBlockReporting = true

		// (whilei): I use `log` instead of `glog` because git diff tells me:
		// > The output of this command is supposed to be machine-readable.
		gasLimit := ctx.GlobalString(aliasableName(TargetGasLimitFlag.Name, ctx))
		if _, ok := core.TargetGasLimit.SetString(gasLimit, 0); !ok {
			log.Fatalf("malformed %s flag value %q", aliasableName(TargetGasLimitFlag.Name, ctx), gasLimit)
		}

		// Set morden chain by default for dev mode.
		if ctx.GlobalBool(aliasableName(DevModeFlag.Name, ctx)) {
			if !ctx.GlobalIsSet(aliasableName(ChainIdentityFlag.Name, ctx)) {
				if e := ctx.Set(aliasableName(ChainIdentityFlag.Name, ctx), "morden"); e != nil {
						log.Fatalf("failed to set chain value: %v", e)
				}
			}
		}

		return nil
	}

	app.After = func(ctx *cli.Context) error {
		logger.Flush()
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}

	app.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Fprintf(c.App.Writer, "Invalid command: %q. Please find `geth` usage below. \n", command)
		cli.ShowAppHelp(c)
		os.Exit(3)
	}
	return app
}

func main() {
	app := makeCLIApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// geth is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func geth(ctx *cli.Context) error {

	// Force RPC enabling if --log-pace is set.
	if ctx.GlobalIsSet(LogPaceFlag.Name) && !ctx.GlobalBool(aliasableName(RPCEnabledFlag.Name, ctx)) {
		ctx.Set(aliasableName(RPCEnabledFlag.Name, ctx), "true")
	}

	n := MakeSystemNode(Version, ctx)
	ethe := startNode(ctx, n)

	if ctx.GlobalIsSet(LogPaceFlag.Name) {
		go startPacedLogging(ctx, n, ethe)
	}


	//blockchain := nodeEth.BlockChain()
	//
	//client.Send()



	//by := []byte{}
	//b := bytes.NewBuffer(by)
	//r := rpc.NewClient(b)
	//node.Attach(r, )

	//eAPI := eth.
	//netAPI := eth.PublicNetAPI{}
	//
	//
	//startTime := time.Now()
	//lastLogTime := time.Now()
	//go func() {
	//	for {
	//		select {
	//		case <-pm.quitSync:
	//			return
	//		case time.Since(lastLogTime) > time.Second*5:
	//			// "RO" -> "ReadOut"
	//			modeRO := "Import"
	//			if pm.downloader.GetMode() == downloader.FastSync {
	//				modeRO = "Sync"
	//			}
	//			_, current, highest, _, _ := pm.downloader.Progress()
	//			currentBlock := pm.blockchain.GetBlockByNumber(current)
	//			highestBlock := pm.blockchain.GetBlockByNumber(highest)
	//			bodyMean := metrics.DLBodies.RateMean()
	//			headerMean := metrics.DLHeaders.RateMean()
	//
	//			glog.V(logger.Info).Infof("%s", modeRO)
	//		}
	//	}
	//}()

	n.Wait()

	return nil
}

func startPacedLogging(ctx *cli.Context, n *node.Node, e *eth.Ethereum) {

	intervalI := 60

	if v := ctx.GlobalString(aliasableName(LogPaceFlag.Name, ctx)); v != "" {
		u := string(v[len(v)-1]) // m, s
		if !(u == "m" || u == "s") {
			glog.V(logger.Error).Infof("unknown unit suffix: %s; use 'm' (minutes) or 's' (seconds)", u)
			return
		}
		vv := string(v[:(len(v) - 1)])
		i, e := strconv.Atoi(vv)
		if e != nil {
			glog.V(logger.Error).Infof("could not parse %v argument: %v, :%v", aliasableName(LogPaceFlag.Name, ctx), v, e)
			return
		}
		if u == "m" {
			i = i * 60
		}
		intervalI = i
	}
	glog.V(logger.Error).Infof("Log-pace [STATUS] interval set: %d seconds", intervalI)

	// Need:
	//2017-02-03 16:49:00  Sync   #3124227 of #3124363 c76c…34e7   77/ 242/ 7 blk/tx/mgas sec    1/ 4/25 peers
	//2017-02-03 16:50:00  Sync   #3124247 of #3124363 75e4…8eff   51/  51/ 5 blk/tx/mgas sec    1/ 4/25 peers
	//2017-02-03 16:51:00  Sync   #3124567 of #3124363 9af3…34ae  117/ 129/11 blk/tx/mgas sec    2/ 5/25 peers
	//2017-02-03 16:52:00  Sync   #3124787 of #3124363 1e3a…8351    9/   6/ 1 blk/tx/mgas sec    1/ 7/25 peers
	//2017-02-03 16:52:05  Import #3124788             84e1…1ff4        15/ 7 tx/mgas            3/10/25 peers
	//2017-02-03 16:52:25  Import #3124789             9e45…a241         5/ 1 tx/mgas            5/12/25 peers
	//2017-02-03 16:52:45  Import #3124790             d819…f71c         0/ 0 tx/mgas           11/18/25 peers
	//
	// - Sync type (Fast/Sync/Import)
	//   |  downloader....
	// - #3124787 of #3124363 - block X of total height Y
	// - c76c…34e7 block hash
	// - 77/ 242/ 7 blk/tx/mgas sec performance for past minute, avg blocks/transactions/mgas processed per second. 3 character for block, 4 for transactions, 2 for mgas
	// -- blocks processed per second
	// -- txs processed per second
	// -- mgas processed per second
	// - 1/ 4/25 peers download from 1 peer, connected to 4, of max 25. 2 characters for each part

	// TODO: check and possibly modify existing verbsosity so pace is not interrupted... or?

	//client, err := n.Attach()
	//if err != nil {
	//	glog.Fatalln(err)
	//}
	tickerInterval := time.Second * time.Duration(int32(intervalI))
	ticker := time.NewTicker(tickerInterval)

	var lastLoggedBlockNumber uint64

	for {
		select {
			case <-ticker.C:
				peers := e.Downloader().GetPeers()
				lenpeers := peers.Len()
				//_, lennodedataidlepeers := peers.NodeDataIdlePeers()
				_, lenblockidlepeers := peers.BlockIdlePeers()
				_, lenbodyidlepeers := peers.BodyIdlePeers()
				_, lenheaderidlepeers := peers.HeaderIdlePeers()
				_, lenreceiptidlepeers := peers.ReceiptIdlePeers()
				peers.AllPeers()

				// An ugly, rough way to estimate actively connected/downloading-from peers
				activepeers := 0
				if lenblockidlepeers < lenpeers {
					activepeers += lenblockidlepeers
				}
				if lenbodyidlepeers < lenpeers {
					activepeers += lenbodyidlepeers
				}
				if lenheaderidlepeers < lenpeers {
					activepeers += lenheaderidlepeers
				}
				if lenreceiptidlepeers < lenpeers {
					activepeers += lenreceiptidlepeers
				}

				maxpeers := ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx))


				//origin, current, height, pulled, known := e.Downloader().Progress()
				_, current, height, _, _ := e.Downloader().Progress()
				mode := e.Downloader().GetMode()

				fmode := ""
				ofheight := fmt.Sprintf(" of #%7d", height)
				heightratio := float64(current) / float64(height)
				heightratio = heightratio * 100
				percentheight := fmt.Sprintf("(%4.2f", heightratio)
				percentheight += "%)"
				switch mode {
				case downloader.FullSync:
					fmode = "FullSync"
				case downloader.FastSync:
					fmode = "FastSync"
				}
				if current == height && !(current == 0 && height == 0) {
					fmode = "Import"
					ofheight = strings.Repeat(" ", 12)
					percentheight = strings.Repeat(" ", 8)
				}
				if height == 0 {
					ofheight = strings.Repeat(" ", 12)
					percentheight = strings.Repeat(" ", 8)
				}

				//t := time.Now()
				//y, m, d := t.Date()
				//hour := t.Hour()
				//minute := t.Minute()
				//second := t.Second()

				blockchain := e.BlockChain()
				//td, currentblock, genesisblock := blockchain.Status()
				_, currentblock, _:= blockchain.Status()

				numBlocksDiff := current - lastLoggedBlockNumber
				numTxsDiff := 0
				mGas := new(big.Int)

				var numBlocksDiffPerSecond uint64
				var numTxsDiffPerSecond int
				var mGasPerSecond = new(big.Int)

				if numBlocksDiff > 0 && numBlocksDiff != current {
					for i := lastLoggedBlockNumber; i <= current; i++ {
						b := blockchain.GetBlockByNumber(i)
						if b != nil {
							numTxsDiff += b.Transactions().Len()
							mGas = new(big.Int).Add(mGas, b.GasUsed())
						}
					}
				}

				// Convert to per-second stats
				// FIXME(?): Some degree of rounding will happen.
				// For example, if interval is 10s and we get 6 blocks imported in that span,
				// stats will show '0' blocks/second. Looks a little strange; but on the other hand,
				// precision costs visual space, and mostly just looks weird on when starting up sync or
				// syncing slowly.
				numBlocksDiffPerSecond = numBlocksDiff / uint64(intervalI)
				// Don't show initial current / per second val
				if lastLoggedBlockNumber == 0 {
					numBlocksDiffPerSecond = 0
				}
				numTxsDiffPerSecond = numTxsDiff / intervalI
				mGasPerSecond = new(big.Int).Div(mGas, big.NewInt(int64(intervalI)))
				mGasPerSecond = new(big.Int).Div(mGasPerSecond, big.NewInt(1000000))
				mGasPerSecondI := mGasPerSecond.Int64()

				// Update last logged current block number
				lastLoggedBlockNumber = current

				// TODO: possibly convert mGas to better unit

				cbhex := currentblock.Hex()
				cbhexstart := cbhex[2:5] // trim off '0x' prefix
				cbhexend := cbhex[(len(cbhex) - 3):]

				//datetime := fmt.Sprintf("%4d-%2d-%2d %2d:%2d:%2d", y, m, d, hour, minute, second)
				blockprogress := fmt.Sprintf("#%7d%s", current, ofheight)
				cbhexdisplay := fmt.Sprintf("%s…%s", cbhexstart, cbhexend)
				peersdisplay := fmt.Sprintf("%2d/%2d/%2d peers", activepeers, lenpeers, maxpeers)
				blocksprocesseddisplay := fmt.Sprintf("%3d/%4d/%2d blks/txs/mgas sec", numBlocksDiffPerSecond, numTxsDiffPerSecond, mGasPerSecondI)

				glog.V(logger.Error).Infof("STATUS %s %s %s %s %s %s", fmode, blockprogress, percentheight, cbhexdisplay, blocksprocesseddisplay, peersdisplay)
				//glog.V(logger.Error).Infof("STATUS %s %s %s %s", fmode, blockprogress, cbhexdisplay, peersdisplay)
			case <-sigc:
				ticker.Stop()
				return
		}
	}

	//miner := e.Miner()
	//mining := miner.Mining()
	//hashrate := miner.HashRate()

	//n.Attach()

	//sub := e.EventMux().Subscribe()
	//e := sub.Chan()

}

func status(ctx *cli.Context) error {

	shouldUseExisting := false
	datadir := MustMakeChainDataDir(ctx)
	chaindatadir := filepath.Join(datadir, "chaindata")
	if di, e := os.Stat(chaindatadir); e == nil && di.IsDir() {
		shouldUseExisting = true
	}
	// Makes sufficient configuration from JSON file or DB pending flags.
	// Delegates flag usage.
	config := mustMakeSufficientChainConfig(ctx)

	// Configure the Ethereum service
	ethConf := mustMakeEthConf(ctx, config)

	// Configure node's service container.
	name := makeNodeName(Version, ctx)
	stackConf, _ := mustMakeStackConf(ctx, name, config)

	sep := glog.Separator("-")
	printme := []struct {
		title   string
		keyVals []string
	}{
		{"Chain configuration", formatSufficientChainConfigPretty(config)},
		{"Ethereum configuration", formatEthConfigPretty(ethConf)},
		{"Node configuration", formatStackConfigPretty(stackConf)},
	}

	s := "\n"

	for _, p := range printme {
		s += withLineBreak(sep)
		// right align category title
		s += withLineBreak(strings.Repeat(" ", len(sep)-len(p.title)) + colorBlue(p.title))
		for _, v := range p.keyVals {
			s += v
		}
	}
	glog.V(logger.Info).Info(s)

	// Return here if database has not been initialized.
	if !shouldUseExisting {
		glog.V(logger.Info).Info("Geth has not been initialized; no database information available yet.")
		return nil
	}

	chaindata, cdb := MakeChain(ctx)
	defer cdb.Close()
	s = "\n"
	s += withLineBreak(sep)
	title := "Chain database status"
	s += withLineBreak(strings.Repeat(" ", len(sep)-len(title)) + colorBlue(title))
	for _, v := range formatChainDataPretty(datadir, chaindata) {
		s += v
	}
	glog.V(logger.Info).Info(s)

	return nil
}

func rollback(ctx *cli.Context) error {
	index := ctx.Args().First()
	if len(index) == 0 {
		log.Fatal("missing argument: use `rollback 12345` to specify required block number to roll back to")
		return errors.New("invalid flag usage")
	}

	blockIndex, err := strconv.ParseUint(index, 10, 64)
	if err != nil {
		glog.Fatalf("invalid argument: use `rollback 12345`, were '12345' is a required number specifying which block number to roll back to")
		return errors.New("invalid flag usage")
	}

	bc, chainDB := MakeChain(ctx)
	defer chainDB.Close()

	glog.Warning("Rolling back blockchain...")

	bc.SetHead(blockIndex)

	nowCurrentState := bc.CurrentBlock().Number().Uint64()
	if nowCurrentState != blockIndex {
		glog.Fatalf("ERROR: Wanted rollback to set head to: %v, instead current head is: %v", blockIndex, nowCurrentState)
	} else {
		glog.Infof("SUCCESS: Head block set to: %v", nowCurrentState)
	}

	return nil
}

// dumpChainConfig exports chain configuration based on context to JSON file.
// It is not compatible with --chain-config flag; it is intended to move from flags -> file,
// and not the other way around.
func dumpChainConfig(ctx *cli.Context) error {

	chainIdentity := mustMakeChainIdentity(ctx)
	if !(chainIdentitiesMain[chainIdentity] || chainIdentitiesMorden[chainIdentity]) {
		glog.Fatal("Dump config should only be used with default chain configurations (mainnet or morden).")
	}

	glog.V(logger.Info).Infof("Dumping configuration for: %v", chainIdentity)

	chainConfigFilePath := ctx.Args().First()
	chainConfigFilePath = filepath.Clean(chainConfigFilePath)

	if chainConfigFilePath == "" || chainConfigFilePath == "/" || chainConfigFilePath == "." {
		glog.Fatalf("Given filepath to export chain configuration was blank or invalid; it was: '%v'. It cannot be blank. You typed: %v ", chainConfigFilePath, ctx.Args().First())
		return errors.New("invalid required filepath argument")
	}

	fb := filepath.Dir(chainConfigFilePath)
	di, de := os.Stat(fb)
	if de != nil {
		if os.IsNotExist(de) {
			glog.V(logger.Warn).Infof("Directory path '%v' does not yet exist. Will create.", fb)
			if e := os.MkdirAll(fb, os.ModePerm); e != nil {
				glog.Fatalf("Could not create necessary directories: %v", e)
			}
			di, _ = os.Stat(fb) // update var with new dir info
		} else {
			glog.V(logger.Error).Infof("err: %v (at '%v')", de, fb)
		}
	}
	if !di.IsDir() {
		glog.Fatalf("'%v' must be a directory", fb)
	}

	// Implicitly favor Morden because it is a smaller, simpler configuration,
	// so I expect it to be used more frequently than mainnet.
	genesisDump := core.TestNetGenesis
	netId := 2
	stateConf := &core.StateConfig{StartingNonce: state.DefaultTestnetStartingNonce}
	if !chainIsMorden(ctx) {
		genesisDump = core.DefaultGenesis
		netId = eth.NetworkId
		stateConf = nil
	}

	// Note that we use default configs (not externalizable).
	chainConfig := MustMakeChainConfigFromDefaults(ctx)
	var nodes []string
	for _, node := range MakeBootstrapNodesFromContext(ctx) {
		nodes = append(nodes, node.String())
	}

	var currentConfig = &core.SufficientChainConfig{
		Identity:    chainIdentity,
		Name:        mustMakeChainConfigNameDefaulty(ctx),
		Network:     netId,
		State:       stateConf,
		Consensus:   "ethash",
		Genesis:     genesisDump,
		ChainConfig: chainConfig.SortForks(), // get current/contextualized chain config
		Bootstrap:   nodes,
	}

	if writeError := currentConfig.WriteToJSONFile(chainConfigFilePath); writeError != nil {
		glog.Fatalf("An error occurred while writing chain configuration: %v", writeError)
		return writeError
	}

	glog.V(logger.Info).Info(fmt.Sprintf("Wrote chain config file to \x1b[32m%s\x1b[39m.", chainConfigFilePath))
	return nil
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node) *eth.Ethereum {
	// Start up the node itself
	StartNode(stack)

	// Unlock any account specifically requested
	var ethereum *eth.Ethereum
	if err := stack.Service(&ethereum); err != nil {
		log.Fatal("ethereum service not running: ", err)
	}

	// Start auxiliary services if enabled
	if ctx.GlobalBool(aliasableName(MiningEnabledFlag.Name, ctx)) {
		if err := ethereum.StartMining(ctx.GlobalInt(aliasableName(MinerThreadsFlag.Name, ctx)), ctx.GlobalString(aliasableName(MiningGPUFlag.Name, ctx))); err != nil {
			log.Fatalf("Failed to start mining: %v", err)
		}
	}
	return ethereum
}

func makedag(ctx *cli.Context) error {
	args := ctx.Args()
	wrongArgs := func() {
		log.Fatal(`Usage: geth makedag <block number> <outputdir>`)
	}
	switch {
	case len(args) == 2:
		blockNum, err := strconv.ParseUint(args[0], 0, 64)
		dir := args[1]
		if err != nil {
			wrongArgs()
		} else {
			dir = filepath.Clean(dir)
			// seems to require a trailing slash
			if !strings.HasSuffix(dir, "/") {
				dir = dir + "/"
			}
			_, err = ioutil.ReadDir(dir)
			if err != nil {
				log.Fatal("Can't find dir")
			}
			fmt.Println("making DAG, this could take awhile...")
			ethash.MakeDAG(blockNum, dir)
		}
	default:
		wrongArgs()
	}
	return nil
}

func gpuinfo(ctx *cli.Context) error {
	eth.PrintOpenCLDevices()
	return nil
}

func gpubench(ctx *cli.Context) error {
	args := ctx.Args()
	wrongArgs := func() {
		log.Fatal(`Usage: geth gpubench <gpu number>`)
	}
	switch {
	case len(args) == 1:
		n, err := strconv.ParseUint(args[0], 0, 64)
		if err != nil {
			wrongArgs()
		}
		eth.GPUBench(n)
	case len(args) == 0:
		eth.GPUBench(0)
	default:
		wrongArgs()
	}
	return nil
}

func version(ctx *cli.Context) error {
	fmt.Println("Geth")
	fmt.Println("Version:", Version)
	fmt.Println("Protocol Versions:", eth.ProtocolVersions)
	fmt.Println("Network Id:", ctx.GlobalInt(aliasableName(NetworkIdFlag.Name, ctx)))
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())

	return nil
}
