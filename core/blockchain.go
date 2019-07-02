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

// Package core implements the Ethereum consensus protocol.
package core

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	mrand "math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"reflect"
	"strconv"

	"encoding/binary"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/core/state"
	"github.com/eth-classic/go-ethereum/core/types"
	"github.com/eth-classic/go-ethereum/core/vm"
	"github.com/eth-classic/go-ethereum/crypto"
	"github.com/eth-classic/go-ethereum/ethdb"
	"github.com/eth-classic/go-ethereum/event"
	"github.com/eth-classic/go-ethereum/logger"
	"github.com/eth-classic/go-ethereum/logger/glog"
	"github.com/eth-classic/go-ethereum/pow"
	"github.com/eth-classic/go-ethereum/rlp"
	"github.com/eth-classic/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
)

var (
	ErrNoGenesis = errors.New("Genesis not found in chain")
	errNilBlock  = errors.New("nil block")
	errNilHeader = errors.New("nil header")
)

const (
	headerCacheLimit    = 512
	bodyCacheLimit      = 256
	tdCacheLimit        = 1024
	blockCacheLimit     = 256
	maxFutureBlocks     = 256
	maxTimeFutureBlocks = 30
	// must be bumped when consensus algorithm is changed, this forces the upgradedb
	// command to be run (forces the blocks to be imported again using the new algorithm)
	BlockChainVersion = 3
)

// BlockChain represents the canonical chain given a database with a genesis
// block. The Blockchain manages chain imports, reverts, chain reorganisations.
//
// Importing blocks in to the block chain happens according to the set of rules
// defined by the two stage Validator. Processing of blocks is done using the
// Processor which processes the included transaction. The validation of the state
// is done in the second part of the Validator. Failing results in aborting of
// the import.
//
// The BlockChain also helps in returning blocks from **any** chain included
// in the database as well as blocks that represents the canonical chain. It's
// important to note that GetBlock can return any block and does not need to be
// included in the canonical one where as GetBlockByNumber always represents the
// canonical chain.
type BlockChain struct {
	config *ChainConfig // chain & network configuration

	hc           *HeaderChain
	chainDb      ethdb.Database
	eventMux     *event.TypeMux
	genesisBlock *types.Block

	mu      sync.RWMutex // global mutex for locking chain operations
	chainmu sync.RWMutex // blockchain insertion lock
	procmu  sync.RWMutex // block processor lock

	currentBlock     *types.Block // Current head of the block chain
	currentFastBlock *types.Block // Current head of the fast-sync chain (may be above the block chain!)

	stateCache   *state.StateDB // State database to reuse between imports (contains state cache)
	bodyCache    *lru.Cache     // Cache for the most recent block bodies
	bodyRLPCache *lru.Cache     // Cache for the most recent block bodies in RLP encoded format
	blockCache   *lru.Cache     // Cache for the most recent entire blocks
	futureBlocks *lru.Cache     // future blocks are blocks added for later processing

	quit    chan struct{} // blockchain quit channel
	running int32         // running must be called atomically
	// procInterrupt must be atomically called
	procInterrupt int32          // interrupt signaler for block processing
	wg            sync.WaitGroup // chain processing wait group for shutting down

	pow       pow.PoW
	processor Processor // block processor interface
	validator Validator // block and state validator interface

	atxi *AtxiT
}

type ChainInsertResult struct {
	ChainInsertEvent
	Index int
	Error error
}

type ReceiptChainInsertResult struct {
	ReceiptChainInsertEvent
	Index int
	Error error
}

type HeaderChainInsertResult struct {
	HeaderChainInsertEvent
	Index int
	Error error
}

func (bc *BlockChain) GetHeaderByHash(h common.Hash) *types.Header {
	return bc.hc.GetHeader(h)
}

func (bc *BlockChain) GetBlockByHash(h common.Hash) *types.Block {
	return bc.GetBlock(h)
}

// NewBlockChain returns a fully initialised block chain using information
// available in the database. It initialises the default Ethereum Validator and
// Processor.
func NewBlockChain(chainDb ethdb.Database, config *ChainConfig, pow pow.PoW, mux *event.TypeMux) (*BlockChain, error) {
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	futureBlocks, _ := lru.New(maxFutureBlocks)

	bc := &BlockChain{
		config:       config,
		chainDb:      chainDb,
		eventMux:     mux,
		quit:         make(chan struct{}),
		bodyCache:    bodyCache,
		bodyRLPCache: bodyRLPCache,
		blockCache:   blockCache,
		futureBlocks: futureBlocks,
		pow:          pow,
	}
	bc.SetValidator(NewBlockValidator(config, bc, pow))
	bc.SetProcessor(NewStateProcessor(config, bc))

	gv := func() HeaderValidator { return bc.Validator() }
	var err error
	bc.hc, err = NewHeaderChain(chainDb, config, mux, gv, bc.getProcInterrupt)
	if err != nil {
		return nil, err
	}

	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}

	if err := bc.LoadLastState(false); err != nil {
		return nil, err
	}
	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for i := range config.BadHashes {
		if header := bc.GetHeader(config.BadHashes[i].Hash); header != nil && header.Number.Cmp(config.BadHashes[i].Block) == 0 {
			glog.V(logger.Error).Infof("Found bad hash, rewinding chain to block #%d [%s]", header.Number, header.ParentHash.Hex())
			bc.SetHead(header.Number.Uint64() - 1)
			glog.V(logger.Error).Infoln("Chain rewind was successful, resuming normal operation")
		}
	}
	// Take ownership of this particular state
	go bc.update()
	return bc, nil
}

func NewBlockChainDryrun(chainDb ethdb.Database, config *ChainConfig, pow pow.PoW, mux *event.TypeMux) (*BlockChain, error) {
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	futureBlocks, _ := lru.New(maxFutureBlocks)

	bc := &BlockChain{
		config:       config,
		chainDb:      chainDb,
		eventMux:     mux,
		quit:         make(chan struct{}),
		bodyCache:    bodyCache,
		bodyRLPCache: bodyRLPCache,
		blockCache:   blockCache,
		futureBlocks: futureBlocks,
		pow:          pow,
	}
	bc.SetValidator(NewBlockValidator(config, bc, pow))
	bc.SetProcessor(NewStateProcessor(config, bc))

	gv := func() HeaderValidator { return bc.Validator() }
	var err error
	bc.hc, err = NewHeaderChain(chainDb, config, mux, gv, bc.getProcInterrupt)
	if err != nil {
		return nil, err
	}

	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}

	//if err := bc.loadLastState(); err != nil {
	//	return nil, err
	//}
	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for i := range config.BadHashes {
		if header := bc.GetHeader(config.BadHashes[i].Hash); header != nil && header.Number.Cmp(config.BadHashes[i].Block) == 0 {
			glog.V(logger.Error).Infof("Found bad hash, rewinding chain to block #%d [%s]", header.Number, header.ParentHash.Hex())
			bc.SetHead(header.Number.Uint64() - 1)
			glog.V(logger.Error).Infoln("Chain rewind was successful, resuming normal operation")
		}
	}
	// // Take ownership of this particular state
	//go bc.update()
	return bc, nil
}

// GetEventMux returns the blockchain's event mux
func (bc *BlockChain) GetEventMux() *event.TypeMux {
	return bc.eventMux
}

// SetAtxi sets the db and in-use var for atx indexing.
func (bc *BlockChain) SetAtxi(a *AtxiT) {
	bc.atxi = a
}

// GetAtxi return indexes db and if atx index in use.
func (bc *BlockChain) GetAtxi() *AtxiT {
	return bc.atxi
}

func (bc *BlockChain) getProcInterrupt() bool {
	return atomic.LoadInt32(&bc.procInterrupt) == 1
}

func (bc *BlockChain) blockIsGenesis(b *types.Block) bool {
	if bc.Genesis() != nil {
		return reflect.DeepEqual(b, bc.Genesis())
	}
	ht, _ := DefaultConfigMorden.Genesis.Header()
	hm, _ := DefaultConfigMainnet.Genesis.Header()
	return b.Hash() == ht.Hash() || b.Hash() == hm.Hash()

}

