package miner

import "github.com/eth-classic/go-ethereum/logger"

var mlogMiner = logger.MLogRegisterAvailable("miner", mlogMinerLines)

var mlogMinerLines = []*logger.MLogT{
	mlogMinerStart,
	mlogMinerStop,
	mlogMinerCommitWorkBlock,
	mlogMinerCommitUncle,
	mlogMinerCommitTx,
	mlogMinerMineBlock,
	mlogMinerConfirmMinedBlock,
	mlogMinerSubmitWork,
}

var mlogMinerStart = &logger.MLogT{
	Description: `Called once when the miner starts.`,
	Receiver:    "MINER",
	Verb:        "START",
	Subject:     "COINBASE",
	Details: []logger.MLogDetailT{
		{Owner: "COINBASE", Key: "ADDRESS", Value: "STRING"},
		{Owner: "MINER", Key: "THREADS", Value: "INT"},
	},
}

var mlogMinerStop = &logger.MLogT{
	Description: `Called once when the miner stops.`,
	Receiver:    "MINER",
	Verb:        "STOP",
	Subject:     "COINBASE",
	Details: []logger.MLogDetailT{
		{Owner: "COINBASE", Key: "ADDRESS", Value: "STRING"},
		{Owner: "MINER", Key: "THREADS", Value: "INT"},
	},
}

var mlogMinerCommitWorkBlock = &logger.MLogT{
	Description: `Called when the miner commits new work on a block.`,
	Receiver:    "MINER",
	Verb:        "COMMIT_WORK",
	Subject:     "BLOCK",
	Details: []logger.MLogDetailT{
		{Owner: "BLOCK", Key: "NUMBER", Value: "BIGINT"},
		{Owner: "COMMIT_WORK", Key: "TXS_COUNT", Value: "INT"},
		{Owner: "COMMIT_WORK", Key: "UNCLES_COUNT", Value: "INT"},
		{Owner: "COMMIT_WORK", Key: "TIME_ELAPSED", Value: "DURATION"},
	},
}

var mlogMinerCommitUncle = &logger.MLogT{
	Description: `Called when the miner commits an uncle.
If $COMMIT_UNCLE is non-nil, uncle is not committed.`,
	Receiver: "MINER",
	Verb:     "COMMIT",
	Subject:  "UNCLE",
	Details: []logger.MLogDetailT{
		{Owner: "UNCLE", Key: "BLOCK_NUMBER", Value: "BIGINT"},
		{Owner: "UNCLE", Key: "HASH", Value: "STRING"},
		{Owner: "COMMIT", Key: "ERROR", Value: "STRING_OR_NULL"},
	},
}

var mlogMinerCommitTx = &logger.MLogT{
	Description: `Called when the miner commits (or attempts to commit) a transaction.
If $COMMIT.ERROR is non-nil, the transaction is not committed.`,
	Receiver: "MINER",
	Verb:     "COMMIT",
	Subject:  "TX",
	Details: []logger.MLogDetailT{
		{Owner: "COMMIT", Key: "BLOCK_NUMBER", Value: "BIGINT"},
		{Owner: "TX", Key: "HASH", Value: "STRING"},
		{Owner: "COMMIT", Key: "ERROR", Value: "STRING"},
	},
}

var mlogMinerMineBlock = &logger.MLogT{
	Description: `Called when the miner has mined a block.
$BLOCK.STATUS will be either 'stale' or 'wait_confirm'. $BLOCK.WAIT_CONFIRM is an integer
specifying _n_ blocks to wait for confirmation based on block depth, relevant only for when $BLOCK.STATUS is 'wait_confirm'.`,
	Receiver: "MINER",
	Verb:     "MINE",
	Subject:  "BLOCK",
	Details: []logger.MLogDetailT{
		{Owner: "BLOCK", Key: "NUMBER", Value: "BIGINT"},
		{Owner: "BLOCK", Key: "HASH", Value: "STRING"},
		{Owner: "BLOCK", Key: "STATUS", Value: "STRING"},
		{Owner: "BLOCK", Key: "WAIT_CONFIRM", Value: "INT"},
	},
}

var mlogMinerConfirmMinedBlock = &logger.MLogT{
	Description: `Called once to confirm the miner has mined a block _n_ blocks back,
where _n_ refers to the $BLOCK.WAIT_CONFIRM value from MINER MINE BLOCK.
This is only a logging confirmation message, and is not related to work done.`,
	Receiver: "MINER",
	Verb:     "CONFIRM_MINED",
	Subject:  "BLOCK",
	Details: []logger.MLogDetailT{
		{Owner: "BLOCK", Key: "NUMBER", Value: "INT"},
	},
}

var mlogMinerSubmitWork = &logger.MLogT{
	Description: `Called when a remote agent submits work to the miner. Note that this method only
acknowledges that work was indeed submitted, but does not confirm nor deny that the PoW was correct.`,
	Receiver: "REMOTE_AGENT",
	Verb:     "SUBMIT",
	Subject:  "WORK",
	Details: []logger.MLogDetailT{
		{Owner: "WORK", Key: "NONCE", Value: "INT"},
		{Owner: "WORK", Key: "MIX_DIGEST", Value: "STRING"},
		{Owner: "WORK", Key: "HASH", Value: "STRING"},
		{Owner: "WORK", Key: "EXISTS", Value: "BOOL"},
	},
}
