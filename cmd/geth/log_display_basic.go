// ---
//2017-02-03 16:44:00  Discover                                                              0/25 peers
//2017-02-03 16:45:00  Discover                                                              1/25 peers
//2017-02-03 16:46:00  Fast   #2481951 of #3124363    79.4%   1211/  554    blk/mgas sec     6/25 peers
//2017-02-03 16:47:00  Fast   #2689911 of #3124363    86.1%    611/  981    blk/mgas sec     6/25 peers
//2017-02-03 16:48:00  Fast   #2875913 of #3124363    92.0%    502/  760    blk/mgas sec     4/25 peers
//2017-02-03 16:49:00  Sync   #3124227 of #3124363 c76c34e7   77/ 242/ 7 blk/tx/mgas sec     4/25 peers
//2017-02-03 16:50:00  Sync   #3124247 of #3124363 75e48eff   51/  51/ 5 blk/tx/mgas sec     4/25 peers
//2017-02-03 16:51:00  Sync   #3124567 of #3124363 9af334ae  117/ 129/11 blk/tx/mgas sec     5/25 peers
//2017-02-03 16:52:00  Sync   #3124787 of #3124363 1e3a8351    9/   6/ 1 blk/tx/mgas sec     7/25 peers
//2017-02-03 16:52:05  Import #3124788             84e11ff4        15/ 7 tx/mgas            10/25 peers
//2017-02-03 16:52:25  Import #3124789             9e45a241         5/ 1 tx/mgas            12/25 peers
//2017-02-03 16:52:45  Import #3124790             d819f71c         0/ 0 tx/mgas            18/25 peers
//2017-02-03 16:52:46  Mined  #3124791             b719f31b         7/ 1 tx/mgas            18/25 peers
// ---

package main

import (
	"fmt"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/eth/downloader"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"strconv"
	"time"
)

// basicDisplaySystem is the basic display system spec'd in #127.
var basicDisplaySystem = displayEventHandlers{
	{
		eventT: logEventChainInsert,
		ev:     core.ChainInsertEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				// Conditional prevents chain insert event logs during full/fast sync
				if currentMode == lsModeImport {
					switch d := evData.(type) {
					case core.ChainInsertEvent:
						currentBlockNumber = PrintStatusBasic(e, tickerInterval, &d, ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx)))
						chainEventLastSent = time.Now()
					}
				}
			},
		},
	},
	{
		eventT: logEventMinedBlock,
		ev: core.NewMinedBlockEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				switch d := evData.(type) {
				case core.NewMinedBlockEvent:
					glog.D(logger.Warn).Infof(basicScanLn,
						"Mined",
							formatBlockNumber(d.Block.NumberU64()),
							d.Block.Hash().Hex()[2 : 2+len(xlocalHeadHashD)],
							fmt.Sprintf("%3d/%2d", d.Block.Transactions().Len(), new(big.Int).Div(d.Block.GasUsed(), big.NewInt(1000000)).Int64()),
							"txs/mgas",
							fmt.Sprintf("%2d/%2d peers", e.Downloader().GetPeers().Len(), ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx))),
					)
					currentBlockNumber = d.Block.NumberU64()
				}
			},
		},
	},
	{
		eventT: logEventDownloaderStart,
		ev:     downloader.StartEvent{},
	},
	{
		eventT: logEventDownloaderDone,
		ev:     downloader.DoneEvent{},
	},
	{
		eventT: logEventDownloaderFailed,
		ev:     downloader.FailedEvent{},
	},
	{
		eventT: logEventInterval,
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				// If not in import mode OR if we haven't logged a chain event.
				if currentMode != lsModeImport || chainEventLastSent.IsZero() {
					currentBlockNumber = PrintStatusBasic(e, tickerInterval, nil, ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx)))
				}
			},
		},
	},
}

func formatBlockNumber(i uint64) string {
	return "#" + strconv.FormatUint(i, 10)
}

