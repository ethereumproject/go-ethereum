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
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/metrics"
	"github.com/ethereumproject/go-ethereum/node"
	"github.com/ethereumproject/go-ethereum/common"
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
		upgradedbCommand,
		removedbCommand,
		dumpCommand,
		monitorCommand,
		accountCommand,
		walletCommand,
		consoleCommand,
		attachCommand,
		javascriptCommand,
		{
			Action: makedag,
			Name:   "makedag",
			Usage:  "generate ethash dag (for testing)",
			Description: `
The makedag command generates an ethash DAG in /tmp/dag.

This command exists to support the system testing project.
Regular users do not need to execute it.
`,
		},
		{
			Action: gpuinfo,
			Name:   "gpuinfo",
			Usage:  "gpuinfo",
			Description: `
Prints OpenCL device info for all found GPUs.
`,
		},
		{
			Action: gpubench,
			Name:   "gpubench",
			Usage:  "benchmark GPU",
			Description: `
Runs quick benchmark on first GPU found.
`,
		},
		{
			Action: version,
			Name:   "version",
			Usage:  "print ethereum version numbers",
			Description: `
The output of this command is supposed to be machine-readable.
`,
		},
		{
			Action: initGenesis,
			Name:   "init",
			Usage:  "bootstraps and initialises a new genesis block (JSON)",
			Description: `
The init command initialises a new genesis block and definition for the network.
This is a destructive action and changes the network in which you will be
participating.
`,
		},
		{
			Action: dumpChainConfig,
			Name:   "dumpChainConfig",
			Usage:  "dump current chain configuration to JSON file [REQUIRED argument: filepath.json]",
			Description: `
The dump external configuration command writes a JSON file containing pertinent configuration data for
the configuration of a chain database. It includes genesis block data as well as chain fork settings.
`,
		},
		{
			Action: rollback,
			Name: "rollback",
			Aliases: []string{"set-head", "sethead"},
			Usage: "rollback [block index number] - set current head for blockchain",
			Description: `
Rollback set the current head block for block chain already in the database.
This is a destructive action, purging any block more recent than the index specified.
Syncing will require downloading contemporary block information from the index onwards.
`,
		},
	}

	app.Flags = []cli.Flag{
		IdentityFlag,
		UnlockedAccountFlag,
		PasswordFileFlag,
		BootnodesFlag,
		DataDirFlag,
		UseChainConfigFlag,
		KeyStoreDirFlag,
		ChainIDFlag,
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
		glog.SetToStderr(true)

		if s := ctx.String("metrics"); s != "" {
			go metrics.Collect(s)
		}

		// This should be the only place where reporting is enabled
		// because it is not intended to run while testing.
		// In addition to this check, bad block reports are sent only
		// for chains with the main network genesis block and network id 1.
		eth.EnableBadBlockReporting = true

		gasLimit := ctx.GlobalString(TargetGasLimitFlag.Name)
		if _, ok := core.TargetGasLimit.SetString(gasLimit, 0); !ok {
			log.Fatalf("malformed %s flag value %q", TargetGasLimitFlag.Name, gasLimit)
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
	node := MakeSystemNode(Version, ctx)
	startNode(ctx, node)
	node.Wait()

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
	glog.Info("Original head block of blockchain was: %v", bc.CurrentBlock().Number())

	bc.SetHead(blockIndex)
	glog.Warning("Setting current (head) block to: %v", blockIndex)

	nowCurrentState := bc.CurrentBlock().Number().Uint64()
	if nowCurrentState != blockIndex {
		glog.Warningf("ERROR: Expected rollback to set head to: %v, instead head is: %v", blockIndex, nowCurrentState)
	} else {
		glog.Info("SUCCESS: Head block set to: %v", nowCurrentState)
	}

	return nil

	// TODO: add more contextual information?
}

// initGenesis will initialise the given JSON format genesis file and writes it as
// the zero'd block (i.e. genesis) or will fail hard if it can't succeed.
func initGenesis(ctx *cli.Context) error {
	path := ctx.Args().First()
	if len(path) == 0 {
		log.Fatal("need path argument to genesis JSON file")
		return errors.New("need path argument to genesis JSON file")
	}

	chainDB, err := ethdb.NewLDBDatabase(filepath.Join(MustMakeChainDataDir(ctx), "chaindata"), 0, 0)
	defer chainDB.Close()
	if err != nil {
		log.Fatalf("could not open database: ", err)
		return err
	}

	dump, err := core.ReadGenesisFromJSONFile(path)
	if err != nil {
		log.Fatalf("%s: %s", path, err)
	}

	block, err := core.WriteGenesisBlock(chainDB, dump)
	if err != nil {
		log.Fatal("failed to write genesis block: ", err)
	}
	log.Printf("successfully wrote genesis block and/or chain rule set: %x", block.Hash())
	return nil
}


// dumpExternailChainConfig exports chain configuration based on database to JSON file
func dumpChainConfig(ctx *cli.Context) error {

	chainConfigFilePath := ctx.Args().First()
	chainConfigFilePath = filepath.Clean(chainConfigFilePath)

	if chainConfigFilePath == "" || chainConfigFilePath == "/" || chainConfigFilePath == "." {
		log.Fatalf("Given filepath to export chain configuration was blank or invalid; it was: '%v'. It cannot be blank. You typed: %v ", chainConfigFilePath, ctx.Args().First())
		return errors.New("invalid required filepath argument")
	}

	// pretty printy
	cwd, _ := os.Getwd()
	glog.V(logger.Info).Info(fmt.Sprintf("Dumping chain configuration JSON to \x1b[32m%s\x1b[39m, it may take a moment to tally genesis allocations...", common.EnsureAbsolutePath(cwd, chainConfigFilePath)))

	db := MakeChainDatabase(ctx)

	genesisDump, err := core.MakeGenesisDump(db)
	if err != nil {
		log.Fatalf("An error occurred dumping the genesis block state: %v", err)
		return err
	}
	db.Close() // required to free for MustMakeChainConfig below

	// Case: No genesis block exists in the given db.
	// This case is probably that a user is running `dumpChainConfig` without
	// having initialized any chaindata yet. If so, we should just dump defaulty values.
	//
	// FYI: here would be an inflection point for if we used a --genesis flag.
	if genesisDump == nil {
		if ctx.GlobalBool(TestNetFlag.Name) {
			glog.V(logger.Info).Info("WARNING: No genesis block found in database. Dumping the testnet default genesis.")
			genesisDump = core.TestNetGenesis
		} else {
			glog.V(logger.Info).Info("WARNING: No genesis block found in database. Dumping the mainnet default genesis.")
			genesisDump = core.DefaultGenesis
		}
	} else {
		// it'll only take a while if nondefaulty
		glog.V(logger.Info).Info("Finished building genesis state dump. Whew!")
	}

	chainConfig := MustMakeChainConfig(ctx)
	var nodes []string
	for _, node := range MakeBootstrapNodesFromContext(ctx) {
		nodes = append(nodes, node.String())
	}

	var currentConfig = &core.SufficientChainConfig{
		ID:          getChainConfigIDFromContext(ctx),
		Name:        getChainConfigNameFromContext(ctx),
		ChainConfig: chainConfig.SortForks(), // get current/contextualized chain config
		Genesis:     genesisDump,
		Bootstrap:   nodes,
	}

	if writeError := currentConfig.WriteToJSONFile(chainConfigFilePath); writeError != nil {
		log.Fatalf("An error occurred while writing chain configuration: %v", writeError)
		return writeError
	}

	glog.V(logger.Info).Info(fmt.Sprintf("Wrote chain config file to \x1b[32m%s\x1b[39m.", chainConfigFilePath))
	return nil
}


// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node) {
	// Start up the node itself
	StartNode(stack)

	// Unlock any account specifically requested
	var ethereum *eth.Ethereum
	if err := stack.Service(&ethereum); err != nil {
		log.Fatal("ethereum service not running: ", err)
	}
	accman := ethereum.AccountManager()
	passwords := MakePasswordList(ctx)

	accounts := strings.Split(ctx.GlobalString(UnlockedAccountFlag.Name), ",")
	for i, account := range accounts {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			unlockAccount(ctx, accman, trimmed, i, passwords)
		}
	}
	// Start auxiliary services if enabled
	if ctx.GlobalBool(MiningEnabledFlag.Name) {
		if err := ethereum.StartMining(ctx.GlobalInt(MinerThreadsFlag.Name), ctx.GlobalString(MiningGPUFlag.Name)); err != nil {
			log.Fatalf("Failed to start mining: ", err)
		}
	}
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

func version(c *cli.Context) error {
	fmt.Println("Geth")
	fmt.Println("Version:", Version)
	fmt.Println("Protocol Versions:", eth.ProtocolVersions)
	fmt.Println("Network Id:", c.GlobalInt(NetworkIdFlag.Name))
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())

	return nil
}
