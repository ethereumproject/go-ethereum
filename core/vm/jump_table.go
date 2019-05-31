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

import "math/big"

type jumpPtr struct {
	fn    instrFn
	valid bool

	jumps   bool // indicates whether the program counter should not increment
	halts   bool // indicates whether the operation should halt further execution
	reverts bool // determines whether the operation reverts state (implicitly halts)
	returns bool // Indicates whether return data should be overwritten
	writes  bool // determines whether this a state modifying operation
}

type vmJumpTable [256]jumpPtr

func newJumpTable(ruleset RuleSet, blockNumber *big.Int) vmJumpTable {
	jumpTable := newFrontierInstructionSet()

	// when initialising a new VM execution we must first check the homestead
	// changes.
	if ruleset.IsHomestead(blockNumber) {
		jumpTable[DELEGATECALL] = jumpPtr{
			fn:      opDelegateCall,
			valid:   true,
			returns: true,
		}
	}

	if ruleset.IsAtlantis(blockNumber) {
		jumpTable[REVERT] = jumpPtr{
			fn:      opRevert,
			valid:   true,
			reverts: true,
			returns: true,
		}
		jumpTable[RETURNDATASIZE] = jumpPtr{
			fn:    opReturnDataSize,
			valid: true,
		}
		jumpTable[RETURNDATACOPY] = jumpPtr{
			fn:    opReturnDataCopy,
			valid: true,
		}
		jumpTable[STATICCALL] = jumpPtr{
			fn:      opStaticCall,
			valid:   true,
			returns: true,
		}
	}

	return jumpTable
}

