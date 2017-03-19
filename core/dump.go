// Copyright 2017 The go-ethereum Authors
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

package core

import (
	hexlib "encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereumproject/go-ethereum/core/types"
)

// GenesisDump is the geth JSON format.
// https://github.com/ethereumproject/wiki/wiki/Ethereum-Chain-Spec-Format#subformat-genesis
type GenesisDump struct {
	Nonce      prefixedHex
	Timestamp  prefixedHex
	ParentHash prefixedHex
	ExtraData  prefixedHex
	GasLimit   prefixedHex
	Difficulty prefixedHex
	Mixhash    prefixedHex
	Coinbase   prefixedHex

	// Alloc maps accounts by their address.
	Alloc map[hex]*GenesisDumpAlloc
}

// GenesisDumpAlloc is a GenesisDump.Alloc entry.
type GenesisDumpAlloc struct {
	Code    prefixedHex
	Storage map[hex]hex
	Balance string // decimal string
}

// Header returns the mapping.
func (g *GenesisDump) Header() (*types.Header, error) {
	var h types.Header

	var err error
	if err = g.Nonce.Decode(h.Nonce[:]); err != nil {
		return nil, fmt.Errorf("malformed nonce: %s", err)
	}
	if h.Time, err = g.Timestamp.Int(); err != nil {
		return nil, fmt.Errorf("malformed timestamp: %s", err)
	}
	if err = g.ParentHash.Decode(h.ParentHash[:]); err != nil {
		return nil, fmt.Errorf("malformed parentHash: %s", err)
	}
	if h.Extra, err = g.ExtraData.Bytes(); err != nil {
		return nil, fmt.Errorf("malformed extraData: %s", err)
	}
	if h.GasLimit, err = g.GasLimit.Int(); err != nil {
		return nil, fmt.Errorf("malformed gasLimit: %s", err)
	}
	if h.Difficulty, err = g.Difficulty.Int(); err != nil {
		return nil, fmt.Errorf("malformed difficulty: %s", err)
	}
	if err = g.Mixhash.Decode(h.MixDigest[:]); err != nil {
		return nil, fmt.Errorf("malformed mixhash: %s", err)
	}
	if err := g.Coinbase.Decode(h.Coinbase[:]); err != nil {
		return nil, fmt.Errorf("malformed coinbase: %s", err)
	}

	return &h, nil
}

// hex is a hexadecimal string.
type hex string

// Decode fills buf when h is not empty.
func (h hex) Decode(buf []byte) error {
	if len(h) != 2*len(buf) {
		return fmt.Errorf("want %d hexadecimals", 2*len(buf))
	}

	_, err := hexlib.Decode(buf, []byte(h))
	return err
}

// prefixedHex is a hexadecimal string with an "0x" prefix.
type prefixedHex string

var errNoHexPrefix = errors.New("want 0x prefix")

// Decode fills buf when h is not empty.
func (h prefixedHex) Decode(buf []byte) error {
	i := len(h)
	if i == 0 {
		return nil
	}
	if i == 1 || h[0] != '0' || h[1] != 'x' {
		return errNoHexPrefix
	}
	if i == 2 {
		return nil
	}
	if i != 2*len(buf)+2 {
		return fmt.Errorf("want %d hexadecimals with 0x prefix", 2*len(buf))
	}

	_, err := hexlib.Decode(buf, []byte(h[2:]))
	return err
}

func (h prefixedHex) Bytes() ([]byte, error) {
	l := len(h)
	if l == 0 {
		return nil, nil
	}
	if l == 1 || h[0] != '0' || h[1] != 'x' {
		return nil, errNoHexPrefix
	}
	if l == 2 {
		return nil, nil
	}

	bytes := make([]byte, l/2-1)
	_, err := hexlib.Decode(bytes, []byte(h[2:]))
	return bytes, err
}

func (h prefixedHex) Int() (*big.Int, error) {
	bytes, err := h.Bytes()
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(bytes), nil
}
