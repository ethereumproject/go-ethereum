package classic

import (
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/common"
	"math/big"
	"errors"
)

type command struct {
	code uint8
}

const (
	cmdStepCode = iota
	cmdFireCode
)

var (
	cmdStep = &command{cmdStepCode}
	cmdFire = &command{cmdFireCode}
)

type vmEnv struct {
	cmdc  chan *command
	errc  chan error
	rules vm.RuleSet
	//state     *state.StateDB // State to use for executing
	evm     *EVM           // The Ethereum Virtual Machine
	depth   int            // Current execution depth
	//header    *types.Header  // Header information
	//getHashFn func(uint64) common.Hash // getHashFn callback is used to retrieve block hashes
	//from      common.Address
	//value     *big.Int
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
	return
}

func (self *vmAdapter) CommitBlockhash(number *big.Int, hash *big.Int) (err error) {
	return
}

func (self *vmAdapter) Status() (status vm.Status, err error) {
	status = self.status
	return
}

func (self *vmAdapter) Launch() (err error) {
	if self.env == nil {
		env := &vmEnv{}
		env.evm = NewVM(env)
		self.env = env
		go func (e *vmEnv) {
			for {
				cmd, ok := <-e.cmdc
				if !ok {
					return
				}
				switch cmd.(type) {

				}
			}
		}(env)
	} else {
		err = errors.New("already started")
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
	return
}

func (self *vmAdapter) Out() (result []byte, err error) {
	return
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
	return nil
}

func (self *vmEnv) BlockNumber() *big.Int {
	return nil
}

func (self *vmEnv) Coinbase() common.Address {
	return nil
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
	return nil
}

func (self *vmEnv) Depth() int {
	return self.depth
}

func (self *vmEnv) SetDepth(i int) {
	self.depth = i
}

func (self *vmEnv) GetHash(n uint64) common.Hash {
	return nil
}

func (self *vmEnv) AddLog(log *vm.Log) {

}

func (self *vmEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	//return self.state.GetBalance(from).Cmp(balance) >= 0
}

func (self *vmEnv) SnapshotDatabase() int {
	//return self.state.Snapshot()
}

func (self *vmEnv) RevertToSnapshot(snapshot int) {
	//self.state.RevertToSnapshot(snapshot)
}

func (self *vmEnv) Transfer(from, to vm.Account, amount *big.Int) {
	//Transfer(from, to, amount)
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
