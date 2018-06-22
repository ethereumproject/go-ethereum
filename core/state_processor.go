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
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/params"
	"github.com/ethereumproject/go-ethereum/rlp"
)

var (
	MaximumBlockReward       = big.NewInt(5e+18) // that's shiny 5 ether
	big8                     = big.NewInt(8)
	big32                    = big.NewInt(32)
	DisinflationRateQuotient = big.NewInt(4)
	DisinflationRateDivisor  = big.NewInt(5)

	ErrConfiguration = errors.New("invalid configuration")
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig
	bc     *BlockChain
	// engine consensus.Engine
}

// NewStateProcessor initialises a new StateProcessor.
// func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		// engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		err      error
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())

		cfg = vm.Config{}
	)
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		// PTAL these seem like "sugary" "pre-flight" safety checks and warnings that are actually redundant to the VM processing
		// if tx.Protected() {
		// 	chainId := p.config.GetChainID()
		// 	if chainId.Cmp(new(big.Int)) == 0 {
		// 		return nil, nil, nil, fmt.Errorf("chainID is not set for EIP-155 in chain configuration at block number: %v. \n  Tx ChainID: %v", block.Number(), tx.ChainId())
		// 	}
		// 	if tx.ChainId() == nil || tx.ChainId().Cmp(chainId) != 0 {
		// 		return nil, nil, nil, fmt.Errorf("invalid transaction chain id. Current chain id: %v tx chain id: %v", p.config.GetChainID(), tx.ChainId())
		// 	}
		// }
		// statedb.StartRecord(tx.Hash(), block.Hash(), i)
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		if !UseSputnikVM {
			// (config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config)
			receipt, _, err := ApplyTransaction(p.config, p.bc, nil, gp, statedb, header, tx, usedGas, cfg)
			if err != nil {
				return nil, nil, 0, err
			}
			receipts = append(receipts, receipt)
			allLogs = append(allLogs, receipt.Logs...)
			continue
		}
		receipt, _, err := ApplyMultiVmTransaction(p.config, p.bc, gp, statedb, header, tx, usedGas)
		if err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	AccumulateRewards(p.config, statedb, header, block.Uncles())

	if *usedGas == 0 && block.Transactions().Len() > 0 {
		panic("zero gas processor")
	}

	return receipts, allLogs, *usedGas, err
}

func (p *StateProcessor) ReplayTransaction(txHash common.Hash, statedb *state.StateDB, cfg vm.Config) (*types.Receipt, error) {
	statedb = statedb.Copy()

	blockHash, _, index, err := getTransactionBlockData(p.bc.chainDb, txHash)
	if err != nil {
		return nil, err
	}

	block := GetBlock(p.bc.chainDb, blockHash)
	tx := block.Transactions()[index]

	var (
		usedGas = new(uint64)
		header  = block.Header()
		gp      = new(GasPool).AddGas(block.GasLimit())
	)

	// PTAL again, as above
	// if tx.Protected() {
	// 	chainId := p.config.GetChainID()
	// 	if chainId.Cmp(new(big.Int)) == 0 {
	// 		return nil, fmt.Errorf("ChainID is not set for EIP-155 in chain configuration at block number: %v. \n  Tx ChainID: %v", block.Number(), tx.ChainId())
	// 	}
	// 	if tx.ChainId() == nil || tx.ChainId().Cmp(chainId) != 0 {
	// 		return nil, fmt.Errorf("Invalid transaction chain id. Current chain id: %v tx chain id: %v", p.config.GetChainID(), tx.ChainId())
	// 	}
	// }
	//statedb.StartRecord(tx.Hash(), block.Hash(), i)
	if !UseSputnikVM {
		receipt, _, err := ApplyTransaction(p.config, p.bc, nil, gp, statedb, header, tx, usedGas, cfg)
		if err != nil {
			return nil, err
		}
		return receipt, nil
	}
	receipt, _, err := ApplyMultiVmTransaction(p.config, p.bc, gp, statedb, header, tx, usedGas)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// TODO(tzdybal) - refactor (duplicate from eth/api.go)
// getTransactionBlockData fetches the meta data for the given transaction from the chain database. This is useful to
// retrieve block information for a hash. It returns the block hash, block index and transaction index.
func getTransactionBlockData(chainDb ethdb.Database, txHash common.Hash) (common.Hash, uint64, uint64, error) {
	var txBlock struct {
		BlockHash  common.Hash
		BlockIndex uint64
		Index      uint64
	}

	blockData, err := chainDb.Get(append(txHash.Bytes(), 0x0001))
	if err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	reader := bytes.NewReader(blockData)
	if err = rlp.Decode(reader, &txBlock); err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	return txBlock.BlockHash, txBlock.BlockIndex, txBlock.Index, nil
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error) {
	msg, err := tx.AsMessage(config.GetSigner(header.Number))
	if err != nil {
		return nil, 0, err
	}
	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, header, bc, author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, statedb, config, cfg)
	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err := ApplyMessage(vmenv, msg, gp)
	if err != nil {
		return nil, 0, err
	}
	if gas == 0 {
		panic("zero gas apply message")
	}
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += gas

	if *usedGas == 0 {
		panic("used gas zero apply tx")
	}

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root, failed, big.NewInt(0).SetUint64(*usedGas))
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = new(big.Int).SetUint64(gas)
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return receipt, gas, err
}

