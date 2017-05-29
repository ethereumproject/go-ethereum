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
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/ethereumproject/go-ethereum/common"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
)

var (
	cachetestDir, _   = filepath.Abs(filepath.Join("testdata", "keystore"))
	cachetestAccounts = []Account{
		{
			Address: common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			File:    filepath.Join(cachetestDir, "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
		},
		{
			Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			File:    filepath.Join(cachetestDir, "aaa"),
		},
		{
			Address: common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			File:    filepath.Join(cachetestDir, "zzz"),
		},
	}
)

func TestWatchNewFile(t *testing.T) {
	t.Parallel()

	dir, am := tmpManager(t)
	defer os.RemoveAll(dir)

	// Ensure the watcher is started before adding any files.
	am.Accounts()
	time.Sleep(5 * time.Second)
	if w := am.ac.getWatcher(); !w.running {
		t.Fatalf("watcher not running after %v: %v", 5*time.Second, spew.Sdump(w))
	}

	// Move in the files.
	wantAccounts := make([]Account, len(cachetestAccounts))
	for i := range cachetestAccounts {
		a := cachetestAccounts[i]
		a.File = filepath.Join(dir, filepath.Base(a.File)) // rename test file to base from temp dir
		wantAccounts[i] = a
		data, err := ioutil.ReadFile(cachetestAccounts[i].File) // but we still have to read from original file path
		if err != nil {
			t.Fatal(err)
		}
		// write to temp dir
		if err := ioutil.WriteFile(a.File, data, 0666); err != nil {
			t.Fatal(err)
		}
	}
	sort.Sort(accountsByFile(wantAccounts))

	// am should see the accounts.
	var list []Account
	for d := 200 * time.Millisecond; d < 5*time.Second; d *= 2 {
		list = am.Accounts()
		if reflect.DeepEqual(list, wantAccounts) {
			return
		}
		time.Sleep(d)
	}
	t.Errorf("got %s, want %s", spew.Sdump(list), spew.Sdump(wantAccounts))
}

func TestWatchNoDir(t *testing.T) {
	t.Parallel()

	// Create am but not the directory that it watches.
	rand.Seed(time.Now().UnixNano())
	rp := fmt.Sprintf("eth-keystore-watch-test-%d-%d", os.Getpid(), rand.Int())
	dir, e := ioutil.TempDir("", rp)
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)

	am, err := NewManager(dir, LightScryptN, LightScryptP, false)
	if err != nil {
		t.Fatal(err)
	}

	list := am.Accounts()
	if len(list) > 0 {
		t.Error("initial account list not empty:", list)
	}
	time.Sleep(5 * time.Second)
	if w := am.ac.getWatcher(); !w.running {
		t.Fatalf("watcher not running after %v: %v", 5*time.Second, spew.Sdump(w))
	}

	// Create the directory and copy a key file into it.
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "aaa")
	// This filepath-ing is redundant but ensure parallel tests don't fuck each other up... I think.
	data, err := ioutil.ReadFile(filepath.Join(cachetestDir, filepath.Base(cachetestAccounts[0].File)))
	if err != nil {
		t.Fatal(err)
	}
	ff, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ff.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	ff.Close()

	// am should see the account.
	a := cachetestAccounts[0]
	a.File = file
	wantAccounts := []Account{a}
	var gotAccounts []Account
	for d := 500 * time.Millisecond; d <= 2*minReloadInterval; d *= 2 {
		gotAccounts = am.Accounts()
		if reflect.DeepEqual(gotAccounts, wantAccounts) {
			return
		}
		time.Sleep(d)
	}
	t.Errorf("account watcher never saw changes: got: %v, want: %v", spew.Sdump(gotAccounts), spew.Sdump(wantAccounts))
}

func TestCacheInitialReload(t *testing.T) {

	cache := newAddrCache(cachetestDir)

	accounts := cache.accounts()

	if !reflect.DeepEqual(accounts, cachetestAccounts) {
		t.Errorf("got initial accounts: %swant %s", spew.Sdump(accounts), spew.Sdump(cachetestAccounts))
	}
}