// blockIsInvalid sanity checks for a block's health.
func (bc *BlockChain) blockIsInvalid(b *types.Block) error {

	type testCases struct {
		explanation string
		test        func(b *types.Block) error
	}

	blockAgnosticCases := []testCases{
		{
			"block is nil",
			func(b *types.Block) error {
				if b == nil {
					return errNilBlock
				}
				return nil
			},
		},
		{
			"block has nil header",
			func(b *types.Block) error {
				if b.Header() == nil {
					return errNilHeader
				}
				return nil
			},
		},
		{
			"block has nil body",
			func(b *types.Block) error {
				if b.Body() == nil {
					return errors.New("nil body")
				}
				return nil
			},
		},
		{
			"block has invalid hash",
			func(b *types.Block) error {
				if b.Hash() == (common.Hash{}) {
					return errors.New("block hash empty")
				}
				// Note that we're confirming that blockchain "has" this block;
				// later we'll use "has-with-state" to differentiate fast/full blocks, and want to be sure
				// that not only is the hash valid, but also that only the state is missing if HasBlock returns false.
				if !bc.HasBlock(b.Hash()) {
					return fmt.Errorf("blockchain cannot find block with hash=%x", b.Hash())
				}
				return nil
			},
		},
		{
			"block has invalid td",
			func(b *types.Block) error {
				if td := b.Difficulty(); td == nil {
					return fmt.Errorf("invalid TD=%v for block #%d", td, b.NumberU64())
				}
				pTd := bc.GetTd(b.ParentHash())
				externTd := new(big.Int).Add(pTd, b.Difficulty())
				if gotTd := bc.GetTd(b.Hash()); gotTd != nil && externTd.Cmp(gotTd) != 0 {
					return fmt.Errorf("invalid TD=%v (want=%v) for block #%d", b.Difficulty(), gotTd, b.NumberU64())
				}
				return nil
			},
		},
	}

	for _, c := range blockAgnosticCases {
		if e := c.test(b); e != nil {
			return e
		}
	}

	// Assume genesis block is healthy from chain configuration.
	if bc.blockIsGenesis(b) {
		return nil
	}

	// If header number is 0 and hash is not genesis hash,
	// something is wrong; possibly missing/malformed header.
	if b.NumberU64() == 0 {
		return fmt.Errorf("block number: 0, but is not genesis block: block: %v, \ngenesis: %v", b, bc.genesisBlock)
	}

	// sharedBlockCheck does everything that bc.Validator().ValidateBlock does, except that it
	// 1. won't return an early error if the block is already known (eg. ErrKnownBlock)
	// 2. won't check state
	sharedBlockCheck := func(b *types.Block) error {
		// Fast and full blocks should always have headers.
		pHash := b.ParentHash()
		if pHash == (common.Hash{}) {
			return ParentError(pHash)
		}
		pHeader := bc.hc.GetHeader(pHash)
		if pHeader == nil {
			return fmt.Errorf("nil parent header for hash: %x", pHash)
		}
		// Use parent header hash to get block (instead of b.ParentHash)
		parent := bc.GetBlock(pHeader.Hash())
		if parent == nil {
			return ParentError(b.ParentHash())
		}

		if err := bc.Validator().ValidateHeader(b.Header(), parent.Header(), true); err != nil {
			return err
		}

		// verify the uncles are correctly rewarded
		if err := bc.Validator().VerifyUncles(b, parent); err != nil {
			return err
		}

		// Verify UncleHash before running other uncle validations
		unclesSha := types.CalcUncleHash(b.Uncles())
		if unclesSha != b.Header().UncleHash {
			return fmt.Errorf("invalid uncles root hash. received=%x calculated=%x", b.Header().UncleHash, unclesSha)
		}

		// The transactions Trie's root (R = (Tr [[i, RLP(T1)], [i, RLP(T2)], ... [n, RLP(Tn)]]))
		// can be used by light clients to make sure they've received the correct Txs
		txSha := types.DeriveSha(b.Transactions())
		if txSha != b.Header().TxHash {
			return fmt.Errorf("invalid transaction root hash. received=%x calculated=%x", b.Header().TxHash, txSha)
		}
		return nil
	}

	// fullBlockCheck ensures state exists for parent and current blocks.
	fullBlockCheck := func(b *types.Block) error {
		parent := bc.GetBlock(b.ParentHash())
		if parent == nil {
			return ParentError(b.ParentHash())
		}
		// Parent does not have state; this is only a corruption if it's the not the point where fast
		// sync was disabled and full states started to be synced.
		if !bc.HasBlockAndState(parent.Hash()) {
			grandparent := bc.GetBlock(parent.ParentHash())
			// If grandparent DOES have state, then so should parent.
			// If grandparent doesn't have state, then assume that it was the first full block synced.
			if bc.HasBlockAndState(grandparent.Hash()) {
				return ParentError(b.ParentHash())
			}
		}
		return nil
	}

	// Since we're using absent state to decide that a full block check isn't required,
	// we need to be sure that an absent state isn't actually a db corruption/inconsistency.
	// Check parent block and confirm that there is NOT any state available for that block, either.
	fastBlockCheck := func(b *types.Block) error {
		pi := b.NumberU64() - 1
		cb := bc.GetBlockByNumber(pi)
		if cb == nil {
			return fmt.Errorf("preceding nil block=#%d, while checking block=#%d health", pi, b.NumberU64())
		}
		if cb.Header() == nil {
			return fmt.Errorf("preceding nil header block=#%d, while checking block=#%d health", pi, b.NumberU64())
		}
		// Genesis will have state.
		if pi > 1 {
			if bc.HasBlockAndState(cb.Hash()) {
				return fmt.Errorf("checking block=%d without state, found nonabsent state for block #%d", b.NumberU64(), pi)
			}
		}
		return nil
	}

	// Use shared block check. The only thing this doesn't check is state.
	if e := sharedBlockCheck(b); e != nil {
		return e
	}

	// Separate checks for fast/full blocks.
	//
	// Assume state is not missing.
	// Fast block.
	if !bc.HasBlockAndState(b.Hash()) {
		glog.V(logger.Debug).Infof("Validating recovery FAST block #%d", b.Number())
		return fastBlockCheck(b)
	}

	// Full block.
	glog.V(logger.Debug).Infof("Validating recovery FULL block #%d", b.Number())
	return fullBlockCheck(b)
}

