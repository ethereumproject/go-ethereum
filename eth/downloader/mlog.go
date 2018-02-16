package downloader

import "github.com/ethereumproject/go-ethereum/logger"

var mlogDownloader = logger.MLogRegisterAvailable("downloader", mLogLines)

var mLogLines = []*logger.MLogT{
	mlogDownloaderRegisterPeer,
	mlogDownloaderUnregisterPeer,
	mlogDownloaderTuneQOS,
	mlogDownloaderStartSync,
	mlogDownloaderStopSync,
}

var mlogDownloaderRegisterPeer = &logger.MLogT{
	Description: "Called when the downloader registers a peer.",
	Receiver:    "DOWNLOADER",
	Verb:        "REGISTER",
	Subject:     "PEER",
	Details: []logger.MLogDetailT{
		{"PEER", "ID", "STRING"},
		{"PEER", "VERSION", "INT"},
		{"REGISTER", "ERROR", "STRING_OR_NULL"},
	},
}

var mlogDownloaderUnregisterPeer = &logger.MLogT{
	Description: "Called when the downloader unregisters a peer.",
	Receiver:    "DOWNLOADER",
	Verb:        "UNREGISTER",
	Subject:     "PEER",
	Details: []logger.MLogDetailT{
		{"PEER", "ID", "STRING"},
		{"UNREGISTER", "ERROR", "STRING_OR_NULL"},
	},
}

var mlogDownloaderTuneQOS = &logger.MLogT{
	Description: `Called at intervals to gather peer latency statistics. Estimates request round trip time.

RTT reports the estimated Request Round Trip time, confidence is measures from 0-1 (1 is ultimately confidenct),
and TTL reports the Timeout Allowance for a single downloader request to finish within.`,
	Receiver: "DOWNLOADER",
	Verb: "TUNE",
	Subject: "QOS",
	Details: []logger.MLogDetailT{
		{"QOS", "RTT", "DURATION"},
		{"QOS", "CONFIDENCE", "NUMBER"},
		{"QOS", "TTL", "DURATION"},
	},
}

var mlogDownloaderStartSync = &logger.MLogT{
	Description: `Called when the downloader initiates synchronisation with a peer.`,
	Receiver: "DOWNLOADER",
	Verb: "START",
	Subject: "SYNC",
	Details: []logger.MLogDetailT{
		{"DOWNLOADER", "MODE", "STRING"},
		{"SYNC", "PEER_ID", "STRING"},
		{"SYNC", "PEER_NAME", "STRING"},
		{"SYNC", "PEER_VERSION", "NUMBER"},
		{"SYNC", "HASH", "STRING"},
		{"SYNC", "TD", "BIGINT"},
	},
}

var mlogDownloaderStopSync = &logger.MLogT{
	Description: `Called when the downloader terminates synchronisation with a peer. If ERROR is null, synchronisation was sucessful.`,
	Receiver: "DOWNLOADER",
	Verb: "STOP",
	Subject: "SYNC",
	Details: []logger.MLogDetailT{
		{"DOWNLOADER", "MODE", "STRING"},
		{"SYNC", "PEER_ID", "STRING"},
		{"SYNC", "PEER_NAME", "STRING"},
		{"SYNC", "PEER_VERSION", "NUMBER"},
		{"SYNC", "PEER_HASH", "STRING"},
		{"SYNC", "PEER_TD", "BIGINT"},

		{"SYNC", "PIVOT", "NUMBER"},
		{"SYNC", "ORIGIN", "NUMBER"},
		{"SYNC", "HEIGHT", "NUMBER"},
		{"SYNC", "ELAPSED", "DURATION"},
		{"SYNC", "ERR", "STRING_OR_NULL"},
	},
}