package accounts

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

//func BenchmarkAccountSignScaling(b *testing.B) {
//	var cases = []struct {
//		dir                  string
//		numKeyFiles          int
//		resetAll, resetCache bool
//	}{
//		// You can use resetCache: false to save some time if you've already run the benchmark.
//		// Note that if you make any changes to the structure of the cachedb you'll need to
//		// dump and reinitialize accounts.db.
//		{dir: "benchmark_keystore100", numKeyFiles: 100, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore100", numKeyFiles: 100, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore500", numKeyFiles: 500, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore500", numKeyFiles: 500, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore1k", numKeyFiles: 1000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore1k", numKeyFiles: 1000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore5k", numKeyFiles: 5000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore5k", numKeyFiles: 5000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore10k", numKeyFiles: 10000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore10k", numKeyFiles: 10000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore20k", numKeyFiles: 20000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore20k", numKeyFiles: 20000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore100k", numKeyFiles: 100000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore100k", numKeyFiles: 100000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore500k", numKeyFiles: 500000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore500k", numKeyFiles: 500000, resetAll: false, resetCache: false},
//	}
//
//	for _, c := range cases {
//		b.Run(fmt.Sprintf("KeyFiles#:%v, CacheFromScratch:%v", c.numKeyFiles, c.resetCache), func(b *testing.B) {
//			am := setupBenchmarkAccountFlowFast(filepath.Join("testdata", c.dir), c.numKeyFiles, c.resetAll, c.resetCache, b)
//			benchmarkAccountSignFast(am.keyStore.baseDir, am, c.numKeyFiles-1, b)
//			am = nil
//		})
//	}
//}
//
//func BenchmarkAccountFlowScaling(b *testing.B) {
//	var cases = []struct {
//		dir                  string
//		numKeyFiles          int
//		resetAll, resetCache bool
//	}{
//		{dir: "benchmark_keystore100", numKeyFiles: 100, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore100", numKeyFiles: 100, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore500", numKeyFiles: 500, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore500", numKeyFiles: 500, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore1k", numKeyFiles: 1000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore1k", numKeyFiles: 1000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore5k", numKeyFiles: 5000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore5k", numKeyFiles: 5000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore10k", numKeyFiles: 10000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore10k", numKeyFiles: 10000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore20k", numKeyFiles: 20000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore20k", numKeyFiles: 20000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore100k", numKeyFiles: 100000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore100k", numKeyFiles: 100000, resetAll: false, resetCache: false},
//		{dir: "benchmark_keystore500k", numKeyFiles: 500000, resetAll: false, resetCache: true},
//		//{dir: "benchmark_keystore500k", numKeyFiles: 500000, resetAll: false, resetCache: false},
//	}
//
//	for _, c := range cases {
//		b.Run(fmt.Sprintf("KeyFiles#:%v, CacheFromScratch:%v", c.numKeyFiles, c.resetCache), func(b *testing.B) {
//			am := setupBenchmarkAccountFlowFast(filepath.Join("testdata", c.dir), c.numKeyFiles, c.resetAll, c.resetCache, b)
//			benchmarkAccountFlowFast(filepath.Join("testdata", c.dir), am, b)
//			am = nil
//		})
//	}
//}
//
//// Signing an account requires finding the keyfile.
//func testAccountSign(am *Manager, account Account, dir string) error {
//	if _, err := am.SignWithPassphrase(account.Address, "foo", testSigData); err != nil {
//		return err
//	}
//	return nil
//}
//
func testAccountFlow(am *Manager, dir string) error {

	// Create.
	a, err := am.NewAccount("foo")
	if err != nil {
		return err
	}
	p, e := filepath.Abs(dir)
	if e != nil {
		return fmt.Errorf("could not determine absolute path for temp dir: %v", e)
	}
	if !strings.HasPrefix(a.File, p+"/") {
		return fmt.Errorf("account file %s doesn't have dir prefix; %v", a.File, p)
	}
	stat, err := os.Stat(a.File)
	if err != nil {
		return fmt.Errorf("account file %s doesn't exist (%v)", a.File, err)
	}
	if runtime.GOOS != "windows" && stat.Mode() != 0600 {
		return fmt.Errorf("account file has wrong mode: got %o, want %o", stat.Mode(), 0600)
	}
	if !am.HasAddress(a.Address) {
		return fmt.Errorf("HasAddres(%x) should've returned true", a.Address)
	}

	// Update.
	if err := am.Update(a, "foo", "bar"); err != nil {
		return fmt.Errorf("Update error: %v", err)
	}

	// Sign with passphrase.
	_, err = am.SignWithPassphrase(a.Address, "bar", testSigData) // testSigData is an empty [32]byte established in manager_test.go
	if err != nil {
		return fmt.Errorf("should be able to sign from account: %v", err)
	}

	// Delete.
	if err := am.DeleteAccount(a, "bar"); err != nil {
		return fmt.Errorf("DeleteAccount error: %v", err)
	}
	if _, err := os.Stat(a.File); err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("account file %s should be gone after DeleteAccount", a.File)
	}
	if am.HasAddress(a.Address) {
		return fmt.Errorf("HasAddress(%x) should've returned true after DeleteAccount", a.Address)
	}
	return nil
}