func TestCacheAddDeleteOrder(t *testing.T) {
	cache := newAddrCache("testdata/no-such-dir")
	defer os.RemoveAll("testdata/no-such-dir")
	cache.watcher.running = true // prevent unexpected reloads

	accounts := []Account{
		{
			Address: common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			File:    "-309830980",
		},
		{
			Address: common.HexToAddress("2cac1adea150210703ba75ed097ddfe24e14f213"),
			File:    "ggg",
		},
		{
			Address: common.HexToAddress("8bda78331c916a08481428e4b07c96d3e916d165"),
			File:    "zzzzzz-the-very-last-one.keyXXX",
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			File:    "SOMETHING.key",
		},
		{
			Address: common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			File:    "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8",
		},
		{
			Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			File:    "aaa",
		},
		{
			Address: common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			File:    "zzz",
		},
	}
	for _, a := range accounts {
		cache.add(a)
	}
	// Add some of them twice to check that they don't get reinserted.
	cache.add(accounts[0])
	cache.add(accounts[2])

	// Check that the account list is sorted by filename.
	wantAccounts := make([]Account, len(accounts))
	copy(wantAccounts, accounts)
	sort.Sort(accountsByFile(wantAccounts))
	list := cache.accounts()
	if !reflect.DeepEqual(list, wantAccounts) {
		t.Fatalf("got accounts: %s\nwant %s", spew.Sdump(accounts), spew.Sdump(wantAccounts))
	}
	for _, a := range accounts {
		if !cache.hasAddress(a.Address) {
			t.Errorf("expected hasAccount(%x) to return true", a.Address)
		}
	}
	if cache.hasAddress(common.HexToAddress("fd9bd350f08ee3c0c19b85a8e16114a11a60aa4e")) {
		t.Errorf("expected hasAccount(%x) to return false", common.HexToAddress("fd9bd350f08ee3c0c19b85a8e16114a11a60aa4e"))
	}

	// Delete a few keys from the cache.
	for i := 0; i < len(accounts); i += 2 {
		cache.delete(wantAccounts[i])
	}
	cache.delete(Account{Address: common.HexToAddress("fd9bd350f08ee3c0c19b85a8e16114a11a60aa4e"), File: "something"})

	// Check content again after deletion.
	wantAccountsAfterDelete := []Account{
		wantAccounts[1],
		wantAccounts[3],
		wantAccounts[5],
	}
	list = cache.accounts()
	if !reflect.DeepEqual(list, wantAccountsAfterDelete) {
		t.Fatalf("got accounts after delete: %s\nwant %s", spew.Sdump(list), spew.Sdump(wantAccountsAfterDelete))
	}
	for _, a := range wantAccountsAfterDelete {
		if !cache.hasAddress(a.Address) {
			t.Errorf("expected hasAccount(%x) to return true", a.Address)
		}
	}
	if cache.hasAddress(wantAccounts[0].Address) {
		t.Errorf("expected hasAccount(%x) to return false", wantAccounts[0].Address)
	}
}

func TestCacheFind(t *testing.T) {
	dir := filepath.Join("testdata", "dir")
	defer os.RemoveAll(dir)
	cache := newAddrCache(dir)
	cache.watcher.running = true // prevent unexpected reloads

	accounts := []Account{
		{
			Address: common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			File:    filepath.Join(dir, "a.key"),
		},
		{
			Address: common.HexToAddress("2cac1adea150210703ba75ed097ddfe24e14f213"),
			File:    filepath.Join(dir, "b.key"),
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			File:    filepath.Join(dir, "c.key"),
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			File:    filepath.Join(dir, "c2.key"),
		},
	}
	for _, a := range accounts {
		cache.add(a)
	}

	if lca := cache.accounts(); len(lca) != len(accounts) {
		t.Fatalf("wrong number of accounts, got: %v, want: %v", len(lca), len(accounts))
	}

	if !reflect.DeepEqual(cache.accounts(), accounts) {
		t.Fatalf("not matching initial accounts: got %v, want: %v", spew.Sdump(cache.accounts()), spew.Sdump(accounts))
	}

	nomatchAccount := Account{
		Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
		File:    filepath.Join(dir, "something"),
	}
	tests := []struct {
		Query      Account
		WantResult Account
		WantError  error
	}{
		// by address
		{Query: Account{Address: accounts[0].Address}, WantResult: accounts[0]},
		// by file
		{Query: Account{File: accounts[0].File}, WantResult: accounts[0]},
		// by basename
		{Query: Account{File: filepath.Base(accounts[0].File)}, WantResult: accounts[0]},
		// by file and address
		{Query: accounts[0], WantResult: accounts[0]},
		// ambiguous address, tie resolved by file
		{Query: accounts[2], WantResult: accounts[2]},
		// ambiguous address error
		{
			Query: Account{Address: accounts[2].Address},
			WantError: &AmbiguousAddrError{
				Addr:    accounts[2].Address,
				Matches: []Account{accounts[2], accounts[3]},
			},
		},
		// no match error
		{Query: nomatchAccount, WantError: ErrNoMatch},
		{Query: Account{File: nomatchAccount.File}, WantError: ErrNoMatch},
		{Query: Account{File: filepath.Base(nomatchAccount.File)}, WantError: ErrNoMatch},
		{Query: Account{Address: nomatchAccount.Address}, WantError: ErrNoMatch},
	}
	for i, test := range tests {
		a, err := cache.find(test.Query)
		if !reflect.DeepEqual(err, test.WantError) {
			t.Errorf("test %d: error mismatch for query %v\ngot %q\nwant %q", i, test.Query, err, test.WantError)
			continue
		}
		if a != test.WantResult {
			t.Errorf("test %d: result mismatch for query %v\ngot %v\nwant %v", i, test.Query, a, test.WantResult)
			continue
		}
	}
}

