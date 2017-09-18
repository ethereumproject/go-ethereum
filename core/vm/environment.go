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

package vm

import (
	"math/big"
	"reflect"
	"errors"

	"github.com/ethereumproject/go-ethereum/common"
)

// Type is the VM type
type Type byte

const (
	ClassicVmTy Type = iota // Default standard VM
	SputnikVmTy
	MaxVmTy
)

// RuleSet is an interface that defines the current rule set during the
// execution of the EVM instructions (e.g. whether it's homestead)
type RuleSet interface {
	IsHomestead(*big.Int) bool
	// GasTable returns the gas prices for this phase, which is based on
	// block number passed in.
	GasTable(*big.Int) *GasTable
}

// ContractRef is a reference to the contract's backing object
type ContractRef interface {
	ReturnGas(*big.Int, *big.Int)
	Address() common.Address
	Value() *big.Int
	SetCode(common.Hash, []byte)
	ForEachStorage(callback func(key, value common.Hash) bool)
}

// Environment is an EVM requirement and helper which allows access to outside
// information such as states.
type Environment interface {
	// The current ruleset
	RuleSet() RuleSet
	// The state database
	Db() Database
	// Creates a restorable snapshot
	SnapshotDatabase() int
	// Set database to previous snapshot
	RevertToSnapshot(int)
	// Address of the original invoker (first occurrence of the VM invoker)
	Origin() common.Address
	// The block number this VM is invoked on
	BlockNumber() *big.Int
	// The n'th hash ago from this block number
	GetHash(uint64) common.Hash
	// The handler's address
	Coinbase() common.Address
	// The current time (block time)
	Time() *big.Int
	// Difficulty set on the current block
	Difficulty() *big.Int
	// The gas limit of the block
	GasLimit() *big.Int
	// Determines whether it's possible to transact
	CanTransfer(from common.Address, balance *big.Int) bool
	// Transfers amount from one account to the other
	Transfer(from, to Account, amount *big.Int)
	// Adds a LOG to the state
	AddLog(*Log)
	// Get the curret calling depth
	Depth() int
	// Set the current calling depth
	SetDepth(i int)
	// Call another contract
	Call(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	// Take another's contract code and execute within our own context
	CallCode(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	// Same as CallCode except sender and value is propagated from parent to child scope
	DelegateCall(me ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error)
	// Create a new contract
	Create(me ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error)
	// VmType the type of VM
	VmType() Type
}

// Database is a EVM database for full state querying.
type Database interface {
	GetAccount(common.Address) Account
	CreateAccount(common.Address) Account

	AddBalance(common.Address, *big.Int)
	GetBalance(common.Address) *big.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCodeSize(common.Address) int
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)

	AddRefund(*big.Int)
	GetRefund() *big.Int

	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	// Exist reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(common.Address) bool
}

// Account represents a contract or basic ethereum account.
type Account interface {
	SubBalance(amount *big.Int)
	AddBalance(amount *big.Int)
	SetBalance(*big.Int)
	SetNonce(uint64)
	Balance() *big.Int
	Address() common.Address
	ReturnGas(*big.Int, *big.Int)
	SetCode(common.Hash, []byte)
	ForEachStorage(cb func(key, value common.Hash) bool)
	Value() *big.Int
}

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


var (
	OutOfGasError          = errors.New("Out of gas")
	CodeStoreOutOfGasError = errors.New("Contract creation code storage out of gas")
)

type Status byte

type VirtualMachine interface {
	// Commit an account information to this VM. This should only
	// be used when receiving `RequireError`.
	CommitAccount(commitment Account) error
	// Commit a block hash to this VM. This should only be used when
	// receiving `RequireError`.
	CommitBlockhash(number *big.Int, hash *big.Int) error
	// Returns the current status of the VM.
	Status() (Status, error)
	// Run one instruction and return. If it succeeds, VM status can
	// still be `Running`. If the call stack has more than one items,
	// this will only executes the last items' one single
	// instruction.
	Step() error
	// Run instructions until it reaches a `RequireError` or
	// exits. If this function succeeds, the VM status can only be
	// either `ExitedOk` or `ExitedErr`.
	Fire() error
	// Returns the changed or committed accounts information up to
	// current execution status.
	Accounts() (map[common.Address]Account, error)
	// Returns the out value, if any.
	Out() ([]byte, error)
	// Returns the available gas of this VM.
	AvailableGas() (*big.Int, error)
	// Returns the refunded gas of this VM.
	RefundedGas() (*big.Int, error)
	// Returns logs to be appended to the current block if the user
	// decided to accept the running status of this VM.
	Logs() ([]Log, error)
	// Returns all removed account addresses as for current VM execution.
	Removed() ([]common.Address, error)
}
