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
	//cachedbtestDir, _   = filepath.Abs(filepath.Join("testdata", "keystore"))
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

// Setup/teardown.
func TestMain(m *testing.M) {
	m.Run()
}

func tmpManager_CacheDB(t *testing.T) (string, *Manager) {
	dir, err := ioutil.TempDir("", "eth-manager-test")
	if err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(dir, veryLightScryptN, veryLightScryptP, true)
	if err != nil {
		t.Fatal(err)
	}
	return dir, m
}

func TestWatchNewFile_CacheDB(t *testing.T) {
	t.Parallel()

	dir, am := tmpManager_CacheDB(t)
	defer os.RemoveAll(dir)

	// Ensure the watcher is started before adding any files.
	am.Accounts()
	time.Sleep(5 * time.Second)
	if w := am.ac.getWatcher(); !w.running {
		t.Fatalf("watcher not running after %v: %v", 5*time.Second, spew.Sdump(w))
	}

	initAccs := am.Accounts()
	if len(initAccs) != 0 {
		t.Fatalf("initial accounts not empty: %v", len(initAccs))
	}
	initAccs = nil

	// Move in the files.
	wantAccounts := make([]Account, len(cachedbtestAccounts))
	for i := range cachedbtestAccounts {
		a := cachedbtestAccounts[i]
		//a.File = filepath.Join(dir, filepath.Base(a.File)) // rename test file to base from temp dir
		wantAccounts[i] = a
		data, err := ioutil.ReadFile(filepath.Join(cachetestDir, cachedbtestAccounts[i].File)) // but we still have to read from original file path
		if err != nil {
			t.Fatal(err)
		}
		// write to temp dir
		if err := ioutil.WriteFile(filepath.Join(dir, a.File), data, 0666); err != nil {
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

func TestWatchNoDir_CacheDB(t *testing.T) {
	t.Parallel()

	// Create am but not the directory that it watches.
	rand.Seed(time.Now().UnixNano())
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("eth-keystore-watch-test-%d-%d", os.Getpid(), rand.Int()))
	am, err := NewManager(dir, LightScryptN, LightScryptP, true)
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

	file := filepath.Join(dir, cachedbtestAccounts[0].File)
	data, err := ioutil.ReadFile(filepath.Join(cachetestDir, cachedbtestAccounts[0].File))
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
	wantAccounts := []Account{cachedbtestAccounts[0]}
	//wantAccounts[0].File = file
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

func TestCacheInitialReload_CacheDB(t *testing.T) {

	cache := newCacheDB(cachetestDir)

	accounts := cache.accounts()

	if !reflect.DeepEqual(accounts, cachedbtestAccounts) {
		t.Errorf("got initial accounts: %swant %s", spew.Sdump(accounts), spew.Sdump(cachedbtestAccounts))
	}
}

func TestCacheAddDeleteOrder_CacheDB(t *testing.T) {
	cache := newCacheDB("testdata/no-such-dir")
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

func TestCacheFind_CacheFind(t *testing.T) {
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
	// test manager has 3 accounts
	initAccs = ma.Accounts()
	if !reflect.DeepEqual(initAccs, cachedbtestAccounts[1:]) {
		t.Errorf("got %v, want: %v", spew.Sdump(initAccs), spew.Sdump(cachedbtestAccounts[1:]))
	}
}

func TestAccountCache_CacheDB_WatchRemove(t *testing.T) {
	// setup temp dir
	tmpDir, e := ioutil.TempDir("", "cachedb-remover-test")
	if e != nil {
		t.Fatalf("create temp dir: %v", e)
	}
	defer os.RemoveAll(tmpDir)

	// copy 3 test account files into temp dir
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

	// make manager in temp dir
	ma, e := NewManager(tmpDir, veryLightScryptN, veryLightScryptP, true)
	if e != nil {
		t.Errorf("create manager in temp dir: %v", e)
	}

	// test manager has all accounts
	initAccs := ma.Accounts()
	if !reflect.DeepEqual(initAccs, cachedbtestAccounts) {
		t.Errorf("got %v, want: %v", spew.Sdump(initAccs), spew.Sdump(cachedbtestAccounts))
	}
	time.Sleep(2 * time.Second)

	// test watcher is watching
	if w := ma.ac.getWatcher(); !w.running {
		t.Errorf("watcher not running")
	}

	// remove file
	rmPath := filepath.Join(tmpDir, cachedbtestAccounts[0].File)
	if e := os.Remove(rmPath); e != nil {
		t.Fatalf("removing key file: %v", e)
	}
	// ensure it's gone
	if _, e := os.Stat(filepath.Join(tmpDir, cachedbtestAccounts[0].File)); e == nil {
		t.Fatalf("removed file not actually rm'd")
	}

	// test manager does not have account
	wantAccounts := cachedbtestAccounts[1:]
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