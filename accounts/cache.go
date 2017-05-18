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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/rjeczalik/notify"
	"sort"
	"bytes"
	"errors"
	"gopkg.in/mgo.v2/bson"
)

// Minimum amount of time between cache reloads. This limit applies if the platform does
// not support change notifications. It also applies if the keystore directory does not
// exist yet, the code will attempt to create a watcher at most this often.
const minReloadInterval = 2 * time.Second

var addrBucketName = []byte("byAddr")
var fileBucketName = []byte("byFile")
var statsBucketName = []byte("stats")
var ErrCacheDBNoUpdateStamp = errors.New("cachedb has no updated timestamp; expected for newborn dbs.")

type accountsByFile []Account

func (s accountsByFile) Len() int           { return len(s) }
func (s accountsByFile) Less(i, j int) bool { return s[i].File < s[j].File }
func (s accountsByFile) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// AmbiguousAddrError is returned when attempting to unlock
// an address for which more than one file exists.
type AmbiguousAddrError struct {
	Addr    common.Address
	Matches []Account
}

func (err *AmbiguousAddrError) Error() string {
	files := ""
	for i, a := range err.Matches {
		files += a.File
		if i < len(err.Matches)-1 {
			files += ", "
		}
	}
	return fmt.Sprintf("multiple keys match address (%s)", files)
}

// addrCache is a live index of all accounts in the keystore.
type addrCache struct {
	keydir   string
	watcher  *watcher
	mu       sync.Mutex
	db       *bolt.DB
	throttle *time.Timer
}

