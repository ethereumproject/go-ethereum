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

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ethereumproject/ethash"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/event"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/node"
	"github.com/ethereumproject/go-ethereum/pow"
	"github.com/ethereumproject/go-ethereum/rlp"
	"gopkg.in/urfave/cli.v1"
)

const (
	importBatchSize = 2500
)

// Fatalf formats a message to standard error and exits the program.
// The message is also printed to standard output if standard error
// is redirected to a different file.
func Fatalf(format string, args ...interface{}) {
	w := io.MultiWriter(os.Stdout, os.Stderr)
	if runtime.GOOS == "windows" {
		// The SameFile check below doesn't work on Windows.
		// stdout is unlikely to get redirected though, so just print there.
		w = os.Stdout
	} else {
		outf, _ := os.Stdout.Stat()
		errf, _ := os.Stderr.Stat()
		if outf != nil && errf != nil && os.SameFile(outf, errf) {
			w = os.Stderr
		}
	}
	fmt.Fprintf(w, "Fatal: "+format+"\n", args...)
	logger.Flush()
	os.Exit(1)
}

func StartNode(stack *node.Node) {
	if err := stack.Start(); err != nil {
		Fatalf("Error starting protocol stack: %v", err)
	}
	go func() {
		// sigc is a single-val channel for listening to program interrupt
		var sigc = make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigc)
		sig := <-sigc
		glog.V(logger.Warn).Warnf("Got %v, shutting down...", sig)
		glog.D(logger.Warn).Warnf("Got %v, shutting down...", sig)
		fails := make(chan error, 1)
		go func(fs chan error) {
			for {
				select {
				case e := <-fs:
					if e != nil {
						glog.V(logger.Error).Errorf("node stop failure: %v", e)
					}
				}
			}
		}(fails)

		go func(stack *node.Node) {
			defer func() {
				close(fails)
				// Ensure any write-pending I/O gets written.
				glog.Flush()
			}()
			fails <- stack.Stop()
		}(stack)

		// WTF?
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				glog.D(logger.Warn).Warnf("Already shutting down, interrupt %d more times for panic.", i-1)
			}
		}
		glog.Fatal("boom")
	}()
}

// Chain imports a blockchain.
func ImportChain(chain *core.BlockChain, fn string) error {
	// Watch for Ctrl-C while the import is running.
	// If a signal is received, the import will stop at the next batch.
	interrupt := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	defer close(interrupt)
	go func() {
		if _, ok := <-interrupt; ok {
			glog.D(logger.Warn).Warnln("caught interrupt during import, will stop at next batch")
		}
		close(stop)
	}()
	checkInterrupt := func() bool {
		select {
		case <-stop:
			return true
		default:
			return false
		}
	}

	glog.D(logger.Error).Infoln("Importing blockchain ", fn)
	fh, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer fh.Close()
	stream := rlp.NewStream(fh, 0)

	// Run actual the import.
	blocks := make(types.Blocks, importBatchSize)
	n := 0
	for batch := 0; ; batch++ {
		// Load a batch of RLP blocks.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		i := 0
		for ; i < importBatchSize; i++ {
			var b types.Block
			if err := stream.Decode(&b); err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("at block %d: %v", n, err)
			}
			// don't import first block
			if b.NumberU64() == 0 {
				i--
				continue
			}
			blocks[i] = &b
			n++
		}
		if i == 0 {
			break
		}
		// Import the batch.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		if hasAllBlocks(chain, blocks[:i]) {
			glog.D(logger.Warn).Warnf("skipping batch %d, all blocks present [%x / %x]",
				batch, blocks[0].Hash().Bytes()[:4], blocks[i-1].Hash().Bytes()[:4])
			continue
		}

		if _, err := chain.InsertChain(blocks[:i]); err != nil {
			return fmt.Errorf("invalid block %d: %v", n, err)
		}
	}
	return nil
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash()) {
			return false
		}
	}
	return true
}

func ExportChain(blockchain *core.BlockChain, fn string) error {
	glog.D(logger.Warn).Infoln("Exporting blockchain to", fn, "(this may take a while)...")
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := blockchain.Export(fh); err != nil {
		return err
	}
	glog.D(logger.Error).Infoln("Exported blockchain to ", fn)
	return nil
}

