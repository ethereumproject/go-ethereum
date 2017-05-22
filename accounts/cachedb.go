// Copyright 2016 The go-ethereum Authors
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

package accounts

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/boltdb/bolt"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"sort"
	"bytes"
	"errors"
	"sync"
	"github.com/mailru/easyjson"
	"fmt"
	"runtime"
)

var addrBucketName = []byte("byAddr")
var fileBucketName = []byte("byFile")
var statsBucketName = []byte("stats")
var ErrCacheDBNoUpdateStamp = errors.New("cachedb has no updated timestamp; expected for newborn dbs.")

// addrCache is a live index of all accounts in the keystore.
type cacheDB struct {
	keydir   string
	watcher  *watcher
	mu       sync.Mutex
	throttle *time.Timer
	db       *bolt.DB
}

func newCacheDB(keydir string) *cacheDB {
	if e := os.MkdirAll(keydir, os.ModePerm); e != nil {
		panic(e)
	}

	dbpath := filepath.Join(keydir, "accounts.db")
	bdb, e := bolt.Open(dbpath, 0600, nil) // TODO configure more?
	if e != nil {
		panic(e)
	}

	cdb := &cacheDB{
		db:     bdb,
	}
	cdb.keydir = keydir

	if e := cdb.db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists(addrBucketName); e != nil {
			return e
		}
		if _, e := tx.CreateBucketIfNotExists(fileBucketName); e != nil {
			return e
		}
		if _, e := tx.CreateBucketIfNotExists(statsBucketName); e != nil {
			return e
		}
		return nil
	}); e != nil {
		panic(e)
	}

	// Initializes db to match fs.
	lu, e := cdb.getLastUpdated()
	if e != nil && e != ErrCacheDBNoUpdateStamp {
		glog.V(logger.Error).Infof("cachedb getupdated error: %v", e)
	}
	if errs := cdb.Syncfs2db(lu); len(errs) != 0 {
		for _, e := range errs {
			if e != nil {
				glog.V(logger.Error).Infof("error db sync: %v", e)
			}
		}
	}
	
	//cdb.watcher = newWatcher(cdb)
	return cdb
}

// Getter functions to implement caching interface.
func (cdb *cacheDB) muLock() {
	cdb.mu.Lock()
}

func (cdb *cacheDB) muUnlock() {
	cdb.mu.Unlock()
}

func (cdb *cacheDB) getKeydir() string {
	return cdb.keydir
}

func (cdb *cacheDB) getWatcher() *watcher {
	return cdb.watcher
}

func (cdb *cacheDB) getThrottle() *time.Timer {
	return cdb.throttle
}

// Gets all accounts _byFile_, which contains and possibly exceed byAddr content
// because it may contain dupe address/key pairs (given dupe files)
func (cdb *cacheDB) accounts() []Account {
	cdb.maybeReload()

	var as []Account
	if e := cdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(fileBucketName)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			a := bytesToAccount(v)
			a.File = string(k)
			as = append(as, a)
		}

		return nil
	}); e != nil {
		panic(e)
	}

	sort.Sort(accountsByFile(as)) // this is important for getting AccountByIndex

	cpy := make([]Account, len(as))
	copy(cpy, as)

	return cpy
}

// note, again, that this return an _slice_
func (cdb *cacheDB) getCachedAccountsByAddress(addr common.Address) (accounts []Account, err error) {
	err = cdb.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(addrBucketName).Cursor()

		prefix := []byte(addr.Hex())
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			accounts = append(accounts, Account{Address: addr, File: string(bytes.Replace(k, prefix, []byte(""), 1))})
		}
		return nil
	})
	if err == nil && (len(accounts) == 0) {
		return accounts, ErrNoMatch
	}
	return accounts, err
}

// ... and this returns an Account
func (cdb *cacheDB) getCachedAccountByFile(file string) (account Account, err error) {
	if file == "" {
		return Account{}, ErrNoMatch
	}
	err = cdb.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(fileBucketName)
		if v := b.Get([]byte(file)); v != nil {
			account = bytesToAccount(v)
		}
		return nil
	})
	if err == nil && (account == Account{}) {
		return account, ErrNoMatch
	}
	return account, err
}

func (cdb *cacheDB) hasAddress(addr common.Address) bool {
	cdb.maybeReload()

	as, e := cdb.getCachedAccountsByAddress(addr)
	return e == nil && len(as) > 0
}

