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
	"fmt"
	"math/big"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/core/vm"
	"github.com/eth-classic/go-ethereum/crypto"
)

var (
	callCreateDepthMax = 1024 // limit call/create stack
	errCallCreateDepth = fmt.Errorf("Max call depth exceeded (%d)", callCreateDepthMax)

	maxCodeSize            = 24576
	errMaxCodeSizeExceeded = fmt.Errorf("Max Code Size exceeded (%d)", maxCodeSize)
)

// Call executes within the given contract
func Call(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice, value *big.Int) (ret []byte, err error) {
	ret, _, err = exec(env, caller, &addr, &addr, env.Db().GetCodeHash(addr), input, env.Db().GetCode(addr), gas, gasPrice, value, false)
	return ret, err
}

// CallCode executes the given address' code as the given contract address
func CallCode(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice, value *big.Int) (ret []byte, err error) {
	callerAddr := caller.Address()
	ret, _, err = exec(env, caller, &callerAddr, &addr, env.Db().GetCodeHash(addr), input, env.Db().GetCode(addr), gas, gasPrice, value, false)
	return ret, err
}

// DelegateCall is equivalent to CallCode except that sender and value propagates from parent scope to child scope
func DelegateCall(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice *big.Int) (ret []byte, err error) {
	callerAddr := caller.Address()
	originAddr := env.Origin()
	callerValue := caller.Value()
	ret, _, err = execDelegateCall(env, caller, &originAddr, &callerAddr, &addr, env.Db().GetCodeHash(addr), input, env.Db().GetCode(addr), gas, gasPrice, callerValue)
	return ret, err
}

// StaticCall executes within the given contract and throws exception if state is attempted to be changed
func StaticCall(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice *big.Int) (ret []byte, err error) {
	ret, _, err = exec(env, caller, &addr, &addr, env.Db().GetCodeHash(addr), input, env.Db().GetCode(addr), gas, gasPrice, new(big.Int), true)
	return ret, err
}

// Create creates a new contract with the given code
func Create(env vm.Environment, caller vm.ContractRef, code []byte, gas, gasPrice, value *big.Int) (ret []byte, address common.Address, err error) {
	ret, address, err = exec(env, caller, nil, nil, crypto.Keccak256Hash(code), nil, code, gas, gasPrice, value, false)
	// Here we get an error if we run into maximum stack depth,
	// See: https://github.com/ethereum/yellowpaper/pull/131
	// and YP definitions for CREATE

	//if there's an error we return nothing
	if err != nil && err != vm.ErrRevert {
		return nil, address, err
	}
	return ret, address, err
}

func exec(env vm.Environment, caller vm.ContractRef, address, codeAddr *common.Address, codeHash common.Hash, input, code []byte, gas, gasPrice, value *big.Int, readOnly bool) (ret []byte, addr common.Address, err error) {
	evm := env.Vm()
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth() > callCreateDepthMax {
		caller.ReturnGas(gas, gasPrice)

		return nil, common.Address{}, errCallCreateDepth
	}

	if !env.CanTransfer(caller.Address(), value) {
		caller.ReturnGas(gas, gasPrice)

		return nil, common.Address{}, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.Db().GetBalance(caller.Address()))
	}

	var createAccount bool
	if address == nil {
		// Create a new account on the state
		nonce := env.Db().GetNonce(caller.Address())
		env.Db().SetNonce(caller.Address(), nonce+1)
		addr = crypto.CreateAddress(caller.Address(), nonce)
		address = &addr
		createAccount = true
	}

	snapshotPreTransfer := env.SnapshotDatabase()
	var (
		from = env.Db().GetAccount(caller.Address())
		to   vm.Account
	)

	if createAccount {
		to = env.Db().CreateAccount(*address)

		if env.RuleSet().IsAtlantis(env.BlockNumber()) {
			env.Db().SetNonce(*address, 1)
		}
	} else {
		if !env.Db().Exist(*address) {
			//no account may change state from non-existent to existent-but-empty. Refund sender.
			if vm.PrecompiledAtlantis[(*address).Str()] == nil && env.RuleSet().IsAtlantis(env.BlockNumber()) && value.BitLen() == 0 {
				caller.ReturnGas(gas, gasPrice)
				return nil, common.Address{}, nil
			}
			to = env.Db().CreateAccount(*address)
		} else {
			to = env.Db().GetAccount(*address)
		}
	}

	env.Transfer(from, to, value)

	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := vm.NewContract(caller, to, value, gas, gasPrice)
	contract.SetCallCode(codeAddr, codeHash, code)
	defer contract.Finalise()

	ret, err = evm.Run(contract, input, readOnly)

	maxCodeSizeExceeded := len(ret) > maxCodeSize && env.RuleSet().IsAtlantis(env.BlockNumber())
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && createAccount && !maxCodeSizeExceeded {
		dataGas := big.NewInt(int64(len(ret)))
		// create data gas
		dataGas.Mul(dataGas, big.NewInt(200))
		if contract.UseGas(dataGas) {
			env.Db().SetCode(*address, ret)
		} else {
			err = vm.CodeStoreOutOfGasError
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if createAccount && maxCodeSizeExceeded || (err != nil && (env.RuleSet().IsHomestead(env.BlockNumber()) || err != vm.CodeStoreOutOfGasError)) {
		env.RevertToSnapshot(snapshotPreTransfer)
		if err != vm.ErrRevert {
			contract.UseGas(contract.Gas)
		}
	}

	// When there are no errors but the maxCodeSize is still exceeded, makes more sense than just failing
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}

	return ret, addr, err
}

func execDelegateCall(env vm.Environment, caller vm.ContractRef, originAddr, toAddr, codeAddr *common.Address, codeHash common.Hash, input, code []byte, gas, gasPrice, value *big.Int) (ret []byte, addr common.Address, err error) {
	evm := env.Vm()
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if env.Depth() > callCreateDepthMax {
		caller.ReturnGas(gas, gasPrice)
		return nil, common.Address{}, errCallCreateDepth
	}

	snapshot := env.SnapshotDatabase()

	var to vm.Account
	if !env.Db().Exist(*toAddr) {
		to = env.Db().CreateAccount(*toAddr)
	} else {
		to = env.Db().GetAccount(*toAddr)
	}

	// Iinitialise a new contract and make initialise the delegate values
	contract := vm.NewContract(caller, to, value, gas, gasPrice).AsDelegate()
	contract.SetCallCode(codeAddr, codeHash, code)
	defer contract.Finalise()

	ret, err = evm.Run(contract, input, false)

	if err != nil {
		env.RevertToSnapshot(snapshot)
		if err != vm.ErrRevert {
			contract.UseGas(contract.Gas)
		}
	}

	return ret, addr, err
}

// generic transfer method
func Transfer(from, to vm.Account, amount *big.Int) {
	from.SubBalance(amount)
	to.AddBalance(amount)
}