func ExportAppendChain(blockchain *core.BlockChain, fn string, first uint64, last uint64) error {
	glog.D(logger.Warn).Infoln("Exporting blockchain to ", fn)
	// TODO verify mode perms
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := blockchain.ExportN(fh, first, last); err != nil {
		return err
	}
	glog.D(logger.Error).Infoln("Exported blockchain to ", fn)
	return nil
}

func withLineBreak(s string) string {
	return s + "\n"
}

func formatStatusKeyValue(prefix string, ss ...interface{}) (s string) {

	s = ""
	// Single arg; category? ie Forks?
	if len(ss) == 1 {
		s += logger.ColorBlue(fmt.Sprintf("%v", ss[0]))
	}
	if len(ss) == 2 {
		if ss[0] == "" {
			s += fmt.Sprintf("%v", logger.ColorGreen(fmt.Sprintf("%v", ss[1])))
		} else {
			s += fmt.Sprintf("%v: %v", ss[0], logger.ColorGreen(fmt.Sprintf("%v", ss[1])))
		}
	}
	if len(ss) > 2 {
		s += fmt.Sprintf("%v:", ss[0])
		for i := 2; i < len(ss); i++ {
			s += withLineBreak(fmt.Sprintf("    %v", logger.ColorGreen(fmt.Sprintf("%v", ss[i]))))
		}
	}

	return withLineBreak(prefix + s)
}

type printable struct {
	indent int
	key    string
	val    interface{}
}

var indent string = "    "

// For use by 'status' command.
// These are one of doing it, with each key/val per line and some appropriate indentations to signal grouping/parentage.
// ... But there might be a more elegant way using columns and stuff. VeryPretty?
func formatSufficientChainConfigPretty(config *core.SufficientChainConfig) (s []string) {
	ss := []printable{}

	// Chain identifiers.
	ss = append(ss, printable{0, "Chain identity", config.Identity})
	ss = append(ss, printable{0, "Chain name", config.Name})

	// Genesis.
	ss = append(ss, printable{0, "Genesis", nil})
	ss = append(ss, printable{1, "Nonce", config.Genesis.Nonce})
	ss = append(ss, printable{1, "Coinbase", config.Genesis.Coinbase})
	ss = append(ss, printable{1, "Extra data", config.Genesis.ExtraData})
	ss = append(ss, printable{1, "Gas limit", config.Genesis.GasLimit})
	ss = append(ss, printable{1, "Difficulty", config.Genesis.Difficulty})
	ss = append(ss, printable{1, "Time", config.Genesis.Timestamp})

	lenAlloc := len(config.Genesis.Alloc)
	ss = append(ss, printable{1, "Number of allocations", lenAlloc})

	// Chain configuration.
	ss = append(ss, printable{0, "Chain Configuration", nil})
	ss = append(ss, printable{1, "Forks", nil})
	for _, v := range config.ChainConfig.Forks {
		ss = append(ss, printable{2, "Name", v.Name})
		ss = append(ss, printable{2, "Block", v.Block})
		if !v.RequiredHash.IsEmpty() {
			ss = append(ss, printable{2, "Required hash", v.RequiredHash.Hex()})
		}
		for _, fv := range v.Features {
			ss = append(ss, printable{3, fv.ID, nil})
			for k, ffv := range fv.Options {
				ss = append(ss, printable{4, k, ffv})
			}
		}
	}
	ss = append(ss, printable{1, "Bad hashes", nil})
	for _, v := range config.ChainConfig.BadHashes {
		ss = append(ss, printable{2, "Block", v.Block})
		ss = append(ss, printable{2, "Hash", v.Hash.Hex()})
	}

	for _, v := range ss {
		if v.val != nil {
			s = append(s, formatStatusKeyValue(strings.Repeat(indent, v.indent), v.key, v.val))
		} else {
			s = append(s, formatStatusKeyValue(strings.Repeat(indent, v.indent), v.key))
		}
	}

	return s
}

