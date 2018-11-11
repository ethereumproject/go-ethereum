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

package core

import (
	"fmt"
	"math/big"

	"github.com/ethereumproject/go-ethereum/consensus"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/params"
)

var (
	DurationLimit          = big.NewInt(13) // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.
	ExpDiffPeriod          = big.NewInt(100000)
	MinimumDifficulty      = big.NewInt(131072)
	MinGasLimit            = big.NewInt(5000)    // Minimum the gas limit may ever be.
	TargetGasLimit         = big.NewInt(4712388) // The artificial target
	DifficultyBoundDivisor = big.NewInt(2048)    // The bound divisor of the difficulty, used in the update calculations.
	GasLimitBoundDivisor   = big.NewInt(1024)    // The bound divisor of the gas limit, used in update calculations.
)

var (
	big10      = big.NewInt(10)
	bigMinus99 = big.NewInt(-99)
)

// Difficulty allows passing configurable options to a given difficulty algorithm.
type DifficultyConfig struct {
	Name    string                 `json:"name"`
	Options map[string]interface{} `json:"options"`
}

// BlockValidator is responsible for validating block headers, uncles and
// processed state.
//
// BlockValidator implements Validator.
type BlockValidator struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine
	// Pow    pow.PoW // Proof of work used for validating
}

// NewBlockValidator returns a new block validator which is safe for re-use
func NewBlockValidator(config *params.ChainConfig, blockchain *BlockChain, engine consensus.Engine) *BlockValidator {
	validator := &BlockValidator{
		config: config,
		engine: engine,
		bc:     blockchain,
	}
	return validator
}

// ValidateBody validates the given block's uncles and verifies the the block
// header's transaction and uncle roots. The headers are assumed to be already
// validated at this point.
func (v *BlockValidator) ValidateBody(block *types.Block) error {
	// Check whether the block's known, and if not, that it's linkable
	if v.bc.HasBlockAndState(block.Hash()) {
		return ErrKnownBlock
	}
	if !v.bc.HasBlockAndState(block.ParentHash()) {
		if !v.bc.HasBlock(block.ParentHash()) {
			return consensus.ErrUnknownAncestor
		}
		return consensus.ErrPrunedAncestor
	}
	// Header validity is known at this point, check the uncles and transactions
	header := block.Header()
	if err := v.engine.VerifyUncles(v.bc, block); err != nil {
		return err
	}
	if hash := types.CalcUncleHash(block.Uncles()); hash != header.UncleHash {
		return fmt.Errorf("uncle root hash mismatch: have %x, want %x", hash, header.UncleHash)
	}
	if hash := types.DeriveSha(block.Transactions()); hash != header.TxHash {
		return fmt.Errorf("transaction root hash mismatch: have %x, want %x", hash, header.TxHash)
	}
	return nil
}

// // ValidateBlock validates the given block's header and uncles and verifies the
// // the block header's transaction and uncle roots.
// //
// // ValidateBlock does not validate the header's pow. The pow work validated
// // separately so we can process them in parallel.
// //
// // ValidateBlock also validates and makes sure that any previous state (or present)
// // state that might or might not be present is checked to make sure that fast
// // sync has done it's job proper. This prevents the block validator form accepting
// // false positives where a header is present but the state is not.
// func (v *BlockValidator) ValidateBlock(block *types.Block) error {
// 	if v.bc.HasBlock(block.Hash(), block.NumberU64()) {
// 		if _, err := state.New(block.Root(), state.NewDatabase(v.bc.chainDb)); err == nil {
// 			return &KnownBlockError{block.Number(), block.Hash()}
// 		}
// 	}
// 	parent := v.bc.GetBlock(block.ParentHash())
// 	if parent == nil {
// 		return ParentError(block.ParentHash())
// 	}
// 	if _, err := state.New(parent.Root(), state.NewDatabase(v.bc.chainDb)); err != nil {
// 		return ParentError(block.ParentHash())
// 	}

// 	header := block.Header()
// 	// validate the block header
// 	if err := ValidateHeader(v.config, v.Pow, header, parent.Header(), false, false); err != nil {
// 		return err
// 	}
// 	// verify the uncles are correctly rewarded
// 	if err := v.VerifyUncles(block, parent); err != nil {
// 		return err
// 	}