func TestAccountCache_WatchRemove(t *testing.T) {
	t.Parallel()

	// setup temp dir
	rp := fmt.Sprintf("eth-cacheremove-watch-test-%d-%d", os.Getpid(), rand.Int())
	tmpDir, e := ioutil.TempDir("", rp)
	if e != nil {
		t.Fatalf("create temp dir: %v", e)
	}
	defer os.RemoveAll(tmpDir)

	// copy 3 test account files into temp dir
	wantAccounts := make([]Account, len(cachetestAccounts))
	for i := range cachetestAccounts {
		a := cachetestAccounts[i]
		a.File = filepath.Join(tmpDir, filepath.Base(a.File))
		wantAccounts[i] = a
		data, err := ioutil.ReadFile(cachetestAccounts[i].File)
		if err != nil {
			t.Fatal(err)
		}
		ff, err := os.Create(a.File)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ff.Write(data)
		if err != nil {
			t.Fatal(err)
		}
		ff.Close()
	}

	// make manager in temp dir
	ma, e := NewManager(tmpDir, veryLightScryptN, veryLightScryptP, false)
	if e != nil {
		t.Errorf("create manager in temp dir: %v", e)
	}

	// test manager has all accounts
	initAccs := ma.Accounts()
	if !reflect.DeepEqual(initAccs, wantAccounts) {
		t.Errorf("got %v, want: %v", spew.Sdump(initAccs), spew.Sdump(wantAccounts))
	}
	time.Sleep(minReloadInterval)

	// test watcher is watching
	if w := ma.ac.getWatcher(); !w.running {
		t.Errorf("watcher not running")
	}

	// remove file
	rmPath := filepath.Join(tmpDir, filepath.Base(wantAccounts[0].File))
	if e := os.Remove(rmPath); e != nil {
		t.Fatalf("removing key file: %v", e)
	}
	// ensure it's gone
	if _, e := os.Stat(rmPath); e == nil {
		t.Fatalf("removed file not actually rm'd")
	}

	// test manager does not have account
	wantAccounts = wantAccounts[1:]
	if len(wantAccounts) != 2 {
		t.Errorf("dummy")
	}

	gotAccounts := []Account{}
	for d := 500 * time.Millisecond; d < 5*time.Second; d *= 2 {
		gotAccounts = ma.Accounts()
		// If it's ever all the same, we're good. Exit with aplomb.
		if reflect.DeepEqual(gotAccounts, wantAccounts) {
			return
		}
		time.Sleep(d)
	}
	t.Errorf("got: %v, want: %v", spew.Sdump(gotAccounts), spew.Sdump(wantAccounts))
}

func TestCacheFilePath(t *testing.T) {
	dir := filepath.Join("testdata", "keystore")
	dir, _ = filepath.Abs(dir)
	cache := newAddrCache(dir)

	accs := cache.accounts()

	for _, a := range accs {
		if !filepath.IsAbs(a.File) {
			t.Errorf("wanted absolute filepath, got: %v", a.File)
		}
	}

	cache.close()
}
