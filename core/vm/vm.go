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

package vm

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/crypto"
	"github.com/eth-classic/go-ethereum/logger"
	"github.com/eth-classic/go-ethereum/logger/glog"
)

var (
	OutOfGasError          = errors.New("Out of gas")
	CodeStoreOutOfGasError = errors.New("Contract creation code storage out of gas")
	ErrRevert              = errors.New("Execution reverted")
)

// VirtualMachine is an EVM interface
type VirtualMachine interface {
	Run(*Contract, []byte) ([]byte, error)
}

// EVM is used to run Ethereum based contracts and will utilise the
// passed environment to query external sources for state information.
// The EVM will run the byte code VM or JIT VM based on the passed
// configuration.
type EVM struct {
	env       Environment
	jumpTable vmJumpTable
	gasTable  GasTable
	readOnly  bool
}

// New returns a new instance of the EVM.
func New(env Environment) *EVM {
	return &EVM{
		env:       env,
		jumpTable: newJumpTable(env.RuleSet(), env.BlockNumber()),
		gasTable:  *env.RuleSet().GasTable(env.BlockNumber()),
	}
}

// Run loops and evaluates the contract's code with the given input data
func (evm *EVM) Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error) {
	evm.env.SetDepth(evm.env.Depth() + 1)
	defer evm.env.SetDepth(evm.env.Depth() - 1)

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This makes also sure that the readOnly flag isn't removed for child calls.
	if readOnly && !evm.readOnly {
		evm.readOnly = true
		defer func() { evm.readOnly = false }()
	}

	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	evm.env.SetReturnData(nil)

	if contract.CodeAddr != nil {
		if evm.env.RuleSet().IsAtlantis(evm.env.BlockNumber()) {
			if p := PrecompiledAtlantis[contract.CodeAddr.Str()]; p != nil {
				return evm.RunPrecompiled(p, input, contract)
			}
		} else {
			if p := PrecompiledPreAtlantis[contract.CodeAddr.Str()]; p != nil {
				return evm.RunPrecompiled(p, input, contract)
			}
		}

	}

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	codehash := contract.CodeHash // codehash is used when doing jump dest caching
	if codehash == (common.Hash{}) {
		codehash = crypto.Keccak256Hash(contract.Code)
	}

	var (
		caller     = contract.caller
		instrCount = 0

		isAtlantis = evm.env.RuleSet().IsAtlantis(evm.env.BlockNumber())

		op      OpCode         // current opcode
		mem     = NewMemory()  // bound memory
		stack   = newstack()   // local stack
		statedb = evm.env.Db() // current state
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC to be uint256. Practically much less so feasible.
		pc = uint64(0) // program counter

		newMemSize *big.Int
		cost       *big.Int
	)
	contract.Input = input

	if glog.V(logger.Debug) {
		glog.Infof("running byte VM %x\n", codehash[:4])
		tstart := time.Now()
		defer func() {
			glog.Infof("byte VM %x done. time: %v instrc: %v\n", codehash[:4], time.Since(tstart), instrCount)
		}()
	}

	for ; ; instrCount++ {
		// Get the memory location of pc
		op = contract.GetOp(pc)
		operation := evm.jumpTable[op]
		// calculate the new memory size and gas price for the current executing opcode
		newMemSize, cost, err = calculateGasAndSize(&evm.gasTable, evm.env, contract, caller, op, statedb, mem, stack)
		if err != nil {
			return nil, err
		}

		// If the operation is valid, enforce and write restrictions
		if evm.readOnly && isAtlantis {
			// If the interpreter is operating in readonly mode, make sure no
			// state-modifying operation is performed. The 3rd stack item
			// for a call operation is the value. Transferring value from one
			// account to the others means the state is modified and should also
			// return with an error.
			if operation.writes || (op == CALL && stack.back(2).Sign() != 0) {
				return nil, errWriteProtection
			}
		}

		// Use the calculated gas. When insufficient gas is present, use all gas and return an
		// Out Of Gas error
		if !contract.UseGas(cost) {
			return nil, OutOfGasError
		}

		// Resize the memory calculated previously
		mem.Resize(newMemSize.Uint64())
		if !operation.valid {
			return nil, fmt.Errorf("Invalid opcode %x", op)
		}

		res, err := operation.fn(&pc, evm.env, contract, mem, stack)

		if operation.returns {
			evm.env.SetReturnData(res)
		}

		switch {
		case err != nil:
			return nil, err
		case operation.reverts:
			return res, ErrRevert
		case operation.halts:
			return res, nil
		case !operation.jumps:
			pc++
		}
	}
}

