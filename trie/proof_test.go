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

package trie

import (
	"bytes"
	crand "crypto/rand"
	"errors"
	"fmt"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/crypto/sha3"
	"github.com/ethereumproject/go-ethereum/rlp"
)

func init() {
	mrand.Seed(time.Now().Unix())
}

func TestProof(t *testing.T) {
	trie, vals := randomTrie(500)
	root := trie.Hash()
	for _, kv := range vals {
		proof := trie.Prove(kv.k)
		if proof == nil {
			t.Fatalf("missing key %x while constructing proof", kv.k)
		}
		val, err := verifyProof(root, kv.k, proof)
		if err != nil {
			t.Fatalf("VerifyProof error for key %x: %v\nraw proof: %x", kv.k, err, proof)
		}
		if !bytes.Equal(val, kv.v) {
			t.Fatalf("VerifyProof returned wrong value for key %x: got %x, want %x", kv.k, val, kv.v)
		}
	}
}

func TestOneElementProof(t *testing.T) {
	trie := new(Trie)
	updateString(trie, "k", "v")
	proof := trie.Prove([]byte("k"))
	if proof == nil {
		t.Fatal("nil proof")
	}
	if len(proof) != 1 {
		t.Error("proof should have one element")
	}
	val, err := verifyProof(trie.Hash(), []byte("k"), proof)
	if err != nil {
		t.Fatalf("VerifyProof error: %v\nraw proof: %x", err, proof)
	}
	if !bytes.Equal(val, []byte("v")) {
		t.Fatalf("VerifyProof returned wrong value: got %x, want 'k'", val)
	}
}

func TestVerifyBadProof(t *testing.T) {
	trie, vals := randomTrie(800)
	root := trie.Hash()
	for _, kv := range vals {
		proof := trie.Prove(kv.k)
		if proof == nil {
			t.Fatal("nil proof")
		}
		mutateByte(proof[mrand.Intn(len(proof))])
		if _, err := verifyProof(root, kv.k, proof); err == nil {
			t.Fatalf("expected proof to fail for key %x", kv.k)
		}
	}
}

// mutateByte changes one byte in b.
func mutateByte(b []byte) {
	for r := mrand.Intn(len(b)); ; {
		new := byte(mrand.Intn(255))
		if new != b[r] {
			b[r] = new
			break
		}
	}
}

func BenchmarkProve(b *testing.B) {
	trie, vals := randomTrie(100)
	var keys []string
	for k := range vals {
		keys = append(keys, k)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kv := vals[keys[i%len(keys)]]
		if trie.Prove(kv.k) == nil {
			b.Fatalf("nil proof for %x", kv.k)
		}
	}
}

func randomTrie(n int) (*Trie, map[string]*kv) {
	trie := new(Trie)
	vals := make(map[string]*kv)
	for i := byte(0); i < 100; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{i + 10}, 32), []byte{i}, false}
		trie.Update(value.k, value.v)
		trie.Update(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}
	for i := 0; i < n; i++ {
		value := &kv{randBytes(32), randBytes(20), false}
		trie.Update(value.k, value.v)
		vals[string(value.k)] = value
	}
	return trie, vals
}

func randBytes(n int) []byte {
	r := make([]byte, n)
	crand.Read(r)
	return r
}

// VerifyProof checks merkle proofs. The given proof must contain the
// value for key in a trie with the given root hash. VerifyProof
// returns an error if the proof contains invalid trie nodes or the
// wrong value.
func verifyProof(rootHash common.Hash, key []byte, proof []rlp.RawValue) (value []byte, err error) {
	key = compactHexDecode(key)
	sha := sha3.NewKeccak256()
	wantHash := rootHash.Bytes()
	for i, buf := range proof {
		sha.Reset()
		sha.Write(buf)
		if !bytes.Equal(sha.Sum(nil), wantHash) {
			return nil, fmt.Errorf("bad proof node %d: hash mismatch", i)
		}
		n, err := decodeNode(wantHash, buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %d: %v", i, err)
		}
		keyrest, cld := get(n, key)
		switch cld := cld.(type) {
		case nil:
			if i != len(proof)-1 {
				return nil, fmt.Errorf("key mismatch at proof node %d", i)
			} else {
				// The trie doesn't contain the key.
				return nil, nil
			}
		case hashNode:
			key = keyrest
			wantHash = cld
		case valueNode:
			if i != len(proof)-1 {
				return nil, errors.New("additional nodes at end of proof")
			}
			return cld, nil
		}
	}
	return nil, errors.New("unexpected end of proof")
}

func get(tn node, key []byte) ([]byte, node) {
	for len(key) > 0 {
		switch n := tn.(type) {
		case *shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				return nil, nil
			}
			tn = n.Val
			key = key[len(n.Key):]
		case *fullNode:
			tn = n.Children[key[0]]
			key = key[1:]
		case hashNode:
			return key, n
		case nil:
			return key, nil
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
	return nil, tn.(valueNode)
}
