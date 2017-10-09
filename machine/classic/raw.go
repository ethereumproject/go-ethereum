package classic

import (
	"math/big"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/crypto"
	"fmt"
)

type RawContext struct {
	RuleSet_     vm.RuleSet
	State_       *state.StateDB
	Gas          *big.Int
	Origin_   	 common.Address
	Coinbase_ 	 common.Address
	Number_      *big.Int
	Time_        *big.Int
	Difficulty_  *big.Int
	GasLimit_    *big.Int
	HashFn_		 func(n uint64) common.Hash
	output		 []byte
	address		 common.Address
	skipTransfer bool
	initial      bool
	vmTest 		 bool
	depth        int
	err          error
	To_			 *common.Address
	Price_		 *big.Int
	Value_ 		 *big.Int
	Input_		 []byte
	evm 		 *EVM
}

type RawMachine struct{
	RawContext
}

func NewRawMachine() vm.Machine {
	return &RawMachine{}
}

func (self *RawMachine) _SetTestFeatures(int) error {
	return nil
}

func (self *RawMachine) Call(caller common.Address, to common.Address, data []byte, gas, price, value *big.Int) (vm.Context, error) {
	self.Gas = new(big.Int).Set(gas)
	self.Price_ = price
	self.Value_ = value
	self.Origin_ = caller
	self.To_ = &to
	self.Input_ = data
	self.evm = NewVM(&self.RawContext)
	fmt.Printf("call %s => %s\n",caller.Hex(),to.Hex())
	return &self.RawContext, nil
}

func (self *RawMachine) Create(caller common.Address, code []byte, gas, price, value *big.Int) (vm.Context, error) {
	self.Gas = new(big.Int).Set(gas)
	self.Price_ = price
	self.Value_ = value
	self.Origin_ = caller
	self.Input_ = code
	self.evm = NewVM(&self.RawContext)
	fmt.Printf("create %s => ?\n",caller.Hex())
	return &self.RawContext, nil
}

func (*RawMachine) Type() vm.Type { return vm.ClassicRawVm }
func (*RawMachine) Name() string {return "RAW CLASSIC VM" }

func (*RawContext) CommitAccount(address common.Address, nonce uint64, balance *big.Int) error {
	return nil
}
func (*RawContext) CommitBlockhash(number uint64, hash common.Hash) error {
	return nil
}
func (*RawContext) CommitCode(address common.Address, hash common.Hash, code []byte) error {
	return nil
}
func (*RawContext) CommitInfo(blockNumber uint64, coinbase common.Address, table *vm.GasTable, fork vm.Fork, difficulty, gasLimit, time *big.Int) error {
	return nil
}
func (*RawContext) CommitValue(address common.Address, key common.Hash, value common.Hash) error {
	return nil
}
func (*RawContext) Finish() error {
	return nil
}
func (*RawContext) Accounts() ([]vm.Account, error) {
	return []vm.Account{}, nil
}
func (*RawContext) Logs() (state.Logs, error) {
	return state.Logs{}, nil
}
func (self *RawContext) Address() (common.Address, error) {
	return self.address, nil
}
func (self *RawContext) Out() ([]byte, *big.Int, *big.Int, error) {
	return self.output, self.Gas, new(big.Int), nil
}
func (self *RawContext) Status() vm.Status {
	fmt.Printf("STATUS: err => %v\n",self.err)
	if self.err == nil {
		return vm.ExitedOk
	} else if self.err == OutOfGasError {
		return vm.OutOfGasErr
	} else if IsValueTransferErr(self.err) {
		return vm.TransferErr
	} else {
		return vm.ExitedErr
	}
}
func (self *RawContext) Err() error {
	return self.err
}
// Run instructions until it reaches a `RequireErr` or exits
func (self *RawContext) Fire() *vm.Require {
	if self.To_ != nil {
		caller := self.State_.GetAccount(self.Origin_)
		self.output, self.err = self.Call(
			caller,*self.To_,self.Input_,self.Gas,self.Price_,self.Value_)
	} else {
		caller := self.State_.GetAccount(self.Origin_)
		self.output, self.address, self.err = self.Create(
			caller,self.Input_,self.Gas,self.Price_,self.Value_)
	}
	return nil
}

func (self *RawContext) GetHash(n uint64) common.Hash { return self.HashFn_(n) }

func (self *RawContext) RuleSet() vm.RuleSet      { return self.RuleSet_ }
func (self *RawContext) Origin() common.Address   { return self.Origin_ }
func (self *RawContext) BlockNumber() *big.Int    { return self.Number_ }
func (self *RawContext) Coinbase() common.Address { return self.Coinbase_ }
func (self *RawContext) Time() *big.Int           { return self.Time_ }
func (self *RawContext) Difficulty() *big.Int     { return self.Difficulty_ }
func (self *RawContext) Db() Database     		  { return self.State_ }
func (self *RawContext) GasLimit() *big.Int       { return self.GasLimit_ }
func (self *RawContext) VmType() vm.Type          { return vm.ClassicRawVm }
func (self *RawContext) AddLog(log *state.Log) 	  { self.State_.AddLog(log) }
func (self *RawContext) Depth() int               { return self.depth }
func (self *RawContext) SetDepth(i int) 		  { self.depth = i }

func (self *RawContext) CanTransfer(from common.Address, balance *big.Int) bool {
	if self.skipTransfer {
		if self.initial {
			self.initial = false
			return true
		}
	}

	return self.State_.GetBalance(from).Cmp(balance) >= 0
}

func (self *RawContext) SnapshotDatabase() int {
	return self.State_.Snapshot()
}

func (self *RawContext) RevertToSnapshot(snapshot int) {
	self.State_.RevertToSnapshot(snapshot)
}

func (self *RawContext) Transfer(from, to state.AccountObject, amount *big.Int) {
	if self.skipTransfer {
		return
	}
	Transfer(from, to, amount)
}

func (self *RawContext) Call(caller ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	if self.vmTest && self.depth > 0 {
		caller.ReturnGas(gas, price)
		return nil, nil
	}
	ret, err := Call(self, caller, addr, data, gas, price, value)
	return ret, err

}

func (self *RawContext) CallCode(caller ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	if self.vmTest && self.depth > 0 {
		caller.ReturnGas(gas, price)
		return nil, nil
	}
	return CallCode(self, caller, addr, data, gas, price, value)
}

func (self *RawContext) DelegateCall(caller ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	if self.vmTest && self.depth > 0 {
		caller.ReturnGas(gas, price)
		return nil, nil
	}
	return DelegateCall(self, caller.(*Contract), addr, data, gas, price)
}

func (self *RawContext) Create(caller ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	if self.vmTest {
		caller.ReturnGas(gas, price)

		nonce := self.State_.GetNonce(caller.Address())
		obj := self.State_.GetOrNewStateObject(crypto.CreateAddress(caller.Address(), nonce))

		return nil, obj.Address(), nil
	} else {
		return Create(self, caller, data, gas, price, value)
	}
}

func (self *RawContext) Run(contract *Contract, input []byte) (ret []byte, err error) {
	return self.evm.Run(contract,input)
}