// 	// Verify UncleHash before running other uncle validations
// 	unclesSha := types.CalcUncleHash(block.Uncles())
// 	if unclesSha != header.UncleHash {
// 		return fmt.Errorf("invalid uncles root hash. received=%x calculated=%x", header.UncleHash, unclesSha)
// 	}

// 	// The transactions Trie's root (R = (Tr [[i, RLP(T1)], [i, RLP(T2)], ... [n, RLP(Tn)]]))
// 	// can be used by light clients to make sure they've received the correct Txs
// 	txSha := types.DeriveSha(block.Transactions())
// 	if txSha != header.TxHash {
// 		return fmt.Errorf("invalid transaction root hash. received=%x calculated=%x", header.TxHash, txSha)
// 	}

// 	return nil
// }

// ValidateState validates the various changes that happen after a state
// transition, such as amount of used gas, the receipt roots and the state root
// itself. ValidateState returns a database batch if the validation was a success
// otherwise nil and an error is returned.
func (v *BlockValidator) ValidateState(block, parent *types.Block, statedb *state.StateDB, receipts types.Receipts, usedGas uint64) (err error) {
	header := block.Header()
	if block.GasUsed() != usedGas {
		return validateError(fmt.Sprintf("gas used error (%v / %v)", block.GasUsed(), usedGas))
	}
	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if rbloom != header.Bloom {
		return fmt.Errorf("unable to replicate block's bloom=%x vs calculated bloom=%x", header.Bloom, rbloom)
	}
	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, R1]]))
	receiptSha := types.DeriveSha(receipts)
	if receiptSha != header.ReceiptHash {
		return fmt.Errorf("invalid receipt root hash. received=%x calculated=%x", header.ReceiptHash, receiptSha)
	}
	// Validate the state root against the received state root and throw
	// an error if they don't match.
	if root := statedb.IntermediateRoot(false); header.Root != root {
		return fmt.Errorf("invalid merkle root: header=%x computed=%x", header.Root, root)
	}
	return nil
}

// CalcGasLimit computes the gas limit of the next block after parent.
// This is miner strategy, not consensus protocol.
func CalcGasLimit(parent *types.Block) uint64 {
	// contrib := (parent.GasUsed() * 3 / 2) / params.GasLimitBoundDivisor // 1024
	contrib := (parent.GasUsed() + parent.GasUsed()/2) / params.GasLimitBoundDivisor

	// decay := (parent.GasLimit() / params.GasLimitBoundDivisor) - 1
	decay := parent.GasLimit()/params.GasLimitBoundDivisor - 1

	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	limit := parent.GasLimit() - decay + contrib
	if limit < params.MinGasLimit {
		limit = params.MinGasLimit
	}
	// however, if we're now below the target (TargetGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if limit < params.TargetGasLimit {
		limit = parent.GasLimit() + decay
		if limit > params.TargetGasLimit {
			limit = params.TargetGasLimit
		}
	}
	return limit
}

// // VerifyUncles verifies the given block's uncles and applies the Ethereum
// // consensus rules to the various block headers included; it will return an
// // error if any of the included uncle headers were invalid. It returns an error
// // if the validation failed.
// func (v *BlockValidator) VerifyUncles(block, parent *types.Block) error {
// 	// validate that there at most 2 uncles included in this block
// 	if len(block.Uncles()) > 2 {
// 		return validateError(fmt.Sprintf("Block can only contain maximum 2 uncles (contained %d)", len(block.Uncles())))
// 	}

// 	uncles := set.New()
// 	ancestors := make(map[common.Hash]*types.Block)
// 	for _, ancestor := range v.bc.GetBlocksFromHash(block.ParentHash(), 7) {
// 		ancestors[ancestor.Hash()] = ancestor
// 		// Include ancestors uncles in the uncle set. Uncles must be unique.
// 		for _, uncle := range ancestor.Uncles() {
// 			uncles.Add(uncle.Hash())
// 		}
// 	}
// 	ancestors[block.Hash()] = block
// 	uncles.Add(block.Hash())

