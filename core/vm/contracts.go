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

package vm

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

// PrecompiledAccount represents a native ethereum contract
type PrecompiledAccount struct {
	Gas func(l int) *big.Int
	fn  func(in []byte) []byte
}

// Call calls the native function
func (self PrecompiledAccount) Call(in []byte) []byte {
	return self.fn(in)
}

// Precompiled contains the default set of ethereum contracts
var Precompiled = PrecompiledContracts()

// PrecompiledContracts returns the default set of precompiled ethereum
// contracts defined by the ethereum yellow paper.
func PrecompiledContracts() map[string]*PrecompiledAccount {
	return map[string]*PrecompiledAccount{
		// ECRECOVER
		string(common.LeftPadBytes([]byte{1}, 20)): {func(l int) *big.Int {
			return big.NewInt(3000)
		}, ecrecoverFunc},

		// SHA256
		string(common.LeftPadBytes([]byte{2}, 20)): {func(l int) *big.Int {
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, big.NewInt(12))
			return n.Add(n, big.NewInt(60))
		}, sha256Func},

		// RIPEMD160
		string(common.LeftPadBytes([]byte{3}, 20)): {func(l int) *big.Int {
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, big.NewInt(120))
			return n.Add(n, big.NewInt(600))
		}, ripemd160Func},

		string(common.LeftPadBytes([]byte{4}, 20)): {func(l int) *big.Int {
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, big.NewInt(3))
			return n.Add(n, big.NewInt(15))
		}, memCpy},
	}
}

func sha256Func(in []byte) []byte {
	return crypto.Sha256(in)
}

func ripemd160Func(in []byte) []byte {
	return common.LeftPadBytes(crypto.Ripemd160(in), 32)
}

func ecrecoverFunc(in []byte) []byte {
	in = common.RightPadBytes(in, 128)
	// "in" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(in[64:96])
	s := new(big.Int).SetBytes(in[96:128])
	// Treat V as a 256bit integer
	vbig := new(big.Int).SetBytes(in[32:64])
	v := byte(vbig.Uint64())

	// tighter sig s values in homestead only apply to tx sigs
	if !crypto.ValidateSignatureValues(v, r, s, false) {
		glog.V(logger.Detail).Infof("ECRECOVER error: v, r or s value invalid")
		return nil
	}

	// v needs to be at the end and normalized for libsecp256k1
	vbignormal := new(big.Int).Sub(vbig, big.NewInt(27))
	vnormal := byte(vbignormal.Uint64())
	rsv := append(in[64:128], vnormal)
	pubKey, err := crypto.Ecrecover(in[:32], rsv)
	// make sure the public key is a valid one
	if err != nil {
		glog.V(logger.Detail).Infoln("ECRECOVER error: ", err)
		return nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(crypto.Keccak256(pubKey[1:])[12:], 32)
}

func memCpy(in []byte) []byte {
	return in
}
