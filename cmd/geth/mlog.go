package main

import (
	"github.com/ethereumproject/go-ethereum/logger"
)

var mlogClient = logger.MLogRegisterAvailable("client", mlogLinesClient)

var mlogLinesClient = []*logger.MLogT{
	mlogClientStartup,
	mlogClientShutdown,
}

var clientDetails = []logger.MLogDetailT{
	{"CLIENT", "SERVER_ID", "STRING"},
	{"CLIENT", "SERVER_NAME", "STRING"},
	{"CLIENT", "SERVER_ENODE", "STRING"},
	{"CLIENT", "SERVER_IP", "STRING"},
	{"CLIENT", "SERVER_MAXPEERS", "INT"},

	{"CLIENT", "CONFIG_CHAINNAME", "STRING"},
	{"CLIENT", "CONFIG_CHAINID", "INT"},
	{"CLIENT", "CONFIG_NETWORK", "INT"},
}

var mlogClientStartup = &logger.MLogT{
	Description: `Called when the geth client starts up.`,
	Receiver: "CLIENT",
	Verb:     "START",
	Subject:  "SESSION",
	Details: clientDetails,
}

var mlogClientShutdown = &logger.MLogT{
	Description: "Called when the geth client shuts down because of an interceptable signal, eg. SIGINT.",
	Receiver:    "CLIENT",
	Verb:        "STOP",
	Subject:     "SESSION",
	Details: append(clientDetails,
		[]logger.MLogDetailT{
			{"STOP", "SIGNAL", "STRING"},
			{"STOP", "ERROR", "STRING_OR_NULL"},
			{"CLIENT", "DURATION", "INT"}}...),
}