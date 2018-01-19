package main

import (
	"gopkg.in/urfave/cli.v1"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/core/types"
	"os"
	"github.com/ethereumproject/go-ethereum/logger"
	"time"
	"github.com/ethereumproject/go-ethereum/core"
)

func buildAddrTxIndexCmd(ctx *cli.Context) error {
	startIndex := uint64(ctx.Int("start"))
	var stopIndex uint64

	indexDb := MakeIndexDatabase(ctx)
	if indexDb == nil {
		glog.Fatalln("indexes db is nil")
	}
	defer indexDb.Close()

	// Use persistent placeholder in case start not spec'd
	if !ctx.IsSet("start") {
		startIndex = core.GetATXIBookmark(indexDb)
	}

	bc, chainDB := MakeChain(ctx)
	if bc == nil || chainDB == nil {
		glog.Fatalln("bc or cdb is nil")
	}
	defer chainDB.Close()

	stopIndex = uint64(ctx.Int("stop"))
	if stopIndex == 0 {
		stopIndex = bc.CurrentBlock().NumberU64()
		if n := bc.CurrentFastBlock().NumberU64(); n > stopIndex {
			stopIndex = n
		}
	}

	if stopIndex < startIndex {
		glog.Fatalln("start must be prior to (smaller than) or equal to stop, got start=", startIndex, "stop=", stopIndex)
	}
	if startIndex == stopIndex {
		glog.D(logger.Error).Infoln("atxi is up to date, exiting")
		os.Exit(0)
	}

	var block *types.Block
	blockIndex := startIndex
	block = bc.GetBlockByNumber(blockIndex)
	if block == nil {
		glog.Fatalln(blockIndex, "block is nil")
	}

	var inc = uint64(ctx.Int("step"))
	startTime := time.Now()
	totalTxCount := uint64(0)
	glog.D(logger.Error).Infoln("Address/tx indexing (atxi) start:", startIndex, "stop:", stopIndex, "step:", inc, "| This may take a while.")
	breaker := false
	for i := startIndex; i <= stopIndex; i = i+inc {
		if i+inc > stopIndex {
			inc = stopIndex - i
			breaker = true
		}

		stepStartTime := time.Now()

		// It may seem weird to pass i, i+inc, and inc, but its just a "coincidence"
		// The function could accepts a smaller step for batch putting (in this case, inc),
		// or a larger stopBlock (i+inc), but this is just how this cmd is using the fn now
		// We could mess around a little with exploring batch optimization...
		txsCount, err := bc.WriteBlockAddrTxIndexesBatch(indexDb, i, i+inc, inc)
		if err != nil {
			return err
		}
		totalTxCount += uint64(txsCount)

		if err := core.SetATXIBookmark(indexDb, i+inc); err != nil {
			glog.Fatalln(err)
		}

		glog.D(logger.Error).Infof("atxi-build: block %d / %d txs: %d took: %v %.2f bps %.2f txps", i+inc, stopIndex, txsCount, time.Since(stepStartTime).Round(time.Millisecond), float64(inc)/time.Since(stepStartTime).Seconds(), float64(txsCount)/time.Since(stepStartTime).Seconds())
		glog.V(logger.Info).Infof("atxi-build: block %d / %d txs: %d took: %v %.2f bps %.2f txps", i+inc, stopIndex, txsCount, time.Since(stepStartTime).Round(time.Millisecond), float64(inc)/time.Since(stepStartTime).Seconds(), float64(txsCount)/time.Since(stepStartTime).Seconds())

		if breaker {
			break
		}
	}

	if err := core.SetATXIBookmark(indexDb, stopIndex); err != nil {
		glog.Fatalln(err)
	}

	// Print summary
	totalBlocksF := float64(stopIndex - startIndex)
	totalTxsF := float64(totalTxCount)
	took := time.Since(startTime)
	glog.D(logger.Error).Infof(`Finished atxi-build in %v: %d blocks (~ %.2f blocks/sec), %d txs (~ %.2f txs/sec)`,
		took.Round(time.Second),
		stopIndex - startIndex,
		totalBlocksF/took.Seconds(),
		totalTxCount,
		totalTxsF/took.Seconds(),
		)
	return nil
}


