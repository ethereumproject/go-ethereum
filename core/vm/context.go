
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
	ClassicVm Type = iota	// classic VM (legacy implementation from ethereum)
	SputnikVm				// modern VM
	OtherVm					// other external VM connected through RPC
)

// ManagerType says how to manage VM
type ManagerType byte
const (
	ManageLocalVm ManagerType = iota // Automaticaly start/fork VM process and manage it
	UseExternalVm	 // VM process managed separately
)

// ConnectionType says how to connect to VM
type ConnectionType byte
const (
	LocalVm ConnectionType = iota  // VM is in the same process
	LocalIpcVm 		// VM is in the other copy of the process
	LocalRpcVm		// VM is another executable listening on specified host:port
	RemoteRpcVm	    // VM is started on other host
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
	Inactive	  // vm is inactive
	Running       // vm is running
	RequireErr    // vm requires some information
	TransferErr   // account has insufficient balance and transfer is not possible
	OutOfGasErr   // out of gas error occurred
	BadCodeErr    // bad contract code
	ExitedErr     // unspecified error occured
	Broken        // connection with vm is broken or vm is not response
)

type Fork byte
const (
	Frontier Fork = iota
	Homestead
	Diehard
)

type KeyValue interface {
	Address() common.Address
	Key() common.Hash
	Value() common.Hash
}

type Account interface {
	Address() common.Address
	Nonce() uint64
	Balance() *big.Int
}

type RequireID byte
const (
	RequireAccount RequireID = iota
	RequireCode
	RequireHash
	RequireInfo
	RequireValue
)

type Require struct {
	ID RequireID
	Address common.Address
	Hash    common.Hash
	Number  uint64
}

type Context interface {
	// Commit an account information
	CommitAccount(address common.Address, nonce uint64, balance *big.Int) error
	// Commit a block hash
	CommitBlockhash(number uint64,hash common.Hash) error
	// Commit a contract code
	CommitCode(address common.Address, hash common.Hash, code []byte) error
	// Commit an info
	CommitInfo(blockNumber uint64, coinbase common.Address, table *GasTable, fork Fork, difficulty, gasLimit, time *big.Int) error
	// Commit a state
	CommitValue(address common.Address, key common.Hash, value common.Hash) error
	// Finish VM context
	Finish() error
	// Returns the changed accounts information
	Modified() ([]Account, error)
	// Returns all addresses of removed accounts
	Removed() ([]common.Address, error)
	// Returns the created accounts code
	Code(common.Address) (common.Hash, []byte, error)
	// Returns modified stored values
	Values() ([]KeyValue, error)
	// Returns new contract address on Create and called contract address on Call
	Address() (common.Address, error)
	// Returns the out value and gas refund if any
	Out() ([]byte, *big.Int, error)
	// Returns logs to be appended to the current block if the user
	// decided to accept the running status of this VM
	Logs() (state.Logs, error)
	// Returns the current status of the VM
	Status() Status
	// Returns the current error in details
	Err() error
	// Run instructions until it reaches a `RequireErr` or exits
	Fire() *Require
}

const (
	TestFeaturesDisabled = iota*2
	TestSkipTransfer
	TestSkipSubCall
	TestSkipCreate
)

const AllTestFeatures = TestSkipTransfer | TestSkipSubCall | TestSkipCreate

type Machine interface {
	// Test feature
	SetTestFeatures(int) error
	// Call contract in new VM context
	Call(caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (Context, error)
	// Create contract in new VM context
	Create(caller common.Address, code []byte, gas, price, value *big.Int) (Context, error)
	// Type of VM
	Type() Type
	// Name of VM
	Name() string
}

var (
	BrokenError = errors.New("VM context broken")
)
