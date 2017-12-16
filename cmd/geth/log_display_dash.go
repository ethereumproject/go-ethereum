package main

import (
	"time"
	"gopkg.in/urfave/cli.v1"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/gizak/termui"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/core"
	"fmt"
)

const (
	tuiSmallHeight = 3
	tuiMediumHeight = 5
	tuiLargeHeight = 8
	tuiSmallWidth = 20
	tuiMediumWidth = 50
	tuiLargeWidth = 100

	tuiSpaceHeight = 1

	tuiDataLimit = 100
)

var (
	syncheightGauge *termui.Gauge

	peerCountSpark termui.Sparkline
	peerCountSparkHolder *termui.Sparklines

	peerList *termui.List
	peerListData []string

	mgasSpark             termui.Sparkline
	txsSpark              termui.Sparkline
	blkSpark              termui.Sparkline
	blkMgasTxsSparkHolder *termui.Sparklines
)


func tuiDrawDash(e *eth.Ethereum) {
	if currentMode == lsModeImport || currentMode == lsModeDiscover {
		syncheightGauge.Label = ""
	}
	if e != nil && e.IsListening() {
		cb := e.BlockChain().GetBlockByNumber(currentBlockNumber)
		syncheightGauge.BorderLabel = fmt.Sprintf("%s | local_head ◼ n=%d ⬡=%s txs=%d time=%v ago", lsModeName[currentMode], currentBlockNumber, cb.Hash().Hex()[:10] + "…", cb.Transactions().Len(), time.Since(time.Unix(cb.Time().Int64(), 0)).Round(time.Second))
	}
	termui.Render(syncheightGauge, peerCountSparkHolder, peerList, blkMgasTxsSparkHolder)
}

func tuiSetupDashComponents() {
	//// Sync height gauge
	syncheightGauge = termui.NewGauge()
	syncheightGauge.Percent = 0
	syncheightGauge.BarColor = termui.ColorRed
	syncheightGauge.Height = tuiSmallHeight
	syncheightGauge.Width = tuiLargeWidth
	//// Mgas spark
	mgasSpark = termui.Sparkline{}
	mgasSpark.Title = "Mgas"
	mgasSpark.Data = []int{}
	mgasSpark.Height = tuiSmallHeight
	mgasSpark.LineColor = termui.ColorYellow
	//// Txs spark
	txsSpark = termui.Sparkline{}
	txsSpark.Title = "Txs"
	txsSpark.Data = []int{}
	txsSpark.Height = tuiSmallHeight
	txsSpark.LineColor = termui.ColorMagenta
	//// Blk spark
	blkSpark = termui.Sparkline{}
	blkSpark.Title = "Blks"
	blkSpark.Data = []int{}
	blkSpark.Height = tuiSmallHeight
	blkSpark.LineColor = termui.ColorGreen
	//// MgasTxs spark holder
	blkMgasTxsSparkHolder = termui.NewSparklines(blkSpark, mgasSpark, txsSpark)
	blkMgasTxsSparkHolder.Height = mgasSpark.Height + txsSpark.Height + blkSpark.Height + tuiSpaceHeight*6
	blkMgasTxsSparkHolder.Width = syncheightGauge.Width
	blkMgasTxsSparkHolder.Y = syncheightGauge.Y + syncheightGauge.Height
	blkMgasTxsSparkHolder.X = syncheightGauge.X

	//// Peer count spark
	peerCountSpark = termui.Sparkline{}
	peerCountSpark.LineColor = termui.ColorBlue
	peerCountSpark.Data = []int{0}
	peerCountSpark.Height = tuiMediumHeight
	//// Peer count spark holder
	peerCountSparkHolder = termui.NewSparklines(peerCountSpark)
	peerCountSparkHolder.BorderLabel = "Peers (0)"
	peerCountSparkHolder.BorderLabelFg = termui.ColorBlue
	peerCountSparkHolder.BorderBottom = false
	peerCountSparkHolder.X = 0
	peerCountSparkHolder.Y = blkMgasTxsSparkHolder.Y + blkMgasTxsSparkHolder.Height
	peerCountSparkHolder.Height = tuiMediumHeight + tuiSpaceHeight*3
	peerCountSparkHolder.Width = syncheightGauge.Width

	//// Peer list
	peerList = termui.NewList()
	peerList.Items = peerListData
	peerList.X = 0
	peerList.Y = peerCountSparkHolder.Y + peerCountSparkHolder.Height
	peerList.Width = peerCountSparkHolder.Width
	peerList.Height = tuiLargeHeight*2
	peerList.BorderTop = false
}

func addDataWithLimit(sl []int, dataPoint int, maxLen int) []int {
	if len(sl) > maxLen {
		sl = append(sl[1:], dataPoint)
		return sl
	}
	sl = append(sl, dataPoint)
	return sl
}

