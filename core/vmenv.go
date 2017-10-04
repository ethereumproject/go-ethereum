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
	"fmt"
)

// for compatibility reasons
type VmEnv interface {
	Coinbase() common.Address
	BlockNumber() *big.Int
	RuleSet() vm.RuleSet
	Db() *state.StateDB
	Call(sender common.Address, to common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	Create(me state.AccountObject, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error)
}

type Env struct {
	Coinbase_ 	common.Address
	Number_  	*big.Int
	Difficulty_ *big.Int
	GasLimit_	*big.Int
	Time_		*big.Int
	Db_      	*state.StateDB
	Hashfn_		func(n uint64) common.Hash
	Fork_		vm.Fork
	Rules_      vm.RuleSet
	Machine_	vm.Machine
}

func (self *Env) Coinbase() common.Address {
	return self.Coinbase_
}

func (self *Env) BlockNumber() *big.Int {
	return self.Number_
}

func (self *Env) RuleSet() vm.RuleSet {
	return self.Rules_
}

func (self *Env) Db() *state.StateDB {
	return self.Db_
}

func (self *Env) do(callOrCreate func(*Env)(vm.Context,error)) ([]byte, common.Address, *big.Int, error) {
	for {
		var (
			context vm.Context
			err error
			out []byte
			address common.Address
			pa *common.Address
			gas *big.Int
		)
		context, err = callOrCreate(self)
		if err == nil {
			out, pa, gas, err = self.execute(context)
			if pa != nil {
				address = *pa
			}
			context.Finish()
		}
		if gas == nil {
			gas = big.NewInt(0)
		}
		if err != vm.BrokenError {
			return out, address, gas, err
		} else {
			// ?? mybe will better to switch to the classic vm and try again before ??
			panic(err)
		}
	}
}

func (self *Env) Call(sender common.Address, to common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	out, _, gasRefund, err := self.do(func(e *Env)(vm.Context,error){
		return e.Machine_.Call(sender,to,data,gas,price,value)
	})
	//fmt.Printf("CALL GAS %v => %v\n",gas.Int64(),gasRefund.Int64())
	gas.Set(gasRefund)
	return out, err
}

func (self *Env) Create(caller state.AccountObject, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	out, addr, gasRefund, err := self.do(func(e *Env)(vm.Context,error){
		return e.Machine_.Create(caller.Address(),data,gas,price,value)
	})
	//fmt.Printf("CREATE GAS %v => %v\n",gas.Int64(),gasRefund.Int64())
	gas.Set(gasRefund)
	return out, addr, err
}

func (self *Env) execute(ctx vm.Context) ([]byte,*common.Address,*big.Int,error) {
	if err := ctx.CommitInfo(self.Number_.Uint64(),self.Coinbase_,
							 self.Rules_.GasTable(self.Number_),self.Fork_,
							 self.Difficulty_,self.GasLimit_,self.Time_);
		err != nil {
		return nil, nil, nil, err
	}
	for {
		req := ctx.Fire()
		if req != nil {
			switch req.ID {
			case vm.RequireAccount:
				if self.Db_.Exist(req.Address) {
					a := self.Db_.GetAccount(req.Address)
					ctx.CommitAccount(a.Address(),a.Nonce(),a.Balance())
				} else {
					ctx.CommitAccount(req.Address,0,nil)
				}
			case vm.RequireCode:
				addr := req.Address
				code := self.Db_.GetCode(addr)
				hash := self.Db_.GetCodeHash(addr)
				ctx.CommitCode(addr,hash,code)
			case vm.RequireHash:
				number := req.Number
				hash := self.Hashfn_(number)
				ctx.CommitBlockhash(number,hash)
			case vm.RequireInfo:
				panic("info already commited")
			case vm.RequireValue:
				value := self.Db_.GetState(req.Address,req.Hash)
				ctx.CommitValue(req.Address,req.Hash,value)
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
				out, gas, err := ctx.Out()
				if err != nil {
					return nil, nil, nil, err
				}
				address, err := ctx.Address()
				if err != nil {
					return nil, nil, nil, err
				}
				modified, err := ctx.Modified()
				if err != nil {
					return nil, nil, nil, err
				}
				removed, _ := ctx.Removed()
				if err != nil {
					return nil, nil, nil, err
				}
				values, _ := ctx.Values()
				if err != nil {
					return nil, nil, nil, err
				}
				logs, _ := ctx.Logs()
				if err != nil {
					return nil, nil, nil, err
				}
				// applying state here
				snapshot := self.Db_.Snapshot()
				for _, v := range modified {
					var o state.AccountObject
					address := v.Address()
					if self.Db_.Exist(address) {
						o = self.Db_.GetAccount(address)
					} else {
						o = self.Db_.CreateAccount(address)
						hash, code, err := ctx.Code(address)
						if err != nil {
							self.Db_.RevertToSnapshot(snapshot)
							return nil, nil, nil, err
						}
						if code != nil {
							o.SetCode(hash,code)
						}
					}
					o.SetBalance(v.Balance())
					o.SetNonce(v.Nonce())
				}
				for _, a := range removed {
					self.Db_.Suicide(a)
				}
				for _, kv := range values {
					self.Db_.SetState(kv.Address(),kv.Key(),kv.Value())
				}
				for _, l := range logs {
					self.Db_.AddLog(l)
				}
				return out, &address, gas, nil
			case vm.TransferErr:
				_, gas,_ := ctx.Out()
				return nil, nil, gas, InvalidTxError(ctx.Err())
			case vm.Broken, vm.BadCodeErr, vm.ExitedErr :
				_, gas,_ := ctx.Out()
				return nil, nil, gas, ctx.Err()
			case vm.OutOfGasErr:
				return nil, nil, nil, OutOfGasError
			default:
				// ?? unsupported VM implementaion ??
				// should we panic or use known VM instead?
				panic(fmt.Sprintf("incorrect VM state %d",ctx.Status()))
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

func NewEnv(statedb *state.StateDB, chainConfig *ChainConfig, chain *BlockChain, msg Message, header *types.Header) VmEnv {
	machine, err := VmManager.ConnectMachine()
	if err != nil {
		panic(err)
	}
	fork := getFork(header.Number, chainConfig)
	vmenv := &Env{
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
	return vmenv
}

var (
 	OutOfGasError = errors.New("Out of gas")
)

type VmManagerSingleton struct {

}
var VmManager = &VmManagerSingleton{}

func (self *VmManagerSingleton) ConnectMachine() (vm.Machine,error) {
	return classic.NewMachine(), nil
}