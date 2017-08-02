// Copyright 2017 The go-ethereum Authors
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


// This file 'mlog' is home to the 'discover' package implementation of mlog.
// All available mlog lines should be established here as variables and documented.
// For each instance of an mlog call, the respective MLogT variable SetDetailValues()
// method should be called with per-use instance details.

package discover

import (
	"github.com/ethereumproject/go-ethereum/logger"
	"sync"
)

var mlog *logger.Logger
var mlogOnce sync.Once

// initMLogging registers a logger for the discoverpackage
// It should only be called once.
// You can ensure this via:
// mlogOnce.Do(initMLogging) when the package is initialized
func initMLogging() {
	mlog = logger.NewLogger("discover")
	mlog.Infoln("[mlog] ON")
}

// Collect and document available mlog lines.

// mlogPingHandleFrom is called once for each ping request from a node FROM
var mlogPingHandleFrom = logger.MLogT{
	Receiver: "PING",
	Verb: "HANDLE",
	Subject: "FROM",
	Details: []logger.MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"PING", "EXPIRED", "BOOL"},
	},
}

// mlogPongHandleFrom is called once for each pong request from a node FROM
var mlogPongHandleFrom = logger.MLogT{
	Receiver: "PONG",
	Verb: "HANDLE",
	Subject: "FROM",
	Details: []logger.MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"PING", "EXPIRED", "BOOL"},
	},
}

// mlogFindNodeHandleFrom is called once for each findnode request from a node FROM
var mlogFindNodeHandleFrom = logger.MLogT{
	Receiver: "FIND_NODE",
	Verb: "HANDLE",
	Subject: "FROM",
	Details: []logger.MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"FIND_NODE", "EXPIRED", "BOOL"},
	},
}

// mlogFindNodeSendNeighbors is called once for each sent NEIGHBORS request
var mlogFindNodeSendNeighbors = logger.MLogT{
	Receiver: "FIND_NODE",
	Verb: "SEND",
	Subject: "NEIGHBORS",
	Details: []logger.MLogDetailT{
		{"FIND_NODE", "UDP_ADDRESS", "STRING"},
		{"FIND_NODE", "ID", "STRING"},
		{"NEIGHBORS", "CHUNK", "INT"},
		{"NEIGHBORS", "CHUNKS", "INT"},
		{"NEIGHBORS", "NODES_LEN", "INT"},
	},
}

// mlogNeighborsHandleFrom is called once for each neighbors request from a node FROM
var mlogNeighborsHandleFrom = logger.MLogT{
	Receiver: "NEIGHBORS",
	Verb: "HANDLE",
	Subject: "FROM",
	Details: []logger.MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"NEIGHBORS", "EXPIRED", "BOOL"},
	},
}