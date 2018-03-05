package eth

import "github.com/ethereumproject/go-ethereum/logger"

var mlogWwireProtocol = logger.MLogRegisterAvailable("wire", mlogLines)

var mlogLines = []*logger.MLogT{
	mlogWireReceiveTransfer,
	mlogWireSendTransfer,
}

var mlogWireReceiveTransfer = &logger.MLogT{
	Description: "Called once for each Wire Protocol event.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: "TRANSFER",
	Details: []logger.MLogDetailT{
		{Owner: "TRANSFER", Key: "CODE", Value: "INT"},
		{Owner: "TRANSFER", Key: "NAME", Value: "STRING"},
		{Owner: "TRANSFER", Key: "SIZE", Value: "INT"}, // size in bytes
		{Owner: "RECEIVE", Key: "REMOTE_ID", Value: "STRING"},
		{Owner: "RECEIVE", Key: "REMOTE_ADDR", Value: "STRING"},
		{Owner: "RECEIVE", Key: "REMOTE_VERSION", Value: "STRING"},
	},
}

var mlogWireSendTransfer = &logger.MLogT{
	Description: "Called once for each Wire Protocol event.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: "TRANSFER",
	Details: []logger.MLogDetailT{
		{Owner: "TRANSFER", Key: "CODE", Value: "INT"},
		{Owner: "TRANSFER", Key: "NAME", Value: "STRING"},
		{Owner: "TRANSFER", Key: "SIZE", Value: "INT"}, // size in bytes
		{Owner: "RECEIVE", Key: "REMOTE_ID", Value: "STRING"},
		{Owner: "RECEIVE", Key: "REMOTE_ADDR", Value: "STRING"},
		{Owner: "RECEIVE", Key: "REMOTE_VERSION", Value: "STRING"},
	},
}