// AccumulateRewards credits the coinbase of the given block with the
// mining reward. The total reward consists of the static block reward
// and rewards for included uncles. The coinbase of each uncle block is
// also rewarded.
func AccumulateRewards(config *params.ChainConfig, statedb *state.StateDB, header *types.Header, uncles []*types.Header) {

	// An uncle is a block that would be considered an orphan because its not on the longest chain (it's an alternative block at the same height as your parent).
	// https://www.reddit.com/r/ethereum/comments/3c9jbf/wtf_are_uncles_and_why_do_they_matter/

	// uncle.Number = 2,535,998 // assuming "latest" uncle...
	// block.Number = 2,534,999 // uncles can be at same height as each other
	// ... as uncles get older (within validation; <=n-7), reward drops

	// Since ECIP1017 impacts "Era 1" idempotently and with constant 0-block based eras,
	// we don't care about where the block/fork implementing it is.
	feat, _, configured := config.HasFeature("reward")
	if !configured {
		reward := new(big.Int).Set(MaximumBlockReward)
		r := new(big.Int)

		for _, uncle := range uncles {
			r.Add(uncle.Number, big8)    // 2,534,998 + 8              = 2,535,006
			r.Sub(r, header.Number)      // 2,535,006 - 2,534,999        = 7
			r.Mul(r, MaximumBlockReward) // 7 * 5e+18               = 35e+18
			r.Div(r, big8)               // 35e+18 / 8                            = 7/8 * 5e+18

			statedb.AddBalance(uncle.Coinbase, r) // $$

			r.Div(MaximumBlockReward, big32) // 5e+18 / 32
			reward.Add(reward, r)            // 5e+18 + (1/32*5e+18)
		}
		statedb.AddBalance(header.Coinbase, reward) //  $$ => 5e+18 + (1/32*5e+18)
	} else {
		// Check that configuration specifies ECIP1017.
		val, ok := feat.GetString("type")
		if !ok || val != "ecip1017" {
			panic(ErrConfiguration)
		}

		// Ensure value 'era' is configured.
		eraLen, ok := feat.GetBigInt("era")
		if !ok || eraLen.Cmp(big.NewInt(0)) <= 0 {
			panic(ErrConfiguration)
		}

		era := GetBlockEra(header.Number, eraLen)

		wr := GetBlockWinnerRewardByEra(era) // wr "winner reward". 5, 4, 3.2, 2.56, ...

		wurs := GetBlockWinnerRewardForUnclesByEra(era, uncles) // wurs "winner uncle rewards"
		wr.Add(wr, wurs)

		statedb.AddBalance(header.Coinbase, wr) // $$

		// Reward uncle miners.
		for _, uncle := range uncles {
			ur := GetBlockUncleRewardByEra(era, header, uncle)
			statedb.AddBalance(uncle.Coinbase, ur) // $$
		}
	}
}