// Recovery progressively validates the health of stored blockchain data.
// Soft resets should only be called in case of probable corrupted or invalid stored data,
// and which are invalid for known or expected reasons.
// It requires that the blockchain state be loaded so that cached head values are available, eg CurrentBlock(), etc.
func (bc *BlockChain) Recovery(from int, increment int) (checkpoint uint64) {

	// Function for random dynamic incremental stepping through recoverable blocks.
	// This avoids a set pattern for checking block validity, which might present
	// a vulnerability.
	// Random increments are only implemented with using an interval > 1.
	randomizeIncrement := func(i int) int {
		halfIncrement := i / 2
		ri := mrand.Int63n(int64(halfIncrement))
		if ri%2 == 0 {
			return i + int(ri)
		}
		return i - int(ri)
	}

	if from > 1 {
		checkpoint = uint64(from)
	}

	// Hold setting if should randomize incrementing.
	shouldRandomizeIncrement := increment > 1
	// Always base randomization off of original increment,
	// otherwise we'll eventually skew (narrow) the random set.
	dynamicalIncrement := increment

	// Establish initial increment value.
	if shouldRandomizeIncrement {
		dynamicalIncrement = randomizeIncrement(increment)
	}

	// Set up logging for block recovery progress.
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			glog.V(logger.Info).Warnf("Recovered checkpoints through block #%d", checkpoint)
			glog.D(logger.Warn).Warnf("Recovered checkpoints through block #%d", checkpoint)
		}
	}()

	// Step next block.
	// Genesis is used only as a shorthand memory placeholder at a *types.Block; it will be replaced
	// or never returned.
	var checkpointBlockNext = bc.Genesis()

	// In order for this number to be available at the time this function is called, the number (eg. `from`)
	// may need to be persisted locally from the last connection.
	for i := from; i > 0 && checkpointBlockNext != nil; i += dynamicalIncrement {

		checkpointBlockNext = bc.GetBlockByNumber(uint64(i))

		// If block does not exist in db.
		if checkpointBlockNext == nil {
			// Traverse in small steps (increment =1) from last known big step (increment >1) checkpoint.
			if increment > 1 && i-increment > 1 {
				glog.V(logger.Debug).Warnf("Reached nil block #%d, retrying recovery beginning from #%d, incrementing +%d", i, i-increment, 1)
				return bc.Recovery(i-increment, 1) // hone in
			}
			glog.V(logger.Debug).Warnf("No block data available for block #%d", uint64(i))
			break
		}

		// blockIsInvalid runs various block sanity checks, over and above Validator efforts to ensure
		// no known block strangenesses.
		ee := bc.blockIsInvalid(checkpointBlockNext)
		if ee == nil {
			// Everything seems to be fine, set as the head block
			glog.V(logger.Debug).Infof("Found OK later block #%d", checkpointBlockNext.NumberU64())

			checkpoint = checkpointBlockNext.NumberU64()

			bc.hc.SetCurrentHeader(checkpointBlockNext.Header())
			bc.currentFastBlock = checkpointBlockNext

			// If state information is available for block, it is a full block.
			// State validity has been confirmed by `blockIsInvalid == nil`
			if bc.HasBlockAndState(checkpointBlockNext.Hash()) {
				bc.currentBlock = checkpointBlockNext
			}

			if shouldRandomizeIncrement {
				dynamicalIncrement = randomizeIncrement(increment)
			}
			continue
		}
		glog.V(logger.Error).Errorf("Found unhealthy block #%d (%v): \n\n%v", i, ee, checkpointBlockNext)
		if increment == 1 {
			break
		}
		return bc.Recovery(i-increment, 1)
	}
	if checkpoint > 0 {
		glog.V(logger.Warn).Warnf("Found recoverable blockchain data through block #%d", checkpoint)
	}
	return checkpoint
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (bc *BlockChain) LoadLastState(dryrun bool) error {

	bc.mu.Lock()
	defer bc.mu.Unlock()

	// recoverOrReset checks for recoverable block data.
	// If no recoverable block data is found, plain Reset() is called.
	// If safe blockchain data exists (so head block was wrong/invalid), blockchain db head
	// is set to last available safe checkpoint.
	recoverOrReset := func() error {
		glog.V(logger.Warn).Infoln("Checking database for recoverable block data...")

		recoveredHeight := bc.Recovery(1, 100)

		bc.mu.Unlock()
		defer bc.mu.Lock()

		if recoveredHeight == 0 {
			glog.V(logger.Error).Errorln("No recoverable data found, resetting to genesis.")
			return bc.Reset()
		}
		// Remove all block header and canonical data above recoveredHeight
		bc.PurgeAbove(recoveredHeight + 1)
		return bc.SetHead(recoveredHeight)
	}

	// Restore the last known head block
	head := GetHeadBlockHash(bc.chainDb)
	if head == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		if !dryrun {
			glog.V(logger.Warn).Errorln("Empty database, attempting chain reset with recovery.")
			return recoverOrReset()
		}
		return errors.New("empty HeadBlockHash")
	}

	// Get head block by hash
	currentBlock := bc.GetBlock(head)

	// Make sure head block is available.
	if currentBlock == nil {
		// Corrupt or empty database, init from scratch
		if !dryrun {
			glog.V(logger.Warn).Errorf("Head block missing, hash: %x\nAttempting chain reset with recovery.", head)
			return recoverOrReset()
		}
		return errors.New("nil currentBlock")
	}

	// If currentBlock (fullblock) is not genesis, check that it is valid
	// and that it has a state associated with it.
	if currentBlock.Number().Cmp(new(big.Int)) > 0 {
		glog.V(logger.Info).Infof("Validating currentBlock: %v", currentBlock.Number())
		if e := bc.blockIsInvalid(currentBlock); e != nil {
			if !dryrun {
				glog.V(logger.Warn).Errorf("Found unhealthy head full block #%d (%x): %v \nAttempting chain reset with recovery.", currentBlock.Number(), currentBlock.Hash(), e)
				return recoverOrReset()
			}
			return fmt.Errorf("invalid currentBlock: %v", e)
		}
	}

	// Everything seems to be fine, set as the head block
	bc.currentBlock = currentBlock

	// Restore the last known head header
	currentHeader := bc.currentBlock.Header()

	// Get head header by hash.
	if head := GetHeadHeaderHash(bc.chainDb); head != (common.Hash{}) {
		if header := bc.GetHeader(head); header != nil {
			currentHeader = header
		}
	}
	// Ensure total difficulty exists and is valid for current header.
	if td := currentHeader.Difficulty; td == nil || td.Sign() < 1 {
		if !dryrun {
			glog.V(logger.Warn).Errorf("Found current header #%d with invalid TD=%v\nAttempting chain reset with recovery...", currentHeader.Number, td)
			return recoverOrReset()
		}
		return fmt.Errorf("invalid TD=%v for currentHeader=#%d", td, currentHeader.Number)
	}

	bc.hc.SetCurrentHeader(currentHeader)

	// Restore the last known head fast block from placeholder
	bc.currentFastBlock = bc.currentBlock
	if head := GetHeadFastBlockHash(bc.chainDb); head != (common.Hash{}) {
		if block := bc.GetBlock(head); block != nil {
			bc.currentFastBlock = block
		}
	}

	// If currentBlock (fullblock) is not genesis, check that it is valid
	// and that it has a state associated with it.
	if bc.currentFastBlock.Number().Cmp(new(big.Int)) > 0 {
		glog.V(logger.Info).Infof("Validating currentFastBlock: %v", bc.currentFastBlock.Number())
		if e := bc.blockIsInvalid(bc.currentFastBlock); e != nil {
			if !dryrun {
				glog.V(logger.Warn).Errorf("Found unhealthy head fast block #%d (%x): %v \nAttempting chain reset with recovery.", bc.currentFastBlock.Number(), bc.currentFastBlock.Hash(), e)
				return recoverOrReset()
			}
			return fmt.Errorf("invalid currentFastBlock: %v", e)
		}
	}

	// Case: Head full block and fastblock, where there actually exist blocks beyond that
	// in the chain, but last-h/b/fb key-vals (placeholders) are corrupted/wrong, causing it to appear
	// as if the latest block is below the actual blockchain db head block.
	// Update: initial reported case had genesis as constant apparent head.
	// New logs also can show `1` as apparent head.
	// -- Solve: instead of checking via hardcode if apparent genesis block and block `1` is nonnil, check if
	// any block data exists ahead of whatever apparent head is.
	//
	// Look to "rewind" FORWARDS in case of missing/malformed header/placeholder at or near
	// actual stored block head, caused by improper/unfulfilled `WriteHead*` functions,
	// possibly caused by ungraceful stopping on SIGKILL.
	// WriteHead* failing to write in this scenario would cause the head header to be
	// misidentified an artificial non-purged re-sync to begin.
	// I think.

	// If dryrun, don't use additional safety checks or logging and just return state as-is.
	if dryrun {
		return nil
	}

	// Use highest known header number to check for "invisible" block data beyond.
	highestApparentHead := bc.hc.CurrentHeader().Number.Uint64()
	aboveHighestApparentHead := highestApparentHead + 2048

	if b := bc.GetBlockByNumber(aboveHighestApparentHead); b != nil {
		glog.V(logger.Warn).Errorf("Found block data beyond apparent head (head=%d, found=%d)", highestApparentHead, aboveHighestApparentHead)
		return recoverOrReset()
	}

	// Check head block number congruent to hash.
	if b := bc.GetBlockByNumber(bc.currentBlock.NumberU64()); b != nil && b.Header() != nil && b.Header().Hash() != bc.currentBlock.Hash() {
		glog.V(logger.Error).Errorf("Found head block number and hash mismatch: number=%d, hash=%x", bc.currentBlock.NumberU64(), bc.currentBlock.Hash())
		return recoverOrReset()
	}

	// Check head header number congruent to hash.
	if h := bc.hc.GetHeaderByNumber(bc.hc.CurrentHeader().Number.Uint64()); h != nil && bc.hc.GetHeader(h.Hash()) != h {
		glog.V(logger.Error).Errorf("Found head header number and hash mismatch: number=%d, hash=%x", bc.hc.CurrentHeader().Number.Uint64(), bc.hc.CurrentHeader().Hash())
		return recoverOrReset()
	}

	// Case: current header below fast or full block.
	//
	highestCurrentBlockFastOrFull := bc.currentBlock.Number()
	if bc.currentFastBlock.Number().Cmp(highestCurrentBlockFastOrFull) > 0 {
		highestCurrentBlockFastOrFull = bc.currentFastBlock.Number()
	}
	// If the current header is behind head full block OR fast block, we should reset to the height of last OK header.
	if bc.hc.CurrentHeader().Number.Cmp(highestCurrentBlockFastOrFull) < 0 {
		glog.V(logger.Error).Errorf("Found header height below block height, attempting reset with recovery...")
		return recoverOrReset()
	}

	// If header is beyond genesis, but both current and current fast blocks are 0, maybe something has gone wrong,
	// like a write error on previous shutdown.
	if bc.hc.CurrentHeader().Number.Cmp(big.NewInt(0)) > 0 && highestCurrentBlockFastOrFull.Cmp(big.NewInt(0)) == 0 {
		glog.V(logger.Error).Errorf("Found unaccompanied headerchain (headers > 0 && current|fast ==0), attempting reset with recovery...")
	}

	// Initialize a statedb cache to ensure singleton account bloom filter generation
	statedb, err := state.New(bc.currentBlock.Root(), state.NewDatabase(bc.chainDb))
	if err != nil {
		return err
	}
	bc.stateCache = statedb
	bc.stateCache.GetAccount(common.Address{})

	// Issue a status log and return
	headerTd := bc.GetTd(bc.hc.CurrentHeader().Hash())
	blockTd := bc.GetTd(bc.currentBlock.Hash())
	fastTd := bc.GetTd(bc.currentFastBlock.Hash())
	glog.V(logger.Warn).Infof("Last header: #%d [%x…] TD=%v", bc.hc.CurrentHeader().Number, bc.hc.CurrentHeader().Hash().Bytes()[:4], headerTd)
	glog.V(logger.Warn).Infof("Last block: #%d [%x…] TD=%v", bc.currentBlock.Number(), bc.currentBlock.Hash().Bytes()[:4], blockTd)
	glog.V(logger.Warn).Infof("Fast block: #%d [%x…] TD=%v", bc.currentFastBlock.Number(), bc.currentFastBlock.Hash().Bytes()[:4], fastTd)
	glog.D(logger.Warn).Infof("Local head header:     #%s [%s…] TD=%s",
		logger.ColorGreen(strconv.FormatUint(bc.hc.CurrentHeader().Number.Uint64(), 10)),
		logger.ColorGreen(bc.hc.CurrentHeader().Hash().Hex()[:8]),
		logger.ColorGreen(fmt.Sprintf("%v", headerTd)))
	glog.D(logger.Warn).Infof("Local head full block: #%s [%s…] TD=%s",
		logger.ColorGreen(strconv.FormatUint(bc.currentBlock.Number().Uint64(), 10)),
		logger.ColorGreen(bc.currentBlock.Hash().Hex()[:8]),
		logger.ColorGreen(fmt.Sprintf("%v", blockTd)))
	glog.D(logger.Warn).Infof("Local head fast block: #%s [%s…] TD=%s",
		logger.ColorGreen(strconv.FormatUint(bc.currentFastBlock.Number().Uint64(), 10)),
		logger.ColorGreen(bc.currentFastBlock.Hash().Hex()[:8]),
		logger.ColorGreen(fmt.Sprintf("%v", fastTd)))

	return nil
}

