package core

import (
	"math/big"
	"testing"
)

func TestGetBlockEra(t *testing.T) {
	cases := map[*big.Int]*big.Int{
		big.NewInt(0):         big.NewInt(0),
		big.NewInt(1):         big.NewInt(0),
		big.NewInt(4999999):   big.NewInt(0),
		big.NewInt(5000000):   big.NewInt(0),
		big.NewInt(5000001):   big.NewInt(1),
		big.NewInt(9999999):   big.NewInt(1),
		big.NewInt(10000000):  big.NewInt(1),
		big.NewInt(10000001):  big.NewInt(2),
		big.NewInt(14999999):  big.NewInt(2),
		big.NewInt(15000000):  big.NewInt(2),
		big.NewInt(15000001):  big.NewInt(3),
		big.NewInt(100000001): big.NewInt(20),
		big.NewInt(123456789): big.NewInt(24),
	}

	for bn, expectedEra := range cases {
		gotEra := GetBlockEra(bn, EraLength)
		if gotEra.Cmp(expectedEra) != 0 {
			t.Errorf("got: %v, want: %v", gotEra, expectedEra)
		}
	}
}

func TestGetRewardByEra(t *testing.T) {

	cases := map[*big.Int]*big.Int{
		big.NewInt(0):        MaximumBlockReward,
		big.NewInt(1):        MaximumBlockReward,
		big.NewInt(4999999):  MaximumBlockReward,
		big.NewInt(5000000):  MaximumBlockReward,
		big.NewInt(5000001):  big.NewInt(4e+18),
		big.NewInt(9999999):  big.NewInt(4e+18),
		big.NewInt(10000000): big.NewInt(4e+18),
		big.NewInt(10000001): big.NewInt(3.2e+18),
		big.NewInt(14999999): big.NewInt(3.2e+18),
		big.NewInt(15000000): big.NewInt(3.2e+18),
		big.NewInt(15000001): big.NewInt(2.56e+18),
	}

	for bn, expectedReward := range cases {
		gotReward := GetRewardByEra(GetBlockEra(bn, EraLength))
		if gotReward.Cmp(expectedReward) != 0 {
			t.Errorf("@ %v, got: %v, want: %v", bn, gotReward, expectedReward)
		}
		if gotReward.Cmp(big.NewInt(0)) <= 0 {
			t.Errorf("@ %v, got: %v, want: %v", bn, gotReward, expectedReward)
		}
		if gotReward.Cmp(MaximumBlockReward) > 0 {
			t.Errorf("@ %v, got: %v, want %v", bn, gotReward, expectedReward)
		}
	}

}
