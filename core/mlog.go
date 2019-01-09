package core

import (
	"github.com/ethereumproject/go-ethereum/logger"
)

var mlogBlockchain = logger.MLogRegisterAvailable("blockchain", mLogLinesBlockchain)
var mlogHeaderchain = logger.MLogRegisterAvailable("headerchain", mLogLinesHeaderchain)
var mlogTxPool = logger.MLogRegisterAvailable("txpool", mLogLinesTxPool)

// mLogLines is an private slice of all available mlog LINES.
// May be used for automatic mlog documentation generator, or
// for API usage/display/documentation otherwise.
var mLogLinesBlockchain = []*logger.MLogT{
	mlogBlockchainWriteBlock,
	mlogBlockchainInsertBlocks,
	mlogBlockchainReorgBlocks,
}

var mLogLinesHeaderchain = []*logger.MLogT{
	mlogHeaderchainWriteHeader,
	mlogHeaderchainInsertHeaders,
}

var mLogLinesTxPool = []*logger.MLogT{
	mlogTxPoolAddTx,
	mlogTxPoolValidateTx,
}

// Collect and document available mlog lines.

var mlogBlockchainWriteBlock = &logger.MLogT{
	Description: `Called when a single block is written to the chain database.
A STATUS of NONE means it was written _without_ any abnormal chain event, such as a split.`,
	Receiver: "BLOCKCHAIN",
	Verb:     "WRITE",
	Subject:  "BLOCK",
	Details: []logger.MLogDetailT{
		{Owner: "WRITE", Key: "STATUS", Value: "STRING"},
		{Owner: "WRITE", Key: "ERROR", Value: "STRING_OR_NULL"},
		{Owner: "BLOCK", Key: "NUMBER", Value: "BIGINT"},
		{Owner: "BLOCK", Key: "HASH", Value: "STRING"},
		{Owner: "BLOCK", Key: "SIZE", Value: "INT64"},
		{Owner: "BLOCK", Key: "TRANSACTIONS_COUNT", Value: "INT"},
		{Owner: "BLOCK", Key: "GAS_USED", Value: "BIGINT"},
		{Owner: "BLOCK", Key: "COINBASE", Value: "STRING"},
		{Owner: "BLOCK", Key: "TIME", Value: "BIGINT"},
		{Owner: "BLOCK", Key: "DIFFICULTY", Value: "BIGINT"},
		{Owner: "BLOCK", Key: "UNCLES", Value: "INT"},
		{Owner: "BLOCK", Key: "RECEIVED_AT", Value: "BIGINT"},
		{Owner: "BLOCK", Key: "DIFF_PARENT_TIME", Value: "BIGINT"},
	},
}

var mlogBlockchainInsertBlocks = &logger.MLogT{
	Description: "Called when a chain of blocks is inserted into the chain database.",
	Receiver:    "BLOCKCHAIN",
	Verb:        "INSERT",
	Subject:     "BLOCKS",
	Details: []logger.MLogDetailT{
		{Owner: "BLOCKS", Key: "COUNT", Value: "INT"},
		{Owner: "BLOCKS", Key: "QUEUED", Value: "INT"},
		{Owner: "BLOCKS", Key: "IGNORED", Value: "INT"},
		{Owner: "BLOCKS", Key: "TRANSACTIONS_COUNT", Value: "INT"},
		{Owner: "BLOCKS", Key: "LAST_NUMBER", Value: "BIGINT"},
		{Owner: "BLOCKS", Key: "FIRST_HASH", Value: "STRING"},
		{Owner: "BLOCKS", Key: "LAST_HASH", Value: "STRING"},
		{Owner: "INSERT", Key: "TIME", Value: "DURATION"},
	},
}

