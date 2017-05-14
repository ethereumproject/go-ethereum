package accounts

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"
	"io/ioutil"
	"strings"
	"time"
	"math/rand"
	"path/filepath"
)

// Global constants.
var accountsNInit int = 1000
var accountsNMax int = 5000
var accountsNDiff int = accountsNMax - accountsNInit
var scaleTestBasePath = "testdata" // use relative directory (instead of passing "" to ioutil.TempDir which select defaulty)
var scaleTestTmpPrefix = "scale-acct-test"

// At 100ms signing fails because account locks at index 2464.
var gracePeriodUnlockToSign time.Duration = 200*time.Millisecond // max length for unlock in order to sign with an account

// Global to assign.
var scaleTestTmpDirName string // global, abs, set on new tmp dir by ioutil.TempDir
var amG *Manager               // set on create initial accounts

// TestMain is called *once per file*.
func TestMain(m *testing.M) {
	if e := createTestAccounts(accountsNInit); e != nil {
		log.Fatal(e)
	}
	os.Exit(m.Run())
	p, err := filepath.Abs(scaleTestTmpDirName)
	if err != nil {
		log.Fatal(err)
	}
	if e := os.RemoveAll(p); e != nil {
		log.Fatal(e)
	}
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

func createTestAccounts(n int) error {
	am, tmpDirName, err := scaleTmpManager(scaleTestBasePath)
	if err != nil {
		return err
	}
	// assign globals
	amG = am
	scaleTestTmpDirName = tmpDirName

	// Only creates account *initially*.
	if len(amG.Accounts()) == 0 {
		for i := 0; i < n; i++ {
			if err := createTestAccount(am, tmpDirName); err != nil {
				return err
			}
		}
	}
	return nil
}

// Can create and manage _more_ accounts?

func TestManager_Accounts_Scale_CreateUpdateSignDelete(t *testing.T) {
	if amG == nil {
		t.Fatal("global account manager not established")
	}
	if scaleTestTmpDirName == "" {
		t.Fatal("empty scale test tmp dir")
	}

	// Create.
	// Ensure got expected number of initial test accounts.
	if l := len(amG.Accounts()); l != accountsNInit {
		t.Fatalf("wrong number of initial accounts: got: %v, want: %v", l, accountsNInit)
	}

	for i := 0; i < accountsNDiff; i++ {
		// Get time to create one new account 10 times linearly over new accounts n.
		if i != 0 && (accountsNDiff/10) % i == 0 {
			start := time.Now()
			if err := createTestAccount(amG, scaleTestTmpDirName); err != nil {
				t.Fatalf("error creating new account #%v", accountsNInit+i)
			}
			dur := time.Since(start)
			t.Logf("creating %v account took %v", accountsNInit+i, dur)
		} else {
			if err := createTestAccount(amG, scaleTestTmpDirName); err != nil {
				t.Fatalf("error creating new account #%v", accountsNInit+i)
			}
		}
	}

	// Update.
	amG = nil // clear mem
	am, err := NewManager(scaleTestTmpDirName, veryLightScryptN, veryLightScryptP)
	if err != nil {
		t.Fatal(err)
	}
	amG = am

	if l := len(amG.Accounts()); l != accountsNInit+accountsNDiff {
		t.Fatalf("wrong number of final accounts: got: %v, want: %v", l, accountsNInit)
	}

	for i, a := range amG.Accounts() {
		// Get time to update one account 10 times over new accounts n.
		if i != 0 && (accountsNDiff/10) % i == 0 {
			start := time.Now()
			if err := amG.Update(a, "foo", "bar"); err != nil {
				t.Errorf("Update error: %v", err)
			}
			dur := time.Since(start)
			t.Logf("updating %v account took %v", accountsNInit+i, dur)
		} else {
			if err := amG.Update(a, "foo", "bar"); err != nil {
				t.Errorf("Update error: %v", err)
			}
		}
	}

	// Sign.

	if l := len(amG.Accounts()); l != accountsNInit+accountsNDiff {
		t.Fatalf("wrong number of final accounts: got: %v, want: %v", l, accountsNInit)
	}

	for i, a := range amG.Accounts() {
		// reset unlock to a shorter period, invalidates the previous unlock
		if err := amG.TimedUnlock(a, "bar", gracePeriodUnlockToSign); err != nil {
			t.Error(err)
		}

		// Signing without passphrase still works because account is temp unlocked
		_, err := amG.Sign(a.Address, testSigData) // testSigData is an empty [32]byte established in manager_test.go
		if err != nil {
			t.Errorf("should be able to sign from account: index: %v: %v", i, err)
		}
	}


	// Delete.
	accts := amG.Accounts()
	l := len(accts)
	if l != accountsNInit+accountsNDiff {
		t.Fatalf("wrong number of final accounts: got: %v, want: %v", l, accountsNInit)
	}

	smallPortionN := float64(l) * 0.05
	smallPortionNInt := int(smallPortionN)
	for i := 0; i < smallPortionNInt; i++ {

		rand.Seed(time.Now().UTC().UnixNano())
		randInt := int(rand.Int31n(int32(l-1)))
		a := accts[randInt] // pick a random account

		if err := amG.DeleteAccount(a, "bar"); err != nil {
			t.Errorf("DeleteAccount error #%v @%v, %v", randInt, i, err)
		}
		if _, err := os.Stat(a.File); err == nil || !os.IsNotExist(err) {
			t.Errorf("account file %s should be gone after DeleteAccount", a.File)
		}
		if amG.HasAddress(a.Address) {
			t.Errorf("HasAddress(%x) should've returned true after DeleteAccount", a.Address)
		}
	}
}

func scaleTmpManager(tpath string) (*Manager, string, error) {
	name, err := ioutil.TempDir(scaleTestBasePath, scaleTestTmpPrefix)
	scaleTestTmpDirName = name // assign global
	if err != nil {
		return nil, name, err
	}

	m, err := NewManager(name, veryLightScryptN, veryLightScryptP)
	if err != nil {
		return nil, "", err
	}
	return m, name, nil
}
