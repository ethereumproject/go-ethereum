package accounts

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func tmpManager_CacheDB(t *testing.T) (string, *Manager) {
	rand.Seed(time.Now().UnixNano())
	dir, err := ioutil.TempDir("", fmt.Sprintf("eth-manager-cachedb-test-%d-%d", os.Getpid(), rand.Int()))
	if err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(dir, veryLightScryptN, veryLightScryptP, true)
	if err != nil {
		t.Fatal(err)
	}
	return dir, m
}

func TestManager_DB(t *testing.T) {

	dir, am := tmpManager_CacheDB(t)
	defer os.RemoveAll(dir)

	a, err := am.NewAccount("foo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(a.File, dir) {
		t.Errorf("account file %s doesn't have dir prefix", a.File)
	}
	stat, err := os.Stat(a.File)
	if err != nil {
		t.Fatalf("account file %s doesn't exist (%v)", a.File, err)
	}
	if runtime.GOOS != "windows" && stat.Mode() != 0600 {
		t.Fatalf("account file has wrong mode: got %o, want %o", stat.Mode(), 0600)
	}
	if !am.HasAddress(a.Address) {
		t.Errorf("HasAddres(%x) should've returned true", a.Address)
	}
	if err := am.Update(a, "foo", "bar"); err != nil {
		t.Errorf("Update error: %v", err)
	}
	if err := am.DeleteAccount(a, "bar"); err != nil {
		t.Errorf("DeleteAccount error: %v", err)
	}
	if _, err := os.Stat(a.File); err == nil || !os.IsNotExist(err) {
		t.Errorf("account file %s should be gone after DeleteAccount", a.File)
	}
	if am.HasAddress(a.Address) {
		t.Errorf("HasAddress(%x) should've returned true after DeleteAccount", a.Address)
	}
	am.ac.close()
	am = nil
}

func TestManager_Accounts_CacheDB(t *testing.T) {
	// bug(whilei): I don't know why you have to do rm.
	// Running the file as a standalone test is no problem.
	// Running the suite (ie go test -v ./accounts/), it hangs here.
	// Again, I think it has to do with test concurrency.
	os.Remove(filepath.Join(cachetestDir, "accounts.db"))
	am, err := NewManager(cachetestDir, LightScryptN, LightScryptP, true)
	if err != nil {
		t.Fatal(err)
	}
	am.ac.Syncfs2db(time.Now())
	accounts := am.Accounts()
	if !reflect.DeepEqual(accounts, cachedbtestAccounts) {
		t.Fatalf("cachedb got initial accounts: %swant %s", spew.Sdump(accounts), spew.Sdump(cachedbtestAccounts))
	}
	am.ac.close()
	am = nil
}

func TestManager_AccountsByIndex_CacheDB(t *testing.T) {
	os.Remove(filepath.Join(cachetestDir, "accounts.db"))
	am, err := NewManager(cachetestDir, LightScryptN, LightScryptP, true)
	if err != nil {
		t.Fatal(err)
	}
	am.ac.Syncfs2db(time.Now())

	for i := range cachedbtestAccounts {
		wantAccount := cachedbtestAccounts[i]
		gotAccount, e := am.AccountByIndex(i)
		if e != nil {
			t.Fatalf("manager cache db #accountsbyindex: %v", e)
		}
		if !reflect.DeepEqual(wantAccount, gotAccount) {
			t.Fatalf("got: %v, want: %v", spew.Sdump(gotAccount), spew.Sdump(wantAccount))
		}
	}
	am.ac.close()
	am = nil
}

func TestSignWithPassphrase_DB(t *testing.T) {
	dir, am := tmpManager_CacheDB(t)
	defer os.RemoveAll(dir)

	pass := "passwd"
	acc, err := am.NewAccount(pass)
	if err != nil {
		t.Fatal(err)
	}

	if _, unlocked := am.unlocked[acc.Address]; unlocked {
		t.Fatal("expected account to be locked")
	}

	_, err = am.SignWithPassphrase(acc.Address, pass, testSigData)
	if err != nil {
		t.Fatal(err)
	}

	if _, unlocked := am.unlocked[acc.Address]; unlocked {
		t.Fatal("expected account to be locked")
	}

	if _, err = am.SignWithPassphrase(acc.Address, "invalid passwd", testSigData); err == nil {
		t.Fatal("expected SignHash to fail with invalid password")
	}
	am.ac.close()
	am = nil
}

func TestTimedUnlock_DB(t *testing.T) {
	dir, am := tmpManager_CacheDB(t)
	defer os.RemoveAll(dir)

	pass := "foo"
	a1, err := am.NewAccount(pass)

	// Signing without passphrase fails because account is locked
	_, err = am.Sign(a1.Address, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked before unlocking, got ", err)
	}

	// Signing with passphrase works
	if err = am.TimedUnlock(a1, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1.Address, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(250 * time.Millisecond)
	_, err = am.Sign(a1.Address, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}
	am.ac.close()
	am = nil
}

func TestOverrideUnlock_DB(t *testing.T) {
	dir, am := tmpManager_CacheDB(t)
	defer os.RemoveAll(dir)

	pass := "foo"
	a1, err := am.NewAccount(pass)

	// Unlock indefinitely.
	if err = am.TimedUnlock(a1, pass, 5*time.Minute); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1.Address, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// reset unlock to a shorter period, invalidates the previous unlock
	if err = am.TimedUnlock(a1, pass, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase still works because account is temp unlocked
	_, err = am.Sign(a1.Address, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(250 * time.Millisecond)
	_, err = am.Sign(a1.Address, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}
	am.ac.close()
	am = nil
}

// unlocks account from manager created in existing testdata/keystore dir
func TestTimedUnlock_DB2(t *testing.T) {

	am, err := NewManager(cachetestDir, veryLightScryptN, veryLightScryptP, true)
	if err != nil {
		t.Fatal(err)
	}

	a1 := cachetestAccounts[1]

	// Signing with passphrase works
	if err := am.TimedUnlock(a1, "foobar", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// Signing without passphrase works because account is temp unlocked
	_, err = am.Sign(a1.Address, testSigData)
	if err != nil {
		t.Fatal("Signing shouldn't return an error after unlocking, got ", err)
	}

	// Signing fails again after automatic locking
	time.Sleep(250 * time.Millisecond)
	_, err = am.Sign(a1.Address, testSigData)
	if err != ErrLocked {
		t.Fatal("Signing should've failed with ErrLocked timeout expired, got ", err)
	}
	am.ac.close()
	am = nil
}

// This test should fail under -race if signing races the expiration goroutine.
func TestSignRace_DB(t *testing.T) {
	dir, am := tmpManager_CacheDB(t)
	defer os.RemoveAll(dir)

	// Create a test account.
	a1, err := am.NewAccount("")
	if err != nil {
		t.Fatal("could not create the test account", err)
	}

	if err := am.TimedUnlock(a1, "", 15*time.Millisecond); err != nil {
		t.Fatal("could not unlock the test account", err)
	}
	end := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(end) {
		if _, err := am.Sign(a1.Address, testSigData); err == ErrLocked {
			return
		} else if err != nil {
			t.Errorf("Sign error: %v", err)
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	t.Error("Account did not lock within the timeout")
	am.ac.close()
	am = nil
}
