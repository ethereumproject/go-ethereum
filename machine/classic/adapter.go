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

package classic

import (
	"math/big"

	"errors"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"reflect"
)

type command struct {
	code  uint8
	value interface{}
}

const (
	cmdStepID = iota
	cmdFireID
	cmdAccountID
	cmdHashID
	cmdCodeID
	cmdInfoID
	cmdValueID
)

var (
	cmdStep = &command{cmdStepID, nil}
	cmdFire = &command{cmdFireID, nil}
)

type machine struct {
	features int
}

type context struct {
	env vmEnv
}

type ChangeLevel byte

const (
	None ChangeLevel = iota
	Absent
	Committed
	Modified
	Suicided
)

type vmHash struct {
	number uint64
	hash   common.Hash
}

type snapshot interface {
	Value() int
}

type vmSnapshot struct {
	history      []snapshot
	currSnapshot *int
}

type vmAccountData struct {
	snapshot int
	balance  *big.Int
	level    ChangeLevel
	nonce    uint64
	newborn  bool
}

func (self *vmAccountData) Value() int { return self.snapshot }

type vmAccount struct {
	vmSnapshot
	address common.Address
	data    *vmAccountData
}

func newAccount(address common.Address, currSnapshot *int, level ChangeLevel) *vmAccount {
	acc := &vmAccount{
		address: address,
		data: &vmAccountData{
			*currSnapshot,
			new(big.Int),
			level,
			0,
			false},
	}
	acc.vmSnapshot.currSnapshot = currSnapshot
	return acc
}

func (self *vmSnapshot) popSnapshot(snapshot int) interface{} {
	for len(self.history) > 0 {
		l := len(self.history) - 1
		ss := self.history[l]
		self.history = self.history[:l]
		if ss.Value() <= snapshot {
			return ss
		}
	}
	return nil
}

func (self *vmAccount) RevertToSnapshot(snapshot int) bool {
	if self.data.snapshot <= snapshot {
		return true
	}
	if ss := self.popSnapshot(snapshot); ss != nil {
		self.data = ss.(*vmAccountData)
		return true
	}
	return false
}

func (self *vmAccount) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	self.setBalance(new(big.Int).Sub(self.Balance(), amount))
}

func (self *vmAccount) AddBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	self.setBalance(new(big.Int).Add(self.Balance(), amount))
}

func (self *vmAccount) Snap() {
	if *self.currSnapshot < self.data.snapshot {
		panic("assertion: invalid snapshot value")
	}
	if *self.currSnapshot != self.data.snapshot {
		data := self.data
		self.history = append(self.history, data)
		self.data = &vmAccountData{
			*self.currSnapshot,
			data.balance,
			data.level,
			data.nonce,
			data.newborn,
		}
	}
	if self.data.level != Suicided {
		self.data.level = Modified
	}
}

func (self *vmAccount) SetNewborn() {
	self.SetNonce(0)
	self.data.newborn = true
}

func (self *vmAccount) IsNewborn() bool {
	return self.data.newborn
}

func (self *vmAccount) SetBalance(amount *big.Int) {
	self.setBalance(new(big.Int).Set(amount))
}

func (self *vmAccount) setBalance(amount *big.Int) {
	self.Snap()
	self.data.balance = amount
}

func (self *vmAccount) SetNonce(nonce uint64) {
	self.Snap()
	self.data.nonce = nonce
}

func (self *vmAccount) Suicide() bool {
	switch self.data.level {
	case Committed, Modified, Suicided:
		self.Snap()
		self.data.balance = new(big.Int)
		self.data.level = Suicided
		return true
	default:
		return false
	}
}

func (self *vmAccount) ReturnGas(gas, price *big.Int) {}

func (self *vmAccount) Nonce() uint64           { return self.data.nonce }
func (self *vmAccount) Balance() *big.Int       { return new(big.Int).Set(self.data.balance) }
func (self *vmAccount) Address() common.Address { return self.address }

func (self *vmAccount) SetCode(hash common.Hash, code []byte)               { panic("unimplemented") }
func (self *vmAccount) ForEachStorage(cb func(key, value common.Hash) bool) { panic("unimplemented") }

func (self *vmAccount) Level() ChangeLevel {
	return self.data.level
}

type vmCodeData struct {
	code  []byte
	hash  common.Hash
	level ChangeLevel

	snapshot int
}

func (self *vmCodeData) Value() int { return self.snapshot }

type vmCode struct {
	vmSnapshot
	address common.Address
	data    *vmCodeData
}

