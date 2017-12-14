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
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"strconv"
)

// availableLogStatusFeatures stores state of implemented log STATUS features.
// New features should be registered here, and their status updates by dispatchStatusLogs if in use (to avoid dupe goroutine logging).
var availableLogStatusFeatures = map[string]time.Duration{
	"sync": time.Duration(0),
}

type lsMode int

const (
	lsModeDiscover lsMode = iota
	lsModeFullSync
	lsModeFastSync
	lsModeImport
)

var lsModeName = []string{
	"Discover",
	"Sync",
	"Fast",
	"Import",
}

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

// dispatchStatusLogs handle parsing --log-status=argument and toggling appropriate goroutine status feature logging.
func dispatchStatusLogs(ctx *cli.Context, ethe *eth.Ethereum) {
	flagName := aliasableName(LogStatusFlag.Name, ctx)
	v := ctx.GlobalString(flagName)
	if v == "" {
		glog.Fatalf("%v: %v", flagName, ErrInvalidFlag)
	}

	parseStatusInterval := func(statusModule string, interval string) (tickerInterval time.Duration) {
		upcaseModuleName := strings.ToUpper(statusModule)
		if interval != "" {
			if ti, err := parseDuration(interval); err != nil {
				glog.Fatalf("%s %v: could not parse argument: %v", upcaseModuleName, err, interval)
			} else {
				tickerInterval = ti
			}
		}
		//glog.V(logger.Info).Infof("Rolling %s log interval set: %v", upcaseModuleName, tickerInterval)
		return tickerInterval
	}

	for _, p := range strings.Split(v, ",") {
		// Ignore hanging or double commas
		if p == "" {
			continue
		}

		// If possible, split sync=60 into ["sync", "60"], otherwise yields ["sync"], ["60"], or ["someothernonsense"]
		eqs := strings.Split(p, "=")
		if len(eqs) < 2 {
			glog.Errorf("Invalid log status value: %v. Must be comma-separated pairs of module=interval.", eqs)
			os.Exit(1)
		}

		// Catch unavailable and duplicate status feature logs
		if status, ok := availableLogStatusFeatures[eqs[0]]; !ok {
			glog.Errorf("%v: %v: unavailable status feature by name of '%v'", flagName, ErrInvalidFlag, eqs[0])
			os.Exit(1)
		} else if status.Seconds() != 0 {
			glog.Errorf("%v: %v: duplicate status feature by name of '%v'", flagName, ErrInvalidFlag, eqs[0])
			os.Exit(1)
		}

		// If user just uses "sync" instead of "sync=42", append empty string and delegate to each status log function how to handle it
		if len(eqs) == 1 {
			eqs = append(eqs, "")
		}

		d := parseStatusInterval(eqs[0], eqs[1])

		switch eqs[0] {
		case "sync":
			availableLogStatusFeatures["sync"] = d
			go runStatusSyncLogs(ctx, ethe, d)
		}
	}
}

