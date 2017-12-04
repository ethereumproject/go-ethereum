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

package classic

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/vm"
)

// destinations stores one map per contract (keyed by hash of code).
// The maps contain an entry for each location of a JUMPDEST
// instruction.
type destinations map[common.Hash][]byte

// has checks whether code has a JUMPDEST at dest.
func (d destinations) has(codehash common.Hash, code []byte, dest *big.Int) bool {
	// PC cannot go beyond len(code) and certainly can't be bigger than 63bits.
	// Don't bother checking for JUMPDEST in that case.
	udest := dest.Uint64()
	if dest.BitLen() >= 63 || udest >= uint64(len(code)) {
		return false
	}

	m, analysed := d[codehash]
	if !analysed {
		m = jumpdests(code)
		d[codehash] = m
	}
	return (m[udest/8] & (1 << (udest % 8))) != 0
}

// jumpdests creates a map that contains an entry for each
// PC location that is a JUMPDEST instruction.
func jumpdests(code []byte) []byte {
	m := make([]byte, len(code)/8+1)
	for pc := uint64(0); pc < uint64(len(code)); pc++ {
		op := vm.OpCode(code[pc])
		if op == vm.JUMPDEST {
			m[pc/8] |= 1 << (pc % 8)
		} else if op >= vm.PUSH1 && op <= vm.PUSH32 {
			a := uint64(op) - uint64(vm.PUSH1) + 1
			pc += a
		}
	}
	return m
}
