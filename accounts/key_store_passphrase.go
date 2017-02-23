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
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/crypto/randentropy"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

const (
	// n,r,p = 2^18, 8, 1 uses 256MB memory and approx 1s CPU time on a modern CPU.
	StandardScryptN = 1 << 18
	StandardScryptP = 1

	// n,r,p = 2^12, 8, 6 uses 4MB memory and approx 100ms CPU time on a modern CPU.
	LightScryptN = 1 << 12
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
)

// keyStorePassphrase behaves as keyStorePlain with the difference that
// the private key is encrypted and on disk uses another JSON encoding.
type keyStorePassphrase struct {
	keysDirPath string
	scryptN     int
	scryptP     int
}

func (ks keyStorePassphrase) GetKey(addr common.Address, filename string, secret string) (*key, error) {
	// Load the key from the keystore and decrypt its contents
	keyjson, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	key, err := decryptKey(keyjson, secret)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

func (ks keyStorePassphrase) StoreKey(filename string, key *key, secret string) error {
	keyjson, err := encryptKey(key, secret, ks.scryptN, ks.scryptP)
	if err != nil {
		return err
	}
	return writeKeyFile(filename, keyjson)
}

func (ks keyStorePassphrase) JoinPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	} else {
		return filepath.Join(ks.keysDirPath, filename)
	}
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
			return nil, fmt.Errorf("unsupported Web3 version: %d", version)
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
