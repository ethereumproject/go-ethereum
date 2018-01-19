package main

import (
	"gopkg.in/urfave/cli.v1"
	"strconv"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/core/types"
	"path/filepath"
	"io/ioutil"
	"os"
	"github.com/ethereumproject/go-ethereum/logger"
	"time"
)

func buildAddrTxIndexCmd(ctx *cli.Context) error {
	startIndex := uint64(ctx.Int("start"))
	var stopIndex uint64

	// Use persistent placeholder in case start not spec'd
	placeholderFilename := filepath.Join(MustMakeChainDataDir(ctx), "index.at")
	if !ctx.IsSet("start") {
		bs, err := ioutil.ReadFile(placeholderFilename)
		if err == nil { // ignore errors for now
			startIndex, _ = strconv.ParseUint(string(bs), 10, 64)
		}
	}

	bc, chainDB := MakeChain(ctx)
	if bc == nil || chainDB == nil {
		glog.Fatal("bc or cdb is nil")
	}
	defer chainDB.Close()

	stopIndex = uint64(ctx.Int("stop"))
	if stopIndex == 0 {
		stopIndex = bc.CurrentHeader().Number.Uint64()
	}

	if stopIndex < startIndex {
		glog.Fatal("start must be prior to (smaller than) or equal to stop, got start=", startIndex, "stop=", stopIndex)
	}
	if startIndex == stopIndex {
		glog.D(logger.Error).Infoln("Up to date. Exiting.")
		os.Exit(0)
	}

	indexDb := MakeIndexDatabase(ctx)
	if indexDb == nil {
		glog.Fatal("indexes db is nil")
	}
	defer indexDb.Close()

	var block *types.Block
	blockIndex := startIndex
	block = bc.GetBlockByNumber(blockIndex)
	if block == nil {
		glog.Fatal(blockIndex, "block is nil")
	}

	var inc = uint64(ctx.Int("step"))
	startTime := time.Now()
	glog.D(logger.Error).Infoln("Address/tx indexing (atxi) start:", startIndex, "stop:", stopIndex, "step:", inc)
	for i := startIndex; i <= stopIndex; i = i+inc {
		if i+inc > stopIndex {
			inc = stopIndex - startIndex
		}
		// It may seem weird to pass i, i+inc, and inc, but its just a "coincidence"
		// The function could accepts a smaller step for batch putting (in this case, inc),
		// or a larger stopBlock (i+inc), but this is just how this cmd is using the fn now
		// We could mess around a little with exploring batch optimization...
		if err := bc.WriteBlockAddrTxIndexesBatch(indexDb, i, i+inc, inc); err != nil {
			return err
		}
		ioutil.WriteFile(placeholderFilename, []byte(strconv.Itoa(int(i+inc))), os.ModePerm)
	}
	ioutil.WriteFile(placeholderFilename, []byte(strconv.Itoa(int(stopIndex))), os.ModePerm)
	glog.D(logger.Error).Infoln("Finished atxi. Took:", time.Since(startTime))
	return nil
}


