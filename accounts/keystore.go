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
	"bytes"
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/crypto/randentropy"
	"github.com/ethereumproject/go-ethereum/crypto/secp256k1"
)

type key struct {
	UUID string
	// to simplify lookups we also store the address
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
}

type plainKeyJSON struct {
	ID         string `json:"id"`
	Address    string `json:"address"`
	PrivateKey string `json:"privatekey"`
	Version    int    `json:"version"`
}

func (k *key) MarshalJSON() (j []byte, err error) {
	jStruct := plainKeyJSON{
		ID:         k.UUID,
		Address:    hex.EncodeToString(k.Address[:]),
		PrivateKey: hex.EncodeToString(crypto.FromECDSA(k.PrivateKey)),
		Version:    3,
	}
	j, err = json.Marshal(jStruct)
	return j, err
}

func (k *key) UnmarshalJSON(j []byte) (err error) {
	keyJSON := new(plainKeyJSON)
	err = json.Unmarshal(j, &keyJSON)
	if err != nil {
		return err
	}

	k.UUID = keyJSON.ID
	addr, err := hex.DecodeString(keyJSON.Address)
	if err != nil {
		return err
	}
	k.Address = common.BytesToAddress(addr)

	privkey, err := hex.DecodeString(keyJSON.PrivateKey)
	if err != nil {
		return err
	}
	k.PrivateKey = crypto.ToECDSA(privkey)

	return nil
}

// newKeyUUID returns an identifier for key.
func newKeyUUID() (string, error) {
	var u [16]byte
	if _, err := rand.Read(u[:]); err != nil {
		return "", err
	}

	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", u[:4], u[4:6], u[6:8], u[8:10], u[10:]), nil
}

func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) (*key, error) {
	id, err := newKeyUUID()
	if err != nil {
		return nil, err
	}

	return &key{
		UUID:       id,
		Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
	}, nil
}

func storeNewKey(store *keyStore, secret string) (*key, Account, error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		return nil, Account{}, err
	}
	key, err := newKeyFromECDSA(privateKeyECDSA)
	if err != nil {
		return nil, Account{}, err
	}

	file, err := store.Insert(key, secret)
	if err != nil {
		return nil, Account{}, err
	}

	return key, Account{Address: key.Address, File: file}, err
}

type keyStore struct {
	baseDir string // absolute filepath to default/flagged value for eg datadir/mainnet/keystore
	scryptN int
	scryptP int
}

