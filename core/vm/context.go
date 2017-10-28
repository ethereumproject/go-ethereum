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

	"errors"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
)

const UnknwonStringValue = "-"

// Type is the VM type
type Type byte

const (
	ClassicVm    Type = iota // classic VM (legacy implementation from ethereum)
	ClassicRawVm             // classic VM with direct usage (w/o RequireErr)
	SputnikVm                // modern embedded VM
	OtherVm                  // other external VM connected through RPC/IPC
)

func (tp Type) String() string {
	switch tp {
	case ClassicVm:
		return "ClassicVm"
	case ClassicRawVm:
		return "ClassicRawVm"
	case SputnikVm:
		return "SputnikVm"
	case OtherVm:
		return "OtherVm"
	default:
		return UnknwonStringValue
	}
}

// ManagerType says how to manage VM
type ManagerType byte

const (
	ManageVm ManagerType = iota // Automaticaly start/fork VM process and manage it
	UseVm                       // VM process managed separately
)

func (tp ManagerType) String() string {
	switch tp {
	case ManageVm:
		return "ManageVm"
	case UseVm:
		return "UseVm"
	default:
		return UnknwonStringValue
	}
}

// ConnectionType says how to connect to VM
type ConnectionType byte

const (
	InprocVm ConnectionType = iota // VM is in the same process
	LocalVm                        // VM is in the other copy of the process
	IpcVm                          // VM is another executable listening on specified port
	RpcVm                          // VM is started on other host
)

func (tp ConnectionType) String() string {
	switch tp {
	case InprocVm:
		return "InprocVm"
	case LocalVm:
		return "LocalVm"
	case IpcVm:
		return "IpcVm"
	case RpcVm:
		return "RpcVm"
	default:
		return UnknwonStringValue
	}
}

const (
	DefaultVm      = ClassicVm
	DefaultVmProto = InprocVm
	DefaultVmUsage = UseVm
	DefaultVmPort  = 30301
	DefaultVmHost  = "localhost"
	DefaultVmIpc   = "gethvm"
)

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
	ExitedOk    Status = iota
	Inactive           // vm is inactive
	Running            // vm is running
	RequireErr         // vm requires some information
	TransferErr        // account has insufficient balance and transfer is not possible
	OutOfGasErr        // out of gas error occurred
	ExitedErr          // unspecified error occured
	Broken             // connection with vm is broken or vm is not response
)

func (st Status) String() string {
	switch st {
	case ExitedOk:
		return "ExitedOk"
	case Inactive:
		return "Inactive"
	case Running:
		return "Running"
	case RequireErr:
		return "RequireError"
	case TransferErr:
		return "TransferErr"
	case OutOfGasErr:
		return "OutOfGasErr"
	case ExitedErr:
		return "ExitedErr"
	case Broken:
		return "Broken"
	default:
		return UnknwonStringValue
	}
}

type Fork byte

const (
	Frontier Fork = iota
	Homestead
	Diehard
)

func (fork Fork) String() string {
	switch fork {
	case Frontier:
		return "Frontier"
	case Homestead:
		return "Homestead"
	case Diehard:
		return "Diehard"
	default:
		return UnknwonStringValue
	}
}

type Account interface {
	Address() common.Address
	Nonce() uint64
	Balance() *big.Int
	IsSuicided() bool
	IsNewborn() bool
	Code() (common.Hash, []byte)
	Store(func(common.Hash, common.Hash) error) error
}

type RequireID byte

const (
	RequireAccount RequireID = iota
	RequireCode
	RequireHash
	RequireInfo
	RequireValue
)

func (id RequireID) String() string {
	switch id {
	case RequireAccount:
		return "RequireAccount"
	case RequireCode:
		return "RequireCode"
	case RequireHash:
		return "RequireHash"
	case RequireInfo:
		return "RequireInfo"
	case RequireValue:
		return "RequireValue"
	default:
		return UnknwonStringValue
	}
}

type Require struct {
	ID      RequireID
	Address common.Address
	Hash    common.Hash
	Number  uint64
}

type Context interface {
	// Commit an account information
	CommitAccount(address common.Address, nonce uint64, balance *big.Int) error
	// Commit a block hash
	CommitBlockhash(number uint64, hash common.Hash) error
	// Commit a contract code
	CommitCode(address common.Address, hash common.Hash, code []byte) error
	// Commit an info
	CommitInfo(blockNumber uint64, coinbase common.Address, table *GasTable, fork Fork, difficulty, gasLimit, time *big.Int) error
	// Commit a state
	CommitValue(address common.Address, key common.Hash, value common.Hash) error
	// Finish VM context
	Finish() error
	// Returns the changed accounts information
	Accounts() ([]Account, error)
	// Returns new contract address on Create and called contract address on Call
	Address() (common.Address, error)
	// Returns the out value and gas remaining/refunded if any
	Out() ([]byte, *big.Int, *big.Int, error)
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

type Machine interface {
	// Call contract in new VM context
	Call(caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (Context, error)
	// Create contract in new VM context
	Create(caller common.Address, code []byte, gas, price, value *big.Int) (Context, error)
	// Type of VM
	Type() Type
	// Name of VM
	Name() string
}

type MachineConn interface {
	// Close connection to machine
	Close()
	// Get Machine interface
	Machine() Machine
	// Check that connection is alive and machine is on duty
	IsOnDuty() bool
}

const (
	TestFeaturesDisabled = iota * 2
	TestSkipTransfer
	TestSkipSubCall
	TestSkipCreate
)

const AllTestFeatures = TestSkipTransfer | TestSkipSubCall | TestSkipCreate

type TestMachine interface {
	// Test feature
	SetTestFeatures(int) error
}

var (
	BrokenError = errors.New("VM context broken")
)