// runStatusSyncLogs starts STATUS SYNC logging at a given interval.
// It should be run as a goroutine.
// eg. --log-status="sync=42" logs SYNC information every 42 seconds
func runStatusSyncLogs(ctx *cli.Context, e *eth.Ethereum, tickerInterval time.Duration) {

	// Set up ticker based on established interval.
	ticker := time.NewTicker(tickerInterval)
	var chainEventLastSent time.Time

	// Bookmark vars.
	var lastLoggedBlockNumber uint64
	var lastLoggedMode = lsModeDiscover

	var sigc = make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigc)

	// Should listen for events.
	// Proof of concept create event subscription
	ethEvents := e.EventMux().Subscribe(
		//// ChainEvent is called when a single block is inserted into the local blockchain.
		//// core.ChainEvent{},
		//// ChainSideEvent is called when a forked block is inserted into the local blockchain.
		//core.ChainSideEvent{},
		//// NewMinedBlockEvent is called when a new block is mined locally.
		//core.NewMinedBlockEvent{},
		// ChainInsertEvent is called when a batch of block is finished processing through the bc.InsertChain fn.
		// It includes statistics. Processed, queued, ignored, txcount, etc.
		core.ChainInsertEvent{},
		//// HeaderChainInsertEvent is called when headers are inserted into the headerchain, ie. fastsync.
		//core.HeaderChainInsertEvent{},
		// StartEvent is called when a peer is selected for synchronisation and sync begins.
		downloader.StartEvent{},
		// DoneEvent is called when synchronisation with a peer finishes without error.
		downloader.DoneEvent{},
		// FailedEvent is called when synchronisation with a peer finishes with an error.
		downloader.FailedEvent{},
	)
	greenParenify := func(s string) string {
		return logger.ColorGreen("⟪") + s + logger.ColorGreen("⟫")
	}
	//redParenify := func(s string) string {
	//	return logger.ColorRed("⟪") + s + logger.ColorRed("⟫")
	//}
	handleDownloaderEvent := func(ev interface{}) {
		//s := downloaderIcon + " "
		switch ev.(type) {
		//switch d := ev.(type) {
		case downloader.StartEvent:
			lastLoggedMode = getLogStatusMode(e)
			//s += "Start " + greenParenify(fmt.Sprintf("%s", d.Peer)) + " hash=" + greenParenify(d.Hash.Hex()[:9]+"…") + " TD=" + greenParenify(fmt.Sprintf("%v", d.TD))
			//glog.D(logger.Info).Infoln(s)
		case downloader.DoneEvent:
			lastLoggedMode = getLogStatusMode(e)
			//s += "Done  " + greenParenify(fmt.Sprintf("%s", d.Peer)) + " hash=" + greenParenify(d.Hash.Hex()[:9]+"…") + " TD=" + greenParenify(fmt.Sprintf("%v", d.TD))
			//glog.D(logger.Info).Infoln(s)
		case downloader.FailedEvent:
			lastLoggedMode = getLogStatusMode(e)
			//s += "Fail  " + greenParenify(fmt.Sprintf("%s", d.Peer)) + " " + logger.ColorRed("err") + "=" + redParenify(d.Err.Error())
			//glog.D(logger.Info).Warnln(s)
		}
	}

	go func() {
		for ev := range ethEvents.Chan() {
			switch d := ev.Data.(type) {
			// case core.ChainEvent:
			// 	glog.D(logger.Info).Infof("chainevent time=%v block=%v", e.Time, d.Block.NumberU64())
			case core.ChainInsertEvent:
				if lastLoggedMode == lsModeImport {
					lastLoggedBlockNumber = delegateStatusLogs(ctx, e, lastLoggedMode, lastLoggedBlockNumber, tickerInterval)
				}
				//glog.D(logger.Info).Infof(chainIcon+" Insert "+logger.ColorGreen("blocks")+"=%s "+logger.ColorGreen("◼")+"=%s "+logger.ColorGreen("took")+"=%s",
				//	greenParenify(fmt.Sprintf("processed=%4d queued=%4d ignored=%4d txs=%4d", d.Processed, d.Queued, d.Ignored, d.TxCount)),
				//	greenParenify(fmt.Sprintf("n=%8d hash=%s… time=%v ago", d.LastNumber, d.LastHash.Hex()[:9], time.Since(d.LatestBlockTime).Round(time.Millisecond))),
				//	greenParenify(fmt.Sprintf("%v", d.Elasped.Round(time.Millisecond))),
				//)
				//if bool(glog.D(logger.Info)) {
				//	chainEventLastSent = time.Now()
				//}
			case core.ChainSideEvent:
				glog.D(logger.Info).Infof(forkIcon+" Insert "+logger.ColorGreen("forked block")+"=%s", greenParenify(fmt.Sprintf("n=%8d hash=%s…", d.Block.NumberU64(), d.Block.Hash().Hex()[:9])))
			case core.HeaderChainInsertEvent:
				glog.D(logger.Info).Infof(headerIcon+" Insert "+logger.ColorGreen("headers")+"=%s "+logger.ColorGreen("❐")+"=%s"+logger.ColorGreen("took")+"=%s",
					greenParenify(fmt.Sprintf("processed=%4d ignored=%4d", d.Processed, d.Ignored)),
					greenParenify(fmt.Sprintf("n=%4d hash=%s…", d.LastNumber, d.LastHash.Hex()[:9])),
					greenParenify(fmt.Sprintf("%v", d.Elasped.Round(time.Microsecond))),
				)
				if bool(glog.D(logger.Info)) {
					chainEventLastSent = time.Now()
				}
			case core.NewMinedBlockEvent:
				glog.D(logger.Info).Infof(minedIcon + " Mined " + logger.ColorGreen("◼") + "=" + greenParenify(fmt.Sprintf("n=%8d hash=%s… coinbase=%s… txs=%3d uncles=%d",
					d.Block.NumberU64(),
					d.Block.Hash().Hex()[:9],
					d.Block.Coinbase().Hex()[:9],
					len(d.Block.Transactions()),
					len(d.Block.Uncles()),
				)))
			default:
				handleDownloaderEvent(d)
			}
		}
	}()

	// Initial log line so user knows is active.
	lastLoggedBlockNumber = delegateStatusLogs(ctx, e, lastLoggedMode, lastLoggedBlockNumber, tickerInterval)

	for {
		select {
		case <-ticker.C:
			if lastLoggedMode != lsModeImport {
				lastLoggedBlockNumber = delegateStatusLogs(ctx, e, lastLoggedMode, lastLoggedBlockNumber, tickerInterval)
			}
		case <-sigc:
			// Listen for interrupt
			ticker.Stop()
			glog.D(logger.Warn).Warnln("SYNC Stopping.")
			return
		}
	}
}