func newKeyStore(dir string, scryptN, scryptP int) (*keyStore, error) {
	if !filepath.IsAbs(dir) {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	return &keyStore{
		baseDir: dir,
		scryptN: scryptN,
		scryptP: scryptP,
	}, nil
}

func (store *keyStore) DecryptKey(data []byte, secret string) (*key, error) {
	key, err := decryptKey(data, secret)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (store *keyStore) Lookup(file string, secret string) (*key, error) {
	if !filepath.IsAbs(file) {
		file = filepath.Join(store.baseDir, file)
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := decryptKey(data, secret)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (store *keyStore) Insert(key *key, secret string) (file string, err error) {
	data, err := encryptKey(key, secret, store.scryptN, store.scryptP)
	if err != nil {
		return "", err
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.999999999")
	file = fmt.Sprintf("UTC--%sZ--%x", timestamp, key.Address[:])
	file = filepath.Join(store.baseDir, file)

	if err := writeKeyFile(file, data); err != nil {
		return "", err
	}
	return file, nil
}

func (store keyStore) Update(file string, key *key, secret string) error {
	data, err := encryptKey(key, secret, store.scryptN, store.scryptP)
	if err != nil {
		return err
	}

	if !filepath.IsAbs(file) {
		file = filepath.Join(store.baseDir, file)
	}
	return writeKeyFile(file, data)
}

// https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition

// web3v3 is a version 3 encrypted key store record.
type web3v3 struct {
	ID      string     `json:"id"`
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Version int        `json:"version"`
}

// web3v3 is a version 1 encrypted key store record.
type web3v1 struct {
	ID      string     `json:"id"`
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Version string     `json:"version"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

// encryptKey encrypts key as version 3.
func encryptKey(key *key, secret string, scryptN, scryptP int) ([]byte, error) {
	salt := randentropy.GetEntropyCSPRNG(32)
	derivedKey, err := scrypt.Key([]byte(secret), salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	keyBytes := crypto.FromECDSA(key.PrivateKey)

	iv := randentropy.GetEntropyCSPRNG(aes.BlockSize) // 16
	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)

	return json.Marshal(web3v3{
		ID:      key.UUID,
		Address: hex.EncodeToString(key.Address[:]),
		Crypto: cryptoJSON{
			Cipher:     "aes-128-ctr",
			CipherText: hex.EncodeToString(cipherText),
			CipherParams: cipherparamsJSON{
				IV: hex.EncodeToString(iv),
			},
			KDF: "scrypt",
			KDFParams: map[string]interface{}{
				"n":     scryptN,
				"r":     scryptR,
				"p":     scryptP,
				"dklen": scryptDKLen,
				"salt":  hex.EncodeToString(salt),
			},
			MAC: hex.EncodeToString(mac),
		},
		Version: 3,
	})
}

// Web3PrivateKey decrypts the record with secret and returns the private key.
func Web3PrivateKey(web3JSON []byte, secret string) (*ecdsa.PrivateKey, error) {
	k, err := decryptKey(web3JSON, secret)
	if err != nil {
		return nil, err
	}
	return k.PrivateKey, nil
}

// decryptKey decrypts a key from a JSON blob, returning the private key itself.
func decryptKey(web3JSON []byte, secret string) (*key, error) {
	// Parse the JSON into a simple map to fetch the key version
	m := make(map[string]interface{})
	if err := json.Unmarshal(web3JSON, &m); err != nil {
		return nil, err
	}

	// Depending on the version try to parse one way or another
	var (
		keyBytes []byte
		keyUUID  string
	)
	if version, ok := m["version"].(string); ok && version == "1" {
		w := new(web3v1)
		if err := json.Unmarshal(web3JSON, w); err != nil {
			return nil, err
		}

		keyUUID = w.ID

		var err error
		keyBytes, err = decryptKeyV1(w, secret)
		if err != nil {
			return nil, err
		}
	} else {
		w := new(web3v3)
		if err := json.Unmarshal(web3JSON, w); err != nil {
			return nil, err
		}
		if w.Version != 3 {
			return nil, fmt.Errorf("unsupported Web3 version: %v", version)
		}

		keyUUID = w.ID

		var err error
		keyBytes, err = decryptKeyV3(w, secret)
		if err != nil {
			return nil, err
		}
	}

	k := crypto.ToECDSA(keyBytes)
	return &key{
		UUID:       keyUUID,
		Address:    crypto.PubkeyToAddress(k.PublicKey),
		PrivateKey: k,
	}, nil
}

func decryptKeyV3(keyProtected *web3v3, secret string) (keyBytes []byte, err error) {
	if keyProtected.Crypto.Cipher != "aes-128-ctr" {
		return nil, fmt.Errorf("Cipher not supported: %v", keyProtected.Crypto.Cipher)
	}

	mac, err := hex.DecodeString(keyProtected.Crypto.MAC)
	if err != nil {
		return nil, err
	}

	iv, err := hex.DecodeString(keyProtected.Crypto.CipherParams.IV)
	if err != nil {
		return nil, err
	}

	cipherText, err := hex.DecodeString(keyProtected.Crypto.CipherText)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getKDFKey(keyProtected.Crypto, secret)
	if err != nil {
		return nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, ErrDecrypt
	}

	plainText, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, err
	}
	return plainText, err
}

func decryptKeyV1(keyProtected *web3v1, secret string) (keyBytes []byte, err error) {
	mac, err := hex.DecodeString(keyProtected.Crypto.MAC)
	if err != nil {
		return nil, err
	}

	iv, err := hex.DecodeString(keyProtected.Crypto.CipherParams.IV)
	if err != nil {
		return nil, err
	}

	cipherText, err := hex.DecodeString(keyProtected.Crypto.CipherText)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getKDFKey(keyProtected.Crypto, secret)
	if err != nil {
		return nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, ErrDecrypt
	}

	plainText, err := aesCBCDecrypt(crypto.Keccak256(derivedKey[:16])[:16], cipherText, iv)
	if err != nil {
		return nil, err
	}
	return plainText, err
}

func getKDFKey(cryptoJSON cryptoJSON, secret string) ([]byte, error) {
	salt, err := hex.DecodeString(cryptoJSON.KDFParams["salt"].(string))
	if err != nil {
		return nil, err
	}
	dkLen := ensureInt(cryptoJSON.KDFParams["dklen"])

	if cryptoJSON.KDF == "scrypt" {
		n := ensureInt(cryptoJSON.KDFParams["n"])
		r := ensureInt(cryptoJSON.KDFParams["r"])
		p := ensureInt(cryptoJSON.KDFParams["p"])
		return scrypt.Key([]byte(secret), salt, n, r, p, dkLen)

	} else if cryptoJSON.KDF == "pbkdf2" {
		c := ensureInt(cryptoJSON.KDFParams["c"])
		prf := cryptoJSON.KDFParams["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key([]byte(secret), salt, c, dkLen, sha256.New)
		return key, nil
	}

	return nil, fmt.Errorf("Unsupported KDF: %s", cryptoJSON.KDF)
}

// TODO: can we do without this when unmarshalling dynamic JSON?
// why do integers in KDF params end up as float64 and not int after
// unmarshal?
func ensureInt(x interface{}) int {
	res, ok := x.(int)
	if !ok {
		res = int(x.(float64))
	}
	return res
}

func writeKeyFile(file string, content []byte) error {
	dir, basename := filepath.Split(file)

	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(dir, "."+basename+".tmp")
	if err != nil {
		return err
	}

	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// BUG(pascaldekloe): Windows won't allow updates to a keyfile when it is being read.
	return os.Rename(f.Name(), file)
}
