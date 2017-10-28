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
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/machine/classic"
	extvm "github.com/ethereumproject/go-ethereum/machine/rpc/client"
	"os"
	"strings"
	"unicode"
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
				if st := ctx.Status(); st == vm.RequireErr {
					// ?? unsupported VM implementaion ??
					// should we panic or use known VM instead?
					panic("unsupported VM RequireError occured")
				} else {
					// ?? incorrect VM implementation ??
					// Fire or Step can return nil or vm.Require only!
					panic(fmt.Sprintf("incorrect VM state %d (%s)", st, st.String()))
				}
			}
		} else {
			switch st := ctx.Status(); st {
			case vm.ExitedOk, vm.OutOfGasErr, vm.ExitedErr:
				out, gas, gasRefund, err := ctx.Out()
				if err != nil {
					return nil, nil, nil, err
				}

				address, err := ctx.Address()
				if err != nil {
					return nil, nil, nil, err
				}

				logs, _ := ctx.Logs()
				if err != nil {
					return nil, nil, nil, err
				}

				accounts, err := ctx.Accounts()
				if err != nil {
					return nil, nil, nil, err
				}

				self.Db.AddRefund(gasRefund)
				for _, a := range accounts {
					address := a.Address()
					var o state.AccountObject
					if a.IsNewborn() {
						o = self.Db.CreateAccount(address)
					} else {
						o = self.Db.GetOrNewStateObject(address)
					}
					o.SetBalance(a.Balance())
					o.SetNonce(a.Nonce())
					a.Store(func(key, value common.Hash) error {
						self.Db.SetState(a.Address(), key, value)
						return nil
					})
					if a.IsSuicided() {
						self.Db.Suicide(address)
					} else {
						hash, code := a.Code()
						if code != nil {
							o.SetCode(hash, code)
						}
					}
					//fmt.Printf("account: %v balance: %v nonce: %v\n",o.Address().Hex(), o.Balance().Int64(), o.Nonce())
					//fmt.Printf("\t hash: %v codesize: %v\n",self.Db.GetCodeHash(o.Address()).Hex(),len(self.Db.GetCode(o.Address())))
					//fmt.Printf("\t suicided: %v\n",self.Db.HasSuicided(o.Address()))
				}
				for _, l := range logs {
					self.Db.AddLog(l)
				}
				if st == vm.OutOfGasErr {
					return nil, nil, gas, OutOfGasError
				} else if st == vm.ExitedErr {
					return nil, nil, gas, ctx.Err()
				} else {
					return out, &address, gas, nil
				}
			case vm.TransferErr:
				_, gas, _, _ := ctx.Out()
				return nil, nil, gas, InvalidTxError(ctx.Err())
			case vm.Broken:
				return nil, nil, nil, vm.BrokenError
			default:
				// ?? unsupported VM implementaion ??
				// should we panic or use known VM instead?
				st := ctx.Status()
				panic(fmt.Sprintf("incorrect VM state %d (%s)", st, st.String()))
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

func NewEnvWithMachine(machine vm.Machine, db *state.StateDB, chainConfig *ChainConfig, chain *BlockChain, msg Message, header *types.Header) *VmEnv {
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
	if machine.Type() == vm.ClassicRawVm {
		machine.(*classic.RawMachine).Bind(db, vmenv.Hashfn)
		// be aware, commitInfo should be called before Fire
	}
	return vmenv
}

func NewEnv(db *state.StateDB, chainConfig *ChainConfig, chain *BlockChain, msg Message, header *types.Header) *VmEnv {
	machine, err := VmManager.ConnectMachine()
	if err != nil {
		panic(err)
	}
	return NewEnvWithMachine(machine, db, chainConfig, chain, msg, header)
}

var OutOfGasError = errors.New("Out of gas")

type vmManager struct {
	isConfigured    bool
	useVmType       vm.Type
	useVmManagment  vm.ManagerType
	useVmConnection vm.ConnectionType
	useVmRemotePort int
	useVmRemoteHost string
	useVmIpcName    string

	conn vm.MachineConn
}

var VmManager = &vmManager{}

func (self *vmManager) Close() {
	if self.conn != nil {
		self.conn.Close()
		self.conn = nil
	}
}

func (self *vmManager) autoConfig() {

	self.Close()

	self.useVmType = vm.DefaultVm
	self.useVmManagment = vm.DefaultVmUsage
	self.useVmConnection = vm.DefaultVmProto
	self.useVmRemoteHost = vm.DefaultVmHost
	self.useVmRemotePort = vm.DefaultVmPort
	self.useVmIpcName = vm.DefaultVmIpc
}

func (self *vmManager) EnvConfig() {

	vmSelector := os.Getenv("USE_GETH_VM")
	vmOpt := strings.Split(vmSelector, ":")
	switch tp := strings.ToUpper(vmOpt[0]); tp {
	case "CLASSIC":
		self.SwitchToClassicVm()
		return
	case "RAWCLASSIC":
		self.SwitchToRawClassicVm()
		return
	case "IPC":
		connName := vm.DefaultVmIpc
		manageVm := ""
		if len(vmOpt) > 1 {
			var name []rune
			for _, c := range vmOpt[1] {
				if unicode.IsLetter(c) ||
					unicode.IsNumber(c) ||
					c == '_' || c == '-' {
					name = append(name, c)
				}
			}
			if len(name) > 0 {
				connName = string(name)
			}
		}
		self.SwitchToIpc(connName, manageVm)
	case "":
		self.Autoconfig()
	default:
		glog.Error("found unknown virtual machine usage in environment variable USE_GETH_VM")
		self.Autoconfig()
	}
}

func (self *vmManager) Autoconfig() {
	self.autoConfig()
	self.isConfigured = true
	self.WriteConfigToLog()
}

func (self *vmManager) SwitchToIpc(connName string, manageVm string) {
	self.autoConfig()
	self.useVmType = vm.OtherVm
	self.useVmConnection = vm.IpcVm
	self.useVmIpcName = connName
	self.isConfigured = true
	self.WriteConfigToLog()
}

func (self *vmManager) SwitchToRawClassicVm() {
	self.autoConfig()
	self.useVmType = vm.ClassicRawVm
	self.useVmConnection = vm.InprocVm
	self.isConfigured = true
	self.WriteConfigToLog()
}

func (self *vmManager) SwitchToClassicVm() {
	self.autoConfig()
	self.useVmType = vm.ClassicVm
	self.useVmConnection = vm.InprocVm
	self.isConfigured = true
	self.WriteConfigToLog()
}

func (self *vmManager) WriteConfigToLog() {
	glog.V(logger.Debug).Infof("use virtual machine %s/%s (%s)\n", self.useVmType.String(), self.useVmConnection.String(), self.useVmManagment.String())
	switch self.useVmConnection {
	case vm.IpcVm, vm.LocalVm:
		glog.V(logger.Debug).Infof("virtual machine IPC name '%s' to connect\n", self.useVmIpcName)
	case vm.RpcVm:
		glog.V(logger.Debug).Infof("virtual machine RPC addr '%s:%d' to connect\n", self.useVmRemoteHost, self.useVmRemotePort)
	}
}

var BadVmUsageError = errors.New("VM can't be used in this way")

func (self *vmManager) ConnectMachine() (vm.Machine, error) {

	if !self.isConfigured {
		self.autoConfig()
		self.isConfigured = true
	}

	if self.conn != nil {
		if self.conn.IsOnDuty() {
			return self.conn.Machine(), nil
		} else {
			self.conn.Close()
			self.conn = nil
		}
	}

	switch self.useVmConnection {
	case vm.InprocVm:
		switch self.useVmType {
		case vm.ClassicVm:
			return classic.NewMachine(), nil
		case vm.ClassicRawVm:
			return classic.NewRawMachine(), nil
		default:
			return nil, BadVmUsageError
		}
	case vm.IpcVm:
		var err error
		self.conn, err = extvm.ConnectIpc(self.useVmIpcName)
		if err != nil {
			return nil, err
		}
		return self.conn.Machine(), nil
	default:
		return nil, BadVmUsageError
	}
}
