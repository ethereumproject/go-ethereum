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
)

// Minimum amount of time between cache reloads. This limit applies if the platform does
// not support change notifications. It also applies if the keystore directory does not
// exist yet, the code will attempt to create a watcher at most this often.
const minReloadInterval = 2 * time.Second

var addrBucketName = []byte("byAddr")
var fileBucketName = []byte("byFile")

type accountsByFile []Account

func (as accountsByFile) MarshalJSON() ([]byte, error) {
	type aux struct {
		Address string `json:"address,omitempty"`
		File    string `json:"file,omitempty"`
	}
	var auxs []aux
	for _, a := range as {
		auxs = append(auxs, aux{
			Address: a.Address.Hex(),
			File:    a.File,
		})
	}
	return json.Marshal(auxs)
}

func UnmarshalJSONBytesToAccounts(raw []byte) ([]Account, error) {
	type aux struct {
		Address string `json:"address,omitempty"`
		File    string `json:"file,omitempty"`
	}
	var auxs []aux
	var accs []Account
	if e := json.Unmarshal(raw, &auxs); e != nil {
		return accs, e
	}
	for _, x := range auxs {
		accs = append(accs, Account{
			Address: common.HexToAddress(x.Address),
			File:    x.File,
		})
	}
	return accs, nil
}

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
	bdb, e := bolt.Open(filepath.Join(keydir, "accounts.db"), 0600, nil) // TODO configure more?
	if e != nil {
		panic(e) // FIXME
	}
	//defer bdb.Close()

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
		return nil
	}); e != nil {
		panic(e)
	}

	// Initializes db to match fs.
	if errs := ac.scan(); len(errs) != 0 {
		for _, e := range errs {
			if e != nil {
				glog.V(logger.Error).Infof("error: %v", e)
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

	//ac.mu.Lock()
	//defer ac.mu.Unlock()
	//cpy := make([]Account, len(ac.all))
	//copy(cpy, ac.all)
	//return cpy
}

// note, again, that this return an _slice_
func (ac *addrCache) getCachedAccountsByAddress(addr common.Address) (accounts []Account, err error) {
	err = ac.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(addrBucketName)
		if v := b.Get(addr.Bytes()); v != nil {
			accounts = bytesAccountFilesToAccounts(v)
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

	//ac.mu.Lock()
	//defer ac.mu.Unlock()
	//return len(ac.byAddr[addr]) > 0
}

// add is smart (kind of); it is where adding logic is. it could be other places, but it is here.
// add checks if it _should_ add, and if it should, it does. otherwise it doesn't.
// it makes the assumption that if the account is not cached by file, it won't be listed
// by address either. thusly, when and iff it adds an account to the cache(s), it adds bothly.
func (ac *addrCache) add(newAccount Account) {

	// check cached accounts by file
	exists := false
	if newAccount.File != "" {
		a, e := ac.getCachedAccountByFile(newAccount.File)
		exists = a != Account{} && e == nil
	}

	if !exists {
		ac.db.Update(func(tx *bolt.Tx) error {
			// like the original implementation, we're not going to check for .File == "" or common.Address{

			// since it doesn't exist yet, we know that we don't have to do any fancy appending for addr cache
			b := tx.Bucket(addrBucketName)
			efs := b.Get(newAccount.Address.Bytes())
			if efs == nil {
				b.Put(newAccount.Address.Bytes(), accountsToAccountFilesBytes([]Account{newAccount}))
			}
			efsas := bytesAccountFilesToAccounts(efs)
			hasEvilTwin := false
			for _, a := range efsas {
				if a.File == newAccount.File {
					hasEvilTwin = true
				}
			}
			if !hasEvilTwin {
				efsas = append(efsas, newAccount)
				efsasb := accountsToAccountFilesBytes(efsas)
				b.Put(newAccount.Address.Bytes(), efsasb)
			}
			// hasEvilTwin means file already is accounted for... get it?

			b = tx.Bucket(fileBucketName)
			b.Put([]byte(newAccount.File), accountToBytes(newAccount))

			return nil
		})
	}

	//ac.mu.Lock()
	//defer ac.mu.Unlock()
	//
	//i := sort.Search(len(ac.all), func(i int) bool { return ac.all[i].File >= newAccount.File })
	//if i < len(ac.all) && ac.all[i] == newAccount {
	//	return
	//}
	//// newAccount is not in the cache.
	//ac.all = append(ac.all, Account{})
	//copy(ac.all[i+1:], ac.all[i:])
	//ac.all[i] = newAccount
	//ac.byAddr[newAccount.Address] = append(ac.byAddr[newAccount.Address], newAccount)
}

// note: removed needs to be unique here (i.e. both File and Address must be set).
func (ac *addrCache) delete(removed Account) {

	if as, e := ac.getCachedAccountsByAddress(removed.Address); e == nil {
		if ass := removeAccount(as, removed); len(ass) > 0 {
			ac.db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket(addrBucketName)
				return b.Put(removed.Address.Bytes(), accountsToAccountFilesBytes(ass))
			})
		} else {
			ac.db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket(addrBucketName)
				return b.Delete(removed.Address.Bytes())
			})
		}
	}

	ac.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(fileBucketName)
		return b.Delete([]byte(removed.File))
	})

	//ac.mu.Lock()
	//defer ac.mu.Unlock()
	//ac.all = removeAccount(ac.all, removed)
	//if ba := removeAccount(ac.byAddr[removed.Address], removed); len(ba) == 0 {
	//	delete(ac.byAddr, removed.Address)
	//} else {
	//	ac.byAddr[removed.Address] = ba
	//}
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

	// Limit search to address candidates if possible.
	if (a.Address != common.Address{}) {
		matches, e = ac.getCachedAccountsByAddress(a.Address)
	}

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

	//// Limit search to address candidates if possible.
	//matches := ac.all
	//if (a.Address != common.Address{}) {
	//	matches = ac.byAddr[a.Address]
	//}
	//if a.File != "" {
	//	// If only the basename is specified, complete the path.
	//	if !strings.ContainsRune(a.File, filepath.Separator) {
	//		a.File = filepath.Join(ac.keydir, a.File)
	//	}
	//	for i := range matches {
	//		if matches[i].File == a.File {
	//			return matches[i], nil
	//		}
	//	}
	//	if (a.Address == common.Address{}) {
	//		return Account{}, ErrNoMatch
	//	}
	//}
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