func getLogStatusMode(e *eth.Ethereum) lsMode {
	if e.Downloader().Synchronising() {
		switch e.Downloader().GetMode() {
		case downloader.FullSync:
			return lsModeFullSync
		case downloader.FastSync:
			return lsModeFastSync
		}
	}
	_, current, height, _, _ := e.Downloader().Progress() // origin, current, height, pulled, known
	if e.Downloader().GetPeers().Len() > 0 && current >= height && !(current == 0 && height == 0) {
		return lsModeImport
	}
	return lsModeDiscover
}

//func formatTabbedString(scanLn string, minwidth, tabwidth, padding int, padchar byte, flags uint) string {
//	w := new(tabwriter.Writer)
//	buf := bytes.NewBufferString("")
//	w.Init(buf, minwidth, tabwidth, padding, padchar, flags)
//	if w == nil {
//		return ""
//	}
//	fmt.Fprintf(w, scanLn)
//	w.Flush() // ignore error
//	return buf.String()
//}

// Spec:
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


func formatBlockNumber(i uint64) string {
	return "#"+strconv.FormatUint(i, 10)
}

// Examples of spec'd output.
var xlocalOfMaxD = "#92481951 of #93124363"        // #2481951 of #3124363
//var xpercentD = "   92.0%"           //    92.0%
var xlocalHeadHashD = "c76c34e7"     // c76c34e7
var xprogressRateD = " 117/ 129/ 11"      //  117/ 129/11
var xprogressRateUnitsD = "blk/txs/mgas sec" // blk/tx/mgas sec
var xpeersD = "18/25 peers"             //  18/25 peers

func strScanLenOf(s string, leftAlign bool) string {
	if leftAlign {
		return "%" + strconv.Itoa(len(s)) + "s"
	}
	return "%-" + strconv.Itoa(len(s)) + "s"
}