// add makes the assumption that if the account is not cached by file, it won't be listed
// by address either. thusly, when and iff it adds an account to the cache(s), it adds bothly.
func (cdb *cacheDB) add(newAccount Account) {
	defer cdb.setLastUpdated()
	cdb.db.Update(func(tx *bolt.Tx) error {
		newAccount.File = filepath.Base(newAccount.File)
		if newAccount.File != "" {
			bf := tx.Bucket(fileBucketName)
			bf.Put([]byte(newAccount.File), accountToBytes(newAccount))
		}
		if (newAccount.Address != common.Address{}) {
			b := tx.Bucket(addrBucketName)
			return b.Put([]byte(newAccount.Address.Hex()+newAccount.File), []byte(time.Now().String()))
		}
		return nil
	})
}

// note: removed needs to be unique here (i.e. both File and Address must be set).
func (cdb *cacheDB) delete(removed Account) {
	defer cdb.setLastUpdated()
	if e := cdb.db.Update(func(tx *bolt.Tx) error {
		removed.File = filepath.Base(removed.File)

		b := tx.Bucket(fileBucketName)
		if e := b.Delete([]byte(removed.File)); e != nil {
			return e
		}

		ba := tx.Bucket(addrBucketName)
		if e := ba.Delete([]byte(removed.Address.Hex() + removed.File)); e != nil {
			return e
		}
		return nil
	}); e != nil {
		glog.V(logger.Error).Infof("failed to delete from cache: %v \n%v", e, removed.File)
	}
}

// find returns the cached account for address if there is a unique match.
// The exact matching rules are explained by the documentation of Account.
// Callers must hold ac.mu.
func (cdb *cacheDB) find(a Account) (Account, error) {

	var acc Account
	var matches []Account
	var e error

	if a.File != "" {
		acc, e = cdb.getCachedAccountByFile(a.File)
		if e == nil && (acc != Account{}) {
			return acc, e
		}
		// no other possible way
		if (a.Address == common.Address{}) {
			return Account{}, ErrNoMatch
		}
	}

	if (a.Address != common.Address{}) {
		matches, e = cdb.getCachedAccountsByAddress(a.Address)
	}

	switch len(matches) {
	case 1:
		return matches[0], e
	case 0:
		return Account{}, ErrNoMatch
	default:
		err := &AmbiguousAddrError{Addr: a.Address, Matches: make([]Account, len(matches))}
		copy(err.Matches, matches)
		return Account{}, err
	}
}

func (cdb *cacheDB) close() {
	cdb.mu.Lock()
	//cdb.watcher.close()
	//if cdb.throttle != nil {
	//	cdb.throttle.Stop()
	//}
	cdb.db.Close()
	cdb.mu.Unlock()
}

func (cdb *cacheDB) maybeReload() {
	//cdb.mu.Lock()
	//defer cdb.mu.Unlock()
	//if cdb.watcher.running {
	//	return // A watcher is running and will keep the cache up-to-date.
	//}
	//if cdb.throttle == nil {
	//	cdb.throttle = time.NewTimer(0)
	//} else {
	//	select {
	//	case <-cdb.throttle.C:
	//	default:
	//		return // The cache was reloaded recently.
	//	}
	//}
	//cdb.watcher.start()
	cdb.reload()
	//cdb.throttle.Reset(minReloadInterval)
}

func (cdb *cacheDB) setLastUpdated() error {
	return cdb.db.Update(func (tx *bolt.Tx) error {
		b := tx.Bucket(statsBucketName)
		return b.Put([]byte("lastUpdated"), []byte(time.Now().String()))
	})
}

func (cdb *cacheDB) getLastUpdated() (t time.Time, err error) {
	e := cdb.db.View(func (tx *bolt.Tx) error {
		b := tx.Bucket(statsBucketName)
		v := b.Get([]byte("lastUpdated"))
		if v == nil {
			t, err = time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", "1900-01-02 15:04:05.999999999 -0700 MST")
			return ErrCacheDBNoUpdateStamp
		}
		pt, e := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", string(v))
		if e != nil {
			return e
		}
		t = pt
		return nil
	})
	return t, e
}

// setBatchAccounts sets many accounts in a single db tx.
// It saves a lot of time in disk write.
func (cdb *cacheDB) setBatchAccounts(accs []Account) (errs []error) {
	if len(accs) == 0 {
		return nil
	}
	defer cdb.setLastUpdated()

	tx, err := cdb.db.Begin(true)
	if err != nil {
		return append(errs, err)
	}

	ba := tx.Bucket(addrBucketName)
	bf := tx.Bucket(fileBucketName)

	for _, a := range accs {
		// Put in byAddr bucket.
		if e := ba.Put([]byte(a.Address.Hex() + a.File), []byte(time.Now().String())); e != nil {
			errs = append(errs, e)
		}
		// Put in byFile bucket.
		if e := bf.Put([]byte(a.File), accountToBytes(a)); e != nil {
			errs = append(errs, e)
		}
	}

	if len(errs) == 0 {
		// Close tx.
		if err := tx.Commit(); err != nil {
			return append(errs, err)
		}
	} else {
		tx.Rollback()
	}
	return errs
}

