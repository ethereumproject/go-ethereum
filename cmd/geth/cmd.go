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
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"

	"strings"

	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/node"
	"github.com/ethereumproject/go-ethereum/rlp"
	"time"
	"syscall"
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
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigc)
		sig := <-sigc
		glog.Infof("Got %v, shutting down...", sig)

		fails := make(chan error, 1)
		go func (fs chan error) {
			for {
				select {
				case e := <-fs:
					if e != nil {
						glog.V(logger.Error).Infof("node stop failure: %v", e)
					}
				}
			}
		}(fails)
		go func(stack *node.Node) {
			defer close(fails)
			fails <- stack.Stop()
		}(stack)

		// WTF?
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				glog.Infof("Already shutting down, interrupt %d more times for panic.", i-1)
			}
		}
		glog.Fatal("boom")
	}()
}

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
			glog.Info("caught interrupt during import, will stop at next batch")
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

	glog.Infoln("Importing blockchain ", fn)
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
			glog.Infof("skipping batch %d, all blocks present [%x / %x]",
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
	glog.Infoln("Exporting blockchain to ", fn)
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := blockchain.Export(fh); err != nil {
		return err
	}
	glog.Infoln("Exported blockchain to ", fn)
	return nil
}

func ExportAppendChain(blockchain *core.BlockChain, fn string, first uint64, last uint64) error {
	glog.Infoln("Exporting blockchain to ", fn)
	// TODO verify mode perms
	fh, err := os.OpenFile(fn, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()
	if err := blockchain.ExportN(fh, first, last); err != nil {
		return err
	}
	glog.Infoln("Exported blockchain to ", fn)
	return nil
}

func withLineBreak(s string) string {
	return s + "\n"
}

func colorGreen(s interface{}) string {
	return fmt.Sprintf("\x1b[32m%v\x1b[39m", s)
}

func colorBlue(s interface{}) string {
	return fmt.Sprintf("\x1b[36m%v\x1b[39m", s)
}

func formatStatusKeyValue(prefix string, ss ...interface{}) (s string) {

	s = ""
	// Single arg; category? ie Forks?
	if len(ss) == 1 {
		s += colorBlue(ss[0])
	}
	if len(ss) == 2 {
		if ss[0] == "" {
			s += fmt.Sprintf("%v", colorGreen(ss[1]))
		} else {
			s += fmt.Sprintf("%v: %v", ss[0], colorGreen(ss[1]))
		}
	}
	if len(ss) > 2 {
		s += fmt.Sprintf("%v:", ss[0])
		for i := 2; i < len(ss); i++ {
			s += withLineBreak(fmt.Sprintf("    %v", colorGreen(ss[i])))
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