type printUnit struct {
	value string
	example string
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

//getLogStatusMode(e).PrintBasic(e, interval, lastLoggedBlockN, ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx)))
func (l lsMode) PrintBasic(e *eth.Ethereum, d time.Duration, lastLoggedBlockN uint64, maxPeers int) (uint64) {

	localOfMaxD := &printUnit{"", xlocalOfMaxD, true}
	percentOrHash := &printUnit{"", xlocalHeadHashD, false}
	progressRateD := &printUnit{"", xprogressRateD, false}      //  117/ 129/11
	progressRateUnitsD := &printUnit{"", xprogressRateUnitsD, false} // blk/tx/mgas sec
	peersD := &printUnit{"", xpeersD, false}             //  18/25 peers

	formatLocalOfMaxD := func(localheight, syncheight uint64) string {
		if localheight < syncheight {
			return fmt.Sprintf("%9s of %9s", formatBlockNumber(localheight), formatBlockNumber(syncheight))
		}
		return fmt.Sprintf(strScanLenOf(xlocalOfMaxD, true), formatBlockNumber(localheight))
	}

	formatPercentD := func(localheight, syncheight uint64) string {
		// Calculate and format percent sync of known height
		heightRatio := float64(localheight) / float64(syncheight)
		heightRatio = heightRatio * 100
		fHeightRatio := fmt.Sprintf("%4.2f%%", heightRatio)
		return fmt.Sprintf(strScanLenOf(xlocalHeadHashD, false), fHeightRatio)
	}

	formatBlockHashD := func(b *types.Block) string {
		return b.Hash().Hex()[2:2+len(xlocalHeadHashD)]
	}

	formatProgressRateD := func(blksN, txsN, mgasN int) string {
		if blksN < 0 {
			return fmt.Sprintf("    %4d/%2d",  txsN, mgasN)
		}
		if txsN < 0 {
			return fmt.Sprintf("%3d/    /%2d", blksN, mgasN)
		}
		return fmt.Sprintf("%3d/%4d/%2d", blksN, txsN, mgasN)
	}

	formatPeersD := func(peersN, maxpeersN int) string {
		return fmt.Sprintf("%2d/%2d peers", peersN, maxpeersN)
	}

	formatOutput := func(printunits ...*printUnit) string {
		o := []string{}
		for _, u := range printunits {
			o = append(o, strScanLenOf(u.example, u.leftAlign))
		}
		return strings.Join(o, " ")
	}

	peersD.value = formatPeersD(e.Downloader().GetPeers().Len(), maxPeers)
	defer func() {
		glog.D(logger.Warn).Infof("%-8s " + formatOutput(localOfMaxD, percentOrHash, progressRateD, progressRateUnitsD, peersD),
			lsModeName[l], localOfMaxD, percentOrHash, progressRateD, progressRateUnitsD, peersD)

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
		progressRateD.value = formatProgressRateD(blks / int(d.Seconds()), -1, mgas / int(d.Seconds()))
		progressRateUnitsD.value = fmt.Sprintf(strScanLenOf(xprogressRateUnitsD, false), "blk/   /mgas sec")
	case lsModeFullSync:
		localOfMaxD.value = formatLocalOfMaxD(localHead.NumberU64(), chainSyncHeight)
		percentOrHash.value = formatBlockHashD(localHead)
		progressRateD.value = formatProgressRateD(blks / int(d.Seconds()), txs / int(d.Seconds()), mgas / int(d.Seconds()))
		progressRateUnitsD.value = fmt.Sprintf(strScanLenOf(xprogressRateUnitsD, false), "blk/txs/mgas sec")
	case lsModeImport:
		localOfMaxD.value = formatLocalOfMaxD(localHead.NumberU64(), chainSyncHeight)
		percentOrHash.value = formatBlockHashD(localHead)
		progressRateD.value = fmt.Sprintf(strScanLenOf(xprogressRateD, false), formatProgressRateD(-1, txs, mgas))
		progressRateUnitsD.value = fmt.Sprintf(strScanLenOf(xprogressRateUnitsD, false), "    txs/mgas    ")
	default:
		panic("unreachable")
	}
	return current
}

func (l lsMode) PrintFancyGreen() {

}

func delegateStatusLogs(ctx *cli.Context, e *eth.Ethereum, mode lsMode, lastLoggedBlockN uint64, interval time.Duration) uint64 {
	return mode.PrintBasic(e, interval, lastLoggedBlockN, ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx)))

	//lenPeers := e.Downloader().GetPeers().Len()
	//
	//rtt, ttl, conf := e.Downloader().Qos()
	//confS := fmt.Sprintf("%01.2f", conf)
	//qosDisplay := fmt.Sprintf("rtt=%v ttl=%v conf=%s", rtt.Round(time.Millisecond), ttl.Round(time.Millisecond), confS)
	//
	//_, current, height, _, _ := e.Downloader().Progress() // origin, current, height, pulled, known
	//mode := e.Downloader().GetMode()
	//if mode == downloader.FastSync {
	//	current = e.BlockChain().CurrentFastBlock().NumberU64()
	//}
	//
	//// Get our head block
	//blockchain := e.BlockChain()
	//currentBlockHex := blockchain.CurrentBlock().Hash().Hex()
	//
	//// Discover -> not synchronising (searching for peers)
	//// FullSync/FastSync -> synchronising
	//// Import -> synchronising, at full height
	//fOfHeight := fmt.Sprintf("%7d", height)
	//
	//// Calculate and format percent sync of known height
	//heightRatio := float64(current) / float64(height)
	//heightRatio = heightRatio * 100
	//fHeightRatio := fmt.Sprintf("%4.2f%%", heightRatio)
	//
	//// Wait until syncing because real dl mode will not be engaged until then
	//lsMode = lsModeDiscover
	//if e.Downloader().Synchronising() {
	//	switch mode {
	//	case downloader.FullSync:
	//		lsMode = lsModeFullSync
	//	case downloader.FastSync:
	//		lsMode = lsModeFastSync
	//		currentBlockHex = blockchain.CurrentFastBlock().Hash().Hex()
	//	}
	//}
	//importMode := lenPeers > 0 && lsMode == lsModeDiscover && current >= height && !(current == 0 && height == 0)
	//if importMode {
	//	lsMode = lsModeImport
	//	fOfHeight = ""    // strings.Repeat(" ", 12)
	//	fHeightRatio = "" // strings.Repeat(" ", 7)
	//}
	//if height == 0 {
	//	fOfHeight = ""    // strings.Repeat(" ", 12)
	//	fHeightRatio = "" // strings.Repeat(" ", 7)
	//}
	//
	//// Calculate block stats for interval
	//numBlocksDiff := current - lastLoggedBlockN
	//numTxsDiff := 0
	//mGas := new(big.Int)
	//
	//var numBlocksDiffPerSecond uint64
	//var numTxsDiffPerSecond int
	//var mGasPerSecond = new(big.Int)
	//
	//var dominoGraph string
	//var nDom int
	//if numBlocksDiff > 0 && numBlocksDiff != current {
	//	for i := lastLoggedBlockN + 1; i <= current; i++ {
	//		b := blockchain.GetBlockByNumber(i)
	//		if b != nil {
	//			txLen := b.Transactions().Len()
	//			// Add to tallies
	//			numTxsDiff += txLen
	//			mGas = new(big.Int).Add(mGas, b.GasUsed())
	//			// Domino effect
	//			if lsMode == lsModeImport {
	//				if txLen > len(dominoes)-1 {
	//					// prevent slice out of bounds
	//					txLen = len(dominoes) - 1
	//				}
	//				if nDom <= 20 {
	//					dominoGraph += dominoes[txLen]
	//				}
	//				nDom++
	//			}
	//		}
	//	}
	//	if nDom > 20 {
	//		dominoGraph += "…"
	//	}
	//}
	//dominoGraph = logger.ColorGreen(dominoGraph)
	//
	//// Convert to per-second stats
	//// FIXME(?): Some degree of rounding will happen.
	//// For example, if interval is 10s and we get 6 blocks imported in that span,
	//// stats will show '0' blocks/second. Looks a little strange; but on the other hand,
	//// precision costs visual space, and normally just looks weird when starting up sync or
	//// syncing slowly.
	//numBlocksDiffPerSecond = numBlocksDiff / uint64(tickerInterval.Seconds())
	//
	//// Don't show initial current / per second val
	//if lastLoggedBlockNumber == 0 {
	//	numBlocksDiffPerSecond = 0
	//	numBlocksDiff = 0
	//}
	//
	//// Divide by interval to yield per-second stats
	//numTxsDiffPerSecond = numTxsDiff / int(tickerInterval.Seconds())
	//mGasPerSecond = new(big.Int).Div(mGas, big.NewInt(int64(tickerInterval.Seconds())))
	//mGasPerSecond = new(big.Int).Div(mGasPerSecond, big.NewInt(1000000))
	//mGasPerSecondI := mGasPerSecond.Int64()
	//
	//// Update last logged current block number
	//lastLoggedBlockNumber = current
	//
	//// Format head block hex for printing (eg. d4e…fa3)
	//cbhexstart := currentBlockHex[:9] // trim off '0x' prefix
	//
	//localHeadHeight := fmt.Sprintf("#%7d", current)
	//localHeadHex := fmt.Sprintf("%s…", cbhexstart)
	//peersOfMax := fmt.Sprintf("%2d/%2d peers", lenPeers, maxPeers)
	//domOrHeight := fOfHeight + " " + fHeightRatio
	//if len(strings.Replace(domOrHeight, " ", "", -1)) != 0 {
	//	domOrHeight = logger.ColorGreen("height") + "=" + greenParenify(domOrHeight)
	//} else {
	//	domOrHeight = ""
	//}
	//var blocksprocesseddisplay string
	//qosDisplayable := logger.ColorGreen("qos") + "=" + greenParenify(qosDisplay)
	//if lsMode != lsModeImport {
	//	blocksprocesseddisplay = logger.ColorGreen("~") + greenParenify(fmt.Sprintf("%4d blks %4d txs %2d mgas  "+logger.ColorGreen("/sec"), numBlocksDiffPerSecond, numTxsDiffPerSecond, mGasPerSecondI))
	//} else {
	//	blocksprocesseddisplay = logger.ColorGreen("+") + greenParenify(fmt.Sprintf("%4d blks %4d txs %8d mgas", numBlocksDiff, numTxsDiff, mGas.Uint64()))
	//	domOrHeight = dominoGraph
	//	qosDisplayable = ""
	//}
	//
	//// Log to ERROR.
	//headDisplay := greenParenify(localHeadHeight + " " + localHeadHex)
	//peerDisplay := greenParenify(peersOfMax)
	//
	//modeIcon := logger.ColorGreen(lsModeIcon[lsMode])
	//if lsMode == lsModeDiscover {
	//	modeIcon = lsModeDiscoverSpinners[lsModeN%3]
	//}
	//modeIcon = logger.ColorGreen(modeIcon)
	//lsModeN++
	//
	//// This allows maximum user optionality for desired integration with rest of event-based logging.
	//glog.D(logger.Warn).Infof("SYNC %s "+modeIcon+"%s %s "+logger.ColorGreen("✌︎︎︎")+"%s %s %s", lsModeName[lsMode], headDisplay, blocksprocesseddisplay, peerDisplay, domOrHeight, qosDisplayable)
}