// reload caches addresses of existing accounts.
// Callers must hold ac.mu.
func (cdb *cacheDB) reload() {
	defer cdb.setLastUpdated()
	cdb.Syncfs2db(time.Now().Round(time.Second).Add(-minReloadInterval))
}

// Syncfs2db syncronises an existing cachedb with a corresponding fs.
func (cdb *cacheDB) Syncfs2db(lastUpdated time.Time) (errs []error) {
	defer cdb.setLastUpdated()

	var (
		accounts []Account
		removedKeyAddrsFiles [][]byte
		removedKeyFiles []string
	)

	// SYNC: DB --> FS.
	// Iterate addrFiles and touch all in FS, so ensure have "updated" all files which are present in db.
	// Any _new_ files will not have been touched.

	// Time rounded down to second because mod times are store with second accuracy.
	entryCounter := 0
	n := time.Now().Round(time.Second).Add(-minReloadInterval)

	waiterTouch := &sync.WaitGroup{}

	e := cdb.db.Update(func (tx *bolt.Tx) error {

		ab := tx.Bucket(addrBucketName)
		fb := tx.Bucket(fileBucketName)

		c := ab.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			entryCounter++

			// Has address prefix.
			// 0x3ce3a47893f187b0e9f9b2b7e8bdf9052b8e6968UTC--2017-05-19T19-32-22.434229436Z--3ce3a47893f187b0...
			fp := string(k)

			// FIXME: don't hardcode this number.
			// common.Address len?
			if len(fp) >= 42 {

				// Get just the file name.
				fp = fp[42:]
				glog.V(logger.Debug).Infof("(%v) Checking key: %v", entryCounter, fp)

				// No file available as suffix to account address.
				if fp == "" {
					// FIXME: if no file available, this address will become stagnant
					// if more than a week old, remove it?
					tt, te := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", string(v))
					if te != nil {
						errs = append(errs, te)
					}
					if tt.Before(time.Now().Add(-24 * 60 * time.Minute)) {
						removedKeyAddrsFiles = append(removedKeyAddrsFiles, k)
					}
				// Key includes file name.
				} else {
					// /home/data-dir/mainnet/keystore/UTC--2017-05-19T19-32-22.434229436Z--3ce3a47893f187b0...
					p := filepath.Join(cdb.keydir, fp)
					waiterTouch.Add(1)
					go func (path string, rkf []string, waiter *sync.WaitGroup) {
						defer waiter.Done()
						fi, e := os.Stat(p)
						// File exists. Check modification time.
						if e == nil {

							// Only touch files that haven't been modified since entry was updated
							entryTime, te := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", string(v))
							if te != nil {
								errs = append(errs, e)
							} else {
								// DB has been updated more recently than file.
								if entryTime.After(fi.ModTime()) {

									// FIXME: `touch`; is there a better way?
									if cherr := os.Chtimes(p, n, n); cherr != nil {
										errs = append(errs, cherr)
									}
								}
							}
						}
						// This account file has been removed.
						if e != nil && os.IsNotExist(e) {
							glog.V(logger.Debug).Infof("Found key to remove from db: %v", fp)
							removedKeyFiles = append(removedKeyFiles, fp)
						} else {
							errs = append(errs, e)
						}
					}(p, removedKeyFiles, waiterTouch)
				}
			}
		}

		waiterTouch.Wait()

		// Remove from both caches.
		for _, kaf := range removedKeyAddrsFiles {
			glog.V(logger.Debug).Infof("removing key addr file from cachedb: %v", kaf)
			if e := ab.Delete(kaf); e != nil {
				errs = append(errs, e)
			}
		}
		for _, kf := range removedKeyFiles {
			glog.V(logger.Debug).Infof("removing key file from cachedb: %v", kf)
			if e := fb.Delete([]byte(kf)); e != nil {
				errs = append(errs, e)
			}
		}
		return nil
	})
	if e != nil {
		errs = append(errs, e)
	}

	files, err := ioutil.ReadDir(cdb.keydir)
	if err != nil {
		return append(errs, err)
	}
	numFiles := len(files)

	glog.V(logger.Debug).Infof("Syncing index db: %v files", numFiles)

	waitUp := &sync.WaitGroup{}
	achan := make(chan Account)
	echan := make(chan error)
	done := make(chan bool, 1)

	// SYNC: FS --> DB.

	// Handle receiving errors/accounts.
	go func() {
		for j := 0; j < len(files); j++ {
			select {
			case a := <-achan:
				//glog.V(logger.Debug).Infof("[%v/%v] got account from channel: %v", j, len(files), a)
				if (a == Account{}) {
					continue
				}
				accounts = append(accounts, a)
				if len(accounts) == 5000 {
					if e := cdb.setBatchAccounts(accounts); len(e) != 0 {
						for _, ee := range e {
							if ee != nil {
								errs = append(errs, e...)
							}
						}
					}
					accounts = nil
				}
			case e := <-echan:
				//glog.V(logger.Debug).Infof("[%v/%v] got err from channel: %v", j, len(files), e)
				if e != nil {
					errs = append(errs, e)
				}
			}
		}
	}()

	// Iterate files.
	for i, fi := range files {

		// fi.Name() is used for storing the file in the case db
		// This assumes that the keystore/ dir is not designed to walk recursively.
		// See testdata/keystore/foo/UTC-afd..... compared with cacheTestAccounts for
		// test proof of this assumption.
		path := filepath.Join(cdb.keydir, fi.Name())
		if e != nil {
			errs = append(errs, e)
		}
		waitUp.Add(1)
		if runtime.NumGoroutine() > 500 {
			processKeyFile(waitUp, path, fi, i, n, numFiles, achan, echan)
		} else {
			go processKeyFile(waitUp, path, fi, i, n, numFiles, achan, echan)
		}
	}

	// Cleanup.
	go func(wg *sync.WaitGroup, achan chan Account, echan chan error) {
		waitUp.Wait()
		close(achan)
		close(echan)
		done <- true
	}(waitUp, achan, echan)
	<-done

	if e := cdb.setBatchAccounts(accounts); len(e) != 0 {
		for _, ee := range e {
			if ee != nil {
				errs = append(errs, e...)
			}
		}
	}
	accounts = nil

	for _, e := range errs {
		if e != nil {
			glog.V(logger.Debug).Infof("Error: %v", e)
		}
	}

	return errs
}