func formatEthConfigPretty(ethConfig *eth.Config) (s []string) {
	ss := []printable{}

	// NetworkID
	ss = append(ss, printable{0, "Network", ethConfig.NetworkId})
	// FastSync?
	ss = append(ss, printable{0, "Fast sync", ethConfig.FastSync})
	// BlockChainVersion
	ss = append(ss, printable{0, "Blockchain version", ethConfig.BlockChainVersion})
	// DatabaseCache
	ss = append(ss, printable{0, "Database cache (MB)", ethConfig.DatabaseCache})
	// DatabaseHandles
	ss = append(ss, printable{0, "Database file handles", ethConfig.DatabaseHandles})
	// NatSpec?
	ss = append(ss, printable{0, "NAT spec", ethConfig.NatSpec})
	// AutoDAG?
	ss = append(ss, printable{0, "Auto DAG", ethConfig.AutoDAG})
	// PowTest?
	ss = append(ss, printable{0, "Pow test", ethConfig.PowTest})
	// PowShared?
	ss = append(ss, printable{0, "Pow shared", ethConfig.PowShared})
	// SolcPath
	ss = append(ss, printable{0, "Solc path", ethConfig.SolcPath})

	// Account Manager
	lenAccts := len(ethConfig.AccountManager.Accounts())
	ss = append(ss, printable{0, "Account Manager", nil})
	//Number of accounts
	ss = append(ss, printable{1, "Number of accounts", lenAccts})
	// keystore not exported
	// Keystore
	//		Dir
	//		ScryptN
	//		ScryptP
	// Etherbase (if set)
	ss = append(ss, printable{0, "Etherbase", ethConfig.Etherbase.Hex()})
	// GasPrice
	ss = append(ss, printable{0, "Gas price", ethConfig.GasPrice})
	ss = append(ss, printable{0, "GPO min gas price", ethConfig.GpoMinGasPrice})
	ss = append(ss, printable{0, "GPO max gas price", ethConfig.GpoMaxGasPrice})
	// MinerThreads
	ss = append(ss, printable{0, "Miner threads", ethConfig.MinerThreads})

	for _, v := range ss {
		if v.val != nil {
			s = append(s, formatStatusKeyValue(strings.Repeat(indent, v.indent), v.key, v.val))
		} else {
			s = append(s, formatStatusKeyValue(strings.Repeat(indent, v.indent), v.key))
		}
	}
	return s
}

func formatStackConfigPretty(stackConfig *node.Config) (s []string) {

	ss := []printable{}
	// Name
	ss = append(ss, printable{0, "Name", stackConfig.Name})
	// Datadir
	ss = append(ss, printable{0, "Node data dir", stackConfig.DataDir})
	// IPCPath
	ss = append(ss, printable{0, "IPC path", stackConfig.IPCPath})
	// PrivateKey?
	if stackConfig.PrivateKey != nil {
		ss = append(ss, printable{1, "Private key", nil})
		ss = append(ss, printable{2, "Private key", nil})
		ss = append(ss, printable{2, "X", stackConfig.PrivateKey.PublicKey.X})
		ss = append(ss, printable{2, "Y", stackConfig.PrivateKey.PublicKey.Y})
	}
	// Discovery?
	ss = append(ss, printable{0, "Discovery", !stackConfig.NoDiscovery})
	// BoostrapNodes
	ss = append(ss, printable{0, "Bootstrap nodes", nil})
	for _, n := range stackConfig.BootstrapNodes {
		ss = append(ss, printable{1, "", n.String()})
	}
	// ListenAddrg
	sla := stackConfig.ListenAddr
	if sla == ":0" {
		sla += " (OS will pick)"
	}
	ss = append(ss, printable{0, "Listen address", sla})
	// NAT
	ss = append(ss, printable{0, "NAT", stackConfig.NAT.String()})
	// MaxPeers
	ss = append(ss, printable{0, "Max peers", stackConfig.MaxPeers})
	// MaxPendingPeers
	ss = append(ss, printable{0, "Max pending peers", stackConfig.MaxPendingPeers})
	// HTTP
	ss = append(ss, printable{0, "HTTP", nil})
	// HTTPHost
	ss = append(ss, printable{1, "host", stackConfig.HTTPHost})
	// HTTPPort
	ss = append(ss, printable{1, "port", stackConfig.HTTPPort})
	// HTTPCors
	ss = append(ss, printable{1, "CORS", stackConfig.HTTPCors})
	// HTTPModules[]
	ss = append(ss, printable{1, "modules", stackConfig.HTTPModules})
	// Endpoint()
	ss = append(ss, printable{1, "endpoint", stackConfig.HTTPEndpoint()})
	// WS
	ss = append(ss, printable{0, "WS", nil})
	// WSHost
	ss = append(ss, printable{1, "host", stackConfig.WSHost})
	// WSPort
	ss = append(ss, printable{1, "port", stackConfig.WSPort})
	// WSOrigins
	ss = append(ss, printable{1, "origins", stackConfig.WSOrigins})
	// WSModules[]
	ss = append(ss, printable{1, "modules", stackConfig.WSModules})
	// Endpoint()
	ss = append(ss, printable{1, "endpoint", stackConfig.WSEndpoint()})

	for _, v := range ss {
		if v.val != nil {
			s = append(s, formatStatusKeyValue(strings.Repeat(indent, v.indent), v.key, v.val))
		} else {
			s = append(s, formatStatusKeyValue(strings.Repeat(indent, v.indent), v.key))
		}
	}
	return s
}

