package core

import (
	"math/big"
	"testing"

	"math/rand"
	"time"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/ethdb"
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
		gotEra := GetBlockEra(bn, DefaultEraLength)
		if gotEra.Cmp(expectedEra) != 0 {
			t.Errorf("got: %v, want: %v", gotEra, expectedEra)
		}
	}
}

func TestGetBlockWinnerRewardByEra(t *testing.T) {

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
		gotReward := GetBlockWinnerRewardByEra(GetBlockEra(bn, DefaultEraLength))
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

func TestGetBlockUncleRewardByEra(t *testing.T) {

	var we1, we2, we3, we4 *big.Int = new(big.Int), new(big.Int), new(big.Int), new(big.Int)

	we2.Div(GetBlockWinnerRewardByEra(big.NewInt(1)), big.NewInt(32))
	we3.Div(GetBlockWinnerRewardByEra(big.NewInt(2)), big.NewInt(32))
	we4.Div(GetBlockWinnerRewardByEra(big.NewInt(3)), big.NewInt(32))

	cases := map[*big.Int]*big.Int{
		big.NewInt(0):        nil,
		big.NewInt(1):        nil,
		big.NewInt(4999999):  nil,
		big.NewInt(5000000):  nil,
		big.NewInt(5000001):  we2,
		big.NewInt(9999999):  we2,
		big.NewInt(10000000): we2,
		big.NewInt(10000001): we3,
		big.NewInt(14999999): we3,
		big.NewInt(15000000): we3,
		big.NewInt(15000001): we4,
	}

	for bn, want := range cases {

		era := GetBlockEra(bn, DefaultEraLength)

		var header, uncle *types.Header = &types.Header{}, &types.Header{}
		header.Number = bn

		rand.Seed(time.Now().UTC().UnixNano())
		uncle.Number = big.NewInt(0).Sub(header.Number, big.NewInt(int64(rand.Int31n(int32(7)))))

		got := GetBlockUncleRewardByEra(era, header, uncle)

		// "Era 1"
		if want == nil {
			we1.Add(uncle.Number, big8)      // 2,534,998 + 8              = 2,535,006
			we1.Sub(we1, header.Number)      // 2,535,006 - 2,534,999        = 7
			we1.Mul(we1, MaximumBlockReward) // 7 * 5e+18               = 35e+18
			we1.Div(we1, big8)               // 35e+18 / 8                            = 7/8 * 5e+18

			if got.Cmp(we1) != 0 {
				t.Errorf("@ %v, want: %v, got: %v", bn, we1, got)
			}
		} else {
			if got.Cmp(want) != 0 {
				t.Errorf("@ %v, want: %v, got: %v", bn, want, got)
			}
		}
	}
}

func TestGetBlockWinnerRewardForUnclesByEra(t *testing.T) {

	// "want era 1", "want era 2", ...
	var we1, we2, we3, we4 *big.Int = new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	we1.Div(MaximumBlockReward, big.NewInt(32))
	we2.Div(GetBlockWinnerRewardByEra(big.NewInt(1)), big.NewInt(32))
	we3.Div(GetBlockWinnerRewardByEra(big.NewInt(2)), big.NewInt(32))
	we4.Div(GetBlockWinnerRewardByEra(big.NewInt(3)), big.NewInt(32))

	cases := map[*big.Int]*big.Int{
		big.NewInt(0):        we1,
		big.NewInt(1):        we1,
		big.NewInt(4999999):  we1,
		big.NewInt(5000000):  we1,
		big.NewInt(5000001):  we2,
		big.NewInt(9999999):  we2,
		big.NewInt(10000000): we2,
		big.NewInt(10000001): we3,
		big.NewInt(14999999): we3,
		big.NewInt(15000000): we3,
		big.NewInt(15000001): we4,
	}

	var uncleSingle, uncleDouble []*types.Header = []*types.Header{{}}, []*types.Header{{}, {}}

	for bn, want := range cases {
		// test single uncle
		got := GetBlockWinnerRewardForUnclesByEra(GetBlockEra(bn, DefaultEraLength), uncleSingle)
		if got.Cmp(want) != 0 {
			t.Errorf("@ %v: want: %v, got: %v", bn, want, got)
		}

		// test double uncle
		got = GetBlockWinnerRewardForUnclesByEra(GetBlockEra(bn, DefaultEraLength), uncleDouble)
		dub := new(big.Int)
		if got.Cmp(dub.Mul(want, big.NewInt(2))) != 0 {
			t.Errorf("@ %v: want: %v, got: %v", bn, want, got)
		}
	}

}

func TestAccumulateRewards(t *testing.T) {
	configs := []*ChainConfig{DefaultConfig, TestConfig}
	for i, config := range configs {
		db, _ := ethdb.NewMemDatabase()
		defer db.Close()
		stateDB, err := state.New(common.Hash{}, db)
		if err != nil {
			t.Fatalf("could not open statedb: %v", err)
		}

		var header *types.Header = &types.Header{}
		var uncles []*types.Header = []*types.Header{{}, {}}

		header.Coinbase = common.StringToAddress("0000000000000000000000000000000000000001")
		uncles[0].Coinbase = common.StringToAddress("0000000000000000000000000000000000000002")
		uncles[1].Coinbase = common.StringToAddress("0000000000000000000000000000000000000003")

		// Manual tallies for reward accumulation.
		winnerB, totalB := new(big.Int), new(big.Int)
		unclesB := []*big.Int{new(big.Int), new(big.Int)}

		winnerB = stateDB.GetBalance(header.Coinbase)
		unclesB[0] = stateDB.GetBalance(uncles[0].Coinbase)
		unclesB[1] = stateDB.GetBalance(uncles[1].Coinbase)

		cases := []*big.Int{
			//big.NewInt(0),
			big.NewInt(1),
			big.NewInt(4999999),
			big.NewInt(5000000),
			big.NewInt(5000001),
			big.NewInt(9999999),
			big.NewInt(10000000),
			big.NewInt(10000001),
			big.NewInt(14999999),
			big.NewInt(15000000),
			big.NewInt(15000001),
		}

		for _, bn := range cases {
			era := GetBlockEra(bn, DefaultEraLength)

			header.Number = bn

			for i, uncle := range uncles {
				//rand.Seed(time.Now().UTC().UnixNano())
				// TODO: is it ok to reuse same uncle numbers? it could happen...
				//uncle.Number.Sub(header.Number, big.NewInt(int64(rand.Int31n(int32(7)))))

				uncle.Number = big.NewInt(0).Sub(header.Number, big.NewInt(1))

				ur := GetBlockUncleRewardByEra(era, header, uncle)
				unclesB[i].Add(unclesB[i], ur)

				totalB.Add(totalB, ur)
			}

			wr := GetBlockWinnerRewardByEra(era)
			wr.Add(wr, GetBlockWinnerRewardForUnclesByEra(era, uncles))
			winnerB.Add(winnerB, wr)

			totalB.Add(totalB, winnerB)

			AccumulateRewards(config, stateDB, header, uncles)

			// Check balances.
			if wb := stateDB.GetBalance(header.Coinbase); wb.Cmp(winnerB) != 0 {
				t.Errorf("winner balance @ %v, want: %v, got: %v (config: %v)", bn, winnerB, wb, i)
			}
			if uB0 := stateDB.GetBalance(uncles[0].Coinbase); unclesB[0].Cmp(uB0) != 0 {
				t.Errorf("uncle1 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[0], uB0, i)
			}
			if uB1 := stateDB.GetBalance(uncles[1].Coinbase); unclesB[1].Cmp(uB1) != 0 {
				t.Errorf("uncle2 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[1], uB1, i)
			}
			// overflows int64
			//if bn.Cmp(big.NewInt(1)) == 0 && totalB.Cmp(big.NewInt(14.0625e+18)) != 0 {
			//	t.Errorf("total balance @ 1, want: %v, got: %v", bn, big.NewInt(14.0625e+18), totalB)
			//}
		}
	}
}