func processKeyFile(wg *sync.WaitGroup, path string, fi os.FileInfo, i int, n time.Time, numFiles int, aChan chan Account, errs chan error) {
	defer wg.Done()
	if skipKeyFile(fi) {
		glog.V(logger.Debug).Infof("(%v/%v) Ignoring file %s", i, numFiles, fi.Name())
		errs <- nil
		return
	} else {
		// Check touch time from above iterator.
		newy := false
		if fi, fe := os.Stat(path); fe == nil {
			// newy == mod time is before n because we just touched the files we have indexed
			if fi.ModTime().Before(n) {
				newy = true
			}
		} else if fe != nil {
			// just track the error
			errs <- fe
			return
		}
		if !newy {
			glog.V(logger.Debug).Infof("(%v, %v) File existing in index, skipping: %v", i, numFiles, fi.Name())
			errs <- nil
			return
		} else {
			glog.V(logger.Debug).Infof("(%v/%v) Adding key file to db: %v", i, numFiles, fi.Name())

			keyJSON := struct {
				Address common.Address `json:"address"`
			}{}

			buf := new(bufio.Reader)
			fd, err := os.Open(path)
			if err != nil {
				errs <- err
				return
			}
			buf.Reset(fd)
			// Parse the address.
			keyJSON.Address = common.Address{}
			err = json.NewDecoder(buf).Decode(&keyJSON)
			fd.Close()

			web3JSON := []byte{}
			web3JSON, err = ioutil.ReadFile(path)
			if err != nil {
				errs <- err
				return
			}

			switch {
			case err != nil:
				glog.V(logger.Debug).Infof("(%v/%v) can't decode key %s: %v", i, numFiles, path, err)
				errs <- err
			case (keyJSON.Address == common.Address{}):
				glog.V(logger.Debug).Infof("(%v/%v) can't decode key %s: missing or zero address", i, numFiles, path)
				errs <- fmt.Errorf("(%v/%v) can't decode key %s: missing or zero address", i, numFiles, path)
			default:
				aChan <- Account{Address: keyJSON.Address, File: fi.Name(), EncryptedKey: string(web3JSON)}
			}
		}
	}
}

func bytesToAccount(bs []byte) Account {
	var aux AccountJSON
	if e := easyjson.Unmarshal(bs, &aux); e != nil {
		panic(e)
		//return Account{}
	}
	return Account{
		Address: common.HexToAddress(aux.Address),
		EncryptedKey: aux.EncryptedKey,
		File: aux.File,
	}
}

func accountToBytes(account Account) []byte {
	aux := &AccountJSON{
		Address: account.Address.Hex(),
		EncryptedKey: account.EncryptedKey,
		File: account.File,
	}
	b, e := easyjson.Marshal(aux)
	if e != nil {
		panic(e)
		// return nil
	}
	return b
}