//
//func createTestAccount(am *Manager, dir string) error {
//	a, err := am.NewAccount("foo")
//	if err != nil {
//		return err
//	}
//	p, e := filepath.Abs(dir)
//	if e != nil {
//		return fmt.Errorf("could not determine absolute path for temp dir: %v", e)
//	}
//	if !strings.HasPrefix(a.File, p+"/") {
//		return fmt.Errorf("account file %s doesn't have dir prefix; %v", a.File, p)
//	}
//	stat, err := os.Stat(a.File)
//	if err != nil {
//		return fmt.Errorf("account file %s doesn't exist (%v)", a.File, err)
//	}
//	if runtime.GOOS != "windows" && stat.Mode() != 0600 {
//		return fmt.Errorf("account file has wrong mode: got %o, want %o", stat.Mode(), 0600)
//	}
//	if !am.HasAddress(a.Address) {
//		return fmt.Errorf("HasAddres(%x) should've returned true", a.Address)
//	}
//	return nil
//}
//
//func getRandomIntN(n int) int {
//	rand.Seed(time.Now().UTC().UnixNano())
//	return int(rand.Int31n(int32(n)))
//}
//
//// Test benchmark for CRUSD/account; create, update, sign, delete.
//// Runs against setting of 10, 100, 1000, 10k, (100k, 1m) _existing_ accounts.
//func benchmarkAccountSignFast(dir string, am *Manager, accountsN int, b *testing.B) {
//	for i := 0; i < b.N; i++ {
//		j := getRandomIntN(accountsN)
//		b.Logf("signing with account index: %v", j)
//		account, e := am.AccountByIndex(j)
//		j = 0
//		if e != nil {
//			b.Fatal(e)
//		}
//		if e := testAccountSign(am, account, dir); e != nil {
//			b.Fatalf("error signing with account: %v", e)
//		}
//	}
//}
//
//func getFSvsCacheAccountN(dir string, ac *addrCache, b *testing.B) (fN, acN int) {
//
//	files, err := ioutil.ReadDir(ac.keydir)
//	if err != nil {
//		b.Fatalf("readdir: %v", err)
//	}
//
//	acN = len(ac.accounts())
//	fN = len(files) - 1 // - 1 because accounts.db is there too
//
//	return fN, acN
//}
//
//func setupBenchmarkAccountFlowFast(dir string, n int, resetAll, resetCache bool, b *testing.B) {
//	// Optionally: don't remove so we can compound accounts more quickly.
//	if resetAll {
//		b.Log("removing testdata keystore")
//		os.RemoveAll(dir)
//	} else if resetCache {
//		b.Log("removing existing cache")
//		os.Remove(filepath.Join(dir, "accounts.db")) // Remove cache db so we have to set up (scan()) every time.
//	} else {
//		b.Log("using existing cache and keystore")
//	}
//
//	// Ensure any removed dir exists.
//	if e := os.MkdirAll(dir, os.ModePerm); e != nil {
//		b.Fatalf("could not mkdir -p '%v': %v", dir, e)
//	}
//
//	files, err := ioutil.ReadDir(dir)
//	if err != nil {
//		b.Fatalf("readdir: %v", err)
//	}
//
//	ks, e := newKeyStore(dir, veryLightScryptN, veryLightScryptP)
//	if e != nil {
//		b.Fatalf("keystore: %v", e)
//	}
//
//	for i := len(files); i < n+1; i++ {
//		_, _, err := storeNewKey(ks, "foo")
//		if err != nil {
//			b.Fatalf("storenewkey: %v", err)
//		}
//	}
//	ks = nil
//	files = nil
//	//
//	//manStart := time.Now()
//	//am, err := NewManager(dir, veryLightScryptN, veryLightScryptP)
//	//if err != nil {
//	//	b.Fatal(err)
//	//}
//	//
//	//am.get.watcher.running = true // cache.watcher.running = true // prevent unexpected reloads
//	//
//	//b.Logf("setup time for manager: %v", time.Since(manStart))
//	//
//	//fsN, acN := getFSvsCacheAccountN(dir, am.cache, b)
//	//
//	//if acN > fsN { // Can allow greater number of keyfiles, in the case that there are invalids or dupes.
//	//	b.Errorf("accounts/files count mismatch: keyfiles: %v, accounts: %v", fsN, acN)
//	//} else {
//	//	b.Logf("files: %v, accounts: %v", fsN, acN)
//	//}
//	//
//	//b.Logf("setup time for manager: %v", time.Since(manStart))
//	//
//	//b.ResetTimer() // _benchmark_ timer, not setup timer.
//	//
//	//return am
//}
//
//// Test benchmark for CRUSD/account; create, update, sign, delete.
//// Runs against setting of 10, 100, 1000, 10k, (100k, 1m) _existing_ accounts.
//func benchmarkAccountFlowFast(dir string, am *Manager, b *testing.B) {
//	for i := 0; i < b.N; i++ {
//		if e := testAccountFlow(am, dir); e != nil {
//			b.Fatalf("error setting up account: %v", e)
//		}
//	}
//}

