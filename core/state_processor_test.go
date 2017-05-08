package core

import (
	"testing"
	"math/big"
	"github.com/ethereumproject/go-ethereum/common"
)

func TestgetBlockEra(t *testing.T) {
	cases := map[*big.Int]*big.Int{
		common.Big1: common.Big1,
		big.NewInt(4999999): common.Big1,
		big.NewInt(5000000): common.Big2,
		big.NewInt(5000001): common.Big2,
		big.NewInt(9999999): common.Big2,
		big.NewInt(10000000): common.Big3,
		big.NewInt(10000001): common.Big3,
		big.NewInt(14999999): common.Big3,
		big.NewInt(15000000): big.NewInt(4),
		big.NewInt(15000001): big.NewInt(4),
		big.NewInt(100000001): big.NewInt(26),
	}
	eraLength := big.NewInt(5000000)

	for bn, expectedEra := range cases {
		gotEra := getBlockEra(bn, eraLength)
		if gotEra.Cmp(expectedEra) == 0 {
			t.Errorf("got: %v, want: %v", gotEra, expectedEra)
		}
	}
}