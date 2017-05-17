package accounts

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"io/ioutil"
)

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

func createTestAccount(am *Manager, dir string) error {
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
	return nil
}

// Test benchmark for CRUSD/account; create, update, sign, delete.
// Runs against setting of 10, 100, 1000, 10k, (100k, 1m) _existing_ accounts.
func benchmarkAccountFlow(dir string, n int, reset bool, b *testing.B) {
	start := time.Now()
	//dir, err := ioutil.TempDir("", "eth-acctmanager-test")
	//if err != nil {
	//	b.Fatal(err)
	//}

	if e := os.MkdirAll(dir, os.ModePerm); e != nil {
		b.Fatalf("could not create dir: %v", e)
	}

	// Optionally: don't remove so we can compound accounts more quickly.
	if reset {
		defer func () {
			b.Log("removing testdata keydir")
			os.RemoveAll(dir)
		}()
	}

	am, err := NewManager(dir, veryLightScryptN, veryLightScryptP)
	if err != nil {
		b.Fatal(err)
	}

	initAccountsN := len(am.Accounts())

	for len(am.Accounts()) < n { //  + initAccountsN
		if e := createTestAccount(am, dir); e != nil {
			b.Fatalf("error setting up acount: %v", e)
		}
	}
	elapsed := time.Since(start)
	defer b.Logf("setting up %v(want)/%v(existing) accounts took %v", n, initAccountsN, elapsed)
	if len(am.Accounts()) != n {
		b.Fatalf("wrong number accounts: want: %v, got: %v", n, len(am.Accounts()))
	}

	files, _ := ioutil.ReadDir(dir)
	if len(files) - 1 != n {
		b.Fatalf("files/account mismatch: files: %v, cacheaccounts: %v", len(files)-1, n)
	}

	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		if e := testAccountFlow(am, dir); e != nil {
			b.Fatalf("error setting up acount: %v", e)
		}
	}
	am.cache.close()
}

// These compound now...
func BenchmarkAccountFlow100(b *testing.B) {
	benchmarkAccountFlow(filepath.Join("testdata", "benchmark_keystore100"), 100, true, b)
}
func BenchmarkAccountFlow500(b *testing.B) {
	benchmarkAccountFlow(filepath.Join("testdata", "benchmark_keystore500"), 500, true, b)
} // ie 600
func BenchmarkAccountFlow1k(b *testing.B) {
	benchmarkAccountFlow(filepath.Join("testdata", "benchmark_keystore1k"), 1000, true, b)
} // ie >=1600
func BenchmarkAccountFlow5k(b *testing.B) {
	benchmarkAccountFlow(filepath.Join("testdata", "benchmark_keystore5k"), 5000, false, b)
} // ie >=6600
func BenchmarkAccountFlow10k(b *testing.B) {
	benchmarkAccountFlow(filepath.Join("testdata", "benchmark_keystore10k"), 10000, false, b)
} // >=16600
func BenchmarkAccountFlow20kFast(b *testing.B) {
	benchmarkAccountFlowFast(filepath.Join("testdata", "benchmark_keystore20k"), 20000, true, true, b)
} // >=36600
func BenchmarkAccountFlow100kFast(b *testing.B) {
	benchmarkAccountFlowFast(filepath.Join("testdata", "benchmark_keystore100k"), 100000, false, false, b)
}
func BenchmarkAccountFlow500kFast(b *testing.B) {
	benchmarkAccountFlowFast(filepath.Join("testdata", "benchmark_keystore500k"), 500000, false, false, b)
}

func getFSvsCacheAccountN(dir string, ac *addrCache, b *testing.B) (fN, acN int) {

	files, err := ioutil.ReadDir(ac.keydir)
	if err != nil {
		b.Fatalf("readdir: %v", err)
	}

	acN = len(ac.accounts())
	fN = len(files) - 1 // - 1 because accounts.db is there too

	return fN, acN
}

// Test benchmark for CRUSD/account; create, update, sign, delete.
// Runs against setting of 10, 100, 1000, 10k, (100k, 1m) _existing_ accounts.
func benchmarkAccountFlowFast(dir string, n int, resetAll bool, resetCache bool, b *testing.B) {
	//start := time.Now()
	//dir, err := ioutil.TempDir("", "eth-acctmanager-test")
	//if err != nil {
	//	b.Fatal(err)
	//}

	// Optionally: don't remove so we can compound accounts more quickly.
	if resetAll {
		b.Log("removing testdata keystore")
		os.RemoveAll(dir)
	} else if resetCache {
		b.Log("removing existing cache")
		os.Remove(filepath.Join(dir, "accounts.db"))  // Remove cache db so we have to set up (scan()) every time.
	} else {
		b.Log("using existing cache")
	}

	if e := os.MkdirAll(dir, os.ModePerm); e != nil {
		b.Fatalf("could not mkdir -p '%v': %v", dir, e)
	}

	ks, e := newKeyStore(dir, veryLightScryptN, veryLightScryptP)
	if e != nil {
		b.Fatalf("keystore: %v", e)
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		b.Fatalf("readdir: %v", err)
	}

	// note: -1 assumes cache accounts.db already exists
	if n > len(files)-1 {
		for i := 0; i < n-len(files)-1; i++ {
			_, _, err := storeNewKey(ks, "foo")
			if err != nil {
				b.Fatalf("storenewkey: %v", err)
			}
		}
	}

	ks = nil

	cacheStart := time.Now()
	ac := newAddrCache(dir) // will syncfs2db or scan depending if it exists or not
	b.Logf("setup time for cache: %v", time.Since(cacheStart))

	fsN, acN := getFSvsCacheAccountN(dir, ac, b)



	//accs := []Account{}
	//defer ac.setBatchAccounts(accs) // case closing case
	//if n > fsN {
	//
	//	for i := 0; i < n - fsN; i++ {
	//		_, account, err := storeNewKey(ks, "foo")
	//		if err != nil {
	//			b.Fatalf("storenewkey: %v", err)
	//		}
	//		accs = append(accs, account)
	//
	//		if (len(accs)) == 10000 || i == n - fsN - 1 {
	//			if es := ac.setBatchAccounts(accs); len(es) > 0 {
	//				b.Fatalf("setbatchaccounts: %v", es)
	//			}
	//			accs = nil
	//		}
	//	}
	//
	//	elapsed := time.Since(start)
	//	defer b.Logf("setting up %v/%v new accounts took %v", n - fsN, fsN, elapsed)
	//}

	ac.close()

	//if acN < fsN {
	//	b.Errorf("too few cacheaccounts: want: >=%v, got: %v", fsN, acN)
	//} else {
		b.Logf("keystore filesN: %v, cacheAccountsN: %v", fsN, acN)
	//}

	am, err := NewManager(dir, veryLightScryptN, veryLightScryptP)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		if e := testAccountFlow(am, dir); e != nil {
			b.Fatalf("error setting up account: %v", e)
		}
	}
}
