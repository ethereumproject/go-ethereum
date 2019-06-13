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

package ethdb

import (
	"path/filepath"

	"strconv"

	"github.com/eth-classic/go-ethereum/logger"
	"github.com/eth-classic/go-ethereum/logger/glog"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	ldbutil "github.com/syndtr/goleveldb/leveldb/util"
	"sync"
)

var OpenFileLimit = 64

// cacheRatio specifies how the total allotted cache is distributed between the
// various system databases.
var cacheRatio = map[string]float64{
	"dapp":      0.0,
	"chaindata": 1.0,
}

// handleRatio specifies how the total alloted file descriptors is distributed
// between the various system databases.
var handleRatio = map[string]float64{
	"dapp":      0.0,
	"chaindata": 1.0,
}

func SetCacheRatio(db string, ratio float64) {
	cacheRatio[db] = ratio
}

func SetHandleRatio(db string, ratio float64) {
	handleRatio[db] = ratio
}

type LDBDatabase struct {
	file string
	db   *leveldb.DB

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLDBDatabase(file string, cache int, handles int) (*LDBDatabase, error) {
	// Calculate the cache and file descriptor allowance for this particular database
	cache = int(float64(cache) * cacheRatio[filepath.Base(file)])
	if cache < 16 {
		cache = 16
	}
	handles = int(float64(handles) * handleRatio[filepath.Base(file)])
	if handles < 16 {
		handles = 16
	}
	glog.V(logger.Info).Infof("Allotted %dMB cache and %d file handles to %s", cache, handles, file)
	glog.D(logger.Warn).Infof("Allotted %s cache and %s file handles to %s", logger.ColorGreen(strconv.Itoa(cache)+"MB"), logger.ColorGreen(strconv.Itoa(handles)), logger.ColorGreen(file))

	// Open the db and recover any potential corruptions
	db, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}
	return &LDBDatabase{
		file: file,
		db:   db,
	}, nil
}

// Path returns the path to the database directory.
func (db *LDBDatabase) Path() string {
	return db.file
}

// Put puts the given key / value to the queue
func (self *LDBDatabase) Put(key []byte, value []byte) error {
	return self.db.Put(key, value, nil)
}

// Get returns the given key if it's present.
func (self *LDBDatabase) Get(key []byte) ([]byte, error) {
	// Retrieve the key and increment the miss counter if not found
	dat, err := self.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

// Delete deletes the key from the queue and database
func (self *LDBDatabase) Delete(key []byte) error {
	// Execute the actual operation
	return self.db.Delete(key, nil)
}

func (self *LDBDatabase) NewIterator() iterator.Iterator {
	return self.db.NewIterator(nil, nil)
}

func (self *LDBDatabase) NewIteratorRange(slice *ldbutil.Range) iterator.Iterator {
	return self.db.NewIterator(slice, nil)
}

func NewBytesPrefix(prefix []byte) *ldbutil.Range {
	return ldbutil.BytesPrefix(prefix)
}

func (self *LDBDatabase) Close() {
	if err := self.db.Close(); err != nil {
		glog.Errorf("eth: DB %s: %s", self.file, err)
	}
}

func (self *LDBDatabase) LDB() *leveldb.DB {
	return self.db
}

// TODO: remove this stuff and expose leveldb directly

func (db *LDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: db.db, b: new(leveldb.Batch)}
}

type ldbBatch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *ldbBatch) Write() error {
	return b.db.Write(b.b, nil)
}

func (b *ldbBatch) ValueSize() int {
	return b.size
}

type table struct {
	db     Database
	prefix string
}

// NewTable returns a Database object that prefixes all keys with a given
// string.
func NewTable(db Database, prefix string) Database {
	return &table{
		db:     db,
		prefix: prefix,
	}
}

func (dt *table) Put(key []byte, value []byte) error {
	return dt.db.Put(append([]byte(dt.prefix), key...), value)
}

func (dt *table) Has(key []byte) (bool, error) {
	return dt.db.Has(append([]byte(dt.prefix), key...))
}

func (dt *table) Get(key []byte) ([]byte, error) {
	return dt.db.Get(append([]byte(dt.prefix), key...))
}

func (dt *table) Delete(key []byte) error {
	return dt.db.Delete(append([]byte(dt.prefix), key...))
}

func (dt *table) Close() {
	// Do nothing; don't close the underlying DB.
}

type tableBatch struct {
	batch  Batch
	prefix string
}

// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db Database, prefix string) Batch {
	return &tableBatch{db.NewBatch(), prefix}
}

func (dt *table) NewBatch() Batch {
	return &tableBatch{dt.db.NewBatch(), dt.prefix}
}

func (tb *tableBatch) Put(key, value []byte) error {
	return tb.batch.Put(append([]byte(tb.prefix), key...), value)
}

func (tb *tableBatch) Write() error {
	return tb.batch.Write()
}

func (tb *tableBatch) ValueSize() int {
	return tb.batch.ValueSize()
}
