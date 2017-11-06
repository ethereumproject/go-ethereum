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

package client

import (
	"encoding/json"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/rpc"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func IpcNameFrom(name string) string {
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(name, `\\.\pipe\`) {
			return name
		}
		return `\\.\pipe\` + name
	}
	if filepath.Base(name) == name {
		return filepath.Join(os.TempDir(), name)
	}
	return name
}

func ConnectIpc(name string) (vm.MachineConn, error) {
	conn, err := rpc.NewIPCConnection(IpcNameFrom(name))
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

const VmSvc = "vm"
const VmSvcPfx = VmSvc + "_"

type Err struct {
	Text string	`json:text`
}

func (self *Err) Error() string {
	return self.Text
}

var callFailed = &Err{"call failed"}

type machine struct {
	out  *json.Encoder
	inp  *json.Decoder
	conn net.Conn
}

func New(conn net.Conn) vm.MachineConn {
	return &machine{
		json.NewEncoder(conn),
		json.NewDecoder(conn),
		conn,
	}
}

func (self *machine) Machine() vm.Machine {
	return self
}

func (self *machine) IsOnDuty() bool {
	return self.conn != nil
}

func (self *machine) Close() {
	if self.conn != nil {
		self.conn.Close()
		self.conn = nil
	}
}

func (self *machine) doCall(method string, ctx uint64, params []interface{}, result interface{}) error {
	request := map[string]interface{}{
		"id":      ctx,
		"method":  VmSvcPfx + method,
		"version": rpc.JSONRPCVersion,
		"params":  params,
	}

	if err := self.out.Encode(request); err != nil {
		return err
	}

	response := &rpc.JSONResponse{Result: result}
	if err := self.inp.Decode(response); err != nil {
		return err
	}

	return nil
}

type OnCallResult struct {
	Context uint64		`json:context`
	Status  vm.Status	`json:status`
	Err     *Err		`json:error`
}

func (self *machine) Call(caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (vm.Context, error) {
	var err error
	var ctx *context
	res := &OnCallResult{Status: vm.Broken, Err: callFailed}
	if data == nil {
		data = []byte{}
	}
	if err = self.doCall("call", 0, []interface{}{caller, to, data, gas, price, value}, res); err == nil {
		if res.Err != nil {
			err = res.Err
		} else {
			ctx = &context{res.Context, self, res.Status}
		}
	}
	return ctx, err
}

func (self *machine) Create(caller common.Address, code []byte, gas, price, value *big.Int) (vm.Context, error) {
	var err error
	var ctx *context
	res := &OnCallResult{Status: vm.Broken, Err: callFailed}
	if code == nil {
		code = []byte{}
	}
	args := []interface{}{caller, code, gas, price, value}
	if err = self.doCall("create", 0, args, res); err == nil {
		if res.Err != nil {
			err = res.Err
		} else {
			ctx = &context{res.Context, self, res.Status}
		}
	}
	return ctx, err
}

func (self *machine) Type() vm.Type {
	return vm.OtherVm
}

type OnNameResult struct {
	Name string
}

func (self *machine) Name() string {
	res := &OnNameResult{}
	if err := self.doCall("name", 0, nil, res); err != nil {
		return vm.UnknwonStringValue
	}
	return res.Name
}

type context struct {
	context uint64
	machine *machine
	status  vm.Status
}

type OnCommitResult struct {
	Status vm.Status	`json:status`
	Err    *Err			`json:error`
}

func (self *context) doCommit(method string, args []interface{}) (err error) {
	res := &OnCommitResult{Status: vm.Broken, Err: callFailed}
	if err = self.machine.doCall(method, self.context, args, res); err != nil {
		self.status = vm.Broken
		err = err
	} else {
		self.status = res.Status
		if res.Err != nil {
			err = res.Err
		}
	}
	return
}

func (self *context) CommitAccount(address common.Address, nonce uint64, balance *big.Int) error {
	return self.doCommit("commitAccount", []interface{}{address, nonce, balance})
}

func (self *context) CommitBlockhash(number uint64, hash common.Hash) error {
	return self.doCommit("commitBlockhash", []interface{}{number, hash})
}

func (self *context) CommitCode(address common.Address, hash common.Hash, code []byte) error {
	if code == nil {
		code = []byte{}
	}
	return self.doCommit("commitCode", []interface{}{address, code, hash})
}

func (self *context) CommitInfo(blockNumber uint64, coinbase common.Address, table *vm.GasTable, fork vm.Fork, difficulty, gasLimit, time *big.Int) error {
	return self.doCommit("commitInfo", []interface{}{blockNumber, coinbase, table, fork, difficulty, gasLimit, time})
}

func (self *context) CommitValue(address common.Address, key common.Hash, value common.Hash) error {
	return self.doCommit("commitValue", []interface{}{address, key, value})
}

type OnFinishResult struct {
	Status vm.Status	`json:status`
	Err    *Err			`json:error`
}

func (self *context) Finish() (err error) {
	res := &OnFinishResult{Status: vm.Broken, Err: callFailed}
	if err = self.machine.doCall("finish", self.context, nil, res); err != nil {
		self.status = vm.Broken
	} else {
		self.status = res.Status
		if res.Err != nil {
			err = res.Err
		}
	}
	return err
}

type Account struct {
	Address_  common.Address			`json:address`
	Nonce_    uint64					`json:nonce`
	Balance_  *big.Int					`json:balance`
	Suicided_ bool						`json:suicided`
	Newborn_  bool						`json:newborn`
	Code_     []byte					`json:code`
	Hash_     common.Hash				`json:hash`
	Value_    map[string]common.Hash	`json:value`
}

func (self *Account) Address() common.Address { return self.Address_ }
func (self *Account) Nonce() uint64           { return self.Nonce_ }
func (self *Account) Balance() *big.Int       { return self.Balance_ }
func (self *Account) IsSuicided() bool        { return self.Suicided_ }
func (self *Account) IsNewborn() bool         { return self.Newborn_ }

func (self *Account) Store(f func(common.Hash, common.Hash) error) error {
	for k, v := range self.Value_ {
		if err := f(common.HexToHash(k), v); err != nil {
			return err
		}
	}
	return nil
}

func (self *Account) Code() (common.Hash, []byte) {
	if len(self.Code_) == 0 {
		return common.Hash{}, nil
	}
	return self.Hash_, self.Code_
}

type OnAccountsResult struct {
	Accounts []*Account	`json:accounts`
	Status   vm.Status	`json:status`
	Err      *Err		`json:error`
}

func (self *context) Accounts() (accounts []vm.Account, err error) {
	res := &OnAccountsResult{Status: vm.Broken, Err: callFailed}
	if err = self.machine.doCall("accounts", self.context, nil, res); err != nil {
		self.status = vm.Broken
	} else {
		self.status = res.Status
		if res.Err != nil {
			err = res.Err
		}
		if err == nil {
			accounts = make([]vm.Account, len(res.Accounts))
			for i, a := range res.Accounts {
				accounts[i] = a
			}
		}
	}
	return
}

type OnAddressResult struct {
	Address common.Address	`json:address`
	Status  vm.Status		`json:status`
	Err     *Err			`json:error`
}

func (self *context) Address() (address common.Address, err error) {
	res := &OnAddressResult{Status: vm.Broken, Err: callFailed}
	if err = self.machine.doCall("address", self.context, nil, res); err != nil {
		self.status = vm.Broken
	} else {
		self.status = res.Status
		if res.Err != nil {
			err = res.Err
		}
		address = res.Address
	}
	return
}

type OnOutResult struct {
	Out    []byte		`json:"out"`
	Gas    *big.Int		`json:"gas"`
	Refund *big.Int		`json:"refund"`
	Status vm.Status	`json:"status"`
	Err    *Err			`json:"error"`
}

func (self *context) Out() ([]byte, *big.Int, *big.Int, error) {
	res := &OnOutResult{Status: vm.Broken, Err: callFailed}
	if err := self.machine.doCall("out", self.context, nil, res); err != nil {
		self.status = vm.Broken
		return nil, nil, nil, err
	} else {
		self.status = res.Status
		if res.Err != nil {
			err = res.Err
		}
		return res.Out, res.Gas, res.Refund, err
	}
}

type Log struct {
	Address     common.Address	`json:"address"`
	Topics      []common.Hash	`json:"topics"`
	Data        []byte			`json:"data"`
	BlockNumber *big.Int		`json:"number"`
	TxHash      common.Hash		`json:"txhash"`
	TxIndex     uint			`json:"txindex"`
	BlockHash   common.Hash		`json:"hash"`
	Index       uint			`json:"index"`
}

type OnLogsResult struct {
	Logs   []*Log
	Status vm.Status
	Err    *Err
}

func (self *context) Logs() (logs state.Logs, err error) {
	res := &OnLogsResult{Status: vm.Broken, Err: callFailed}
	if err = self.machine.doCall("logs", self.context, nil, res); err != nil {
		self.status = vm.Broken
	} else {
		self.status = res.Status
		if res.Err != nil {
			err = res.Err
		}
		if err == nil {
			logs = make([]*state.Log, len(res.Logs))
			for i, v := range res.Logs {
				logs[i] = &state.Log{
					v.Address,
					v.Topics,
					v.Data,
					v.BlockNumber.Uint64(),
					v.TxHash,
					v.TxIndex,
					v.BlockHash,
					v.Index,
				}
			}
		}
	}
	return
}

func (self *context) Status() vm.Status {
	return self.status
}

type OnErrResult struct {
	Err *Err	`json:"error"`
}

func (self *context) Err() (err error) {
	res := &OnErrResult{Err: callFailed}
	if err = self.machine.doCall("err", self.context, nil, res); err != nil {
		self.status = vm.Broken
	} else {
		err = res.Err
	}
	return
}

type OnFireResult struct {
	Require *vm.Require	`json:"require"`
	Status  vm.Status	`json:"status"`
	Err     *Err		`json:"error"`
}

func (self *context) Fire() (r *vm.Require) {
	res := &OnFireResult{Status: vm.Broken, Err: callFailed}
	if err := self.machine.doCall("fire", self.context, nil, res); err != nil {
		self.status = vm.Broken
	} else {
		self.status = res.Status
		if self.status == vm.RequireErr {
			r = res.Require
		}
	}
	return
}
