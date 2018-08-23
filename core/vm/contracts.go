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
	"github.com/ethereumproject/go-ethereum/crypto/bn256"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"math"
)

// PrecompiledAccount represents a native ethereum contract
type PrecompiledAccount struct {
	Gas func(in []byte) *big.Int
	fn  func(in []byte) []byte
}

// Call calls the native function
func (pa PrecompiledAccount) Call(in []byte) ([]byte, error) {
	return pa.fn(in)
}

// Precompiled contains the default set of ethereum contracts
var Precompiled = PrecompiledContracts()

func preCByteAddress(i []byte) string {
	return string(common.LeftPadBytes(i, 20))
}

// PrecompiledContracts returns the default set of precompiled ethereum
// contracts defined by the ethereum yellow paper.
func PrecompiledContracts() map[string]*PrecompiledAccount {
	return map[string]*PrecompiledAccount{
		// ECRECOVER
		preCByteAddress([]byte{1}): {func(in []byte) *big.Int {
			l := len(in)
			return big.NewInt(3000)
		}, ecrecoverFunc},

		// SHA256
		preCByteAddress([]byte{2}): {func(in []byte) *big.Int {
			l := len(in)
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, big.NewInt(12))
			return n.Add(n, big.NewInt(60))
		}, sha256Func},

		// RIPEMD160
		preCByteAddress([]byte{3}): {func(in []byte) *big.Int {
			l := len(in)
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, big.NewInt(120))
			return n.Add(n, big.NewInt(600))
		}, ripemd160Func},

		preCByteAddress([]byte{4}): {func(in []byte) *big.Int {
			l := len(in)
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, big.NewInt(3))
			return n.Add(n, big.NewInt(15))
		}, memCpy},
	}
}

var (
	big8    = big.NewInt(8)
	big32   = big.NewInt(32)
	big64   = big.NewInt(64)
	big1024 = big.NewInt(1024)
)

