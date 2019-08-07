// Copyright 2014 The go-ethereum Authors
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

package core

import (
	"errors"
	"math/big"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/core/vm"
	"github.com/eth-classic/go-ethereum/logger"
	"github.com/eth-classic/go-ethereum/logger/glog"
)

var (
	TxGas                        = big.NewInt(21000) // Per transaction not creating a contract. NOTE: Not payable on data of calls between transactions.
	TxGasContractCreation        = big.NewInt(53000) // Per transaction that creates a contract. NOTE: Not payable on data of calls between transactions.
	TxDataZeroGas                = big.NewInt(4)     // Per byte of data attached to a transaction that equals zero. NOTE: Not payable on data of calls between transactions.
	TxDataNonZeroGas             = big.NewInt(68)    // Per byte of data attached to a transaction that is not equal to zero. NOTE: Not payable on data of calls between transactions.
	errInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
)

/*
The State Transitioning Model

A state transition is a change made when a transaction is applied to the current world state
The state transitioning model does all all the necessary work to work out a valid new state root.

1) Nonce handling
2) Pre pay gas
3) Create a new state object if the recipient is \0*32
4) Value transfer
== If contract creation ==
  4a) Attempt to run transaction data
  4b) If valid, use result as code for the new state object
== end ==
5) Run Script section
6) Derive new state root
*/
type StateTransition struct {
	gp            *GasPool
	msg           Message
	gas, gasPrice *big.Int
	initialGas    *big.Int
	value         *big.Int
	data          []byte
	state         vm.Database

	env vm.Environment
}

// Message represents a message sent to a contract.
type Message interface {
	From() (common.Address, error)
	To() *common.Address

	GasPrice() *big.Int
	Gas() *big.Int
	Value() *big.Int

	Nonce() uint64
	Data() []byte
}

func MessageCreatesContract(msg Message) bool {
	return msg.To() == nil
}

// IntrinsicGas computes the 'intrinsic gas' for a message
// with the given data.
func IntrinsicGas(data []byte, contractCreation, homestead bool) *big.Int {
	igas := new(big.Int)
	if contractCreation && homestead {
		igas.Set(TxGasContractCreation)
	} else {
		igas.Set(TxGas)
	}
	if len(data) > 0 {
		var nz int64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		m := big.NewInt(nz)
		m.Mul(m, TxDataNonZeroGas)
		igas.Add(igas, m)
		m.SetInt64(int64(len(data)) - nz)
		m.Mul(m, TxDataZeroGas)
		igas.Add(igas, m)
	}
	return igas
}

// NewStateTransition initialises and returns a new state transition object.
func NewStateTransition(env vm.Environment, msg Message, gp *GasPool) *StateTransition {
	return &StateTransition{
		gp:         gp,
		env:        env,
		msg:        msg,
		gas:        new(big.Int),
		gasPrice:   msg.GasPrice(),
		initialGas: new(big.Int),
		value:      msg.Value(),
		data:       msg.Data(),
		state:      env.Db(),
	}
}

// ApplyMessage computes the new state by applying the given message
// against the old state within the environment.
//
// ApplyMessage returns the bytes returned by any EVM execution (if it took place),
// the gas used (which includes gas refunds) and an error if it failed. An error always
// indicates a core error meaning that the message would always fail for that particular
// state and would never be accepted within a block.
func ApplyMessage(env vm.Environment, msg Message, gp *GasPool) ([]byte, *big.Int, bool, error) {
	st := NewStateTransition(env, msg, gp)

	ret, gasUsed, failed, err := st.TransitionDb()
	return ret, gasUsed, failed, err
}

// to returns the recipient of the message.
func (st *StateTransition) to() common.Address {
	if st.msg == nil || st.msg.To() == nil /* contract creation */ {
		return common.Address{}
	}
	return *st.msg.To()
}

func (st *StateTransition) useGas(amount *big.Int) error {
	if st.gas.Cmp(amount) < 0 {
		return vm.OutOfGasError
	}
	st.gas.Sub(st.gas, amount)

	return nil
}

func (st *StateTransition) addGas(amount *big.Int) {
	st.gas.Add(st.gas, amount)
}

