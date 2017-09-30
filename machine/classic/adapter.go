package classic

import (
	"math/big"
	"errors"

	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/common"
)

type command struct {
	code uint8
	value interface{}
}

const (
	cmdStepID = iota
	cmdFireID
	cmdAccountID
	cmdHashID
	cmdCodeID
	cmdRuleID
)

var (
	cmdStep = &command{cmdStepID, nil}
	cmdFire = &command{cmdFireID, nil}
)

type vmHash struct {
	number uint64
	hash   common.Hash
}

type vmAccount struct {
	address common.Address
	nonce   uint64
	balance *big.Int
	storage map[string]string
	changes map[string]bool
	modified bool
}

func (self *vmAccount) SubBalance(amount *big.Int) {}
func (self *vmAccount) AddBalance(amount *big.Int) {}
func (self *vmAccount) SetBalance(amount *big.Int) { self.balance = new(big.Int).Set(amount) }
func (self *vmAccount) SetNonce(nonce uint64) { self.nonce = nonce }
func (self *vmAccount) Nonce() uint64 { return self.nonce }
func (self *vmAccount) Balance() *big.Int { return self.balance }
func (self *vmAccount) Address() common.Address { return self.address }
func (self *vmAccount) ReturnGas(*big.Int, *big.Int) {}
func (self *vmAccount) SetCode(common.Hash, []byte) {}
func (self *vmAccount) ForEachStorage(cb func(key, value common.Hash) bool) {}
func (self *vmAccount) Set(vm.Account) {}
func (self *vmAccount) Code() []byte { return nil }
func (self *vmAccount) CodeHash() common.Hash { return common.Hash{} }

type vmCode struct {
	address common.Address
	code []byte
	size int
	hash common.Hash
}

func (self *vmCode) Address() common.Address {
	return self.address
}

func (self *vmCode) Code() []byte {
	return self.code
}

func (self *vmCode) Hash() common.Hash {
	return self.hash
}

func (self *vmCode) Size() int {
	return self.size
}

type vmRule struct {
	number uint64
	table *vm.GasTable
	fork  vm.Fork
}

type vmEnv struct {
	cmdc    	chan *command
	rqc			chan *vm.Require
	rules   	map[uint64]*vmRule
	evm     	*EVM
	depth   	int
	contract 	*Contract
	input   	[]byte
	output      []byte
	stepByStep	bool

	address     common.Address
	accounts 	map[common.Address]*vmAccount
	codes 		map[common.Address]*vmCode
	hashes 		map[uint64]*vmHash
	origin 		common.Address
	coinbase 	common.Address
	blocknumber int64

	err 		error
}

type vmAdapter struct {
	status vm.Status
	env    *vmEnv
}

func NewMachine() vm.Machine {
	adapter := &vmAdapter{}
	return adapter
}

func (self *vmAdapter) Address() (common.Address, error) {
	if self.env == nil {
		return common.Address{}, vm.TerminatedError
	} else {
		return self.env.address, nil
	}
}

func (self *vmAdapter) CommitRules(number uint64, table *vm.GasTable, fork vm.Fork) (err error) {
	rule := vmRule{number,table,fork}
	self.env.cmdc <- &command{cmdRuleID, &rule}
	return
}

func (self *vmAdapter) 	CommitAccount(address common.Address, nonce uint64, balance *big.Int) (err error) {
	account := vmAccount{address:address,nonce:nonce,balance:balance}
	self.env.cmdc <- &command{cmdAccountID, &account}
	return
}

func (self *vmAdapter) CommitBlockhash(number uint64, hash common.Hash) (err error) {
	value := vmHash{number, hash}
	self.env.cmdc <- &command{cmdHashID, &value}
	return
}

func (self *vmAdapter) CommitCode(address common.Address, hash common.Hash, code []byte) (err error) {
	value := vmCode{address, code, len(code), hash}
	self.env.cmdc <- &command{cmdCodeID, &value}
	return
}

func (self *vmAdapter) Status() vm.Status {
	return self.status
}

