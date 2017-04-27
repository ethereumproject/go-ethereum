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

package vm

import (
	"fmt"
	"math/big"
	"reflect"
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

type GasTable struct {
	ExtcodeSize *big.Int
	ExtcodeCopy *big.Int
	Balance     *big.Int
	SLoad       *big.Int
	Calls       *big.Int
	Suicide     *big.Int
	ExpByte     *big.Int

	// CreateBySuicide occurs when the
	// refunded account is one that does
	// not exist. This logic is similar
	// to call. May be left nil. Nil means
	// not charged.
	CreateBySuicide *big.Int
}

// calcGas returns the actual gas cost of the call.
//
// The cost of gas was changed during the homestead price change HF. To allow for EIP150
// to be implemented. The returned gas is gas - base * 63 / 64.
func callGas(gasTable *GasTable, availableGas, base, callCost *big.Int) *big.Int {
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

// IsEmpty return true if all values are zero values,
// which useful for checking JSON-decoded empty state.
func (g *GasTable) IsEmpty() bool {
	return reflect.DeepEqual(g, GasTable{})
}

// baseCheck checks for any stack error underflows
func baseCheck(op OpCode, stack *stack, gas *big.Int) error {
	// PUSH and DUP are a bit special. They all cost the same but we do want to have checking on stack push limit
	// PUSH is also allowed to calculate the same price for all PUSHes
	// DUP requirements are handled elsewhere (except for the stack limit check)
	if op >= PUSH1 && op <= PUSH32 {
		op = PUSH1
	}
	if op >= DUP1 && op <= DUP16 {
		op = DUP1
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

var _baseCheck = map[OpCode]req{
	// opcode  |  stack pop | gas price | stack push
	ADD:          {2, GasFastestStep, 1},
	LT:           {2, GasFastestStep, 1},
	GT:           {2, GasFastestStep, 1},
	SLT:          {2, GasFastestStep, 1},
	SGT:          {2, GasFastestStep, 1},
	EQ:           {2, GasFastestStep, 1},
	ISZERO:       {1, GasFastestStep, 1},
	SUB:          {2, GasFastestStep, 1},
	AND:          {2, GasFastestStep, 1},
	OR:           {2, GasFastestStep, 1},
	XOR:          {2, GasFastestStep, 1},
	NOT:          {1, GasFastestStep, 1},
	BYTE:         {2, GasFastestStep, 1},
	CALLDATALOAD: {1, GasFastestStep, 1},
	CALLDATACOPY: {3, GasFastestStep, 1},
	MLOAD:        {1, GasFastestStep, 1},
	MSTORE:       {2, GasFastestStep, 0},
	MSTORE8:      {2, GasFastestStep, 0},
	CODECOPY:     {3, GasFastestStep, 0},
	MUL:          {2, GasFastStep, 1},
	DIV:          {2, GasFastStep, 1},
	SDIV:         {2, GasFastStep, 1},
	MOD:          {2, GasFastStep, 1},
	SMOD:         {2, GasFastStep, 1},
	SIGNEXTEND:   {2, GasFastStep, 1},
	ADDMOD:       {3, GasMidStep, 1},
	MULMOD:       {3, GasMidStep, 1},
	JUMP:         {1, GasMidStep, 0},
	JUMPI:        {2, GasSlowStep, 0},
	EXP:          {2, GasSlowStep, 1},
	ADDRESS:      {0, GasQuickStep, 1},
	ORIGIN:       {0, GasQuickStep, 1},
	CALLER:       {0, GasQuickStep, 1},
	CALLVALUE:    {0, GasQuickStep, 1},
	CODESIZE:     {0, GasQuickStep, 1},
	GASPRICE:     {0, GasQuickStep, 1},
	COINBASE:     {0, GasQuickStep, 1},
	TIMESTAMP:    {0, GasQuickStep, 1},
	NUMBER:       {0, GasQuickStep, 1},
	CALLDATASIZE: {0, GasQuickStep, 1},
	DIFFICULTY:   {0, GasQuickStep, 1},
	GASLIMIT:     {0, GasQuickStep, 1},
	POP:          {1, GasQuickStep, 0},
	PC:           {0, GasQuickStep, 1},
	MSIZE:        {0, GasQuickStep, 1},
	GAS:          {0, GasQuickStep, 1},
	BLOCKHASH:    {1, GasExtStep, 1},
	BALANCE:      {1, new(big.Int), 1},
	EXTCODESIZE:  {1, new(big.Int), 1},
	EXTCODECOPY:  {4, new(big.Int), 0},
	SLOAD:        {1, big.NewInt(50), 1},
	SSTORE:       {2, new(big.Int), 0},
	SHA3:         {2, big.NewInt(30), 1},
	CREATE:       {3, big.NewInt(32000), 1},
	// Zero is calculated in the gasSwitch
	CALL:         {7, new(big.Int), 1},
	CALLCODE:     {7, new(big.Int), 1},
	DELEGATECALL: {6, new(big.Int), 1},
	SUICIDE:      {1, new(big.Int), 0},
	JUMPDEST:     {0, big.NewInt(1), 0},
	RETURN:       {2, new(big.Int), 0},
	PUSH1:        {0, GasFastestStep, 1},
	DUP1:         {0, new(big.Int), 1},
}