func (st *StateTransition) buyGas() error {
	mgas := st.msg.Gas()
	mgval := new(big.Int).Mul(mgas, st.gasPrice)

	address, err := st.msg.From()
	if err != nil {
		return err
	}
	sender := st.state.GetAccount(address)

	if st.state.GetBalance(address).Cmp(mgval) < 0 {
		return errInsufficientBalanceForGas
	}

	if err = st.gp.SubGas(mgas); err != nil {
		return err
	}
	st.addGas(mgas)
	st.initialGas.Set(mgas)
	sender.SubBalance(mgval)
	return nil
}

func (st *StateTransition) preCheck() (err error) {
	msg := st.msg
	address, err := st.msg.From()
	if err != nil {
		return err
	}

	// Make sure this transaction's nonce is correct
	if n := st.state.GetNonce(address); n != msg.Nonce() {
		return NonceError(msg.Nonce(), n)
	}

	// Pre-pay gas
	if err = st.buyGas(); err != nil {
		if IsGasLimitErr(err) {
			return err
		}
		return InvalidTxError(err)
	}

	return nil
}

// TransitionDb will move the state by applying the message against the given environment.
func (st *StateTransition) TransitionDb() (ret []byte, gas *big.Int, failed bool, err error) {
	if err = st.preCheck(); err != nil {
		return
	}
	msg := st.msg
	address, err := st.msg.From()
	if err != nil {
		return nil, nil, false, err
	}
	var sender vm.Account
	if !st.state.Exist(address) {
		sender = st.state.CreateAccount(address)
	} else {
		sender = st.state.GetAccount(address)
	}
	homestead := st.env.RuleSet().IsHomestead(st.env.BlockNumber())
	contractCreation := MessageCreatesContract(msg)
	// Pay intrinsic gas
	if err = st.useGas(IntrinsicGas(st.data, contractCreation, homestead)); err != nil {
		return nil, nil, false, InvalidTxError(err)
	}

	vmenv := st.env
	//var addr common.Address
	var vmerr error
	if contractCreation {
		ret, _, vmerr = vmenv.Create(sender, st.data, st.gas, st.gasPrice, st.value)

		if vmerr == errContractAddressCollision {
			st.gas = big.NewInt(0)
		}
		if homestead && vmerr == vm.CodeStoreOutOfGasError {
			st.gas = big.NewInt(0)
		}

		if vmerr != nil {
			glog.V(logger.Core).Infoln("VM create err:", vmerr)
		}
	} else {
		// Increment the nonce for the next transaction
		st.state.SetNonce(address, st.state.GetNonce(sender.Address())+1)
		ret, vmerr = vmenv.Call(sender, st.to(), st.data, st.gas, st.gasPrice, st.value)
		if vmerr != nil {
			glog.V(logger.Core).Infoln("VM call err:", vmerr)
		}
	}

	if vmerr != nil && IsValueTransferErr(vmerr) {
		// if the vmerr was a value transfer error, return immediately
		// transaction receipt status will be set to TxSuccess
		return nil, nil, false, InvalidTxError(vmerr)
	}

	st.refundGas()
	st.state.AddBalance(st.env.Coinbase(), new(big.Int).Mul(st.gasUsed(), st.gasPrice))

	return ret, st.gasUsed(), vmerr != nil, err
}

func (st *StateTransition) refundGas() {
	// Return eth for remaining gas to the sender account,
	// exchanged at the original rate.
	address, err := st.msg.From()
	if err != nil {
		return
	}

	remaining := new(big.Int).Mul(st.gas, st.gasPrice)
	st.state.AddBalance(address, remaining)

	// Apply refund counter, capped to half of the used gas.
	uhalf := remaining.Div(st.gasUsed(), common.Big2)
	refund := common.BigMin(uhalf, st.state.GetRefund())
	st.gas.Add(st.gas, refund)
	st.state.AddBalance(address, refund.Mul(refund, st.gasPrice))
	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	st.gp.AddGas(st.gas)
}

func (st *StateTransition) gasUsed() *big.Int {
	return new(big.Int).Sub(st.initialGas, st.gas)
}
