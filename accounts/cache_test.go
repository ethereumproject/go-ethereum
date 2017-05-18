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
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereumproject/go-ethereum/common"
)

var (
	cachetestDir, _   = filepath.Abs(filepath.Join("testdata", "keystore"))
	cachetestAccounts = []Account{
		{
			Address: common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			File:    "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8",
			EncryptedKey: "{\"address\":\"7ef5a6135f1fd6a02593eedc869c6d41d934aef8\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"1d0839166e7a15b9c1333fc865d69858b22df26815ccf601b28219b6192974e1\",\"cipherparams\":{\"iv\":\"8df6caa7ff1b00c4e871f002cb7921ed\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":8,\"p\":16,\"r\":8,\"salt\":\"e5e6ef3f4ea695f496b643ebd3f75c0aa58ef4070e90c80c5d3fb0241bf1595c\"},\"mac\":\"6d16dfde774845e4585357f24bce530528bc69f4f84e1e22880d34fa45c273e5\"},\"id\":\"950077c7-71e3-4c44-a4a1-143919141ed4\",\"version\":3}",
		},
		{
			Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			File:    "aaa",
			EncryptedKey: "{\"address\":\"f466859ead1932d743d622cb74fc058882e8648a\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"cb664472deacb41a2e995fa7f96fe29ce744471deb8d146a0e43c7898c9ddd4d\",\"cipherparams\":{\"iv\":\"dfd9ee70812add5f4b8f89d0811c9158\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":8,\"p\":16,\"r\":8,\"salt\":\"0d6769bf016d45c479213990d6a08d938469c4adad8a02ce507b4a4e7b7739f1\"},\"mac\":\"bac9af994b15a45dd39669fc66f9aa8a3b9dd8c22cb16e4d8d7ea089d0f1a1a9\"},\"id\":\"472e8b3d-afb6-45b5-8111-72c89895099a\",\"version\":3}",
		},
		{
			Address: common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			File:    "zzz",
			EncryptedKey: "{\"address\":\"289d485d9771714cce91d3393d764e1311907acc\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"faf32ca89d286b107f5e6d842802e05263c49b78d46eac74e6109e9a963378ab\",\"cipherparams\":{\"iv\":\"558833eec4a665a8c55608d7d503407d\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":8,\"p\":16,\"r\":8,\"salt\":\"d571fff447ffb24314f9513f5160246f09997b857ac71348b73e785aab40dc04\"},\"mac\":\"21edb85ff7d0dab1767b9bf498f2c3cb7be7609490756bd32300bb213b59effe\"},\"id\":\"3279afcf-55ba-43ff-8997-02dcc46a6525\",\"version\":3}",
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
	if !am.cache.watcher.running {
		t.Fatalf("watcher not running after %v: %v", 5 * time.Second, spew.Sdump(am.cache.watcher))
	}

	// Move in the files.
	wantAccounts := make([]Account, len(cachetestAccounts))
	for i, a := range cachetestAccounts {
		p := filepath.Join(dir, a.File)
		a.File = p
			wantAccounts[i] = a
		data, err := ioutil.ReadFile(filepath.Join(cachetestDir, filepath.Base(a.File)))
		if err != nil {
			t.Fatal(err)
		}
		if err := ioutil.WriteFile(p, data, 0666); err != nil {
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
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("eth-keystore-watch-test-%d-%d", os.Getpid(), rand.Int()))
	am, err := NewManager(dir, LightScryptN, LightScryptP)
	if err != nil {
		t.Fatal(err)
	}

	list := am.Accounts()
	if len(list) > 0 {
		t.Error("initial account list not empty:", list)
	}
	time.Sleep(5 * time.Second)
	if !am.cache.watcher.running {
		t.Fatalf("watcher not running after %v: %v", 5 * time.Second, spew.Sdump(am.cache.watcher))
	}

	// Create the directory and copy a key file into it.
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "aaa")
	data, err := ioutil.ReadFile(filepath.Join(cachetestDir, cachetestAccounts[0].File))
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
	wantAccounts := []Account{cachetestAccounts[0]}
	wantAccounts[0].File = file
	var seen, gotAccounts = make(map[time.Duration]bool), make(map[time.Duration][]Account)
	for d := 500 * time.Millisecond; d < 5*time.Second; d *= 2 {
		list = am.Accounts()
		seen[d] = false
		if reflect.DeepEqual(list, wantAccounts) {
			seen[d] = true
		} else {
		}
		gotAccounts[d] = list
		time.Sleep(d)
	}
	numSaw := 0
	for i, saw := range seen {
		if !saw {
			t.Logf("account watcher DID NOT see changes at %v/%v...", i, 5*time.Second)
		} else {
			t.Logf("account watcher DID see changes at %v/%v...", i, 5*time.Second)
			numSaw++
		}
	}
	if numSaw == 0 {
		t.Errorf("account watcher never saw changes: want: %v, got %v", spew.Sdump(wantAccounts), spew.Sdump(gotAccounts))
	}
}

func TestCacheInitialReload(t *testing.T) {
	cachedbpath := filepath.Join(cachetestDir, "accounts.db")
	os.Remove(cachedbpath)

	cache := newAddrCache(cachetestDir)

	accounts := cache.accounts()

	if !reflect.DeepEqual(accounts, cachetestAccounts) {
		t.Errorf("got initial accounts: %swant %s", spew.Sdump(accounts), spew.Sdump(cachetestAccounts))
	}
	// check cachedb file exists properly
	fi, e := os.Stat(cachedbpath)
	if e != nil {
		t.Errorf("missing cache db: %v", e)
	}
	if fi.Name() != "accounts.db" {
		t.Errorf("wrong db name: want: %v, got %v", "accounts.db", fi.Name())
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
