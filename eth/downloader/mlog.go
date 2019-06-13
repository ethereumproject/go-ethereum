package downloader

import "github.com/eth-classic/go-ethereum/logger"

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
		{Owner: "PEER", Key: "ID", Value: "STRING"},
		{Owner: "PEER", Key: "VERSION", Value: "INT"},
		{Owner: "REGISTER", Key: "ERROR", Value: "STRING_OR_NULL"},
	},
}

var mlogDownloaderUnregisterPeer = &logger.MLogT{
	Description: "Called when the downloader unregisters a peer.",
	Receiver:    "DOWNLOADER",
	Verb:        "UNREGISTER",
	Subject:     "PEER",
	Details: []logger.MLogDetailT{
		{Owner: "PEER", Key: "ID", Value: "STRING"},
		{Owner: "UNREGISTER", Key: "ERROR", Value: "STRING_OR_NULL"},
	},
}

var mlogDownloaderTuneQOS = &logger.MLogT{
	Description: `Called at intervals to gather peer latency statistics. Estimates request round trip time.

RTT reports the estimated Request Round Trip time, confidence is measures from 0-1 (1 is ultimately confidenct),
and TTL reports the Timeout Allowance for a single downloader request to finish within.`,
	Receiver: "DOWNLOADER",
	Verb:     "TUNE",
	Subject:  "QOS",
	Details: []logger.MLogDetailT{
		{Owner: "QOS", Key: "RTT", Value: "DURATION"},
		{Owner: "QOS", Key: "CONFIDENCE", Value: "NUMBER"},
		{Owner: "QOS", Key: "TTL", Value: "DURATION"},
	},
}

var mlogDownloaderStartSync = &logger.MLogT{
	Description: `Called when the downloader initiates synchronisation with a peer.`,
	Receiver:    "DOWNLOADER",
	Verb:        "START",
	Subject:     "SYNC",
	Details: []logger.MLogDetailT{
		{Owner: "DOWNLOADER", Key: "MODE", Value: "STRING"},
		{Owner: "SYNC", Key: "PEER_ID", Value: "STRING"},
		{Owner: "SYNC", Key: "PEER_NAME", Value: "STRING"},
		{Owner: "SYNC", Key: "PEER_VERSION", Value: "NUMBER"},
		{Owner: "SYNC", Key: "HASH", Value: "STRING"},
		{Owner: "SYNC", Key: "TD", Value: "BIGINT"},
	},
}

var mlogDownloaderStopSync = &logger.MLogT{
	Description: `Called when the downloader terminates synchronisation with a peer. If ERROR is null, synchronisation was sucessful.`,
	Receiver:    "DOWNLOADER",
	Verb:        "STOP",
	Subject:     "SYNC",
	Details: []logger.MLogDetailT{
		{Owner: "DOWNLOADER", Key: "MODE", Value: "STRING"},
		{Owner: "SYNC", Key: "PEER_ID", Value: "STRING"},
		{Owner: "SYNC", Key: "PEER_NAME", Value: "STRING"},
		{Owner: "SYNC", Key: "PEER_VERSION", Value: "NUMBER"},
		{Owner: "SYNC", Key: "PEER_HASH", Value: "STRING"},
		{Owner: "SYNC", Key: "PEER_TD", Value: "BIGINT"},

		{Owner: "SYNC", Key: "PIVOT", Value: "NUMBER"},
		{Owner: "SYNC", Key: "ORIGIN", Value: "NUMBER"},
		{Owner: "SYNC", Key: "HEIGHT", Value: "NUMBER"},
		{Owner: "SYNC", Key: "ELAPSED", Value: "DURATION"},
		{Owner: "SYNC", Key: "ERR", Value: "STRING_OR_NULL"},
	},
}
