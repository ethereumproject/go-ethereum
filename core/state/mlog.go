package state

import "github.com/eth-classic/go-ethereum/logger"

var mlogState = logger.MLogRegisterAvailable("state", mlogStateLines)

var mlogStateLines = []*logger.MLogT{
	mlogStateCreateObject,
	mlogStateAddBalanceObject,
	mlogStateSubBalanceObject,
}

var mlogStateCreateObject = &logger.MLogT{
	Description: `Called once when a state object is created.
$OBJECT.NEW is the address of the newly-created object.
If there was an existing account with the same address, it is overwritten and its address returned as the $OBJECT.PREV value.`,
	Receiver: "STATE",
	Verb:     "CREATE",
	Subject:  "OBJECT",
	Details: []logger.MLogDetailT{
		{Owner: "OBJECT", Key: "NEW", Value: "STRING"},
		{Owner: "OBJECT", Key: "PREV", Value: "STRING"},
	},
}

var mlogStateAddBalanceObject = &logger.MLogT{
	Description: "Called once when the balance of an account (state object) is added to.",
	Receiver:    "STATE",
	Verb:        "ADD_BALANCE",
	Subject:     "OBJECT",
	Details: []logger.MLogDetailT{
		{Owner: "OBJECT", Key: "ADDRESS", Value: "STRING"},
		{Owner: "OBJECT", Key: "NONCE", Value: "INT"},
		{Owner: "OBJECT", Key: "BALANCE", Value: "BIGINT"},
		{Owner: "ADD_BALANCE", Key: "AMOUNT", Value: "BIGINT"},
	},
}

var mlogStateSubBalanceObject = &logger.MLogT{
	Description: "Called once when the balance of an account (state object) is subtracted from.",
	Receiver:    "STATE",
	Verb:        "SUB_BALANCE",
	Subject:     "OBJECT",
	Details: []logger.MLogDetailT{
		{Owner: "OBJECT", Key: "ADDRESS", Value: "STRING"},
		{Owner: "OBJECT", Key: "NONCE", Value: "INT"},
		{Owner: "OBJECT", Key: "BALANCE", Value: "BIGINT"},
		{Owner: "SUB_BALANCE", Key: "AMOUNT", Value: "BIGINT"},
	},
}