// 	for i, uncle := range block.Uncles() {
// 		hash := uncle.Hash()
// 		if uncles.Has(hash) {
// 			// Error not unique
// 			return UncleError("uncle[%d](%x) not unique", i, hash[:4])
// 		}
// 		uncles.Add(hash)

// 		if ancestors[hash] != nil {
// 			branch := fmt.Sprintf("  O - %x\n  |\n", block.Hash())
// 			for h := range ancestors {
// 				branch += fmt.Sprintf("  O - %x\n  |\n", h)
// 			}
// 			glog.Infoln(branch)
// 			return UncleError("uncle[%d](%x) is ancestor", i, hash[:4])
// 		}

// 		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == parent.Hash() {
// 			return UncleError("uncle[%d](%x)'s parent is not ancestor (%x)", i, hash[:4], uncle.ParentHash[0:4])
// 		}

// 		if err := ValidateHeader(v.config, v.Pow, uncle, ancestors[uncle.ParentHash].Header(), true, true); err != nil {
// 			return validateError(fmt.Sprintf("uncle[%d](%x) header invalid: %v", i, hash[:4], err))
// 		}
// 	}

// 	return nil
// }

// // ValidateHeader validates the given header and, depending on the pow arg,
// // checks the proof of work of the given header. Returns an error if the
// // validation failed.
// func (v *BlockValidator) ValidateHeader(header, parent *types.Header, checkPow bool) error {
// 	// Short circuit if the parent is missing.
// 	if parent == nil {
// 		return ParentError(header.ParentHash)
// 	}
// 	// Short circuit if the header's already known or its parent missing
// 	if v.bc.HasHeader(header.Hash()) {
// 		return nil
// 	}
// 	return ValidateHeader(v.config, v.Pow, header, parent, checkPow, false)
// }

// // Validates a header. Returns an error if the header is invalid.
// //
// // See YP section 4.3.4. "Block Header Validity"
// func ValidateHeader(config *ChainConfig, pow pow.PoW, header *types.Header, parent *types.Header, checkPow, uncle bool) error {
// 	if len(header.Extra) > types.HeaderExtraMax {
// 		return fmt.Errorf("extra data size %d exceeds limit of %d", len(header.Extra), types.HeaderExtraMax)
// 	}

// 	if uncle {
// 		if header.Time.Cmp(common.MaxBig) == 1 {
// 			return BlockTSTooBigErr
// 		}
// 	} else {
// 		if header.Time.Cmp(big.NewInt(time.Now().Unix())) == 1 {
// 			return BlockFutureErr
// 		}
// 	}
// 	if header.Time.Cmp(parent.Time) != 1 {
// 		return BlockEqualTSErr
// 	}

// 	expd := CalcDifficulty(config, header.Time.Uint64(), parent.Time.Uint64(), parent.Number, parent.Difficulty)
// 	if expd.Cmp(header.Difficulty) != 0 {
// 		return fmt.Errorf("Difficulty check failed for header %v != %v at %v", header.Difficulty, expd, header.Number)
// 	}

// 	a := new(big.Int).Set(parent.GasLimit)
// 	a = a.Sub(a, header.GasLimit)
// 	a.Abs(a)
// 	b := new(big.Int).Set(parent.GasLimit)
// 	b = b.Div(b, GasLimitBoundDivisor)
// 	if !(a.Cmp(b) < 0) || (header.GasLimit.Cmp(MinGasLimit) == -1) {
// 		return fmt.Errorf("GasLimit check failed for header %v (%v > %v)", header.GasLimit, a, b)
// 	}

// 	num := new(big.Int).Set(parent.Number)
// 	num.Sub(header.Number, num)
// 	if num.Cmp(big.NewInt(1)) != 0 {
// 		return BlockNumberErr
// 	}

// 	if checkPow {
// 		// Verify the nonce of the header. Return an error if it's not valid
// 		if !pow.Verify(types.NewBlockWithHeader(header)) {
// 			return &BlockNonceErr{header.Number, header.Hash(), header.Nonce.Uint64()}
// 		}
// 	}
// 	// If all checks passed, validate the extra-data field for hard forks
// 	return nil
// }
