// Copyright (c) 2017 ETCDEV Team

// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package server

import (
	"fmt"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/machine/rpc/client"
	"math/big"
)

type MachineSvc struct {
	Machine vm.Machine
	cID     uint64
	context vm.Context
}

func (self *MachineSvc) StringifyState() string {
	machine := self.Machine.Name()
	tp := self.Machine.Type().String()
	if self.context != nil {
		var errS string
		err := self.context.Err()
		if err == nil {
			errS = ""
		} else {
			errS = "ERROR '" + err.Error() + "'"
		}
		status := self.context.Status().String()
		return fmt.Sprintf("%s (%s) %s %s", machine, tp, status, errS)
	} else {
		return fmt.Sprintf("%s (%s) OutOfContext", machine, tp)
	}
}

func (self *MachineSvc) Call(caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (res client.OnCallResult) {
	var err error
	if self.context != nil {
		self.context.Finish()
		self.context = nil
	}
	self.cID++
	if self.context, err = self.Machine.Call(caller, to, data, gas, price, value); err != nil {
		res.Err = &client.Err{err.Error()}
		res.Status = vm.Broken
	} else {
		res.Context = self.cID
		res.Status = vm.Inactive
	}
	glog.V(logger.Debug).Infof("vmsvc.Call: %v, %v, %v, %v, %v => %s\n", caller.Hex(), to.Hex(), gas, price, value, self.StringifyState())
	return res
}

func (self *MachineSvc) Create(caller common.Address, code []byte, gas, price, value *big.Int) (res client.OnCallResult) {
	var err error
	if self.context != nil {
		self.context.Finish()
		self.context = nil
	}
	self.cID++
	if self.context, err = self.Machine.Create(caller, code, gas, price, value); err != nil {
		res.Status = vm.Broken
		res.Err = &client.Err{err.Error()}
	} else {
		res.Context = self.cID
		res.Status = vm.Inactive
	}
	glog.V(logger.Debug).Infof("vmsvc.Create: %v, %v, %v, %v => %s\n", caller.Hex(), gas, price, value, self.StringifyState())
	return res
}

func (self *MachineSvc) Fire() (res client.OnFireResult) {
	if req := self.context.Fire(); req != nil {
		res.Require = req
	}
	res.Status = self.context.Status()
	if err := self.context.Err(); err != nil {
		res.Err = &client.Err{err.Error()}
	}
	if res.Require != nil {
		glog.V(logger.Debug).Infof("vmsvc.Fire : %s => %s\n", res.Require.ID.String(), self.StringifyState())
	} else {
		glog.V(logger.Debug).Infof("vmsvc.Fire : nil => %s\n", self.StringifyState())
	}
	return
}

func (self *MachineSvc) CommitAccount(address common.Address, nonce uint64, balance *big.Int) (res client.OnCommitResult) {
	if err := self.context.CommitAccount(address, nonce, balance); err != nil {
		res.Err = &client.Err{err.Error()}
	}
	res.Status = self.context.Status()
	glog.V(logger.Debug).Infof("vmsvc.CommitAccount %v, %v, %v => %s\n", address.Hex(), nonce, balance, self.StringifyState())
	return
}

func (self *MachineSvc) CommitBlockhash(number uint64, hash common.Hash) (res client.OnCommitResult) {
	if err := self.context.CommitBlockhash(number, hash); err != nil {
		res.Err = &client.Err{err.Error()}
	}
	res.Status = self.context.Status()
	glog.V(logger.Debug).Infof("vmsvc.CommitBlockHash %v, %v => %s\n", number, hash.Hex(), self.StringifyState())
	return
}

func (self *MachineSvc) CommitCode(address common.Address, code []byte, hash common.Hash) (res client.OnCommitResult) {
	var emptyHash = common.Hash{}
	if len(code) == 0 && hash == emptyHash {
		code = nil
	}
	if err := self.context.CommitCode(address, hash, code); err != nil {
		res.Err = &client.Err{err.Error()}
	}
	res.Status = self.context.Status()
	glog.V(logger.Debug).Infof("vmsvc.CommitCode %v, %v, %v => %s\n", address.Hex(), hash.Hex(), code, self.StringifyState())
	return
}

func (self *MachineSvc) CommitInfo(blockNumber uint64, coinbase common.Address, table *vm.GasTable, fork vm.Fork, difficulty, gasLimit, time *big.Int) (res client.OnCommitResult) {
	if err := self.context.CommitInfo(blockNumber, coinbase, table, fork, difficulty, gasLimit, time); err != nil {
		res.Err = &client.Err{err.Error()}
	}
	res.Status = self.context.Status()
	glog.V(logger.Debug).Infof("vmsvc.CommitInfo %v, %v, %v, %v, %v, %v, %v => %s\n", blockNumber, coinbase.Hex(), table, fork, difficulty, gasLimit, time, self.StringifyState())
	return
}

func (self *MachineSvc) CommitValue(address common.Address, key common.Hash, value common.Hash) (res client.OnCommitResult) {
	if err := self.context.CommitValue(address, key, value); err != nil {
		res.Err = &client.Err{err.Error()}
	}
	res.Status = self.context.Status()
	glog.V(logger.Debug).Infof("vmsvc.CommitValue %v, %v, %v => %s\n", address.Hex(), key.Hex(), value.Hex(), self.StringifyState())
	return
}

func (self *MachineSvc) Accounts() (res client.OnAccountsResult) {
	if accounts, err := self.context.Accounts(); err != nil {
		res.Err = &client.Err{err.Error()}
	} else {
		res.Accounts = make([]*client.Account, len(accounts))
		for i, a := range accounts {
			hash, code := a.Code()
			res.Accounts[i] = &client.Account{
				a.Address(),
				a.Nonce(),
				a.Balance(),
				a.IsSuicided(),
				a.IsNewborn(),
				code,
				hash,
				make(map[string]common.Hash),
			}
			a.Store(func(k, v common.Hash) error {
				glog.V(logger.Debug).Infof("STORE %v[%v] = %v\n",
					res.Accounts[i].Address_.Hex(),
					k.Hex(),
					v.Hex())
				res.Accounts[i].Value_[k.Hex()] = v
				return nil
			})
			glog.V(logger.Debug).Infof("%d: %v => NONCE %v, BALANCE %v, CODE: %v bytes\n",
				i,
				res.Accounts[i].Address_.Hex(),
				res.Accounts[i].Nonce_,
				res.Accounts[i].Balance_,
				len(res.Accounts[i].Code_))
		}
	}
	res.Status = self.context.Status()
	glog.V(logger.Debug).Infof("vmsvc.Accounts => %s\n", self.StringifyState())
	return
}

func (self *MachineSvc) Address() (res client.OnAddressResult) {
	if address, err := self.context.Address(); err != nil {
		res.Err = &client.Err{err.Error()}
	} else {
		res.Address = address
	}
	res.Status = self.context.Status()
	glog.V(logger.Debug).Infof("vmsvc.Address : %s => %s\n",
		res.Address.Hex(), self.StringifyState())
	return
}

func (self *MachineSvc) Out() (res client.OnOutResult) {
	if out, gas, refund, err := self.context.Out(); err != nil {
		res.Err = &client.Err{err.Error()}
	} else {
		res.Out = out
		res.Gas = gas
		res.Refund = refund
	}
	res.Status = self.context.Status()

	glog.V(logger.Debug).Infof("vmsvc.Out : %v bytes, %v, %v => %s\n",
		len(res.Out), res.Gas, res.Refund, self.StringifyState())
	return
}

func (self *MachineSvc) Logs() (res client.OnLogsResult) {
	if logs, err := self.context.Logs(); err != nil {
		res.Err = &client.Err{err.Error()}
	} else {
		res.Logs = make([]*client.Log, len(logs))
		for i, v := range logs {
			res.Logs[i] = &client.Log{
				v.Address,
				v.Topics,
				v.Data,
				new(big.Int).SetUint64(v.BlockNumber),
				v.TxHash,
				v.TxIndex,
				v.BlockHash,
				v.Index,
			}
		}
	}
	res.Status = self.context.Status()
	glog.V(logger.Debug).Infof("vmsvc.Logs => %d, %s\n", len(res.Logs), self.StringifyState())
	return
}

func (self *MachineSvc) Name() (res client.OnNameResult) {
	res.Name = self.Machine.Name()
	glog.V(logger.Debug).Info("vmsvc.Name => %s", res.Name)
	return
}

func (self *MachineSvc) Finish() (res client.OnFinishResult) {
	if err := self.context.Finish(); err != nil {
		res.Err = &client.Err{err.Error()}
	}
	res.Status = self.context.Status()
	glog.V(logger.Debug).Infof("vmsvc.Finish => %s\n", self.StringifyState())
	return
}