func BenchmarkManager_SignWithPassphrase(b *testing.B) {
	var cases = []struct {
		numKeyFiles               int
		wantCacheDB, resetCacheDB bool
	}{
		// You can use resetCache: false to save some time if you've already run the benchmark.
		// Note that if you make any changes to the structure of the cachedb you'll need to
		// dump and reinitialize accounts.db.
		{numKeyFiles: 100, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 100, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 500, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 500, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 1000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 1000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 5000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 5000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 10000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 10000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 20000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 20000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 100000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 100000, wantCacheDB: true, resetCacheDB: false},
		//{numKeyFiles: 200000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 200000, wantCacheDB: true, resetCacheDB: false},
		//{numKeyFiles: 500000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 500000, wantCacheDB: true, resetCacheDB: false},
	}

	for _, c := range cases {
		b.Run(fmt.Sprintf("CRUSD:KeyFiles#:%v,DB:%v,reset:%v", c.numKeyFiles, c.wantCacheDB, c.resetCacheDB), func(b *testing.B) {
			benchmarkManager_SignWithPassphrase(c.numKeyFiles, c.wantCacheDB, c.resetCacheDB, b)
		})
	}
}

func benchmarkManager_SignWithPassphrase(n int, wantcachedb bool, resetcachedb bool, b *testing.B) {

	// 20000 -> 20k
	staticKeyFilesResourcePath := strconv.Itoa(n)
	if strings.HasSuffix(staticKeyFilesResourcePath, "000") {
		staticKeyFilesResourcePath = strings.TrimSuffix(staticKeyFilesResourcePath, "000")
		staticKeyFilesResourcePath += "k"
	}

	staticKeyFilesResourcePath = filepath.Join("testdata", "benchmark_keystore"+staticKeyFilesResourcePath)

	if !wantcachedb {
		p, e := filepath.Abs(staticKeyFilesResourcePath)
		if e != nil {
			b.Fatal(e)
		}
		staticKeyFilesResourcePath = p
	}

	start := time.Now()
	am, me := NewManager(staticKeyFilesResourcePath, veryLightScryptN, veryLightScryptP, wantcachedb)
	if me != nil {
		b.Fatal(me)
	}
	elapsed := time.Since(start)
	b.Logf("establishing manager for %v accs: %v", n, elapsed)

	accs := am.Accounts()
	if !wantcachedb {
		am.ac.getWatcher().running = true
	}
	if len(accs) == 0 {
		b.Fatal("no accounts")
	}

	// Set up 1 DNE and 3 existing accs.
	// Using the last accs because they should take the longest to iterate to.
	var signAccounts []Account
	signAccounts = accs[(len(accs) - 4):]
	accs = nil

	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		for _, a := range signAccounts {
			if _, e := am.SignWithPassphrase(a.Address, "foo", testSigData); e != nil {
				b.Errorf("signing with passphrase: %v", e)
			}
		}
	}
	am.ac.close()
	am = nil
}