func newAddrCache(keydir string) *addrCache {
	if e := os.MkdirAll(keydir, os.ModePerm); e != nil {
		panic(e)
	}

	dbpath := filepath.Join(keydir, "accounts.db")
	bdb, e := bolt.Open(dbpath, 0600, nil) // TODO configure more?
	if e != nil {
		panic(e) // FIXME
	}

	ac := &addrCache{
		keydir: keydir,
		db:     bdb,
	}

	if e := ac.db.Update(func(tx *bolt.Tx) error {
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
	lu, e := ac.getLastUpdated()
	if e != nil && e != ErrCacheDBNoUpdateStamp {
		glog.V(logger.Error).Infof("cachedb getupdated error: %v", e)
	}
	if errs := ac.syncfs2db(lu); len(errs) != 0 {
		for _, e := range errs {
			if e != nil {
				glog.V(logger.Error).Infof("error db sync: %v", e)
			}
		}
	}
	
	ac.watcher = newWatcher(ac)
	return ac
}

// Gets all accounts _byFile_, which contains and possibly exceed byAddr content
// because it may contain dupe address/key pairs (given dupe files)
func (ac *addrCache) accounts() []Account {
	ac.maybeReload()

	var as []Account
	if e := ac.db.View(func(tx *bolt.Tx) error {
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
	sort.Sort(accountsByFile(as))
	cpy := make([]Account, len(as))
	copy(cpy, as)
	return cpy
}

// note, again, that this return an _slice_
func (ac *addrCache) getCachedAccountsByAddress(addr common.Address) (accounts []Account, err error) {
	err = ac.db.View(func(tx *bolt.Tx) error {
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
func (ac *addrCache) getCachedAccountByFile(file string) (account Account, err error) {
	if file == "" {
		return Account{}, ErrNoMatch
	}
	err = ac.db.View(func(tx *bolt.Tx) error {
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

func (ac *addrCache) hasAddress(addr common.Address) bool {
	ac.maybeReload()

	as, e := ac.getCachedAccountsByAddress(addr)
	return e == nil && len(as) > 0
}

// add makes the assumption that if the account is not cached by file, it won't be listed
// by address either. thusly, when and iff it adds an account to the cache(s), it adds bothly.
func (ac *addrCache) add(newAccount Account) {
	defer ac.setLastUpdated()
	ac.db.Update(func(tx *bolt.Tx) error {
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
func (ac *addrCache) delete(removed Account) {
	defer ac.setLastUpdated()
	if e := ac.db.Update(func(tx *bolt.Tx) error {
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

func removeAccount(slice []Account, elem Account) []Account {
	for i := range slice {
		if slice[i] == elem {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// find returns the cached account for address if there is a unique match.
// The exact matching rules are explained by the documentation of Account.
// Callers must hold ac.mu.
func (ac *addrCache) find(a Account) (Account, error) {

	var acc Account
	var matches []Account
	var e error

	if a.File != "" {
		// If only the basename is specified, complete the path.
		if !strings.ContainsRune(a.File, filepath.Separator) {
			a.File = filepath.Join(ac.keydir, a.File)
		}
		acc, e = ac.getCachedAccountByFile(a.File)
		if e == nil && (acc != Account{}) {
			return acc, e
		}
		// no other possible way
		if (a.Address == common.Address{}) {
			return Account{}, ErrNoMatch
		}
	}

	if (a.Address != common.Address{}) {
		matches, e = ac.getCachedAccountsByAddress(a.Address)
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

func (ac *addrCache) close() {
	ac.mu.Lock()
	ac.watcher.close()
	if ac.throttle != nil {
		ac.throttle.Stop()
	}
	ac.db.Close()
	ac.mu.Unlock()
}

// set is used by the fs watcher to update the cache from a given file path.
// it has some logic;
// -- it will _overwrite_ any existing cache entry by file and addr, making it useful for CREATE and UPDATE
func (ac *addrCache) setViaFile(name string) error {
	// first sync fs -> cachedb, update all accounts in cache from fs
	var (
		buf     = new(bufio.Reader)
		acc Account
		keyJSON struct {
			   Address common.Address `json:"address"`
		   }
		web3JSON []byte
	)

	path := filepath.Join(ac.keydir, name)
	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	buf.Reset(fd)
	// Parse the address.
	keyJSON.Address = common.Address{}
	err = json.NewDecoder(buf).Decode(&keyJSON)
	fd.Close()

	// Slurp file contents to store in the cachedb, saving an io read at
	// keystore.Lookup.
	web3JSON, e := ioutil.ReadFile(path)
	if e != nil {
		return e
	}

	switch {
	case err != nil:
		return fmt.Errorf("can't decode key %s: %v", path, err)
	case (keyJSON.Address == common.Address{}):
		return fmt.Errorf("can't decode key %s: missing or zero address", path)
	default:
		acc = Account{Address: keyJSON.Address, File: name, EncryptedKey: string(web3JSON)}
	}

	ab := accountToBytes(acc)
	return ac.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(fileBucketName)
		if e := b.Put([]byte(path), ab); e != nil {
			return e
		}
		b = tx.Bucket(addrBucketName)
		return b.Put([]byte(keyJSON.Address.Hex()+name), []byte(time.Now().String()))
	})
}

// remove is used by the fs watcher to update the cache from a given path.
func (ac *addrCache) removeViaFile(path string) error {

	var acc Account

	if e := ac.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(fileBucketName)
		ab := b.Get([]byte(path))
		if ab == nil {
			return ErrNoMatch
		}
		acc = bytesToAccount(ab)
		if e := b.Delete([]byte(path)); e != nil {
			return e
		}

		return tx.Bucket(addrBucketName).Delete([]byte(acc.Address.Hex() + acc.File))

	}); e != nil {
		return e
	}
	return nil
}

func (ac *addrCache) maybeReload() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if ac.watcher.running {
		return // A watcher is running and will keep the cache up-to-date.
	}
	if ac.throttle == nil {
		ac.throttle = time.NewTimer(0)
	} else {
		select {
		case <-ac.throttle.C:
		default:
			return // The cache was reloaded recently.
		}
	}
	ac.watcher.start()
	ac.reload(ac.watcher.evs)
	ac.throttle.Reset(minReloadInterval)
}

// reload caches addresses of existing accounts.
// Callers must hold ac.mu.
func (ac *addrCache) reload(events []notify.EventInfo) []notify.EventInfo {
	defer ac.setLastUpdated()

	// Decide kind of event.
	for _, ev := range events {

		glog.V(logger.Debug).Infof("reloading event: %v", ev)

		p := ev.Path() // provides a clean absolute path

		// Nuance of Notify package Path():
		// on /tmp will report events with paths rooted at /private/tmp etc.
		if strings.HasPrefix(p, "/private") {
			p = strings.Replace(p, "/private","",1) // only replace first occurance
		}
		fi, e := os.Stat(p)
		if e != nil {
			continue // TODO handle better
		}
		if fi.IsDir() { // don't expect many of these from Notify, but just in case
			continue // only want files, no dirs
		}

		p = fi.Name()

		// TODO: don't ignore the returned errors
		switch ev.Event() {
		case notify.Create:
			glog.V(logger.Debug).Infof("reloading create event: %v", ev.Event())
			ac.setViaFile(p)
		case notify.Rename:
			glog.V(logger.Debug).Infof("reloading rename event (doing nothing): %v", ev.Event())
			// TODO: do something... how to get old vs. new paths?
		case notify.Remove:
			glog.V(logger.Debug).Infof("reloading remove event: %v", ev.Event())
			ac.removeViaFile(p) // TODO: write test
		case notify.Write:
			glog.V(logger.Debug).Infof("reloading write event: %v", ev.Event())
			ac.setViaFile(p)
		default:
			// do nothing
		}
	}
	return []notify.EventInfo{}
}

func (ac *addrCache) setLastUpdated() error {
	return ac.db.Update(func (tx *bolt.Tx) error {
		b := tx.Bucket(statsBucketName)
		return b.Put([]byte("lastUpdated"), []byte(time.Now().String()))
	})
}

func (ac *addrCache) getLastUpdated() (t time.Time, err error) {
	e := ac.db.View(func (tx *bolt.Tx) error {
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
func (ac *addrCache) setBatchAccounts(accs []Account) (errs []error) {
	if len(accs) == 0 {
		return nil
	}
	defer ac.setLastUpdated()

	tx, err := ac.db.Begin(true)
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

// syncfs2db syncronises an existing cachedb with a corresponding fs.
func (ac *addrCache) syncfs2db(lastUpdated time.Time) (errs []error) {
	defer ac.setLastUpdated()

	di, de := os.Stat(ac.keydir)
	if de != nil {
		errs = append(errs, de)
		return errs
	}
	if lastUpdated.After(di.ModTime()) {
		return errs
	}

	files, err := ioutil.ReadDir(ac.keydir)
	if err != nil {
		return append(errs, err)
	}

	var (
		buf     = new(bufio.Reader)
		accounts []Account
		keyJSON struct {
			Address common.Address `json:"address"`
		}
		web3JSON []byte
	)

	// SYNC: DB --> FS.
	// Iterate addrFiles and touch all in FS, so ensure have "updated" all files which are present in db.
	// Any _new_ files will not have been touched.
	n := time.Now()
	e := ac.db.Update(func (tx *bolt.Tx) error {
		var removedAccounts []Account
		fb := tx.Bucket(fileBucketName)
		ab := tx.Bucket(addrBucketName)
		c := ab.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {

			fp := string(k) // Has address prefix.
			if len(fp) >= common.AddressLength {
				fp = fp[common.AddressLength:]
				if fp == "" {
					continue
				}
			}

			p := filepath.Join(ac.keydir, fp)
			fi, e := os.Stat(p)
			if e == nil {
				// only touch files that haven't been modified since entry was updated
				tt, te := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", string(v))
				if te != nil {
					errs = append(errs, err)
				} else {
					if tt.After(fi.ModTime()) {
						// FIXME: is there a better way to `touch`?
						if cherr := os.Chtimes(p, n, n); cherr != nil {
							errs = append(errs, err)
						}
					}
				}
			}
			// This account file has been removed.
			if e != nil && os.IsNotExist(e) {
				removedAccounts = append(removedAccounts, bytesToAccount(v))
			} else {
				errs = append(errs, err)
				continue
			}
		}

		// Remove from both caches.
		for _, ra := range removedAccounts {
			if e := fb.Delete([]byte(ra.File)); e != nil {
				errs = append(errs, e)
			}
			if e := ab.Delete([]byte(ra.Address.Hex() + ra.File)); e != nil {
				errs = append(errs, e)
			}
		}
		return nil
	})
	if e != nil {
		errs = append(errs, e)
	}

	// SYNC: FS --> DB.
	for i, fi := range files {

		newy := false
		path := filepath.Join(ac.keydir, fi.Name())

		if skipKeyFile(fi) {
			glog.V(logger.Detail).Infof("ignoring file %s", path)

		} else {
			// Check touch time from above iterator.
			if fi, fe := os.Stat(path); fe == nil {
				// newy == mod time is before n because we just touched the files we have indexed
				if fi.ModTime().UTC().Before(n) {
					newy = true
				}
			} else if fe != nil {
				errs = append(errs, fe)
				continue
			}

			if newy {
				fd, err := os.Open(path)
				if err != nil {
					errs = append(errs, err)
				}
				buf.Reset(fd)
				// Parse the address.
				keyJSON.Address = common.Address{}
				err = json.NewDecoder(buf).Decode(&keyJSON)
				fd.Close()

				web3JSON, err = ioutil.ReadFile(path)
				if err != nil {
					errs = append(errs, err)
				}

				switch {
				case err != nil:
					errs = append(errs, fmt.Errorf("can't decode key %s: %v", path, err))
				case (keyJSON.Address == common.Address{}):
					errs = append(errs, fmt.Errorf("can't decode key %s: missing or zero address", path))
				default:
					accounts = append(accounts, Account{Address: keyJSON.Address, File: fi.Name(), EncryptedKey: string(web3JSON)})
				}
			}
		}

		// Stash a batch or finish up.
		if (len(accounts) == 10000) || (i == len(files) - 1 ) {
			if e := ac.setBatchAccounts(accounts); len(e) != 0 {
				errs = append(errs, e...)
			} else {
				accounts = nil
			}
		}
	}
	return errs
}

func skipKeyFile(fi os.FileInfo) bool {
	// Skip editor backups and UNIX-style hidden files.
	if strings.HasSuffix(fi.Name(), "~") || strings.HasPrefix(fi.Name(), ".") {
		return true
	}
	if strings.HasSuffix(fi.Name(), "accounts.db") {
		return true
	}
	// Skip misc special files, directories (yes, symlinks too).
	if fi.IsDir() || fi.Mode()&os.ModeType != 0 {
		return true
	}
	return false
}

func bytesToAccount(bs []byte) Account {
	var a Account
	if e := bson.Unmarshal(bs, &a); e != nil {
		panic(e)
	}
	return a
}

func accountToBytes(account Account) []byte {
	b, e := bson.Marshal(account)
	if e != nil {
		panic(e)
	}
	return b
}