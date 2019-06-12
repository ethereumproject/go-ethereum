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

package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/common/hexutil"
	"github.com/eth-classic/go-ethereum/crypto/secp256k1"
)

var testAddrHex = "970e8128ab834e8eac17ab8e3812f010678cf791"
var testPrivHex = "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"

var (
	testmsg     = hexutil.MustDecode("0xce0677bb30baa8cf067c88db9811f4333d131bf8bcf12fe7065d211dce971008")
	testsig     = hexutil.MustDecode("0x90f27b8b488db00b00606796d2987f6a5f59ae62ea05effe84fef5b8b0e549984a691139ad57a3f0b906637673aa2f63d1f55cb1a69199d4009eea23ceaddc9301")
	testpubkey  = hexutil.MustDecode("0x04e32df42865e97135acfb65f3bae71bdc86f4d49150ad6a440b6f15878109880a0a2b2667f7e725ceea70c673093bf67663e0312623c8e091b13cf2c0f11ef652")
	testpubkeyc = hexutil.MustDecode("0x02e32df42865e97135acfb65f3bae71bdc86f4d49150ad6a440b6f15878109880a")
)

func TestEcrecover(t *testing.T) {
	pubkey, err := Ecrecover(testmsg, testsig)
	if err != nil {
		t.Fatalf("recover error: %s", err)
	}
	if !bytes.Equal(pubkey, testpubkey) {
		t.Errorf("pubkey mismatch: want: %x have: %x", testpubkey, pubkey)
	}
}

// These tests are sanity checks.
// They should ensure that we don't e.g. use Sha3-224 instead of Sha3-256
// and that the sha3 library uses keccak-f permutation.
func TestSha3(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	checkhash(t, "Sha3-256", func(in []byte) []byte { return Keccak256(in) }, msg, exp)
}

func TestSha3Hash(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	checkhash(t, "Sha3-256-array", func(in []byte) []byte { h := Keccak256Hash(in); return h[:] }, msg, exp)
}

func TestSha256(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
	checkhash(t, "Sha256", Sha256, msg, exp)
}

func TestRipemd160(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("8eb208f7e05d987a9b044a8e98c6b087f15a0bfc")
	checkhash(t, "Ripemd160", Ripemd160, msg, exp)
}

func BenchmarkSha3(b *testing.B) {
	a := []byte("hello world")
	amount := 1000000
	start := time.Now()
	for i := 0; i < amount; i++ {
		Keccak256(a)
	}

	fmt.Println(amount, ":", time.Since(start))
}

func Test0Key(t *testing.T) {
	key := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")
	_, err := secp256k1.GeneratePubKey(key)
	if err == nil {
		t.Errorf("expected error due to zero privkey")
	}
}