func formatBlockPretty(b *types.Block) (ss []printable) {
	bh := b.Header()
	ss = append(ss, printable{1, "Number", bh.Number})
	ss = append(ss, printable{1, "Hash (hex)", bh.Hash().Hex()})
	ss = append(ss, printable{1, "Parent hash (hex)", bh.ParentHash.Hex()})
	ss = append(ss, printable{1, "Nonce (uint64)", bh.Nonce.Uint64()})
	ss = append(ss, printable{1, "Time", fmt.Sprintf("%v (%v)", bh.Time, time.Unix(int64(bh.Time.Uint64()), 0))})
	ss = append(ss, printable{1, "Difficulty", bh.Difficulty})
	ss = append(ss, printable{1, "Coinbase", bh.Coinbase.Hex()})
	ss = append(ss, printable{1, "Tx hash (hex)", bh.TxHash.Hex()})
	ss = append(ss, printable{1, "Bloom (string)", string(bh.Bloom.Bytes())})
	ss = append(ss, printable{1, "Gas limit", bh.GasLimit})
	ss = append(ss, printable{1, "Gas used", bh.GasUsed})
	ss = append(ss, printable{1, "Extra data (bytes)", bh.Extra})
	return ss
}

func formatChainDataPretty(datadir string, chain *core.BlockChain) (s []string) {
	ss := []printable{}

	ss = append(ss, printable{0, "Chain data dir", datadir})

	ss = append(ss, printable{0, "Genesis", nil})
	ss = append(ss, formatBlockPretty(chain.Genesis())...)

	currentBlock := chain.CurrentBlock()
	ss = append(ss, printable{0, "Current block", nil})
	ss = append(ss, formatBlockPretty(currentBlock)...)

	currentFastBlock := chain.CurrentFastBlock()
	ss = append(ss, printable{0, "Current fast block", nil})
	ss = append(ss, formatBlockPretty(currentFastBlock)...)

	for _, v := range ss {
		if v.val != nil {
			s = append(s, formatStatusKeyValue(strings.Repeat(indent, v.indent), v.key, v.val))
		} else {
			s = append(s, formatStatusKeyValue(strings.Repeat(indent, v.indent), v.key))
		}
	}
	return s
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
		s += withLineBreak(strings.Repeat(" ", len(sep)-len(p.title)) + logger.ColorBlue(p.title))
		for _, v := range p.keyVals {
			s += v
		}
	}
	glog.D(logger.Warn).Infoln(s)

	// Return here if database has not been initialized.
	if !shouldUseExisting {
		glog.D(logger.Warn).Warnln("Geth has not been initialized; no database information available yet.")
		return nil
	}

	chaindata, cdb := MakeChain(ctx)
	defer cdb.Close()
	s = "\n"
	s += withLineBreak(sep)
	title := "Chain database status"
	s += withLineBreak(strings.Repeat(" ", len(sep)-len(title)) + logger.ColorBlue(title))
	for _, v := range formatChainDataPretty(datadir, chaindata) {
		s += v
	}
	glog.D(logger.Warn).Infoln(s)

	return nil
}

func rollback(ctx *cli.Context) error {
	index := ctx.Args().First()
	if len(index) == 0 {
		glog.Fatal("missing argument: use `rollback 12345` to specify required block number to roll back to")
		return errors.New("invalid flag usage")
	}

	blockIndex, err := strconv.ParseUint(index, 10, 64)
	if err != nil {
		glog.Fatalf("invalid argument: use `rollback 12345`, were '12345' is a required number specifying which block number to roll back to")
		return errors.New("invalid flag usage")
	}

	bc, chainDB := MakeChain(ctx)
	defer chainDB.Close()

	glog.D(logger.Warn).Infoln("Rolling back blockchain...")

	if err := bc.SetHead(blockIndex); err != nil {
		glog.D(logger.Error).Errorf("error setting head: %v", err)
	}

	// Check if *neither* block nor fastblock numbers match desired head number
	nowCurrentHead := bc.CurrentBlock().Number().Uint64()
	nowCurrentFastHead := bc.CurrentFastBlock().Number().Uint64()
	if nowCurrentHead != blockIndex && nowCurrentFastHead != blockIndex {
		glog.Fatalf("ERROR: Wanted rollback to set head to: %v, instead current head is: %v", blockIndex, nowCurrentHead)
	}
	glog.D(logger.Error).Infof("Success. Head block set to: %v", nowCurrentHead)
	return nil
}