// Examples of spec'd output.
const (
	xlocalOfMaxD = "#92481951 of #93124363" // #2481951 of #3124363
	// xpercentD = "   92.0%"                //    92.0% // commented because it is an item in the spec, but shorter than xLocalHeadHashD
	xlocalHeadHashD     = "c76c34e7"         // c76c34e7
	xprogressRateD      = " 117/ 129/ 11"    //  117/ 129/11
	xprogressRateUnitsD = "blk/txs/mgas sec" // blk/tx/mgas sec
	xpeersD             = "18/25 peers"      //  18/25 peers
)

const basicScanLn = "%-8s %-22s %8s %13s %-16s %11s"

func strScanLenOf(s string, leftAlign bool) string {
	if leftAlign {
		return "%-" + strconv.Itoa(len(s)) + "s"
	}
	return "%" + strconv.Itoa(len(s)) + "s"
}

type printUnit struct {
	value     string
	example   string
	leftAlign bool
}

func (p *printUnit) String() string {
	return fmt.Sprintf("%s", p.value)
}

func calcBlockDiff(e *eth.Ethereum, lastLoggedBlockN uint64, localHead *types.Block) (blks, txs, mgas int) {
	// Calculate block stats for interval
	localHeadN := localHead.NumberU64()
	blks = int(localHeadN - lastLoggedBlockN)
	txs = 0
	mGas := new(big.Int)

	for i := lastLoggedBlockN + 1; i <= localHeadN; i++ {
		b := e.BlockChain().GetBlockByNumber(i)
		if b != nil {
			// Add to tallies
			txs += b.Transactions().Len()
			mGas = mGas.Add(mGas, b.GasUsed())
		}
	}
	mGas.Div(mGas, big.NewInt(1000000))
	return blks, txs, int(mGas.Int64())
}

func calcPercent(quotient, divisor uint64) float64 {
	out := float64(quotient) / float64(divisor)
	return out * 100
}