func newFrontierInstructionSet() vmJumpTable {
	return vmJumpTable{
		ADD: {
			fn:    opAdd,
			valid: true,
		},
		SUB: {
			fn:    opSub,
			valid: true,
		},
		MUL: {
			fn:    opMul,
			valid: true,
		},
		DIV: {
			fn:    opDiv,
			valid: true,
		},
		SDIV: {
			fn:    opSdiv,
			valid: true,
		},
		MOD: {
			fn:    opMod,
			valid: true,
		},
		SMOD: {
			fn:    opSmod,
			valid: true,
		},
		EXP: {
			fn:    opExp,
			valid: true,
		},
		SIGNEXTEND: {
			fn:    opSignExtend,
			valid: true,
		},
		NOT: {
			fn:    opNot,
			valid: true,
		},
		LT: {
			fn:    opLt,
			valid: true,
		},
		GT: {
			fn:    opGt,
			valid: true,
		},
		SLT: {
			fn:    opSlt,
			valid: true,
		},
		SGT: {
			fn:    opSgt,
			valid: true,
		},
		EQ: {
			fn:    opEq,
			valid: true,
		},
		ISZERO: {
			fn:    opIszero,
			valid: true,
		},
		AND: {
			fn:    opAnd,
			valid: true,
		},
		OR: {
			fn:    opOr,
			valid: true,
		},
		XOR: {
			fn:    opXor,
			valid: true,
		},
		BYTE: {
			fn:    opByte,
			valid: true,
		},
		ADDMOD: {
			fn:    opAddmod,
			valid: true,
		},
		MULMOD: {
			fn:    opMulmod,
			valid: true,
		},
		SHA3: {
			fn:    opSha3,
			valid: true,
		},
		ADDRESS: {
			fn:    opAddress,
			valid: true,
		},
		BALANCE: {
			fn:    opBalance,
			valid: true,
		},
		ORIGIN: {
			fn:    opOrigin,
			valid: true,
		},
		CALLER: {
			fn:    opCaller,
			valid: true,
		},
		CALLVALUE: {
			fn:    opCallValue,
			valid: true,
		},
		CALLDATALOAD: {
			fn:    opCalldataLoad,
			valid: true,
		},
		CALLDATASIZE: {
			fn:    opCalldataSize,
			valid: true,
		},
		CALLDATACOPY: {
			fn:    opCalldataCopy,
			valid: true,
		},
		CODESIZE: {
			fn:    opCodeSize,
			valid: true,
		},
		EXTCODESIZE: {
			fn:    opExtCodeSize,
			valid: true,
		},
		CODECOPY: {
			fn:    opCodeCopy,
			valid: true,
		},
		EXTCODECOPY: {
			fn:    opExtCodeCopy,
			valid: true,
		},
		GASPRICE: {
			fn:    opGasprice,
			valid: true,
		},
		BLOCKHASH: {
			fn:    opBlockhash,
			valid: true,
		},
		COINBASE: {
			fn:    opCoinbase,
			valid: true,
		},
		TIMESTAMP: {
			fn:    opTimestamp,
			valid: true,
		},
		NUMBER: {
			fn:    opNumber,
			valid: true,
		},
		DIFFICULTY: {
			fn:    opDifficulty,
			valid: true,
		},
		GASLIMIT: {
			fn:    opGasLimit,
			valid: true,
		},
		POP: {
			fn:    opPop,
			valid: true,
		},
		MLOAD: {
			fn:    opMload,
			valid: true,
		},
		MSTORE: {
			fn:    opMstore,
			valid: true,
		},
		MSTORE8: {
			fn:    opMstore8,
			valid: true,
		},
		SLOAD: {
			fn:    opSload,
			valid: true,
		},
		SSTORE: {
			fn:     opSstore,
			valid:  true,
			writes: true,
		},
		JUMPDEST: {
			fn:    opJumpdest,
			valid: true,
		},
		PC: {
			fn:    opPc,
			valid: true,
		},
		MSIZE: {
			fn:    opMsize,
			valid: true,
		},
		GAS: {
			fn:    opGas,
			valid: true,
		},
		CREATE: {
			fn:      opCreate,
			valid:   true,
			writes:  true,
			returns: true,
		},
		CALL: {
			fn:      opCall,
			valid:   true,
			returns: true,
		},
		CALLCODE: {
			fn:      opCallCode,
			valid:   true,
			returns: true,
		},
		LOG0: {
			fn:     makeLog(0),
			valid:  true,
			writes: true,
		},
		LOG1: {
			fn:     makeLog(1),
			valid:  true,
			writes: true,
		},
		LOG2: {
			fn:     makeLog(2),
			valid:  true,
			writes: true,
		},
		LOG3: {
			fn:     makeLog(3),
			valid:  true,
			writes: true,
		},
		LOG4: {
			fn:     makeLog(4),
			valid:  true,
			writes: true,
		},
		SWAP1: {
			fn:    makeSwap(1),
			valid: true,
		},
		SWAP2: {
			fn:    makeSwap(2),
			valid: true,
		},
		SWAP3: {
			fn:    makeSwap(3),
			valid: true,
		},
		SWAP4: {
			fn:    makeSwap(4),
			valid: true,
		},
		SWAP5: {
			fn:    makeSwap(5),
			valid: true,
		},
		SWAP6: {
			fn:    makeSwap(6),
			valid: true,
		},
		SWAP7: {
			fn:    makeSwap(7),
			valid: true,
		},
		SWAP8: {
			fn:    makeSwap(8),
			valid: true,
		},
		SWAP9: {
			fn:    makeSwap(9),
			valid: true,
		},
		SWAP10: {
			fn:    makeSwap(10),
			valid: true,
		},
		SWAP11: {
			fn:    makeSwap(11),
			valid: true,
		},
		SWAP12: {
			fn:    makeSwap(12),
			valid: true,
		},
		SWAP13: {
			fn:    makeSwap(13),
			valid: true,
		},
		SWAP14: {
			fn:    makeSwap(14),
			valid: true,
		},
		SWAP15: {
			fn:    makeSwap(15),
			valid: true,
		},
		SWAP16: {
			fn:    makeSwap(16),
			valid: true,
		},
		PUSH1: {
			fn:    makePush(1, big.NewInt(1)),
			valid: true,
		},
		PUSH2: {
			fn:    makePush(2, big.NewInt(2)),
			valid: true,
		},
		PUSH3: {
			fn:    makePush(3, big.NewInt(3)),
			valid: true,
		},
		PUSH4: {
			fn:    makePush(4, big.NewInt(4)),
			valid: true,
		},
		PUSH5: {
			fn:    makePush(5, big.NewInt(5)),
			valid: true,
		},
		PUSH6: {
			fn:    makePush(6, big.NewInt(6)),
			valid: true,
		},
		PUSH7: {
			fn:    makePush(7, big.NewInt(7)),
			valid: true,
		},
		PUSH8: {
			fn:    makePush(8, big.NewInt(8)),
			valid: true,
		},
		PUSH9: {
			fn:    makePush(9, big.NewInt(9)),
			valid: true,
		},
		PUSH10: {
			fn:    makePush(10, big.NewInt(10)),
			valid: true,
		},
		PUSH11: {
			fn:    makePush(11, big.NewInt(11)),
			valid: true,
		},
		PUSH12: {
			fn:    makePush(12, big.NewInt(12)),
			valid: true,
		},
		PUSH13: {
			fn:    makePush(13, big.NewInt(13)),
			valid: true,
		},
		PUSH14: {
			fn:    makePush(14, big.NewInt(14)),
			valid: true,
		},
		PUSH15: {
			fn:    makePush(15, big.NewInt(15)),
			valid: true,
		},
		PUSH16: {
			fn:    makePush(16, big.NewInt(16)),
			valid: true,
		},
		PUSH17: {
			fn:    makePush(17, big.NewInt(17)),
			valid: true,
		},
		PUSH18: {
			fn:    makePush(18, big.NewInt(18)),
			valid: true,
		},
		PUSH19: {
			fn:    makePush(19, big.NewInt(19)),
			valid: true,
		},
		PUSH20: {
			fn:    makePush(20, big.NewInt(20)),
			valid: true,
		},
		PUSH21: {
			fn:    makePush(21, big.NewInt(21)),
			valid: true,
		},
		PUSH22: {
			fn:    makePush(22, big.NewInt(22)),
			valid: true,
		},
		PUSH23: {
			fn:    makePush(23, big.NewInt(23)),
			valid: true,
		},
		PUSH24: {
			fn:    makePush(24, big.NewInt(24)),
			valid: true,
		},
		PUSH25: {
			fn:    makePush(25, big.NewInt(25)),
			valid: true,
		},
		PUSH26: {
			fn:    makePush(26, big.NewInt(26)),
			valid: true,
		},
		PUSH27: {
			fn:    makePush(27, big.NewInt(27)),
			valid: true,
		},
		PUSH28: {
			fn:    makePush(28, big.NewInt(28)),
			valid: true,
		},
		PUSH29: {
			fn:    makePush(29, big.NewInt(29)),
			valid: true,
		},
		PUSH30: {
			fn:    makePush(30, big.NewInt(30)),
			valid: true,
		},
		PUSH31: {
			fn:    makePush(31, big.NewInt(31)),
			valid: true,
		},
		PUSH32: {
			fn:    makePush(32, big.NewInt(32)),
			valid: true,
		},
		DUP1: {
			fn:    makeDup(1),
			valid: true,
		},
		DUP2: {
			fn:    makeDup(2),
			valid: true,
		},
		DUP3: {
			fn:    makeDup(3),
			valid: true,
		},
		DUP4: {
			fn:    makeDup(4),
			valid: true,
		},
		DUP5: {
			fn:    makeDup(5),
			valid: true,
		},
		DUP6: {
			fn:    makeDup(6),
			valid: true,
		},
		DUP7: {
			fn:    makeDup(7),
			valid: true,
		},
		DUP8: {
			fn:    makeDup(8),
			valid: true,
		},
		DUP9: {
			fn:    makeDup(9),
			valid: true,
		},
		DUP10: {
			fn:    makeDup(10),
			valid: true,
		},
		DUP11: {
			fn:    makeDup(11),
			valid: true,
		},
		DUP12: {
			fn:    makeDup(12),
			valid: true,
		},
		DUP13: {
			fn:    makeDup(13),
			valid: true,
		},
		DUP14: {
			fn:    makeDup(14),
			valid: true,
		},
		DUP15: {
			fn:    makeDup(15),
			valid: true,
		},
		DUP16: {
			fn:    makeDup(16),
			valid: true,
		},
		RETURN: {
			fn:    opReturn,
			valid: true,
			halts: true,
		},
		SUICIDE: {
			fn:     opSuicide,
			valid:  true,
			halts:  true,
			writes: true,
		},
		JUMP: {
			fn:    opJump,
			valid: true,
			jumps: true,
		},
		JUMPI: {
			fn:    opJumpi,
			valid: true,
			jumps: true,
		},
		STOP: {
			fn:    opStop,
			valid: true,
			halts: true,
		},
	}
}
