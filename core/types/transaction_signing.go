// Copyright 2016 The go-ethereum Authors
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

package types

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/crypto"
)

var ErrInvalidChainId = errors.New("invalid chain id for signer")

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 are considered unprotected
	return true
}

// normaliseV returns the Ethereum version of the V parameter
func normaliseV(s Signer, v *big.Int) byte {
	if s, ok := s.(ChainIdSigner); ok {
		stdV := v.BitLen() <= 8 && (v.Uint64() == 27 || v.Uint64() == 28)
		if s.chainId.BitLen() > 0 && !stdV {
			nv := byte((new(big.Int).Sub(v, s.chainIdMul).Uint64()) - 35 + 27)
			return nv
		}
	}
	return byte(v.Uint64())
}

// deriveChainId derives the chain id from the given v parameter
func deriveChainId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}

// From derives the sender from the tx using the signer derivation
// functions.

// sigCache is used to cache the derived sender and contains
// the signer used to derive it.
type sigCache struct {
	signer Signer
	from   common.Address
}

// From returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// From may cache the address, allowing it to be used regardless of
// signing method.
func Sender(signer Signer, tx *Transaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	pubkey, err := signer.PublicKey(tx)
	if err != nil {
		return common.Address{}, err
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pubkey[1:])[12:])
	tx.from.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

// SignatureValues returns the ECDSA signature values contained in the transaction.
func SignatureValues(signer Signer, tx *Transaction) (v byte, r *big.Int, s *big.Int) {
	return normaliseV(signer, tx.data.V), new(big.Int).Set(tx.data.R), new(big.Int).Set(tx.data.S)
}

type Signer interface {
	// Hash returns the rlp encoded hash for signatures
	Hash(tx *Transaction) common.Hash
	// PubilcKey returns the public key derived from the signature
	PublicKey(tx *Transaction) ([]byte, error)
	// SignECDSA signs the transaction with the given and returns a copy of the tx
	SignECDSA(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error)
	// WithSignature returns a copy of the transaction with the given signature
	WithSignature(tx *Transaction, sig []byte) (*Transaction, error)
	// Checks for equality on the signers
	Equal(Signer) bool
}

// EIP155Transaction implements TransactionInterface using the
// EIP155 rules
type ChainIdSigner struct {
	BasicSigner

	chainId, chainIdMul *big.Int
}

func NewChainIdSigner(chainId *big.Int) ChainIdSigner {
	return ChainIdSigner{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}

func (s ChainIdSigner) Equal(s2 Signer) bool {
	other, ok := s2.(ChainIdSigner)
	if !ok {
		return false
	}
	if other.chainId == nil || s.chainId == nil {
		return false
	}

	return other.chainId.Cmp(s.chainId) == 0
}

func (s ChainIdSigner) SignECDSA(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return s.WithSignature(tx, sig)
}

func (s ChainIdSigner) PublicKey(tx *Transaction) ([]byte, error) {
	// if the transaction is not protected fall back to homestead signer
	if !tx.Protected() {
		return (BasicSigner{}).PublicKey(tx)
	}

	if tx.ChainId() == nil || s.chainId == nil || tx.ChainId().Cmp(s.chainId) != 0 {
		return nil, ErrInvalidChainId
	}

	V := normaliseV(s, tx.data.V)
	if !crypto.ValidateSignatureValues(V, tx.data.R, tx.data.S, true) {
		return nil, ErrInvalidSig
	}

	// encode the signature in uncompressed format
	R, S := tx.data.R.Bytes(), tx.data.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(R):32], R)
	copy(sig[64-len(S):64], S)
	sig[64] = V - 27

	// recover the public key from the signature
	hash := s.Hash(tx)
	pub, err := crypto.Ecrecover(hash[:], sig)
	if err != nil {
		return nil, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return nil, errors.New("invalid public key")
	}
	return pub, nil
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (s ChainIdSigner) WithSignature(tx *Transaction, sig []byte) (*Transaction, error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for snature: got %d, want 65", len(sig)))
	}

	cpy := &Transaction{signer: tx.signer, data: tx.data}
	cpy.data.R = new(big.Int).SetBytes(sig[:32])
	cpy.data.S = new(big.Int).SetBytes(sig[32:64])
	cpy.data.V = new(big.Int).SetBytes([]byte{sig[64]})
	if s.chainId.BitLen() > 0 {
		cpy.data.V = big.NewInt(int64(sig[64] + 35))
		cpy.data.V.Add(cpy.data.V, s.chainIdMul)
	}
	return cpy, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s ChainIdSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
		s.chainId, uint(0), uint(0),
	})
}

func (s ChainIdSigner) SigECDSA(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return s.WithSignature(tx, sig)
}

type BasicSigner struct{}

func (s BasicSigner) Equal(s2 Signer) bool {
	_, ok := s2.(BasicSigner)
	return ok
}

// WithSignature returns a new transaction with the given snature.
// This snature needs to be formatted as described in the yellow paper (v+27).
func (fs BasicSigner) WithSignature(tx *Transaction, sig []byte) (*Transaction, error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for snature: got %d, want 65", len(sig)))
	}
	cpy := &Transaction{signer: tx.signer, data: tx.data}
	cpy.data.R = new(big.Int).SetBytes(sig[:32])
	cpy.data.S = new(big.Int).SetBytes(sig[32:64])
	cpy.data.V = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return cpy, nil
}

func (fs BasicSigner) SignECDSA(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := fs.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return fs.WithSignature(tx, sig)
}

// Hash returns the hash to be sned by the sender.
// It does not uniquely identify the transaction.
func (fs BasicSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.data.AccountNonce,
		tx.data.Price,
		tx.data.GasLimit,
		tx.data.Recipient,
		tx.data.Amount,
		tx.data.Payload,
	})
}

func (fs BasicSigner) PublicKey(tx *Transaction) ([]byte, error) {
	if tx.data.V.BitLen() > 8 {
		return nil, ErrInvalidSig
	}

	V := byte(tx.data.V.Uint64())
	if !crypto.ValidateSignatureValues(V, tx.data.R, tx.data.S, false) {
		return nil, ErrInvalidSig
	}
	// encode the snature in uncompressed format
	r, s := tx.data.R.Bytes(), tx.data.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V - 27

	// recover the public key from the snature
	hash := fs.Hash(tx)
	pub, err := crypto.Ecrecover(hash[:], sig)
	if err != nil {
		return nil, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return nil, errors.New("invalid public key")
	}
	return pub, nil
}
