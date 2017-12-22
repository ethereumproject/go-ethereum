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
// For each instance of an mlog call, the respective MLogT variable AssignDetails()
// method should be called with per-use instance details.

package discover

import (
	"github.com/ethereumproject/go-ethereum/logger"
)

var mlogDiscover = logger.MLogRegisterAvailable("discover", mLogLines)

// mLogLines is a private slice of all available mlog LINES.
// May be used for automatic mlog docmentation generator, or
// for API usage/display/documentation otherwise.
var mLogLines = []logger.MLogT{
	*mlogPingHandleFrom,
	*mlogPingSendTo,
	*mlogPongHandleFrom,
	*mlogPongSendTo,
	*mlogFindNodeHandleFrom,
	*mlogFindNodeSendTo,
	*mlogNeighborsHandleFrom,
	*mlogNeighborsSendTo,
}

// Collect and document available mlog lines.

// PING
// mlogPingHandleFrom is called once for each ping request from a node FROM
var mlogPingHandleFrom = &logger.MLogT{
	Description: "Called once for each received PING request from peer FROM.",
	Receiver:    "PING",
	Verb:        "HANDLE",
	Subject:     "FROM",
	Details: []logger.MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"PING", "BYTES_TRANSFERRED", "INT"},
	},
}

var mlogPingSendTo = &logger.MLogT{
	Description: "Called once for each outgoing PING request to peer TO.",
	Receiver:    "PING",
	Verb:        "SEND",
	Subject:     "TO",
	Details: []logger.MLogDetailT{
		{"TO", "UDP_ADDRESS", "STRING"},
		{"PING", "BYTES_TRANSFERRED", "INT"},
	},
}

// PONG
// mlogPongHandleFrom is called once for each pong request from a node FROM
var mlogPongHandleFrom = &logger.MLogT{
	Description: "Called once for each received PONG request from peer FROM.",
	Receiver:    "PONG",
	Verb:        "HANDLE",
	Subject:     "FROM",
	Details: []logger.MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"PONG", "BYTES_TRANSFERRED", "INT"},
	},
}

// mlogPingHandleFrom is called once for each ping request from a node FROM
var mlogPongSendTo = &logger.MLogT{
	Description: "Called once for each outgoing PONG request to peer TO.",
	Receiver:    "PONG",
	Verb:        "SEND",
	Subject:     "TO",
	Details: []logger.MLogDetailT{
		{"TO", "UDP_ADDRESS", "STRING"},
		{"PONG", "BYTES_TRANSFERRED", "INT"},
	},
}

// FINDNODE
// mlogFindNodeHandleFrom is called once for each findnode request from a node FROM
var mlogFindNodeHandleFrom = &logger.MLogT{
	Description: "Called once for each received FIND_NODE request from peer FROM.",
	Receiver:    "FINDNODE",
	Verb:        "HANDLE",
	Subject:     "FROM",
	Details: []logger.MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"FINDNODE", "BYTES_TRANSFERRED", "INT"},
	},
}

// mlogFindNodeHandleFrom is called once for each findnode request from a node FROM
var mlogFindNodeSendTo = &logger.MLogT{
	Description: "Called once for each received FIND_NODE request from peer FROM.",
	Receiver:    "FINDNODE",
	Verb:        "SEND",
	Subject:     "TO",
	Details: []logger.MLogDetailT{
		{"TO", "UDP_ADDRESS", "STRING"},
		{"FINDNODE", "BYTES_TRANSFERRED", "INT"},
	},
}

// NEIGHBORS
// mlogNeighborsHandleFrom is called once for each neighbors request from a node FROM
var mlogNeighborsHandleFrom = &logger.MLogT{
	Description: `Called once for each received NEIGHBORS request from peer FROM.`,
	Receiver:    "NEIGHBORS",
	Verb:        "HANDLE",
	Subject:     "FROM",
	Details: []logger.MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"NEIGHBORS", "BYTES_TRANSFERRED", "INT"},
	},
}

// mlogFindNodeSendNeighbors is called once for each sent NEIGHBORS request
var mlogNeighborsSendTo = &logger.MLogT{
	Description: `Called once for each outgoing NEIGHBORS request to peer TO.`,
	Receiver:    "NEIGHBORS",
	Verb:        "SEND",
	Subject:     "TO",
	Details: []logger.MLogDetailT{
		{"TO", "UDP_ADDRESS", "STRING"},
		{"NEIGHBORS", "BYTES_TRANSFERRED", "INT"},
	},
}