func TestSign(t *testing.T) {
	key, _ := HexToECDSA(testPrivHex)
	addr := common.HexToAddress(testAddrHex)

	msg := Keccak256([]byte("foo"))
	sig, err := Sign(msg, key)
	if err != nil {
		t.Errorf("Sign error: %s", err)
	}
	if len(sig) != 65 {
		t.Error("wrong signature length", len(sig))
	}
	recoveredPub, err := Ecrecover(msg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	recoveredAddr := PubkeyToAddress(*ToECDSAPub(recoveredPub))
	if addr != recoveredAddr {
		t.Errorf("Address mismatch: want: %x have: %x", addr, recoveredAddr)
	}

	// should be equal to SigToPub
	recoveredPub2, err := SigToPub(msg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	recoveredAddr2 := PubkeyToAddress(*recoveredPub2)
	if addr != recoveredAddr2 {
		t.Errorf("Address mismatch: want: %x have: %x", addr, recoveredAddr2)
	}

}

func TestInvalidSign(t *testing.T) {
	_, err := Sign(make([]byte, 1), nil)
	if err == nil {
		t.Errorf("expected sign with hash 1 byte to error")
	}

	_, err = Sign(make([]byte, 33), nil)
	if err == nil {
		t.Errorf("expected sign with hash 33 byte to error")
	}
}

func TestNewContractAddress(t *testing.T) {
	key, _ := HexToECDSA(testPrivHex)
	addr := common.HexToAddress(testAddrHex)
	genAddr := PubkeyToAddress(key.PublicKey)
	// sanity check before using addr to create contract address
	checkAddr(t, genAddr, addr)

	caddr0 := CreateAddress(addr, 0)
	caddr1 := CreateAddress(addr, 1)
	caddr2 := CreateAddress(addr, 2)
	checkAddr(t, common.HexToAddress("333c3310824b7c685133f2bedb2ca4b8b4df633d"), caddr0)
	checkAddr(t, common.HexToAddress("8bda78331c916a08481428e4b07c96d3e916d165"), caddr1)
	checkAddr(t, common.HexToAddress("c9ddedf451bc62ce88bf9292afb13df35b670699"), caddr2)
}

func TestLoadECDSAFile(t *testing.T) {
	keyBytes := common.FromHex(testPrivHex)
	fileName0 := "test_key0"
	fileName1 := "test_key1"
	checkKey := func(k *ecdsa.PrivateKey) {
		checkAddr(t, PubkeyToAddress(k.PublicKey), common.HexToAddress(testAddrHex))
		loadedKeyBytes := FromECDSA(k)
		if !bytes.Equal(loadedKeyBytes, keyBytes) {
			t.Fatalf("private key mismatch: want: %x have: %x", keyBytes, loadedKeyBytes)
		}
	}

	ioutil.WriteFile(fileName0, []byte(testPrivHex), 0600)
	defer os.Remove(fileName0)

	f, err := os.Open(fileName0)
	if err != nil {
		t.Fatal(err)
	}
	key0, err := LoadECDSA(f)
	if e := f.Close(); e != nil {
		t.Fatal(e)
	}
	if err != nil {
		t.Fatal(err)
	}
	checkKey(key0)

	// again, this time with WriteECDSAKey instead of manual save:
	f, err = os.Create(fileName1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = WriteECDSAKey(f, key0)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName1)

	f1, err := os.Open(fileName1)
	if err != nil {
		t.Fatal(err)
	}
	key1, err := LoadECDSA(f1)
	if err != nil {
		t.Fatal(err)
	}
	if err := f1.Close(); err != nil {
		t.Fatal(err)
	}
	checkKey(key1)
}

func TestValidateSignatureValues(t *testing.T) {
	check := func(expected bool, v byte, r, s *big.Int) {
		if ValidateSignatureValues(v, r, s, false) != expected {
			t.Errorf("mismatch for v: %d r: %d s: %d want: %v", v, r, s, expected)
		}
	}
	minusOne := big.NewInt(-1)
	one := common.Big1
	zero := new(big.Int)
	secp256k1nMinus1 := new(big.Int).Sub(secp256k1.N, common.Big1)

	// correct v,r,s
	check(true, 27, one, one)
	check(true, 28, one, one)
	// incorrect v, correct r,s,
	check(false, 30, one, one)
	check(false, 26, one, one)

	// incorrect v, combinations of incorrect/correct r,s at lower limit
	check(false, 0, zero, zero)
	check(false, 0, zero, one)
	check(false, 0, one, zero)
	check(false, 0, one, one)

	// correct v for any combination of incorrect r,s
	check(false, 27, zero, zero)
	check(false, 27, zero, one)
	check(false, 27, one, zero)

	check(false, 28, zero, zero)
	check(false, 28, zero, one)
	check(false, 28, one, zero)

	// correct sig with max r,s
	check(true, 27, secp256k1nMinus1, secp256k1nMinus1)
	// correct v, combinations of incorrect r,s at upper limit
	check(false, 27, secp256k1.N, secp256k1nMinus1)
	check(false, 27, secp256k1nMinus1, secp256k1.N)
	check(false, 27, secp256k1.N, secp256k1.N)

	// current callers ensures r,s cannot be negative, but let's test for that too
	// as crypto package could be used stand-alone
	check(false, 27, minusOne, one)
	check(false, 27, one, minusOne)
}

func checkhash(t *testing.T, name string, f func([]byte) []byte, msg, exp []byte) {
	sum := f(msg)
	if bytes.Compare(exp, sum) != 0 {
		t.Fatalf("hash %s mismatch: want: %x have: %x", name, exp, sum)
	}
}

func checkAddr(t *testing.T, addr0, addr1 common.Address) {
	if addr0 != addr1 {
		t.Fatalf("address mismatch: want: %x have: %x", addr0, addr1)
	}
}

// test to help Python team with integration of libsecp256k1
// skip but keep it after they are done
func TestPythonIntegration(t *testing.T) {
	kh := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	k0, _ := HexToECDSA(kh)
	k1 := FromECDSA(k0)

	msg0 := Keccak256([]byte("foo"))
	sig0, _ := secp256k1.Sign(msg0, k1)

	msg1 := common.FromHex("00000000000000000000000000000000")
	sig1, _ := secp256k1.Sign(msg0, k1)

	fmt.Printf("msg: %x, privkey: %x sig: %x\n", msg0, k1, sig0)
	fmt.Printf("msg: %x, privkey: %x sig: %x\n", msg1, k1, sig1)
}
