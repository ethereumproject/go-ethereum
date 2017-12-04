// Copyright 2015 The go-ethereum Authors
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
	"fmt"
	"math/big"

	"github.com/ethereumproject/go-ethereum/core/vm"
)

const stackLimit = 1024 // maximum size of VM stack allowed.

var (
	GasQuickStep   = big.NewInt(2)
	GasFastestStep = big.NewInt(3)
	GasFastStep    = big.NewInt(5)
	GasMidStep     = big.NewInt(8)
	GasSlowStep    = big.NewInt(10)
	GasExtStep     = big.NewInt(20)

	GasReturn = big.NewInt(0)
	GasStop   = big.NewInt(0)

	GasContractByte = big.NewInt(200)

	n64 = big.NewInt(64)
)

// calcGas returns the actual gas cost of the call.
//
// The cost of gas was changed during the homestead price change HF. To allow for EIP150
// to be implemented. The returned gas is gas - base * 63 / 64.
func callGas(gasTable *vm.GasTable, availableGas, base, callCost *big.Int) *big.Int {
	if gasTable.CreateBySuicide != nil {
		availableGas = new(big.Int).Sub(availableGas, base)
		g := new(big.Int).Div(availableGas, n64)
		g.Sub(availableGas, g)

		if g.Cmp(callCost) < 0 {
			return g
		}
	}
	return callCost
}

// baseCheck checks for any stack error underflows
func baseCheck(op vm.OpCode, stack *stack, gas *big.Int) error {
	// PUSH and DUP are a bit special. They all cost the same but we do want to have checking on stack push limit
	// PUSH is also allowed to calculate the same price for all PUSHes
	// DUP requirements are handled elsewhere (except for the stack limit check)
	if op >= vm.PUSH1 && op <= vm.PUSH32 {
		op = vm.PUSH1
	}
	if op >= vm.DUP1 && op <= vm.DUP16 {
		op = vm.DUP1
	}

	if r, ok := _baseCheck[op]; ok {
		err := stack.require(r.stackPop)
		if err != nil {
			return err
		}

		if r.stackPush > 0 && stack.len()-r.stackPop+r.stackPush > stackLimit {
			return fmt.Errorf("stack length %d exceed limit %d", stack.len(), stackLimit)
		}

		gas.Add(gas, r.gas)
	}
	return nil
}

// casts a arbitrary number to the amount of words (sets of 32 bytes)
func toWordSize(size *big.Int) *big.Int {
	tmp := new(big.Int)
	tmp.Add(size, u256(31))
	tmp.Div(tmp, u256(32))
	return tmp
}

type req struct {
	stackPop  int
	gas       *big.Int
	stackPush int
}

var _baseCheck = map[vm.OpCode]req{
	// opcode  |  stack pop | gas price | stack push
	vm.ADD:          {2, GasFastestStep, 1},
	vm.LT:           {2, GasFastestStep, 1},
	vm.GT:           {2, GasFastestStep, 1},
	vm.SLT:          {2, GasFastestStep, 1},
	vm.SGT:          {2, GasFastestStep, 1},
	vm.EQ:           {2, GasFastestStep, 1},
	vm.ISZERO:       {1, GasFastestStep, 1},
	vm.SUB:          {2, GasFastestStep, 1},
	vm.AND:          {2, GasFastestStep, 1},
	vm.OR:           {2, GasFastestStep, 1},
	vm.XOR:          {2, GasFastestStep, 1},
	vm.NOT:          {1, GasFastestStep, 1},
	vm.BYTE:         {2, GasFastestStep, 1},
	vm.CALLDATALOAD: {1, GasFastestStep, 1},
	vm.CALLDATACOPY: {3, GasFastestStep, 1},
	vm.MLOAD:        {1, GasFastestStep, 1},
	vm.MSTORE:       {2, GasFastestStep, 0},
	vm.MSTORE8:      {2, GasFastestStep, 0},
	vm.CODECOPY:     {3, GasFastestStep, 0},
	vm.MUL:          {2, GasFastStep, 1},
	vm.DIV:          {2, GasFastStep, 1},
	vm.SDIV:         {2, GasFastStep, 1},
	vm.MOD:          {2, GasFastStep, 1},
	vm.SMOD:         {2, GasFastStep, 1},
	vm.SIGNEXTEND:   {2, GasFastStep, 1},
	vm.ADDMOD:       {3, GasMidStep, 1},
	vm.MULMOD:       {3, GasMidStep, 1},
	vm.JUMP:         {1, GasMidStep, 0},
	vm.JUMPI:        {2, GasSlowStep, 0},
	vm.EXP:          {2, GasSlowStep, 1},
	vm.ADDRESS:      {0, GasQuickStep, 1},
	vm.ORIGIN:       {0, GasQuickStep, 1},
	vm.CALLER:       {0, GasQuickStep, 1},
	vm.CALLVALUE:    {0, GasQuickStep, 1},
	vm.CODESIZE:     {0, GasQuickStep, 1},
	vm.GASPRICE:     {0, GasQuickStep, 1},
	vm.COINBASE:     {0, GasQuickStep, 1},
	vm.TIMESTAMP:    {0, GasQuickStep, 1},
	vm.NUMBER:       {0, GasQuickStep, 1},
	vm.CALLDATASIZE: {0, GasQuickStep, 1},
	vm.DIFFICULTY:   {0, GasQuickStep, 1},
	vm.GASLIMIT:     {0, GasQuickStep, 1},
	vm.POP:          {1, GasQuickStep, 0},
	vm.PC:           {0, GasQuickStep, 1},
	vm.MSIZE:        {0, GasQuickStep, 1},
	vm.GAS:          {0, GasQuickStep, 1},
	vm.BLOCKHASH:    {1, GasExtStep, 1},
	vm.BALANCE:      {1, new(big.Int), 1},
	vm.EXTCODESIZE:  {1, new(big.Int), 1},
	vm.EXTCODECOPY:  {4, new(big.Int), 0},
	vm.SLOAD:        {1, big.NewInt(50), 1},
	vm.SSTORE:       {2, new(big.Int), 0},
	vm.SHA3:         {2, big.NewInt(30), 1},
	vm.CREATE:       {3, big.NewInt(32000), 1},
	// Zero is calculated in the gasSwitch
	vm.CALL:         {7, new(big.Int), 1},
	vm.CALLCODE:     {7, new(big.Int), 1},
	vm.DELEGATECALL: {6, new(big.Int), 1},
	vm.SUICIDE:      {1, new(big.Int), 0},
	vm.JUMPDEST:     {0, big.NewInt(1), 0},
	vm.RETURN:       {2, new(big.Int), 0},
	vm.PUSH1:        {0, GasFastestStep, 1},
	vm.DUP1:         {0, new(big.Int), 1},
}
