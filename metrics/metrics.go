// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package metrics centralizes the registration.
package metrics

import (
	"encoding/json"
	"os"
	"runtime"
	"time"

	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/rcrowley/go-metrics"
)

// Reg is the metrics destination.
var reg = metrics.NewRegistry()

var (
	MsgTXNIn           = metrics.NewRegisteredMeter("msg/txn/in", reg)
	MsgTXNInBytes      = metrics.NewRegisteredMeter("msg/txn/in/bytes", reg)
	MsgTXNOut          = metrics.NewRegisteredMeter("msg/txn/out", reg)
	MsgTXNOutBytes     = metrics.NewRegisteredMeter("msg/txn/out/bytes", reg)
	MsgHashIn          = metrics.NewRegisteredMeter("msg/hash/in", reg)
	MsgHashInBytes     = metrics.NewRegisteredMeter("msg/hash/out/bytes", reg)
	MsgHashOut         = metrics.NewRegisteredMeter("msg/hash/in", reg)
	MsgHashOutBytes    = metrics.NewRegisteredMeter("msg/hash/out/bytes", reg)
	MsgBlockIn         = metrics.NewRegisteredMeter("msg/block/in", reg)
	MsgBlockInBytes    = metrics.NewRegisteredMeter("msg/block/in/bytes", reg)
	MsgBlockOut        = metrics.NewRegisteredMeter("msg/block/out", reg)
	MsgBlockOutBytes   = metrics.NewRegisteredMeter("msg/block/out/bytes", reg)
	MsgHeaderIn        = metrics.NewRegisteredMeter("msg/header/in", reg)
	MsgHeaderInBytes   = metrics.NewRegisteredMeter("msg/header/in/bytes", reg)
	MsgHeaderOut       = metrics.NewRegisteredMeter("msg/header/out", reg)
	MsgHeaderOutBytes  = metrics.NewRegisteredMeter("msg/header/out/bytes", reg)
	MsgBodyIn          = metrics.NewRegisteredMeter("msg/body/in", reg)
	MsgBodyInBytes     = metrics.NewRegisteredMeter("msg/body/in/bytes", reg)
	MsgBodyOut         = metrics.NewRegisteredMeter("msg/body/out", reg)
	MsgBodyOutBytes    = metrics.NewRegisteredMeter("msg/body/out/bytes", reg)
	MsgStateIn         = metrics.NewRegisteredMeter("msg/state/in", reg)
	MsgStateInBytes    = metrics.NewRegisteredMeter("msg/state/in/bytes", reg)
	MsgStateOut        = metrics.NewRegisteredMeter("msg/state/out", reg)
	MsgStateOutBytes   = metrics.NewRegisteredMeter("msg/state/out/bytes", reg)
	MsgReceiptIn       = metrics.NewRegisteredMeter("msg/receipt/in", reg)
	MsgReceiptInBytes  = metrics.NewRegisteredMeter("msg/receipt/in/bytes", reg)
	MsgReceiptOut      = metrics.NewRegisteredMeter("msg/receipt/out", reg)
	MsgReceiptOutBytes = metrics.NewRegisteredMeter("msg/receipt/out/bytes", reg)
	MsgMiscIn          = metrics.NewRegisteredMeter("msg/misc/in", reg)
	MsgMiscInBytes     = metrics.NewRegisteredMeter("msg/misc/in/bytes", reg)
	MsgMiscOut         = metrics.NewRegisteredMeter("msg/misc/out", reg)
	MsgMiscOutBytes    = metrics.NewRegisteredMeter("msg/misc/out/bytes", reg)
)

var (
	DLHeaders        = metrics.NewRegisteredMeter("download/header", reg)
	DLHeaderTimer    = metrics.NewRegisteredTimer("download/header", reg)
	DLHeaderDrops    = metrics.NewRegisteredMeter("download/header/drop", reg)
	DLHeaderTimeouts = metrics.NewRegisteredMeter("download/header/timeout", reg)

	DLBodies       = metrics.NewRegisteredMeter("download/body", reg)
	DLBodyTimer    = metrics.NewRegisteredTimer("download/body", reg)
	DLBodyDrops    = metrics.NewRegisteredMeter("download/body/drop", reg)
	DLBodyTimeouts = metrics.NewRegisteredMeter("download/body/timeout", reg)

	DLReceipts        = metrics.NewRegisteredMeter("download/receipt", reg)
	DLReceiptTimer    = metrics.NewRegisteredTimer("download/receipt", reg)
	DLReceiptDrops    = metrics.NewRegisteredMeter("download/receipt/drop", reg)
	DLReceiptTimeouts = metrics.NewRegisteredMeter("download/receipt/timeout", reg)

	DLStates        = metrics.NewRegisteredMeter("download/state", reg)
	DLStateTimer    = metrics.NewRegisteredTimer("download/state", reg)
	DLStateDrops    = metrics.NewRegisteredMeter("download/state/drop", reg)
	DLStateTimeouts = metrics.NewRegisteredMeter("download/state/timeout", reg)
)

