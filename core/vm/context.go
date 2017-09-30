
// Copyright 2017 (c) ETCDEV Team

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

package vm

import (
	"math/big"
	"reflect"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"errors"
)

// Type is the VM type
type Type byte
const (
	// classic VM
	ClassicVm Type = iota
	// sputnik VM
	SputnikVm
	// other external VM connected through RPC
	OtherVm
)

const (
	ManageLocalVm = iota
	UseExternalVm
)
const (
	LocalVm = iota
	LocalIpcVm
	LocalRpcVm
	RemoteRpcVm
)

const DefaultVm      = ClassicVm
const DefaultVmProto = LocalVm
const DefaultVmUsage = ManageLocalVm
const DefaultVmPort  = 30301
const DefaultVmHost  = "localhost"

type GasTable struct {
	ExtcodeSize *big.Int
	ExtcodeCopy *big.Int
	Balance     *big.Int
	SLoad       *big.Int
	Calls       *big.Int
	Suicide     *big.Int
	ExpByte     *big.Int

	// CreateBySuicide occurs when the
	// refunded account is one that does
	// not exist. This logic is similar
	// to call. May be left nil. Nil means
	// not charged.
	CreateBySuicide *big.Int
}

// IsEmpty return true if all values are zero values,
// which useful for checking JSON-decoded empty state.
func (g *GasTable) IsEmpty() bool {
	return reflect.DeepEqual(g, GasTable{})
}

type Status byte

const (
	ExitedOk Status = iota
	Running       // vm is running
	RequireErr    // vm requires some information
	TransferErr   // account has insufficient balance and transfer is not possible
	OutOfGas      // out of gas error occurred
	BadCode       // bad contract code
	Terminated    // vm context terminated
	Broken        // connection with vm is broken or vm is not response
)

type Fork byte

const (
	Frontier Fork = iota
	Homestead
	Diehard
)

type Account interface {
	Address() common.Address
	Nonce() uint64
	Balance() *big.Int
	Code() []byte
	CodeHash() common.Hash
}

type RequireID byte

const (
	RequireAccount RequireID = iota
	RequireCode
	RequireHash
	RequireRules
)

type Require struct {
	ID RequireID
	Address common.Address
	Number  uint64
}

type Debugger interface {
	// Run one instruction and return. If it succeeds, VM status can
	// still be `Running`.
	Step() *Require
}

type Context interface {
	// Commit an account information to this VM. This should only
	// be used when receiving `RequireAccountError`.
	CommitAccount(address common.Address, nonce uint64, balance *big.Int) error
	// Commit a block hash to this VM. This should only be used when
	// receiving `RequireBlockhashError`.
	CommitBlockhash(number uint64,hash common.Hash) error
	// Commit a contract code to this VM. This should only be used when
	// receiving `RequireCodeError`.
	CommitCode(address common.Address, hash common.Hash, code []byte) error
	// Commit a contract code to this VM. This should only be used when
	// receiving `RequireRulesError`.
	CommitRules(number uint64, table *GasTable, fork Fork) error
	// Finish VM context
	Finish() error
	// Returns the changed or committed accounts information up to
	// current execution status.
	Accounts() ([]Account, error)
	// Returns new contract address on Create and called contract address on Call.
	Address() (common.Address, error)
	// Returns the out value, if any.
	Out() ([]byte, error)
	// Returns the available gas of this VM.
	AvailableGas() (*big.Int, error)
	// Returns the refunded gas of this VM.
	RefundedGas() (*big.Int, error)
	// Returns logs to be appended to the current block if the user
	// decided to accept the running status of this VM.
	Logs() (state.Logs, error)
	// Returns all removed account addresses as for current VM execution.
	Removed() ([]common.Address, error)
	// Debug VM
	Debug() (Debugger, error)
	// Type of VM
	Type() Type
	// Name of VM
	Name() string
	// Returns the current status of the VM.
	Status() Status
	// Returns the current error in details.
	Err() error
	// Run instructions until it reaches a `RequireErr` or
	// exits.
	Fire() *Require
}

type Machine interface {
	// Call contract in new VM context
	Call(blockNumber uint64, caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (Context, error)
	// Create contract in new VM context
	Create(blockNumber uint64, caller common.Address, code []byte, gas, price, value *big.Int) (Context, error)
}

var (
	TerminatedError = errors.New("VM context terminated")
	BrokenError = errors.New("VM context broken")
)