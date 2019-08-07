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
	"fmt"
	"math/big"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/core/state"
	"github.com/eth-classic/go-ethereum/core/vm"
	"github.com/eth-classic/go-ethereum/crypto"
	"github.com/eth-classic/go-ethereum/params"
)

var (
	emptyCodeHash = crypto.Keccak256Hash(nil)

	callCreateDepthMax = 1024 // limit call/create stack
	errCallCreateDepth = fmt.Errorf("Max call depth exceeded (%d)", callCreateDepthMax)

	maxCodeSize            = 24576
	errMaxCodeSizeExceeded = fmt.Errorf("Max Code Size exceeded (%d)", maxCodeSize)

	errContractAddressCollision = errors.New("contract address collision")
)

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func Call(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice, value *big.Int) (ret []byte, err error) {
	evm := env.Vm()
	// Depth check execution. Fail if we're trying to execute above the limit.
	if env.Depth() > callCreateDepthMax {
		caller.ReturnGas(gas, gasPrice)

		return nil, errCallCreateDepth
	}

	if !env.CanTransfer(caller.Address(), value) {
		caller.ReturnGas(gas, gasPrice)

		return nil, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.Db().GetBalance(caller.Address()))
	}

	var (
		from       = env.Db().GetAccount(caller.Address())
		to         vm.Account
		snapshot   = env.SnapshotDatabase()
		isAtlantis = env.RuleSet().IsAtlantis(env.BlockNumber())
	)
	if !env.Db().Exist(addr) {
		precompiles := vm.PrecompiledPreAtlantis
		if isAtlantis {
			precompiles = vm.PrecompiledAtlantis
		}
		if precompiles[addr.Str()] == nil && isAtlantis && value.BitLen() == 0 {
			caller.ReturnGas(gas, gasPrice)
			return nil, nil
		}
		to = env.Db().CreateAccount(addr)
	} else {
		to = env.Db().GetAccount(addr)
	}
	env.Transfer(from, to, value)
	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := vm.NewContract(caller, to, value, gas, gasPrice)
	contract.SetCallCode(&addr, env.Db().GetCodeHash(addr), env.Db().GetCode(addr))
	defer contract.Finalise()

	// Even if the account has no code, we need to continue because it might be a precompile
	ret, err = evm.Run(contract, input, false)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		env.RevertToSnapshot(snapshot)
		if err != vm.ErrRevert {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, err
}

