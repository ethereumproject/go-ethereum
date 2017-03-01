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
	"bufio"
	"encoding/json"
	"os"
	"runtime"
	"time"

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

var (
	MemAllocs = metrics.GetOrRegisterGauge("memory/allocs", reg)
	MemFrees  = metrics.GetOrRegisterGauge("memory/frees", reg)
	MemInuse  = metrics.GetOrRegisterGauge("memory/inuse", reg)
	MemPauses = metrics.GetOrRegisterGauge("memory/pauses", reg)

	DiskReads      = metrics.GetOrRegisterGauge("disk/readcount", reg)
	DiskReadBytes  = metrics.GetOrRegisterGauge("disk/readdata", reg)
	DiskWrites     = metrics.GetOrRegisterGauge("disk/writecount", reg)
	DiskWriteBytes = metrics.GetOrRegisterGauge("disk/writedata", reg)
)

// diskStats is the per process disk I/O statistics.
type diskStats struct {
	ReadCount  int64 // Number of read operations executed
	ReadBytes  int64 // Total number of bytes read
	WriteCount int64 // Number of write operations executed
	WriteBytes int64 // Total number of byte written
}

// Collect writes metrics to the given file.
func Collect(file string) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		glog.Fatal(err)
	}
	defer f.Close()

	encoder := json.NewEncoder(bufio.NewWriter(f))

	for range time.Tick(3 * time.Second) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		MemAllocs.Update(int64(mem.Mallocs))
		MemFrees.Update(int64(mem.Frees))
		MemInuse.Update(int64(mem.Alloc))
		MemPauses.Update(int64(mem.PauseTotalNs))

		var disk diskStats
		readDiskStats(&disk)
		DiskReads.Update(disk.ReadCount)
		DiskReadBytes.Update(disk.ReadBytes)
		DiskWrites.Update(disk.WriteCount)
		DiskWriteBytes.Update(disk.WriteBytes)

		if err := encoder.Encode(reg); err != nil {
			glog.Errorf("metrics: log to %q: %s", file, err)
		}
	}
}
