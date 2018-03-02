package fetcher

import "github.com/ethereumproject/go-ethereum/logger"

var mlogFetcher = logger.MLogRegisterAvailable("fetcher", mLogLines)

var mLogLines = []logger.MLogT{
	*mlogFetcherDiscardAnnouncement,
}

var mlogFetcherDiscardAnnouncement = &logger.MLogT{
	Description: "Called when a block announcement is discarded.",
	Receiver:    "FETCHER",
	Verb:        "DISCARD",
	Subject:     "ANNOUNCEMENT",
	Details: []logger.MLogDetailT{
		{"ANNOUNCEMENT", "ORIGIN", "STRING"},
		{"ANNOUNCEMENT", "NUMBER", "INT"},
		{"ANNOUNCEMENT", "HASH", "STRING"},
		{"ANNOUNCEMENT", "DISTANCE", "INT"},
	},
}