// dumpChainConfig exports chain configuration based on context to JSON file.
// It is not compatible with --chain flag; it is intended to move from default configs -> file,
// and not the other way around.
func dumpChainConfig(ctx *cli.Context) error {

	chainIdentity := mustMakeChainIdentity(ctx)
	if !(chainIdentitiesMain[chainIdentity] || chainIdentitiesMorden[chainIdentity]) {
		glog.Fatal("Dump config should only be used with default chain configurations (mainnet or morden).")
	}

	glog.D(logger.Warn).Infof("Dumping configuration for: %v", chainIdentity)

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
			glog.V(logger.Error).Errorf("err: %v (at '%v')", de, fb)
		}
	}
	if !di.IsDir() {
		glog.Fatalf("'%v' must be a directory", fb)
	}

	// Implicitly favor Morden because it is a smaller, simpler configuration,
	// so I expect it to be used more frequently than mainnet.
	genesisDump := core.DefaultConfigMorden.Genesis
	netId := 2
	stateConf := &core.StateConfig{StartingNonce: state.DefaultTestnetStartingNonce}
	if !chainIsMorden(ctx) {
		genesisDump = core.DefaultConfigMainnet.Genesis
		netId = eth.NetworkId
		stateConf = nil
	}

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

	glog.D(logger.Error).Infoln(fmt.Sprintf("Wrote chain config file to \x1b[32m%s\x1b[39m.", chainConfigFilePath))
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
		glog.Fatal("ethereum service not running: ", err)
	}

	// Start auxiliary services if enabled
	if ctx.GlobalBool(aliasableName(MiningEnabledFlag.Name, ctx)) {
		if err := ethereum.StartMining(ctx.GlobalInt(aliasableName(MinerThreadsFlag.Name, ctx)), ctx.GlobalString(aliasableName(MiningGPUFlag.Name, ctx))); err != nil {
			glog.Fatalf("Failed to start mining: %v", err)
		}
	}
	return ethereum
}

