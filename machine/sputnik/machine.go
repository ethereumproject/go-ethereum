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

package sputnik

/*

#cgo amd64,windows LDFLAGS: -L${SRCDIR}/windows_amd64 -lsputnikvm
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
	"fmt"
	"os"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

type machine struct {
}

type context struct {
	machine *machine

	gas    string
	price  string
	value  string
	caller string
	target string
	data   []byte

	st     vm.Status

	gasLimit 	string
	coinbase 	string
	fork     	vm.Fork
	blocknum 	string
	time       	uint64
	difficulty 	string

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

	ctx.gas = gas.Text(16)
	ctx.price = price.Text(16)
	ctx.value = value.Text(16)
	ctx.caller = caller.Hex()
	ctx.target = to.Hex()
	ctx.data = data

	return ctx, nil
}

// Create contract in new VM context
func (self *machine) Create(caller common.Address, code []byte, gas, price, value *big.Int) (vm.Context, error) {
	ctx := &context{machine: self, st: vm.Inactive}

	ctx.gas = gas.Text(16)
	ctx.price = price.Text(16)
	ctx.value = value.Text(16)
	ctx.caller = caller.Hex()
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

// Commit an account information
func (self *context) CommitAccount(address common.Address, nonce uint64, balance *big.Int) (err error) {
	addressPtr := C.CString(address.Hex())
	defer C.free(unsafe.Pointer(addressPtr))
	balancePtr := C.CString(balance.Text(16))
	defer C.free(unsafe.Pointer(balancePtr))
	var codePtr unsafe.Pointer = nil
	if self.subCode != nil {
		codePtr = C.CBytes(self.subCode)
		defer C.free(codePtr)
	}
	C.sputnikvm_commit_account(self.Context(),addressPtr,C.uint64_t(nonce),balancePtr,codePtr,C.size_t(len(self.subCode)))
	self.subCode = nil
	return
}

// Commit a block hash
func (self *context) CommitBlockhash(number uint64, hash common.Hash) (err error) {
	return
}

// Commit a contract code
func (self *context) CommitCode(address common.Address, hash common.Hash, code []byte) (err error) {
	if self.subRequire != nil {
		self.subCode = code
	} else {
		addressPtr := C.CString(address.Hex())
		defer C.free(unsafe.Pointer(addressPtr))
		var codePtr unsafe.Pointer = nil
		if code != nil {
			codePtr = C.CBytes(code)
			defer C.free(codePtr)
		}
		C.sputnikvm_commit_code(self.Context(), addressPtr, codePtr, C.size_t(len(code)))
	}
	return
}

// Commit an info
func (self *context) CommitInfo(blockNumber uint64, coinbase common.Address, table *vm.GasTable, fork vm.Fork, difficulty, gasLimit, time *big.Int) (err error) {
	self.gasLimit = gasLimit.Text(16)
	self.coinbase = coinbase.Hex()
	self.fork = fork
	self.blocknum = new(big.Int).SetUint64(blockNumber).Text(16)
	self.time = time.Uint64()
	self.difficulty = difficulty.Text(16)
	return
}

// Commit a state
func (self *context) CommitValue(address common.Address, key common.Hash, value common.Hash) (err error) {
	return
}

// Finish VM context
func (self *context) Finish() (err error) {
	return
}

// Returns the changed accounts information
func (self *context) Accounts() (accounts []vm.Account, err error) {
	return
}

// Returns new contract address on Create and called contract address on Call
func (self *context) Address() (address common.Address, err error) {
	return
}

// Returns the out value and gas remaining/refunded if any
func (self *context) Out() (out []byte, gas *big.Int, refund *big.Int, err error) {
	return
}

// Returns logs to be appended to the current block if the user
// decided to accept the running status of this VM
func (self *context) Logs() (logs state.Logs, err error) {
	return
}

// Returns the current status of the VM
func (self *context) Status() (status vm.Status) {
	return
}

// Returns the current error in details
func (self *context) Err() (err error) {
	return
}

func (self *context) Context() unsafe.Pointer {

	if self.ctxPtr == nil {

		var op C.int32_t = C.SPUTNIK_VM_CALL
		if len(self.target) == 0 {
			op = C.SPUTNIK_VM_CREATE
		}

		gas := C.CString(self.gas)
		defer C.free(unsafe.Pointer(gas))
		price := C.CString(self.price)
		defer C.free(unsafe.Pointer(price))
		value := C.CString(self.value)
		defer C.free(unsafe.Pointer(value))
		caller := C.CString(self.caller)
		defer C.free(unsafe.Pointer(caller))
		target := C.CString(self.target)
		defer C.free(unsafe.Pointer(target))
		data := C.CBytes(self.data)
		defer C.free(data)
		gasLimit := C.CString(self.gasLimit)
		defer C.free(unsafe.Pointer(gasLimit))
		coinbase := C.CString(self.coinbase)
		defer C.free(unsafe.Pointer(coinbase))
		difficulty := C.CString(self.difficulty)
		defer C.free(unsafe.Pointer(difficulty))
		blocknum := C.CString(self.blocknum)

		self.ctxPtr = C.sputnikvm_context(
			op,
			gas,
			price,
			value,
			caller,
			target,
			data,
			C.size_t(len(self.data)),
			gasLimit,
			coinbase,
			C.int32_t(self.fork),
			blocknum,
			C.uint64_t(self.time),
			difficulty)

	}

	return self.ctxPtr
}

// Run instructions until it reaches a `RequireErr` or exits
func (self *context) Fire() (*vm.Require) {

	if len(self.gasLimit) == 0 {
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

	fmt.Fprintf(os.Stderr,"fire => %v\n", r)

	switch r {
	case C.SPUTNIK_VM_REQUIRE_ACCOUNT:
		address := C.GoString(C.sputnikvm_req_address(ctx))
		self.subRequire = &vm.Require{ID: vm.RequireAccount, Address: common.HexToAddress(address) }
		req := &vm.Require{ID: vm.RequireCode, Address: common.HexToAddress(address) }
		glog.V(logger.Debug).Infof("Fire => %v\n",*req)
		return req
	case C.SPUTNIK_VM_REQUIRE_CODE:
		address := C.GoString(C.sputnikvm_req_address(ctx))
		req := &vm.Require{ID: vm.RequireCode, Address: common.HexToAddress(address) }
		glog.V(logger.Debug).Infof("Fire => %v\n",*req)
		return req
	case C.SPUTNIK_VM_REQUIRE_HASH:
		blocknum := uint64(C.sputnikvm_req_blocknum(ctx))
		req := &vm.Require{ID: vm.RequireHash, Number: blocknum}
		glog.V(logger.Debug).Infof("Fire => %v\n",*req)
		return req
	case C.SPUTNIK_VM_REQUIRE_VALUE:
		address := C.GoString(C.sputnikvm_req_address(ctx))
		key := C.GoString(C.sputnikvm_req_hash(ctx))
		req := &vm.Require{ID: vm.RequireValue, Address: common.HexToAddress(address), Hash: common.HexToHash(key) }
		glog.V(logger.Debug).Infof("Fire => %v\n",*req)
		return req
	}

	return nil
}
