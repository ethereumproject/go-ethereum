package eth

import (
	"bytes"
	"github.com/eth-classic/go-ethereum/core/types"
	"github.com/eth-classic/go-ethereum/logger"
	"strings"
	"testing"
)

func TestMlogWireDelegateError(t *testing.T) {
	logger.SetMlogEnabled(true)
	logger.Reset()
	// Set up a log sys with a local buffer instead of file
	var b = new(bytes.Buffer)
	sys := logger.NewMLogSystem(b, 0, logger.LogLevel(1), false)
	logger.AddLogSystem(sys)

	logger.MLogRegisterActive("wire")

	// test with error
	err := errResp(ErrMsgTooLarge, "%v > %v", 42, ProtocolMaxMsgSize)
	mlogWireDelegate(nil, "receive", TxMsg, 42, "not castable to txs", err) // TODO: all msg codes

	logger.Flush() // wait for messages to be delivered

	if !strings.Contains(b.String(), errorToString[ErrMsgTooLarge]) {
		t.Errorf("got: %v, want: %v", b.String(), errorToString[ErrMsgTooLarge])
	}
	b.Reset()

	// test without error
	err = nil
	txs := []*types.Transaction{}                         // don't want to be nil
	mlogWireDelegate(nil, "receive", TxMsg, 42, txs, err) // TODO: all msg codes

	logger.Flush() // wait for messages to be delivered

	if strings.Contains(b.String(), errorToString[ErrMsgTooLarge]) {
		t.Errorf("got: %v, want: %v", errorToString[ErrMsgTooLarge], b.String())
	}
	b.Reset()
}