func (self *vmEnv) handleCmdc(cmd *command) {
	switch cmd.code {
	case cmdHashID:
		h := cmd.value.(*vmHash)
		self.hashes[h.number] = h
	case cmdCodeID:
		c := cmd.value.(*vmCode)
		self.codes[c.Address()] = c
	case cmdAccountID:
		a := cmd.value.(*vmAccount)
		self.accounts[a.Address()] = a
	case cmdRuleID:
		r := cmd.value.(*vmRule)
		self.rules[r.number] = r
	case cmdStepID:
		self.stepByStep = true
		return
	case cmdFireID:
		self.stepByStep = false
		return
	default:
		panic("invalid command")
	}
}

func (self *vmEnv) handleOnError(rq *vm.Require) {
	self.rqc <- rq
	for {
		cmd := <-self.cmdc
		self.handleCmdc(cmd)
	}
}

func (self *vmAdapter) Call(blockNumber uint64, caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (ctx vm.Context, err error) {
	if self.env == nil {
		ctx = self
		env := &vmEnv{}
		env.evm = NewVM(env)
		self.env = env
		go func (e *vmEnv) {
			cmd, ok := <-e.cmdc
			if !ok {
				return
			}
			if cmd.code == cmdStepID || cmd.code == cmdFireID {
				e.stepByStep = cmd.code == cmdStepID
				callerRef := e.QueryAccount(caller)
				e.address = to
				e.output, e.err = e.Call(callerRef,to,data,gas,price,value)
				e.rqc <- nil
				return
			} else {
				e.handleCmdc(cmd)
			}
		}(env)
	} else {
		err = errors.New("already started")
	}
	return
}

func (self *vmAdapter) Create(blockNumber uint64, caller common.Address, code []byte, gas, price, value *big.Int) (ctx vm.Context, err error) {
	if self.env == nil {
		ctx = self
		env := &vmEnv{}
		env.evm = NewVM(env)
		self.env = env
		go func (e *vmEnv) {
			cmd, ok := <-e.cmdc
			if !ok {
				return
			}
			if cmd.code == cmdStepID || cmd.code == cmdFireID {
				e.stepByStep = cmd.code == cmdStepID
				callerRef := e.QueryAccount(caller)
				e.output, e.address, e.err = e.Create(callerRef,code,gas,price,value)
				e.rqc <- nil
				return
			} else {
				e.handleCmdc(cmd)
			}
		}(env)
	} else {
		err = errors.New("already started")
	}
	return
}


func (self *vmAdapter) Finish() (err error) {
	if self.env == nil {
		err = errors.New("not started")
	} else {
		close(self.env.cmdc)
		self.env = nil
	}
	return
}


func (self *vmAdapter) Debug() (vm.Debugger, error) {
	return nil, nil
}

func (self *vmAdapter) Fire() (*vm.Require) {
	self.env.cmdc <- cmdFire
	rq := <-self.env.rqc
	return rq
}

func (self *vmAdapter) Accounts() (accounts []vm.Account, err error) {
	accounts = []vm.Account{}
	for _,v := range self.env.accounts {
		accounts = append(accounts,v)
	}
	return
}

func (self *vmAdapter) Out() ([]byte,error) {
	return self.env.output, self.env.err
}

func (self *vmAdapter) AvailableGas() (gas *big.Int, err error) {
	return
}

func (self *vmAdapter) RefundedGas() (gas *big.Int, err error) {
	return
}

func (self *vmAdapter) Logs() (logs state.Logs, err error) {
	return
}

func (self *vmAdapter) Removed() (addresses []common.Address, err error) {
	return
}

func (self *vmAdapter) Err() error {
	if self.env == nil {
		return vm.TerminatedError
	} else {
		return self.env.err
	}
}

func (self *vmAdapter) Name() string {
	return "CLASSIC VM"
}

func (self *vmAdapter) Type() vm.Type {
	return vm.ClassicVm
}

func (self *vmEnv) QueryRule(number uint64) *vmRule {
	for {
		if r, exists := self.rules[number]; exists {
			return r
		} else {
			self.handleOnError(&vm.Require{ID:vm.RequireRules,Number:number})
		}
	}
}

func (self *vmEnv) GasTable(block *big.Int) *vm.GasTable{
	number := block.Uint64()
	return self.QueryRule(number).table
}

func (self *vmEnv) IsHomestead(block *big.Int) bool{
	number := block.Uint64()
	return self.QueryRule(number).fork >= vm.Homestead
}

func (self *vmEnv) RuleSet() vm.RuleSet {
	return self
}

func (self *vmEnv) Origin() common.Address {
	return self.origin
}

func (self *vmEnv) BlockNumber() *big.Int {
	return big.NewInt(self.blocknumber)
}

func (self *vmEnv) Coinbase() common.Address {
	return self.coinbase
}

func (self *vmEnv) Time() *big.Int {
	return nil
}

func (self *vmEnv) Difficulty() *big.Int {
	return nil
}

func (self *vmEnv) GasLimit() *big.Int {
	return nil
}

func (self *vmEnv) Value() *big.Int {
	return nil
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
		if h, exists := self.hashes[n]; exists {
			return h.hash
		} else {
			self.handleOnError(&vm.Require{ID:vm.RequireHash,Number:n})
		}
	}
}