func (self *vmCode) RevertToSnapshot(snapshot int) bool {
	if self.data.snapshot <= snapshot {
		return true
	}
	if ss := self.popSnapshot(snapshot); ss != nil {
		self.data = ss.(*vmCodeData)
		return true
	}
	return false
}

func newCode(address common.Address, currSnapshot *int, level ChangeLevel) *vmCode {
	code := &vmCode{
		address: address,
		data: &vmCodeData{
			snapshot: *currSnapshot,
			level:    level},
	}
	code.vmSnapshot.currSnapshot = currSnapshot
	return code
}

func (self *vmCode) Snap() {
	if *self.currSnapshot < self.data.snapshot {
		panic("assertion: invalid snapshot value")
	}
	if *self.currSnapshot != self.data.snapshot {
		data := self.data
		self.history = append(self.history, data)
		self.data = &vmCodeData{
			data.code,
			data.hash,
			data.level,
			*self.currSnapshot,
		}
	}
	self.data.level = Modified
}

func (self *vmCode) Code() []byte {
	switch self.data.level {
	case Committed, Modified, Suicided:
		return self.data.code
	default:
		return nil
	}
}

func (self *vmCode) Hash() common.Hash {
	switch self.data.level {
	case Committed, Modified, Suicided:
		return self.data.hash
	default:
		return common.Hash{}
	}
}

func (self *vmCode) SetCode(hash common.Hash, code []byte) {
	self.Snap()
	self.data.code = code
	self.data.hash = hash
}

func (self *vmCode) Level() ChangeLevel {
	return self.data.level
}

type vmInfo struct {
	table      *vm.GasTable
	fork       vm.Fork
	difficulty *big.Int
	gasLimit   *big.Int
	time       *big.Int
	coinbase   common.Address
	number     uint64
}

type vmValueData struct {
	value    common.Hash
	level    ChangeLevel
	snapshot int
}

func (self *vmValueData) Value() int { return self.snapshot }

type vmValue struct {
	vmSnapshot
	address common.Address
	key     common.Hash
	data    *vmValueData
}

func (self *vmValue) RevertToSnapshot(snapshot int) bool {
	if self.data.snapshot <= snapshot {
		return true
	}
	if ss := self.popSnapshot(snapshot); ss != nil {
		self.data = ss.(*vmValueData)
		return true
	}
	return false
}

func newValue(address common.Address, key common.Hash, currSnapshot *int, level ChangeLevel) *vmValue {
	value := &vmValue{
		address: address,
		key:     key,
		data: &vmValueData{
			snapshot: *currSnapshot,
			level:    level},
	}
	value.vmSnapshot.currSnapshot = currSnapshot
	return value
}

func (self *vmValue) Address() common.Address {
	return self.address
}

func (self *vmValue) Key() common.Hash {
	return self.key
}

func (self *vmValue) Value() common.Hash {
	return self.data.value
}

func (self *vmValue) Snap() {
	if *self.currSnapshot < self.data.snapshot {
		panic("assertion: invalid snapshot value")
	}
	if *self.currSnapshot != self.data.snapshot {
		data := self.data
		self.history = append(self.history, data)
		self.data = &vmValueData{
			data.value,
			data.level,
			*self.currSnapshot,
		}
	}
	self.data.level = Modified
}

func (self *vmValue) Set(value common.Hash) {
	self.Snap()
	self.data.value = value
}

func (self *vmValue) Get() common.Hash {
	switch self.data.level {
	case Committed, Modified:
		return self.data.value
	default:
		return common.Hash{}
	}
}

func (self *vmValue) Level() ChangeLevel {
	return self.data.level
}

type vmLog struct {
	log      *state.Log
	snapshot int
}

type vmRefund struct {
	gas      *big.Int
	snapshot int
}

type vmEnv struct {
	cmdc       chan *command
	rqc        chan *vm.Require
	info       *vmInfo
	evm        *EVM
	depth      int
	contract   *Contract
	output     []byte
	stepByStep bool

	followingTransfer bool

	origin  common.Address
	address common.Address

	account map[common.Address]*vmAccount
	code    map[common.Address]*vmCode
	hash    map[uint64]*vmHash
	value   map[string]*vmValue

	logs []vmLog

	Gas       *big.Int
	gasRefund []vmRefund

	status  vm.Status
	err     error
	machine *machine

	currSnapshot int
}

func (self *vmEnv) SnapshotDatabase() int {
	snapshot := self.currSnapshot
	self.currSnapshot = self.currSnapshot + 1
	return snapshot
}

