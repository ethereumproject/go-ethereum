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
}

type vmJumpTable [256]jumpPtr

func newJumpTable(ruleset RuleSet, blockNumber *big.Int) vmJumpTable {
	var jumpTable vmJumpTable

	// when initialising a new VM execution we must first check the homestead
	// changes.
	if ruleset.IsHomestead(blockNumber) {
		jumpTable[DELEGATECALL] = jumpPtr{
			fn:      opDelegateCall,
			valid:   true,
			returns: true,
		}
	}
	if ruleset.IsECIP1045(blockNumber) {
		jumpTable[STATICCALL] = jumpPtr{
			fn:      opStaticCall,
			valid:   true,
			returns: true,
		}
		jumpTable[RETURNDATASIZE] = jumpPtr{
			fn:    opReturnDataSize,
			valid: true,
		}
		jumpTable[RETURNDATASIZE] = jumpPtr{
			fn:    opReturnDataSize,
			valid: true,
		}
		jumpTable[RETURNDATACOPY] = jumpPtr{
			// This is called manually during EVM.Run in order to do error handling in case return data size is out of bounds.
			// fn:    opReturnDataCopy,
			valid: true,
		}
		jumpTable[REVERT] = jumpPtr{
			// This is called manually during EVM.Run since implicity halt (akin to RETURN).
			fn:    nil,
			valid: true,
		}
	}

	jumpTable[ADD] = jumpPtr{
		fn:    opAdd,
		valid: true,
	}
	jumpTable[SUB] = jumpPtr{
		fn:    opSub,
		valid: true,
	}
	jumpTable[MUL] = jumpPtr{
		fn:    opMul,
		valid: true,
	}
	jumpTable[DIV] = jumpPtr{
		fn:    opDiv,
		valid: true,
	}
	jumpTable[SDIV] = jumpPtr{
		fn:    opSdiv,
		valid: true,
	}
	jumpTable[MOD] = jumpPtr{
		fn:    opMod,
		valid: true,
	}
	jumpTable[SMOD] = jumpPtr{
		fn:    opSmod,
		valid: true,
	}
	jumpTable[EXP] = jumpPtr{
		fn:    opExp,
		valid: true,
	}
	jumpTable[SIGNEXTEND] = jumpPtr{
		fn:    opSignExtend,
		valid: true,
	}
	jumpTable[NOT] = jumpPtr{
		fn:    opNot,
		valid: true,
	}
	jumpTable[LT] = jumpPtr{
		fn:    opLt,
		valid: true,
	}
	jumpTable[GT] = jumpPtr{
		fn:    opGt,
		valid: true,
	}
	jumpTable[SLT] = jumpPtr{
		fn:    opSlt,
		valid: true,
	}
	jumpTable[SGT] = jumpPtr{
		fn:    opSgt,
		valid: true,
	}
	jumpTable[EQ] = jumpPtr{
		fn:    opEq,
		valid: true,
	}
	jumpTable[ISZERO] = jumpPtr{
		fn:    opIszero,
		valid: true,
	}
	jumpTable[AND] = jumpPtr{
		fn:    opAnd,
		valid: true,
	}
	jumpTable[OR] = jumpPtr{
		fn:    opOr,
		valid: true,
	}
	jumpTable[XOR] = jumpPtr{
		fn:    opXor,
		valid: true,
	}
	jumpTable[BYTE] = jumpPtr{
		fn:    opByte,
		valid: true,
	}
	jumpTable[ADDMOD] = jumpPtr{
		fn:    opAddmod,
		valid: true,
	}
	jumpTable[MULMOD] = jumpPtr{
		fn:    opMulmod,
		valid: true,
	}
	jumpTable[SHA3] = jumpPtr{
		fn:    opSha3,
		valid: true,
	}
	jumpTable[ADDRESS] = jumpPtr{
		fn:    opAddress,
		valid: true,
	}
	jumpTable[BALANCE] = jumpPtr{
		fn:    opBalance,
		valid: true,
	}
	jumpTable[ORIGIN] = jumpPtr{
		fn:    opOrigin,
		valid: true,
	}
	jumpTable[CALLER] = jumpPtr{
		fn:    opCaller,
		valid: true,
	}
	jumpTable[CALLVALUE] = jumpPtr{
		fn:    opCallValue,
		valid: true,
	}
	jumpTable[CALLDATALOAD] = jumpPtr{
		fn:    opCalldataLoad,
		valid: true,
	}
	jumpTable[CALLDATASIZE] = jumpPtr{
		fn:    opCalldataSize,
		valid: true,
	}
	jumpTable[CALLDATACOPY] = jumpPtr{
		fn:    opCalldataCopy,
		valid: true,
	}
	jumpTable[CODESIZE] = jumpPtr{
		fn:    opCodeSize,
		valid: true,
	}
	jumpTable[EXTCODESIZE] = jumpPtr{
		fn:    opExtCodeSize,
		valid: true,
	}
	jumpTable[CODECOPY] = jumpPtr{
		fn:    opCodeCopy,
		valid: true,
	}
	jumpTable[EXTCODECOPY] = jumpPtr{
		fn:    opExtCodeCopy,
		valid: true,
	}
	jumpTable[GASPRICE] = jumpPtr{
		fn:    opGasprice,
		valid: true,
	}
	jumpTable[BLOCKHASH] = jumpPtr{
		fn:    opBlockhash,
		valid: true,
	}
	jumpTable[COINBASE] = jumpPtr{
		fn:    opCoinbase,
		valid: true,
	}
	jumpTable[TIMESTAMP] = jumpPtr{
		fn:    opTimestamp,
		valid: true,
	}
	jumpTable[NUMBER] = jumpPtr{
		fn:    opNumber,
		valid: true,
	}
	jumpTable[DIFFICULTY] = jumpPtr{
		fn:    opDifficulty,
		valid: true,
	}
	jumpTable[GASLIMIT] = jumpPtr{
		fn:    opGasLimit,
		valid: true,
	}
	jumpTable[POP] = jumpPtr{
		fn:    opPop,
		valid: true,
	}
	jumpTable[MLOAD] = jumpPtr{
		fn:    opMload,
		valid: true,
	}
	jumpTable[MSTORE] = jumpPtr{
		fn:    opMstore,
		valid: true,
	}
	jumpTable[MSTORE8] = jumpPtr{
		fn:    opMstore8,
		valid: true,
	}
	jumpTable[SLOAD] = jumpPtr{
		fn:    opSload,
		valid: true,
	}
	jumpTable[SSTORE] = jumpPtr{
		fn:    opSstore,
		valid: true,
	}
	jumpTable[JUMPDEST] = jumpPtr{
		fn:    opJumpdest,
		valid: true,
	}
	jumpTable[PC] = jumpPtr{
		fn:    nil,
		valid: true,
	}
	jumpTable[MSIZE] = jumpPtr{
		fn:    opMsize,
		valid: true,
	}
	jumpTable[GAS] = jumpPtr{
		fn:    opGas,
		valid: true,
	}
	jumpTable[CREATE] = jumpPtr{
		fn:    opCreate,
		valid: true,
	}
	jumpTable[CALL] = jumpPtr{
		fn:      opCall,
		valid:   true,
		returns: true,
	}
	jumpTable[CALLCODE] = jumpPtr{
		fn:      opCallCode,
		valid:   true,
		returns: true,
	}
	jumpTable[LOG0] = jumpPtr{
		fn:    makeLog(0),
		valid: true,
	}
	jumpTable[LOG1] = jumpPtr{
		fn:    makeLog(1),
		valid: true,
	}
	jumpTable[LOG2] = jumpPtr{
		fn:    makeLog(2),
		valid: true,
	}
	jumpTable[LOG3] = jumpPtr{
		fn:    makeLog(3),
		valid: true,
	}
	jumpTable[LOG4] = jumpPtr{
		fn:    makeLog(4),
		valid: true,
	}
	jumpTable[SWAP1] = jumpPtr{
		fn:    makeSwap(1),
		valid: true,
	}
	jumpTable[SWAP2] = jumpPtr{
		fn:    makeSwap(2),
		valid: true,
	}
	jumpTable[SWAP3] = jumpPtr{
		fn:    makeSwap(3),
		valid: true,
	}
	jumpTable[SWAP4] = jumpPtr{
		fn:    makeSwap(4),
		valid: true,
	}
	jumpTable[SWAP5] = jumpPtr{
		fn:    makeSwap(5),
		valid: true,
	}
	jumpTable[SWAP6] = jumpPtr{
		fn:    makeSwap(6),
		valid: true,
	}
	jumpTable[SWAP7] = jumpPtr{
		fn:    makeSwap(7),
		valid: true,
	}
	jumpTable[SWAP8] = jumpPtr{
		fn:    makeSwap(8),
		valid: true,
	}
	jumpTable[SWAP9] = jumpPtr{
		fn:    makeSwap(9),
		valid: true,
	}
	jumpTable[SWAP10] = jumpPtr{
		fn:    makeSwap(10),
		valid: true,
	}
	jumpTable[SWAP11] = jumpPtr{
		fn:    makeSwap(11),
		valid: true,
	}
	jumpTable[SWAP12] = jumpPtr{
		fn:    makeSwap(12),
		valid: true,
	}
	jumpTable[SWAP13] = jumpPtr{
		fn:    makeSwap(13),
		valid: true,
	}
	jumpTable[SWAP14] = jumpPtr{
		fn:    makeSwap(14),
		valid: true,
	}
	jumpTable[SWAP15] = jumpPtr{
		fn:    makeSwap(15),
		valid: true,
	}
	jumpTable[SWAP16] = jumpPtr{
		fn:    makeSwap(16),
		valid: true,
	}
	jumpTable[PUSH1] = jumpPtr{
		fn:    makePush(1, big.NewInt(1)),
		valid: true,
	}
	jumpTable[PUSH2] = jumpPtr{
		fn:    makePush(2, big.NewInt(2)),
		valid: true,
	}
	jumpTable[PUSH3] = jumpPtr{
		fn:    makePush(3, big.NewInt(3)),
		valid: true,
	}
	jumpTable[PUSH4] = jumpPtr{
		fn:    makePush(4, big.NewInt(4)),
		valid: true,
	}
	jumpTable[PUSH5] = jumpPtr{
		fn:    makePush(5, big.NewInt(5)),
		valid: true,
	}
	jumpTable[PUSH6] = jumpPtr{
		fn:    makePush(6, big.NewInt(6)),
		valid: true,
	}
	jumpTable[PUSH7] = jumpPtr{
		fn:    makePush(7, big.NewInt(7)),
		valid: true,
	}
	jumpTable[PUSH8] = jumpPtr{
		fn:    makePush(8, big.NewInt(8)),
		valid: true,
	}
	jumpTable[PUSH9] = jumpPtr{
		fn:    makePush(9, big.NewInt(9)),
		valid: true,
	}
	jumpTable[PUSH10] = jumpPtr{
		fn:    makePush(10, big.NewInt(10)),
		valid: true,
	}
	jumpTable[PUSH11] = jumpPtr{
		fn:    makePush(11, big.NewInt(11)),
		valid: true,
	}
	jumpTable[PUSH12] = jumpPtr{
		fn:    makePush(12, big.NewInt(12)),
		valid: true,
	}
	jumpTable[PUSH13] = jumpPtr{
		fn:    makePush(13, big.NewInt(13)),
		valid: true,
	}
	jumpTable[PUSH14] = jumpPtr{
		fn:    makePush(14, big.NewInt(14)),
		valid: true,
	}
	jumpTable[PUSH15] = jumpPtr{
		fn:    makePush(15, big.NewInt(15)),
		valid: true,
	}
	jumpTable[PUSH16] = jumpPtr{
		fn:    makePush(16, big.NewInt(16)),
		valid: true,
	}
	jumpTable[PUSH17] = jumpPtr{
		fn:    makePush(17, big.NewInt(17)),
		valid: true,
	}
	jumpTable[PUSH18] = jumpPtr{
		fn:    makePush(18, big.NewInt(18)),
		valid: true,
	}
	jumpTable[PUSH19] = jumpPtr{
		fn:    makePush(19, big.NewInt(19)),
		valid: true,
	}
	jumpTable[PUSH20] = jumpPtr{
		fn:    makePush(20, big.NewInt(20)),
		valid: true,
	}
	jumpTable[PUSH21] = jumpPtr{
		fn:    makePush(21, big.NewInt(21)),
		valid: true,
	}
	jumpTable[PUSH22] = jumpPtr{
		fn:    makePush(22, big.NewInt(22)),
		valid: true,
	}
	jumpTable[PUSH23] = jumpPtr{
		fn:    makePush(23, big.NewInt(23)),
		valid: true,
	}
	jumpTable[PUSH24] = jumpPtr{
		fn:    makePush(24, big.NewInt(24)),
		valid: true,
	}
	jumpTable[PUSH25] = jumpPtr{
		fn:    makePush(25, big.NewInt(25)),
		valid: true,
	}
	jumpTable[PUSH26] = jumpPtr{
		fn:    makePush(26, big.NewInt(26)),
		valid: true,
	}
	jumpTable[PUSH27] = jumpPtr{
		fn:    makePush(27, big.NewInt(27)),
		valid: true,
	}
	jumpTable[PUSH28] = jumpPtr{
		fn:    makePush(28, big.NewInt(28)),
		valid: true,
	}
	jumpTable[PUSH29] = jumpPtr{
		fn:    makePush(29, big.NewInt(29)),
		valid: true,
	}
	jumpTable[PUSH30] = jumpPtr{
		fn:    makePush(30, big.NewInt(30)),
		valid: true,
	}
	jumpTable[PUSH31] = jumpPtr{
		fn:    makePush(31, big.NewInt(31)),
		valid: true,
	}
	jumpTable[PUSH32] = jumpPtr{
		fn:    makePush(32, big.NewInt(32)),
		valid: true,
	}
	jumpTable[DUP1] = jumpPtr{
		fn:    makeDup(1),
		valid: true,
	}
	jumpTable[DUP2] = jumpPtr{
		fn:    makeDup(2),
		valid: true,
	}
	jumpTable[DUP3] = jumpPtr{
		fn:    makeDup(3),
		valid: true,
	}
	jumpTable[DUP4] = jumpPtr{
		fn:    makeDup(4),
		valid: true,
	}
	jumpTable[DUP5] = jumpPtr{
		fn:    makeDup(5),
		valid: true,
	}
	jumpTable[DUP6] = jumpPtr{
		fn:    makeDup(6),
		valid: true,
	}
	jumpTable[DUP7] = jumpPtr{
		fn:    makeDup(7),
		valid: true,
	}
	jumpTable[DUP8] = jumpPtr{
		fn:    makeDup(8),
		valid: true,
	}
	jumpTable[DUP9] = jumpPtr{
		fn:    makeDup(9),
		valid: true,
	}
	jumpTable[DUP10] = jumpPtr{
		fn:    makeDup(10),
		valid: true,
	}
	jumpTable[DUP11] = jumpPtr{
		fn:    makeDup(11),
		valid: true,
	}
	jumpTable[DUP12] = jumpPtr{
		fn:    makeDup(12),
		valid: true,
	}
	jumpTable[DUP13] = jumpPtr{
		fn:    makeDup(13),
		valid: true,
	}
	jumpTable[DUP14] = jumpPtr{
		fn:    makeDup(14),
		valid: true,
	}
	jumpTable[DUP15] = jumpPtr{
		fn:    makeDup(15),
		valid: true,
	}
	jumpTable[DUP16] = jumpPtr{
		fn:    makeDup(16),
		valid: true,
	}

	jumpTable[RETURN] = jumpPtr{
		fn:    nil,
		valid: true,
	}
	jumpTable[SUICIDE] = jumpPtr{
		fn:    nil,
		valid: true,
	}
	jumpTable[JUMP] = jumpPtr{
		fn:    nil,
		valid: true,
	}
	jumpTable[JUMPI] = jumpPtr{
		fn:    nil,
		valid: true,
	}
	jumpTable[STOP] = jumpPtr{
		fn:    nil,
		valid: true,
	}

	return jumpTable
}
