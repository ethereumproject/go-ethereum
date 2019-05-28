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
	"errors"
	"math/big"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/crypto"
)

var callStipend = big.NewInt(2300) // Free gas given at beginning of call.

type instrFn func(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error)

type instruction struct {
	op   OpCode
	pc   uint64
	fn   instrFn
	data *big.Int

	gas   *big.Int
	spop  int
	spush int

	returns bool
}

var (
	errReturnDataOutOfBounds = errors.New("evm: return data out of bounds")
	errInvalidJump           = errors.New("evm: invalid jump destination")
)

func (instr instruction) halts() bool {
	return instr.returns
}

func (instr instruction) Op() OpCode {
	return instr.op
}

func opAdd(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Add(x, y)))
	return nil, nil
}

func opSub(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Sub(x, y)))
	return nil, nil
}

func opMul(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Mul(x, y)))
	return nil, nil
}

func opDiv(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if y.Sign() != 0 {
		stack.push(U256(x.Div(x, y)))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opSdiv(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := S256(stack.pop()), S256(stack.pop())
	if y.Sign() == 0 {
		stack.push(new(big.Int))
		return nil, nil
	} else {
		n := new(big.Int)
		if new(big.Int).Mul(x, y).Sign() < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Div(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.push(U256(res))
	}
	return nil, nil
}

func opMod(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if y.Sign() == 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(U256(x.Mod(x, y)))
	}
	return nil, nil
}

func opSmod(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := S256(stack.pop()), S256(stack.pop())

	if y.Sign() == 0 {
		stack.push(new(big.Int))
	} else {
		n := new(big.Int)
		if x.Sign() < 0 {
			n.SetInt64(-1)
		} else {
			n.SetInt64(1)
		}

		res := x.Mod(x.Abs(x), y.Abs(y))
		res.Mul(res, n)

		stack.push(U256(res))
	}
	return nil, nil
}

func opExp(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(U256(x.Exp(x, y, Pow256)))
	return nil, nil
}

func opSignExtend(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	back := stack.pop()
	if back.Cmp(big.NewInt(31)) < 0 {
		bit := uint(back.Uint64()*8 + 7)
		num := stack.pop()
		mask := back.Lsh(common.Big1, bit)
		mask.Sub(mask, common.Big1)
		if common.BitTest(num, int(bit)) {
			num.Or(num, mask.Not(mask))
		} else {
			num.And(num, mask)
		}

		stack.push(U256(num))
	}
	return nil, nil
}

func opNot(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x := stack.pop()
	stack.push(U256(x.Not(x)))
	return nil, nil
}

func opLt(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) < 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opGt(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) > 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opSlt(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := S256(stack.pop()), S256(stack.pop())
	if x.Cmp(S256(y)) < 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opSgt(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := S256(stack.pop()), S256(stack.pop())
	if x.Cmp(y) > 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opEq(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	if x.Cmp(y) == 0 {
		stack.push(big.NewInt(1))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opIszero(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x := stack.pop()
	if x.Sign() != 0 {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	return nil, nil
}

func opAnd(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.And(x, y))
	return nil, nil
}
func opOr(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Or(x, y))
	return nil, nil
}
func opXor(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y := stack.pop(), stack.pop()
	stack.push(x.Xor(x, y))
	return nil, nil
}
func opByte(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	th, val := stack.pop(), stack.pop()
	if th.Cmp(big.NewInt(32)) < 0 {
		byte := big.NewInt(int64(common.LeftPadBytes(val.Bytes(), 32)[th.Int64()]))
		stack.push(byte)
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}
func opAddmod(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Sign() > 0 {
		add := x.Add(x, y)
		add.Mod(add, z)
		stack.push(U256(add))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}
func opMulmod(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	x, y, z := stack.pop(), stack.pop(), stack.pop()
	if z.Sign() > 0 {
		mul := x.Mul(x, y)
		mul.Mod(mul, z)
		stack.push(U256(mul))
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opSha3(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	hash := crypto.Keccak256(memory.Get(offset.Int64(), size.Int64()))

	stack.push(new(big.Int).SetBytes(hash))
	return nil, nil
}

func opAddress(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(new(big.Int).SetBytes(contract.Address().Bytes()))
	return nil, nil
}

func opBalance(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	addr := common.BigToAddress(stack.pop())
	balance := env.Db().GetBalance(addr)

	stack.push(new(big.Int).Set(balance))
	return nil, nil
}

func opOrigin(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(env.Origin().Big())
	return nil, nil
}

func opCaller(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(contract.Caller().Big())
	return nil, nil
}

func opCallValue(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(new(big.Int).Set(contract.value))
	return nil, nil
}

func opCalldataLoad(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(new(big.Int).SetBytes(getData(contract.Input, stack.pop(), common.Big32)))
	return nil, nil
}

func opCalldataSize(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(big.NewInt(int64(len(contract.Input))))
	return nil, nil
}

func opCalldataCopy(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	var (
		mOff = stack.pop()
		cOff = stack.pop()
		l    = stack.pop()
	)
	memory.Set(mOff.Uint64(), l.Uint64(), getData(contract.Input, cOff, l))
	return nil, nil
}

func opExtCodeSize(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	addr := common.BigToAddress(stack.pop())
	l := big.NewInt(int64(env.Db().GetCodeSize(addr)))
	stack.push(l)
	return nil, nil
}

func opCodeSize(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	l := big.NewInt(int64(len(contract.Code)))
	stack.push(l)
	return nil, nil
}

func opCodeCopy(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	var (
		mOff = stack.pop()
		cOff = stack.pop()
		l    = stack.pop()
	)
	codeCopy := getData(contract.Code, cOff, l)

	memory.Set(mOff.Uint64(), l.Uint64(), codeCopy)
	return nil, nil
}

func opExtCodeCopy(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	var (
		addr = common.BigToAddress(stack.pop())
		mOff = stack.pop()
		cOff = stack.pop()
		l    = stack.pop()
	)
	codeCopy := getData(env.Db().GetCode(addr), cOff, l)

	memory.Set(mOff.Uint64(), l.Uint64(), codeCopy)
	return nil, nil
}

func opGasprice(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(new(big.Int).Set(contract.Price))
	return nil, nil
}

func opBlockhash(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	num := stack.pop()

	n := new(big.Int).Sub(env.BlockNumber(), common.Big257)
	if num.Cmp(n) > 0 && num.Cmp(env.BlockNumber()) < 0 {
		stack.push(env.GetHash(num.Uint64()).Big())
	} else {
		stack.push(new(big.Int))
	}
	return nil, nil
}

func opCoinbase(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(env.Coinbase().Big())
	return nil, nil
}

func opTimestamp(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(U256(new(big.Int).Set(env.Time())))
	return nil, nil
}

func opNumber(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(U256(new(big.Int).Set(env.BlockNumber())))
	return nil, nil
}

func opDifficulty(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(U256(new(big.Int).Set(env.Difficulty())))
	return nil, nil
}

func opGasLimit(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(U256(new(big.Int).Set(env.GasLimit())))
	return nil, nil
}

func opPop(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.pop()
	return nil, nil
}

func opMload(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	offset := stack.pop()
	val := new(big.Int).SetBytes(memory.Get(offset.Int64(), 32))
	stack.push(val)
	return nil, nil
}

func opMstore(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	// pop value of the stack
	mStart, val := stack.pop(), stack.pop()
	memory.Set(mStart.Uint64(), 32, common.BigToBytes(val, 256))
	return nil, nil
}

func opMstore8(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	off, val := stack.pop().Int64(), stack.pop().Int64()
	memory.store[off] = byte(val & 0xff)
	return nil, nil
}

func opSload(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	loc := common.BigToHash(stack.pop())
	val := env.Db().GetState(contract.Address(), loc).Big()
	stack.push(val)
	return nil, nil
}

func opSstore(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	loc := common.BigToHash(stack.pop())
	val := stack.pop()
	env.Db().SetState(contract.Address(), loc, common.BigToHash(val))
	return nil, nil
}

func opJump(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	pos := stack.pop()
	if !contract.isValidJump(pc, pos) {
		return nil, errInvalidJump
	}

	*pc = pos.Uint64()
	return nil, nil
}

func opJumpi(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	pos, cond := stack.pop(), stack.pop()
	if cond.Sign() != 0 {
		if !contract.isValidJump(pc, pos) {
			return nil, errInvalidJump
		}

		*pc = pos.Uint64()
		return nil, nil
	}
	*pc++
	return nil, nil
}

func opJumpdest(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	return nil, nil
}

func opPc(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(new(big.Int).SetUint64(*pc))
	return nil, nil
}

func opMsize(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(big.NewInt(int64(memory.Len())))
	return nil, nil
}

func opGas(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	stack.push(new(big.Int).Set(contract.Gas))
	return nil, nil
}

func opCreate(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	var (
		value        = stack.pop()
		offset, size = stack.pop(), stack.pop()
		input        = memory.Get(offset.Int64(), size.Int64())
		gas          = new(big.Int).Set(contract.Gas)
	)
	if env.RuleSet().GasTable(env.BlockNumber()).CreateBySuicide != nil {
		gas.Div(gas, n64)
		gas = gas.Sub(contract.Gas, gas)
	}

	contract.UseGas(gas)
	ret, addr, suberr := env.Create(contract, input, gas, contract.Price, value)
	// Push item on the stack based on the returned error. If the ruleset is
	// homestead we must check for CodeStoreOutOfGasError (homestead only
	// rule) and treat as an error, if the ruleset is frontier we must
	// ignore this error and pretend the operation was successful.
	if env.RuleSet().IsHomestead(env.BlockNumber()) && suberr == CodeStoreOutOfGasError {
		stack.push(new(big.Int))
	} else if suberr != nil && suberr != CodeStoreOutOfGasError {
		stack.push(new(big.Int))
	} else {
		stack.push(addr.Big())
	}

	if suberr == ErrRevert {
		return ret, nil
	}
	return nil, nil
}

func opCall(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	gas := stack.pop()
	// pop gas and value of the stack.
	addr, value := stack.pop(), stack.pop()
	value = U256(value)
	// pop input size and offset
	inOffset, inSize := stack.pop(), stack.pop()
	// pop return size and offset
	retOffset, retSize := stack.pop(), stack.pop()

	address := common.BigToAddress(addr)

	// Get the arguments from the memory
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if len(value.Bytes()) > 0 {
		gas.Add(gas, callStipend)
	}

	ret, err := env.Call(contract, address, args, gas, contract.Price, value)

	if err != nil {
		stack.push(new(big.Int))

	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == ErrRevert {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	return ret, nil
}

func opCallCode(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	gas := stack.pop()
	// pop gas and value of the stack.
	addr, value := stack.pop(), stack.pop()
	value = U256(value)
	// pop input size and offset
	inOffset, inSize := stack.pop(), stack.pop()
	// pop return size and offset
	retOffset, retSize := stack.pop(), stack.pop()

	address := common.BigToAddress(addr)

	// Get the arguments from the memory
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	if len(value.Bytes()) > 0 {
		gas.Add(gas, callStipend)
	}

	ret, err := env.CallCode(contract, address, args, gas, contract.Price, value)

	if err != nil {
		stack.push(new(big.Int))

	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == ErrRevert {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	return ret, nil
}

func opDelegateCall(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	gas, to, inOffset, inSize, outOffset, outSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()

	toAddr := common.BigToAddress(to)
	args := memory.Get(inOffset.Int64(), inSize.Int64())
	ret, err := env.DelegateCall(contract, toAddr, args, gas, contract.Price)
	if err != nil {
		stack.push(new(big.Int))
	} else {
		stack.push(big.NewInt(1))
	}
	if err == nil || err == ErrRevert {
		memory.Set(outOffset.Uint64(), outSize.Uint64(), ret)
	}
	return ret, nil
}

func opReturn(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	return ret, nil
}

func opRevert(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	return ret, nil
}

func opStop(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	return nil, nil
}

func opSuicide(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	balance := env.Db().GetBalance(contract.Address())
	env.Db().AddBalance(common.BigToAddress(stack.pop()), balance)

	env.Db().Suicide(contract.Address())
	return nil, nil
}

func opReturnDataSize(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	returnDataSize := new(big.Int).SetUint64((uint64(len(env.ReturnData()))))
	stack.push(returnDataSize)
	return nil, nil
}

func opReturnDataCopy(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		dataOffset = stack.pop()
		length     = stack.pop()

		end = new(big.Int).Add(dataOffset, length)
	)

	if !end.IsUint64() || uint64(len(env.ReturnData())) < end.Uint64() {
		return nil, errReturnDataOutOfBounds
	}

	memory.Set(memOffset.Uint64(), length.Uint64(), getData(env.ReturnData(), dataOffset, length))

	return nil, nil
}

// following functions are used by the instruction jump  table

// make log instruction function
func makeLog(size int) instrFn {
	return func(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
		topics := make([]common.Hash, size)
		mStart, mSize := stack.pop(), stack.pop()
		for i := 0; i < size; i++ {
			topics[i] = common.BigToHash(stack.pop())
		}

		d := memory.Get(mStart.Int64(), mSize.Int64())
		log := NewLog(contract.Address(), topics, d, env.BlockNumber().Uint64())
		env.AddLog(log)
		return nil, nil
	}
}

// make push instruction function
func makePush(size uint64, bsize *big.Int) instrFn {
	return func(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
		bytes := getData(contract.Code, new(big.Int).SetUint64(*pc+1), bsize)
		stack.push(new(big.Int).SetBytes(bytes))
		*pc += size
		return nil, nil
	}
}

// make push instruction function
func makeDup(size int64) instrFn {
	return func(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
		stack.dup(int(size))
		return nil, nil
	}
}

// make swap instruction function
func makeSwap(size int64) instrFn {
	// switch n + 1 otherwise n would be swapped with n
	size += 1
	return func(pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
		stack.swap(int(size))
		return nil, nil
	}
}
