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
	"errors"
	"math/big"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/crypto"
	"github.com/eth-classic/go-ethereum/crypto/bn256"
	"github.com/eth-classic/go-ethereum/logger"
	"github.com/eth-classic/go-ethereum/logger/glog"
)

var (
	big0      = big.NewInt(0)
	big1      = big.NewInt(1)
	big4      = big.NewInt(4)
	big8      = big.NewInt(8)
	big16     = big.NewInt(16)
	big32     = big.NewInt(32)
	big64     = big.NewInt(64)
	big96     = big.NewInt(96)
	big480    = big.NewInt(480)
	big1024   = big.NewInt(1024)
	big3072   = big.NewInt(3072)
	big199680 = big.NewInt(199680)
)

const (
	EcrecoverGas            uint64 = 3000   // Elliptic curve sender recovery gas price
	Sha256BaseGas           uint64 = 60     // Base price for a SHA256 operation
	Sha256PerWordGas        uint64 = 12     // Per-word price for a SHA256 operation
	Ripemd160BaseGas        uint64 = 600    // Base price for a RIPEMD160 operation
	Ripemd160PerWordGas     uint64 = 120    // Per-word price for a RIPEMD160 operation
	IdentityBaseGas         uint64 = 15     // Base price for a data copy operation
	IdentityPerWordGas      uint64 = 3      // Per-work price for a data copy operation
	ModExpQuadCoeffDiv      uint64 = 20     // Divisor for the quadratic particle of the big int modular exponentiation
	Bn256AddGas             uint64 = 500    // Gas needed for an elliptic curve addition
	Bn256ScalarMulGas       uint64 = 40000  // Gas needed for an elliptic curve scalar multiplication
	Bn256PairingBaseGas     uint64 = 100000 // Base price for an elliptic curve pairing check
	Bn256PairingPerPointGas uint64 = 80000  // Per-point price for an elliptic curve pairing check
)

// PrecompiledAccount represents a native ethereum contract
type PrecompiledAccount struct {
	Gas func(in []byte) *big.Int
	fn  func(in []byte) ([]byte, error)
}

// Call calls the native function
func (self PrecompiledAccount) Call(in []byte) ([]byte, error) {
	return self.fn(in)
}

// Precompiled contains the default set of ethereum contracts
var PrecompiledPreAtlantis = PrecompiledContracts()
var PrecompiledAtlantis = func() map[string]*PrecompiledAccount {
	a := PrecompiledContracts()
	b := PrecompiledContractsAtlantis()
	precompiles := make(map[string]*PrecompiledAccount)
	for k, c := range a {
		precompiles[k] = c
	}
	for k, c := range b {
		precompiles[k] = c
	}
	return precompiles
}()

// PrecompiledContractsPreAtlantis returns the default set of precompiled ethereum
// contracts defined by the ethereum yellow paper pre-Atlantis.
func PrecompiledContracts() map[string]*PrecompiledAccount {
	return map[string]*PrecompiledAccount{
		// ECRECOVER
		string(common.LeftPadBytes([]byte{1}, 20)): {func(in []byte) *big.Int {
			return big.NewInt(3000)
		}, ecrecoverFunc},

		// SHA256
		string(common.LeftPadBytes([]byte{2}, 20)): {func(in []byte) *big.Int {
			l := len(in)
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, big.NewInt(12))
			return n.Add(n, big.NewInt(60))
		}, sha256Func},

		// RIPEMD160
		string(common.LeftPadBytes([]byte{3}, 20)): {func(in []byte) *big.Int {
			l := len(in)
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, big.NewInt(120))
			return n.Add(n, big.NewInt(600))
		}, ripemd160Func},

		// memCpy
		string(common.LeftPadBytes([]byte{4}, 20)): {func(in []byte) *big.Int {
			l := len(in)
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, big.NewInt(3))
			return n.Add(n, big.NewInt(15))
		}, memCpy},
	}
}

