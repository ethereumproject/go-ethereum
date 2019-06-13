// Copyright 2014 The go-ethereum Authors
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
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/crypto/secp256k1"
)

func tmpKeyStore(t *testing.T) (dir string, ks *keyStore) {
	dir, err := ioutil.TempDir("", "geth-keystore-test")
	if err != nil {
		t.Fatal(err)
	}

	store, err := newKeyStore(dir, veryLightScryptN, veryLightScryptP)
	if err != nil {
		t.Fatal(err)
	}

	return dir, store
}

func TestKeyStore(t *testing.T) {
	dir, ks := tmpKeyStore(t)
	defer os.RemoveAll(dir)

	pass := "foo"
	key, account, err := storeNewKey(ks, pass)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ks.Lookup(account.File, pass)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Address, key.Address) {
		t.Errorf("got address %x, want %x", got.Address, key.Address)
	}
	if !reflect.DeepEqual(key.PrivateKey, key.PrivateKey) {
		t.Errorf("got private key %x, want %x", got.PrivateKey, key.PrivateKey)
	}
}

func TestKeyStoreDecryptionFail(t *testing.T) {
	dir, ks := tmpKeyStore(t)
	defer os.RemoveAll(dir)

	pass := "foo"
	_, account, err := storeNewKey(ks, pass)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ks.Lookup(account.File, "bar"); err != ErrDecrypt {
		t.Fatalf("wrong error for invalid passphrase\ngot %q\nwant %q", err, ErrDecrypt)
	}
}

func TestImportPreSaleKey(t *testing.T) {
	dir, ks := tmpKeyStore(t)
	defer os.RemoveAll(dir)

	// file content of a presale key file generated with:
	// python pyethsaletool.py genwallet
	// with password "foo"
	fileContent := "{\"encseed\": \"26d87f5f2bf9835f9a47eefae571bc09f9107bb13d54ff12a4ec095d01f83897494cf34f7bed2ed34126ecba9db7b62de56c9d7cd136520a0427bfb11b8954ba7ac39b90d4650d3448e31185affcd74226a68f1e94b1108e6e0a4a91cdd83eba\", \"ethaddr\": \"d4584b5f6229b7be90727b0fc8c6b91bb427821f\", \"email\": \"gustav.simonsson@gmail.com\", \"btcaddr\": \"1EVknXyFC68kKNLkh6YnKzW41svSRoaAcx\"}"
	pass := "foo"
	account, _, err := importPreSaleKey(ks, []byte(fileContent), pass)
	if err != nil {
		t.Fatal(err)
	}
	if account.Address != common.HexToAddress("d4584b5f6229b7be90727b0fc8c6b91bb427821f") {
		t.Errorf("imported account has wrong address %x", account.Address)
	}
	if !strings.HasPrefix(account.File, dir) {
		t.Errorf("imported account file not in keystore directory: %q", account.File)
	}
}

// Test and utils for the key store tests in the Ethereum JSON tests;
// testdataKeyStoreTests/basic_tests.json
type KeyStoreTestV3 struct {
	Json     web3v3
	Password string
	Priv     string
}

type KeyStoreTestV1 struct {
	Json     web3v1
	Password string
	Priv     string
}

func TestV3_PBKDF2_1(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTestV3("testdata/v3_test_vector.json", t)
	testDecryptV3(tests["wikipage_test_vector_pbkdf2"], t)
}

func TestV3_PBKDF2_2(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTestV3("../tests/files/KeyStoreTests/basic_tests.json", t)
	testDecryptV3(tests["test1"], t)
}

func TestV3_PBKDF2_3(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTestV3("../tests/files/KeyStoreTests/basic_tests.json", t)
	testDecryptV3(tests["python_generated_test_with_odd_iv"], t)
}

func TestV3_PBKDF2_4(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTestV3("../tests/files/KeyStoreTests/basic_tests.json", t)
	testDecryptV3(tests["evilnonce"], t)
}

func TestV3_Scrypt_1(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTestV3("testdata/v3_test_vector.json", t)
	testDecryptV3(tests["wikipage_test_vector_scrypt"], t)
}

func TestV3_Scrypt_2(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTestV3("../tests/files/KeyStoreTests/basic_tests.json", t)
	testDecryptV3(tests["test2"], t)
}

func TestV1_1(t *testing.T) {
	t.Parallel()
	tests := loadKeyStoreTestV1("testdata/v1_test_vector.json", t)
	testDecryptV1(tests["test1"], t)
}

