package main

import (
	"strings"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/logger"
	"fmt"
	"math/big"
	"time"
	"strconv"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core"
	"gopkg.in/urfave/cli.v1"
	"github.com/ethereumproject/go-ethereum/eth/downloader"
)

// basicDisplaySystem is the basic display system spec'd in #127.
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
//
// FIXME: '16:52:45  Import #3124790                ' aligns right instead of left
var basicDisplaySystem = displayEventHandlers{
	{
		eventT: logEventChainInsert,
		ev:     core.ChainInsertEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				if currentMode == lsModeImport {
					currentBlockNumber = PrintStatusBasic(e, tickerInterval, ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx)))
					chainEventLastSent = time.Now()
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
				// If not in import mode, OR if we haven't yet logged a chain event.
				if currentMode != lsModeImport || time.Since(chainEventLastSent) > tickerInterval {
					currentBlockNumber = PrintStatusBasic(e, tickerInterval, ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx)))
				}
			},
		},
	},
}

func formatBlockNumber(i uint64) string {
	return "#" + strconv.FormatUint(i, 10)
}

// Examples of spec'd output.
var xlocalOfMaxD = "#92481951 of #93124363" // #2481951 of #3124363
//var xpercentD = "   92.0%"           //    92.0%
var xlocalHeadHashD = "c76c34e7"             // c76c34e7
var xprogressRateD = " 117/ 129/ 11"         //  117/ 129/11
var xprogressRateUnitsD = "blk/txs/mgas sec" // blk/tx/mgas sec
var xpeersD = "18/25 peers"                  //  18/25 peers

func strScanLenOf(s string, leftAlign bool) string {
	if leftAlign {
		return "%" + strconv.Itoa(len(s)) + "s"
	}
	return "%-" + strconv.Itoa(len(s)) + "s"
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
var PrintStatusBasic = func(e *eth.Ethereum, tickerInterval time.Duration, maxPeers int) uint64 {

	l := currentMode
	lastLoggedBlockN := currentBlockNumber

	localOfMaxD := &printUnit{"", xlocalOfMaxD, true}
	percentOrHash := &printUnit{"", xlocalHeadHashD, false}
	progressRateD := &printUnit{"", xprogressRateD, false}           //  117/ 129/11
	progressRateUnitsD := &printUnit{"", xprogressRateUnitsD, false} // blk/tx/mgas sec
	peersD := &printUnit{"", xpeersD, false}                         //  18/25 peers

	formatLocalOfMaxD := func(localheight, syncheight uint64) string {
		if localheight < syncheight {
			return fmt.Sprintf("%9s of %9s", formatBlockNumber(localheight), formatBlockNumber(syncheight))
		}
		return fmt.Sprintf(strScanLenOf(xlocalOfMaxD, true), formatBlockNumber(localheight))
	}

	formatPercentD := func(localheight, syncheight uint64) string {
		// Calculate and format percent sync of known height
		fHeightRatio := fmt.Sprintf("%4.2f%%", calcPercent(localheight, syncheight))
		return fmt.Sprintf(strScanLenOf(xlocalHeadHashD, false), fHeightRatio)
	}

	formatBlockHashD := func(b *types.Block) string {
		return b.Hash().Hex()[2 : 2+len(xlocalHeadHashD)]
	}

	formatProgressRateD := func(blksN, txsN, mgasN int) string {
		if blksN < 0 {
			return fmt.Sprintf("    %4d/%2d", txsN, mgasN)
		}
		if txsN < 0 {
			return fmt.Sprintf("%3d/    /%2d", blksN, mgasN)
		}
		return fmt.Sprintf("%3d/%4d/%2d", blksN, txsN, mgasN)
	}

	formatPeersD := func(peersN, maxpeersN int) string {
		return fmt.Sprintf("%2d/%2d peers", peersN, maxpeersN)
	}

	// formatOutputScanLn accepts printUnits and returns a scanln based on their example string length and
	// printUnit configured alignment.
	// eg. %12s %-8s %5s %15s
	formatOutputScanLn := func(printunits ...*printUnit) string {
		o := []string{}
		for _, u := range printunits {
			o = append(o, strScanLenOf(u.example, u.leftAlign))
		}
		return strings.Join(o, " ")
	}

	peersD.value = formatPeersD(e.Downloader().GetPeers().Len(), maxPeers)
	defer func() {
		glog.D(logger.Warn).Infof("%-8s "+formatOutputScanLn(localOfMaxD, percentOrHash, progressRateD, progressRateUnitsD, peersD),
			l, localOfMaxD, percentOrHash, progressRateD, progressRateUnitsD, peersD)

	}()
	if l == lsModeDiscover {
		return lastLoggedBlockN
	}

	origin, current, chainSyncHeight, _, _ := e.Downloader().Progress() // origin, current, height, pulled, known
	mode := e.Downloader().GetMode()
	if mode == downloader.FastSync {
		current = e.BlockChain().CurrentFastBlock().NumberU64()
	}
	localHead := e.BlockChain().GetBlockByNumber(current)

	// Calculate progress rates
	var blks, txs, mgas int
	if lastLoggedBlockN == 0 {
		blks, txs, mgas = calcBlockDiff(e, origin, localHead)
	} else {
		blks, txs, mgas = calcBlockDiff(e, lastLoggedBlockN, localHead)
	}

	switch l {
	case lsModeFastSync:
		lh := localHead.NumberU64()
		localOfMaxD.value = formatLocalOfMaxD(lh, chainSyncHeight)
		percentOrHash.value = formatPercentD(lh, chainSyncHeight)
		progressRateD.value = formatProgressRateD(blks/int(tickerInterval.Seconds()), -1, mgas/int(tickerInterval.Seconds()))
		progressRateUnitsD.value = fmt.Sprintf(strScanLenOf(xprogressRateUnitsD, false), "blk/   /mgas sec")
	case lsModeFullSync:
		localOfMaxD.value = formatLocalOfMaxD(localHead.NumberU64(), chainSyncHeight)
		percentOrHash.value = formatBlockHashD(localHead)
		progressRateD.value = formatProgressRateD(blks/int(tickerInterval.Seconds()), txs/int(tickerInterval.Seconds()), mgas/int(tickerInterval.Seconds()))
		progressRateUnitsD.value = fmt.Sprintf(strScanLenOf(xprogressRateUnitsD, false), "blk/txs/mgas sec")
	case lsModeImport:
		localOfMaxD.value = fmt.Sprintf(strScanLenOf(xlocalOfMaxD, true), formatBlockNumber(localHead.NumberU64()))
		percentOrHash.value = formatBlockHashD(localHead)
		progressRateD.value = fmt.Sprintf(strScanLenOf(xprogressRateD, false), formatProgressRateD(-1, txs, mgas))
		progressRateUnitsD.value = fmt.Sprintf(strScanLenOf(xprogressRateUnitsD, false), "    txs/mgas    ")
	default:
		panic("unreachable")
	}
	return current
}