func (self *vmEnv) RevertAccountsToSnapshot(snapshot int) {
	keys := reflect.ValueOf(self.account).MapKeys()
	for _, k := range keys {
		addr := k.Interface().(common.Address)
		if !self.account[addr].RevertToSnapshot(snapshot) {
			delete(self.account, addr)
		}
	}
}

func (self *vmEnv) RevertCodesToSnapshot(snapshot int) {
	keys := reflect.ValueOf(self.code).MapKeys()
	for _, k := range keys {
		addr := k.Interface().(common.Address)
		if !self.code[addr].RevertToSnapshot(snapshot) {
			delete(self.code, addr)
		}
	}
}

func (self *vmEnv) RevertValuesToSnapshot(snapshot int) {
	keys := reflect.ValueOf(self.value).MapKeys()
	for _, k := range keys {
		addr := k.Interface().(string)
		if !self.value[addr].RevertToSnapshot(snapshot) {
			delete(self.value, addr)
		}
	}
}

func (self *vmEnv) RevertLogsToSnapshot(snapshot int) {
	for {
		l := len(self.logs)
		if l == 0 || self.logs[l-1].snapshot <= snapshot {
			return
		}
		self.logs = self.logs[:l-1]
	}
}

func (self *vmEnv) RevertRefundToSnapshot(snapshot int) {
	for {
		l := len(self.gasRefund)
		if l == 0 || self.gasRefund[l-1].snapshot <= snapshot {
			return
		}
		self.gasRefund = self.gasRefund[:l-1]
	}
}

func (self *vmEnv) RevertToSnapshot(snapshot int) {
	self.RevertAccountsToSnapshot(snapshot)
	self.RevertCodesToSnapshot(snapshot)
	self.RevertValuesToSnapshot(snapshot)
	self.RevertLogsToSnapshot(snapshot)
	self.RevertRefundToSnapshot(snapshot)
}

func NewMachine() vm.Machine {
	return &machine{}
}

func mkEnv(env *vmEnv, m *machine) *vmEnv {
	env.rqc = make(chan *vm.Require)
	env.cmdc = make(chan *command)
	env.account = make(map[common.Address]*vmAccount)
	env.code = make(map[common.Address]*vmCode)
	env.hash = make(map[uint64]*vmHash)
	env.value = make(map[string]*vmValue)
	env.status = vm.Inactive
	env.machine = m
	env.gasRefund = []vmRefund{{new(big.Int), 0}}
	return env
}

func (self *machine) Name() string {
	return "CLASSIC VM"
}

func (self *machine) Type() vm.Type {
	return vm.ClassicVm
}

func (self *machine) SetTestFeatures(features int) error {
	self.features = features
	return nil
}

func (self *vmEnv) updateStatus() {
	if self.err == nil {
		self.status = vm.ExitedOk
	} else if self.err == OutOfGasError {
		self.status = vm.OutOfGasErr
	} else if IsValueTransferErr(self.err) {
		self.status = vm.TransferErr
	} else {
		self.status = vm.ExitedErr
	}
}

