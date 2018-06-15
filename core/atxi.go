package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

var (
	errAtxiNotEnabled = errors.New("atxi not intialized")
	errAtxiInvalidUse = errors.New("invalid parameters passed to ATXI")

	txAddressIndexPrefix = []byte("atx-")
	txAddressBookmarkKey = []byte("ATXIBookmark")
)

type AtxiT struct {
	Db       ethdb.Database
	AutoMode bool
	Progress *AtxiProgressT
	Step     uint64
}

type AtxiProgressT struct {
	Start, Stop, Current uint64
	LastError            error
}

func dbGetATXIBookmark(db ethdb.Database) uint64 {
	v, err := db.Get(txAddressBookmarkKey)
	if err != nil || v == nil {
		return 0
	}
	i := binary.LittleEndian.Uint64(v)
	return i
}

func (a *AtxiT) GetATXIBookmark() uint64 {
	return dbGetATXIBookmark(a.Db)
}

func dbSetATXIBookmark(db ethdb.Database, i uint64) error {
	bn := make([]byte, 8)
	binary.LittleEndian.PutUint64(bn, i)
	return db.Put(txAddressBookmarkKey, bn)
}

func (a *AtxiT) SetATXIBookmark(i uint64) error {
	return dbSetATXIBookmark(a.Db, i)
}

// formatAddrTxIterator formats the index key prefix iterator, eg. atx-<address>
func formatAddrTxIterator(address common.Address) (iteratorPrefix []byte) {
	iteratorPrefix = append(iteratorPrefix, txAddressIndexPrefix...)
	iteratorPrefix = append(iteratorPrefix, address.Bytes()...)
	return
}

// formatAddrTxBytesIndex formats the index key, eg. atx-<addr><blockNumber><t|f><s|c|><txhash>
// The values for these arguments should be of determinate length and format, see test TestFormatAndResolveAddrTxBytesKey
// for example.
func formatAddrTxBytesIndex(address, blockNumber, direction, kindof, txhash []byte) (key []byte) {
	key = make([]byte, 0, 66) // 66 is the total capacity of the key = prefix(4)+addr(20)+blockNumber(8)+dir(1)+kindof(1)+txhash(32)
	key = append(key, txAddressIndexPrefix...)
	key = append(key, address...)
	key = append(key, blockNumber...)
	key = append(key, direction...)
	key = append(key, kindof...)
	key = append(key, txhash...)
	return
}

// resolveAddrTxBytes resolves the index key to individual []byte values
func resolveAddrTxBytes(key []byte) (address, blockNumber, direction, kindof, txhash []byte) {
	// prefix = key[:4]
	address = key[4:24]      // common.AddressLength = 20
	blockNumber = key[24:32] // uint64 via little endian
	direction = key[32:33]   // == key[32] (1 byte)
	kindof = key[33:34]
	txhash = key[34:]
	return
}

// WriteBlockAddTxIndexes writes atx-indexes for a given block.
func WriteBlockAddTxIndexes(indexDb ethdb.Database, block *types.Block) error {
	batch := indexDb.NewBatch()
	if _, err := putBlockAddrTxsToBatch(batch, block); err != nil {
		return err
	}
	return batch.Write()
}

// putBlockAddrTxsToBatch formats and puts keys for a given block to a db Batch.
// Batch can be written afterward if no errors, ie. batch.Write()
func putBlockAddrTxsToBatch(putBatch ethdb.Batch, block *types.Block) (txsCount int, err error) {
	for _, tx := range block.Transactions() {
		txsCount++

		from, err := tx.From()
		if err != nil {
			return txsCount, err
		}
		to := tx.To()
		// s: standard
		// c: contract
		txKindOf := []byte("s")
		if to == nil || to.IsEmpty() {
			to = &common.Address{}
			txKindOf = []byte("c")
		}

		// Note that len 8 because uint64 guaranteed <= 8 bytes.
		bn := make([]byte, 8)
		binary.LittleEndian.PutUint64(bn, block.NumberU64())

		if err := putBatch.Put(formatAddrTxBytesIndex(from.Bytes(), bn, []byte("f"), txKindOf, tx.Hash().Bytes()), nil); err != nil {
			return txsCount, err
		}
		if err := putBatch.Put(formatAddrTxBytesIndex(to.Bytes(), bn, []byte("t"), txKindOf, tx.Hash().Bytes()), nil); err != nil {
			return txsCount, err
		}
	}
	return txsCount, nil
}

type atxi struct {
	blockN uint64
	tx     string
}
type sortableAtxis []atxi

// Len implements sort.Sort interface.
func (s sortableAtxis) Len() int {
	return len(s)
}

// Less implements sort.Sort interface.
// By default newer transactions by blockNumber are first.
func (s sortableAtxis) Less(i, j int) bool {
	return s[i].blockN > s[j].blockN
}

// Swap implements sort.Sort interface.
func (s sortableAtxis) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortableAtxis) TxStrings() []string {
	var out = make([]string, 0, len(s))
	for _, str := range s {
		out = append(out, str.tx)
	}
	return out
}