// As of "Era 2" (zero-index era 1), uncle miners and winners are rewarded equally for each included block.
// So they share this function.
func getEraUncleBlockReward(era *big.Int) *big.Int {
	return new(big.Int).Div(GetBlockWinnerRewardByEra(era), big32)
}

// GetBlockUncleRewardByEra gets called _for each uncle miner_ associated with a winner block's uncles.
func GetBlockUncleRewardByEra(era *big.Int, header, uncle *types.Header) *big.Int {
	// Era 1 (index 0):
	//   An extra reward to the winning miner for including uncles as part of the block, in the form of an extra 1/32 (0.15625ETC) per uncle included, up to a maximum of two (2) uncles.
	if era.Cmp(big.NewInt(0)) == 0 {
		r := new(big.Int)
		r.Add(uncle.Number, big8)    // 2,534,998 + 8              = 2,535,006
		r.Sub(r, header.Number)      // 2,535,006 - 2,534,999        = 7
		r.Mul(r, MaximumBlockReward) // 7 * 5e+18               = 35e+18
		r.Div(r, big8)               // 35e+18 / 8                            = 7/8 * 5e+18

		return r
	}
	return getEraUncleBlockReward(era)
}

// GetBlockWinnerRewardForUnclesByEra gets called _per winner_, and accumulates rewards for each included uncle.
// Assumes uncles have been validated and limited (@ func (v *BlockValidator) VerifyUncles).
func GetBlockWinnerRewardForUnclesByEra(era *big.Int, uncles []*types.Header) *big.Int {
	r := big.NewInt(0)

	for range uncles {
		r.Add(r, getEraUncleBlockReward(era)) // can reuse this, since 1/32 for winner's uncles remain unchanged from "Era 1"
	}
	return r
}

// GetRewardByEra gets a block reward at disinflation rate.
// Constants MaxBlockReward, DisinflationRateQuotient, and DisinflationRateDivisor assumed.
func GetBlockWinnerRewardByEra(era *big.Int) *big.Int {
	if era.Cmp(big.NewInt(0)) == 0 {
		return new(big.Int).Set(MaximumBlockReward)
	}

	// MaxBlockReward _r_ * (4/5)**era == MaxBlockReward * (4**era) / (5**era)
	// since (q/d)**n == q**n / d**n
	// qed
	var q, d, r *big.Int = new(big.Int), new(big.Int), new(big.Int)

	q.Exp(DisinflationRateQuotient, era, nil)
	d.Exp(DisinflationRateDivisor, era, nil)

	r.Mul(MaximumBlockReward, q)
	r.Div(r, d)

	return r
}

// GetBlockEra gets which "Era" a given block is within, given an era length (ecip-1017 has era=5,000,000 blocks)
// Returns a zero-index era number, so "Era 1": 0, "Era 2": 1, "Era 3": 2 ...
func GetBlockEra(blockNum, eraLength *big.Int) *big.Int {
	// If genesis block or impossible negative-numbered block, return zero-val.
	if blockNum.Sign() < 1 {
		return new(big.Int)
	}

	remainder := big.NewInt(0).Mod(big.NewInt(0).Sub(blockNum, big.NewInt(1)), eraLength)
	base := big.NewInt(0).Sub(blockNum, remainder)

	d := big.NewInt(0).Div(base, eraLength)
	dremainder := big.NewInt(0).Mod(d, big.NewInt(1))

	return new(big.Int).Sub(d, dremainder)
}
