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

package classic

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
)

// Database is a EVM database for full state querying.
type Database interface {
	GetAccount(common.Address) state.AccountObject
	CreateAccount(common.Address) state.AccountObject

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

// Environment is an EVM requirement and helper which allows access to outside
// information such as states.
type Environment interface {
	// The current ruleset
	RuleSet() vm.RuleSet
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
	Transfer(from, to state.AccountObject, amount *big.Int)
	// Adds a LOG to the state
	AddLog(*state.Log)
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
}

type VMEnv struct {
	rules     vm.RuleSet
	state     *state.StateDB           // State to use for executing
	evm       *EVM                     // The Ethereum Virtual Machine
	depth     int                      // Current execution depth
	header    *types.Header            // Header information
	getHashFn func(uint64) common.Hash // getHashFn callback is used to retrieve block hashes
	from      common.Address
	value     *big.Int
}

func NewEnv(
	state *state.StateDB, rules vm.RuleSet, header *types.Header,
	from common.Address, value *big.Int,
	hashFn func(uint64) common.Hash) (Environment, error) {

	env := &VMEnv{
		rules:     rules,
		state:     state,
		header:    header,
		getHashFn: hashFn,
		from:      from,
		value:     value,
	}

	env.evm = NewVM(env)
	return env, nil
}

func (self *VMEnv) RuleSet() vm.RuleSet {
	return self.rules
}

func (self *VMEnv) Origin() common.Address {
	return self.from
}

func (self *VMEnv) BlockNumber() *big.Int {
	return self.header.Number
}

func (self *VMEnv) Coinbase() common.Address {
	return self.header.Coinbase
}

func (self *VMEnv) Time() *big.Int {
	return self.header.Time
}

func (self *VMEnv) Difficulty() *big.Int {
	return self.header.Difficulty
}

func (self *VMEnv) GasLimit() *big.Int {
	return self.header.GasLimit
}

func (self *VMEnv) Value() *big.Int {
	return self.value
}

func (self *VMEnv) Db() Database {
	return self.state
}

func (self *VMEnv) Depth() int {
	return self.depth
}

func (self *VMEnv) SetDepth(i int) {
	self.depth = i
}

func (self *VMEnv) GetHash(n uint64) common.Hash {
	return self.getHashFn(n)
}

func (self *VMEnv) AddLog(log *state.Log) {
	self.state.AddLog(log)
}

func (self *VMEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	return self.state.GetBalance(from).Cmp(balance) >= 0
}

func (self *VMEnv) SnapshotDatabase() int {
	return self.state.Snapshot()
}

func (self *VMEnv) RevertToSnapshot(snapshot int) {
	self.state.RevertToSnapshot(snapshot)
}

func (self *VMEnv) Transfer(from, to state.AccountObject, amount *big.Int) {
	Transfer(from, to, amount)
}

func (self *VMEnv) Call(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return Call(self, me, addr, data, gas, price, value)
}

func (self *VMEnv) CallCode(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return CallCode(self, me, addr, data, gas, price, value)
}

func (self *VMEnv) DelegateCall(me ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return DelegateCall(self, me.(*Contract), addr, data, gas, price)
}

func (self *VMEnv) Create(me ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return Create(self, me, data, gas, price, value)
}

type EVMRunner interface {
	Run(*Contract, []byte) ([]byte, error)
}

func (self *VMEnv) Run(contract *Contract, input []byte) (ret []byte, err error) {
	return self.evm.Run(contract, input)
}
