package main

import (
	"time"
	"fmt"
	"math/big"
	"github.com/ethereumproject/go-ethereum/logger"
	"strings"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/eth/downloader"
	"gopkg.in/urfave/cli.v1"
	"github.com/ethereumproject/go-ethereum/core"
)

var lsModeIcon = []string{
	"",
	"︎◉",
	"◎",
	"▶︎",
}

var dominoes = []string{"🁣", "🁤", "🁥", "🁦", "🁭", "🁴", "🁻", "🁼", "🂃", "🂄", "🂋", "🂌", "🂓"} // 🁣🁤🁥🁦🁭🁴🁻🁼🂃🂄🂋🂌🂓
var chainIcon = "◼⋯⋯" + logger.ColorGreen("◼")
var forkIcon = "◼⋯⦦" + logger.ColorGreen("◼")
var headerIcon = "◼⋯⋯" + logger.ColorGreen("❐")
var downloaderIcon = "◼⋯⋯" + logger.ColorGreen("⬇")
var minedIcon = "◼⋯⋯" + logger.ColorGreen("⟠")
var lsModeDiscoverSpinners = []string{"➫", "➬", "➭"}

func greenParenify(s string) string {
	return logger.ColorGreen("⟪") + s + logger.ColorGreen("⟫")
}
func redParenify(s string) string {
	return logger.ColorRed("⟪") + s + logger.ColorRed("⟫")
}

// greenDisplaySystem is "spec'd" in PR #423 and is a little fancier/more detailed and colorful than basic.
var greenDisplaySystem = displayEventHandlers{
	{
		eventT: logEventChainInsert,
		ev:     core.ChainInsertEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				switch d := evData.(type) {
				case core.ChainInsertEvent:
					glog.D(logger.Info).Infof(chainIcon+" Insert "+logger.ColorGreen("blocks")+"=%s "+logger.ColorGreen("◼")+"=%s "+logger.ColorGreen("took")+"=%s",
						greenParenify(fmt.Sprintf("processed=%4d queued=%4d ignored=%4d txs=%4d", d.Processed, d.Queued, d.Ignored, d.TxCount)),
						greenParenify(fmt.Sprintf("n=%8d hash=%s… time=%v ago", d.LastNumber, d.LastHash.Hex()[:9], time.Since(d.LatestBlockTime).Round(time.Millisecond))),
						greenParenify(fmt.Sprintf("%v", d.Elasped.Round(time.Millisecond))),
					)
					if bool(glog.D(logger.Info)) {
						chainEventLastSent = time.Now()
					}
				}
			},
		},
	},
	{
		eventT: logEventChainInsertSide,
		ev:     core.ChainSideEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				switch d := evData.(type) {
				case core.ChainSideEvent:
					glog.D(logger.Info).Infof(forkIcon+" Insert "+logger.ColorGreen("forked block")+"=%s", greenParenify(fmt.Sprintf("n=%8d hash=%s…", d.Block.NumberU64(), d.Block.Hash().Hex()[:9])))
				}
			},
		},
	},
	{
		eventT: logEventHeaderChainInsert,
		ev:     core.HeaderChainInsertEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				switch d := evData.(type) {
				case core.HeaderChainInsertEvent:
					glog.D(logger.Info).Infof(headerIcon+" Insert "+logger.ColorGreen("headers")+"=%s "+logger.ColorGreen("❐")+"=%s"+logger.ColorGreen("took")+"=%s",
						greenParenify(fmt.Sprintf("processed=%4d ignored=%4d", d.Processed, d.Ignored)),
						greenParenify(fmt.Sprintf("n=%4d hash=%s…", d.LastNumber, d.LastHash.Hex()[:9])),
						greenParenify(fmt.Sprintf("%v", d.Elasped.Round(time.Microsecond))),
					)
					if bool(glog.D(logger.Info)) {
						chainEventLastSent = time.Now()
					}
				}
			},
		},
	},
	{
		eventT: logEventMinedBlock,
		ev:     core.NewMinedBlockEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				switch d := evData.(type) {
				case core.NewMinedBlockEvent:
					glog.D(logger.Info).Infof(minedIcon + " Mined " + logger.ColorGreen("◼") + "=" + greenParenify(fmt.Sprintf("n=%8d hash=%s… coinbase=%s… txs=%3d uncles=%d",
						d.Block.NumberU64(),
						d.Block.Hash().Hex()[:9],
						d.Block.Coinbase().Hex()[:9],
						len(d.Block.Transactions()),
						len(d.Block.Uncles()),
					)))
				}
			},
		},
	},
	{
		eventT: logEventDownloaderStart,
		ev:     downloader.StartEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				switch d := evData.(type) {
				case downloader.StartEvent:
					s := downloaderIcon + " Start " + greenParenify(fmt.Sprintf("%s", d.Peer)) + " hash=" + greenParenify(d.Hash.Hex()[:9]+"…") + " TD=" + greenParenify(fmt.Sprintf("%v", d.TD))
					glog.D(logger.Info).Warnln(s)
				}
			},
		},
	},
	{
		eventT: logEventDownloaderDone,
		ev:     downloader.DoneEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				switch d := evData.(type) {
				case downloader.DoneEvent:
					s := downloaderIcon + " Done  " + greenParenify(fmt.Sprintf("%s", d.Peer)) + " hash=" + greenParenify(d.Hash.Hex()[:9]+"…") + " TD=" + greenParenify(fmt.Sprintf("%v", d.TD))
					glog.D(logger.Info).Warnln(s)
				}
			},
		},
	},
	{
		eventT: logEventDownloaderFailed,
		ev:     downloader.FailedEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				switch d := evData.(type) {
				case downloader.FailedEvent:
					s := downloaderIcon + " Fail  " + greenParenify(fmt.Sprintf("%s", d.Peer)) + " " + logger.ColorRed("err") + "=" + redParenify(d.Err.Error())
					glog.D(logger.Info).Warnln(s)
				}
			},
		},
	},
	{
		eventT: logEventInterval,
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				if time.Since(chainEventLastSent) > time.Duration(time.Second*time.Duration(int32(tickerInterval.Seconds()/2))) {
					currentBlockNumber = PrintStatusGreen(e, tickerInterval, ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx)))
				}
			},
		},
	},
}

