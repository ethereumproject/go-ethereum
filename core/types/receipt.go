// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/rlp"
)

type ReceiptStatus byte

const (
	TxFailure       ReceiptStatus = 0
	TxSuccess       ReceiptStatus = 1
	TxStatusUnknown ReceiptStatus = 0xFF
)

// Receipt represents the results of a transaction.
type Receipt struct {
	// Consensus fields
	PostState         []byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	Logs              []*Log

	// Implementation fields
	TxHash          common.Hash
	ContractAddress common.Address
	GasUsed         *big.Int
	Status          ReceiptStatus
}

// NewReceipt creates a barebone transaction receipt, copying the init fields.
func NewReceipt(root []byte, failed bool, cumulativeGasUsed *big.Int) *Receipt {
	r := &Receipt{PostState: common.CopyBytes(root), CumulativeGasUsed: new(big.Int).Set(cumulativeGasUsed)}
	if failed {
		r.Status = TxFailure
	} else {
		r.Status = TxSuccess
	}
	return r
}

// EncodeRLP implements rlp.Encoder, and flattens the consensus fields of a receipt
// into an RLP stream.
func (r *Receipt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{r.PostState, r.CumulativeGasUsed, r.Bloom, r.Logs})
}

// DecodeRLP implements rlp.Decoder, and loads the consensus fields of a receipt
// from an RLP stream.
func (r *Receipt) DecodeRLP(s *rlp.Stream) error {
	var receipt struct {
		PostState         []byte
		CumulativeGasUsed *big.Int
		Bloom             Bloom
		Logs              []*Log
	}
	if err := s.Decode(&receipt); err != nil {
		return err
	}
	r.PostState, r.CumulativeGasUsed, r.Bloom, r.Logs = receipt.PostState, receipt.CumulativeGasUsed, receipt.Bloom, receipt.Logs
	return nil
}

// RlpEncode implements common.RlpEncode required for SHA3 derivation.
func (r *Receipt) RlpEncode() []byte {
	bytes, err := rlp.EncodeToBytes(r)
	if err != nil {
		panic(err)
	}
	return bytes
}

// String implements the Stringer interface.
func (r *Receipt) String() string {
	return fmt.Sprintf("receipt{med=%x cgas=%v bloom=%x logs=%v}", r.PostState, r.CumulativeGasUsed, r.Bloom, r.Logs)
}

// ReceiptForStorage is a wrapper around a Receipt that flattens and parses the
// entire content of a receipt, as opposed to only the consensus fields originally.
type ReceiptForStorage Receipt

// EncodeRLP implements rlp.Encoder, and flattens all content fields of a receipt
// into an RLP stream.
func (r *ReceiptForStorage) EncodeRLP(w io.Writer) error {
	logs := make([]*LogForStorage, len(r.Logs))
	for i, log := range r.Logs {
		logs[i] = (*LogForStorage)(log)
	}
	return rlp.Encode(w, []interface{}{r.PostState, r.CumulativeGasUsed, r.Bloom, r.TxHash, r.ContractAddress, logs, r.GasUsed, r.Status})
}

// DecodeRLP implements rlp.Decoder, and loads both consensus and implementation
// fields of a receipt from an RLP stream.
func (r *ReceiptForStorage) DecodeRLP(s *rlp.Stream) error {
	var oldReceipt struct {
		PostState         []byte
		CumulativeGasUsed *big.Int
		Bloom             Bloom
		TxHash            common.Hash
		ContractAddress   common.Address
		Logs              []*LogForStorage
		GasUsed           *big.Int
	}
	var receipt struct {
		PostState         []byte
		CumulativeGasUsed *big.Int
		Bloom             Bloom
		TxHash            common.Hash
		ContractAddress   common.Address
		Logs              []*LogForStorage
		GasUsed           *big.Int
		Status            ReceiptStatus
	}
	receipt.Status = TxStatusUnknown

	raw, err := s.Raw()
	if err != nil {
		return err
	}

	if err := s.Decode(&receipt); err != nil {
		//if !strings.HasPrefix(err.Error(), "rlp: too few elements for struct") {
		//	return err
		//}
		s.Reset(bytes.NewReader(raw), uint64(len(raw)))
		if err := s.Decode(&oldReceipt); err != nil {
			return err
		}
		receipt.PostState = oldReceipt.PostState
		receipt.CumulativeGasUsed = oldReceipt.CumulativeGasUsed
		receipt.Bloom = oldReceipt.Bloom
		receipt.TxHash = oldReceipt.TxHash
		receipt.ContractAddress = oldReceipt.ContractAddress
		receipt.Logs = oldReceipt.Logs
		receipt.GasUsed = oldReceipt.GasUsed
		receipt.Status = TxStatusUnknown

	}
	// Assign the consensus fields
	r.PostState, r.CumulativeGasUsed, r.Bloom = receipt.PostState, receipt.CumulativeGasUsed, receipt.Bloom
	r.Logs = make([]*Log, len(receipt.Logs))
	for i, log := range receipt.Logs {
		r.Logs[i] = (*Log)(log)
	}
	// Assign the implementation fields
	r.TxHash, r.ContractAddress, r.GasUsed, r.Status = receipt.TxHash, receipt.ContractAddress, receipt.GasUsed, receipt.Status

	return nil
}

// Receipts is a wrapper around a Receipt array to implement types.DerivableList.
type Receipts []*Receipt

// Len returns the number of receipts in this list.
func (r Receipts) Len() int { return len(r) }

// GetRlp returns the RLP encoding of one receipt from the list.
func (r Receipts) GetRlp(i int) []byte {
	bytes, err := rlp.EncodeToBytes(r[i])
	if err != nil {
		panic(err)
	}
	return bytes
}