// PurgeAbove works like SetHead, but instead of rm'ing head <-> bc.currentBlock,
// it removes all stored blockchain data n -> *anyexistingblockdata*
// TODO: possibly replace with kv database iterator
func (bc *BlockChain) PurgeAbove(n uint64) {
	bc.mu.Lock()

	delFn := func(hash common.Hash) {
		DeleteBody(bc.chainDb, hash)
	}
	bc.hc.PurgeAbove(n, delFn)
	bc.mu.Unlock()
}

// SetHead rewinds the local chain to a new head. In the case of headers, everything
// above the new head will be deleted and the new one set. In the case of blocks
// though, the head may be further rewound if block bodies are missing (non-archive
// nodes after a fast sync).
func (bc *BlockChain) SetHead(head uint64) error {
	glog.V(logger.Warn).Infof("Setting blockchain head, target: %v", head)

	bc.mu.Lock()

	delFn := func(hash common.Hash) {
		DeleteBody(bc.chainDb, hash)
	}
	bc.hc.SetHead(head, delFn)
	currentHeader := bc.hc.CurrentHeader()

	// Clear out any stale content from the caches
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.blockCache.Purge()
	bc.futureBlocks.Purge()

	// Rewind the block chain, ensuring we don't end up with a stateless head block
	if bc.currentBlock != nil && currentHeader.Number.Uint64() < bc.currentBlock.NumberU64() {
		bc.currentBlock = bc.GetBlock(currentHeader.Hash())
	}
	if bc.currentBlock != nil {
		if _, err := state.New(bc.currentBlock.Root(), state.NewDatabase(bc.chainDb)); err != nil {
			// Rewound state missing, rolled back to before pivot, reset to genesis
			bc.currentBlock = nil
		}
	}
	// Rewind the fast block in a simpleton way to the target head
	if bc.currentFastBlock != nil && currentHeader.Number.Uint64() < bc.currentFastBlock.NumberU64() {
		bc.currentFastBlock = bc.GetBlock(currentHeader.Hash())
	}
	// If either blocks reached nil, reset to the genesis state
	if bc.currentBlock == nil {
		bc.currentBlock = bc.genesisBlock
	}
	if bc.currentFastBlock == nil {
		bc.currentFastBlock = bc.genesisBlock
	}

	if err := WriteHeadBlockHash(bc.chainDb, bc.currentBlock.Hash()); err != nil {
		glog.Fatalf("failed to reset head block hash: %v", err)
	}
	if err := WriteHeadFastBlockHash(bc.chainDb, bc.currentFastBlock.Hash()); err != nil {
		glog.Fatalf("failed to reset head fast block hash: %v", err)
	}

	if bc.atxi != nil && bc.atxi.AutoMode {
		ldb, ok := bc.atxi.Db.(*ethdb.LDBDatabase)
		if !ok {
			glog.Fatal("could not cast indexes db to level db")
		}

		var removals [][]byte
		deleteRemovalsFn := func(rs [][]byte) {
			for _, r := range rs {
				if e := ldb.Delete(r); e != nil {
					glog.Fatal(e)
				}
			}
		}

		pre := ethdb.NewBytesPrefix(txAddressIndexPrefix)
		it := ldb.NewIteratorRange(pre)

		for it.Next() {
			key := it.Key()
			_, bn, _, _, _ := resolveAddrTxBytes(key)
			n := binary.LittleEndian.Uint64(bn)
			if n > head {
				removals = append(removals, key)
				// Prevent removals from getting too massive in case it's a big rollback
				// 100000 is a guess at a big but not-too-big memory allowance
				if len(removals) > 100000 {
					deleteRemovalsFn(removals)
					removals = [][]byte{}
				}
			}
		}
		it.Release()
		if e := it.Error(); e != nil {
			return e
		}
		deleteRemovalsFn(removals)

		// update atxi bookmark to lower head in the case that its progress was higher than the new head
		if bc.atxi != nil && bc.atxi.AutoMode {
			if i := bc.atxi.GetATXIBookmark(); i > head {
				if err := bc.atxi.SetATXIBookmark(head); err != nil {
					return err
				}
			}
		}
	}

	bc.mu.Unlock()
	return bc.LoadLastState(false)
}

// FastSyncCommitHead sets the current head block to the one defined by the hash
// irrelevant what the chain contents were prior.
func (bc *BlockChain) FastSyncCommitHead(hash common.Hash) error {
	// Make sure that both the block as well at its state trie exists
	block := bc.GetBlock(hash)
	if block == nil {
		return fmt.Errorf("non existent block [%x…]", hash[:4])
	}
	if _, err := trie.NewSecure(block.Root(), bc.chainDb, 0); err != nil {
		return err
	}
	// If all checks out, manually set the head block
	bc.mu.Lock()
	bc.currentBlock = block
	bc.mu.Unlock()

	glog.V(logger.Info).Infof("committed block #%d [%x…] as new head", block.Number(), hash[:4])
	return nil
}

// GasLimit returns the gas limit of the current HEAD block.
func (bc *BlockChain) GasLimit() *big.Int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.currentBlock.GasLimit()
}

// LastBlockHash return the hash of the HEAD block.
func (bc *BlockChain) LastBlockHash() common.Hash {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.currentBlock.Hash()
}

// CurrentBlock retrieves the current head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) CurrentBlock() *types.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.currentBlock
}

// CurrentFastBlock retrieves the current fast-sync head block of the canonical
// chain. The block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) CurrentFastBlock() *types.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.currentFastBlock
}

// Status returns status information about the current chain such as the HEAD Td,
// the HEAD hash and the hash of the genesis block.
func (bc *BlockChain) Status() (td *big.Int, currentBlock common.Hash, genesisBlock common.Hash) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.GetTd(bc.currentBlock.Hash()), bc.currentBlock.Hash(), bc.genesisBlock.Hash()
}

// SetProcessor sets the processor required for making state modifications.
func (bc *BlockChain) SetProcessor(processor Processor) {
	bc.procmu.Lock()
	defer bc.procmu.Unlock()
	bc.processor = processor
}

// SetValidator sets the validator which is used to validate incoming blocks.
func (bc *BlockChain) SetValidator(validator Validator) {
	bc.procmu.Lock()
	defer bc.procmu.Unlock()
	bc.validator = validator
}

// Validator returns the current validator.
func (bc *BlockChain) Validator() Validator {
	bc.procmu.RLock()
	defer bc.procmu.RUnlock()
	return bc.validator
}

// Processor returns the current processor.
func (bc *BlockChain) Processor() Processor {
	bc.procmu.RLock()
	defer bc.procmu.RUnlock()
	return bc.processor
}

// AuxValidator returns the auxiliary validator (Proof of work atm)
func (bc *BlockChain) AuxValidator() pow.PoW { return bc.pow }

// State returns a new mutable state based on the current HEAD block.
func (bc *BlockChain) State() (*state.StateDB, error) {
	return bc.StateAt(bc.CurrentBlock().Root())
}

// StateAt returns a new mutable state based on a particular point in time.
func (bc *BlockChain) StateAt(root common.Hash) (*state.StateDB, error) {
	return state.New(root, state.NewDatabase(bc.chainDb))
}

// Reset purges the entire blockchain, restoring it to its genesis state.
func (bc *BlockChain) Reset() error {
	return bc.ResetWithGenesisBlock(bc.genesisBlock)
}