func PrecompiledContractsECIP1045() map[string]*PrecompiledAccount {
	contracts := PrecompiledContracts()
	// gas functions
	bigMax := func(x, y *big.Int) *big.Int {
		if x.Cmp(y) < 0 {
			return y
		}
		return x
	}

	bigModExpGas := func(input []byte) *big.Int {
		var (
			baseLen = new(big.Int).SetBytes(getData(input, big.NewInt(0), big.NewInt(32)))
			expLen  = new(big.Int).SetBytes(getData(input, big.NewInt(32), big.NewInt(32)))
			modLen  = new(big.Int).SetBytes(getData(input, big.NewInt(64), big.NewInt(32)))
		)
		if len(input) > 96 {
			input = input[96:]
		} else {
			input = input[:0]
		}
		// Retrieve the head 32 bytes of exp for the adjusted exponent length
		var expHead *big.Int
		if big.NewInt(int64(len(input))).Cmp(baseLen) <= 0 {
			expHead = new(big.Int)
		} else {
			if expLen.Cmp(big32) > 0 {
				expHead = new(big.Int).SetBytes(getData(input, baseLen, big32))
			} else {
				expHead = new(big.Int).SetBytes(getData(input, baseLen, expLen))
			}
		}
		// Calculate the adjusted exponent length
		var msb int
		if bitlen := expHead.BitLen(); bitlen > 0 {
			msb = bitlen - 1
		}
		adjExpLen := new(big.Int)
		if expLen.Cmp(big32) > 0 {
			adjExpLen.Sub(expLen, big32)
			adjExpLen.Mul(big8, adjExpLen)
		}
		adjExpLen.Add(adjExpLen, big.NewInt(int64(msb)))

		// Calculate the gas cost of the operation
		gas := new(big.Int).Set(bigMax(modLen, baseLen))
		switch {
		case gas.Cmp(big64) <= 0:
			gas.Mul(gas, gas)
		case gas.Cmp(big1024) <= 0:
			gas = new(big.Int).Add(
				new(big.Int).Div(new(big.Int).Mul(gas, gas), big.NewInt(4)),
				new(big.Int).Sub(new(big.Int).Mul(big.NewInt(96), gas), big.NewInt(3072)),
			)
		default:
			gas = new(big.Int).Add(
				new(big.Int).Div(new(big.Int).Mul(gas, gas), big.NewInt(16)),
				new(big.Int).Sub(new(big.Int).Mul(big.NewInt(480), gas), big.NewInt(199680)),
			)
		}
		gas.Mul(gas, bigMax(adjExpLen, big.NewInt(1)))
		gas.Div(gas, big.NewInt(20)) // ModExpQuadCoeffDiv

		if gas.BitLen() > 64 {
			return new(big.Int).SetUint64(math.MaxUint64)
		}
		return gas
	}
	bn256AddGas := func(in []byte) *big.Int {
		return big.NewInt(500) // params.Bn256AddGas
	}
	bn256ScalarMulGas := func(in []byte) *big.Int {
		return big.NewInt(40000) // params.Bn256ScalarMulGas
	}
	bn256PairingGas := func(in []byte) *big.Int {
		// return params.Bn256PairingBaseGas + uint64(len(input)/192)*params.Bn256PairingPerPointGas
		base := big.NewInt(100000)
		perPoint := big.NewInt(80000)
		lDiv := new(big.Int).SetUint64(uint64(len(in) / 192))
		out := new(big.Int).Add(base, new(big.Int).Mul(lDiv, perPoint))
		return out
	}

	// bigModExp
	contracts[preCByteAddress([]byte{5})] = &PrecompiledAccount{
		Gas: bigModExpGas,
		fn:  bigModExpFunc,
	}
	// bn256Add
	contracts[preCByteAddress([]byte{6})] = &PrecompiledAccount{
		Gas: bn256AddGas,
		fn:  bn256AddFunc,
	}
	// bn256ScalarMul
	contracts[preCByteAddress([]byte{7})] = &PrecompiledAccount{
		Gas: bn256ScalarMulGas,
		fn:  bn256ScalarMulFunc,
	}
	// pairing
	contracts[preCByteAddress([]byte{8})] = &PrecompiledAccount{
		Gas: bn256PairingGas,
		fn:  bn256PairingFunc,
	}
	return contracts
}

func bigModExpFunc(input []byte) []byte {
	var (
		baseLen = new(big.Int).SetBytes(getData(input, new(big.Int), big32))
		expLen  = new(big.Int).SetBytes(getData(input, big32, big32))
		modLen  = new(big.Int).SetBytes(getData(input, big64, big32))
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Handle a special case when both the base and mod length is zero
	if baseLen.Cmp(new(big.Int)) == 0 && modLen.Cmp(new(big.Int)) == 0 {
		return []byte{}
	}
	// Retrieve the operands and execute the exponentiation
	var (
		base = new(big.Int).SetBytes(getData(input, new(big.Int), baseLen))
		exp  = new(big.Int).SetBytes(getData(input, baseLen, expLen))
		mod  = new(big.Int).SetBytes(getData(input, new(big.Int).Add(baseLen, expLen), modLen))
	)
	if mod.BitLen() == 0 {
		// Modulo 0 is undefined, return zero
		return common.LeftPadBytes([]byte{}, int(modLen.Int64()))
	}
	return common.LeftPadBytes(base.Exp(base, exp, mod).Bytes(), int(modLen.Int64()))
}

// newCurvePoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newCurvePoint(blob []byte) (*bn256.G1, error) {
	p := new(bn256.G1)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

// newTwistPoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newTwistPoint(blob []byte) (*bn256.G2, error) {
	p := new(bn256.G2)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

func bn256AddFunc(in []byte) []byte {

}
func bn256ScalarMulFunc(in []byte) []byte {

}
func bn256PairingFunc(in []byte) []byte {

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