// PrecompiledContractsAtlantis returns the set of precompiled contracts introducted in Atlantis
func PrecompiledContractsAtlantis() map[string]*PrecompiledAccount {
	return map[string]*PrecompiledAccount{
		// bigModExp
		string(common.LeftPadBytes([]byte{5}, 20)): {func(in []byte) *big.Int {
			var (
				baseLen = new(big.Int).SetBytes(getData(in, big.NewInt(0), big32))
				expLen  = new(big.Int).SetBytes(getData(in, big32, big32))
				modLen  = new(big.Int).SetBytes(getData(in, big64, big32))
			)
			if len(in) > 96 {
				in = in[96:]
			} else {
				in = in[:0]
			}
			// Retrieve the head 32 bytes of exp for the adjusted exponent length
			var expHead *big.Int
			if big.NewInt(int64(len(in))).Cmp(baseLen) <= 0 {
				expHead = new(big.Int)
			} else {
				if expLen.Cmp(big32) > 0 {
					expHead = new(big.Int).SetBytes(getData(in, baseLen, big32))
				} else {
					expHead = new(big.Int).SetBytes(getData(in, baseLen, expLen))
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
			gas := new(big.Int).Set(common.BigMax(modLen, baseLen))
			switch {
			case gas.Cmp(big64) <= 0:
				gas.Mul(gas, gas)
			case gas.Cmp(big1024) <= 0:
				gas = new(big.Int).Add(
					new(big.Int).Div(new(big.Int).Mul(gas, gas), big4),
					new(big.Int).Sub(new(big.Int).Mul(big96, gas), big3072),
				)
			default:
				gas = new(big.Int).Add(
					new(big.Int).Div(new(big.Int).Mul(gas, gas), big16),
					new(big.Int).Sub(new(big.Int).Mul(big480, gas), big199680),
				)
			}
			gas.Mul(gas, common.BigMax(adjExpLen, big1))
			gas.Div(gas, new(big.Int).SetUint64(ModExpQuadCoeffDiv))

			if gas.BitLen() > 64 {
				return big.NewInt(1<<63 - 1)
			}
			return gas
		}, bigModExp},

		// bn256Add
		string(common.LeftPadBytes([]byte{6}, 20)): {func(in []byte) *big.Int {
			return big.NewInt(500)
		}, bn256Add},

		// bn256ScalarMul
		string(common.LeftPadBytes([]byte{7}, 20)): {func(in []byte) *big.Int {
			return big.NewInt(40000)
		}, bn256ScalarMul},

		// bn256Pairing
		string(common.LeftPadBytes([]byte{8}, 20)): {func(in []byte) *big.Int {
			l := len(in)
			n := big.NewInt(100000)
			p := big.NewInt(int64(l / 192))
			p.Mul(p, big.NewInt(80000))
			return n.Add(n, p)
		}, bn256Pairing},
	}
}

func sha256Func(in []byte) ([]byte, error) {
	return crypto.Sha256(in), nil
}

func ripemd160Func(in []byte) ([]byte, error) {
	return common.LeftPadBytes(crypto.Ripemd160(in), 32), nil
}

func ecrecoverFunc(in []byte) ([]byte, error) {
	in = common.RightPadBytes(in, 128)
	// "in" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(in[64:96])
	s := new(big.Int).SetBytes(in[96:128])
	// Treat V as a 256bit integer
	vbig := new(big.Int).SetBytes(in[32:64])
	v := byte(vbig.Uint64())

	// tighter sig s values in homestead only apply to tx sigs
	if !allZero(in[32:63]) || !crypto.ValidateSignatureValues(v, r, s, false) {
		glog.V(logger.Detail).Infof("ECRECOVER error: v, r or s value invalid")
		return nil, nil
	}

	// v needs to be at the end and normalized for libsecp256k1
	vbignormal := new(big.Int).Sub(vbig, big.NewInt(27))
	vnormal := byte(vbignormal.Uint64())
	rsv := append(in[64:128], vnormal)
	pubKey, err := crypto.Ecrecover(in[:32], rsv)
	// make sure the public key is a valid one
	if err != nil {
		glog.V(logger.Detail).Infoln("ECRECOVER error: ", err)
		return nil, nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(crypto.Keccak256(pubKey[1:])[12:], 32), nil
}

func memCpy(in []byte) ([]byte, error) {
	return in, nil
}

func bigModExp(in []byte) ([]byte, error) {
	var (
		baseLen = new(big.Int).SetBytes(getData(in, big0, big32))
		expLen  = new(big.Int).SetBytes(getData(in, big32, big32))
		modLen  = new(big.Int).SetBytes(getData(in, big64, big32))
	)

	if len(in) > 96 {
		in = in[96:]
	} else {
		in = in[:0]
	}

	// Handle a special case when both the base and mod length is zero
	if baseLen.Cmp(big0) == 0 && modLen.Cmp(big0) == 0 {
		return []byte{}, nil
	}
	// Retrieve the operands and execute the exponentiation
	var (
		base = new(big.Int).SetBytes(getData(in, big0, baseLen))
		exp  = new(big.Int).SetBytes(getData(in, baseLen, expLen))
		mod  = new(big.Int).SetBytes(getData(in, big.NewInt(0).Add(baseLen, expLen), modLen))
	)
	if mod.BitLen() == 0 {
		// Modulo 0 is undefined, return zero
		return common.LeftPadBytes([]byte{}, int(modLen.Int64())), nil
	}
	return common.LeftPadBytes(base.Exp(base, exp, mod).Bytes(), int(modLen.Int64())), nil
}

var (
	// true32Byte is returned if the bn256 pairing check succeeds.
	true32Byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	// false32Byte is returned if the bn256 pairing check fails.
	false32Byte = make([]byte, 32)

	// errBadPairingInput is returned if the bn256 pairing input is invalid.
	errBadPairingInput = errors.New("bad elliptic curve pairing size")
)

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

func bn256Add(in []byte) ([]byte, error) {
	x, err := newCurvePoint(getData(in, big.NewInt(0), big.NewInt(64)))
	if err != nil {
		return nil, err
	}
	y, err := newCurvePoint(getData(in, big.NewInt(64), big.NewInt(64)))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.Add(x, y)
	return res.Marshal(), nil
}

func bn256ScalarMul(in []byte) ([]byte, error) {
	p, err := newCurvePoint(getData(in, big.NewInt(0), big.NewInt(64)))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.ScalarMult(p, new(big.Int).SetBytes(getData(in, big.NewInt(64), big.NewInt(32))))
	return res.Marshal(), nil
}

func bn256Pairing(in []byte) ([]byte, error) {
	// Handle some corner cases cheaply
	if len(in)%192 > 0 {
		return nil, errBadPairingInput
	}
	// Convert the input into a set of coordinates
	var (
		cs []*bn256.G1
		ts []*bn256.G2
	)
	for i := 0; i < len(in); i += 192 {
		c, err := newCurvePoint(in[i : i+64])
		if err != nil {
			return nil, err
		}
		t, err := newTwistPoint(in[i+64 : i+192])
		if err != nil {
			return nil, err
		}
		cs = append(cs, c)
		ts = append(ts, t)
	}
	// Execute the pairing checks and return the results
	if bn256.PairingCheck(cs, ts) {
		return true32Byte, nil
	}
	return false32Byte, nil
}