// ResetWithGenesisBlock purges the entire blockchain, restoring it to the
// specified genesis state.
func (bc *BlockChain) ResetWithGenesisBlock(genesis *types.Block) error {
	// Dump the entire block chain and purge the caches
	if err := bc.SetHead(0); err != nil {
		return err
	}
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Prepare the genesis block and reinitialise the chain
	if err := bc.hc.WriteTd(genesis.Hash(), genesis.Difficulty()); err != nil {
		glog.Fatalf("failed to write genesis block TD: %v", err)
	}
	if err := WriteBlock(bc.chainDb, genesis); err != nil {
		glog.Fatalf("failed to write genesis block: %v", err)
	}
	bc.genesisBlock = genesis
	bc.insert(bc.genesisBlock)
	bc.currentBlock = bc.genesisBlock
	bc.hc.SetGenesis(bc.genesisBlock.Header())
	bc.hc.SetCurrentHeader(bc.genesisBlock.Header())
	bc.currentFastBlock = bc.genesisBlock

	return nil
}

// Export writes the active chain to the given writer.
func (bc *BlockChain) Export(w io.Writer) error {
	if err := bc.ExportN(w, uint64(0), bc.currentBlock.NumberU64()); err != nil {
		return err
	}
	return nil
}

// ExportN writes a subset of the active chain to the given writer.
func (bc *BlockChain) ExportN(w io.Writer, first uint64, last uint64) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if first > last {
		return fmt.Errorf("export failed: first (%d) is greater than last (%d)", first, last)
	}

	glog.V(logger.Info).Infof("exporting %d blocks...\n", last-first+1)

	for nr := first; nr <= last; nr++ {
		block := bc.GetBlockByNumber(nr)
		if block == nil {
			return fmt.Errorf("export failed on #%d: not found", nr)
		}

		if err := block.EncodeRLP(w); err != nil {
			return err
		}
	}

	return nil
}

// insert injects a new head block into the current block chain. This method
// assumes that the block is indeed a true head. It will also reset the head
// header and the head fast sync block to this very same block if they are older
// or if they are on a different side chain.
//
// Note, this function assumes that the `mu` mutex is held!
func (bc *BlockChain) insert(block *types.Block) {
	// If the block is on a side chain or an unknown one, force other heads onto it too
	updateHeads := GetCanonicalHash(bc.chainDb, block.NumberU64()) != block.Hash()

	// Add the block to the canonical chain number scheme and mark as the head
	if err := WriteCanonicalHash(bc.chainDb, block.Hash(), block.NumberU64()); err != nil {
		glog.Fatalf("failed to insert block number: %v", err)
	}
	if err := WriteHeadBlockHash(bc.chainDb, block.Hash()); err != nil {
		glog.Fatalf("failed to insert head block hash: %v", err)
	}
	bc.currentBlock = block

	// If the block is better than our head or is on a different chain, force update heads
	if updateHeads {
		bc.hc.SetCurrentHeader(block.Header())

		if err := WriteHeadFastBlockHash(bc.chainDb, block.Hash()); err != nil {
			glog.Fatalf("failed to insert head fast block hash: %v", err)
		}
		bc.currentFastBlock = block
	}
}

// Accessors
func (bc *BlockChain) Genesis() *types.Block {
	return bc.genesisBlock
}

// GetBody retrieves a block body (transactions and uncles) from the database by
// hash, caching it if found.
func (bc *BlockChain) GetBody(hash common.Hash) *types.Body {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := bc.bodyCache.Get(hash); ok {
		body := cached.(*types.Body)
		return body
	}
	body := GetBody(bc.chainDb, hash)
	if body == nil {
		return nil
	}
	// Cache the found body for next time and return
	bc.bodyCache.Add(hash, body)
	return body
}

// GetBodyRLP retrieves a block body in RLP encoding from the database by hash,
// caching it if found.
func (bc *BlockChain) GetBodyRLP(hash common.Hash) rlp.RawValue {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := bc.bodyRLPCache.Get(hash); ok {
		return cached.(rlp.RawValue)
	}
	body := GetBodyRLP(bc.chainDb, hash)
	if len(body) == 0 {
		return nil
	}
	// Cache the found body for next time and return
	bc.bodyRLPCache.Add(hash, body)
	return body
}

// HasBlock checks if a block is fully present in the database or not, caching
// it if present.
func (bc *BlockChain) HasBlock(hash common.Hash) bool {
	return bc.GetBlock(hash) != nil
}

// HasBlockAndState checks if a block and associated state trie is fully present
// in the database or not, caching it if present.
func (bc *BlockChain) HasBlockAndState(hash common.Hash) bool {
	// Check first that the block itbc is known
	block := bc.GetBlock(hash)
	if block == nil {
		return false
	}
	// Ensure the associated state is also present
	_, err := state.New(block.Root(), state.NewDatabase(bc.chainDb))
	return err == nil
}

// GetBlock retrieves a block from the database by hash, caching it if found.
func (bc *BlockChain) GetBlock(hash common.Hash) *types.Block {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := bc.blockCache.Get(hash); ok {
		return block.(*types.Block)
	}
	block := GetBlock(bc.chainDb, hash)
	if block == nil {
		return nil
	}
	// Cache the found block for next time and return
	bc.blockCache.Add(block.Hash(), block)
	return block
}

// GetBlockByNumber retrieves a block from the database by number, caching it
// (associated with its hash) if found.
func (bc *BlockChain) GetBlockByNumber(number uint64) *types.Block {
	hash := GetCanonicalHash(bc.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return bc.GetBlock(hash)
}

// [deprecated by eth/62]
// GetBlocksFromHash returns the block corresponding to hash and up to n-1 ancestors.
func (bc *BlockChain) GetBlocksFromHash(hash common.Hash, n int) (blocks []*types.Block) {
	for i := 0; i < n; i++ {
		block := bc.GetBlock(hash)
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		hash = block.ParentHash()
	}
	return
}

// GetUnclesInChain retrieves all the uncles from a given block backwards until
// a specific distance is reached.
func (bc *BlockChain) GetUnclesInChain(block *types.Block, length int) []*types.Header {
	uncles := []*types.Header{}
	for i := 0; block != nil && i < length; i++ {
		uncles = append(uncles, block.Uncles()...)
		block = bc.GetBlock(block.ParentHash())
	}
	return uncles
}

// Stop stops the blockchain service. If any imports are currently in progress
// it will abort them using the procInterrupt.
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	close(bc.quit)
	atomic.StoreInt32(&bc.procInterrupt, 1)

	bc.wg.Wait()

	glog.V(logger.Info).Infoln("Chain manager stopped")
}

type WriteStatus byte

const (
	NonStatTy WriteStatus = iota
	CanonStatTy
	SideStatTy
)

// Rollback is designed to remove a chain of links from the database that aren't
// certain enough to be valid.
func (bc *BlockChain) Rollback(chain []common.Hash) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]

		if bc.hc.CurrentHeader().Hash() == hash {
			bc.hc.SetCurrentHeader(bc.GetHeader(bc.hc.CurrentHeader().ParentHash))
		}
		if bc.currentFastBlock.Hash() == hash {
			bc.currentFastBlock = bc.GetBlock(bc.currentFastBlock.ParentHash())
			if err := WriteHeadFastBlockHash(bc.chainDb, bc.currentFastBlock.Hash()); err != nil {
				glog.Fatalf("failed to write fast head block hash: %v", err)
			}
		}
		if bc.currentBlock.Hash() == hash {
			bc.currentBlock = bc.GetBlock(bc.currentBlock.ParentHash())
			if err := WriteHeadBlockHash(bc.chainDb, bc.currentBlock.Hash()); err != nil {
				glog.Fatalf("failed to write head block hash: %v", err)
			}
		}
	}
}

