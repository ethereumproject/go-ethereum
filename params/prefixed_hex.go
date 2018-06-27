package params

import (
	hexlib "encoding/hex"
	"errors"
	"fmt"
	"math/big"
)

/*
PTAL-
This is here only for lazily maintaining compatibility with the config.json implementation
GenesisDump json encoding mechanisms.

This file and everything in it should die as soon as possible.
*/

// Hex is a hexadecimal string.
type Hex string

// Decode fills buf when h is not empty.
func (h Hex) Decode(buf []byte) error {
	if len(h) != 2*len(buf) {
		return fmt.Errorf("want %d hexadecimals", 2*len(buf))
	}

	_, err := hexlib.Decode(buf, []byte(h))
	return err
}

// PrefixedHex is a hexadecimal string with an "0x" prefix.
type PrefixedHex string

var ErrNoHexPrefix = errors.New("want 0x prefix")

// Decode fills buf when h is not empty.
func (h PrefixedHex) Decode(buf []byte) error {
	i := len(h)
	if i == 0 {
		return nil
	}
	if i == 1 || h[0] != '0' || h[1] != 'x' {
		return ErrNoHexPrefix
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

func (h PrefixedHex) Bytes() ([]byte, error) {
	l := len(h)
	if l == 0 {
		return nil, nil
	}
	if l == 1 || h[0] != '0' || h[1] != 'x' {
		return nil, ErrNoHexPrefix
	}
	if l == 2 {
		return nil, nil
	}

	bytes := make([]byte, l/2-1)
	_, err := hexlib.Decode(bytes, []byte(h[2:]))
	return bytes, err
}

func (h PrefixedHex) Int() (*big.Int, error) {
	bytes, err := h.Bytes()
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(bytes), nil
}