func makedag(ctx *cli.Context) error {
	args := ctx.Args()
	wrongArgs := func() {
		glog.Fatal(`Usage: geth makedag <block number> <outputdir>`)
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
				glog.Fatal("Can't find dir")
			}
			glog.V(logger.Info).Infoln("making DAG, this could take awhile...")
			glog.D(logger.Warn).Infoln("making DAG, this could take awhile...")
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
		glog.Fatal(`Usage: geth gpubench <gpu number>`)
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

func stringInSlice(s string, sl []string) bool {
	for _, l := range sl {
		if l == s {
			return true
		}
	}
	return false
}

// makeMLogDocumentation creates markdown documentation text for, eg. wiki
// That's why it uses 'fmt' instead of glog or log; output with prefixes (glog and log)
// will break markdown.
func makeMLogDocumentation(ctx *cli.Context) error {
	wantComponents := ctx.Args()

	// If no components specified, print all.
	if len(wantComponents) == 0 {
		for k := range logger.MLogRegistryAvailable {
			wantComponents = append(wantComponents, string(k))
		}
	}

	// Should throw an error if any unavailable components were specified.
	cs := []string{}
	for c := range logger.MLogRegistryAvailable {
		cs = append(cs, string(c))
	}
	for _, c := range wantComponents {
		if !stringInSlice(c, cs) {
			glog.Fatalf("Specified component does not exist: %s\n ? Available components: %v", c, cs)
		}
	}

	// "table of contents"
	for cmp, lines := range logger.MLogRegistryAvailable {
		if !stringInSlice(string(cmp), wantComponents) {
			continue
		}
		fmt.Printf("\n[%s]\n\n", cmp)
		for _, line := range lines {
			// print anchor links ul list items
			if ctx.Bool("md") {
				fmt.Printf("- [%s](#%s)\n", line.EventName(), strings.Replace(line.EventName(), ".", "-", -1))
			} else {
				fmt.Printf("- %s\n", line.EventName())
			}
		}
	}

	// Only print per-line markdown documentation if -md flag given.
	if ctx.Bool("md") {
		fmt.Println("\n----\n") // hr

		// each LINE
		for cmp, lines := range logger.MLogRegistryAvailable {
			if !stringInSlice(string(cmp), wantComponents) {
				continue
			}
			for _, line := range lines {
				fmt.Println(line.FormatDocumentation(cmp))
				fmt.Println("----")
			}
		}
	}

	fmt.Println()
	return nil
}

func recoverChaindata(ctx *cli.Context) error {

	// Congruent to MakeChain(), but uses special NewBlockChainDryrun. Avoids a one-off function in flags.go.
	var err error
	sconf := mustMakeSufficientChainConfig(ctx)
	bcdb := MakeChainDatabase(ctx)
	defer bcdb.Close()

	pow := pow.PoW(core.FakePow{})
	if !ctx.GlobalBool(aliasableName(FakePoWFlag.Name, ctx)) {
		pow = ethash.New()
	} else {
		glog.V(logger.Warn).Info("Consensus: fake")
	}

	bc, err := core.NewBlockChainDryrun(bcdb, sconf.ChainConfig, pow, new(event.TypeMux))
	if err != nil {
		glog.Fatal("Could not start chain manager: ", err)
	}

	if blockchainLoadError := bc.LoadLastState(true); blockchainLoadError != nil {
		glog.V(logger.Error).Errorf("Error while loading blockchain: %v", blockchainLoadError)
		// but do not return
	}

	header := bc.CurrentHeader()
	currentBlock := bc.CurrentBlock()
	currentFastBlock := bc.CurrentFastBlock()

	glog.D(logger.Error).Infoln("Current status (before recovery attempt):")
	if header != nil {
		glog.D(logger.Error).Infof("Last header: #%d\n", header.Number.Uint64())
		if currentBlock != nil {
			glog.D(logger.Error).Infof("Last block: #%d\n", currentBlock.Number())
		} else {
			glog.D(logger.Error).Errorf("! Last block: nil")
		}
		if currentFastBlock != nil {
			glog.D(logger.Error).Infof("Last fast block: #%d\n", currentFastBlock.Number())
		} else {
			glog.D(logger.Error).Errorln("! Last fast block: nil")
		}
	} else {
		glog.D(logger.Error).Errorln("! Last header: nil")
	}

	glog.D(logger.Error).Infoln(glog.Separator("-"))

	glog.D(logger.Error).Infoln("Checking db validity and recoverable data...")
	checkpoint := bc.Recovery(1, 2048)
	glog.D(logger.Error).Infof("Found last recoverable checkpoint=#%d\n", checkpoint)

	glog.D(logger.Error).Infoln(glog.Separator("-"))

	glog.D(logger.Error).Infoln("Setting blockchain db head to last safe checkpoint...")
	if setHeadErr := bc.SetHead(checkpoint); setHeadErr != nil {
		glog.D(logger.Error).Errorf("Error setting head: %v\n", setHeadErr)
		return setHeadErr
	}
	return nil
}

// https://gist.github.com/r0l1/3dcbb0c8f6cfe9c66ab8008f55f8f28b
// askForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user.
//
// Use 'error' verbosity for logging since this is user-critical and required feedback.
func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		glog.D(logger.Error).Warnf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			glog.Fatalln(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		} else {
			glog.D(logger.Error).Warnln(glog.Separator("*"))
			glog.D(logger.Error).Errorln("* INVALID RESPONSE: Please respond with [y|yes] or [n|no], or use CTRL-C to abort.")
			glog.D(logger.Error).Warnln(glog.Separator("*"))
		}
	}
}

// resetChaindata removes (rm -rf's) the /chaindata directory, ensuring
// eradication of any corrupted chain data.
func resetChaindata(ctx *cli.Context) error {
	dir := MustMakeChainDataDir(ctx)
	dir = filepath.Join(dir, "chaindata")
	prompt := fmt.Sprintf("\n\nThis will remove the directory='%s' and all of it's contents.\n** Are you sure you want to remove ALL chain data?", dir)
	c := askForConfirmation(prompt)
	if c {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		glog.D(logger.Error).Infof("Successfully removed chaindata directory: '%s'\n", dir)
	} else {
		glog.D(logger.Error).Infoln("Leaving chaindata untouched. As you were.")
	}
	return nil
}
