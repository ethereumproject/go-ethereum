package accounts

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"
	"strings"
	"time"
	"path/filepath"
)

// Global constants.
var accountsNInit int = 3
var accountsNMax int = 5
var accountsNDiff int = accountsNMax - accountsNInit
var scaleTestBasePath = "testdata" // use relative directory (instead of passing "" to ioutil.TempDir which select defaulty)
var scaleTestTmpPrefix = "scale-acct-test"

// At 100ms signing fails because account locks at index 2464.
var gracePeriodUnlockToSign time.Duration = 200*time.Millisecond // max length for unlock in order to sign with an account

// Global to assign.
var scaleTestTmpDirName string = filepath.Join("testdata","scale-acct-test") // global, abs, set on new tmp dir by ioutil.TempDir
var amG *Manager               // set on create initial accounts


func scaleTmpManager(tpath string) (*Manager, string, error) {
	name := scaleTestTmpDirName
	m, err := NewManager(name, veryLightScryptN, veryLightScryptP)
	if err != nil {
		return nil, "", err
	}
	return m, name, nil
}


// TestMain is called *once per file*.
func TestMain(m *testing.M) {
	am, tmpDirName, err := scaleTmpManager(scaleTestBasePath)
	if err != nil {
		log.Fatal(err)
	}
	// assign globals
	amG = am
	scaleTestTmpDirName = tmpDirName

	os.Exit(m.Run())
	p, err := filepath.Abs(scaleTestTmpDirName)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		log.Fatal(err)
	}
	//if e := os.RemoveAll(p); e != nil {
	//	log.Fatal(e)
	//}
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
	// Only creates account *initially*.
	for len(amG.Accounts()) < n {
		if err := createTestAccount(amG, scaleTestTmpDirName); err != nil {
			return err
		}
	}
	return nil
}

// Can create and manage _more_ accounts?

func TestManager_Accounts_Scale_CreateUpdateSignDelete(t *testing.T) {

	if e := createTestAccounts(accountsNInit); e != nil {
		log.Fatal(e)
	}

	if amG == nil {
		t.Fatal("global account manager not established")
	}
	if scaleTestTmpDirName == "" {
		t.Fatal("empty scale test tmp dir")
	}

	// Create.
	// Ensure got expected number of initial test accounts.
	if l := len(amG.Accounts()); l < accountsNInit {
		t.Fatalf("too few initial accounts: got: %v, want: %v", l, accountsNInit)
	}

	for i := 0; len(amG.Accounts()) <= accountsNMax; i++ {
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

	if l := len(amG.Accounts()); l < accountsNMax {
		t.Fatalf("too few accounts (@updating): got: %v, want: %v", l, accountsNMax)
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

	if l := len(amG.Accounts()); l < accountsNMax {
		t.Fatalf("too few accounts (@sign): got: %v, want: %v", l, accountsNMax)
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
	if l < accountsNMax {
		t.Fatalf("wrong number of final accounts: got: %v, want: %v", l, accountsNMax)
	}

	// Delete diffN accounts to return to initN number of accounts.
	for i := 0; i < l - accountsNInit; i++ {

		a := accts[i]
		// TODO: randomize accounts to remove? or otherwise some reasonable order

		if err := amG.DeleteAccount(a, "bar"); err != nil {
			t.Errorf("DeleteAccount error (@%v/%v): %v\nacct: %v", i, l, err, a)
		}
		if _, err := os.Stat(a.File); err == nil || !os.IsNotExist(err) {
			t.Errorf("account file %s should be gone after DeleteAccount", a.File)
		}
		if amG.HasAddress(a.Address) {
			t.Errorf("HasAddress(%x) should've returned true after DeleteAccount", a.Address)
		}
	}
}
