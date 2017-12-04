// Copyright 2017 (c) ETCDEV Team
//
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

package sputnik

/*

#cgo amd64,windows LDFLAGS: -L${SRCDIR}/windows_amd64 -lsputnikvm -lws2_32 -luserenv
#cgo amd64,linux LDFLAGS: -L${SRCDIR}/linux_amd64 -lsputnikvm
#cgo amd64,windows amd64,linux CFLAGS: -DSPUTNIK_VM_IMPLEMENTED

#include "sputnikvm.h"

#ifndef SPUTNIK_VM_IMPLEMENTED
#include "unimplemented.cx"
#endif

*/
import "C"

import (
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"math/big"
	"unsafe"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"errors"
	"github.com/ethereumproject/go-ethereum/crypto"
	"strings"
	"math/bits"
)

type machine struct {
}

type context struct {
	machine *machine

	gas    *big.Int
	price  *big.Int
	value  *big.Int
	caller common.Address
	target *common.Address
	data   []byte

	st     vm.Status

	gasLimit 	*big.Int
	coinbase 	common.Address
	fork     	vm.Fork
	blockNumber uint64
	time       	uint64
	difficulty 	*big.Int

	ctxPtr  unsafe.Pointer

	subRequire *vm.Require
	subCode    []byte
}

func NewMachine() vm.Machine {
	if C.sputnikvm_is_implemented() != 0 {
		return &machine{}
	} else {
		return nil
	}
}

