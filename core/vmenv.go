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
	"errors"
	"math/big"

	"fmt"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/machine/classic"
)

type VmEnv struct {
	Coinbase    common.Address
	BlockNumber *big.Int
	Difficulty  *big.Int
	GasLimit    *big.Int
	Time        *big.Int
	Db          *state.StateDB
	Hashfn      func(n uint64) common.Hash
	Fork        vm.Fork
	RuleSet     vm.RuleSet
	Machine     vm.Machine
}

func (self *VmEnv) do(callOrCreate func(*VmEnv) (vm.Context, error)) ([]byte, common.Address, *big.Int, error) {
	for {
		var (
			context vm.Context
			err     error
			out     []byte
			address common.Address
			pa      *common.Address
			gas     *big.Int
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

func (self *VmEnv) Call(sender common.Address, to common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	out, _, gasRefund, err := self.do(func(e *VmEnv) (vm.Context, error) {
		return e.Machine.Call(sender, to, data, gas, price, value)
	})
	gas.Set(gasRefund)
	return out, err
}

func (self *VmEnv) Create(caller state.AccountObject, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	out, addr, gasRefund, err := self.do(func(e *VmEnv) (vm.Context, error) {
		return e.Machine.Create(caller.Address(), data, gas, price, value)
	})
	gas.Set(gasRefund)
	return out, addr, err
}

func (self *VmEnv) execute(ctx vm.Context) ([]byte, *common.Address, *big.Int, error) {
	if err := ctx.CommitInfo(self.BlockNumber.Uint64(), self.Coinbase,
		self.RuleSet.GasTable(self.BlockNumber), self.Fork,
		self.Difficulty, self.GasLimit, self.Time); err != nil {
		return nil, nil, nil, err
	}
	for {
		req := ctx.Fire()
		if req != nil {
			switch req.ID {
			case vm.RequireAccount:
				if self.Db.Exist(req.Address) {
					a := self.Db.GetAccount(req.Address)
					ctx.CommitAccount(a.Address(), a.Nonce(), a.Balance())
				} else {
					ctx.CommitAccount(req.Address, 0, nil)
				}
			case vm.RequireCode:
				addr := req.Address
				code := self.Db.GetCode(addr)
				hash := self.Db.GetCodeHash(addr)
				ctx.CommitCode(addr, hash, code)
			case vm.RequireHash:
				number := req.Number
				hash := self.Hashfn(number)
				ctx.CommitBlockhash(number, hash)
			case vm.RequireInfo:
				panic("info already commited")
			case vm.RequireValue:
				value := self.Db.GetState(req.Address, req.Hash)
				ctx.CommitValue(req.Address, req.Hash, value)
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
				out, gas, gasRefund, err := ctx.Out()
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
				suicides, _ := ctx.Suicides()
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
				snapshot := self.Db.Snapshot()
				fmt.Printf("refund %v + %v\n",self.Db.GetRefund().Int64(),gasRefund.Int64())
				self.Db.AddRefund(gasRefund)
				for _, v := range modified {
					var o state.AccountObject
					address := v.Address()
					if self.Db.Exist(address) {
						o = self.Db.GetAccount(address)
					} else {
						o = self.Db.CreateAccount(address)
						hash, code, err := ctx.Code(address)
						if err != nil {
							self.Db.RevertToSnapshot(snapshot)
							return nil, nil, nil, err
						}
						if code != nil {
							o.SetCode(hash, code)
						}
					}
					o.SetBalance(v.Balance())
					o.SetNonce(v.Nonce())
					fmt.Printf("account: %v balance: %v nunce: %v\n",o.Address().Hex(), o.Balance().Int64(), o.Nonce())
					fmt.Printf("\t hash: %v codesize: %v\n",self.Db.GetCodeHash(o.Address()).Hex(),len(self.Db.GetCode(o.Address())))
				}
				for _, kv := range values {
					fmt.Printf("set vale %v:%v -> %v\n",kv.Address().Hex(), kv.Key().Hex(), kv.Value().Hex())
					self.Db.SetState(kv.Address(), kv.Key(), kv.Value())
				}
				for _, a := range suicides {
					fmt.Printf("suicide %v\n",a.Hex())
					self.Db.Suicide(a)
				}
				for _, l := range logs {
					self.Db.AddLog(l)
				}
				return out, &address, gas, nil
			case vm.TransferErr:
				_, gas, _, _ := ctx.Out()
				return nil, nil, gas, InvalidTxError(ctx.Err())
			case vm.Broken, vm.BadCodeErr, vm.ExitedErr:
				_, gas, _, _ := ctx.Out()
				return nil, nil, gas, ctx.Err()
			case vm.OutOfGasErr:
				return nil, nil, nil, OutOfGasError
			default:
				// ?? unsupported VM implementaion ??
				// should we panic or use known VM instead?
				panic(fmt.Sprintf("incorrect VM state %d", ctx.Status()))
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

func NewEnv(db *state.StateDB, chainConfig *ChainConfig, chain *BlockChain, msg Message, header *types.Header) *VmEnv {
	machine, err := VmManager.ConnectMachine()
	if err != nil {
		panic(err)
	}
	fork := getFork(header.Number, chainConfig)
	vmenv := &VmEnv{
		header.Coinbase,
		header.Number,
		header.Difficulty,
		header.GasLimit,
		header.Time,
		db,
		GetHashFn(header.ParentHash, chain),
		fork,
		chainConfig,
		machine,
	}
	return vmenv
}

var OutOfGasError = errors.New("Out of gas")

type vmManager struct {
	isConfigured    bool
	useVmType       vm.Type
	useVmManagment  vm.ManagerType
	useVmConnection vm.ConnectionType
	useVmRemotePort int
	useVmRemoteHost string
}

var VmManager = &vmManager{}

func (self *vmManager) autoConfig() {

	self.useVmType = vm.DefaultVm
	self.useVmManagment = vm.DefaultVmUsage
	self.useVmConnection = vm.DefaultVmProto
	self.useVmRemoteHost = vm.DefaultVmHost
	self.useVmRemotePort = vm.DefaultVmPort

	self.isConfigured = true
}

var BadVmUsageError = errors.New("VM can't be used in this way")

func (self *vmManager) ConnectMachine() (vm.Machine, error) {
	if !self.isConfigured {
		self.autoConfig()
	}
	if self.useVmType == vm.ClassicVm && self.useVmConnection == vm.LocalVm {
		return classic.NewMachine(), nil
	} else {
		return nil, BadVmUsageError
	}
}