// greenDisplaySystem is "spec'd" in PR #423 and is a little fancier/more detailed and colorful than basic.
var dashDisplaySystem = displayEventHandlers{
	{
		eventT: logEventBefore,
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				// Disable display logging.
				d := glog.GetDisplayable()
				glog.SetD(0)
				go func() {
					// Reset display logs.
					defer glog.SetD(int(*d))

					err := termui.Init()
					if err != nil {
						panic(err)
					}
					tuiSetupDashComponents()
					tuiSetupHandlers()
					if currentBlockNumber == 0 {
						_, c, _, _, _ := e.Downloader().Progress()
						currentBlockNumber = c
					}
					tuiDrawDash(e)

					termui.Loop()
				}()
			},
		},
	},
	{
		eventT: logEventChainInsert,
		ev:     core.ChainInsertEvent{},
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				switch d := evData.(type) {
				case core.ChainInsertEvent:
					localheight := d.LastNumber
					_, head, syncheight, _ ,_ := e.Downloader().Progress()
					if head > localheight {
						localheight = head
					}
					syncheightGauge.Percent = int(calcPercent(localheight, syncheight))
					if localheight >= syncheight {
						syncheightGauge.Label = fmt.Sprintf("%d", localheight)
						syncheightGauge.BarColor = termui.ColorGreen
					} else {
						syncheightGauge.Label = fmt.Sprintf("%d / %d", localheight, syncheight)
						syncheightGauge.BarColor = termui.ColorRed
					}

					if currentBlockNumber != 0 {
						b :=  e.BlockChain().GetBlockByNumber(localheight)
						if b == nil {
							return
						}
						blks, txs, mgas := calcBlockDiff(e, currentBlockNumber, b)
						// blk
						blkMgasTxsSparkHolder.Lines[0].Data = addDataWithLimit(blkMgasTxsSparkHolder.Lines[0].Data, blks, tuiDataLimit)
						blkMgasTxsSparkHolder.Lines[0].Title = fmt.Sprintf("n=%d] ∆ blks=%d (inserted_at=%v took=%v)", localheight, blks, time.Now().Round(time.Second), d.Elasped.Round(time.Millisecond))
						// mgas
						blkMgasTxsSparkHolder.Lines[1].Data = addDataWithLimit(blkMgasTxsSparkHolder.Lines[1].Data, mgas, tuiDataLimit)
						blkMgasTxsSparkHolder.Lines[1].Title = fmt.Sprintf("n=%d] ∑ mgas=%2d/%4dblks", localheight, mgas, blks)
						// txs
						blkMgasTxsSparkHolder.Lines[2].Data = addDataWithLimit(blkMgasTxsSparkHolder.Lines[2].Data, txs, tuiDataLimit)
						blkMgasTxsSparkHolder.Lines[2].Title = fmt.Sprintf("n=%d] ∑ txs=%3d/%4dblks", localheight, txs, blks)
					}
					currentBlockNumber = localheight
					tuiDrawDash(e)
				default:
					panic(d)
				}
			},
		},
	},
	{
		eventT: logEventAfter,
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				termui.StopLoop()
				termui.Close()
				return

			},
		},
	},
	{
		eventT: logEventInterval,
		handlers: displayEventHandlerFns{
			func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration) {
				peers := e.Downloader().GetPeers()
				peerCountSparkHolder.Lines[0].Data = addDataWithLimit(peerCountSparkHolder.Lines[0].Data, int(peers.Len()), tuiDataLimit)
				peerCountSparkHolder.BorderLabel = fmt.Sprintf("Peers (%d / %d)", int(peers.Len()), ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx)))

				peerListData = []string{}
				for _, p := range peers.AllPeers() {
					peerListData = append(peerListData, p.String())
				}
				peerList.Items = peerListData
				tuiDrawDash(e)
			},
		},
	},
}

func tuiSetupHandlers() {
	termui.Handle("interrupt", func(tue termui.Event) {
		glog.V(logger.Error).Errorln("interrupt 1")
		termui.StopLoop()
		termui.Close()
	})
	termui.Handle("/interrupt", func(tue termui.Event) {
		glog.V(logger.Error).Errorln("interrupt 2")
		termui.StopLoop()
		termui.Close()
	})
	termui.Handle("/sys/interrupt", func(tue termui.Event) {
		glog.V(logger.Error).Errorln("interrupt 3")
		termui.StopLoop()
		termui.Close()
	})
	termui.Handle("/sys/kbd/C-c", func(tue termui.Event) {
		glog.V(logger.Error).Errorln("/sys/kbd/C-c")
		termui.StopLoop()
		termui.Close()
	})
	termui.Handle("/sys/kbd/C-x", func(tue termui.Event) {
		glog.V(logger.Error).Errorln("/sys/kbd/C-x")
		termui.StopLoop()
		termui.Close()
	})
	termui.Handle("/sys/kbd/C-z", func(tue termui.Event) {
		glog.V(logger.Error).Errorln("/sys/kbd/C-z")
		termui.StopLoop()
		termui.Close()
	})
	termui.Handle("/sys/kbd/q", func(tue termui.Event) {
		glog.V(logger.Error).Errorln("/sys/kbd/q")
		termui.StopLoop()
		termui.Close()
	})
	termui.Handle("/sys/kbd/r", func(tue termui.Event) {
		// Just check for len > 1 for an arbitrary set because they're all updated simultaneously.
		// This is also a totally opinionated and convenience thing.
		if len(blkMgasTxsSparkHolder.Lines[0].Data) > 1 {
			blkMgasTxsSparkHolder.Lines[0].Data = blkMgasTxsSparkHolder.Lines[0].Data[1:]
			blkMgasTxsSparkHolder.Lines[1].Data = blkMgasTxsSparkHolder.Lines[1].Data[1:]
			blkMgasTxsSparkHolder.Lines[2].Data = blkMgasTxsSparkHolder.Lines[2].Data[1:]
		}
	})
}