var mlogBlockchainReorgBlocks = &logger.MLogT{
	Description: "Called when a chain split is detected and a subset of blocks are reoganized.",
	Receiver:    "BLOCKCHAIN",
	Verb:        "REORG",
	Subject:     "BLOCKS",
	Details: []logger.MLogDetailT{
		{Owner: "REORG", Key: "LAST_COMMON_HASH", Value: "STRING"},
		{Owner: "REORG", Key: "SPLIT_NUMBER", Value: "BIGINT"},
		{Owner: "BLOCKS", Key: "OLD_START_HASH", Value: "STRING"},
		{Owner: "BLOCKS", Key: "NEW_START_HASH", Value: "STRING"},
	},
}

// Headerchain
var mlogHeaderchainWriteHeader = &logger.MLogT{
	Description: `Called when a single header is written to the chain header database.
A STATUS of NONE means it was written _without_ any abnormal chain event, such as a split.`,
	Receiver: "HEADERCHAIN",
	Verb:     "WRITE",
	Subject:  "HEADER",
	Details: []logger.MLogDetailT{
		{Owner: "WRITE", Key: "STATUS", Value: "STRING"},
		{Owner: "WRITE", Key: "ERROR", Value: "STRING_OR_NULL"},
		{Owner: "HEADER", Key: "NUMBER", Value: "BIGINT"},
		{Owner: "HEADER", Key: "HASH", Value: "STRING"},
		{Owner: "HEADER", Key: "GAS_USED", Value: "BIGINT"},
		{Owner: "HEADER", Key: "COINBASE", Value: "STRING"},
		{Owner: "HEADER", Key: "TIME", Value: "BIGINT"},
		{Owner: "HEADER", Key: "DIFFICULTY", Value: "BIGINT"},
		{Owner: "HEADER", Key: "DIFF_PARENT_TIME", Value: "BIGINT"},
	},
}

var mlogHeaderchainInsertHeaders = &logger.MLogT{
	Description: "Called when a chain of headers is inserted into the chain header database.",
	Receiver:    "HEADERCHAIN",
	Verb:        "INSERT",
	Subject:     "HEADERS",
	Details: []logger.MLogDetailT{
		{Owner: "HEADERS", Key: "COUNT", Value: "INT"},
		{Owner: "HEADERS", Key: "IGNORED", Value: "INT"},
		{Owner: "HEADERS", Key: "LAST_NUMBER", Value: "BIGINT"},
		{Owner: "HEADERS", Key: "FIRST_HASH", Value: "STRING"},
		{Owner: "HEADERS", Key: "LAST_HASH", Value: "STRING"},
		{Owner: "INSERT", Key: "TIME", Value: "DURATION"},
	},
}

var mlogTxPoolAddTx = &logger.MLogT{
	Description: `Called once when a valid transaction is added to tx pool.
$TO.NAME will be the account address hex or '[NEW_CONTRACT]' in case of a contract.`,
	Receiver: "TXPOOL",
	Verb:     "ADD",
	Subject:  "TX",
	Details: []logger.MLogDetailT{
		{Owner: "TX", Key: "FROM", Value: "STRING"},
		{Owner: "TX", Key: "TO", Value: "STRING"},
		{Owner: "TX", Key: "VALUE", Value: "BIGINT"},
		{Owner: "TX", Key: "HASH", Value: "STRING"},
	},
}

var mlogTxPoolValidateTx = &logger.MLogT{
	Description: `Called once when validating a single transaction.
If transaction is invalid, TX.ERROR will be non-nil, otherwise it will be nil.`,
	Receiver: "TXPOOL",
	Verb:     "VALIDATE",
	Subject:  "TX",
	Details: []logger.MLogDetailT{
		{Owner: "TX", Key: "HASH", Value: "STRING"},
		{Owner: "TX", Key: "SIZE", Value: "QUOTEDSTRING"},
		{Owner: "TX", Key: "DATA_SIZE", Value: "QUOTEDSTRING"},
		{Owner: "TX", Key: "NONCE", Value: "INT"},
		{Owner: "TX", Key: "GAS", Value: "BIGINT"},
		{Owner: "TX", Key: "GAS_PRICE", Value: "BIGINT"},
		{Owner: "TX", Key: "VALID", Value: "BOOL"},
		{Owner: "TX", Key: "ERROR", Value: "STRING_OR_NULL"},
	},
}
