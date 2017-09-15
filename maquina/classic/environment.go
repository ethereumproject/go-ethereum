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
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/core/state"
)

type VMEnv struct {
	rules     vm.RuleSet
	state     *state.StateDB // State to use for executing
	evm       *EVM           // The Ethereum Virtual Machine
	depth     int            // Current execution depth
	header    *types.Header            // Header information
	getHashFn func(uint64) common.Hash // getHashFn callback is used to retrieve block hashes
 	from      common.Address
 	value     *big.Int

//	msg         Message        // Message appliod
//	chain     *BlockChain              // Blockchain handle
//  chainConfig *ChainConfig   // Chain configuration
}

func NewEnv(
	state *state.StateDB, rules vm.RuleSet, header *types.Header,
	from common.Address, value *big.Int,
	hashFn func(uint64) common.Hash) (vm.Environment,error) {

	env := &VMEnv{
		rules:     rules,
		state:     state,
		header:    header,
		getHashFn: hashFn,
		from:	   from,
		value:     value,
	}

	env.evm = NewVM(env)
	return env, nil
}

func (self *VMEnv) VmType() vm.Type {
	return vm.StdVmTy
}

func (self *VMEnv) RuleSet() vm.RuleSet      { 
	return self.rules 
}

func (self *VMEnv) Origin() common.Address   { 
	return self.from
}

func (self *VMEnv) BlockNumber() *big.Int    { 
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

func (self *VMEnv) Db() vm.Database { 
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

func (self *VMEnv) AddLog(log *vm.Log) {
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

func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) {
	Transfer(from, to, amount)
}

func (self *VMEnv) Call(me vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return Call(self, me, addr, data, gas, price, value)
}

func (self *VMEnv) CallCode(me vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return CallCode(self, me, addr, data, gas, price, value)
}

func (self *VMEnv) DelegateCall(me vm.ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return DelegateCall(self, me, addr, data, gas, price)
}

func (self *VMEnv) Create(me vm.ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return Create(self, me, data, gas, price, value)
}

type EVMRun interface {
	Run(*Contract,[]byte) ([]byte,error)
}

func (self *VMEnv) Run(contract *Contract, input []byte) (ret []byte, err error) {
	return self.evm.Run(contract,input)
}

