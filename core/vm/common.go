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
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
)

// Type is the VM type accepted by **NewVm**
type Type byte

const (
	StdVmTy Type = iota // Default standard VM
	JitVmTy             // LLVM JIT VM
	MaxVmTy
)

var (
	Pow256 = common.BigPow(2, 256) // Pow256 is 2**256

	U256 = common.U256 // Shortcut to common.U256
	S256 = common.S256 // Shortcut to common.S256

	One = common.Big1 // Shortcut to common.Big1
)

// calculates the memory size required for a step
func calcMemSize(off, l *big.Int) *big.Int {
	if l.Sign() == 0 {
		return new(big.Int)
	}

	return new(big.Int).Add(off, l)
}

// calculates the quadratic gas
func quadMemGas(mem *Memory, newMemSize, gas *big.Int) {
	if newMemSize.Sign() > 0 {
		newMemSizeWords := toWordSize(newMemSize)
		newMemSize.Mul(newMemSizeWords, u256(32))

		if newMemSize.Cmp(u256(int64(mem.Len()))) > 0 {
			// be careful reusing variables here when changing.
			// The order has been optimised to reduce allocation
			oldSize := toWordSize(big.NewInt(int64(mem.Len())))
			pow := new(big.Int).Exp(oldSize, common.Big2, new(big.Int))
			linCoef := oldSize.Mul(oldSize, big.NewInt(3))
			quadCoef := new(big.Int).Div(pow, big.NewInt(512))
			oldTotalFee := new(big.Int).Add(linCoef, quadCoef)

			pow.Exp(newMemSizeWords, common.Big2, new(big.Int))
			linCoef = linCoef.Mul(newMemSizeWords, big.NewInt(3))
			quadCoef = quadCoef.Div(pow, big.NewInt(512))
			newTotalFee := linCoef.Add(linCoef, quadCoef)

			fee := newTotalFee.Sub(newTotalFee, oldTotalFee)
			gas.Add(gas, fee)
		}
	}
}

// Simple helper
func u256(n int64) *big.Int {
	return big.NewInt(n)
}

// getData returns a slice from the data based on the start and size and pads
// up to size with zero's. This function is overflow safe.
func getData(data []byte, start, size *big.Int) []byte {
	dlen := big.NewInt(int64(len(data)))

	s := common.BigMin(start, dlen)
	e := common.BigMin(new(big.Int).Add(s, size), dlen)
	return common.RightPadBytes(data[s.Uint64():e.Uint64()], int(size.Uint64()))
}

// useGas attempts to subtract the amount of gas and returns whether it was
// successful
func useGas(gas, amount *big.Int) bool {
	if gas.Cmp(amount) < 0 {
		return false
	}

	// Sub the amount of gas from the remaining
	gas.Sub(gas, amount)
	return true
}
