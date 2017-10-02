package classic

import (
	"fmt"
	"math/big"

	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/ethdb"
)

type command struct {
	code uint8
	value interface{}
}

type account struct {
	address common.Address
	nonce   uint64
	balance *big.Int
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
	onModify func()
	acc state.AccountObject
}

func (self *vmAccount) SubBalance(amount *big.Int) {
	self.acc.SubBalance(amount)
	self.onModify()
}

func (self *vmAccount) AddBalance(amount *big.Int) {
	self.acc.AddBalance(amount)
	self.onModify()
}

func (self *vmAccount) SetBalance(amount *big.Int) {
	self.acc.SetBalance(amount)
	self.onModify()
}

func (self *vmAccount) SetNonce(nonce uint64) {
	self.acc.SetNonce(nonce)
	self.onModify()
}

func (self *vmAccount) ReturnGas(gas, price *big.Int) {
	self.acc.ReturnGas(gas, price)
	self.onModify()
}

func (self *vmAccount) SetCode(hash common.Hash, code []byte) {
	self.acc.SetCode(hash,code)
	self.onModify()
}

func (self *vmAccount) Nonce() uint64 { return self.acc.Nonce() }
func (self *vmAccount) Balance() *big.Int { return self.acc.Balance() }
func (self *vmAccount) Address() common.Address { return self.acc.Address() }
func (self *vmAccount) ForEachStorage(cb func(key, value common.Hash) bool) { panic("unimplemented")}

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
	table 		*vm.GasTable
	fork  		vm.Fork
	difficulty 	*big.Int
	gasLimit 	*big.Int
	time		*big.Int
}

type machine struct {}

type context struct {
	env    vmEnv
}

type ChangeLevel byte
const (
	None ChangeLevel = iota
	Absent
	Committed
	Modified
	Removed
)

type vmEnv struct {
	cmdc    	chan *command
	rqc			chan *vm.Require
	rules   	*vmRule
	evm     	*EVM
	depth   	int
	contract 	*Contract
	output      []byte
	stepByStep	bool

	db			*state.StateDB
	origin 		common.Address
	coinbase 	common.Address
	address     common.Address
	account 	map[common.Address]ChangeLevel
	code		map[common.Address]ChangeLevel
	hash		map[uint64]*vmHash
	blocknumber uint64

	status      vm.Status
	err 		error
}

func NewMachine() vm.Machine {
	return &machine{}
}

func mkEnv(env *vmEnv, number uint64) *vmEnv {
	env.rqc = make(chan *vm.Require)
	env.cmdc = make(chan *command)
	env.blocknumber = number
	db, _ := ethdb.NewMemDatabase()
	env.db, _ = state.New(common.Hash{}, db)
	env.account = make(map[common.Address]ChangeLevel)
	env.code = make(map[common.Address]ChangeLevel)
	env.hash = make(map[uint64]*vmHash)
	env.status = vm.Inactive
	return env
}

func (self *machine) Name() string {
	return "CLASSIC VM"
}

func (self *machine) Type() vm.Type {
	return vm.ClassicVm
}

func (self *machine) Call(blockNumber uint64, caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (vm.Context, error) {
	ctx := &context{}

	go func (e *vmEnv) {
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
			e.output, e.err = e.Call(callerRef,to,data,gas,price,value)
			e.rqc <- nil
			close(e.rqc)
			return
		} else {
			e.handleCmdc(cmd)
		}
	}(mkEnv(&ctx.env,blockNumber))

	return ctx, nil
}

func (self *machine) Create(blockNumber uint64, caller common.Address, code []byte, gas, price, value *big.Int) (vm.Context, error) {
	ctx := &context{}

	go func (e *vmEnv) {
		cmd, ok := <-e.cmdc
		if !ok {
			return
		}
		if cmd.code == cmdStepID || cmd.code == cmdFireID {
			e.status = vm.Running
			e.stepByStep = cmd.code == cmdStepID
			callerRef := e.queryAccount(caller)
			e.evm = NewVM(e)
			e.output, e.address, e.err = e.Create(callerRef,code,gas,price,value)
			e.rqc <- nil
			return
		} else {
			e.handleCmdc(cmd)
		}
	}(mkEnv(&ctx.env,blockNumber))

	return ctx, nil
}

