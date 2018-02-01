// ---
//┌Import | local_head ◼ n=5047652 ⬡=0x33f6878a… txs=1 time=21s ago──────────────────────────────────┐
//│                                                                                                  │
//└──────────────────────────────────────────────────────────────────────────────────────────────────┘
//┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
//│n=5047652] ∆ blks=1 (inserted_at=2017-12-19 09:51:35 -0600 CST took=23ms)                         │
//│                                                                                                  │
//│                                                                                                  │
//│                                                                                                  │
//│n=5047652] ∑ mgas= 0/   1blks                                                                     │
//│                                                                                                  │
//│                                                                                                  │
//│                                                                                                  │
//│n=5047652] ∑ txs=  1/   1blks                                                                     │
//│                                ▃          ▆            ▅                               ▇         │
//│    ▃ ▁    ▁    ▂  ▄                      ▆  ▁▂ ▄   ▃       ▅    ▁    ▁          ▇▆      ▂        │
//│▁▇   ▃  ▆   ▁▁▁   ▁ ▂ ▂▁▂▂▆▂▁▁▁  ▁▄▁ ▃▁▂▅▇     ▅ ▄ ▁      ▂    ▆   ▂▄▇ ▁ ▁ ▃▅▄▇▃   ▃  ▃   ▂ ▁   ▁▁│
//│                                                                                                  │
//└──────────────────────────────────────────────────────────────────────────────────────────────────┘
//┌Peers (13 / 25)───────────────────────────────────────────────────────────────────────────────────┐
//│                                                                                                  │
//│                                                                                                  │
//│                                                                                                  │
//│                                                                                                  │
//│                                                                                                  │
//│                                                                                                  │
//
//│Peer id=5701cea16e32cb06 eth/63 [Parity/v1.7.9-stable-12940e4-20171113/x86_64-linux-gnu/rustc1.21…│
//│Peer id=8310b1f98b9b7afe eth/63 [Parity/v1.7.0-unstable-5f2cabd-20170727/x86_64-linux-gnu/rustc1.…│
//│Peer id=5fbfb426fbb46f8b eth/63 [Parity/v1.7.7-stable-eb7c648-20171015/x86_64-linux-gnu/rustc1.20…│
//│Peer id=1922bfd2acc1e82f eth/63 [Parity/v1.8.2-beta-1b6588c-20171025/x86_64-linux-gnu/rustc1.21.0…│
//│Peer id=8ed199326f981ae9 eth/63 [Parity/v1.8.0-beta-9882902-20171015/x86_64-linux-gnu/rustc1.20.0…│
//│Peer id=c89d548e2a55d324 eth/63 [Parity/v1.8.2-beta-1b6588c-20171025/x86_64-linux-gnu/rustc1.21.0…│
//│Peer id=13c960f1da9a6337 eth/63 [Parity/v1.8.4-beta-c74c8c1-20171211/x86_64-linux-gnu/rustc1.22.1…│
//│Peer id=021eb56de4a7b725 eth/63 [Parity/v1.6.9-stable-d44b008-20170716/x86_64-linux-gnu/rustc1.18…│
//│Peer id=385ffdfb7f7cd2f5 eth/63 [Geth/v4.0.0/windows/go1.8] [hs 0.00/s, bs 0.00/s, rs 0.00/s, ss …│
//│Peer id=d5a5d20cb8b77e14 eth/63 [Parity/v1.8.2-beta-1b6588c-20171025/x86_64-linux-gnu/rustc1.21.0…│
//│Peer id=5f259d71e80f710b eth/63 [Parity/v1.7.9-stable-12940e4-20171113/x86_64-linux-gnu/rustc1.21…│
//│Peer id=e7f3c98896581fee eth/63 [Parity/v1.7.9-unstable-12940e4-20171113/x86_64-linux-gnu/rustc1.…│
//│Peer id=cb0e1d693f950573 eth/63 [Parity/v1.7.10-stable-3931713-20171211/x86_64-linux-gnu/rustc1.2…│
//│                                                                                                  │
//│                                                                                                  │
//└──────────────────────────────────────────────────────────────────────────────────────────────────┘
// ---

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
		syncheightGauge.BorderLabel = fmt.Sprintf("%s | local_head ◼ n=%d ⬡=%s txs=%d time=%v ago", currentMode, currentBlockNumber, cb.Hash().Hex()[:10] + "…", cb.Transactions().Len(), time.Since(time.Unix(cb.Time().Int64(), 0)).Round(time.Second))
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

// dashDisplaySystem is an experimental display system intended as a proof of concept for the display dispatch system
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