// Copyright 2015 The go-ethereum Authors
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
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

var testSigData = make([]byte, 32)

func tmpManager(t *testing.T) (string, *Manager) {
	rand.Seed(time.Now().UnixNano())
	dir, err := ioutil.TempDir("", fmt.Sprintf("eth-manager-mem-test-%d-%d", os.Getpid(), rand.Int()))
	if err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(dir, veryLightScryptN, veryLightScryptP, false)
	if err != nil {
		t.Fatal(err)
	}
	return dir, m
}

func TestManager_Mem(t *testing.T) {
	dir, am := tmpManager(t)
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
}

func TestManager_Accounts_Mem(t *testing.T) {
	am, err := NewManager(cachetestDir, LightScryptN, LightScryptP, false)
	if err != nil {
		t.Fatal(err)
	}
	accounts := am.Accounts()
	if !reflect.DeepEqual(accounts, cachetestAccounts) {
		t.Fatalf("mem got initial accounts: %swant %s", spew.Sdump(accounts), spew.Sdump(cachetestAccounts))
	}
}

func TestManager_AccountsByIndex(t *testing.T) {
	am, err := NewManager(cachetestDir, LightScryptN, LightScryptP, false)
	if err != nil {
		t.Fatal(err)
	}

	for i := range cachetestAccounts {
		wantAccount := cachetestAccounts[i]
		gotAccount, e := am.AccountByIndex(i)
		if e != nil {
			t.Fatalf("manager cache mem #accountsbyindex: %v", e)
		}
		if !reflect.DeepEqual(wantAccount, gotAccount) {
			t.Fatalf("want: %v, got: %v", wantAccount, gotAccount)
		}
	}
}

func TestSignWithPassphrase_Mem(t *testing.T) {
	dir, am := tmpManager(t)
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
}

// unlocks newly created account in temp dir
func TestTimedUnlock_Mem(t *testing.T) {
	dir, am := tmpManager(t)
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
}

// unlocks account from manager created in existing testdata/keystore dir
func TestTimedUnlock_Mem2(t *testing.T) {
	am, err := NewManager(cachetestDir, veryLightScryptN, veryLightScryptP, false)
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
}

func TestOverrideUnlock_Mem(t *testing.T) {
	dir, am := tmpManager(t)
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
}

// This test should fail under -race if signing races the expiration goroutine.
func TestSignRace_Mem(t *testing.T) {
	dir, am := tmpManager(t)
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
}
