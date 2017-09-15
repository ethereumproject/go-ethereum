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
	"math/big"
	"github.com/ethereumproject/go-ethereum/core/vm"
)

type jumpPtr struct {
	fn    instrFn
	valid bool
}

type vmJumpTable [256]jumpPtr

func newJumpTable(ruleset vm.RuleSet, blockNumber *big.Int) vmJumpTable {
	var jumpTable vmJumpTable

	// when initialising a new VM execution we must first check the homestead
	// changes.
	if ruleset.IsHomestead(blockNumber) {
		jumpTable[vm.DELEGATECALL] = jumpPtr{opDelegateCall, true}
	}

	jumpTable[vm.ADD] = jumpPtr{opAdd, true}
	jumpTable[vm.SUB] = jumpPtr{opSub, true}
	jumpTable[vm.MUL] = jumpPtr{opMul, true}
	jumpTable[vm.DIV] = jumpPtr{opDiv, true}
	jumpTable[vm.SDIV] = jumpPtr{opSdiv, true}
	jumpTable[vm.MOD] = jumpPtr{opMod, true}
	jumpTable[vm.SMOD] = jumpPtr{opSmod, true}
	jumpTable[vm.EXP] = jumpPtr{opExp, true}
	jumpTable[vm.SIGNEXTEND] = jumpPtr{opSignExtend, true}
	jumpTable[vm.NOT] = jumpPtr{opNot, true}
	jumpTable[vm.LT] = jumpPtr{opLt, true}
	jumpTable[vm.GT] = jumpPtr{opGt, true}
	jumpTable[vm.SLT] = jumpPtr{opSlt, true}
	jumpTable[vm.SGT] = jumpPtr{opSgt, true}
	jumpTable[vm.EQ] = jumpPtr{opEq, true}
	jumpTable[vm.ISZERO] = jumpPtr{opIszero, true}
	jumpTable[vm.AND] = jumpPtr{opAnd, true}
	jumpTable[vm.OR] = jumpPtr{opOr, true}
	jumpTable[vm.XOR] = jumpPtr{opXor, true}
	jumpTable[vm.BYTE] = jumpPtr{opByte, true}
	jumpTable[vm.ADDMOD] = jumpPtr{opAddmod, true}
	jumpTable[vm.MULMOD] = jumpPtr{opMulmod, true}
	jumpTable[vm.SHA3] = jumpPtr{opSha3, true}
	jumpTable[vm.ADDRESS] = jumpPtr{opAddress, true}
	jumpTable[vm.BALANCE] = jumpPtr{opBalance, true}
	jumpTable[vm.ORIGIN] = jumpPtr{opOrigin, true}
	jumpTable[vm.CALLER] = jumpPtr{opCaller, true}
	jumpTable[vm.CALLVALUE] = jumpPtr{opCallValue, true}
	jumpTable[vm.CALLDATALOAD] = jumpPtr{opCalldataLoad, true}
	jumpTable[vm.CALLDATASIZE] = jumpPtr{opCalldataSize, true}
	jumpTable[vm.CALLDATACOPY] = jumpPtr{opCalldataCopy, true}
	jumpTable[vm.CODESIZE] = jumpPtr{opCodeSize, true}
	jumpTable[vm.EXTCODESIZE] = jumpPtr{opExtCodeSize, true}
	jumpTable[vm.CODECOPY] = jumpPtr{opCodeCopy, true}
	jumpTable[vm.EXTCODECOPY] = jumpPtr{opExtCodeCopy, true}
	jumpTable[vm.GASPRICE] = jumpPtr{opGasprice, true}
	jumpTable[vm.BLOCKHASH] = jumpPtr{opBlockhash, true}
	jumpTable[vm.COINBASE] = jumpPtr{opCoinbase, true}
	jumpTable[vm.TIMESTAMP] = jumpPtr{opTimestamp, true}
	jumpTable[vm.NUMBER] = jumpPtr{opNumber, true}
	jumpTable[vm.DIFFICULTY] = jumpPtr{opDifficulty, true}
	jumpTable[vm.GASLIMIT] = jumpPtr{opGasLimit, true}
	jumpTable[vm.POP] = jumpPtr{opPop, true}
	jumpTable[vm.MLOAD] = jumpPtr{opMload, true}
	jumpTable[vm.MSTORE] = jumpPtr{opMstore, true}
	jumpTable[vm.MSTORE8] = jumpPtr{opMstore8, true}
	jumpTable[vm.SLOAD] = jumpPtr{opSload, true}
	jumpTable[vm.SSTORE] = jumpPtr{opSstore, true}
	jumpTable[vm.JUMPDEST] = jumpPtr{opJumpdest, true}
	jumpTable[vm.PC] = jumpPtr{nil, true}
	jumpTable[vm.MSIZE] = jumpPtr{opMsize, true}
	jumpTable[vm.GAS] = jumpPtr{opGas, true}
	jumpTable[vm.CREATE] = jumpPtr{opCreate, true}
	jumpTable[vm.CALL] = jumpPtr{opCall, true}
	jumpTable[vm.CALLCODE] = jumpPtr{opCallCode, true}
	jumpTable[vm.LOG0] = jumpPtr{makeLog(0), true}
	jumpTable[vm.LOG1] = jumpPtr{makeLog(1), true}
	jumpTable[vm.LOG2] = jumpPtr{makeLog(2), true}
	jumpTable[vm.LOG3] = jumpPtr{makeLog(3), true}
	jumpTable[vm.LOG4] = jumpPtr{makeLog(4), true}
	jumpTable[vm.SWAP1] = jumpPtr{makeSwap(1), true}
	jumpTable[vm.SWAP2] = jumpPtr{makeSwap(2), true}
	jumpTable[vm.SWAP3] = jumpPtr{makeSwap(3), true}
	jumpTable[vm.SWAP4] = jumpPtr{makeSwap(4), true}
	jumpTable[vm.SWAP5] = jumpPtr{makeSwap(5), true}
	jumpTable[vm.SWAP6] = jumpPtr{makeSwap(6), true}
	jumpTable[vm.SWAP7] = jumpPtr{makeSwap(7), true}
	jumpTable[vm.SWAP8] = jumpPtr{makeSwap(8), true}
	jumpTable[vm.SWAP9] = jumpPtr{makeSwap(9), true}
	jumpTable[vm.SWAP10] = jumpPtr{makeSwap(10), true}
	jumpTable[vm.SWAP11] = jumpPtr{makeSwap(11), true}
	jumpTable[vm.SWAP12] = jumpPtr{makeSwap(12), true}
	jumpTable[vm.SWAP13] = jumpPtr{makeSwap(13), true}
	jumpTable[vm.SWAP14] = jumpPtr{makeSwap(14), true}
	jumpTable[vm.SWAP15] = jumpPtr{makeSwap(15), true}
	jumpTable[vm.SWAP16] = jumpPtr{makeSwap(16), true}
	jumpTable[vm.PUSH1] = jumpPtr{makePush(1, big.NewInt(1)), true}
	jumpTable[vm.PUSH2] = jumpPtr{makePush(2, big.NewInt(2)), true}
	jumpTable[vm.PUSH3] = jumpPtr{makePush(3, big.NewInt(3)), true}
	jumpTable[vm.PUSH4] = jumpPtr{makePush(4, big.NewInt(4)), true}
	jumpTable[vm.PUSH5] = jumpPtr{makePush(5, big.NewInt(5)), true}
	jumpTable[vm.PUSH6] = jumpPtr{makePush(6, big.NewInt(6)), true}
	jumpTable[vm.PUSH7] = jumpPtr{makePush(7, big.NewInt(7)), true}
	jumpTable[vm.PUSH8] = jumpPtr{makePush(8, big.NewInt(8)), true}
	jumpTable[vm.PUSH9] = jumpPtr{makePush(9, big.NewInt(9)), true}
	jumpTable[vm.PUSH10] = jumpPtr{makePush(10, big.NewInt(10)), true}
	jumpTable[vm.PUSH11] = jumpPtr{makePush(11, big.NewInt(11)), true}
	jumpTable[vm.PUSH12] = jumpPtr{makePush(12, big.NewInt(12)), true}
	jumpTable[vm.PUSH13] = jumpPtr{makePush(13, big.NewInt(13)), true}
	jumpTable[vm.PUSH14] = jumpPtr{makePush(14, big.NewInt(14)), true}
	jumpTable[vm.PUSH15] = jumpPtr{makePush(15, big.NewInt(15)), true}
	jumpTable[vm.PUSH16] = jumpPtr{makePush(16, big.NewInt(16)), true}
	jumpTable[vm.PUSH17] = jumpPtr{makePush(17, big.NewInt(17)), true}
	jumpTable[vm.PUSH18] = jumpPtr{makePush(18, big.NewInt(18)), true}
	jumpTable[vm.PUSH19] = jumpPtr{makePush(19, big.NewInt(19)), true}
	jumpTable[vm.PUSH20] = jumpPtr{makePush(20, big.NewInt(20)), true}
	jumpTable[vm.PUSH21] = jumpPtr{makePush(21, big.NewInt(21)), true}
	jumpTable[vm.PUSH22] = jumpPtr{makePush(22, big.NewInt(22)), true}
	jumpTable[vm.PUSH23] = jumpPtr{makePush(23, big.NewInt(23)), true}
	jumpTable[vm.PUSH24] = jumpPtr{makePush(24, big.NewInt(24)), true}
	jumpTable[vm.PUSH25] = jumpPtr{makePush(25, big.NewInt(25)), true}
	jumpTable[vm.PUSH26] = jumpPtr{makePush(26, big.NewInt(26)), true}
	jumpTable[vm.PUSH27] = jumpPtr{makePush(27, big.NewInt(27)), true}
	jumpTable[vm.PUSH28] = jumpPtr{makePush(28, big.NewInt(28)), true}
	jumpTable[vm.PUSH29] = jumpPtr{makePush(29, big.NewInt(29)), true}
	jumpTable[vm.PUSH30] = jumpPtr{makePush(30, big.NewInt(30)), true}
	jumpTable[vm.PUSH31] = jumpPtr{makePush(31, big.NewInt(31)), true}
	jumpTable[vm.PUSH32] = jumpPtr{makePush(32, big.NewInt(32)), true}
	jumpTable[vm.DUP1] = jumpPtr{makeDup(1), true}
	jumpTable[vm.DUP2] = jumpPtr{makeDup(2), true}
	jumpTable[vm.DUP3] = jumpPtr{makeDup(3), true}
	jumpTable[vm.DUP4] = jumpPtr{makeDup(4), true}
	jumpTable[vm.DUP5] = jumpPtr{makeDup(5), true}
	jumpTable[vm.DUP6] = jumpPtr{makeDup(6), true}
	jumpTable[vm.DUP7] = jumpPtr{makeDup(7), true}
	jumpTable[vm.DUP8] = jumpPtr{makeDup(8), true}
	jumpTable[vm.DUP9] = jumpPtr{makeDup(9), true}
	jumpTable[vm.DUP10] = jumpPtr{makeDup(10), true}
	jumpTable[vm.DUP11] = jumpPtr{makeDup(11), true}
	jumpTable[vm.DUP12] = jumpPtr{makeDup(12), true}
	jumpTable[vm.DUP13] = jumpPtr{makeDup(13), true}
	jumpTable[vm.DUP14] = jumpPtr{makeDup(14), true}
	jumpTable[vm.DUP15] = jumpPtr{makeDup(15), true}
	jumpTable[vm.DUP16] = jumpPtr{makeDup(16), true}

	jumpTable[vm.RETURN] = jumpPtr{nil, true}
	jumpTable[vm.SUICIDE] = jumpPtr{nil, true}
	jumpTable[vm.JUMP] = jumpPtr{nil, true}
	jumpTable[vm.JUMPI] = jumpPtr{nil, true}
	jumpTable[vm.STOP] = jumpPtr{nil, true}

	return jumpTable
}