func (self *vmEnv) handleCmdc(cmd *command) bool {
	fmt.Println("handleCmd %v",*cmd)
	switch cmd.code {
	case cmdHashID:
		h := cmd.value.(*vmHash)
		self.hash[h.number] = h
	case cmdCodeID:
		c := cmd.value.(*vmCode)
		fmt.Println("handleCmd %v",*c)
		if c.code == nil {
			self.code[c.address] = Absent
		} else {
			var acc state.AccountObject
			if !self.db.Exist(c.address) {
				acc = self.db.CreateAccount(c.address)
			} else {
				acc = self.db.GetAccount(c.address)
			}
			acc.SetCode(c.hash, c.code)
			self.code[c.address] = Committed
		}
	case cmdAccountID:
		a := cmd.value.(*account)
		if a.balance == nil {
			self.account[a.address] = Absent
		} else {
			var acc state.AccountObject
			if !self.db.Exist(a.address) {
				acc = self.db.CreateAccount(a.address)
			} else {
				acc = self.db.GetAccount(a.address)
			}
			acc.SetNonce(a.nonce)
			acc.SetBalance(a.balance)
			self.account[a.address] = Committed
		}
	case cmdRuleID:
		self.rules = cmd.value.(*vmRule)
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
	fmt.Println("handleRequire %v",*rq)
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

func (self *context) Address() (common.Address, error) {
	return self.env.address, nil
}

func (self *context) CommitRules(table *vm.GasTable, fork vm.Fork, difficulty, gasLimit, time *big.Int) (err error) {
	rule := vmRule{table,fork, difficulty, gasLimit, time}
	self.env.cmdc <- &command{cmdRuleID, &rule}
	return
}

func (self *context) CommitAccount(address common.Address, nonce uint64, balance *big.Int) (err error) {
	account := account{address:address,nonce:nonce,balance:balance}
	self.env.cmdc <- &command{cmdAccountID, &account}
	return
}

func (self *context) CommitBlockhash(number uint64, hash common.Hash) (err error) {
	value := vmHash{number, hash}
	self.env.cmdc <- &command{cmdHashID, &value}
	return
}

func (self *context) CommitCode(address common.Address, hash common.Hash, code []byte) (err error) {
	value := vmCode{address, code, len(code), hash}
	self.env.cmdc <- &command{cmdCodeID, &value}
	return
}

func (self *context) Status() vm.Status {
	return self.env.status
}

func (self *context) Finish() (err error) {
	return nil
}

func (self *context) Fire() (*vm.Require) {
	self.env.cmdc <- cmdFire
	rq := <-self.env.rqc
	return rq
}

func (self *context) Code(addr common.Address) (common.Hash, []byte, error) {
	if self.env.account[addr] == Modified {
		return self.env.db.GetCodeHash(addr), self.env.db.GetCode(addr), nil
	} else {
		return common.Hash{}, nil, nil
	}
}

func (self *context) Modified() (accounts []vm.Account, err error) {
	accounts = []vm.Account{}
	for addr, level := range self.env.account {
		if level == Modified {
			acc := self.env.db.GetAccount(addr)
			accounts = append(accounts, acc.(vm.Account))
		}
	}
	return
}

func (self *context) Removed() (addresses []common.Address, err error) {
	addresses = []common.Address{}
	for addr, level := range self.env.account {
		if level == Removed {
			addresses = append(addresses, addr)
		}
	}
	return
}

func (self *context) Committed() (addresses []common.Address, err error) {
	addresses = []common.Address{}
	for addr, level := range self.env.account {
		if level == Committed {
			addresses = append(addresses, addr)
		}
	}
	return
}

func (self *context) Out() ([]byte,error) {
	return self.env.output, self.env.err
}

func (self *context) Logs() (logs state.Logs, err error) {
	logs = self.env.db.Logs()
	return
}

func (self *context) Err() error {
	return self.env.err
}

func (self *vmEnv) queryRule() *vmRule {
	for {
		if self.rules != nil {
			return self.rules
		} else {
			self.handleRequire(&vm.Require{ID:vm.RequireRules,Number:self.blocknumber})
		}
	}
}

func (self *vmEnv) GasTable(block *big.Int) *vm.GasTable{
	number := block.Uint64()
	if number != self.blocknumber {
		self.status = vm.Broken
		panic("invalid block number")
	}
	return self.queryRule().table
}

func (self *vmEnv) IsHomestead(block *big.Int) bool{
	number := block.Uint64()
	if number != self.blocknumber {
		self.status = vm.Broken
		panic("invalid block number")
	}
	return self.queryRule().fork >= vm.Homestead
}

func (self *vmEnv) RuleSet() vm.RuleSet {
	return self
}

func (self *vmEnv) Origin() common.Address {
	return self.origin
}

func (self *vmEnv) BlockNumber() *big.Int {
	return new(big.Int).SetUint64(self.blocknumber)
}

func (self *vmEnv) Coinbase() common.Address {
	return self.coinbase
}

func (self *vmEnv) Time() *big.Int {
	return self.queryRule().time
}

func (self *vmEnv) Difficulty() *big.Int {
	return self.queryRule().difficulty
}

func (self *vmEnv) GasLimit() *big.Int {
	return self.queryRule().gasLimit
}

func (self *vmEnv) Db() Database {
	// wrap database activity
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
			self.handleRequire(&vm.Require{ID:vm.RequireHash,Number:n})
		}
	}
}

