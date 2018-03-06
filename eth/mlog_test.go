package eth

import (
	"testing"
	"github.com/ethereumproject/go-ethereum/core/types"
)

// TestMustGetLenIfPossible should test applicable possible values for aribtrary incoming data
// that could be cast as a slice for mlog length purposes.
func TestMustGetLenIfPossible(t *testing.T) {
	var txs types.Transactions
	if l := mustGetLenIfPossible(txs); l != 0 {
		t.Errorf("got: %v, want: %v", l, 0)
	}
}
