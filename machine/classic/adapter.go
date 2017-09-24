package classic

import (
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/common"
	"math/big"
	"errors"
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
	balance *big.Int
	code    []byte
	hash    common.Hash
	nonce   uint64
	storage map[string]string
	value   *big.Int
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
func (self *vmAccount) Value() *big.Int { return self.value }
func (self *vmAccount) Set(vm.Account) {}

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

type vmEnv struct {
	cmdc    	chan *command
	errc    	chan error
	rules   	vm.RuleSet
	evm     	*EVM
	depth   	int
	contract 	*Contract
	input   	[]byte
	output      []byte
	stepByStep	bool

	accounts 	map[common.Address]*vmAccount
	codes 		map[common.Address]*vmCode
	hashes 		map[uint64]*vmHash
	origin 		common.Address
	coinbase 	common.Address
	blocknumber int64

	err 		error
}

type vmAdapter struct {
	env	*vmEnv
	status vm.Status
}

func AdaptVM() vm.VirtualMachine {
	adapter := &vmAdapter{}
	return adapter
}

func (self *vmAdapter) CommitAccount(commitment vm.Account) (err error) {
	account := vmAccount{}
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

func (self *vmAdapter) Status() (status vm.Status, err error) {
	status = self.status
	return
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

func (self *vmEnv) handleOnError(err error) {
	self.errc <- err
	for {
		cmd := <-self.cmdc
		self.handleCmdc(cmd)
	}
}

func (self *vmAdapter) Start() (err error) {
	if self.env == nil {
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
				e.output, e.err = e.evm.Run(e.contract,e.input)
				e.errc <- e.err
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


func (self *vmAdapter) Step() (err error) {
	self.env.cmdc <-cmdStep
	err = <-self.env.errc
	return
}

func (self *vmAdapter) Fire() (err error) {
	self.env.cmdc <- cmdFire
	err = <-self.env.errc
	return
}

func (self *vmAdapter) Accounts() (accounts map[common.Address]vm.Account, err error) {
	accounts = map[common.Address]vm.Account{}
	for k,v := range self.env.accounts {
		accounts[k] = v
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

func (self *vmAdapter) Logs() (logs []vm.Log, err error) {
	return
}

func (self *vmAdapter) Removed() (addresses []common.Address, err error) {
	return
}

func (self *vmEnv) VmType() vm.Type {
	return vm.ClassicVmTy
}

func (self *vmEnv) RuleSet() vm.RuleSet {
	return nil
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

func (self *vmEnv) Db() vm.Database {
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
			self.handleOnError(&vm.RequireHashError{n})
		}
	}
}

func (self *vmEnv) AddLog(log *vm.Log) {

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

func (self *vmEnv) Transfer(from, to vm.Account, amount *big.Int) {
	Transfer(from, to, amount)
}

func (self *vmEnv) Call(me vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return Call(self, me, addr, data, gas, price, value)
}

func (self *vmEnv) CallCode(me vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return CallCode(self, me, addr, data, gas, price, value)
}

func (self *vmEnv) DelegateCall(me vm.ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return DelegateCall(self, me, addr, data, gas, price)
}

func (self *vmEnv) Create(me vm.ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return Create(self, me, data, gas, price, value)
}

func (self *vmEnv) Run(contract *Contract, input []byte) (ret []byte, err error) {
	return self.evm.Run(contract,input)
}

func (self *vmEnv) GetAccount(addr common.Address) vm.Account {
	return self.QueryAccount(addr)
}

func (self *vmEnv) CreateAccount(common.Address) vm.Account {
	return nil
}

func (self *vmEnv) QueryCode(addr common.Address) *vmCode {
	for {
		if c, exists := self.codes[addr]; exists {
			return c
		} else {
			self.handleOnError(&vm.RequireCodeError{addr})
		}
	}
}

func (self *vmEnv) QueryAccount(addr common.Address) vm.Account {
	for {
		if a, exists := self.accounts[addr]; exists {
			return a
		} else {
			self.handleOnError(&vm.RequireAccountError{addr})
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