func (self *vmEnv) AddLog(log *state.Log) {
	self.db.AddLog(log)
}

func (self *vmEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	return self.GetBalance(from).Cmp(balance) >= 0
}

func (self *vmEnv) SnapshotDatabase() int {
	return self.db.Snapshot()
}

func (self *vmEnv) RevertToSnapshot(snapshot int) {
	self.db.RevertToSnapshot(snapshot)
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
	return self.queryAccount(addr)
}

func (self *vmEnv) CreateAccount(addr common.Address) state.AccountObject {
	self.account[addr] = Modified
	return self.db.CreateAccount(addr)
}

func (self *vmEnv) queryCode(addr common.Address) {
	for {
		if self.code[addr] == None {
			self.handleRequire(&vm.Require{ID:vm.RequireCode,Address:addr})
		}
	}
}

func (self *vmEnv) queryAccount(addr common.Address) state.AccountObject {
	for {
		switch self.account[addr] {
		case None:
			self.handleRequire(&vm.Require{ID:vm.RequireAccount,Address:addr})
		case Committed, Modified:
			return &vmAccount{ func(){ self.account[addr] = Modified }, self.db.GetAccount(addr) }
		default:
			return nil
		}
	}
}

func (self *vmEnv) AddBalance(addr common.Address, ammount *big.Int) {
	self.queryAccount(addr).AddBalance(ammount)
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
	self.queryCode(addr)
	return self.db.GetCodeHash(addr)
}

func (self *vmEnv) GetCodeSize(addr common.Address) int {
	self.queryCode(addr)
	return self.db.GetCodeSize(addr)
}

func (self *vmEnv) GetCode(addr common.Address) []byte {
	self.queryCode(addr)
	return self.db.GetCode(addr)
}

func (self *vmEnv) SetCode(addr common.Address, code []byte) {
	self.code[addr] = Modified
	self.db.SetCode(addr,code)
}

func (self *vmEnv) AddRefund(gas *big.Int) {
	self.db.AddRefund(gas)
}

func (self *vmEnv) GetRefund() *big.Int {
	return self.db.GetRefund()
}

func (self *vmEnv) GetState(common.Address, common.Hash) common.Hash {
	panic("GetState is unimplemented")
	return common.Hash{}
}

func (self *vmEnv) SetState(common.Address, common.Hash, common.Hash) {
	panic("SetState is unimplemented")
}

func (self *vmEnv) Suicide(addr common.Address) bool {
	self.account[addr] = Removed
	return self.db.Suicide(addr)
}

func (self *vmEnv) HasSuicided(addr common.Address) bool {
	return self.db.HasSuicided(addr)
}

func (self *vmEnv) Exist(addr common.Address) bool {
	self.queryAccount(addr)
	return self.db.Exist(addr)
}
