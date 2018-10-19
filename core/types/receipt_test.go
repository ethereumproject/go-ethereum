package types

import (
	"bufio"
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/rlp"
)

func encodeReceipt(r *Receipt) ([]byte, error) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	err := r.EncodeRLP(writer)
	writer.Flush()

	return buf.Bytes(), err
}

func TestEIP658RLPRoundTrip1(t *testing.T) {
	// EIP-658 enabled - PostState is nil, Status is present
	r1 := NewReceipt(nil, big.NewInt(4095))
	r1.Status = TxSuccess

	rlpData, err := encodeReceipt(r1)
	if err != nil {
		t.Error("unexpected error", err)
	}

	var r2 Receipt
	r2.DecodeRLP(rlp.NewStream(bytes.NewReader(rlpData), 0))

	if r1.Status != r2.Status {
		t.Errorf("invalid status: expected %v, got %v", r1.Status, r2.Status)
	}

}

func TestEIP658RLPRoundTrip2(t *testing.T) {
	// EIP-658 disabled - PostState AND Status are present in Receipt
	r1 := NewReceipt(common.Hash{}.Bytes(), big.NewInt(4095))
	for i := 0; i < len(r1.PostState); i++ {
		r1.PostState[i] = byte(i)
	}
	r1.Status = TxSuccess

	rlpData, err := encodeReceipt(r1)
	if err != nil {
		t.Error("unexpected error", err)
	}

	var r2 Receipt
	r2.DecodeRLP(rlp.NewStream(bytes.NewReader(rlpData), 0))

	same := len(r1.PostState) == len(r2.PostState)
	for i := range r1.PostState {
		same = same && r1.PostState[i] == r2.PostState[i]
	}

	if !same {
		t.Errorf("invalid PostState: expected %v, got %v", r1.PostState, r2.PostState)
	}
	if r2.Status != TxStatusUnknown {
		t.Errorf("invalid Status: expected 0xFF, got %v", r2.Status)
	}
}

func TestInvalidReceiptsEncoding(t *testing.T) {
	// case 1: invalid PostState
	r := NewReceipt(make([]byte, 7), big.NewInt(4095))
	_, err := encodeReceipt(r)
	if err == nil {
		t.Error("error was expected")
	} else if strings.Index(err.Error(), "PostState") == -1 || strings.Index(err.Error(), "length") == -1 {
		t.Error("probably invalid error message:", err)
	}

	// case 2: no PostState (EIP-658), unknown transaction status
	r = NewReceipt(nil, big.NewInt(4095))
	_, err = encodeReceipt(r)
	if err == nil {
		t.Error("error was expected")
	} else if strings.Index(err.Error(), "PostState") == -1 || strings.Index(err.Error(), "Status") == -1 || strings.Index(err.Error(), "unknown") == -1 {
		t.Error("probably invalid error message:", err)
	}
}
