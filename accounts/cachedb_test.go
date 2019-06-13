package accounts

import (
	"io/ioutil"
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
	cachedbtestAccounts = []Account{
		{
			Address:      common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			File:         "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8",
			EncryptedKey: "{\"address\":\"7ef5a6135f1fd6a02593eedc869c6d41d934aef8\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"1d0839166e7a15b9c1333fc865d69858b22df26815ccf601b28219b6192974e1\",\"cipherparams\":{\"iv\":\"8df6caa7ff1b00c4e871f002cb7921ed\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":8,\"p\":16,\"r\":8,\"salt\":\"e5e6ef3f4ea695f496b643ebd3f75c0aa58ef4070e90c80c5d3fb0241bf1595c\"},\"mac\":\"6d16dfde774845e4585357f24bce530528bc69f4f84e1e22880d34fa45c273e5\"},\"id\":\"950077c7-71e3-4c44-a4a1-143919141ed4\",\"version\":3}",
		},
		{
			Address:      common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			File:         "aaa",
			EncryptedKey: "{\"address\":\"f466859ead1932d743d622cb74fc058882e8648a\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"cb664472deacb41a2e995fa7f96fe29ce744471deb8d146a0e43c7898c9ddd4d\",\"cipherparams\":{\"iv\":\"dfd9ee70812add5f4b8f89d0811c9158\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":8,\"p\":16,\"r\":8,\"salt\":\"0d6769bf016d45c479213990d6a08d938469c4adad8a02ce507b4a4e7b7739f1\"},\"mac\":\"bac9af994b15a45dd39669fc66f9aa8a3b9dd8c22cb16e4d8d7ea089d0f1a1a9\"},\"id\":\"472e8b3d-afb6-45b5-8111-72c89895099a\",\"version\":3}",
		},
		{
			Address:      common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			File:         "zzz",
			EncryptedKey: "{\"address\":\"289d485d9771714cce91d3393d764e1311907acc\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"faf32ca89d286b107f5e6d842802e05263c49b78d46eac74e6109e9a963378ab\",\"cipherparams\":{\"iv\":\"558833eec4a665a8c55608d7d503407d\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":8,\"p\":16,\"r\":8,\"salt\":\"d571fff447ffb24314f9513f5160246f09997b857ac71348b73e785aab40dc04\"},\"mac\":\"21edb85ff7d0dab1767b9bf498f2c3cb7be7609490756bd32300bb213b59effe\"},\"id\":\"3279afcf-55ba-43ff-8997-02dcc46a6525\",\"version\":3}",
		},
	}
)

func TestCacheInitialReload_CacheDB(t *testing.T) {
	cache := newCacheDB(cachetestDir)
	cache.Syncfs2db(time.Now())
	defer cache.close()

	accounts := cache.accounts()

	if !reflect.DeepEqual(accounts, cachedbtestAccounts) {
		t.Errorf("got: %v, want %v", spew.Sdump(accounts), spew.Sdump(cachedbtestAccounts))
	}
}

func TestCacheAddDeleteOrder_CacheDB(t *testing.T) {
	cache := newCacheDB("testdata/no-such-dir")
	defer cache.close()
	defer os.RemoveAll("testdata/no-such-dir")

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

func TestCacheFind_CacheFind(t *testing.T) {
	dir := filepath.Join("testdata", "dir")
	cache := newCacheDB(dir)
	defer cache.close()
	defer os.RemoveAll(dir)

	accounts := []Account{
		{
			Address: common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			File:    "a.key",
		},
		{
			Address: common.HexToAddress("2cac1adea150210703ba75ed097ddfe24e14f213"),
			File:    "b.key",
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			File:    "c.key",
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			File:    "c2.key",
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

func TestAccountCache_CacheDB_SyncFS2DB(t *testing.T) {
	// setup temp dir
	tmpDir, e := ioutil.TempDir("", "cachedb-remover-test")
	if e != nil {
		t.Fatalf("create temp dir: %v", e)
	}
	defer os.RemoveAll(tmpDir)

	// make manager in temp dir
	ma, e := NewManager(tmpDir, veryLightScryptN, veryLightScryptP, true)
	if e != nil {
		t.Errorf("create manager in temp dir: %v", e)
	}
	// test manager has no accounts
	initAccs := ma.Accounts()
	if !reflect.DeepEqual(initAccs, []Account{}) {
		t.Errorf("got %v, want: %v", spew.Sdump(initAccs), spew.Sdump([]Account{}))
	}
	// close manager
	ma.ac.close()
	ma = nil

	// copy 3 key files to temp dir
	for _, acc := range cachedbtestAccounts {
		data, err := ioutil.ReadFile(filepath.Join(cachetestDir, acc.File))
		if err != nil {
			t.Fatal(err)
		}
		ff, err := os.Create(filepath.Join(tmpDir, acc.File))
		if err != nil {
			t.Fatal(err)
		}
		_, err = ff.Write(data)
		if err != nil {
			t.Fatal(err)
		}
		ff.Close()
	}

	// restart manager in temp dir
	ma, e = NewManager(tmpDir, veryLightScryptN, veryLightScryptP, true)
	if e != nil {
		t.Errorf("create manager in temp dir: %v", e)
	}

	ma.ac.Syncfs2db(time.Now())

	// test manager has 3 accounts
	initAccs = ma.Accounts()
	if !reflect.DeepEqual(initAccs, cachedbtestAccounts) {
		t.Errorf("got %v, want: %v", spew.Sdump(initAccs), spew.Sdump(cachedbtestAccounts))
	}

	// close manager
	ma.ac.close()
	ma = nil

	// remove 1 key file from temp dir
	rmPath := filepath.Join(tmpDir, cachedbtestAccounts[0].File)
	if e := os.Remove(rmPath); e != nil {
		t.Fatalf("removing key file: %v", e)
	}

	// restart manager in temp dir
	ma, e = NewManager(tmpDir, veryLightScryptN, veryLightScryptP, true)
	if e != nil {
		t.Errorf("create manager in temp dir: %v", e)
	}
	ma.ac.Syncfs2db(time.Now())

	// test manager has 2 accounts
	initAccs = ma.Accounts()
	if !reflect.DeepEqual(initAccs, cachedbtestAccounts[1:]) {
		t.Errorf("got %v, want: %v", spew.Sdump(initAccs), spew.Sdump(cachedbtestAccounts[1:]))
	}
	ma.ac.close()
}

func TestCacheDBFilePath(t *testing.T) {
	dir := filepath.Join("testdata", "keystore")
	dir, _ = filepath.Abs(dir)
	cache := newCacheDB(dir)
	defer cache.close()

	accs := cache.accounts()

	for _, a := range accs {
		if filepath.IsAbs(a.File) {
			t.Errorf("wanted relative filepath (wanted basename), got: %v", a.File)
		}
	}
}
