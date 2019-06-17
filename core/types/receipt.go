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

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/core/vm"
	"github.com/eth-classic/go-ethereum/rlp"
)

var (
	receiptStatusFailedRLP     = []byte{}
	receiptStatusSuccessfulRLP = []byte{0x01}
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
	Logs              vm.Logs

	// Implementation fields
	TxHash          common.Hash
	ContractAddress common.Address
	GasUsed         *big.Int
	Status          ReceiptStatus
}

// storedReceiptRLP is the storage encoding of a receipt.
type storedReceiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	TxHash            common.Hash
	ContractAddress   common.Address
	Logs              []*vm.LogForStorage
	GasUsed           *big.Int
}

// storedReceiptRLP is the storage encoding of a receipt.
type oldStoredReceiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed *big.Int
	Bloom             Bloom
	TxHash            common.Hash
	ContractAddress   common.Address
	Logs              []*vm.LogForStorage
	GasUsed           *big.Int
	Status            ReceiptStatus
}

// NewReceipt creates a barebone transaction receipt, copying the init fields.
func NewReceipt(root []byte, cumulativeGasUsed *big.Int) *Receipt {
	return &Receipt{PostState: common.CopyBytes(root), CumulativeGasUsed: new(big.Int).Set(cumulativeGasUsed), Status: TxStatusUnknown}
}

// EncodeRLP implements rlp.Encoder, and flattens the consensus fields of a receipt
// into an RLP stream.
func (r *Receipt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{r.statusEncoding(), r.CumulativeGasUsed, r.Bloom, r.Logs})
}

// DecodeRLP implements rlp.Decoder, and loads the consensus fields of a receipt
// from an RLP stream.
func (r *Receipt) DecodeRLP(s *rlp.Stream) error {
	var receipt struct {
		PostStateOrStatus []byte
		CumulativeGasUsed *big.Int
		Bloom             Bloom
		Logs              vm.Logs
	}
	if err := s.Decode(&receipt); err != nil {
		return err
	}

	if err := r.setStatus(receipt.PostStateOrStatus); err != nil {
		return err
	}

	r.CumulativeGasUsed, r.Bloom, r.Logs = receipt.CumulativeGasUsed, receipt.Bloom, receipt.Logs
	return nil
}

func (r *Receipt) statusEncoding() []byte {
	if len(r.PostState) == 0 {
		if r.Status == TxFailure {
			return receiptStatusFailedRLP
		}
		return receiptStatusSuccessfulRLP
	}
	return r.PostState
}

func (r *Receipt) setStatus(postStateOrStatus []byte) error {
	switch {
	case bytes.Equal(postStateOrStatus, receiptStatusSuccessfulRLP):
		r.Status = TxSuccess
	case bytes.Equal(postStateOrStatus, receiptStatusFailedRLP):
		r.Status = TxFailure
	case len(postStateOrStatus) == len(common.Hash{}):
		r.PostState = postStateOrStatus
	default:
		return fmt.Errorf("invalid receipt status %x", postStateOrStatus)
	}
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
	logs := make([]*vm.LogForStorage, len(r.Logs))
	for i, log := range r.Logs {
		logs[i] = (*vm.LogForStorage)(log)
	}
	receiptToStore := &storedReceiptRLP{
		PostStateOrStatus: (*Receipt)(r).statusEncoding(),
		CumulativeGasUsed: r.CumulativeGasUsed,
		Logs:              logs,
		Bloom:             r.Bloom,
		TxHash:            r.TxHash,
		ContractAddress:   r.ContractAddress,
		GasUsed:           r.GasUsed,
	}
	return rlp.Encode(w, receiptToStore)
}

// DecodeRLP implements rlp.Decoder, and loads both consensus and implementation
// fields of a receipt from an RLP stream.
func (r *ReceiptForStorage) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}

	// Try decoding the receipt without Status first
	if err := decodeStoredReceiptRLP(r, raw); err == nil {
		return nil
	}

	return decodeOldStoredReceiptRLP(r, raw)
}

// Decode stored receipt
func decodeStoredReceiptRLP(r *ReceiptForStorage, raw []byte) error {
	var receipt storedReceiptRLP
	if err := rlp.DecodeBytes(raw, &receipt); err != nil {
		return err
	}

	r.CumulativeGasUsed = receipt.CumulativeGasUsed
	r.Bloom = receipt.Bloom
	r.TxHash = receipt.TxHash
	r.ContractAddress = receipt.ContractAddress
	r.GasUsed = receipt.GasUsed

	r.Logs = make(vm.Logs, len(receipt.Logs))
	for i, log := range receipt.Logs {
		r.Logs[i] = (*vm.Log)(log)
	}

	if err := (*Receipt)(r).setStatus(receipt.PostStateOrStatus); err != nil {
		return err
	}

	return nil
}

// Decode with status field included in storage
func decodeOldStoredReceiptRLP(r *ReceiptForStorage, raw []byte) error {
	var receipt oldStoredReceiptRLP
	if err := rlp.DecodeBytes(raw, &receipt); err != nil {
		return err
	}

	r.PostState = receipt.PostStateOrStatus
	r.Status = receipt.Status
	r.CumulativeGasUsed = receipt.CumulativeGasUsed
	r.Bloom = receipt.Bloom
	r.TxHash = receipt.TxHash
	r.ContractAddress = receipt.ContractAddress
	r.GasUsed = receipt.GasUsed

	r.Logs = make(vm.Logs, len(receipt.Logs))
	for i, log := range receipt.Logs {
		r.Logs[i] = (*vm.Log)(log)
	}

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