// InsertReceiptChain attempts to complete an already existing header chain with
// transaction and receipt data.
func (bc *BlockChain) InsertReceiptChain(blockChain types.Blocks, receiptChain []types.Receipts) (res *ReceiptChainInsertResult) {
	res = &ReceiptChainInsertResult{}

	bc.wg.Add(1)
	defer bc.wg.Done()

	// Collect some import statistics to report on
	stats := struct{ processed, ignored int32 }{}
	start := time.Now()

	// Create the block importing task queue and worker functions
	tasks := make(chan int, len(blockChain))
	for i := 0; i < len(blockChain) && i < len(receiptChain); i++ {
		tasks <- i
	}
	close(tasks)

	errs, failed := make([]error, len(tasks)), int32(0)
	process := func(worker int) {
		for index := range tasks {
			block, receipts := blockChain[index], receiptChain[index]

			// Short circuit insertion if shutting down or processing failed
			if atomic.LoadInt32(&bc.procInterrupt) == 1 {
				return
			}
			if atomic.LoadInt32(&failed) > 0 {
				return
			}
			// Short circuit if the owner header is unknown
			if !bc.HasHeader(block.Hash()) {
				errs[index] = fmt.Errorf("containing header #%d [%x…] unknown", block.Number(), block.Hash().Bytes()[:4])
				atomic.AddInt32(&failed, 1)
				return
			}
			// Skip if the entire data is already known
			if bc.HasBlock(block.Hash()) {
				atomic.AddInt32(&stats.ignored, 1)
				continue
			}
			signer := bc.config.GetSigner(block.Number())
			// Compute all the non-consensus fields of the receipts
			transactions, logIndex := block.Transactions(), uint(0)
			for j := 0; j < len(receipts); j++ {
				// The transaction hash can be retrieved from the transaction itbc
				receipts[j].TxHash = transactions[j].Hash()
				tx := transactions[j]
				from, _ := types.Sender(signer, tx)

				// The contract address can be derived from the transaction itbc
				if MessageCreatesContract(transactions[j]) {
					receipts[j].ContractAddress = crypto.CreateAddress(from, tx.Nonce())
				}
				// The used gas can be calculated based on previous receipts
				if j == 0 {
					receipts[j].GasUsed = new(big.Int).Set(receipts[j].CumulativeGasUsed)
				} else {
					receipts[j].GasUsed = new(big.Int).Sub(receipts[j].CumulativeGasUsed, receipts[j-1].CumulativeGasUsed)
				}
				// The derived log fields can simply be set from the block and transaction
				for k := 0; k < len(receipts[j].Logs); k++ {
					receipts[j].Logs[k].BlockNumber = block.NumberU64()
					receipts[j].Logs[k].BlockHash = block.Hash()
					receipts[j].Logs[k].TxHash = receipts[j].TxHash
					receipts[j].Logs[k].TxIndex = uint(j)
					receipts[j].Logs[k].Index = logIndex
					logIndex++
				}
			}
			// Write all the data out into the database
			if err := WriteBody(bc.chainDb, block.Hash(), block.Body()); err != nil {
				errs[index] = fmt.Errorf("failed to write block body: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteBlockReceipts(bc.chainDb, block.Hash(), receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write block receipts: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteMipmapBloom(bc.chainDb, block.NumberU64(), receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write log blooms: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteTransactions(bc.chainDb, block); err != nil {
				errs[index] = fmt.Errorf("failed to write individual transactions: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteReceipts(bc.chainDb, receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write individual receipts: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			// Store the addr-tx indexes if enabled
			if bc.atxi != nil {
				if err := WriteBlockAddTxIndexes(bc.atxi.Db, block); err != nil {
					glog.Fatalf("failed to write block add-tx indexes, err: %v", err)
				}
				// if buildATXI has been in use (via RPC) and is NOT finished, current < stop
				// if buildATXI has been in use (via RPC) and IS finished, current == stop
				// else if builtATXI has not been in use (via RPC), then current == stop == 0
				if bc.atxi.AutoMode && bc.atxi.Progress.Current == bc.atxi.Progress.Stop {
					if err := bc.atxi.SetATXIBookmark(block.NumberU64()); err != nil {
						glog.Fatalln(err)
					}
				}
			}
			atomic.AddInt32(&stats.processed, 1)
		}
	}
	// Start as many worker threads as goroutines allowed
	pending := new(sync.WaitGroup)
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		pending.Add(1)
		go func(id int) {
			defer pending.Done()
			process(id)
		}(i)
	}
	pending.Wait()

	// If anything failed, report
	if failed > 0 {
		for i, err := range errs {
			if err != nil {
				res.Index = i
				res.Error = err
				return
			}
		}
	}

	// if aborted, db could be closed and the td may not be cached so don't attempt to write bookmark
	if atomic.LoadInt32(&bc.procInterrupt) == 1 {
		glog.V(logger.Debug).Infoln("premature abort during receipt chain processing")
		return
	}

	// Update the head fast sync block if better
	bc.mu.Lock()
	head := blockChain[len(errs)-1]
	if bc.GetTd(bc.currentFastBlock.Hash()).Cmp(bc.GetTd(head.Hash())) < 0 {
		if err := WriteHeadFastBlockHash(bc.chainDb, head.Hash()); err != nil {
			glog.Fatalf("failed to update head fast block hash: %v", err)
		}
		bc.currentFastBlock = head
	}
	bc.mu.Unlock()

	// Report some public statistics so the user has a clue what's going on
	first, last := blockChain[0], blockChain[len(blockChain)-1]

	re := ReceiptChainInsertEvent{
		Processed:         int(stats.processed),
		Ignored:           int(stats.ignored),
		Elasped:           time.Since(start),
		FirstHash:         first.Hash(),
		FirstNumber:       first.NumberU64(),
		LastHash:          last.Hash(),
		LastNumber:        last.NumberU64(),
		LatestReceiptTime: time.Unix(last.Time().Int64(), 0),
	}
	res.ReceiptChainInsertEvent = re

	glog.V(logger.Info).Infof("imported %d receipt(s) (%d ignored) in %v. #%d [%x… / %x…]", res.Processed, res.Ignored,
		res.Elasped, res.LastNumber, res.FirstHash.Bytes()[:4], res.LastHash.Bytes()[:4])
	go bc.eventMux.Post(re)
	return res
}

// WriteBlockAddrTxIndexesBatch builds indexes for a given range of blocks N. It writes batches at increment 'step'.
// If any error occurs during db writing it will be returned immediately.
// It's sole implementation is the command 'atxi-build', since we must use individual block atxi indexing during
// sync and import in order to ensure we're on the canonical chain for each block.
func (bc *BlockChain) WriteBlockAddrTxIndexesBatch(indexDb ethdb.Database, startBlockN, stopBlockN, stepN uint64) (txsCount int, err error) {
	block := bc.GetBlockByNumber(startBlockN)
	batch := indexDb.NewBatch()

	blockProcessedCount := uint64(0)
	blockProcessedHead := func() uint64 {
		return startBlockN + blockProcessedCount
	}

	for block != nil && blockProcessedHead() <= stopBlockN {
		txP, err := putBlockAddrTxsToBatch(batch, block)
		if err != nil {
			return txsCount, err
		}
		txsCount += txP
		blockProcessedCount++

		// Write on stepN mod
		if blockProcessedCount%stepN == 0 {
			if err := batch.Write(); err != nil {
				return txsCount, err
			} else {
				batch = indexDb.NewBatch()
			}
		}
		block = bc.GetBlockByNumber(blockProcessedHead())
	}

	// This will put the last batch
	return txsCount, batch.Write()
}

// WriteBlock writes the block to the chain.
func (bc *BlockChain) WriteBlock(block *types.Block) (status WriteStatus, err error) {

	if logger.MlogEnabled() {
		defer func() {
			mlogWriteStatus := "UNKNOWN"
			switch status {
			case NonStatTy:
				mlogWriteStatus = "NONE"
			case CanonStatTy:
				mlogWriteStatus = "CANON"
			case SideStatTy:
				mlogWriteStatus = "SIDE"
			}
			parent := bc.GetBlock(block.ParentHash())
			parentTimeDiff := new(big.Int)
			if parent != nil {
				parentTimeDiff = new(big.Int).Sub(block.Time(), parent.Time())
			}
			mlogBlockchainWriteBlock.AssignDetails(
				mlogWriteStatus,
				err,
				block.Number(),
				block.Hash().Hex(),
				block.Size().Int64(),
				block.Transactions().Len(),
				block.GasUsed(),
				block.Coinbase().Hex(),
				block.Time(),
				block.Difficulty(),
				len(block.Uncles()),
				block.ReceivedAt,
				parentTimeDiff,
			).Send(mlogBlockchain)
		}()
	}

	bc.wg.Add(1)
	defer bc.wg.Done()

	// Calculate the total difficulty of the block
	ptd := bc.GetTd(block.ParentHash())
	if ptd == nil {
		return NonStatTy, ParentError(block.ParentHash())
	}
	// Make sure no inconsistent state is leaked during insertion
	bc.mu.Lock()
	defer bc.mu.Unlock()

	localTd := bc.GetTd(bc.currentBlock.Hash())
	externTd := new(big.Int).Add(block.Difficulty(), ptd)

	// If the total difficulty is higher than our known, add it to the canonical chain
	// Compare local vs external difficulties
	tdCompare := externTd.Cmp(localTd)

	// Initialize reorg if incoming TD is greater than local.
	reorg := tdCompare > 0

	// If TDs are the same, randomize.
	if tdCompare == 0 {
		// Reduces the vulnerability to bcish mining.
		// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
		// Split same-difficulty blocks by number, then at random
		reorg = block.NumberU64() < bc.currentBlock.NumberU64() || (block.NumberU64() == bc.currentBlock.NumberU64() && mrand.Float64() < 0.5)
	}

	if reorg {
		// Reorganise the chain if the parent is not the head block
		if block.ParentHash() != bc.currentBlock.Hash() {
			if err := bc.reorg(bc.currentBlock, block); err != nil {
				return NonStatTy, err
			}
		}
		bc.insert(block) // Insert the block as the new head of the chain
		status = CanonStatTy
	} else {
		status = SideStatTy
	}
	// Irrelevant of the canonical status, write the block itbc to the database
	if err := bc.hc.WriteTd(block.Hash(), externTd); err != nil {
		glog.Fatalf("failed to write block total difficulty: %v", err)
	}
	if err := WriteBlock(bc.chainDb, block); err != nil {
		glog.Fatalf("failed to write block contents: %v", err)
	}

	bc.futureBlocks.Remove(block.Hash())

	return
}

// InsertChain inserts the given chain into the canonical chain or, otherwise, create a fork.
// If the err return is not nil then chainIndex points to the cause in chain.
func (bc *BlockChain) InsertChain(chain types.Blocks) (res *ChainInsertResult) {
	res = &ChainInsertResult{} // initialize
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		if chain[i].NumberU64() != chain[i-1].NumberU64()+1 || chain[i].ParentHash() != chain[i-1].Hash() {
			// Chain broke ancestry, log a messge (programming error) and skip insertion
			glog.V(logger.Error).Infoln("Non contiguous block insert", "number", chain[i].Number(), "hash", chain[i].Hash(),
				"parent", chain[i].ParentHash(), "prevnumber", chain[i-1].Number(), "prevhash", chain[i-1].Hash())

			res.Error = fmt.Errorf("non contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])", i-1, chain[i-1].NumberU64(),
				chain[i-1].Hash().Bytes()[:4], i, chain[i].NumberU64(), chain[i].Hash().Bytes()[:4], chain[i].ParentHash().Bytes()[:4])
			return
		}
	}

	bc.wg.Add(1)
	defer bc.wg.Done()

	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	// A queued approach to delivering events. This is generally
	// faster than direct delivery and requires much less mutex
	// acquiring.
	var (
		stats         struct{ queued, processed, ignored int }
		events        = make([]interface{}, 0, len(chain))
		coalescedLogs vm.Logs
		tstart        = time.Now()

		nonceChecked = make([]bool, len(chain))
	)

	// Start the parallel nonce verifier.
	nonceAbort, nonceResults := verifyNoncesFromBlocks(bc.pow, chain)
	defer close(nonceAbort)

	txcount := 0
	for i, block := range chain {
		res.Index = i
		if atomic.LoadInt32(&bc.procInterrupt) == 1 {
			glog.V(logger.Debug).Infoln("Premature abort during block chain processing")
			break
		}

		bstart := time.Now()
		// Wait for block i's nonce to be verified before processing
		// its state transition.
		for !nonceChecked[i] {
			r := <-nonceResults
			nonceChecked[r.index] = true
			if !r.valid {
				block := chain[r.index]
				res.Index = r.index
				res.Error = &BlockNonceErr{Hash: block.Hash(), Number: block.Number(), Nonce: block.Nonce()}
				return
			}
		}

		if err := bc.config.HeaderCheck(block.Header()); err != nil {
			res.Error = err
			return
		}

		// Stage 1 validation of the block using the chain's validator
		// interface.
		err := bc.Validator().ValidateBlock(block)
		if err != nil {
			if IsKnownBlockErr(err) {
				stats.ignored++
				continue
			}

			if err == BlockFutureErr {
				// Allow up to MaxFuture second in the future blocks. If this limit
				// is exceeded the chain is discarded and processed at a later time
				// if given.
				max := big.NewInt(time.Now().Unix() + maxTimeFutureBlocks)
				if block.Time().Cmp(max) == 1 {
					res.Error = fmt.Errorf("%v: BlockFutureErr, %v > %v", BlockFutureErr, block.Time(), max)
					return
				}
				bc.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			if IsParentErr(err) && bc.futureBlocks.Contains(block.ParentHash()) {
				bc.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			res.Error = err
			return
		}

		// Create a new statedb using the parent block and report an
		// error if it fails.
		switch {
		case i == 0:
			err = bc.stateCache.Reset(bc.GetBlock(block.ParentHash()).Root())
		default:
			err = bc.stateCache.Reset(chain[i-1].Root())
		}
		res.Error = err
		if err != nil {
			return
		}
		// Process block using the parent state as reference point.
		receipts, logs, usedGas, err := bc.processor.Process(block, bc.stateCache)
		if err != nil {
			res.Error = err
			return
		}
		// Validate the state using the default validator
		err = bc.Validator().ValidateState(block, bc.GetBlock(block.ParentHash()), bc.stateCache, receipts, usedGas)
		if err != nil {
			res.Error = err
			return
		}
		// Write state changes to database
		_, err = bc.stateCache.CommitTo(bc.chainDb, bc.config.IsAtlantis(block.Number()))
		if err != nil {
			res.Error = err
			return
		}

		// coalesce logs for later processing
		coalescedLogs = append(coalescedLogs, logs...)

		if err := WriteBlockReceipts(bc.chainDb, block.Hash(), receipts); err != nil {
			res.Error = err
			return
		}

		txcount += len(block.Transactions())
		// write the block to the chain and get the status
		status, err := bc.WriteBlock(block)
		if err != nil {
			res.Error = err
			return
		}

		switch status {
		case CanonStatTy:
			if glog.V(logger.Debug) {
				glog.Infof("[%v] inserted block #%d (%d TXs %v G %d UNCs) [%s]. Took %v\n", time.Now().UnixNano(), block.Number(), len(block.Transactions()), block.GasUsed(), len(block.Uncles()), block.Hash().Hex(), time.Since(bstart))
			}
			events = append(events, ChainEvent{block, block.Hash(), logs})

			// This puts transactions in a extra db for rpc
			if err := WriteTransactions(bc.chainDb, block); err != nil {
				res.Error = err
				return
			}
			// store the receipts
			if err := WriteReceipts(bc.chainDb, receipts); err != nil {
				res.Error = err
				return
			}
			// Write map map bloom filters
			if err := WriteMipmapBloom(bc.chainDb, block.NumberU64(), receipts); err != nil {
				res.Error = err
				return
			}
			// Store the addr-tx indexes if enabled
			if bc.atxi != nil {
				if err := WriteBlockAddTxIndexes(bc.atxi.Db, block); err != nil {
					res.Error = fmt.Errorf("failed to write block add-tx indexes: %v", err)
					return
				}
				// if buildATXI has been in use (via RPC) and is NOT finished, current < stop
				// if buildATXI has been in use (via RPC) and IS finished, current == stop
				// else if builtATXI has not been in use (via RPC), then current == stop == 0
				if bc.atxi.AutoMode && bc.atxi.Progress.Current == bc.atxi.Progress.Stop {
					if err := bc.atxi.SetATXIBookmark(block.NumberU64()); err != nil {
						res.Error = err
						return
					}
				}
			}
		case SideStatTy:
			if glog.V(logger.Detail) {
				glog.Infof("inserted forked block #%d (TD=%v) (%d TXs %d UNCs) [%s]. Took %v\n", block.Number(), block.Difficulty(), len(block.Transactions()), len(block.Uncles()), block.Hash().Hex(), time.Since(bstart))
			}
			events = append(events, ChainSideEvent{block, logs})
		}
		stats.processed++
	}

	ev := ChainInsertEvent{
		Processed: stats.processed,
		Queued:    stats.queued,
		Ignored:   stats.ignored,
		TxCount:   txcount,
	}
	r := &ChainInsertResult{ChainInsertEvent: ev}
	r.Index = 0 // NOTE/FIXME?(whilei): it's kind of strange that it returns 0 when no error... why not len(blocks)-1?
	if stats.queued > 0 || stats.processed > 0 || stats.ignored > 0 {
		elapsed := time.Since(tstart)
		start, end := chain[0], chain[len(chain)-1]
		// fn result
		r.LastNumber = end.NumberU64()
		r.LastHash = end.Hash()
		r.Elasped = elapsed
		r.LatestBlockTime = time.Unix(end.Time().Int64(), 0)
		// add event
		events = append(events, r.ChainInsertEvent)
		// mlog
		if logger.MlogEnabled() {
			mlogBlockchainInsertBlocks.AssignDetails(
				stats.processed,
				stats.queued,
				stats.ignored,
				txcount,
				end.Number(),
				start.Hash().Hex(),
				end.Hash().Hex(),
				elapsed,
			).Send(mlogBlockchain)
		}
		// glog
		glog.V(logger.Info).Infof("imported %d block(s) (%d queued %d ignored) including %d txs in %v. #%v [%s / %s]\n",
			stats.processed,
			stats.queued,
			stats.ignored,
			txcount,
			elapsed,
			end.Number(),
			start.Hash().Hex(),
			end.Hash().Hex())
	}
	go bc.postChainEvents(events, coalescedLogs)

	return r
}

// reorgs takes two blocks, an old chain and a new chain and will reconstruct the blocks and inserts them
// to be part of the new canonical chain and accumulates potential missing transactions and post an
// event about them
func (bc *BlockChain) reorg(oldBlock, newBlock *types.Block) error {
	var (
		newChain          types.Blocks
		oldChain          types.Blocks
		commonBlock       *types.Block
		oldStart          = oldBlock
		newStart          = newBlock
		deletedTxs        types.Transactions
		deletedLogs       vm.Logs
		deletedLogsByHash = make(map[common.Hash]vm.Logs)
		// collectLogs collects the logs that were generated during the
		// processing of the block that corresponds with the given hash.
		// These logs are later announced as deleted.
		collectLogs = func(h common.Hash) {
			// Coalesce logs
			receipts := GetBlockReceipts(bc.chainDb, h)
			for _, receipt := range receipts {
				deletedLogs = append(deletedLogs, receipt.Logs...)

				deletedLogsByHash[h] = receipt.Logs
			}
		}
	)

	// first reduce whoever is higher bound
	if oldBlock.NumberU64() > newBlock.NumberU64() {
		// reduce old chain
		for ; oldBlock != nil && oldBlock.NumberU64() != newBlock.NumberU64(); oldBlock = bc.GetBlock(oldBlock.ParentHash()) {
			oldChain = append(oldChain, oldBlock)
			deletedTxs = append(deletedTxs, oldBlock.Transactions()...)

			collectLogs(oldBlock.Hash())
		}
	} else {
		// reduce new chain and append new chain blocks for inserting later on
		for ; newBlock != nil && newBlock.NumberU64() != oldBlock.NumberU64(); newBlock = bc.GetBlock(newBlock.ParentHash()) {
			newChain = append(newChain, newBlock)
		}
	}
	if oldBlock == nil {
		return fmt.Errorf("Invalid old chain")
	}
	if newBlock == nil {
		return fmt.Errorf("Invalid new chain")
	}

	numSplit := newBlock.Number()
	for {
		if oldBlock.Hash() == newBlock.Hash() {
			commonBlock = oldBlock
			break
		}

		oldChain = append(oldChain, oldBlock)
		newChain = append(newChain, newBlock)
		deletedTxs = append(deletedTxs, oldBlock.Transactions()...)
		collectLogs(oldBlock.Hash())

		oldBlock, newBlock = bc.GetBlock(oldBlock.ParentHash()), bc.GetBlock(newBlock.ParentHash())
		if oldBlock == nil {
			return fmt.Errorf("Invalid old chain")
		}
		if newBlock == nil {
			return fmt.Errorf("Invalid new chain")
		}
	}

	commonHash := commonBlock.Hash()
	if glog.V(logger.Debug) {
		glog.Infof("Chain split detected @ [%s]. Reorganising chain from #%v %s to %s", commonHash.Hex(), numSplit, oldStart.Hash().Hex(), newStart.Hash().Hex())
	}
	if logger.MlogEnabled() {
		mlogBlockchainReorgBlocks.AssignDetails(
			commonHash.Hex(),
			numSplit,
			oldStart.Hash().Hex(),
			newStart.Hash().Hex(),
		).Send(mlogBlockchain)
	}

	// Remove all atxis from old chain; indexes should only reflect canonical
	// Doesn't matter whether automode or not, they should be removed.
	if bc.atxi != nil {
		for _, block := range oldChain {
			for _, tx := range block.Transactions() {
				if err := RmAddrTx(bc.atxi.Db, tx); err != nil {
					return err
				}
			}
		}
	}

	var addedTxs types.Transactions
	// insert blocks. Order does not matter. Last block will be written in ImportChain itbc which creates the new head properly
	for _, block := range newChain {
		// insert the block in the canonical way, re-writing history
		bc.insert(block)
		// write canonical receipts and transactions
		if err := WriteTransactions(bc.chainDb, block); err != nil {
			return err
		}
		// Store the addr-tx indexes if enabled
		if bc.atxi != nil {
			if err := WriteBlockAddTxIndexes(bc.atxi.Db, block); err != nil {
				return err
			}
			// if buildATXI has been in use (via RPC) and is NOT finished, current < stop
			// if buildATXI has been in use (via RPC) and IS finished, current == stop
			// else if builtATXI has not been in use (via RPC), then current == stop == 0
			if bc.atxi.AutoMode && bc.atxi.Progress.Current == bc.atxi.Progress.Stop {
				if err := bc.atxi.SetATXIBookmark(block.NumberU64()); err != nil {
					return err
				}
			}
		}
		receipts := GetBlockReceipts(bc.chainDb, block.Hash())
		// write receipts
		if err := WriteReceipts(bc.chainDb, receipts); err != nil {
			return err
		}
		// Write map map bloom filters
		if err := WriteMipmapBloom(bc.chainDb, block.NumberU64(), receipts); err != nil {
			return err
		}
		addedTxs = append(addedTxs, block.Transactions()...)
	}

	// calculate the difference between deleted and added transactions
	diff := types.TxDifference(deletedTxs, addedTxs)
	// When transactions get deleted from the database that means the
	// receipts that were created in the fork must also be deleted
	for _, tx := range diff {
		DeleteReceipt(bc.chainDb, tx.Hash())
		DeleteTransaction(bc.chainDb, tx.Hash())
	}
	// Must be posted in a goroutine because of the transaction pool trying
	// to acquire the chain manager lock
	if len(diff) > 0 {
		go bc.eventMux.Post(RemovedTransactionEvent{diff})
	}
	if len(deletedLogs) > 0 {
		go bc.eventMux.Post(RemovedLogsEvent{deletedLogs})
	}

	if len(oldChain) > 0 {
		go func() {
			for _, block := range oldChain {
				bc.eventMux.Post(ChainSideEvent{Block: block, Logs: deletedLogsByHash[block.Hash()]})
			}
		}()
	}

	return nil
}

// postChainEvents iterates over the events generated by a chain insertion and
// posts them into the event mux.
func (bc *BlockChain) postChainEvents(events []interface{}, logs vm.Logs) {
	// post event logs for further processing
	bc.eventMux.Post(logs)
	for _, event := range events {
		if event, ok := event.(ChainEvent); ok {
			// We need some control over the mining operation. Acquiring locks and waiting for the miner to create new block takes too long
			// and in most cases isn't even necessary.
			if bc.LastBlockHash() == event.Hash {
				bc.eventMux.Post(ChainHeadEvent{event.Block})
			}
		}
		// Fire the insertion events individually too
		bc.eventMux.Post(event)
	}
}

func (bc *BlockChain) update() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-bc.quit:
			return
		default:
		}

		blocks := make([]*types.Block, 0, bc.futureBlocks.Len())
		for _, hash := range bc.futureBlocks.Keys() {
			if block, exist := bc.futureBlocks.Get(hash); exist {
				blocks = append(blocks, block.(*types.Block))
			}
		}

		if len(blocks) > 0 {
			types.BlockBy(types.Number).Sort(blocks)
			if res := bc.InsertChain(blocks); res.Error != nil {
				log.Printf("periodic future chain update on block #%d [%s]:  %s", blocks[res.Index].Number(), blocks[res.Index].Hash().Hex(), res.Error)
			}
		}
	}
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify parameter can be used to fine tune whether nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verify nonces, as well as
// because nonces can be verified sparsely, not needing to check each.
func (bc *BlockChain) InsertHeaderChain(chain []*types.Header, checkFreq int) *HeaderChainInsertResult {
	// Make sure only one thread manipulates the chain at once
	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	bc.wg.Add(1)
	defer bc.wg.Done()

	whFunc := func(header *types.Header) error {
		bc.mu.Lock()
		defer bc.mu.Unlock()

		_, err := bc.hc.WriteHeader(header)
		return err
	}

	return bc.hc.InsertHeaderChain(chain, checkFreq, whFunc)
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (bc *BlockChain) CurrentHeader() *types.Header {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.hc.CurrentHeader()
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (bc *BlockChain) GetTd(hash common.Hash) *big.Int {
	return bc.hc.GetTd(hash)
}

// GetHeader retrieves a block header from the database by hash, caching it if
// found.
func (bc *BlockChain) GetHeader(hash common.Hash) *types.Header {
	return bc.hc.GetHeader(hash)
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bc *BlockChain) HasHeader(hash common.Hash) bool {
	return bc.hc.HasHeader(hash)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (bc *BlockChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	return bc.hc.GetBlockHashesFromHash(hash, max)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (bc *BlockChain) GetHeaderByNumber(number uint64) *types.Header {
	return bc.hc.GetHeaderByNumber(number)
}

// Config retrieves the blockchain's chain configuration.
func (bc *BlockChain) Config() *ChainConfig { return bc.config }