func BenchmarkManagerCRUSD(b *testing.B) {
	var cases = []struct {
		numKeyFiles               int
		wantCacheDB, resetCacheDB bool
	}{
		// You can use resetCache: false to save some time if you've already run the benchmark.
		// Note that if you make any changes to the structure of the cachedb you'll need to
		// dump and reinitialize accounts.db.
		{numKeyFiles: 100, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 100, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 500, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 500, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 1000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 1000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 5000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 5000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 10000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 10000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 20000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 20000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 100000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 100000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 200000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 200000, wantCacheDB: true, resetCacheDB: false},
		{numKeyFiles: 500000, wantCacheDB: false, resetCacheDB: false},
		{numKeyFiles: 500000, wantCacheDB: true, resetCacheDB: false},
	}

	for _, c := range cases {
		b.Run(fmt.Sprintf("CRUSD:KeyFiles#:%v,DB:%v,reset:%v", c.numKeyFiles, c.wantCacheDB, c.resetCacheDB), func(b *testing.B) {
			benchmarkManager_CRUSD(c.numKeyFiles, c.wantCacheDB, c.resetCacheDB, b)
		})
	}
}

func benchmarkManager_CRUSD(n int, wantcachedb bool, resetcachedb bool, b *testing.B) {

	// 20000 -> 20k
	staticKeyFilesResourcePath := strconv.Itoa(n)
	if strings.HasSuffix(staticKeyFilesResourcePath, "000") {
		staticKeyFilesResourcePath = strings.TrimSuffix(staticKeyFilesResourcePath, "000")
		staticKeyFilesResourcePath += "k"
	}

	staticKeyFilesResourcePath = filepath.Join("testdata", "benchmark_keystore"+staticKeyFilesResourcePath)

	if !wantcachedb {
		p, e := filepath.Abs(staticKeyFilesResourcePath)
		if e != nil {
			b.Fatal(e)
		}
		staticKeyFilesResourcePath = p
	}

	if resetcachedb {
		os.Remove(filepath.Join(staticKeyFilesResourcePath, "accounts.db"))
	}

	start := time.Now()
	am, me := NewManager(staticKeyFilesResourcePath, veryLightScryptN, veryLightScryptP, wantcachedb)
	if !wantcachedb {
		am.ac.getWatcher().running = true
	}

	if me != nil {
		b.Fatal(me)
	}
	elapsed := time.Since(start)
	b.Logf("establishing manager for %v accs: %v", n, elapsed)

	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		if e := testAccountFlow(am, staticKeyFilesResourcePath); e != nil {
			b.Errorf("account flow (CRUSD): %v (db: %v)", e, wantcachedb)
		}
	}
	am.ac.close()
	am = nil
}