var (
	FetchBlocks  = metrics.NewRegisteredMeter("fetch/block", reg)
	FetchHeaders = metrics.NewRegisteredMeter("fetch/header", reg)
	FetchBodies  = metrics.NewRegisteredMeter("fetch/body", reg)

	FetchFilterBlockIns   = metrics.NewRegisteredMeter("fetch/filter/block/in", reg)
	FetchFilterBlockOuts  = metrics.NewRegisteredMeter("fetch/filter/block/out", reg)
	FetchFilterHeaderIns  = metrics.NewRegisteredMeter("fetch/filter/header/in", reg)
	FetchFilterHeaderOuts = metrics.NewRegisteredMeter("fetch/filter/header/out", reg)
	FetchFilterBodyIns    = metrics.NewRegisteredMeter("fetch/filter/body/in", reg)
	FetchFilterBodyOuts   = metrics.NewRegisteredMeter("fetch/filter/body/out", reg)

	FetchAnnounces     = metrics.NewRegisteredMeter("fetch/announce", reg)
	FetchAnnounceTimer = metrics.NewRegisteredTimer("fetch/announce", reg)
	FetchAnnounceDrops = metrics.NewRegisteredMeter("fetch/announce/drop", reg)
	FetchAnnounceDOS   = metrics.NewRegisteredMeter("fetch/announce/dos", reg)

	FetchBroadcasts     = metrics.NewRegisteredMeter("fetch/broadcast", reg)
	FetchBroadcastTimer = metrics.NewRegisteredTimer("fetch/broadcast", reg)
	FetchBroadcastDrops = metrics.NewRegisteredMeter("fetch/broadcast/drop", reg)
	FetchBroadcastDOS   = metrics.NewRegisteredMeter("fetch/broadcast/dos", reg)
)

var (
	P2PIn       = metrics.NewRegisteredMeter("p2p/in", reg)
	P2PInBytes  = metrics.NewRegisteredMeter("p2p/in/bytes", reg)
	P2POut      = metrics.NewRegisteredMeter("p2p/out", reg)
	P2POutBytes = metrics.NewRegisteredMeter("p2p/out/bytes", reg)
)

// Collect writes metrics to the given destination.
func Collect(dest string) {
	const interval = 3 * time.Second

	go collectProcessMetrics(interval)

	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		glog.Fatal(err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	ticks := time.Tick(interval)
	for _ = range ticks {
		if err := encoder.Encode(reg); err != nil {
			glog.Errorf("metrics: log to %q: %s", dest, err)
		}
	}
}

// collectProcessMetrics periodically collects various metrics about the running
// process.
func collectProcessMetrics(refresh time.Duration) {
	// Create the various data collectors
	memstats := make([]*runtime.MemStats, 2)
	diskstats := make([]*DiskStats, 2)
	for i := 0; i < len(memstats); i++ {
		memstats[i] = new(runtime.MemStats)
		diskstats[i] = new(DiskStats)
	}
	// Define the various metrics to collect
	memAllocs := metrics.GetOrRegisterMeter("system/memory/allocs", reg)
	memFrees := metrics.GetOrRegisterMeter("system/memory/frees", reg)
	memInuse := metrics.GetOrRegisterMeter("system/memory/inuse", reg)
	memPauses := metrics.GetOrRegisterMeter("system/memory/pauses", reg)

	var diskReads, diskReadBytes, diskWrites, diskWriteBytes metrics.Meter
	if err := ReadDiskStats(diskstats[0]); err == nil {
		diskReads = metrics.GetOrRegisterMeter("system/disk/readcount", reg)
		diskReadBytes = metrics.GetOrRegisterMeter("system/disk/readdata", reg)
		diskWrites = metrics.GetOrRegisterMeter("system/disk/writecount", reg)
		diskWriteBytes = metrics.GetOrRegisterMeter("system/disk/writedata", reg)
	} else {
		glog.V(logger.Debug).Infof("failed to read disk metrics: %v", err)
	}
	// Iterate loading the different stats and updating the meters
	for i := 1; ; i++ {
		runtime.ReadMemStats(memstats[i%2])
		memAllocs.Mark(int64(memstats[i%2].Mallocs - memstats[(i-1)%2].Mallocs))
		memFrees.Mark(int64(memstats[i%2].Frees - memstats[(i-1)%2].Frees))
		memInuse.Mark(int64(memstats[i%2].Alloc - memstats[(i-1)%2].Alloc))
		memPauses.Mark(int64(memstats[i%2].PauseTotalNs - memstats[(i-1)%2].PauseTotalNs))

		if ReadDiskStats(diskstats[i%2]) == nil {
			diskReads.Mark(int64(diskstats[i%2].ReadCount - diskstats[(i-1)%2].ReadCount))
			diskReadBytes.Mark(int64(diskstats[i%2].ReadBytes - diskstats[(i-1)%2].ReadBytes))
			diskWrites.Mark(int64(diskstats[i%2].WriteCount - diskstats[(i-1)%2].WriteCount))
			diskWriteBytes.Mark(int64(diskstats[i%2].WriteBytes - diskstats[(i-1)%2].WriteBytes))
		}
		time.Sleep(refresh)
	}
}