// CallCode executes the given address' code as the given contract address
func CallCode(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice, value *big.Int) (ret []byte, err error) {
	evm := env.Vm()
	// Depth check execution. Fail if we're trying to execute above the limit.
	if env.Depth() > callCreateDepthMax {
		caller.ReturnGas(gas, gasPrice)

		return nil, errCallCreateDepth
	}

	if !env.CanTransfer(caller.Address(), value) {
		caller.ReturnGas(gas, gasPrice)

		return nil, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.Db().GetBalance(caller.Address()))
	}

	var (
		to       = env.Db().GetAccount(caller.Address())
		snapshot = env.SnapshotDatabase()
	)
	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := vm.NewContract(caller, to, value, gas, gasPrice)
	contract.SetCallCode(&addr, env.Db().GetCodeHash(addr), env.Db().GetCode(addr))
	defer contract.Finalise()

	// Even if the account has no code, we need to continue because it might be a precompile
	ret, err = evm.Run(contract, input, false)

	if err != nil {
		env.RevertToSnapshot(snapshot)
		if err != vm.ErrRevert {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, err
}

// DelegateCall is equivalent to CallCode except that sender and value propagates from parent scope to child scope
func DelegateCall(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice *big.Int) (ret []byte, err error) {
	evm := env.Vm()
	// Depth check execution. Fail if we're trying to execute above the limit.
	if env.Depth() > callCreateDepthMax {
		caller.ReturnGas(gas, gasPrice)

		return nil, errCallCreateDepth
	}

	var (
		to       vm.Account
		snapshot = env.SnapshotDatabase()
	)
	if !env.Db().Exist(caller.Address()) {
		to = env.Db().CreateAccount(caller.Address())
	} else {
		to = env.Db().GetAccount(caller.Address())
	}

	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := vm.NewContract(caller, to, caller.Value(), gas, gasPrice).AsDelegate()
	contract.SetCallCode(&addr, env.Db().GetCodeHash(addr), env.Db().GetCode(addr))
	defer contract.Finalise()

	// Even if the account has no code, we need to continue because it might be a precompile
	ret, err = evm.Run(contract, input, false)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		env.RevertToSnapshot(snapshot)
		if err != vm.ErrRevert {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, err
}

// StaticCall executes within the given contract and throws exception if state is attempted to be changed
func StaticCall(env vm.Environment, caller vm.ContractRef, addr common.Address, input []byte, gas, gasPrice *big.Int) (ret []byte, err error) {
	evm := env.Vm()
	// Depth check execution. Fail if we're trying to execute above the limit.
	if env.Depth() > callCreateDepthMax {
		caller.ReturnGas(gas, gasPrice)

		return nil, errCallCreateDepth
	}

	var (
		to       vm.Account
		snapshot = env.SnapshotDatabase()
	)
	if !env.Db().Exist(addr) {
		to = env.Db().CreateAccount(addr)
	} else {
		to = env.Db().GetAccount(addr)
	}

	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := vm.NewContract(caller, to, new(big.Int), gas, gasPrice)
	contract.SetCallCode(&addr, env.Db().GetCodeHash(addr), env.Db().GetCode(addr))
	defer contract.Finalise()

	// We do an AddBalance of zero here, just in order to trigger a touch.
	// This is done to keep consensus with other clients since empty objects
	// get touched to be deleted even in a StaticCall context
	env.Db().AddBalance(addr, big.NewInt(0))

	// Even if the account has no code, we need to continue because it might be a precompile
	ret, err = evm.Run(contract, input, true)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		env.RevertToSnapshot(snapshot)
		if err != vm.ErrRevert {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, err
}

// Create creates a new contract with the given code
func Create(env vm.Environment, caller vm.ContractRef, code []byte, gas, gasPrice, value *big.Int) (ret []byte, address common.Address, err error) {
	ret, address, err = create(env, caller, crypto.Keccak256Hash(code), code, gas, gasPrice, value)
	// Here we get an error if we run into maximum stack depth,
	// See: https://github.com/ethereum/yellowpaper/pull/131
	// and YP definitions for CREATE

	//if there's an error we return nothing
	if err != nil && err != vm.ErrRevert {
		return nil, address, err
	}
	return ret, address, err
}

func create(env vm.Environment, caller vm.ContractRef, codeHash common.Hash, code []byte, gas, gasPrice, value *big.Int) (ret []byte, address common.Address, err error) {
	evm := env.Vm()
	// Depth check execution. Fail if we're trying to execute above the limit.
	if env.Depth() > callCreateDepthMax {
		caller.ReturnGas(gas, gasPrice)

		return nil, common.Address{}, errCallCreateDepth
	}

	if !env.CanTransfer(caller.Address(), value) {
		caller.ReturnGas(gas, gasPrice)

		return nil, common.Address{}, ValueTransferErr("insufficient funds to transfer value. Req %v, has %v", value, env.Db().GetBalance(caller.Address()))
	}

	// Create a new account on the state
	nonce := env.Db().GetNonce(caller.Address())
	env.Db().SetNonce(caller.Address(), nonce+1)
	address = crypto.CreateAddress(caller.Address(), nonce)

	// Ensure there's no existing contract already at the designated address
	contractHash := env.Db().GetCodeHash(address)
	if env.Db().GetNonce(address) != state.StartingNonce || (contractHash != (common.Hash{}) && contractHash != emptyCodeHash) {
		return nil, common.Address{}, errContractAddressCollision
	}

	var (
		snapshot = env.SnapshotDatabase()
		from     = env.Db().GetAccount(caller.Address())
		to       = env.Db().CreateAccount(address)
	)

	if env.RuleSet().IsAtlantis(env.BlockNumber()) {
		env.Db().SetNonce(address, state.StartingNonce+1)
	}

	env.Transfer(from, to, value)

	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := vm.NewContract(caller, to, value, gas, gasPrice)
	contract.SetCallCode(nil, codeHash, code)
	defer contract.Finalise()

	ret, err = evm.Run(contract, nil, false)

	maxCodeSizeExceeded := len(ret) > maxCodeSize && env.RuleSet().IsAtlantis(env.BlockNumber())
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		dataGas := big.NewInt(int64(len(ret)))
		// create data gas
		dataGas.Mul(dataGas, params.CreateDataGas)
		if contract.UseGas(dataGas) {
			env.Db().SetCode(address, ret)
		} else {
			err = vm.CodeStoreOutOfGasError
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && (env.RuleSet().IsHomestead(env.BlockNumber()) || err != vm.CodeStoreOutOfGasError)) {
		env.RevertToSnapshot(snapshot)
		if err != vm.ErrRevert {
			contract.UseGas(contract.Gas)
		}
	}

	// When there are no errors but the maxCodeSize is still exceeded, makes more sense than just failing
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}

	return ret, address, err
}

// generic transfer method
func Transfer(from, to vm.Account, amount *big.Int) {
	from.SubBalance(amount)
	to.AddBalance(amount)
}