//func (ac *addrCache) maybeReload() {
//	if ac.watcher.running {
//		return // A watcher is running and will keep the cache up-to-date.
//	}
//	if ac.throttle == nil {
//		ac.throttle = time.NewTimer(0)
//	} else {
//		select {
//		case <-ac.throttle.C:
//		default:
//			return // The cache was reloaded recently.
//		}
//	}
//	ac.watcher.start()
//	ac.reload()
//	ac.throttle.Reset(minReloadInterval)
//
//	//ac.mu.Lock()
//	//defer ac.mu.Unlock()
//	//if ac.watcher.running {
//	//	return // A watcher is running and will keep the cache up-to-date.
//	//}
//	//if ac.throttle == nil {
//	//	ac.throttle = time.NewTimer(0)
//	//} else {
//	//	select {
//	//	case <-ac.throttle.C:
//	//	default:
//	//		return // The cache was reloaded recently.
//	//	}
//	//}
//	//ac.watcher.start()
//	//ac.reload()
//	//ac.throttle.Reset(minReloadInterval)
//}

func (ac *addrCache) close() {

	ac.watcher.close()
	if ac.throttle != nil {
		ac.throttle.Stop()
	}
	ac.db.Close()

	//ac.mu.Lock()
	//ac.watcher.close()
	//if ac.throttle != nil {
	//	ac.throttle.Stop()
	//}
	//ac.mu.Unlock()
}

