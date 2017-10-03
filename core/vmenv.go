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

package core

import (
	"math/big"
	"errors"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/machine/classic"
)

type VmEnv interface {
	Coinbase() common.Address
	BlockNumber() *big.Int
	RuleSet() vm.RuleSet
	Db() *state.StateDB
	Call(sender common.Address, to common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	Create(me state.AccountObject, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error)
}

type vmEnv struct {
	coinbase 	common.Address
	number  	*big.Int
	difficulty  *big.Int
	gasLimit	*big.Int
	time		*big.Int
	db      	*state.StateDB
	hashfn		func(n uint64) common.Hash
	fork		vm.Fork
	rules       vm.RuleSet
	machine		vm.Machine
}

func (self *vmEnv) Coinbase() common.Address {
	return self.coinbase
}

func (self *vmEnv) BlockNumber() *big.Int {
	return self.number
}

func (self *vmEnv) RuleSet() vm.RuleSet {
	return self.rules
}

func (self *vmEnv) Db() *state.StateDB {
	return self.db
}

func (self *vmEnv) do(callOrCreate func(*vmEnv)(vm.Context,error)) ([]byte, common.Address, error) {
	for {
		var (
			context vm.Context
			err error
			out []byte
			address common.Address
			pa *common.Address
		)
		context, err = callOrCreate(self)
		if err == nil {
			out, pa, err = self.execute(context)
			if pa != nil {
				address = *pa
			}
			context.Finish()
		}
		if err != vm.BrokenError {
			return out, address, err
		} else {
			// ?? mybe will better to switch to the classic vm and try again before ??
			panic(err)
		}
	}
}

func (self *vmEnv) Call(sender common.Address, to common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	out, _, err := self.do(func(e *vmEnv)(vm.Context,error){
		return e.machine.Call(e.number.Uint64(),sender,to,data,gas,price,value)
	})
	return out, err
}

func (self *vmEnv) Create(caller state.AccountObject, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return self.do(func(e *vmEnv)(vm.Context,error){
		return e.machine.Create(e.number.Uint64(),caller.Address(),data,gas,price,value)
	})
}

func (self *vmEnv) execute(ctx vm.Context) ([]byte,*common.Address,error) {
	for {
		req := ctx.Fire()
		if req != nil {
			switch req.ID {
			case vm.RequireAccount:
				acc := self.db.GetAccount(req.Address)
				addr := acc.Address()
				nonce := acc.Nonce()
				balance := acc.Balance()
				ctx.CommitAccount(addr,nonce,balance)
			case vm.RequireCode:
				addr := req.Address
				code := self.db.GetCode(addr)
				hash := self.db.GetCodeHash(addr)
				ctx.CommitCode(addr,hash,code)
			case vm.RequireHash:
				number := req.Number
				hash := self.hashfn(number)
				ctx.CommitBlockhash(number,hash)
			case vm.RequireRules:
				ctx.CommitRules(self.rules.GasTable(self.number),self.fork,self.difficulty,self.gasLimit,self.time)
			default:
				if ctx.Status() == vm.RequireErr {
					// ?? unsupported VM implementaion ??
					// should we panic or use known VM instead?
					panic("unsupported VM RequireError occured")
				} else {
					// ?? incorrect VM implementation ??
					// Fire or Step can return nil or vm.Require only!
					panic("incorrect VM state")
				}
			}
		} else {
			switch ctx.Status() {
			case vm.ExitedOk:
				out, _, err := ctx.Out()
				if err != nil {
					return nil, nil, err
				}
				address, err := ctx.Address()
				if err != nil {
					return nil, nil, err
				}
				accounts, err := ctx.Modified()
				if err != nil {
					return nil, nil, err
				}
				// applying state here
				snapshot := self.db.Snapshot()
				for _, v := range accounts {
					var o state.AccountObject
					address := v.Address()
					if self.db.Exist(address) {
						o = self.db.GetAccount(address)
					} else {
						o = self.db.CreateAccount(address)
						hash, code, err := ctx.Code(address)
						if err != nil {
							self.db.RevertToSnapshot(snapshot)
							return nil, nil, err
						}
						if code {
							o.SetCode(hash,code)
						}
					}
					o.SetBalance(v.Balance())
					o.SetNonce(v.Nonce())
				}
				return out, &address, nil
			case vm.TransferErr:
				return nil, nil, InvalidTxError(ctx.Err())
			case vm.Broken, vm.BadCodeErr :
				return nil, nil, ctx.Err()
			case vm.OutOfGasErr:
				return nil, nil, OutOfGasError
			default:
				// ?? unsupported VM implementaion ??
				// should we panic or use known VM instead?
				panic("incorrect VM state")
			}
		}
	}
}

// GetHashFn returns a function for which the VM env can query block hashes through
// up to the limit defined by the Yellow Paper and uses the given block chain
// to query for information.
func GetHashFn(ref common.Hash, chain *BlockChain) func(n uint64) common.Hash {
	return func(n uint64) common.Hash {
		for block := chain.GetBlock(ref); block != nil; block = chain.GetBlock(block.ParentHash()) {
			if block.NumberU64() == n {
				return block.Hash()
			}
		}

		return common.Hash{}
	}
}

func getFork(number *big.Int, chainConfig *ChainConfig) vm.Fork {
	if chainConfig.IsDiehard(number) {
		return vm.Diehard
	} else if chainConfig.IsHomestead(number) {
		return vm.Homestead
	}
	return vm.Frontier
}

func NewEnv(statedb *state.StateDB, chainConfig *ChainConfig, chain *BlockChain, msg Message, header *types.Header) (VmEnv,error) {
	var machine vm.Machine = classic.NewMachine()
	fork := getFork(header.Number, chainConfig)
	vmenv := &vmEnv{
		header.Coinbase,
		header.Number,
		header.Difficulty,
		header.GasLimit,
		header.Time,
		statedb,
		GetHashFn(header.ParentHash, chain),
		fork,
		chainConfig,
		machine,
	}
	return vmenv, nil
}

var (
 	OutOfGasError = errors.New("Out of gas")
)