func TestV1_2(t *testing.T) {
	t.Parallel()
	store, err := newKeyStore("testdata/v1", LightScryptN, LightScryptP)
	if err != nil {
		t.Fatal(err)
	}

	key, err := store.Lookup("cb61d5a9c4896fb9658090b597ef0e7be6f7b67e/cb61d5a9c4896fb9658090b597ef0e7be6f7b67e", "g")
	if err != nil {
		t.Fatal(err)
	}

	got := hex.EncodeToString(key.Address[:])
	want := "cb61d5a9c4896fb9658090b597ef0e7be6f7b67e"
	if got != want {
		t.Errorf("got address %s, want %s", got, want)
	}

	got = hex.EncodeToString(crypto.FromECDSA(key.PrivateKey))
	want = "d1b1178d3529626a1a93e073f65028370d14c7eb0936eb42abef05db6f37ad7d"
	if got != want {
		t.Errorf("got private key %s, want %s", got, want)
	}
}

func testDecryptV3(test KeyStoreTestV3, t *testing.T) {
	privBytes, err := decryptKeyV3(&test.Json, test.Password)
	if err != nil {
		t.Fatal(err)
	}
	privHex := hex.EncodeToString(privBytes)
	if test.Priv != privHex {
		t.Fatal(fmt.Errorf("Decrypted bytes not equal to test, expected %v have %v", test.Priv, privHex))
	}
}

func testDecryptV1(test KeyStoreTestV1, t *testing.T) {
	privBytes, err := decryptKeyV1(&test.Json, test.Password)
	if err != nil {
		t.Fatal(err)
	}
	privHex := hex.EncodeToString(privBytes)
	if test.Priv != privHex {
		t.Fatal(fmt.Errorf("Decrypted bytes not equal to test, expected %v have %v", test.Priv, privHex))
	}
}

func loadKeyStoreTestV3(file string, t *testing.T) map[string]KeyStoreTestV3 {
	tests := make(map[string]KeyStoreTestV3)
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(bytes, &tests); err != nil {
		t.Fatal(err)
	}
	return tests
}

func loadKeyStoreTestV1(file string, t *testing.T) map[string]KeyStoreTestV1 {
	tests := make(map[string]KeyStoreTestV1)
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(bytes, &tests); err != nil {
		t.Fatal(err)
	}
	return tests
}

// WTF?
// newKeyForDirectICAP generates a key whose address fits into < 155 bits so it can fit
// into the Direct ICAP spec. for simplicity and easier compatibility with other libs, we
// retry until the first byte is 0.
func TestKeyForDirectICAP(t *testing.T) {
	t.Parallel()

	for {
		randBytes := make([]byte, 64)
		_, err := rand.Read(randBytes)
		if err != nil {
			t.Fatalf("key generation: could not read from random source: %s", err)
		}

		privateKeyECDSA, err := ecdsa.GenerateKey(secp256k1.S256(), bytes.NewReader(randBytes))
		if err != nil {
			t.Fatalf("key generation: ecdsa.GenerateKey failed: %s", err)
		}

		key, err := newKeyFromECDSA(privateKeyECDSA)
		if err != nil {
			t.Fatal(err)
		}

		if key.Address[0] == 0 {
			return
		}
	}
}

const (
	veryLightScryptN = 2
	veryLightScryptP = 1
)

// Tests that a JSON key file can be decrypted and encrypted in multiple rounds.
func TestKeyEncryptDecrypt(t *testing.T) {
	keyjson, err := ioutil.ReadFile("testdata/very-light-scrypt.json")
	if err != nil {
		t.Fatal(err)
	}
	password := ""
	address := common.HexToAddress("45dea0fb0bba44f4fcf290bba71fd57d7117cbb8")

	// Do a few rounds of decryption and encryption
	for i := 0; i < 3; i++ {
		// try a bad password first
		if _, err := decryptKey(keyjson, password+"bad"); err == nil {
			t.Errorf("test %d: json key decrypted with bad password", i)
		}
		// decrypt with the correct password
		key, err := decryptKey(keyjson, password)
		if err != nil {
			t.Errorf("test %d: json key failed to decrypt: %v", i, err)
		}
		if key.Address != address {
			t.Errorf("test %d: key address mismatch: have %x, want %x", i, key.Address, address)
		}
		// recrypt with a new password and start over
		password += "new data appended"
		if keyjson, err = encryptKey(key, password, veryLightScryptN, veryLightScryptP); err != nil {
			t.Errorf("test %d: failed to recrypt key %v", i, err)
		}
	}
}
