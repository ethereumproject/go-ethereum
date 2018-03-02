package core

import (
	"github.com/ethereumproject/go-ethereum/logger"
)

var mlogBlockchain = logger.MLogRegisterAvailable("blockchain", mLogLinesBlockchain)
var mlogTxPool = logger.MLogRegisterAvailable("txpool", mLogLinesTxPool)

// mLogLines is an private slice of all available mlog LINES.
// May be used for automatic mlog documentation generator, or
// for API usage/display/documentation otherwise.
var mLogLinesBlockchain = []logger.MLogT{
	*mlogBlockchainWriteBlock,
	*mlogBlockchainInsertBlocks,
}

var mLogLinesTxPool = []logger.MLogT{
	*mlogTxPoolAddTx,
	*mlogTxPoolValidateTx,
}

// Collect and document available mlog lines.

var mlogBlockchainWriteBlock = &logger.MLogT{
	Description: `Called when a single block written to the chain database.
A STATUS of NONE means it was written _without_ any abnormal chain event, such as a split.`,
	Receiver: "BLOCKCHAIN",
	Verb:     "WRITE",
	Subject:  "BLOCK",
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
		{"BLOCK", "DIFFICULTY", "BIGINT"},
		{"BLOCK", "UNCLES", "INT"},
		{"BLOCK", "RECEIVED_AT", "BIGINT"},
		{"BLOCK", "DIFF_PARENT_TIME", "BIGINT"},
	},
}

var mlogBlockchainInsertBlocks = &logger.MLogT{
	Description: "Called when a chain of blocks is inserted into the chain database.",
	Receiver:    "BLOCKCHAIN",
	Verb:        "INSERT",
	Subject:     "BLOCKS",
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

var mlogBlockchainReorgBlocks = &logger.MLogT{
	Description: "Called when a chain split is detected and a subset of blocks are reoganized.",
	Receiver: "BLOCKCHAIN",
	Verb: "REORG",
	Subject: "BLOCKS",
	Details: []logger.MLogDetailT{
		{"REORG", "LAST_COMMON_HASH", "STRING"},
		{"REORG", "SPLIT_NUMBER", "BIGINT"},
		{"BLOCKS", "OLD_START_HASH", "STRING"},
		{"BLOCKS", "NEW_START_HASH", "STRING"},
	},
}

var mlogTxPoolAddTx = &logger.MLogT{
	Description: `Called once when a valid transaction is added to tx pool.
$TO.NAME will be the account address hex or '[NEW_CONTRACT]' in case of a contract.`,
	Receiver: "TXPOOL",
	Verb:     "ADD",
	Subject:  "TX",
	Details: []logger.MLogDetailT{
		{"TX", "FROM", "STRING"},
		{"TX", "TO", "STRING"},
		{"TX", "VALUE", "BIGINT"},
		{"TX", "HASH", "STRING"},
	},
}

var mlogTxPoolValidateTx = &logger.MLogT{
	Description: `Called once when validating a single transaction.
If transaction is invalid, TX.ERROR will be non-nil, otherwise it will be nil.`,
	Receiver: "TXPOOL",
	Verb:     "VALIDATE",
	Subject:  "TX",
	Details: []logger.MLogDetailT{
		{"TX", "HASH", "STRING"},
		{"TX", "ERROR", "STRING_OR_NULL"},
	},
}
