package miner

import "github.com/ethereumproject/go-ethereum/logger"

var mlogMiner = logger.MLogRegisterAvailable("miner", mlogMinerLines)

var mlogMinerLines = []logger.MLogT{
	*mlogMinerStart,
	*mlogMinerStop,
	*mlogMinerCommitWorkBlock,
	*mlogMinerCommitUncle,
	*mlogMinerCommitTx,
	*mlogMinerMineBlock,
	*mlogMinerConfirmMinedBlock,
}

var mlogMinerStart = &logger.MLogT{
	Description: `Called once when the miner starts.`,
	Receiver:    "MINER",
	Verb:        "START",
	Subject:     "COINBASE",
	Details: []logger.MLogDetailT{
		{"COINBASE", "ADDRESS", "STRING"},
		{"MINER", "THREADS", "INT"},
	},
}

var mlogMinerStop = &logger.MLogT{
	Description: `Called once when the miner stops.`,
	Receiver:    "MINER",
	Verb:        "STOP",
	Subject:     "COINBASE",
	Details: []logger.MLogDetailT{
		{"COINBASE", "ADDRESS", "STRING"},
		{"MINER", "THREADS", "INT"},
	},
}

var mlogMinerCommitWorkBlock = &logger.MLogT{
	Description: `Called when the miner commits new work on a block.`,
	Receiver:    "MINER",
	Verb:        "COMMIT_WORK",
	Subject:     "BLOCK",
	Details: []logger.MLogDetailT{
		{"BLOCK", "NUMBER", "BIGINT"},
		{"COMMIT_WORK", "TXS_COUNT", "INT"},
		{"COMMIT_WORK", "UNCLES_COUNT", "INT"},
		{"COMMIT_WORK", "TIME_ELAPSED", "DURATION"},
	},
}

var mlogMinerCommitUncle = &logger.MLogT{
	Description: `Called when the miner commits an uncle.
If $COMMIT_UNCLE is non-nil, uncle is not committed.`,
	Receiver: "MINER",
	Verb:     "COMMIT",
	Subject:  "UNCLE",
	Details: []logger.MLogDetailT{
		{"UNCLE", "BLOCK_NUMBER", "BIGINT"},
		{"UNCLE", "HASH", "STRING"},
		{"COMMIT", "ERROR", "STRING_OR_NULL"},
	},
}

var mlogMinerCommitTx = &logger.MLogT{
	Description: `Called when the miner commits (or attempts to commit) a transaction.
If $COMMIT.ERROR is non-nil, the transaction is not committed.`,
	Receiver: "MINER",
	Verb:     "COMMIT",
	Subject:  "TX",
	Details: []logger.MLogDetailT{
		{"COMMIT", "BLOCK_NUMBER", "BIGINT"},
		{"TX", "HASH", "STRING"},
		{"COMMIT", "ERROR", "STRING"},
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
		{"BLOCK", "NUMBER", "BIGINT"},
		{"BLOCK", "HASH", "STRING"},
		{"BLOCK", "STATUS", "STRING"},
		{"BLOCK", "WAIT_CONFIRM", "INT"},
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
		{"BLOCK", "NUMBER", "INT"},
	},
}
