package p2p

import "github.com/eth-classic/go-ethereum/logger"

var mlogServer = logger.MLogRegisterAvailable("server", mLogLinesServer)

var mLogLinesServer = []*logger.MLogT{
	mlogServerPeerAdded,
	mlogServerPeerRemove,
}

var mlogServerPeerAdded = &logger.MLogT{
	Description: "Called once when a peer is added.",
	Receiver:    "SERVER",
	Verb:        "ADD",
	Subject:     "PEER",
	Details: []logger.MLogDetailT{
		{Owner: "SERVER", Key: "PEER_COUNT", Value: "INT"},
		{Owner: "PEER", Key: "ID", Value: "STRING"},
		{Owner: "PEER", Key: "REMOTE_ADDR", Value: "STRING"},
		{Owner: "PEER", Key: "REMOTE_VERSION", Value: "STRING"},
	},
}

var mlogServerPeerRemove = &logger.MLogT{
	Description: "Called once when a peer is removed.",
	Receiver:    "SERVER",
	Verb:        "REMOVE",
	Subject:     "PEER",
	Details: []logger.MLogDetailT{
		{Owner: "SERVER", Key: "PEER_COUNT", Value: "INT"},
		{Owner: "PEER", Key: "ID", Value: "STRING"},
		{Owner: "REMOVE", Key: "REASON", Value: "QUOTEDSTRING"},
	},
}