// PrintStatusBasic implements the displayEventHandlerFn interface
var PrintStatusBasic = func(e *eth.Ethereum, tickerInterval time.Duration, insertEvent *core.ChainInsertEvent, maxPeers int) uint64 {

	// Set variable copy of current mode to avoid issue around currentMode's non-thread safety
	currentModeLocal := currentMode

	localOfMaxD := &printUnit{"", xlocalOfMaxD, true}
	percentOrHash := &printUnit{"", xlocalHeadHashD, false}
	progressRateD := &printUnit{"", xprogressRateD, false}           //  117/ 129/11
	progressRateUnitsD := &printUnit{"", xprogressRateUnitsD, false} // blk/tx/mgas sec
	peersD := &printUnit{"", xpeersD, false}                         //  18/25 peers

	formatLocalOfMaxD := func(localheight, syncheight uint64) string {
		if localheight < syncheight {
			return fmt.Sprintf("%9s of %9s", formatBlockNumber(localheight), formatBlockNumber(syncheight))
		}
		// Show diff if imported more than one block.
		if insertEvent != nil && insertEvent.Processed > 1 {
			return fmt.Sprintf("%9s (+%4d)     ", formatBlockNumber(localheight), insertEvent.Processed)
		}
		return fmt.Sprintf("%9s             ", formatBlockNumber(localheight))
	}

	formatPercentD := func(localheight, syncheight uint64) string {
		// Calculate and format percent sync of known height
		fHeightRatio := fmt.Sprintf("%4.2f%%", calcPercent(localheight, syncheight))
		return fmt.Sprintf("%s", fHeightRatio)
	}

	formatBlockHashD := func(b *types.Block) string {
		return b.Hash().Hex()[2 : 2+len(xlocalHeadHashD)]
	}

	formatProgressRateD := func(blksN, txsN, mgasN int) string {
		if blksN < 0 {
			return fmt.Sprintf("%4d/%2d", txsN, mgasN)
		}
		if txsN < 0 {
			return fmt.Sprintf("%3d/%2d", blksN, mgasN)
		}
		return fmt.Sprintf("%3d/%4d/%2d", blksN, txsN, mgasN)
	}

	formatPeersD := func(peersN, maxpeersN int) string {
		return fmt.Sprintf("%2d/%2d peers", peersN, maxpeersN)
	}

	peersD.value = formatPeersD(e.Downloader().GetPeers().Len(), maxPeers)
	defer func() {
		glog.D(logger.Warn).Infof(basicScanLn,
			currentModeLocal, localOfMaxD, percentOrHash, progressRateD, progressRateUnitsD, peersD)

	}()

	origin, current, chainSyncHeight, _, _ := e.Downloader().Progress() // origin, current, height, pulled, known
	mode := e.Downloader().GetMode()
	if mode == downloader.FastSync {
		current = e.BlockChain().CurrentFastBlock().NumberU64()
	}

	if currentModeLocal == lsModeDiscover {
		return current
	}

	var localHead *types.Block
	if insertEvent != nil {
		if evB := e.BlockChain().GetBlock(insertEvent.LastHash); evB != nil && currentModeLocal == lsModeImport {
			localHead = evB
		}
	} else {
		localHead = e.BlockChain().GetBlockByNumber(current)
	}
	// Sanity/safety check
	if localHead == nil {
		localHead = e.BlockChain().CurrentBlock()
		if mode == downloader.FastSync {
			localHead = e.BlockChain().CurrentFastBlock()
		}
	}

	// Calculate progress rates
	var blks, txs, mgas int
	if currentModeLocal == lsModeImport && insertEvent != nil && insertEvent.Processed == 1 {
		blks, txs, mgas = 1, localHead.Transactions().Len(), int(new(big.Int).Div(localHead.GasUsed(), big.NewInt(1000000)).Uint64())
	} else if insertEvent != nil && insertEvent.Processed > 1 {
		blks, txs, mgas = calcBlockDiff(e, localHead.NumberU64() - uint64(insertEvent.Processed), localHead)
	} else if currentBlockNumber == 0 && origin > 0 {
		blks, txs, mgas = calcBlockDiff(e, origin, localHead)
	} else if currentBlockNumber != 0 && currentBlockNumber < localHead.NumberU64() {
		blks, txs, mgas = calcBlockDiff(e, currentBlockNumber, localHead)
	} else {
		blks, txs, mgas = calcBlockDiff(e, localHead.NumberU64() - 1, localHead)
	}

	switch currentModeLocal {
	case lsModeFastSync:
		lh := localHead.NumberU64()
		localOfMaxD.value = formatLocalOfMaxD(lh, chainSyncHeight)
		percentOrHash.value = formatPercentD(lh, chainSyncHeight)
		progressRateD.value = formatProgressRateD(blks/int(tickerInterval.Seconds()), -1, mgas/int(tickerInterval.Seconds()))
		progressRateUnitsD.value = fmt.Sprintf(strScanLenOf(xprogressRateUnitsD, true), "blk/mgas sec")
	case lsModeFullSync:
		localOfMaxD.value = formatLocalOfMaxD(localHead.NumberU64(), chainSyncHeight)
		percentOrHash.value = formatBlockHashD(localHead)
		progressRateD.value = formatProgressRateD(blks/int(tickerInterval.Seconds()), txs/int(tickerInterval.Seconds()), mgas/int(tickerInterval.Seconds()))
		progressRateUnitsD.value = fmt.Sprintf(strScanLenOf(xprogressRateUnitsD, true), "blk/txs/mgas sec")
	case lsModeImport:
		localOfMaxD.value = formatLocalOfMaxD(localHead.NumberU64(), chainSyncHeight)
		percentOrHash.value = formatBlockHashD(localHead)
		progressRateD.value = fmt.Sprintf(strScanLenOf(xprogressRateD, false), formatProgressRateD(-1, txs, mgas))
		progressRateUnitsD.value = fmt.Sprintf(strScanLenOf(xprogressRateUnitsD, true), "txs/mgas")
	default:
		// Without establishing currentModeLocal it would be possible to reach this case if currentMode changed during
		// execution of last ~40 lines.
	}
	return current
}
