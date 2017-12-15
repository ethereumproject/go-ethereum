package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/eth/downloader"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
)

// availableLogStatusFeatures stores state of implemented log STATUS features.
// New features should be registered here, and their status updates by dispatchStatusLogs if in use (to avoid dupe goroutine logging).
var availableLogStatusFeatures = map[string]time.Duration{
	"sync": time.Duration(0),
}

type lsMode uint

const (
	lsModeDiscover lsMode = iota
	lsModeFullSync
	lsModeFastSync
	lsModeImport
)

// displayEventHandlerFn is a function that gets called when something happens; where that 'something'
// is decided by the displayEventHandler the fn belongs to. It's type accepts a standard interface signature and
// returns nothing. evData can be nil, and will be, particularly, when the handler is the "INTERVAL" callee.
type displayEventHandlerFn func(ctx *cli.Context, e *eth.Ethereum, evData interface{}, tickerInterval time.Duration)
type displayEventHandlerFns []displayEventHandlerFn

// displayEventHandler is a unit of "listening" that can be added to the display system handlers to configure
// what is listened for and how to respond to the given event. 'ev' is an event as received from the Ethereum Mux subscription,
// or nil in the case of INTERVAL. Note, as exemplified below, that in order to make use of the ev data it's required
// to use a (hacky) single switch to .(type) the event data
type displayEventHandler struct {
	eventName string      // used for labeling events and matching to the switch statement
	ev        interface{} // which event to handle. if nil, will run on the ticker.
	// (ctx *cli.Context, e *eth.Ethereum, evData interface{}, mode *lsMode, tickerInterval time.Duration, n *uint64)
	handlers displayEventHandlerFns
}
type displayEventHandlers []displayEventHandler

// getByName looks up a handler by name to see if it's "registered" for a given display system.
func (hs displayEventHandlers) getByName(name string) (*displayEventHandler, bool) {
	for _, h := range hs {
		if h.eventName == name {
			return &h, true
		}
	}
	return nil, false
}

// runAllIfAny runs all configured fns for a given event, if registered.
func (hs *displayEventHandlers) runAllIfAny(ctx *cli.Context, e *eth.Ethereum, d interface{}, tickerInterval time.Duration, name string) {
	if h, ok := hs.getByName(name); ok {
		for _, handler := range h.handlers {
			handler(ctx, e, d, tickerInterval)
		}
	}
}

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

		displaySystem := basicDisplaySystem
		displayFmt := ctx.GlobalString(DisplayFormatFlag.Name)
		if displayFmt == "green" {
			displaySystem = greenDisplaySystem
		}
		switch eqs[0] {
		case "sync":
			availableLogStatusFeatures["sync"] = d
			go runDisplayLogs(ctx, ethe, d, displaySystem)
		}
	}
}

// runDisplayLogs starts STATUS SYNC logging at a given interval.
// It should be run as a goroutine.
// eg. --log-status="sync=42" logs SYNC information every 42 seconds
func runDisplayLogs(ctx *cli.Context, e *eth.Ethereum, tickerInterval time.Duration, handles displayEventHandlers) {

	// Set up ticker based on established interval.
	ticker := time.NewTicker(tickerInterval)

	var sigc = make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigc)

	//// Should listen for events.
	//// Proof of concept create event subscription
	var handledEvents []interface{}
	for _, h := range handles {
		handledEvents = append(handledEvents, h.ev)
	}
	ethEvents := e.EventMux().Subscribe(handledEvents...)

	handleDownloaderEvent := func(d interface{}) {
		switch d.(type) {
		case downloader.StartEvent:
			handles.runAllIfAny(ctx, e, d, tickerInterval, "DOWNLOADER_START")
		case downloader.DoneEvent:
			handles.runAllIfAny(ctx, e, d, tickerInterval, "DOWNLOADER_DONE")
		case downloader.FailedEvent:
			handles.runAllIfAny(ctx, e, d, tickerInterval, "DOWNLOADER_FAILED")
		}
	}

	go func() {
		for ev := range ethEvents.Chan() {
			switch d := ev.Data.(type) {
			case core.ChainInsertEvent:
				handles.runAllIfAny(ctx, e, ev.Data, tickerInterval, "CHAIN_INSERT")
			case core.ChainSideEvent:
				handles.runAllIfAny(ctx, e, ev.Data, tickerInterval, "CHAIN_SIDE")
			case core.HeaderChainInsertEvent:
				handles.runAllIfAny(ctx, e, ev.Data, tickerInterval, "HEADERCHAIN_INSERT")
			case core.NewMinedBlockEvent:
				handles.runAllIfAny(ctx, e, ev.Data, tickerInterval, "MINED_BLOCK")
			default:
				handleDownloaderEvent(d)
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			handles.runAllIfAny(ctx, e, nil, tickerInterval, "INTERVAL")
		case <-sigc:
			// Listen for interrupt
			ticker.Stop()
			glog.D(logger.Warn).Warnln("SYNC Stopping.")
			return
		}
	}
}