func BuildAddrTxIndex(bc *BlockChain, chainDB, indexDB ethdb.Database, startIndex, stopIndex, step uint64) error {
	if bc.atxi == nil {
		return errors.New("atxi not enabled for blockchain")
	}
	// initialize progress T if not yet
	if bc.atxi.Progress == nil {
		bc.atxi.Progress = &AtxiProgressT{}
	}
	// Use persistent placeholder in case start not spec'd
	if startIndex == math.MaxUint64 {
		startIndex = dbGetATXIBookmark(indexDB)
	}
	if step == math.MaxUint64 {
		step = 10000
	}
	if stopIndex == 0 || stopIndex == math.MaxUint64 {
		stopIndex = bc.CurrentBlock().NumberU64()
		if n := bc.CurrentFastBlock().NumberU64(); n > stopIndex {
			stopIndex = n
		}
	}

	if stopIndex <= startIndex {
		bc.atxi.Progress.LastError = fmt.Errorf("start must be prior to (smaller than) or equal to stop, got start=%d stop=%d", startIndex, stopIndex)
		return bc.atxi.Progress.LastError
	}

	var block *types.Block
	blockIndex := startIndex
	block = bc.GetBlockByNumber(blockIndex)
	if block == nil {
		err := fmt.Errorf("block %d is nil", blockIndex)
		bc.atxi.Progress.LastError = err
		glog.Error(err)
		return err
	}
	// sigc is a single-val channel for listening to program interrupt
	var sigc = make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigc)

	startTime := time.Now()
	totalTxCount := uint64(0)
	glog.D(logger.Error).Infoln("Address/tx indexing (atxi) start:", startIndex, "stop:", stopIndex, "step:", step)
	bc.atxi.Progress.LastError = nil
	bc.atxi.Progress.Current = startIndex
	bc.atxi.Progress.Start = startIndex
	bc.atxi.Progress.Stop = stopIndex
	breaker := false
	for i := startIndex; i < stopIndex; i = i + step {
		if i+step > stopIndex {
			step = stopIndex - i
			breaker = true
		}

		stepStartTime := time.Now()

		// It may seem weird to pass i, i+step, and step, but its just a "coincidence"
		// The function could accepts a smaller step for batch putting (in this case, step),
		// or a larger stopBlock (i+step), but this is just how this cmd is using the fn now
		// We could mess around a little with exploring batch optimization...
		txsCount, err := bc.WriteBlockAddrTxIndexesBatch(indexDB, i, i+step, step)
		if err != nil {
			bc.atxi.Progress.LastError = err
			return err
		}
		totalTxCount += uint64(txsCount)

		bc.atxi.Progress.Current = i + step
		if bc.atxi.AutoMode {
			if err := dbSetATXIBookmark(indexDB, bc.atxi.Progress.Current); err != nil {
				bc.atxi.Progress.LastError = err
				return err
			}
		}

		glog.D(logger.Error).Infof("atxi-build: block %d / %d txs: %d took: %v %.2f bps %.2f txps", i+step, stopIndex, txsCount, time.Since(stepStartTime).Round(time.Millisecond), float64(step)/time.Since(stepStartTime).Seconds(), float64(txsCount)/time.Since(stepStartTime).Seconds())
		glog.V(logger.Info).Infof("atxi-build: block %d / %d txs: %d took: %v %.2f bps %.2f txps", i+step, stopIndex, txsCount, time.Since(stepStartTime).Round(time.Millisecond), float64(step)/time.Since(stepStartTime).Seconds(), float64(txsCount)/time.Since(stepStartTime).Seconds())

		// Listen for interrupts, nonblocking
		select {
		case s := <-sigc:
			glog.D(logger.Info).Warnln("atxi build", "got interrupt:", s, "quitting")
			return nil
		default:
		}

		if breaker {
			break
		}
	}

	if bc.atxi.AutoMode {
		if err := dbSetATXIBookmark(indexDB, stopIndex); err != nil {
			bc.atxi.Progress.LastError = err
			return err
		}
	}

	// Print summary
	totalBlocksF := float64(stopIndex - startIndex)
	totalTxsF := float64(totalTxCount)
	took := time.Since(startTime)
	glog.D(logger.Error).Infof(`Finished atxi-build in %v: %d blocks (~ %.2f blocks/sec), %d txs (~ %.2f txs/sec)`,
		took.Round(time.Second),
		stopIndex-startIndex,
		totalBlocksF/took.Seconds(),
		totalTxCount,
		totalTxsF/took.Seconds(),
	)

	bc.atxi.Progress.LastError = nil
	return nil
}

func (bc *BlockChain) GetATXIBuildProgress() (*AtxiProgressT, error) {
	if bc.atxi == nil {
		return nil, errors.New("atxi not enabled")
	}
	return bc.atxi.Progress, nil
}