// calculateGasAndSize calculates the required given the opcode and stack items calculates the new memorysize for
// the operation. This does not reduce gas or resizes the memory.
func calculateGasAndSize(gasTable *GasTable, env Environment, contract *Contract, caller ContractRef, op OpCode, statedb Database, mem *Memory, stack *stack) (*big.Int, *big.Int, error) {
	var (
		gas                 = new(big.Int)
		newMemSize *big.Int = new(big.Int)
		isAtlantis          = env.RuleSet().IsAtlantis(env.BlockNumber())
	)
	err := baseCheck(op, stack, gas)
	if err != nil {
		return nil, nil, err
	}

	// stack Check, memory resize & gas phase
	switch op {
	case RETURNDATACOPY:
		newMemSize = calcMemSize(stack.back(0), stack.back(2))

		words := toWordSize(stack.back(2))
		gas.Add(gas, GasFastestStep)
		gas.Add(gas, words.Mul(words, big.NewInt(3)))

		quadMemGas(mem, newMemSize, gas)
	case REVERT:
		newMemSize = calcMemSize(stack.back(0), stack.back(1))
		quadMemGas(mem, newMemSize, gas)
	case SUICIDE:
		address := common.BigToAddress(stack.back(0))
		// if suicide is not nil: homestead gas fork
		if gasTable.CreateBySuicide != nil {
			gas.Set(gasTable.Suicide)
			if isAtlantis {
				if env.Db().Empty(address) && env.Db().GetBalance(contract.Address()).Sign() != 0 {
					gas.Add(gas, gasTable.CreateBySuicide)
				}
			} else if !env.Db().Exist(address) {
				gas.Add(gas, gasTable.CreateBySuicide)
			}
		}

		if !statedb.HasSuicided(contract.Address()) {
			statedb.AddRefund(big.NewInt(24000))
		}
	case EXTCODESIZE:
		gas.Set(gasTable.ExtcodeSize)
	case BALANCE:
		gas.Set(gasTable.Balance)
	case SLOAD:
		gas.Set(gasTable.SLoad)
	case SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16:
		n := int(op - SWAP1 + 2)
		err := stack.require(n)
		if err != nil {
			return nil, nil, err
		}
		gas.Set(GasFastestStep)
	case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
		n := int(op - DUP1 + 1)
		err := stack.require(n)
		if err != nil {
			return nil, nil, err
		}
		gas.Set(GasFastestStep)
	case LOG0, LOG1, LOG2, LOG3, LOG4:
		n := int(op - LOG0)
		err := stack.require(n + 2)
		if err != nil {
			return nil, nil, err
		}

		mSize, mStart := stack.back(1), stack.back(0)

		// log gas
		gas.Add(gas, big.NewInt(375))
		// log topic gass
		gas.Add(gas, new(big.Int).Mul(big.NewInt(int64(n)), big.NewInt(375)))
		// log data gass
		gas.Add(gas, new(big.Int).Mul(mSize, big.NewInt(8)))

		newMemSize = calcMemSize(mStart, mSize)

		quadMemGas(mem, newMemSize, gas)
	case EXP:
		expByteLen := int64(len(stack.back(1).Bytes()))
		gas.Add(gas, new(big.Int).Mul(big.NewInt(expByteLen), gasTable.ExpByte))
	case SSTORE:
		err := stack.require(2)
		if err != nil {
			return nil, nil, err
		}

		var g *big.Int
		y, x := stack.back(1), stack.back(0)
		val := statedb.GetState(contract.Address(), common.BigToHash(x))

		// This checks for 3 scenario's and calculates gas accordingly
		// 1. From a zero-value address to a non-zero value         (NEW VALUE)
		// 2. From a non-zero value address to a zero-value address (DELETE)
		// 3. From a non-zero to a non-zero                         (CHANGE)
		if common.EmptyHash(val) && !common.EmptyHash(common.BigToHash(y)) {
			// 0 => non 0
			g = big.NewInt(20000) // Once per SLOAD operation.
		} else if !common.EmptyHash(val) && common.EmptyHash(common.BigToHash(y)) {
			statedb.AddRefund(big.NewInt(15000))
			g = big.NewInt(5000)
		} else {
			// non 0 => non 0 (or 0 => 0)
			g = big.NewInt(5000)
		}
		gas.Set(g)

	case MLOAD:
		newMemSize = calcMemSize(stack.back(0), u256(32))
		quadMemGas(mem, newMemSize, gas)
	case MSTORE8:
		newMemSize = calcMemSize(stack.back(0), u256(1))
		quadMemGas(mem, newMemSize, gas)
	case MSTORE:
		newMemSize = calcMemSize(stack.back(0), u256(32))
		quadMemGas(mem, newMemSize, gas)
	case RETURN:
		newMemSize = calcMemSize(stack.back(0), stack.back(1))
		quadMemGas(mem, newMemSize, gas)
	case SHA3:
		newMemSize = calcMemSize(stack.back(0), stack.back(1))

		words := toWordSize(stack.back(1))
		gas.Add(gas, words.Mul(words, big.NewInt(6)))

		quadMemGas(mem, newMemSize, gas)
	case CALLDATACOPY:
		newMemSize = calcMemSize(stack.back(0), stack.back(2))

		words := toWordSize(stack.back(2))
		gas.Add(gas, words.Mul(words, big.NewInt(3)))

		quadMemGas(mem, newMemSize, gas)
	case CODECOPY:
		newMemSize = calcMemSize(stack.back(0), stack.back(2))

		words := toWordSize(stack.back(2))
		gas.Add(gas, words.Mul(words, big.NewInt(3)))

		quadMemGas(mem, newMemSize, gas)
	case EXTCODECOPY:
		gas.Set(gasTable.ExtcodeCopy)

		newMemSize = calcMemSize(stack.back(1), stack.back(3))

		words := toWordSize(stack.back(3))
		gas.Add(gas, words.Mul(words, big.NewInt(3)))

		quadMemGas(mem, newMemSize, gas)
	case CREATE:
		newMemSize = calcMemSize(stack.back(1), stack.back(2))

		quadMemGas(mem, newMemSize, gas)
	case CALL, CALLCODE:
		gas.Set(gasTable.Calls)

		if op == CALL {
			address := common.BigToAddress(stack.back(1))
			transfersValue := stack.back(2).Sign() != 0
			if isAtlantis {
				if transfersValue && env.Db().Empty(address) {
					gas.Add(gas, big.NewInt(25000))
				}
			} else if !env.Db().Exist(address) {
				gas.Add(gas, big.NewInt(25000))
			}
		}
		if len(stack.back(2).Bytes()) > 0 {
			gas.Add(gas, big.NewInt(9000))
		}
		x := calcMemSize(stack.back(5), stack.back(6))
		y := calcMemSize(stack.back(3), stack.back(4))

		newMemSize = common.BigMax(x, y)

		quadMemGas(mem, newMemSize, gas)

		cg := callGas(gasTable, contract.Gas, gas, stack.back(0))
		// Replace the stack item with the new gas calculation. This means that
		// either the original item is left on the stack or the item is replaced by:
		// (availableGas - gas) * 63 / 64
		// We replace the stack item so that it's available when the opCall instruction is
		// called. This information is otherwise lost due to the dependency on *current*
		// available gas.
		stack.data[stack.len()-1] = cg
		gas.Add(gas, cg)

	case DELEGATECALL:
		gas.Set(gasTable.Calls)

		x := calcMemSize(stack.back(4), stack.back(5))
		y := calcMemSize(stack.back(2), stack.back(3))

		newMemSize = common.BigMax(x, y)

		quadMemGas(mem, newMemSize, gas)

		cg := callGas(gasTable, contract.Gas, gas, stack.back(0))
		// Replace the stack item with the new gas calculation. This means that
		// either the original item is left on the stack or the item is replaced by:
		// (availableGas - gas) * 63 / 64
		// We replace the stack item so that it's available when the opCall instruction is
		// called.
		stack.data[stack.len()-1] = cg
		gas.Add(gas, cg)
	case STATICCALL:
		gas.Set(gasTable.Calls)

		x := calcMemSize(stack.back(4), stack.back(5))
		y := calcMemSize(stack.back(2), stack.back(3))

		newMemSize = common.BigMax(x, y)

		quadMemGas(mem, newMemSize, gas)

		cg := callGas(gasTable, contract.Gas, gas, stack.back(0))
		// Replace the stack item with the new gas calculation. This means that
		// either the original item is left on the stack or the item is replaced by:
		// (availableGas - gas) * 63 / 64
		// We replace the stack item so that it's available when the opCall instruction is
		// called.
		stack.data[stack.len()-1] = cg
		gas.Add(gas, cg)
	}

	return newMemSize, gas, nil
}

// RunPrecompile runs and evaluate the output of a precompiled contract defined in contracts.go
func (evm *EVM) RunPrecompiled(p *PrecompiledAccount, input []byte, contract *Contract) (ret []byte, err error) {
	gas := p.Gas(input)
	if contract.UseGas(gas) {
		return p.Call(input)
	} else {
		return nil, OutOfGasError
	}
}
