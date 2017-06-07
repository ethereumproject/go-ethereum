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

	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

var OpenFileLimit = 64

// cacheRatio specifies how the total alloted cache is distributed between the
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

type LDBDatabase struct {
	file string
	db   *leveldb.DB
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

// Delete deletes the key from the queue and database
func (self *LDBDatabase) Delete(key []byte) error {
	// Execute the actual operation
	return self.db.Delete(key, nil)
}

func (self *LDBDatabase) NewIterator() iterator.Iterator {
	return self.db.NewIterator(nil, nil)
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
	db *leveldb.DB
	b  *leveldb.Batch
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	return nil
}

func (b *ldbBatch) Write() error {
	return b.db.Write(b.b, nil)
}