// Call contract in new VM context
func (self *machine) Call(caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (vm.Context, error) {
	ctx := &context{machine: self, st: vm.Inactive}

	ctx.gas = gas
	ctx.price = price
	ctx.value = value
	ctx.caller = caller
	ctx.target = &to
	ctx.data = data

	return ctx, nil
}

// Create contract in new VM context
func (self *machine) Create(caller common.Address, code []byte, gas, price, value *big.Int) (vm.Context, error) {
	ctx := &context{machine: self, st: vm.Inactive}

	ctx.gas = gas
	ctx.price = price
	ctx.value = value
	ctx.caller = caller
	ctx.data = code

	return ctx, nil
}

// Type of VM
func (self *machine) Type() vm.Type {
	return vm.SputnikVm
}

// Name of VM
func (self *machine) Name() string {
	return "SPUTNIK VM"
}

func fillBintBytes(bint *big.Int, bytes []byte) {
	if bint != nil {
		b := bint.Bits()
		if len(b) > 256/bits.UintSize {
			panic("Oh, very big balance!")
		}
		if b != nil {
			for i := 0; i < len(b); i++ {
				n := i * bits.UintSize/8
				w := b[i]
				for j := 0; j < bits.UintSize/8; j++ {
					bytes[n+j] = byte(w)
					w >>= 8
				}
			}
		}
	}
}

func getBintBytes(bint *big.Int) []byte {
	var b = make([]byte,32)
	fillBintBytes(bint,b)
	return b
}

// Commit an account information
func (self *context) CommitAccount(address common.Address, nonce uint64, balance *big.Int) (err error) {
	var codePtr unsafe.Pointer = nil
	if self.subCode != nil {
		codePtr = unsafe.Pointer(&self.subCode[0])
	}
	var bitsPtr unsafe.Pointer = nil
	var bits [32]byte
	if balance != nil {
		fillBintBytes(balance,bits[:])
		bitsPtr = unsafe.Pointer(&bits[0])
	}
	C.sputnikvm_commit_account(
		self.Context(),
		unsafe.Pointer(&address[0]),
		C.uint64_t(nonce),
		bitsPtr,
		codePtr, C.size_t(len(self.subCode)))
	self.subCode = nil
	return
}

// Commit a block hash
func (self *context) CommitBlockhash(number uint64, hash common.Hash) (err error) {
	C.sputnikvm_commit_blockhash(self.Context(),C.uint64_t(number),unsafe.Pointer(&hash[0]))
	return
}

// Commit a contract code
func (self *context) CommitCode(address common.Address, hash common.Hash, code []byte) (err error) {
	if self.subRequire != nil {
		self.subCode = code
	} else {
		var codePtr unsafe.Pointer = nil
		if code != nil {
			codePtr = unsafe.Pointer(&code[0])
		}
		C.sputnikvm_commit_code(self.Context(),
			unsafe.Pointer(&address[0]), codePtr, C.size_t(len(code)))
	}
	return
}

// Commit an info
func (self *context) CommitInfo(blockNumber uint64, coinbase common.Address, table *vm.GasTable, fork vm.Fork, difficulty, gasLimit, time *big.Int) (err error) {
	self.gasLimit = gasLimit
	self.coinbase = coinbase
	self.fork = fork
	self.blockNumber = blockNumber
	self.time = time.Uint64()
	self.difficulty = difficulty
	return
}

// Commit a state
func (self *context) CommitValue(address common.Address, key common.Hash, value common.Hash) (err error) {
	C.sputnikvm_commit_value(
		self.Context(),
		unsafe.Pointer(&address[0]),
		unsafe.Pointer(&key[0]),
		unsafe.Pointer(&value[0]))
	return
}

// Finish VM context
func (self *context) Finish() (err error) {
	if self.ctxPtr != nil {
		C.sputnikvm_terminate(self.ctxPtr)
		self.ctxPtr = nil
	}
	return
}

// Returns the out value and gas remaining/refunded if any
func (self *context) Out() (out []byte, gas *big.Int, refund *big.Int, err error) {
	ctx := self.Context()
	out_len := C.sputnikvm_out_len(ctx)
	if out_len > 0 {
		out = make([]byte, out_len)
		C.sputnikvm_out_copy(ctx, unsafe.Pointer(&out[0]))
	}
	var b []big.Word
	b = make([]big.Word,256/bits.UintSize)
	C.sputnikvm_gas_copy(ctx,unsafe.Pointer(&b[0]))
	gas = new(big.Int).SetBits(b)
	b = make([]big.Word,256/bits.UintSize)
	C.sputnikvm_refund_copy(ctx,unsafe.Pointer(&b[0]))
	refund = new(big.Int).SetBits(b)
	return
}

// Returns the changed accounts information
func (self *context) Accounts() (accounts []vm.Account, err error) {

	ctx := self.Context()

	for ptr := C.sputnikvm_first_account(ctx); ptr != nil; ptr = C.sputnikvm_next_account(ctx) {
		switch vm.AccountChangeLevel(C.sputnikvm_acc_change(ptr)) {
		case vm.UpdateAccount, vm.CreateAccount, vm.AddBalanceAccount, vm.SubBalanceAccount :
			acc := &account{ctx, ptr}
			accounts = append(accounts, acc)
		}
	}

	suicidesCount := int(C.sputnikvm_suicides_count(ctx))
	for i := 0; i < suicidesCount; i++ {
		suicide := &suicided{}
		C.sputnikvm_suicide_copy(ctx,C.size_t(i),unsafe.Pointer(&suicide.address[0]))
		accounts = append(accounts,suicide)
	}

	return
}

func logTopics(s string) (topics []common.Hash){
	hs := strings.Split(s,",")
	topics = make([]common.Hash,len(hs))
	for i, h := range hs {
		topics[i] = common.HexToHash(h)
	}
	return
}

// Returns logs to be appended to the current block if the user
// decided to accept the running status of this VM
func (self *context) Logs() (logs state.Logs, err error) {
	ctx := self.Context()
	logsCount := int(C.sputnikvm_logs_count(ctx))
	logs = make(state.Logs,logsCount)
	for logNo := 0; logNo < logsCount; logNo += 1 {

		ptr := C.sputnikvm_log(ctx,C.size_t(logNo))

		dataLen := C.sputnikvm_log_data_len(ptr)
		data := make([]byte, dataLen)
		if dataLen > 0 {
			C.sputnikvm_log_data_copy(ptr, unsafe.Pointer(&data[0]))
		}

		topicsCount := int(C.sputnikvm_log_topics_count(ptr))
		topics := make([]common.Hash,topicsCount)

		for i := 0; i < topicsCount; i += 1 {
			C.sputnikvm_log_topic_copy(ptr,C.size_t(i),unsafe.Pointer(&topics[i][0]))
		}

		var address common.Address
		C.sputnikvm_log_address_copy(ptr,unsafe.Pointer(&address[0]))
		logs[logNo] = &state.Log{
			Address:address,
			Topics:topics,
			Data:data,
			BlockNumber:self.blockNumber,
			}
	}
	return
}

// Returns the current status of the VM
func (self *context) Status() (status vm.Status) {
	if self.ctxPtr != nil {
		switch st := C.sputnikvm_status(self.ctxPtr); st {
		case C.SPUTNIK_VM_RUNNING:
			status = vm.Running
		case C.SPUTNIK_VM_EXITED_OK:
			status = vm.ExitedOk
		case C.SPUTNIK_VM_EXITED_ERR:
			status = vm.ExitedErr
		case C.SPUTNIK_VM_UNSUPPORTED_ERR:
			status = vm.ExitedErr
		}
	} else {
		status = vm.Inactive
	}
	return
}

// Returns the current error in details
func (self *context) Err() (err error) {
	if self.ctxPtr != nil {
		ptr := C.sputnikvm_error(self.ctxPtr)
		if ptr != nil {
			err = errors.New(C.GoString(ptr))
		}
	}
	return
}

func (self *context) Context() unsafe.Pointer {
	if self.ctxPtr == nil {

		gas := getBintBytes(self.gas)
		price := getBintBytes(self.price)
		value := getBintBytes(self.value)
		var target unsafe.Pointer = nil
		if self.target != nil { target = unsafe.Pointer(&self.target[0]) }
		gasLimit := getBintBytes(self.gasLimit)
		difficulty := getBintBytes(self.difficulty)
		var data unsafe.Pointer = nil
		if self.data != nil && len(self.data) > 0 { data = unsafe.Pointer(&self.data[0])}

		self.ctxPtr = C.sputnikvm_context(
			unsafe.Pointer(&gas[0]),
			unsafe.Pointer(&price[0]),
			unsafe.Pointer(&value[0]),
			unsafe.Pointer(&self.caller[0]),
			target,
			data,
			C.size_t(len(self.data)),
			unsafe.Pointer(&gasLimit[0]),
			unsafe.Pointer(&self.coinbase[0]),
			C.int32_t(self.fork),
			C.uint64_t(self.blockNumber),
			C.uint64_t(self.time),
			unsafe.Pointer(&difficulty[0]))

	}

	return self.ctxPtr
}

// Run instructions until it reaches a `RequireErr` or exits
func (self *context) Fire() (*vm.Require) {

	if self.gasLimit == nil {
		self.st = vm.RequireErr
		req := &vm.Require{ID: vm.RequireInfo}
		glog.V(logger.Debug).Infof("Fire => %v\n",*req)
		return req
	}

	if self.subRequire != nil {
		req := self.subRequire
		self.subRequire = nil
		glog.V(logger.Debug).Infof("Fire => %v\n",*req)
		return req
	}

	ctx := self.Context()
	r := C.sputnikvm_fire(ctx)
	var req *vm.Require = nil
	switch r {
	case C.SPUTNIK_VM_REQUIRE_ACCOUNT:
		req = &vm.Require{ID: vm.RequireCode}
		C.sputnikvm_req_address_copy(ctx,unsafe.Pointer(&req.Address[0]))
		self.subRequire = &vm.Require{ID: vm.RequireAccount, Address: req.Address}
	case C.SPUTNIK_VM_REQUIRE_CODE:
		req = &vm.Require{ID: vm.RequireCode}
		C.sputnikvm_req_address_copy(ctx,unsafe.Pointer(&req.Address[0]))
	case C.SPUTNIK_VM_REQUIRE_HASH:
		req = &vm.Require{ID: vm.RequireHash}
		req.Number = uint64(C.sputnikvm_req_blocknum(ctx))
	case C.SPUTNIK_VM_REQUIRE_VALUE:
		req = &vm.Require{ID: vm.RequireValue}
		C.sputnikvm_req_address_copy(ctx,unsafe.Pointer(&req.Address[0]))
		C.sputnikvm_req_hash_copy(ctx,unsafe.Pointer(&req.Hash[0]))
	}
	glog.V(logger.Debug).Infof("Fire => %*v\n",req)
	return req
}

type account struct {
	ctx unsafe.Pointer
	ptr unsafe.Pointer
}

func (self *account) ChangeLevel() vm.AccountChangeLevel {
	return vm.AccountChangeLevel(C.sputnikvm_acc_change(self.ptr))
}

func (self *account) Address() (address common.Address) {
	C.sputnikvm_acc_address_copy(self.ptr,unsafe.Pointer(&address[0]))
	return
}

func (self *account) Nonce() uint64 {
	return uint64(C.sputnikvm_acc_nonce(self.ptr))
}

func (self *account) Balance() *big.Int {
	b := make([]big.Word,256/bits.UintSize)
	C.sputnikvm_acc_balance_copy(self.ptr,unsafe.Pointer(&b[0]))
	return new(big.Int).SetBits(b)
}

func (self *account) Code() (common.Hash, []byte) {
	code_len := C.int(C.sputnikvm_acc_code_len(self.ptr))
	if code_len > 0 {
		code := make([]byte, code_len)
		C.sputnikvm_acc_code_copy(self.ptr, unsafe.Pointer(&code[0]))
		hash := crypto.Keccak256Hash(code)
		return hash, code
	}
	return common.Hash{}, nil
}

func (self *account) Store(f func(common.Hash, common.Hash) error) error {

	var key common.Hash
	var val common.Hash

	keyPtr := unsafe.Pointer(&key[0])
	valPtr := unsafe.Pointer(&val[0])

	for ok := C.sputnikvm_acc_first_kv_copy(self.ctx,self.ptr,keyPtr,valPtr);
		ok != 0;
		ok = C.sputnikvm_acc_next_kv_copy(self.ctx,keyPtr,valPtr) {
		if err := f(key,val); err != nil {
			return err
		}
	}

	return nil
}

type suicided struct {
	address common.Address
}

func (self *suicided) ChangeLevel() vm.AccountChangeLevel { return vm.SuicideAccount }
func (self *suicided) Address() common.Address { return self.address }
func (self *suicided) Nonce() uint64 { return 0 } // ?!
func (self *suicided) Balance() *big.Int { return new(big.Int) }
func (self *suicided) Code() (common.Hash, []byte) { return common.Hash{}, nil }
func (self *suicided) Store(f func(common.Hash, common.Hash) error) error { return nil }
