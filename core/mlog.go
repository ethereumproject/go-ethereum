package core

import (
	"github.com/ethereumproject/go-ethereum/logger"
)

var mlogBlockchain = logger.MLogRegisterAvailable("blockchain", mLogLines)

// mLogLines is an private slice of all available mlog LINES.
// May be used for automatic mlog documentation generator, or
// for API usage/display/documentation otherwise.
var mLogLines = []logger.MLogT{
	mlogBlockchainWriteBlock,
	mlogBlockchainInsertBlocks,
}

// Collect and document available mlog lines.

var mlogBlockchainWriteBlock = logger.MLogT{
	Description: `Called when a single block written to the chain database.
A STATUS of NONE means it was written _without_ any abnormal chain event, such as a split.`,
	Receiver: "BLOCKCHAIN",
	Verb: "WRITE",
	Subject: "BLOCK",
	Details: []logger.MLogDetailT{
		{"WRITE", "STATUS", "STRING"},
		{"WRITE", "ERROR", "STRING_OR_NULL"},
		{"BLOCK", "NUMBER", "BIGINT"},
		{"BLOCK", "HASH", "STRING"},
		{"BLOCK", "SIZE", "INT64"},
		{"BLOCK", "TRANSACTIONS_COUNT", "INT"},
		{"BLOCK", "GAS_USED", "BIGINT"},
		{"BLOCK", "COINBASE", "STRING"},
		{"BLOCK", "TIME", "BIGINT"},
	},
}


var mlogBlockchainInsertBlocks = logger.MLogT{
	Description: "Called when a chain of blocks is inserted into the chain database.",
	Receiver: "BLOCKCHAIN",
	Verb: "INSERT",
	Subject: "BLOCKS",
	Details: []logger.MLogDetailT{
		{"BLOCKS", "COUNT", "INT"},
		{"BLOCKS", "QUEUED", "INT"},
		{"BLOCKS", "IGNORED", "INT"},
		{"BLOCKS", "TRANSACTIONS_COUNT", "INT"},
		{"BLOCKS", "LAST_NUMBER", "BIGINT"},
		{"BLOCKS", "FIRST_HASH", "STRING"},
		{"BLOCKS", "LAST_HASH", "STRING"},
		{"INSERT", "TIME", "DURATION"},
	},
}
