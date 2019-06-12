package main

import (
	"github.com/eth-classic/go-ethereum/logger"
)

var mlogClient = logger.MLogRegisterAvailable("client", mlogLinesClient)

var mlogLinesClient = []*logger.MLogT{
	mlogClientStartup,
	mlogClientShutdown,
}

var clientDetails = []logger.MLogDetailT{
	{Owner: "CLIENT", Key: "SERVER_ID", Value: "STRING"},
	{Owner: "CLIENT", Key: "SERVER_NAME", Value: "STRING"},
	{Owner: "CLIENT", Key: "SERVER_ENODE", Value: "STRING"},
	{Owner: "CLIENT", Key: "SERVER_IP", Value: "STRING"},
	{Owner: "CLIENT", Key: "SERVER_MAXPEERS", Value: "INT"},

	{Owner: "CLIENT", Key: "CONFIG_CHAINNAME", Value: "QUOTEDSTRING"},
	{Owner: "CLIENT", Key: "CONFIG_CHAINID", Value: "INT"},
	{Owner: "CLIENT", Key: "CONFIG_NETWORK", Value: "INT"},

	{Owner: "CLIENT", Key: "MLOG_COMPONENTS", Value: "STRING"},

	{Owner: "CLIENT", Key: "IDENTITY", Value: "OBJECT"},
}

var mlogClientStartup = &logger.MLogT{
	Description: `Called when the geth client starts up.`,
	Receiver:    "CLIENT",
	Verb:        "START",
	Subject:     "SESSION",
	Details:     clientDetails,
}

var mlogClientShutdown = &logger.MLogT{
	Description: "Called when the geth client shuts down because of an interceptable signal, eg. SIGINT.",
	Receiver:    "CLIENT",
	Verb:        "STOP",
	Subject:     "SESSION",
	Details: append(clientDetails,
		[]logger.MLogDetailT{
			{Owner: "STOP", Key: "SIGNAL", Value: "STRING"},
			{Owner: "STOP", Key: "ERROR", Value: "STRING_OR_NULL"},
			{Owner: "CLIENT", Key: "DURATION", Value: "INT"}}...),
}