// GetAddrTxs gets the indexed transactions for a given account address.
// 'reverse' means "oldest first"
func GetAddrTxs(db ethdb.Database, address common.Address, blockStartN uint64, blockEndN uint64, direction string, kindof string, paginationStart int, paginationEnd int, reverse bool) (txs []string, err error) {
	errWithReason := func(e error, s string) error {
		return fmt.Errorf("%v: %s", e, s)
	}

	// validate params
	if len(direction) > 0 && !strings.Contains("btf", direction[:1]) {
		err = errWithReason(errAtxiInvalidUse, "Address transactions list signature requires direction param to be empty string or [b|t|f] prefix (eg. both, to, or from)")
		return
	}
	if len(kindof) > 0 && !strings.Contains("bsc", kindof[:1]) {
		err = errWithReason(errAtxiInvalidUse, "Address transactions list signature requires 'kind of' param to be empty string or [s|c] prefix (eg. both, standard, or contract)")
		return
	}
	if paginationStart > 0 && paginationEnd > 0 && paginationStart > paginationEnd {
		err = errWithReason(errAtxiInvalidUse, "Pagination start must be less than or equal to pagination end params")
		return
	}
	if paginationStart < 0 {
		paginationStart = 0
	}

	// Have to cast to LevelDB to use iterator. Yuck.
	ldb, ok := db.(*ethdb.LDBDatabase)
	if !ok {
		err = errWithReason(errors.New("internal interface error; please file a bug report"), "could not cast eth db to level db")
		return nil, nil
	}

	// This will be the returnable.
	var hashes []string

	// Map direction -> byte
	var wantDirectionB byte = 'b'
	if len(direction) > 0 {
		wantDirectionB = direction[0]
	}
	var wantKindOf byte = 'b'
	if len(kindof) > 0 {
		wantKindOf = kindof[0]
	}

	// Create address prefix for iteration.
	prefix := ethdb.NewBytesPrefix(formatAddrTxIterator(address))
	it := ldb.NewIteratorRange(prefix)

	var atxis sortableAtxis

	for it.Next() {
		key := it.Key()

		_, blockNum, torf, k, txh := resolveAddrTxBytes(key)

		bn := binary.LittleEndian.Uint64(blockNum)

		// If atxi is smaller than blockstart, skip
		if blockStartN > 0 && bn < blockStartN {
			continue
		}
		// If atxi is greater than blockend, skip
		if blockEndN > 0 && bn > blockEndN {
			continue
		}
		// Ensure matching direction if spec'd
		if wantDirectionB != 'b' && wantDirectionB != torf[0] {
			continue
		}
		// Ensure filter for/agnostic transaction kind of (contract, standard, both)
		if wantKindOf != 'b' && wantKindOf != k[0] {
			continue
		}
		if len(hashes) > 0 {

		}
		tx := common.ToHex(txh)
		atxis = append(atxis, atxi{blockN: bn, tx: tx})
	}
	it.Release()
	err = it.Error()
	if err != nil {
		return
	}

	handleSorting := func(s sortableAtxis) sortableAtxis {
		if len(s) <= 1 {
			return s
		}
		sort.Sort(s) // newest txs (by blockNumber) latest
		if reverse {
			for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
				s[i], s[j] = s[j], s[i]
			}
		}
		if paginationStart > len(s) {
			paginationStart = len(s)
		}
		if paginationEnd < 0 || paginationEnd > len(s) {
			paginationEnd = len(s)
		}
		return s[paginationStart:paginationEnd]
	}
	txs = handleSorting(atxis).TxStrings()
	return
}

// RmAddrTx removes all atxi indexes for a given tx in case of a transaction removal, eg.
// in the case of chain reorg.
// It isn't an elegant function, but not a top priority for optimization because of
// expected infrequency of it's being called.
func RmAddrTx(db ethdb.Database, tx *types.Transaction) error {
	if tx == nil {
		return nil
	}

	ldb, ok := db.(*ethdb.LDBDatabase)
	if !ok {
		return nil
	}

	txH := tx.Hash()
	from, err := tx.From()
	if err != nil {
		return err
	}

	removals := [][]byte{}

	// TODO: not DRY, could be refactored
	pre := ethdb.NewBytesPrefix(formatAddrTxIterator(from))
	it := ldb.NewIteratorRange(pre)
	for it.Next() {
		key := it.Key()
		_, _, _, _, txh := resolveAddrTxBytes(key)
		if bytes.Compare(txH.Bytes(), txh) == 0 {
			removals = append(removals, key)
			break // because there can be only one
		}
	}
	it.Release()
	if e := it.Error(); e != nil {
		return e
	}

	to := tx.To()
	if to != nil {
		toRef := *to
		pre := ethdb.NewBytesPrefix(formatAddrTxIterator(toRef))
		it := ldb.NewIteratorRange(pre)
		for it.Next() {
			key := it.Key()
			_, _, _, _, txh := resolveAddrTxBytes(key)
			if bytes.Compare(txH.Bytes(), txh) == 0 {
				removals = append(removals, key)
				break // because there can be only one
			}
		}
		it.Release()
		if e := it.Error(); e != nil {
			return e
		}
	}

	for _, r := range removals {
		if err := db.Delete(r); err != nil {
			return err
		}
	}
	return nil
}
