package accounts

import (
	"testing"
	"os"
	"strings"
	"runtime"
	"io/ioutil"
	"path/filepath"
	"fmt"
	"time"
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
	if !strings.HasPrefix(a.File, p + "/") {
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
	if !strings.HasPrefix(a.File, p + "/") {
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

// want to run for 10, 100, 1000, 10k, (100k) accounts
func benchmarkAccountFlow(n int, b *testing.B) {
	start := time.Now()
	dir, err := ioutil.TempDir("", "eth-acctmanager-test")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	am, err := NewManager(dir, veryLightScryptN, veryLightScryptP)
	if err != nil {
		b.Fatal(err)
	}

	for len(am.Accounts()) < n {
		if e := createTestAccount(am, dir); e != nil {
			b.Fatalf("error setting up acount: %v", e)
		}
	}
	elapsed := time.Since(start)
	b.Logf("setting up %v accounts took %v", n, elapsed)

	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		if e := testAccountFlow(am, dir); e != nil {
			b.Fatalf("error setting up acount: %v", e)
		}
	}
}

func BenchmarkAccountFlow100(b *testing.B) { benchmarkAccountFlow(100, b)}
func BenchmarkAccountFlow500(b *testing.B) { benchmarkAccountFlow(500, b)}
func BenchmarkAccountFlow1000(b *testing.B) { benchmarkAccountFlow(1000, b)}
func BenchmarkAccountFlow5000(b *testing.B) { benchmarkAccountFlow(5000, b)}
func BenchmarkAccountFlow10000(b *testing.B) { benchmarkAccountFlow(10000, b)}
func BenchmarkAccountFlow20000(b *testing.B) { benchmarkAccountFlow(20000, b)}

