package fetcher

import "github.com/eth-classic/go-ethereum/logger"

var mlogFetcher = logger.MLogRegisterAvailable("fetcher", mLogLines)

var mLogLines = []*logger.MLogT{
	mlogFetcherDiscardAnnouncement,
}

var mlogFetcherDiscardAnnouncement = &logger.MLogT{
	Description: "Called when a block announcement is discarded.",
	Receiver:    "FETCHER",
	Verb:        "DISCARD",
	Subject:     "ANNOUNCEMENT",
	Details: []logger.MLogDetailT{
		{Owner: "ANNOUNCEMENT", Key: "ORIGIN", Value: "STRING"},
		{Owner: "ANNOUNCEMENT", Key: "NUMBER", Value: "INT"},
		{Owner: "ANNOUNCEMENT", Key: "HASH", Value: "STRING"},
		{Owner: "ANNOUNCEMENT", Key: "DISTANCE", Value: "INT"},
	},
}