// set is used by the fs watcher to update the cache from a given file path.
// it has some logic;
// -- it will _overwrite_ any existing cache entry by file and addr, making it useful for CREATE and UPDATE
func (ac *addrCache) setViaFile(path string) error {
	var (
		buf     = new(bufio.Reader)
		acc     Account
		accs    []Account
		keyJSON struct {
			Address common.Address `json:"address"`
		}
	)
	fi, e := os.Stat(path)
	if e != nil {
		return e
	}
	if skipKeyFile(fi) {
		return nil
	}
	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	buf.Reset(fd)
	// Parse the address.
	keyJSON.Address = common.Address{}
	err = json.NewDecoder(buf).Decode(&keyJSON)
	switch {
	case err != nil:
		return fmt.Errorf("can't decode key %s: %v", path, err)
	case (keyJSON.Address == common.Address{}):
		return fmt.Errorf("can't decode key %s: missing or zero address", path)
	default:
		acc = Account{Address: keyJSON.Address, File: path}
	}

	ab := accountToBytes(acc)

	return ac.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(fileBucketName)
		if e := b.Put([]byte(path), ab); e != nil {
			return e
		}
		b = tx.Bucket(addrBucketName)

		// get existing accounts by address, if any
		xasb := b.Get(keyJSON.Address.Bytes())
		if xasb != nil {
			xaccs := bytesAccountFilesToAccounts(xasb)
			accs = append(xaccs, acc)
		} else {
			accs = append(accs, acc)
		}
		asb := accountsToAccountFilesBytes(accs)

		return b.Put(keyJSON.Address.Bytes(), asb)
	})
}

// remove is used by the fs watcher to update the cache from a given path.
func (ac *addrCache) removeViaFile(path string) error {

	var acc Account

	if e := ac.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(fileBucketName)
		ab := b.Get([]byte(path))
		if ab == nil {
			return ErrNoMatch
		}
		acc = bytesToAccount(ab)
		if e := b.Delete([]byte(path)); e != nil {
			return e
		}

		b = tx.Bucket(addrBucketName)
		asb := b.Get(acc.Address.Bytes())
		if asb == nil {
			return nil
		}
		xacs := bytesAccountFilesToAccounts(asb)
		accs := removeAccount(xacs, acc)
		if len(accs) > 0 {
			accsb := accountsToAccountFilesBytes(accs)
			return b.Put([]byte(acc.Address.Bytes()), accsb)
		} else {
			return b.Delete(acc.Address.Bytes())
		}
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
			// TODO: recursively watch tree? not so important, i think
		}



		// TODO: don't ignore the returned errors
		switch ev.Event() {
		case notify.Create:
			glog.V(logger.Debug).Infof("reloading create event: %v", ev.Event())
			ac.setViaFile(p)
		case notify.Rename:
			glog.V(logger.Debug).Infof("reloading rename event (doing nothing): %v", ev.Event())
			// TODO: do something
		case notify.Remove:
			glog.V(logger.Debug).Infof("reloading remove event: %v", ev.Event())
			ac.removeViaFile(p)
		case notify.Write:
			glog.V(logger.Debug).Infof("reloading write event: %v", ev.Event())
			ac.setViaFile(p)
		default:
			// do nothing
			//glog.V(logger.Debug).Infof("reloading default event: %v", ev.Event())
			//ac.setViaFile(p)
		}
	}

	//accounts, err := ac.scan()
	//if err != nil && glog.V(logger.Debug) {
	//	glog.Errorf("can't load keys: %v", err)
	//}
	//ac.all = accounts
	//sort.Sort(ac.all)
	//for k := range ac.byAddr {
	//	delete(ac.byAddr, k)
	//}
	//for _, a := range accounts {
	//	ac.byAddr[a.Address] = append(ac.byAddr[a.Address], a)
	//}
	//glog.V(logger.Debug).Infof("reloaded keys, cache has %d accounts", len(ac.all))
	return []notify.EventInfo{}
}