// PrintStatusGreen implements the displayEventHandlerFn interface
var PrintStatusGreen = func(e *eth.Ethereum, tickerInterval time.Duration, maxPeers int) uint64 {
	lenPeers := e.Downloader().GetPeers().Len()

	rtt, ttl, conf := e.Downloader().Qos()
	confS := fmt.Sprintf("%01.2f", conf)
	qosDisplay := fmt.Sprintf("rtt=%v ttl=%v conf=%s", rtt.Round(time.Millisecond), ttl.Round(time.Millisecond), confS)

	_, current, height, _, _ := e.Downloader().Progress() // origin, current, height, pulled, known
	mode := e.Downloader().GetMode()
	if mode == downloader.FastSync {
		current = e.BlockChain().CurrentFastBlock().NumberU64()
	}

	// Get our head block
	blockchain := e.BlockChain()
	currentBlockHex := blockchain.CurrentBlock().Hash().Hex()

	// Discover -> not synchronising (searching for peers)
	// FullSync/FastSync -> synchronising
	// Import -> synchronising, at full height
	fOfHeight := fmt.Sprintf("%7d", height)

	// Calculate and format percent sync of known height
	heightRatio := float64(current) / float64(height)
	heightRatio = heightRatio * 100
	fHeightRatio := fmt.Sprintf("%4.2f%%", heightRatio)

	// Wait until syncing because real dl mode will not be engaged until then
	if currentMode == lsModeImport {
		fOfHeight = ""    // strings.Repeat(" ", 12)
		fHeightRatio = "" // strings.Repeat(" ", 7)
	}
	if height == 0 {
		fOfHeight = ""    // strings.Repeat(" ", 12)
		fHeightRatio = "" // strings.Repeat(" ", 7)
	}

	// Calculate block stats for interval
	numBlocksDiff := current - currentBlockNumber
	numTxsDiff := 0
	mGas := new(big.Int)

	var numBlocksDiffPerSecond uint64
	var numTxsDiffPerSecond int
	var mGasPerSecond = new(big.Int)

	var dominoGraph string
	var nDom int
	if numBlocksDiff > 0 && numBlocksDiff != current {
		for i := currentBlockNumber + 1; i <= current; i++ {
			b := blockchain.GetBlockByNumber(i)
			if b != nil {
				txLen := b.Transactions().Len()
				// Add to tallies
				numTxsDiff += txLen
				mGas = new(big.Int).Add(mGas, b.GasUsed())
				// Domino effect
				if currentMode == lsModeImport {
					if txLen > len(dominoes)-1 {
						// prevent slice out of bounds
						txLen = len(dominoes) - 1
					}
					if nDom <= 20 {
						dominoGraph += dominoes[txLen]
					}
					nDom++
				}
			}
		}
		if nDom > 20 {
			dominoGraph += "…"
		}
	}
	dominoGraph = logger.ColorGreen(dominoGraph)

	// Convert to per-second stats
	// FIXME(?): Some degree of rounding will happen.
	// For example, if interval is 10s and we get 6 blocks imported in that span,
	// stats will show '0' blocks/second. Looks a little strange; but on the other hand,
	// precision costs visual space, and normally just looks weird when starting up sync or
	// syncing slowly.
	numBlocksDiffPerSecond = numBlocksDiff / uint64(tickerInterval.Seconds())

	// Don't show initial current / per second val
	if currentBlockNumber == 0 {
		numBlocksDiffPerSecond = 0
		numBlocksDiff = 0
	}

	// Divide by interval to yield per-second stats
	numTxsDiffPerSecond = numTxsDiff / int(tickerInterval.Seconds())
	mGasPerSecond = new(big.Int).Div(mGas, big.NewInt(int64(tickerInterval.Seconds())))
	mGasPerSecond = new(big.Int).Div(mGasPerSecond, big.NewInt(1000000))
	mGasPerSecondI := mGasPerSecond.Int64()

	// Format head block hex for printing (eg. d4e…fa3)
	cbhexstart := currentBlockHex[:9] // trim off '0x' prefix

	localHeadHeight := fmt.Sprintf("#%7d", current)
	localHeadHex := fmt.Sprintf("%s…", cbhexstart)
	peersOfMax := fmt.Sprintf("%2d/%2d peers", lenPeers, maxPeers)
	domOrHeight := fOfHeight + " " + fHeightRatio
	if len(strings.Replace(domOrHeight, " ", "", -1)) != 0 {
		domOrHeight = logger.ColorGreen("height") + "=" + greenParenify(domOrHeight)
	} else {
		domOrHeight = ""
	}
	var blocksprocesseddisplay string
	qosDisplayable := logger.ColorGreen("qos") + "=" + greenParenify(qosDisplay)
	if currentMode != lsModeImport {
		blocksprocesseddisplay = logger.ColorGreen("~") + greenParenify(fmt.Sprintf("%4d blks %4d txs %2d mgas  "+logger.ColorGreen("/sec"), numBlocksDiffPerSecond, numTxsDiffPerSecond, mGasPerSecondI))
	} else {
		blocksprocesseddisplay = logger.ColorGreen("+") + greenParenify(fmt.Sprintf("%4d blks %4d txs %8d mgas", numBlocksDiff, numTxsDiff, mGas.Uint64()))
		domOrHeight = dominoGraph
		qosDisplayable = ""
	}

	// Log to ERROR.
	headDisplay := greenParenify(localHeadHeight + " " + localHeadHex)
	peerDisplay := greenParenify(peersOfMax)

	modeIcon := logger.ColorGreen(lsModeIcon[currentMode])
	if currentMode == lsModeDiscover {
		// TODO: spin me
		modeIcon = lsModeDiscoverSpinners[0]
	}
	modeIcon = logger.ColorGreen(modeIcon)

	// This allows maximum user optionality for desired integration with rest of event-based logging.
	glog.D(logger.Warn).Infof("%s "+modeIcon+"%s %s "+logger.ColorGreen("✌︎︎︎")+"%s %s %s",
		currentMode, headDisplay, blocksprocesseddisplay, peerDisplay, domOrHeight, qosDisplayable)
	return current
}
