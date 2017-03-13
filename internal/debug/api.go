// Copyright 2016 The go-ethereum Authors
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

// Package debug interfaces Go runtime debugging facilities.
// This package is mostly glue code making these facilities available
// through the CLI and RPC subsystem. If you want to use them from Go code,
// use package runtime instead.
package debug

import (
	"io"
	"sync"

	"github.com/ethereumproject/go-ethereum/logger/glog"
)

// Handler is the global debugging handler.
var Handler = new(HandlerT)

// HandlerT implements the debugging API.
// Do not create values of this type, use the one
// in the Handler variable instead.
type HandlerT struct {
	mu        sync.Mutex
	cpuW      io.WriteCloser
	cpuFile   string
	traceW    io.WriteCloser
	traceFile string
}

// Verbosity sets the glog verbosity ceiling.
// The verbosity of individual packages and source files
// can be raised using Vmodule.
func (*HandlerT) Verbosity(level int) {
	glog.SetV(level)
}

// Vmodule sets the glog verbosity pattern. See package
// glog for details on pattern syntax.
func (*HandlerT) Vmodule(pattern string) error {
	return glog.GetVModule().Set(pattern)
}

// BacktraceAt sets the glog backtrace location.
// See package glog for details on pattern syntax.
func (*HandlerT) BacktraceAt(location string) error {
	return glog.GetTraceLocation().Set(location)
}
