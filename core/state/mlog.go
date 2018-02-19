package state

import "github.com/ethereumproject/go-ethereum/logger"

var mlogState = logger.MLogRegisterAvailable("state", mlogStateLines)

var mlogStateLines = []logger.MLogT{
	*mlogStateCreateObject,
	*mlogStateAddBalanceObject,
	*mlogStateSubBalanceObject,
}

var mlogStateCreateObject = &logger.MLogT{
	Description: `Called once when a state object is created.
$OBJECT.NEW is the address of the newly-created object.
If there was an existing account with the same address, it is overwritten and its address returned as the $OBJECT.PREV value.`,
	Receiver: "STATE",
	Verb:     "CREATE",
	Subject:  "OBJECT",
	Details: []logger.MLogDetailT{
		{"OBJECT", "NEW", "STRING"},
		{"OBJECT", "PREV", "STRING"},
	},
}

var mlogStateAddBalanceObject = &logger.MLogT{
	Description: "Called once when the balance of an account (state object) is added to.",
	Receiver:    "STATE",
	Verb:        "ADD_BALANCE",
	Subject:     "OBJECT",
	Details: []logger.MLogDetailT{
		{"OBJECT", "ADDRESS", "STRING"},
		{"OBJECT", "NONCE", "INT"},
		{"OBJECT", "BALANCE", "BIGINT"},
		{"ADD_BALANCE", "AMOUNT", "BIGINT"},
	},
}

var mlogStateSubBalanceObject = &logger.MLogT{
	Description: "Called once when the balance of an account (state object) is subtracted from.",
	Receiver:    "STATE",
	Verb:        "SUB_BALANCE",
	Subject:     "OBJECT",
	Details: []logger.MLogDetailT{
		{"OBJECT", "ADDRESS", "STRING"},
		{"OBJECT", "NONCE", "INT"},
		{"OBJECT", "BALANCE", "BIGINT"},
		{"SUB_BALANCE", "AMOUNT", "BIGINT"},
	},
}