func (self *machine) Call(caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (vm.Context, error) {
	ctx := &context{}
	go func(e *vmEnv) {
		e.origin = caller
		for {
			cmd, ok := <-e.cmdc
			if !ok {
				close(e.rqc)
				return
			}
			if cmd.code == cmdStepID || cmd.code == cmdFireID {
				e.status = vm.Running
				e.stepByStep = cmd.code == cmdStepID
				callerRef := e.queryAccount(caller)
				e.address = to
				e.evm = NewVM(e)
				e.Gas = new(big.Int).Set(gas)
				e.output, e.err = e.Call(callerRef, to, data, e.Gas, price, value)
				e.updateStatus()
				e.rqc <- nil
				close(e.rqc)
				return
			} else {
				e.handleCmdc(cmd)
			}
		}
	}(mkEnv(&ctx.env, self))

	return ctx, nil
}

func (self *machine) Create(caller common.Address, code []byte, gas, price, value *big.Int) (vm.Context, error) {
	ctx := &context{}

	go func(e *vmEnv) {
		e.origin = caller
		for {
			cmd, ok := <-e.cmdc
			if !ok {
				return
			}
			if cmd.code == cmdStepID || cmd.code == cmdFireID {
				e.status = vm.Running
				e.stepByStep = cmd.code == cmdStepID
				callerRef := e.queryAccount(caller)
				e.evm = NewVM(e)
				e.Gas = new(big.Int).Set(gas)
				e.output, e.address, e.err = e.Create(callerRef, code, e.Gas, price, value)
				e.updateStatus()
				e.rqc <- nil
				return
			} else {
				e.handleCmdc(cmd)
			}
		}
	}(mkEnv(&ctx.env, self))

	return ctx, nil
}

func (self *vmEnv) handleCmdc(cmd *command) bool {
	switch cmd.code {
	case cmdHashID:
		h := cmd.value.(*vmHash)
		self.hash[h.number] = h
	case cmdCodeID:
		c := cmd.value.(*vmCode)
		self.code[c.address] = c
	case cmdAccountID:
		a := cmd.value.(*vmAccount)
		self.account[a.address] = a
	case cmdValueID:
		v := cmd.value.(*vmValue)
		self.value[v.address.Hex()+":"+v.key.Hex()] = v
	case cmdInfoID:
		self.info = cmd.value.(*vmInfo)
	case cmdStepID:
		self.stepByStep = true
		return true
	case cmdFireID:
		self.stepByStep = false
		return true
	default:
		panic("invalid command")
	}
	return false
}

func (self *vmEnv) handleRequire(rq *vm.Require) {
	self.status = vm.RequireErr
	self.rqc <- rq
	for {
		cmd := <-self.cmdc
		if self.handleCmdc(cmd) {
			self.status = vm.Running
			return
		}
	}
}

func (self *vmEnv) brokeVm(text string) {
	self.status = vm.Broken
	self.err = errors.New(text)
	self.rqc <- nil
	panic(vm.BrokenError)
}

func (self *vmEnv) queryCode(addr common.Address) *vmCode {
	for {
		if c, exists := self.code[addr]; exists {
			return c
		} else {
			self.handleRequire(&vm.Require{ID: vm.RequireCode, Address: addr})
		}
	}
}

func (self *vmEnv) queryAccount(addr common.Address) *vmAccount {
	for {
		if a, exists := self.account[addr]; exists {
			return a
		} else {
			self.handleRequire(&vm.Require{ID: vm.RequireAccount, Address: addr})
		}
	}
}

func (self *vmEnv) queryOrCreateAccount(addr common.Address) *vmAccount {
	acc := self.queryAccount(addr)
	switch acc.Level() {
	case Modified, Committed:
		return acc
	default:
		acc.Snap()
		return acc
	}
}

func (self *vmEnv) queryValue(addr common.Address, key common.Hash) *vmValue {
	valid := addr.Hex() + ":" + key.Hex()
	for {
		if v, exists := self.value[valid]; exists {
			return v
		} else {
			if acc := self.account[addr]; acc != nil && acc.IsNewborn() {
				self.value[valid] = newValue(addr, key, &self.currSnapshot, Absent)
			} else {
				self.handleRequire(&vm.Require{ID: vm.RequireValue, Address: addr, Hash: key})
			}
		}
	}
}

func (self *vmEnv) queryInfo() *vmInfo {
	for {
		if self.info != nil {
			return self.info
		} else {
			self.handleRequire(&vm.Require{ID: vm.RequireInfo})
		}
	}
}

func (self *context) CommitInfo(blockNumber uint64, coinbase common.Address, table *vm.GasTable, fork vm.Fork, difficulty, gasLimit, time *big.Int) (err error) {
	info := vmInfo{table, fork, difficulty, gasLimit, time, coinbase, blockNumber}
	self.env.cmdc <- &command{cmdInfoID, &info}
	return
}

func (self *context) CommitAccount(address common.Address, nonce uint64, balance *big.Int) (err error) {
	a := newAccount(address, &self.env.currSnapshot, Absent)
	if balance != nil {
		a.data.balance = balance
		a.data.nonce = nonce
		a.data.level = Committed
	}
	self.env.cmdc <- &command{cmdAccountID, a}
	return
}

func (self *context) CommitBlockhash(number uint64, hash common.Hash) (err error) {
	h := vmHash{number, hash}
	self.env.cmdc <- &command{cmdHashID, &h}
	return
}

func (self *context) CommitCode(address common.Address, hash common.Hash, code []byte) (err error) {
	c := newCode(address, &self.env.currSnapshot, Absent)
	if code != nil {
		c.data.code = code
		c.data.hash = hash
		c.data.level = Committed
	}
	self.env.cmdc <- &command{cmdCodeID, c}
	return
}

func (self *context) CommitValue(address common.Address, key common.Hash, value common.Hash) (err error) {
	v := newValue(address, key, &self.env.currSnapshot, Committed)
	v.data.value = value
	self.env.cmdc <- &command{cmdValueID, v}
	return
}

func (self *context) Status() vm.Status {
	return self.env.status
}

func (self *context) Finish() (err error) {
	return nil
}

func (self *context) Fire() *vm.Require {
	self.env.cmdc <- cmdFire
	rq := <-self.env.rqc
	return rq
}

func (self *context) Code(addr common.Address) (common.Hash, []byte, error) {
	if c, exists := self.env.code[addr]; exists {
		switch c.Level() {
		case Modified, Committed:
			return c.Hash(), c.Code(), nil
		}
	}
	return common.Hash{}, nil, nil
}

type account struct {
	acc *vmAccount
	env *vmEnv
}

func (self *account) ChangeLevel() vm.AccountChangeLevel {

	change := self.acc.Level()
	if change ==  Suicided { return vm.SuicideAccount }
	if self.acc.IsNewborn() { return vm.CreateAccount }
	return vm.UpdateAccount
}

func (self *account) Address() common.Address { return self.acc.address }
func (self *account) Nonce() uint64           { return self.acc.Nonce() }
func (self *account) Balance() *big.Int       { return self.acc.Balance() }
func (self *account) Code() (common.Hash, []byte) {
	address := self.acc.address
	if code, exists := self.env.code[address]; exists && code.Level() == Modified {
		return code.Hash(), code.Code()
	}
	return common.Hash{}, nil
}
func (self *account) Store(f func(common.Hash, common.Hash) error) error {
	address := self.acc.address
	for _, v := range self.env.value {
		if v.Address() == address {
			if err := f(v.Key(), v.Value()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *context) Accounts() (accounts []vm.Account, err error) {
	accounts = []vm.Account{}
	for _, a := range self.env.account {
		switch a.Level() {
		case Modified, Suicided:
			accounts = append(accounts, &account{a, &self.env})
		}
	}
	return
}

func (self *context) Out() ([]byte, *big.Int, *big.Int, error) {
	return self.env.output, self.env.Gas, self.env.GetRefund(), nil
}

func (self *context) Logs() (logs state.Logs, err error) {
	for _, l := range self.env.logs {
		logs = append(logs, l.log)
	}
	return
}

func (self *context) Err() error {
	return self.env.err
}

func (self *vmEnv) GasTable(block *big.Int) *vm.GasTable {
	info := self.queryInfo()
	if info.number != block.Uint64() {
		self.brokeVm("invalid block number in RulesSet.GasTable")
	}
	return self.queryInfo().table
}

func (self *vmEnv) IsHomestead(block *big.Int) bool {
	info := self.queryInfo()
	if info.number != block.Uint64() {
		self.brokeVm("invalid block number in RulesSet.IsHomestead")
	}
	return self.queryInfo().fork >= vm.Homestead
}

func (self *vmEnv) RuleSet() vm.RuleSet {
	return self
}

func (self *vmEnv) Origin() common.Address {
	return self.origin
}

func (self *vmEnv) BlockNumber() *big.Int {
	return new(big.Int).SetUint64(self.queryInfo().number)
}

func (self *vmEnv) Coinbase() common.Address {
	return self.queryInfo().coinbase
}

func (self *vmEnv) Time() *big.Int {
	return self.queryInfo().time
}

func (self *vmEnv) Difficulty() *big.Int {
	return self.queryInfo().difficulty
}

func (self *vmEnv) GasLimit() *big.Int {
	return self.queryInfo().gasLimit
}

func (self *vmEnv) Db() Database {
	return self
}

func (self *vmEnv) Depth() int {
	return self.depth
}

func (self *vmEnv) SetDepth(i int) {
	self.depth = i
}

func (self *vmEnv) GetHash(n uint64) common.Hash {
	for {
		if h, exists := self.hash[n]; exists {
			return h.hash
		} else {
			self.handleRequire(&vm.Require{ID: vm.RequireHash, Number: n})
		}
	}
}

func (self *vmEnv) AddLog(log *state.Log) {
	self.logs = append(self.logs, vmLog{log, self.currSnapshot})
}

func (self *vmEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	if (self.machine.features & vm.TestSkipTransfer) > 0 {
		if !self.followingTransfer {
			self.followingTransfer = true
			return true
		}
	}
	return self.GetBalance(from).Cmp(balance) >= 0
}

func (self *vmEnv) Transfer(from, to state.AccountObject, amount *big.Int) {
	if (self.machine.features & vm.TestSkipTransfer) == 0 {
		Transfer(from, to, amount)
	}
}

func (self *vmEnv) Call(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	if (self.machine.features&vm.TestSkipSubCall) > 0 && self.depth > 0 {
		me.ReturnGas(gas, price)
		return nil, nil
	}
	return Call(self, me, addr, data, gas, price, value)
}

func (self *vmEnv) CallCode(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	if (self.machine.features&vm.TestSkipSubCall) > 0 && self.depth > 0 {
		me.ReturnGas(gas, price)
		return nil, nil
	}
	return CallCode(self, me, addr, data, gas, price, value)
}

func (self *vmEnv) DelegateCall(me ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	if (self.machine.features&vm.TestSkipSubCall) > 0 && self.depth > 0 {
		me.ReturnGas(gas, price)
		return nil, nil
	}
	return DelegateCall(self, me.(*Contract), addr, data, gas, price)
}

func (self *vmEnv) Create(me ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	if (self.machine.features & vm.TestSkipCreate) > 0 {
		me.ReturnGas(gas, price)
		caller := self.queryAccount(me.Address())
		nonce := caller.Nonce()
		obj := self.queryOrCreateAccount(crypto.CreateAddress(me.Address(), nonce))
		return nil, obj.Address(), nil
	}
	return Create(self, me, data, gas, price, value)
}

func (self *vmEnv) Run(contract *Contract, input []byte) (ret []byte, err error) {
	return self.evm.Run(contract, input)
}

func (self *vmEnv) GetAccount(addr common.Address) state.AccountObject {
	acc := self.queryAccount(addr)
	switch acc.Level() {
	case Modified, Committed, Suicided:
		return acc
	default:
		return nil
	}
}

func (self *vmEnv) CreateAccount(addr common.Address) state.AccountObject {
	acc := self.queryAccount(addr)
	acc.SetNewborn()
	for _, v := range self.value {
		if v.Address() == addr {
			v.Set(common.Hash{})
		}
	}
	return acc
}

func (self *vmEnv) AddBalance(addr common.Address, ammount *big.Int) {
	acc := self.queryOrCreateAccount(addr)
	acc.AddBalance(ammount)
}

func (self *vmEnv) GetBalance(addr common.Address) *big.Int {
	return self.queryAccount(addr).Balance()
}

func (self *vmEnv) GetNonce(addr common.Address) uint64 {
	return self.queryAccount(addr).Nonce()
}

func (self *vmEnv) SetNonce(addr common.Address, nonce uint64) {
	self.queryAccount(addr).SetNonce(nonce)
}

func (self *vmEnv) GetCodeHash(addr common.Address) common.Hash {
	return self.queryCode(addr).Hash()
}

func (self *vmEnv) GetCodeSize(addr common.Address) int {
	return len(self.queryCode(addr).Code())
}

func (self *vmEnv) GetCode(addr common.Address) []byte {
	return self.queryCode(addr).Code()
}

func (self *vmEnv) SetCode(addr common.Address, code []byte) {
	self.queryOrCreateAccount(addr)
	self.queryCode(addr).SetCode(crypto.Keccak256Hash(code), code)
}

func (self *vmEnv) AddRefund(gas *big.Int) {
	refund := self.gasRefund[len(self.gasRefund)-1]
	if refund.snapshot < self.currSnapshot {
		g := new(big.Int)
		g.Add(refund.gas, gas)
		self.gasRefund = append(self.gasRefund, vmRefund{g, self.currSnapshot})
	} else {
		refund.gas.Add(refund.gas, gas)
	}
}

func (self *vmEnv) GetRefund() *big.Int {
	return self.gasRefund[len(self.gasRefund)-1].gas
}

func (self *vmEnv) GetState(address common.Address, key common.Hash) common.Hash {
	return self.queryValue(address, key).Get()
}

func (self *vmEnv) SetState(address common.Address, key common.Hash, value common.Hash) {
	self.queryOrCreateAccount(address).Snap()
	self.queryValue(address, key).Set(value)
}

func (self *vmEnv) Suicide(addr common.Address) bool {
	return self.queryAccount(addr).Suicide()
}

func (self *vmEnv) HasSuicided(addr common.Address) bool {
	return self.queryAccount(addr).Level() == Suicided
}

func (self *vmEnv) Exist(addr common.Address) bool {
	switch self.queryAccount(addr).Level() {
	case Committed, Modified, Suicided:
		return true
	}
	return false
}
