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

package debug

import (
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
)

var (
	verbosityFlag = cli.GenericFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=core, 5=debug, 6=detail",
		Value: glog.GetVerbosity(),
	}
	vmoduleFlag = cli.GenericFlag{
		Name:  "vmodule",
		Usage: "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=6,p2p=5)",
		Value: glog.GetVModule(),
	}
	backtraceAtFlag = cli.GenericFlag{
		Name:  "backtrace",
		Usage: "Request a stack trace at a specific logging statement (e.g. \"block.go:271\")",
		Value: glog.GetTraceLocation(),
	}
)

// Flags holds all command-line flags required for debugging.
var Flags = []cli.Flag{
	verbosityFlag, vmoduleFlag, backtraceAtFlag,
}

// Setup initializes profiling and logging based on the CLI flags.
// It should be called as early as possible in the program.
func Setup(ctx *cli.Context) error {
	// logging
	glog.CopyStandardLogTo("INFO")
	glog.SetToStderr(true)

	return nil
}