func (self *vmEnv) AddLog(log *state.Log) {

}

func (self *vmEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	return self.GetBalance(from).Cmp(balance) >= 0
}

func (self *vmEnv) SnapshotDatabase() int {
	return 0
	//return self.state.Snapshot()
}

func (self *vmEnv) RevertToSnapshot(snapshot int) {
	//self.state.RevertToSnapshot(snapshot)
}

func (self *vmEnv) Transfer(from, to state.AccountObject, amount *big.Int) {
	Transfer(from, to, amount)
}

func (self *vmEnv) Call(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return Call(self, me, addr, data, gas, price, value)
}

func (self *vmEnv) CallCode(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return CallCode(self, me, addr, data, gas, price, value)
}

func (self *vmEnv) DelegateCall(me ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return DelegateCall(self, me.(*Contract), addr, data, gas, price)
}

func (self *vmEnv) Create(me ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return Create(self, me, data, gas, price, value)
}

func (self *vmEnv) Run(contract *Contract, input []byte) (ret []byte, err error) {
	return self.evm.Run(contract,input)
}

func (self *vmEnv) GetAccount(addr common.Address) state.AccountObject {
	return self.QueryAccount(addr)
}

func (self *vmEnv) CreateAccount(common.Address) state.AccountObject {
	return nil
}

func (self *vmEnv) QueryCode(addr common.Address) *vmCode {
	for {
		if c, exists := self.codes[addr]; exists {
			return c
		} else {
			self.handleOnError(&vm.Require{ID:vm.RequireCode,Address:addr})
		}
	}
}

func (self *vmEnv) QueryAccount(addr common.Address) state.AccountObject {
	for {
		if a, exists := self.accounts[addr]; exists {
			return a
		} else {
			self.handleOnError(&vm.Require{ID:vm.RequireAccount,Address:addr})
		}
	}
}

func (self *vmEnv) AddBalance(addr common.Address, ammount *big.Int) {
	self.QueryAccount(addr).AddBalance(ammount)
}

func (self *vmEnv) GetBalance(addr common.Address) *big.Int {
	return self.QueryAccount(addr).Balance()
}

func (self *vmEnv) GetNonce(addr common.Address) uint64 {
	return self.QueryAccount(addr).Nonce()
}

func (self *vmEnv) SetNonce(addr common.Address, nonce uint64) {
	self.QueryAccount(addr).SetNonce(nonce)
}

func (self *vmEnv) GetCodeHash(addr common.Address) common.Hash {
	return self.QueryCode(addr).Hash()
}

func (self *vmEnv) GetCodeSize(addr common.Address) int {
	return self.QueryCode(addr).Size()
}

func (self *vmEnv) GetCode(addr common.Address) []byte {
	return self.QueryCode(addr).Code()
}

func (self *vmEnv) SetCode(common.Address, []byte) {
}

func (self *vmEnv) AddRefund(*big.Int) {
}

func (self *vmEnv) GetRefund() *big.Int {
	return nil
}

func (self *vmEnv) GetState(common.Address, common.Hash) common.Hash {
	panic("GetState is unimplemented")
	return common.Hash{}
}

func (self *vmEnv) SetState(common.Address, common.Hash, common.Hash) {
	panic("SetState is unimplemented")
}

func (self *vmEnv) Suicide(common.Address) bool {
	return false
}

func (self *vmEnv) HasSuicided(common.Address) bool {
	return false
}

func (self *vmEnv) Exist(common.Address) bool {
	return false
}
