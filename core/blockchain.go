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

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/event"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/pow"
	"github.com/ethereumproject/go-ethereum/rlp"
	"github.com/ethereumproject/go-ethereum/trie"
	"github.com/hashicorp/golang-lru"
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

func (self *BlockChain) GetEventMux() *event.TypeMux {
	return self.eventMux
}

func (self *BlockChain) getProcInterrupt() bool {
	return atomic.LoadInt32(&self.procInterrupt) == 1
}

func (self *BlockChain) blockIsGenesis(b *types.Block) bool {
	if self.Genesis() != nil {
		return reflect.DeepEqual(b, self.Genesis())
	}
	ht, _ := DefaultConfigMorden.Genesis.Header()
	hm, _ := DefaultConfigMainnet.Genesis.Header()
	return b.Hash() == ht.Hash() || b.Hash() == hm.Hash()

}

// blockIsInvalid sanity checks for a block's health.
func (self *BlockChain) blockIsInvalid(b *types.Block) error {

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
				// that not only is the hash valid, but also that only the state is missing if HasBlockAndState returns false.
				if !self.HasBlock(b.Hash()) {
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
				pTd := self.GetTd(b.ParentHash())
				externTd := new(big.Int).Add(pTd, b.Difficulty())
				if gotTd := self.GetTd(b.Hash()); gotTd != nil && externTd.Cmp(gotTd) != 0 {
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
	if self.blockIsGenesis(b) {
		return nil
	}

	// If header number is 0 and hash is not genesis hash,
	// something is wrong; possibly missing/malformed header.
	if b.NumberU64() == 0 {
		return fmt.Errorf("block number: 0, but is not genesis block: block: %v, \ngenesis: %v", b, self.genesisBlock)
	}

	// sharedBlockCheck does everything that self.Validator().ValidateBlock does, except that it
	// 1. won't return an early error if the block is already known (eg. ErrKnownBlock)
	// 2. won't check state
	sharedBlockCheck := func(b *types.Block) error {
		// Fast and full blocks should always have headers.
		pHash := b.ParentHash()
		if pHash == (common.Hash{}) {
			return ParentError(pHash)
		}
		pHeader := self.hc.GetHeader(pHash)
		if pHeader == nil {
			return fmt.Errorf("nil parent header for hash: %x", pHash)
		}
		// Use parent header hash to get block (instead of b.ParentHash)
		parent := self.GetBlock(pHeader.Hash())
		if parent == nil {
			return ParentError(b.ParentHash())
		}

		if err := self.Validator().ValidateHeader(b.Header(), parent.Header(), true); err != nil {
			return err
		}

		// verify the uncles are correctly rewarded
		if err := self.Validator().VerifyUncles(b, parent); err != nil {
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
		parent := self.GetBlock(b.ParentHash())
		// == self.HasBlock
		if parent == nil {
			return ParentError(b.ParentHash())
		}
		// Parent does not have state; this is only a corruption if it's the not the point where fast
		// sync was disabled and full states started to be synced.
		if !self.HasBlockAndState(parent.Hash()) {
			grandparent := self.GetBlock(parent.ParentHash())
			// If grandparent DOES have state, then so should parent.
			// If grandparent doesn't have state, then assume that it was the first full block synced.
			if self.HasBlockAndState(grandparent.Hash()) {
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
		cb := self.GetBlockByNumber(pi)
		if cb == nil {
			return fmt.Errorf("preceding nil block=#%d, while checking block=#%d health", pi, b.NumberU64())
		}
		if cb.Header() == nil {
			return fmt.Errorf("preceding nil header block=#%d, while checking block=#%d health", pi, b.NumberU64())
		}
		// Genesis will have state.
		if pi > 1 {
			if self.HasBlockAndState(cb.Hash()) {
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
	if !self.HasBlockAndState(b.Hash()) {
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
func (self *BlockChain) Recovery(from int, increment int) (checkpoint uint64) {

	// Function for random dynamic incremental stepping through recoverable blocks.
	// This avoids a set pattern for checking block validity, which might present
	// a vulnerability.
	// Random increments are only implemented with using an interval > 1.
	randomizeIncrement := func(i int) int {
		mrand.Seed(time.Now().UTC().UnixNano())
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
	var checkpointBlockNext = self.Genesis()

	// In order for this number to be available at the time this function is called, the number (eg. `from`)
	// may need to be persisted locally from the last connection.
	for i := from; i > 0 && checkpointBlockNext != nil; i += dynamicalIncrement {

		checkpointBlockNext = self.GetBlockByNumber(uint64(i))

		// If block does not exist in db.
		if checkpointBlockNext == nil {
			// Traverse in small steps (increment =1) from last known big step (increment >1) checkpoint.
			if increment > 1 && i-increment > 1 {
				glog.V(logger.Debug).Warnf("Reached nil block #%d, retrying recovery beginning from #%d, incrementing +%d", i, i-increment, 1)
				return self.Recovery(i-increment, 1) // hone in
			}
			glog.V(logger.Debug).Warnf("No block data available for block #%d", uint64(i))
			break
		}

		// blockIsInvalid runs various block sanity checks, over and above Validator efforts to ensure
		// no known block strangenesses.
		ee := self.blockIsInvalid(checkpointBlockNext)
		if ee == nil {
			// Everything seems to be fine, set as the head block
			glog.V(logger.Debug).Infof("Found OK later block #%d", checkpointBlockNext.NumberU64())

			checkpoint = checkpointBlockNext.NumberU64()

			self.hc.SetCurrentHeader(checkpointBlockNext.Header())
			self.currentFastBlock = checkpointBlockNext

			// If state information is available for block, it is a full block.
			// State validity has been confirmed by `blockIsInvalid == nil`
			if self.HasBlockAndState(checkpointBlockNext.Hash()) {
				self.currentBlock = checkpointBlockNext
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
		return self.Recovery(i-increment, 1)
	}
	if checkpoint > 0 {
		glog.V(logger.Warn).Warnf("Found recoverable blockchain data through block #%d", checkpoint)
	}
	return checkpoint
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (self *BlockChain) LoadLastState(dryrun bool) error {

	self.mu.Lock()
	defer self.mu.Unlock()

	// recoverOrReset checks for recoverable block data.
	// If no recoverable block data is found, plain Reset() is called.
	// If safe blockchain data exists (so head block was wrong/invalid), blockchain db head
	// is set to last available safe checkpoint.
	recoverOrReset := func() error {
		glog.V(logger.Warn).Infoln("Checking database for recoverable block data...")

		recoveredHeight := self.Recovery(1, 100)

		self.mu.Unlock()
		defer self.mu.Lock()

		if recoveredHeight == 0 {
			glog.V(logger.Error).Errorln("No recoverable data found, resetting to genesis.")
			return self.Reset()
		}
		// Remove all block header and canonical data above recoveredHeight
		self.PurgeAbove(recoveredHeight + 1)
		return self.SetHead(recoveredHeight)
	}

	// Restore the last known head block
	head := GetHeadBlockHash(self.chainDb)
	if head == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		if !dryrun {
			glog.V(logger.Warn).Errorln("Empty database, attempting chain reset with recovery.")
			return recoverOrReset()
		}
		return errors.New("empty HeadBlockHash")
	}

	// Get head block by hash
	currentBlock := self.GetBlock(head)

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
		if e := self.blockIsInvalid(currentBlock); e != nil {
			if !dryrun {
				glog.V(logger.Warn).Errorf("Found unhealthy head full block #%d (%x): %v \nAttempting chain reset with recovery.", currentBlock.Number(), currentBlock.Hash(), e)
				return recoverOrReset()
			}
			return fmt.Errorf("invalid currentBlock: %v", e)
		}
	}

	// Everything seems to be fine, set as the head block
	self.currentBlock = currentBlock

	// Restore the last known head header
	currentHeader := self.currentBlock.Header()

	// Get head header by hash.
	if head := GetHeadHeaderHash(self.chainDb); head != (common.Hash{}) {
		if header := self.GetHeader(head); header != nil {
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

	self.hc.SetCurrentHeader(currentHeader)

	// Restore the last known head fast block from placeholder
	self.currentFastBlock = self.currentBlock
	if head := GetHeadFastBlockHash(self.chainDb); head != (common.Hash{}) {
		if block := self.GetBlock(head); block != nil {
			self.currentFastBlock = block
		}
	}

	// If currentBlock (fullblock) is not genesis, check that it is valid
	// and that it has a state associated with it.
	if self.currentFastBlock.Number().Cmp(new(big.Int)) > 0 {
		glog.V(logger.Info).Infof("Validating currentFastBlock: %v", self.currentFastBlock.Number())
		if e := self.blockIsInvalid(self.currentFastBlock); e != nil {
			if !dryrun {
				glog.V(logger.Warn).Errorf("Found unhealthy head fast block #%d (%x): %v \nAttempting chain reset with recovery.", self.currentFastBlock.Number(), self.currentFastBlock.Hash(), e)
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
	highestApparentHead := self.hc.CurrentHeader().Number.Uint64()
	aboveHighestApparentHead := highestApparentHead + 2048

	if b := self.GetBlockByNumber(aboveHighestApparentHead); b != nil {
		glog.V(logger.Warn).Errorf("Found block data beyond apparent head (head=%d, found=%d)", highestApparentHead, aboveHighestApparentHead)
		return recoverOrReset()
	}

	// Check head block number congruent to hash.
	if b := self.GetBlockByNumber(self.currentBlock.NumberU64()); b != nil && b.Header() != nil && b.Header().Hash() != self.currentBlock.Hash() {
		glog.V(logger.Error).Errorf("Found head block number and hash mismatch: number=%d, hash=%x", self.currentBlock.NumberU64(), self.currentBlock.Hash())
		return recoverOrReset()
	}

	// Check head header number congruent to hash.
	if h := self.hc.GetHeaderByNumber(self.hc.CurrentHeader().Number.Uint64()); h != nil && self.hc.GetHeader(h.Hash()) != h {
		glog.V(logger.Error).Errorf("Found head header number and hash mismatch: number=%d, hash=%x", self.hc.CurrentHeader().Number.Uint64(), self.hc.CurrentHeader().Hash())
		return recoverOrReset()
	}

	// Case: current header below fast or full block.
	//
	highestCurrentBlockFastOrFull := self.currentBlock.Number()
	if self.currentFastBlock.Number().Cmp(highestCurrentBlockFastOrFull) > 0 {
		highestCurrentBlockFastOrFull = self.currentFastBlock.Number()
	}
	// If the current header is behind head full block OR fast block, we should reset to the height of last OK header.
	if self.hc.CurrentHeader().Number.Cmp(highestCurrentBlockFastOrFull) < 0 {
		glog.V(logger.Error).Errorf("Found header height below block height, attempting reset with recovery...")
		return recoverOrReset()
	}

	// Initialize a statedb cache to ensure singleton account bloom filter generation
	statedb, err := state.New(self.currentBlock.Root(), self.chainDb)
	if err != nil {
		return err
	}
	self.stateCache = statedb
	self.stateCache.GetAccount(common.Address{})

	// Issue a status log and return
	headerTd := self.GetTd(self.hc.CurrentHeader().Hash())
	blockTd := self.GetTd(self.currentBlock.Hash())
	fastTd := self.GetTd(self.currentFastBlock.Hash())
	glog.V(logger.Warn).Infof("Last header: #%d [%x…] TD=%v", self.hc.CurrentHeader().Number, self.hc.CurrentHeader().Hash().Bytes()[:4], headerTd)
	glog.V(logger.Warn).Infof("Last block: #%d [%x…] TD=%v", self.currentBlock.Number(), self.currentBlock.Hash().Bytes()[:4], blockTd)
	glog.V(logger.Warn).Infof("Fast block: #%d [%x…] TD=%v", self.currentFastBlock.Number(), self.currentFastBlock.Hash().Bytes()[:4], fastTd)
	glog.D(logger.Warn).Infof("Local head header:     #%s [%s…] TD=%s",
		logger.ColorGreen(strconv.FormatUint(self.hc.CurrentHeader().Number.Uint64(), 10)),
		logger.ColorGreen(self.hc.CurrentHeader().Hash().Hex()[:8]),
		logger.ColorGreen(fmt.Sprintf("%v", headerTd)))
	glog.D(logger.Warn).Infof("Local head full block: #%s [%s…] TD=%s",
		logger.ColorGreen(strconv.FormatUint(self.currentBlock.Number().Uint64(), 10)),
		logger.ColorGreen(self.currentBlock.Hash().Hex()[:8]),
		logger.ColorGreen(fmt.Sprintf("%v", blockTd)))
	glog.D(logger.Warn).Infof("Local head fast block: #%s [%s…] TD=%s",
		logger.ColorGreen(strconv.FormatUint(self.currentFastBlock.Number().Uint64(), 10)),
		logger.ColorGreen(self.currentFastBlock.Hash().Hex()[:8]),
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
		if _, err := state.New(bc.currentBlock.Root(), bc.chainDb); err != nil {
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

	bc.mu.Unlock()
	return bc.LoadLastState(false)
}

// FastSyncCommitHead sets the current head block to the one defined by the hash
// irrelevant what the chain contents were prior.
func (self *BlockChain) FastSyncCommitHead(hash common.Hash) error {
	// Make sure that both the block as well at its state trie exists
	block := self.GetBlock(hash)
	if block == nil {
		return fmt.Errorf("non existent block [%x…]", hash[:4])
	}
	if _, err := trie.NewSecure(block.Root(), self.chainDb, 0); err != nil {
		return err
	}
	// If all checks out, manually set the head block
	self.mu.Lock()
	self.currentBlock = block
	self.mu.Unlock()

	glog.V(logger.Info).Infof("committed block #%d [%x…] as new head", block.Number(), hash[:4])
	return nil
}

// GasLimit returns the gas limit of the current HEAD block.
func (self *BlockChain) GasLimit() *big.Int {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock.GasLimit()
}

// LastBlockHash return the hash of the HEAD block.
func (self *BlockChain) LastBlockHash() common.Hash {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock.Hash()
}

// CurrentBlock retrieves the current head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (self *BlockChain) CurrentBlock() *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock
}

// CurrentFastBlock retrieves the current fast-sync head block of the canonical
// chain. The block is retrieved from the blockchain's internal cache.
func (self *BlockChain) CurrentFastBlock() *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentFastBlock
}

// Status returns status information about the current chain such as the HEAD Td,
// the HEAD hash and the hash of the genesis block.
func (self *BlockChain) Status() (td *big.Int, currentBlock common.Hash, genesisBlock common.Hash) {
	self.mu.RLock()
	defer self.mu.RUnlock()
	return self.GetTd(self.currentBlock.Hash()), self.currentBlock.Hash(), self.genesisBlock.Hash()
}

// SetProcessor sets the processor required for making state modifications.
func (self *BlockChain) SetProcessor(processor Processor) {
	self.procmu.Lock()
	defer self.procmu.Unlock()
	self.processor = processor
}

// SetValidator sets the validator which is used to validate incoming blocks.
func (self *BlockChain) SetValidator(validator Validator) {
	self.procmu.Lock()
	defer self.procmu.Unlock()
	self.validator = validator
}

// Validator returns the current validator.
func (self *BlockChain) Validator() Validator {
	self.procmu.RLock()
	defer self.procmu.RUnlock()
	return self.validator
}

// Processor returns the current processor.
func (self *BlockChain) Processor() Processor {
	self.procmu.RLock()
	defer self.procmu.RUnlock()
	return self.processor
}

// AuxValidator returns the auxiliary validator (Proof of work atm)
func (self *BlockChain) AuxValidator() pow.PoW { return self.pow }

// State returns a new mutable state based on the current HEAD block.
func (self *BlockChain) State() (*state.StateDB, error) {
	return self.StateAt(self.CurrentBlock().Root())
}

// StateAt returns a new mutable state based on a particular point in time.
func (self *BlockChain) StateAt(root common.Hash) (*state.StateDB, error) {
	return self.stateCache.New(root)
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
func (self *BlockChain) Export(w io.Writer) error {
	if err := self.ExportN(w, uint64(0), self.currentBlock.NumberU64()); err != nil {
		return err
	}
	return nil
}

// ExportN writes a subset of the active chain to the given writer.
func (self *BlockChain) ExportN(w io.Writer, first uint64, last uint64) error {
	self.mu.RLock()
	defer self.mu.RUnlock()

	if first > last {
		return fmt.Errorf("export failed: first (%d) is greater than last (%d)", first, last)
	}

	glog.V(logger.Info).Infof("exporting %d blocks...\n", last-first+1)

	for nr := first; nr <= last; nr++ {
		block := self.GetBlockByNumber(nr)
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
func (self *BlockChain) GetBody(hash common.Hash) *types.Body {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := self.bodyCache.Get(hash); ok {
		body := cached.(*types.Body)
		return body
	}
	body := GetBody(self.chainDb, hash)
	if body == nil {
		return nil
	}
	// Cache the found body for next time and return
	self.bodyCache.Add(hash, body)
	return body
}

// GetBodyRLP retrieves a block body in RLP encoding from the database by hash,
// caching it if found.
func (self *BlockChain) GetBodyRLP(hash common.Hash) rlp.RawValue {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := self.bodyRLPCache.Get(hash); ok {
		return cached.(rlp.RawValue)
	}
	body := GetBodyRLP(self.chainDb, hash)
	if len(body) == 0 {
		return nil
	}
	// Cache the found body for next time and return
	self.bodyRLPCache.Add(hash, body)
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
	// Check first that the block itself is known
	block := bc.GetBlock(hash)
	if block == nil {
		return false
	}
	// Ensure the associated state is also present
	_, err := state.New(block.Root(), bc.chainDb)
	return err == nil
}

// GetBlock retrieves a block from the database by hash, caching it if found.
func (self *BlockChain) GetBlock(hash common.Hash) *types.Block {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := self.blockCache.Get(hash); ok {
		return block.(*types.Block)
	}
	block := GetBlock(self.chainDb, hash)
	if block == nil {
		return nil
	}
	// Cache the found block for next time and return
	self.blockCache.Add(block.Hash(), block)
	return block
}

// GetBlockByNumber retrieves a block from the database by number, caching it
// (associated with its hash) if found.
func (self *BlockChain) GetBlockByNumber(number uint64) *types.Block {
	hash := GetCanonicalHash(self.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return self.GetBlock(hash)
}

// [deprecated by eth/62]
// GetBlocksFromHash returns the block corresponding to hash and up to n-1 ancestors.
func (self *BlockChain) GetBlocksFromHash(hash common.Hash, n int) (blocks []*types.Block) {
	for i := 0; i < n; i++ {
		block := self.GetBlock(hash)
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
func (self *BlockChain) GetUnclesInChain(block *types.Block, length int) []*types.Header {
	uncles := []*types.Header{}
	for i := 0; block != nil && i < length; i++ {
		uncles = append(uncles, block.Uncles()...)
		block = self.GetBlock(block.ParentHash())
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
func (self *BlockChain) Rollback(chain []common.Hash) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]

		if self.hc.CurrentHeader().Hash() == hash {
			self.hc.SetCurrentHeader(self.GetHeader(self.hc.CurrentHeader().ParentHash))
		}
		if self.currentFastBlock.Hash() == hash {
			self.currentFastBlock = self.GetBlock(self.currentFastBlock.ParentHash())
			if err := WriteHeadFastBlockHash(self.chainDb, self.currentFastBlock.Hash()); err != nil {
				glog.Fatalf("failed to write fast head block hash: %v", err)
			}
		}
		if self.currentBlock.Hash() == hash {
			self.currentBlock = self.GetBlock(self.currentBlock.ParentHash())
			if err := WriteHeadBlockHash(self.chainDb, self.currentBlock.Hash()); err != nil {
				glog.Fatalf("failed to write head block hash: %v", err)
			}
		}
	}
}

// InsertReceiptChain attempts to complete an already existing header chain with
// transaction and receipt data.
func (self *BlockChain) InsertReceiptChain(blockChain types.Blocks, receiptChain []types.Receipts) (int, error) {
	self.wg.Add(1)
	defer self.wg.Done()

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
			if atomic.LoadInt32(&self.procInterrupt) == 1 {
				return
			}
			if atomic.LoadInt32(&failed) > 0 {
				return
			}
			// Short circuit if the owner header is unknown
			if !self.HasHeader(block.Hash()) {
				errs[index] = fmt.Errorf("containing header #%d [%x…] unknown", block.Number(), block.Hash().Bytes()[:4])
				atomic.AddInt32(&failed, 1)
				return
			}
			// Skip if the entire data is already known
			if self.HasBlock(block.Hash()) {
				atomic.AddInt32(&stats.ignored, 1)
				continue
			}
			signer := self.config.GetSigner(block.Number())
			// Compute all the non-consensus fields of the receipts
			transactions, logIndex := block.Transactions(), uint(0)
			for j := 0; j < len(receipts); j++ {
				// The transaction hash can be retrieved from the transaction itself
				receipts[j].TxHash = transactions[j].Hash()
				tx := transactions[j]
				from, _ := types.Sender(signer, tx)

				// The contract address can be derived from the transaction itself
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
			if err := WriteBody(self.chainDb, block.Hash(), block.Body()); err != nil {
				errs[index] = fmt.Errorf("failed to write block body: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteBlockReceipts(self.chainDb, block.Hash(), receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write block receipts: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteMipmapBloom(self.chainDb, block.NumberU64(), receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write log blooms: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteTransactions(self.chainDb, block); err != nil {
				errs[index] = fmt.Errorf("failed to write individual transactions: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteReceipts(self.chainDb, receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write individual receipts: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
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
				return i, err
			}
		}
	}
	if atomic.LoadInt32(&self.procInterrupt) == 1 {
		glog.V(logger.Debug).Infoln("premature abort during receipt chain processing")
		return 0, nil
	}
	// Update the head fast sync block if better
	self.mu.Lock()
	head := blockChain[len(errs)-1]
	if self.GetTd(self.currentFastBlock.Hash()).Cmp(self.GetTd(head.Hash())) < 0 {
		if err := WriteHeadFastBlockHash(self.chainDb, head.Hash()); err != nil {
			glog.Fatalf("failed to update head fast block hash: %v", err)
		}
		self.currentFastBlock = head
	}
	self.mu.Unlock()

	// Report some public statistics so the user has a clue what's going on
	first, last := blockChain[0], blockChain[len(blockChain)-1]
	glog.V(logger.Info).Infof("imported %d receipt(s) (%d ignored) in %v. #%d [%x… / %x…]", stats.processed, stats.ignored,
		time.Since(start), last.Number(), first.Hash().Bytes()[:4], last.Hash().Bytes()[:4])

	return 0, nil
}

// WriteBlock writes the block to the chain.
func (self *BlockChain) WriteBlock(block *types.Block) (status WriteStatus, err error) {

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
			parent := self.GetBlock(block.ParentHash())
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

	self.wg.Add(1)
	defer self.wg.Done()

	// Calculate the total difficulty of the block
	ptd := self.GetTd(block.ParentHash())
	if ptd == nil {
		return NonStatTy, ParentError(block.ParentHash())
	}
	// Make sure no inconsistent state is leaked during insertion
	self.mu.Lock()
	defer self.mu.Unlock()

	localTd := self.GetTd(self.currentBlock.Hash())
	externTd := new(big.Int).Add(block.Difficulty(), ptd)

	// If the total difficulty is higher than our known, add it to the canonical chain
	// Compare local vs external difficulties
	tdCompare := externTd.Cmp(localTd)

	// Initialize reorg if incoming TD is greater than local.
	reorg := tdCompare > 0

	// If TDs are the same, randomize.
	if tdCompare == 0 {
		// Reduces the vulnerability to selfish mining.
		// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
		reorg = mrand.Float64() < 0.5
	}
	if reorg {
		// Reorganise the chain if the parent is not the head block
		if block.ParentHash() != self.currentBlock.Hash() {
			if err := self.reorg(self.currentBlock, block); err != nil {
				return NonStatTy, err
			}
		}
		self.insert(block) // Insert the block as the new head of the chain
		status = CanonStatTy
	} else {
		status = SideStatTy
	}
	// Irrelevant of the canonical status, write the block itself to the database
	if err := self.hc.WriteTd(block.Hash(), externTd); err != nil {
		glog.Fatalf("failed to write block total difficulty: %v", err)
	}
	if err := WriteBlock(self.chainDb, block); err != nil {
		glog.Fatalf("failed to write block contents: %v", err)
	}

	self.futureBlocks.Remove(block.Hash())

	return
}

// InsertChain inserts the given chain into the canonical chain or, otherwise, create a fork.
// If the err return is not nil then chainIndex points to the cause in chain.
func (self *BlockChain) InsertChain(chain types.Blocks) (chainIndex int, err error) {
	// EPROJECT
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		if chain[i].NumberU64() != chain[i-1].NumberU64()+1 || chain[i].ParentHash() != chain[i-1].Hash() {
			// Chain broke ancestry, log a messge (programming error) and skip insertion
			glog.V(logger.Error).Infof("Non contiguous block insert", "number", chain[i].Number(), "hash", chain[i].Hash(),
				"parent", chain[i].ParentHash(), "prevnumber", chain[i-1].Number(), "prevhash", chain[i-1].Hash())

			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])", i-1, chain[i-1].NumberU64(),
				chain[i-1].Hash().Bytes()[:4], i, chain[i].NumberU64(), chain[i].Hash().Bytes()[:4], chain[i].ParentHash().Bytes()[:4])
		}
	}

	self.wg.Add(1)
	defer self.wg.Done()

	self.chainmu.Lock()
	defer self.chainmu.Unlock()

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
	nonceAbort, nonceResults := verifyNoncesFromBlocks(self.pow, chain)
	defer close(nonceAbort)

	txcount := 0
	var latestBlockTime time.Time
	for i, block := range chain {
		if atomic.LoadInt32(&self.procInterrupt) == 1 {
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
				return r.index, &BlockNonceErr{Hash: block.Hash(), Number: block.Number(), Nonce: block.Nonce()}
			}
		}

		if err := self.config.HeaderCheck(block.Header()); err != nil {
			return i, err
		}

		// Stage 1 validation of the block using the chain's validator
		// interface.
		err := self.Validator().ValidateBlock(block)
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
					return i, fmt.Errorf("%v: BlockFutureErr, %v > %v", BlockFutureErr, block.Time(), max)
				}
				self.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			if IsParentErr(err) && self.futureBlocks.Contains(block.ParentHash()) {
				self.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			return i, err
		}

		// Create a new statedb using the parent block and report an
		// error if it fails.
		switch {
		case i == 0:
			//if self.stateCache == nil {
			//	panic("statecache nil")
			//}
			err = self.stateCache.Reset(self.GetBlock(block.ParentHash()).Root())
		default:
			err = self.stateCache.Reset(chain[i-1].Root())
		}
		if err != nil {
			return i, err
		}
		// Process block using the parent state as reference point.
		receipts, logs, usedGas, err := self.processor.Process(block, self.stateCache)
		if err != nil {
			return i, err
		}
		// Validate the state using the default validator
		err = self.Validator().ValidateState(block, self.GetBlock(block.ParentHash()), self.stateCache, receipts, usedGas)
		if err != nil {
			return i, err
		}
		// Write state changes to database
		_, err = self.stateCache.Commit()
		if err != nil {
			return i, err
		}

		// coalesce logs for later processing
		coalescedLogs = append(coalescedLogs, logs...)

		if err := WriteBlockReceipts(self.chainDb, block.Hash(), receipts); err != nil {
			return i, err
		}

		txcount += len(block.Transactions())
		// write the block to the chain and get the status
		status, err := self.WriteBlock(block)
		if err != nil {
			return i, err
		}
		latestBlockTime = time.Unix(block.Time().Int64(), 0)

		switch status {
		case CanonStatTy:
			if glog.V(logger.Debug) {
				glog.Infof("[%v] inserted block #%d (%d TXs %v G %d UNCs) [%s]. Took %v\n", time.Now().UnixNano(), block.Number(), len(block.Transactions()), block.GasUsed(), len(block.Uncles()), block.Hash().Hex(), time.Since(bstart))
			}
			events = append(events, ChainEvent{block, block.Hash(), logs})

			// This puts transactions in a extra db for rpc
			if err := WriteTransactions(self.chainDb, block); err != nil {
				return i, err
			}
			// store the receipts
			if err := WriteReceipts(self.chainDb, receipts); err != nil {
				return i, err
			}
			// Write map map bloom filters
			if err := WriteMipmapBloom(self.chainDb, block.NumberU64(), receipts); err != nil {
				return i, err
			}
		case SideStatTy:
			if glog.V(logger.Detail) {
				glog.Infof("inserted forked block #%d (TD=%v) (%d TXs %d UNCs) [%s]. Took %v\n", block.Number(), block.Difficulty(), len(block.Transactions()), len(block.Uncles()), block.Hash().Hex(), time.Since(bstart))
			}
			events = append(events, ChainSideEvent{block, logs})
		}
		stats.processed++
	}

	if stats.queued > 0 || stats.processed > 0 || stats.ignored > 0 {
		tend := time.Since(tstart)
		start, end := chain[0], chain[len(chain)-1]
		events = append(events, ChainInsertEvent{
			stats.processed,
			stats.queued,
			stats.ignored,
			txcount,
			end.NumberU64(),
			end.Hash(),
			tend,
			latestBlockTime,
		})
		if logger.MlogEnabled() {
			mlogBlockchainInsertBlocks.AssignDetails(
				stats.processed,
				stats.queued,
				stats.ignored,
				txcount,
				end.Number(),
				start.Hash().Hex(),
				end.Hash().Hex(),
				tend,
			).Send(mlogBlockchain)
		}
		glog.V(logger.Info).Infof("imported %d block(s) (%d queued %d ignored) including %d txs in %v. #%v [%s / %s]\n",
			stats.processed,
			stats.queued,
			stats.ignored,
			txcount,
			tend,
			end.Number(),
			start.Hash().Hex(),
			end.Hash().Hex())
	}
	go self.postChainEvents(events, coalescedLogs)

	return 0, nil
}

// reorgs takes two blocks, an old chain and a new chain and will reconstruct the blocks and inserts them
// to be part of the new canonical chain and accumulates potential missing transactions and post an
// event about them
func (self *BlockChain) reorg(oldBlock, newBlock *types.Block) error {
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
			receipts := GetBlockReceipts(self.chainDb, h)
			for _, receipt := range receipts {
				deletedLogs = append(deletedLogs, receipt.Logs...)

				deletedLogsByHash[h] = receipt.Logs
			}
		}
	)

	// first reduce whoever is higher bound
	if oldBlock.NumberU64() > newBlock.NumberU64() {
		// reduce old chain
		for ; oldBlock != nil && oldBlock.NumberU64() != newBlock.NumberU64(); oldBlock = self.GetBlock(oldBlock.ParentHash()) {
			oldChain = append(oldChain, oldBlock)
			deletedTxs = append(deletedTxs, oldBlock.Transactions()...)

			collectLogs(oldBlock.Hash())
		}
	} else {
		// reduce new chain and append new chain blocks for inserting later on
		for ; newBlock != nil && newBlock.NumberU64() != oldBlock.NumberU64(); newBlock = self.GetBlock(newBlock.ParentHash()) {
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

		oldBlock, newBlock = self.GetBlock(oldBlock.ParentHash()), self.GetBlock(newBlock.ParentHash())
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

	var addedTxs types.Transactions
	// insert blocks. Order does not matter. Last block will be written in ImportChain itself which creates the new head properly
	for _, block := range newChain {
		// insert the block in the canonical way, re-writing history
		self.insert(block)
		// write canonical receipts and transactions
		if err := WriteTransactions(self.chainDb, block); err != nil {
			return err
		}
		receipts := GetBlockReceipts(self.chainDb, block.Hash())
		// write receipts
		if err := WriteReceipts(self.chainDb, receipts); err != nil {
			return err
		}
		// Write map map bloom filters
		if err := WriteMipmapBloom(self.chainDb, block.NumberU64(), receipts); err != nil {
			return err
		}
		addedTxs = append(addedTxs, block.Transactions()...)
	}

	// calculate the difference between deleted and added transactions
	diff := types.TxDifference(deletedTxs, addedTxs)
	// When transactions get deleted from the database that means the
	// receipts that were created in the fork must also be deleted
	for _, tx := range diff {
		DeleteReceipt(self.chainDb, tx.Hash())
		DeleteTransaction(self.chainDb, tx.Hash())
	}
	// Must be posted in a goroutine because of the transaction pool trying
	// to acquire the chain manager lock
	if len(diff) > 0 {
		go self.eventMux.Post(RemovedTransactionEvent{diff})
	}
	if len(deletedLogs) > 0 {
		go self.eventMux.Post(RemovedLogsEvent{deletedLogs})
	}

	if len(oldChain) > 0 {
		go func() {
			for _, block := range oldChain {
				self.eventMux.Post(ChainSideEvent{Block: block, Logs: deletedLogsByHash[block.Hash()]})
			}
		}()
	}

	return nil
}

// postChainEvents iterates over the events generated by a chain insertion and
// posts them into the event mux.
func (self *BlockChain) postChainEvents(events []interface{}, logs vm.Logs) {
	// post event logs for further processing
	self.eventMux.Post(logs)
	for _, event := range events {
		if event, ok := event.(ChainEvent); ok {
			// We need some control over the mining operation. Acquiring locks and waiting for the miner to create new block takes too long
			// and in most cases isn't even necessary.
			if self.LastBlockHash() == event.Hash {
				self.eventMux.Post(ChainHeadEvent{event.Block})
			}
		}
		// Fire the insertion events individually too
		self.eventMux.Post(event)
	}
}

func (chain *BlockChain) update() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-chain.quit:
			return
		default:
		}

		blocks := make([]*types.Block, 0, chain.futureBlocks.Len())
		for _, hash := range chain.futureBlocks.Keys() {
			if block, exist := chain.futureBlocks.Get(hash); exist {
				blocks = append(blocks, block.(*types.Block))
			}
		}

		if len(blocks) > 0 {
			types.BlockBy(types.Number).Sort(blocks)
			if i, err := chain.InsertChain(blocks); err != nil {
				log.Printf("periodic future chain update on block #%d [%s]:  %s", blocks[i].Number(), blocks[i].Hash().Hex(), err)
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
func (self *BlockChain) InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	// Make sure only one thread manipulates the chain at once
	self.chainmu.Lock()
	defer self.chainmu.Unlock()

	self.wg.Add(1)
	defer self.wg.Done()

	whFunc := func(header *types.Header) error {
		self.mu.Lock()
		defer self.mu.Unlock()

		_, err := self.hc.WriteHeader(header)
		return err
	}

	return self.hc.InsertHeaderChain(chain, checkFreq, whFunc)
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (self *BlockChain) CurrentHeader() *types.Header {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.hc.CurrentHeader()
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (self *BlockChain) GetTd(hash common.Hash) *big.Int {
	return self.hc.GetTd(hash)
}

// GetHeader retrieves a block header from the database by hash, caching it if
// found.
func (self *BlockChain) GetHeader(hash common.Hash) *types.Header {
	return self.hc.GetHeader(hash)
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bc *BlockChain) HasHeader(hash common.Hash) bool {
	return bc.hc.HasHeader(hash)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (self *BlockChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	return self.hc.GetBlockHashesFromHash(hash, max)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (self *BlockChain) GetHeaderByNumber(number uint64) *types.Header {
	return self.hc.GetHeaderByNumber(number)
}

// Config retrieves the blockchain's chain configuration.
func (self *BlockChain) Config() *ChainConfig { return self.config }
