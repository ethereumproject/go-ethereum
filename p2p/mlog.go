package p2p

import "github.com/ethereumproject/go-ethereum/logger"

var mlogServer = logger.MLogRegisterAvailable("server", mLogLines)

var mLogLines = []logger.MLogT{
	*mlogServerPeerAdded,
	*mlogServerPeerRemove,
}

var mlogServerPeerAdded = &logger.MLogT{
	Description: "Called once when a peer is added.",
	Receiver:    "SERVER",
	Verb:        "ADD",
	Subject:     "PEER",
	Details: []logger.MLogDetailT{
		{"SERVER", "PEER_COUNT", "INT"},
		{"PEER", "ID", "STRING"},
		{"PEER", "REMOTE_ADDR", "STRING"},
		{"PEER", "REMOTE_VERSION", "STRING"},
	},
}

var mlogServerPeerRemove = &logger.MLogT{
	Description: "Called once when a peer is removed.",
	Receiver:    "SERVER",
	Verb:        "REMOVE",
	Subject:     "PEER",
	Details: []logger.MLogDetailT{
		{"SERVER", "PEER_COUNT", "INT"},
		{"PEER", "ID", "STRING"},
		{"REMOVE", "REASON", "QUOTEDSTRING"},
	},
}