// scan is designed to *initialize* the cachedb, reading all files in keydir/* and adding them to the respective
// buckets
//func (ac *addrCache) scan() ([]Account, error) {
func (ac *addrCache) scan() (errs []error) {
	files, err := ioutil.ReadDir(ac.keydir)
	if err != nil {
		return append(errs, err)
	}

	// first sync fs -> cachedb, update all accounts in cache from fs
	//var (
	//	buf     = new(bufio.Reader)
	//	addrs []Account
	//	keyJSON struct {
	//		Address common.Address `json:"address"`
	//	}
	//)
	for _, fi := range files {

		path := filepath.Join(ac.keydir, fi.Name())
		if e := ac.setViaFile(path); e != nil {
			errs = append(errs, e)
		} // TODO: use bolt Batches
		//if skipKeyFile(fi) {
		//	glog.V(logger.Detail).Infof("ignoring file %s", path)
		//	continue
		//}
		//fd, err := os.Open(path)
		//if err != nil {
		//	glog.V(logger.Detail).Infoln(err)
		//	continue
		//}
		//buf.Reset(fd)
		//// Parse the address.
		//keyJSON.Address = common.Address{}
		//err = json.NewDecoder(buf).Decode(&keyJSON)
		//switch {
		//case err != nil:
		//	glog.V(logger.Debug).Infof("can't decode key %s: %v", path, err)
		//case (keyJSON.Address == common.Address{}):
		//	glog.V(logger.Debug).Infof("can't decode key %s: missing or zero address", path)
		//default:
		//	addrs = append(addrs, Account{Address: keyJSON.Address, File: path})
		//}
		//fd.Close()
	}

	// first sync cachedb -> fs, are there any cached accounts that don't exist in the fs anymore?
	e := ac.db.Update(func(tx *bolt.Tx) error {
		fb := tx.Bucket(fileBucketName)
		ab := tx.Bucket(addrBucketName)
		c := fb.Cursor()

		var removedFilesB [][]byte
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			_, e := os.Stat(string(k))
			if e != nil && e == os.ErrNotExist {

				removedFilesB = append(removedFilesB, k)

				a := bytesToAccount(k)
				if acsb := ab.Get(a.Address.Bytes()); acsb != nil {
					accs := bytesAccountFilesToAccounts(acsb)
					xacs := removeAccount(accs, a)
					if len(xacs) == 0 {
						ab.Delete(a.Address.Bytes())
					} else {
						ab.Put(a.Address.Bytes(), accountsToAccountFilesBytes(xacs))
					}
				}
			}
			if e != nil {
				errs = append(errs, err)
				continue
			}
		}

		for _, rb := range removedFilesB {
			if e := fb.Delete(rb); e != nil {
				errs = append(errs, e)
			}
		}
		return nil
	})
	errs = append(errs, e)
	return errs
	//return addrs, err
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

func bytesAccountFilesToAccounts(bs []byte) []Account {
	if bs == nil {
		return []Account{}
	}
	as, e := UnmarshalJSONBytesToAccounts(bs)
	if e != nil {
		panic(e)
	}
	return as

}
func bytesToAccount(bs []byte) Account {
	var a Account
	if e := a.UnmarshalJSON(bs); e != nil {
		panic(e)
	}
	//if e := json.Unmarshal(bs, &a); e != nil {
	//	panic(e)
	//}
	return a
}
func accountsToAccountFilesBytes(accounts []Account) []byte {
	as := accountsByFile(accounts)
	b, e := as.MarshalJSON()
	if e != nil {
		panic(e)
	}
	return b
	//var afs []string
	//for _, a := range accounts {
	//	afs = append(afs, a.File)
	//}
	//b, e := json.Marshal(afs)
	//if e != nil {
	//	panic(e)
	//}
	//return b
	//js := `[`
	//for i, a := range accounts {
	//	js += `{"address":"`+a.Address.Hex()+`","file":"`+a.File+`"}`
	//	if i != (len(accounts) - 1) {
	//		js += `,`
	//	}
	//}
	//js += `]`
	//
	//b := []byte(js)
	//b, e := json.Marshal(accounts)
	//
	//if e != nil {
	//	panic(e)
	//}
	//return b
	//if b, e :=
}
func accountToBytes(account Account) []byte {
	//js := `{"address": "` + account.Address.Hex() + `", "file": " ` + account.File + `"}`
	b, e := account.MarshalJSON()
	if e != nil {
		panic(e)
	}
	//b := []byte(js)
	return b
}
