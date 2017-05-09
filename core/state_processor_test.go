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
		big.NewInt(1914999):   big.NewInt(0),
		big.NewInt(1915000):   big.NewInt(0),
		big.NewInt(1915001):   big.NewInt(0),
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

	// manually divide maxblockreward/32 to compare to got
	we2.Div(GetBlockWinnerRewardByEra(GetBlockEra(big.NewInt(5000001), DefaultEraLength)), big.NewInt(32))
	we3.Div(GetBlockWinnerRewardByEra(GetBlockEra(big.NewInt(10000001), DefaultEraLength)), big.NewInt(32))
	we4.Div(GetBlockWinnerRewardByEra(GetBlockEra(big.NewInt(15000001), DefaultEraLength)), big.NewInt(32))

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

//
//func fakeAccumulateRewards(config *ChainConfig, states map[common.Address]*big.Int, header *types.Header, uncles []*types.Header) {
//	feat, _, configured := config.GetFeature(header.Number, "reward")
//	if !configured {
//		reward := new(big.Int).Set(MaximumBlockReward)
//		r := new(big.Int)
//
//		for _, uncle := range uncles {
//			r.Add(uncle.Number, big8)    // 2,534,998 + 8              = 2,535,006
//			r.Sub(r, header.Number)      // 2,535,006 - 2,534,999        = 7
//			r.Mul(r, MaximumBlockReward) // 7 * 5e+18               = 35e+18
//			r.Div(r, big8)               // 35e+18 / 8                            = 7/8 * 5e+18
//
//			prevBal := states[uncle.Coinbase]
//			states[uncle.Coinbase] = new(big.Int).Add(prevBal, r)
//			//statedb.AddBalance(uncle.Coinbase, r) // $$
//
//			r.Div(MaximumBlockReward, big32) // 5e+18 / 32
//			reward.Add(reward, r)            // 5e+18 + (1/32*5e+18)
//		}
//		prevBal := states[header.Coinbase]
//		states[header.Coinbase] = new(big.Int).Add(prevBal, reward)
//		//states.AddBalance(header.Coinbase, reward) //  $$ => 5e+18 + (1/32*5e+18)
//	} else {
//		// Check that configuration specifies ECIP1017.
//		val, ok := feat.GetString("type")
//		if !ok || val != "ecip1017" {
//			panic(ErrConfiguration)
//		}
//
//		// Ensure value 'era' is configured.
//		eraLen, ok := feat.GetBigInt("era")
//		if !ok || eraLen.Cmp(big.NewInt(0)) <= 0 {
//			panic(ErrConfiguration)
//		}
//
//		era := GetBlockEra(header.Number, eraLen)
//
//		wr := GetBlockWinnerRewardByEra(era)                    // wr "winner reward". 5, 4, 3.2, 2.56, ...
//		wurs := GetBlockWinnerRewardForUnclesByEra(era, uncles) // wurs "winner uncle rewards"
//		wr.Add(wr, wurs)
//
//		prevBal := states[header.Coinbase]
//		states[header.Coinbase] = new(big.Int).Add(prevBal, wr)
//		//states.AddBalance(header.Coinbase, wr) // $$
//
//		// Reward uncle miners.
//		for _, uncle := range uncles {
//			ur := GetBlockUncleRewardByEra(era, header, uncle)
//			prevBal := states[uncle.Coinbase]
//			states[uncle.Coinbase] = new(big.Int).Add(prevBal, ur)
//			//states.AddBalance(uncle.Coinbase, ur) // $$
//		}
//	}
//}
//
//// Accruing over block cases simulates compounding longevity of an account.
//// Uses two statedbs to maintain separate ledgers, with the 'testwith'db calling
//// AddBalance at tally points (instead of testing strictly through delegation to AccumulateRewards).
//// Ensures problem is not behind-the-scenes of statedb.
//func TestAccumulateRewards0(t *testing.T) {
//	configs := []*ChainConfig{DefaultConfig, TestConfig}
//	for i, config := range configs {
//		dbTeston, _ := ethdb.NewMemDatabase()
//		dbTestwith, _ := ethdb.NewMemDatabase()
//
//		//dumps := []*GenesisDump{DefaultGenesis, TestNetGenesis}
//		//genHead, e := dumps[i].Header()
//		//if e != nil {
//		//	t.Fatalf("unexpected: %v", e)
//		//}
//
//		//stateDB, err := state.New(genHead.Hash(), db)
//		stateDBTeston, err := state.New(common.Hash{}, dbTeston)
//		if err != nil {
//			t.Fatalf("could not open statedb: %v", err)
//		}
//
//		stateDBTestwith, err := state.New(common.Hash{}, dbTestwith)
//		if err != nil {
//			t.Fatalf("could not open statedb: %v", err)
//		}
//
//		var header *types.Header = &types.Header{}
//		var uncles []*types.Header = []*types.Header{{}, {}}
//
//		if i == 0 {
//			header.Coinbase = common.StringToAddress("000d836201318ec6899a67540690382780743280")
//			uncles[0].Coinbase = common.StringToAddress("001762430ea9c3a26e5749afdb70da5f78ddbb8c")
//			uncles[1].Coinbase = common.StringToAddress("001d14804b399c6ef80e64576f657660804fec0b")
//		} else {
//			header.Coinbase = common.StringToAddress("0000000000000000000000000000000000000001")
//			uncles[0].Coinbase = common.StringToAddress("0000000000000000000000000000000000000002")
//			uncles[1].Coinbase = common.StringToAddress("0000000000000000000000000000000000000003")
//		}
//
//		// Manual tallies for reward accumulation.
//		winnerB, totalB := new(big.Int), new(big.Int)
//		unclesB := []*big.Int{new(big.Int), new(big.Int)}
//
//		winnerB = stateDBTeston.GetBalance(header.Coinbase)
//		unclesB[0] = stateDBTeston.GetBalance(uncles[0].Coinbase)
//		unclesB[1] = stateDBTeston.GetBalance(uncles[1].Coinbase)
//
//		totalB.Add(totalB, winnerB)
//		totalB.Add(totalB, unclesB[0])
//		totalB.Add(totalB, unclesB[1])
//
//		if totalB.Cmp(big.NewInt(0)) != 0 {
//			t.Errorf("unexpected: %v", totalB)
//		}
//
//		// Manual tallies for reward accumulation.
//		winnerB, totalB = new(big.Int), new(big.Int)
//		unclesB = []*big.Int{new(big.Int), new(big.Int)}
//
//		winnerB = stateDBTestwith.GetBalance(header.Coinbase)
//		unclesB[0] = stateDBTestwith.GetBalance(uncles[0].Coinbase)
//		unclesB[1] = stateDBTestwith.GetBalance(uncles[1].Coinbase)
//
//		totalB.Add(totalB, winnerB)
//		totalB.Add(totalB, unclesB[0])
//		totalB.Add(totalB, unclesB[1])
//
//		if totalB.Cmp(big.NewInt(0)) != 0 {
//			t.Errorf("unexpected: %v", totalB)
//		}
//
//		cases := []*big.Int{
//			//big.NewInt(0),
//			big.NewInt(13),
//			big.NewInt(4999999),
//			big.NewInt(5000000),
//			big.NewInt(5000001),
//			big.NewInt(9999999),
//			big.NewInt(10000000),
//			big.NewInt(10000000),
//			big.NewInt(10000001),
//			big.NewInt(14999999),
//			big.NewInt(15000000),
//			big.NewInt(15000001),
//		}
//
//		for _, bn := range cases {
//			era := GetBlockEra(bn, DefaultEraLength)
//			t.Logf("era: %v", era)
//
//			header.Number = bn
//
//			for i, uncle := range uncles {
//				// rand.Seed(time.Now().UTC().UnixNano())
//				// uncle.Number = new(big.Int).Sub(header.Number, big.NewInt(int64(rand.Int31n(int32(7)))))
//				uncle.Number = new(big.Int).Sub(header.Number, big.NewInt(int64(1))) // +i
//
//				ur := GetBlockUncleRewardByEra(era, header, uncle)
//				t.Logf("ur: %v", ur)
//				unclesB[i].Add(unclesB[i], ur)
//				stateDBTestwith.AddBalance(uncles[i].Coinbase, ur)
//
//				totalB.Add(totalB, ur)
//			}
//
//			wr := GetBlockWinnerRewardByEra(era)
//			wr.Add(wr, GetBlockWinnerRewardForUnclesByEra(era, uncles))
//			t.Logf("wr: %v", wr)
//			winnerB.Add(winnerB, wr)
//			stateDBTestwith.AddBalance(header.Coinbase, wr)
//
//			totalB.Add(totalB, winnerB)
//
//			AccumulateRewards(config, stateDBTeston, header, uncles)
//
//			// Check balances.
//			if wb := stateDBTeston.GetBalance(header.Coinbase); wb.Cmp(stateDBTestwith.GetBalance(header.Coinbase)) != 0 {
//				t.Errorf("winner balance @ %v, want: %v, got: %v (config: %v)", bn, stateDBTestwith.GetBalance(header.Coinbase), wb, i)
//			}
//			if uB0 := stateDBTeston.GetBalance(uncles[0].Coinbase); unclesB[0].Cmp(stateDBTestwith.GetBalance(uncles[0].Coinbase)) != 0 {
//				t.Errorf("uncle1 balance @ %v, want: %v, got: %v (config: %v)", bn, stateDBTestwith.GetBalance(uncles[0].Coinbase), uB0, i)
//			}
//			if uB1 := stateDBTeston.GetBalance(uncles[1].Coinbase); unclesB[1].Cmp(stateDBTestwith.GetBalance(uncles[1].Coinbase)) != 0 {
//				t.Errorf("uncle2 balance @ %v, want: %v, got: %v (config: %v)", bn, stateDBTestwith.GetBalance(uncles[1].Coinbase), uB1, i)
//			}
//			// overflows int64
//			//if bn.Cmp(big.NewInt(1)) == 0 && totalB.Cmp(big.NewInt(14.0625e+18)) != 0 {
//			//	t.Errorf("total balance @ 1, want: %v, got: %v", bn, big.NewInt(14.0625e+18), totalB)
//			//}
//		}
//		dbTeston.Close()
//		dbTestwith.Close()
//	}
//}
//
//// Accruing over block cases simulates compounding longevity of an account.
//// Tests using maps of running sums for winner & 2 uncles, to keep tally.
//func TestAccumulateRewards1(t *testing.T) {
//	configs := []*ChainConfig{DefaultConfig, TestConfig}
//	for i, config := range configs {
//		db, _ := ethdb.NewMemDatabase()
//
//		//dumps := []*GenesisDump{DefaultGenesis, TestNetGenesis}
//		//genHead, e := dumps[i].Header()
//		//if e != nil {
//		//	t.Fatalf("unexpected: %v", e)
//		//}
//
//		//stateDB, err := state.New(genHead.Hash(), db)
//		stateDB, err := state.New(common.Hash{}, db)
//		if err != nil {
//			t.Fatalf("could not open statedb: %v", err)
//		}
//
//		var header *types.Header = &types.Header{}
//		var uncles []*types.Header = []*types.Header{{}, {}}
//
//		if i == 0 {
//			header.Coinbase = common.StringToAddress("000d836201318ec6899a67540690382780743280")
//			uncles[0].Coinbase = common.StringToAddress("001762430ea9c3a26e5749afdb70da5f78ddbb8c")
//			uncles[1].Coinbase = common.StringToAddress("001d14804b399c6ef80e64576f657660804fec0b")
//		} else {
//			header.Coinbase = common.StringToAddress("0000000000000000000000000000000000000001")
//			uncles[0].Coinbase = common.StringToAddress("0000000000000000000000000000000000000002")
//			uncles[1].Coinbase = common.StringToAddress("0000000000000000000000000000000000000003")
//		}
//
//		// Manual tallies for reward accumulation.
//		winnerB, totalB := new(big.Int), new(big.Int)
//		unclesB := []*big.Int{new(big.Int), new(big.Int)}
//
//		winnerB = stateDB.GetBalance(header.Coinbase)
//		unclesB[0] = stateDB.GetBalance(uncles[0].Coinbase)
//		unclesB[1] = stateDB.GetBalance(uncles[1].Coinbase)
//
//		totalB.Add(totalB, winnerB)
//		totalB.Add(totalB, unclesB[0])
//		totalB.Add(totalB, unclesB[1])
//
//		if totalB.Cmp(big.NewInt(0)) != 0 {
//			t.Errorf("unexpected: %v", totalB)
//		}
//
//		cases := []*big.Int{
//			//big.NewInt(0),
//			big.NewInt(13),
//			big.NewInt(4999999),
//			big.NewInt(5000000),
//			big.NewInt(5000001),
//			big.NewInt(9999999),
//			big.NewInt(10000000),
//			big.NewInt(10000000),
//			big.NewInt(10000001),
//			big.NewInt(14999999),
//			big.NewInt(15000000),
//			big.NewInt(15000001),
//		}
//
//		for _, bn := range cases {
//			era := GetBlockEra(bn, DefaultEraLength)
//			t.Logf("era: %v", era)
//
//			header.Number = bn
//
//			for i, uncle := range uncles {
//				// rand.Seed(time.Now().UTC().UnixNano())
//				// uncle.Number = new(big.Int).Sub(header.Number, big.NewInt(int64(rand.Int31n(int32(7)))))
//				uncle.Number = new(big.Int).Sub(header.Number, big.NewInt(int64(1))) // +i
//
//				ur := GetBlockUncleRewardByEra(era, header, uncle)
//				t.Logf("ur: %v", ur)
//				unclesB[i].Add(unclesB[i], ur)
//
//				totalB.Add(totalB, ur)
//			}
//
//			wr := GetBlockWinnerRewardByEra(era)
//			wr.Add(wr, GetBlockWinnerRewardForUnclesByEra(era, uncles))
//			t.Logf("wr: %v", wr)
//			winnerB.Add(winnerB, wr)
//
//			totalB.Add(totalB, winnerB)
//
//			AccumulateRewards(config, stateDB, header, uncles)
//
//			// Check balances.
//			if wb := stateDB.GetBalance(header.Coinbase); wb.Cmp(winnerB) != 0 {
//				t.Errorf("winner balance @ %v, want: %v, got: %v (config: %v)", bn, winnerB, wb, i)
//			}
//			if uB0 := stateDB.GetBalance(uncles[0].Coinbase); unclesB[0].Cmp(uB0) != 0 {
//				t.Errorf("uncle1 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[0], uB0, i)
//			}
//			if uB1 := stateDB.GetBalance(uncles[1].Coinbase); unclesB[1].Cmp(uB1) != 0 {
//				t.Errorf("uncle2 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[1], uB1, i)
//			}
//			// overflows int64
//			//if bn.Cmp(big.NewInt(1)) == 0 && totalB.Cmp(big.NewInt(14.0625e+18)) != 0 {
//			//	t.Errorf("total balance @ 1, want: %v, got: %v", bn, big.NewInt(14.0625e+18), totalB)
//			//}
//		}
//		db.Close()
//	}
//}
//
//// Tests a fake test, as found above.
//// The fake test does actually call statedb.AddBalance, but tallies sum
//// in a given map 'states'.
//// Accruing over block cases simulates compounding longevity of an account.
//func TestAccumulateRewards1_Fake(t *testing.T) {
//	configs := []*ChainConfig{DefaultConfig, TestConfig}
//	for i, config := range configs {
//		db, _ := ethdb.NewMemDatabase()
//
//		//dumps := []*GenesisDump{DefaultGenesis, TestNetGenesis}
//		//genHead, e := dumps[i].Header()
//		//if e != nil {
//		//	t.Fatalf("unexpected: %v", e)
//		//}
//
//		//stateDB, err := state.New(genHead.Hash(), db)
//		stateDB, err := state.New(common.Hash{}, db)
//		if err != nil {
//			t.Fatalf("could not open statedb: %v", err)
//		}
//
//		var header *types.Header = &types.Header{}
//		var uncles []*types.Header = []*types.Header{{}, {}}
//		states := make(map[common.Address]*big.Int)
//
//		if i == 0 {
//			header.Coinbase = common.StringToAddress("000d836201318ec6899a67540690382780743280")
//			uncles[0].Coinbase = common.StringToAddress("001762430ea9c3a26e5749afdb70da5f78ddbb8c")
//			uncles[1].Coinbase = common.StringToAddress("001d14804b399c6ef80e64576f657660804fec0b")
//		} else {
//			header.Coinbase = common.StringToAddress("0000000000000000000000000000000000000001")
//			uncles[0].Coinbase = common.StringToAddress("0000000000000000000000000000000000000002")
//			uncles[1].Coinbase = common.StringToAddress("0000000000000000000000000000000000000003")
//		}
//		states[header.Coinbase] = big.NewInt(0)
//		states[uncles[0].Coinbase] = big.NewInt(0)
//		states[uncles[1].Coinbase] = big.NewInt(0)
//
//		// Manual tallies for reward accumulation.
//		winnerB, totalB := new(big.Int), new(big.Int)
//		unclesB := []*big.Int{new(big.Int), new(big.Int)}
//
//		winnerB = stateDB.GetBalance(header.Coinbase)
//		unclesB[0] = stateDB.GetBalance(uncles[0].Coinbase)
//		unclesB[1] = stateDB.GetBalance(uncles[1].Coinbase)
//
//		totalB.Add(totalB, winnerB)
//		totalB.Add(totalB, unclesB[0])
//		totalB.Add(totalB, unclesB[1])
//
//		if totalB.Cmp(big.NewInt(0)) != 0 {
//			t.Errorf("unexpected: %v", totalB)
//		}
//
//		cases := []*big.Int{
//			//big.NewInt(0),
//			big.NewInt(13),
//			big.NewInt(4999999),
//			big.NewInt(5000000),
//			big.NewInt(5000001),
//			big.NewInt(9999999),
//			big.NewInt(10000000),
//			big.NewInt(10000000),
//			big.NewInt(10000001),
//			big.NewInt(14999999),
//			big.NewInt(15000000),
//			big.NewInt(15000001),
//		}
//
//		for _, bn := range cases {
//			era := GetBlockEra(bn, DefaultEraLength)
//			t.Logf("era: %v", era)
//
//			header.Number = bn
//
//			for i, uncle := range uncles {
//				// rand.Seed(time.Now().UTC().UnixNano())
//				// uncle.Number = new(big.Int).Sub(header.Number, big.NewInt(int64(rand.Int31n(int32(7)))))
//				uncle.Number = new(big.Int).Sub(header.Number, big.NewInt(int64(1))) // +i
//
//				ur := GetBlockUncleRewardByEra(era, header, uncle)
//				t.Logf("ur: %v", ur)
//				unclesB[i].Add(unclesB[i], ur)
//
//				totalB.Add(totalB, ur)
//			}
//
//			wr := GetBlockWinnerRewardByEra(era)
//			wr.Add(wr, GetBlockWinnerRewardForUnclesByEra(era, uncles))
//			t.Logf("wr: %v", wr)
//			winnerB.Add(winnerB, wr)
//
//			totalB.Add(totalB, winnerB)
//
//			fakeAccumulateRewards(config, states, header, uncles)
//			//AccumulateRewards(config, stateDB, header, uncles)
//
//			// Check balances.
//			if wb := states[header.Coinbase]; wb.Cmp(winnerB) != 0 {
//				t.Errorf("winner balance @ %v, want: %v, got: %v (config: %v)", bn, winnerB, wb, i)
//			}
//			if uB0 := states[uncles[0].Coinbase]; unclesB[0].Cmp(uB0) != 0 {
//				t.Errorf("uncle1 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[0], uB0, i)
//			}
//			if uB1 := states[uncles[1].Coinbase]; unclesB[1].Cmp(uB1) != 0 {
//				t.Errorf("uncle2 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[1], uB1, i)
//			}
//			//if wb := stateDB.GetBalance(header.Coinbase); wb.Cmp(winnerB) != 0 {
//			//	t.Errorf("winner balance @ %v, want: %v, got: %v (config: %v)", bn, winnerB, wb, i)
//			//}
//			//if uB0 := stateDB.GetBalance(uncles[0].Coinbase); unclesB[0].Cmp(uB0) != 0 {
//			//	t.Errorf("uncle1 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[0], uB0, i)
//			//}
//			//if uB1 := stateDB.GetBalance(uncles[1].Coinbase); unclesB[1].Cmp(uB1) != 0 {
//			//	t.Errorf("uncle2 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[1], uB1, i)
//			//}
//			// overflows int64
//			//if bn.Cmp(big.NewInt(1)) == 0 && totalB.Cmp(big.NewInt(14.0625e+18)) != 0 {
//			//	t.Errorf("total balance @ 1, want: %v, got: %v", bn, big.NewInt(14.0625e+18), totalB)
//			//}
//		}
//		db.Close()
//	}
//}
//
//// Non-accruing over block cases simulates instance.
//func TestAccumulateRewards2(t *testing.T) {
//	configs := []*ChainConfig{DefaultConfig, TestConfig}
//	for i, config := range configs {
//
//		cases := []*big.Int{
//			//big.NewInt(0),
//			big.NewInt(13),
//			big.NewInt(4999999),
//			big.NewInt(5000000),
//			big.NewInt(5000001),
//			big.NewInt(9999999),
//			big.NewInt(10000000),
//			big.NewInt(10000001),
//			big.NewInt(14999999),
//			big.NewInt(15000000),
//			big.NewInt(15000001),
//		}
//
//		for _, bn := range cases {
//
//			db, _ := ethdb.NewMemDatabase()
//
//			//dumps := []*GenesisDump{DefaultGenesis, TestNetGenesis}
//			//genHead, e := dumps[i].Header()
//			//if e != nil {
//			//	t.Fatalf("unexpected: %v", e)
//			//}
//
//			//stateDB, err := state.New(genHead.Hash(), db)
//			stateDB, err := state.New(common.Hash{}, db)
//			if err != nil {
//				t.Fatalf("could not open statedb: %v", err)
//			}
//			if err != nil {
//				t.Fatalf("could not open statedb: %v", err)
//			}
//
//			var header *types.Header = &types.Header{}
//			var uncles []*types.Header = []*types.Header{{}, {}}
//
//			if i == 0 {
//				header.Coinbase = common.StringToAddress("000d836201318ec6899a67540690382780743280")
//				uncles[0].Coinbase = common.StringToAddress("001762430ea9c3a26e5749afdb70da5f78ddbb8c")
//				uncles[1].Coinbase = common.StringToAddress("001d14804b399c6ef80e64576f657660804fec0b")
//			} else {
//				header.Coinbase = common.StringToAddress("0000000000000000000000000000000000000001")
//				uncles[0].Coinbase = common.StringToAddress("0000000000000000000000000000000000000002")
//				uncles[1].Coinbase = common.StringToAddress("0000000000000000000000000000000000000003")
//			}
//
//			// Manual tallies for reward accumulation.
//			winnerB, totalB := new(big.Int), new(big.Int)
//			unclesB := []*big.Int{new(big.Int), new(big.Int)}
//
//			winnerB = stateDB.GetBalance(header.Coinbase)
//			unclesB[0] = stateDB.GetBalance(uncles[0].Coinbase)
//			unclesB[1] = stateDB.GetBalance(uncles[1].Coinbase)
//
//			totalB.Add(totalB, winnerB)
//			totalB.Add(totalB, unclesB[0])
//			totalB.Add(totalB, unclesB[1])
//
//			if totalB.Cmp(big.NewInt(0)) != 0 {
//				t.Errorf("unexpected: %v", totalB)
//			}
//
//			era := GetBlockEra(bn, DefaultEraLength)
//			t.Logf("era: %v", era)
//
//			header.Number = bn
//
//			for i, uncle := range uncles {
//				// rand.Seed(time.Now().UTC().UnixNano())
//				// uncle.Number = new(big.Int).Sub(header.Number, big.NewInt(int64(rand.Int31n(int32(7)))))
//				uncle.Number = new(big.Int).Sub(header.Number, big.NewInt(int64(1))) // +i
//
//				ur := GetBlockUncleRewardByEra(era, header, uncle)
//				t.Logf("ur: %v", ur)
//				unclesB[i].Add(unclesB[i], ur)
//
//				totalB.Add(totalB, ur)
//			}
//
//			wr := GetBlockWinnerRewardByEra(era)
//			wr.Add(wr, GetBlockWinnerRewardForUnclesByEra(era, uncles))
//			t.Logf("wr: %v", wr)
//			winnerB.Add(winnerB, wr)
//
//			totalB.Add(totalB, winnerB)
//
//			AccumulateRewards(config, stateDB, header, uncles)
//
//			// Check balances.
//			if wb := stateDB.GetBalance(header.Coinbase); wb.Cmp(winnerB) != 0 {
//				t.Errorf("winner balance @ %v, want: %v, got: %v (config: %v)", bn, winnerB, wb, i)
//			}
//			if uB0 := stateDB.GetBalance(uncles[0].Coinbase); unclesB[0].Cmp(uB0) != 0 {
//				t.Errorf("uncle1 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[0], uB0, i)
//			}
//			if uB1 := stateDB.GetBalance(uncles[1].Coinbase); unclesB[1].Cmp(uB1) != 0 {
//				t.Errorf("uncle2 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[1], uB1, i)
//			}
//			// overflows int64
//			//if bn.Cmp(big.NewInt(1)) == 0 && totalB.Cmp(big.NewInt(14.0625e+18)) != 0 {
//			//	t.Errorf("total balance @ 1, want: %v, got: %v", bn, big.NewInt(14.0625e+18), totalB)
//			//}
//			db.Close()
//		}
//	}
//}

var (
	WinnerCoinbase = common.StringToAddress("0000000000000000000000000000000000000001")
	Uncle1Coinbase = common.StringToAddress("0000000000000000000000000000000000000002")
	Uncle2Coinbase = common.StringToAddress("0000000000000000000000000000000000000003")

	Era1WinnerReward      = big.NewInt(5e+18)
	Era1WinnerUncleReward = big.NewInt(156250000000000000)
	Era1UncleReward       = big.NewInt(4375000000000000000)

	Era2WinnerReward      = big.NewInt(4e+18)
	Era2WinnerUncleReward = new(big.Int).Div(big.NewInt(4e+18), big32)
	Era2UncleReward       = new(big.Int).Div(big.NewInt(4e+18), big32)

	Era3WinnerReward      = new(big.Int).Mul(new(big.Int).Div(big.NewInt(4e+18), big.NewInt(5)), big.NewInt(4))
	Era3WinnerUncleReward = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Div(big.NewInt(4e+18), big.NewInt(5)), big.NewInt(4)), big32)
	Era3UncleReward       = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Div(big.NewInt(4e+18), big.NewInt(5)), big.NewInt(4)), big32)
)

// Non-accruing over block cases simulates instance.
func TestAccumulateRewards3_Testnet(t *testing.T) {

	type rewards map[common.Address]*big.Int

	cases := []struct {
		block   *big.Int
		rewards rewards
	}{
		//{
		//	block: big.NewInt(13),
		//	rewards: rewards{
		//		WinnerCoinbase: new(big.Int).Add(Era1WinnerReward, new(big.Int).Mul(Era1WinnerUncleReward, common.Big2)),
		//		Uncle1Coinbase: Era1UncleReward,
		//		Uncle2Coinbase: Era1UncleReward,
		//	},
		//},
		//{
		//	block: big.NewInt(1914999),
		//	rewards: rewards{
		//		WinnerCoinbase: new(big.Int).Add(Era1WinnerReward, new(big.Int).Mul(Era1WinnerUncleReward, common.Big2)),
		//		Uncle1Coinbase: Era1UncleReward,
		//		Uncle2Coinbase: Era1UncleReward,
		//	},
		//},
		//{
		//	block: big.NewInt(1915000),
		//	rewards: rewards{
		//		WinnerCoinbase: new(big.Int).Add(Era1WinnerReward, new(big.Int).Mul(Era1WinnerUncleReward, common.Big2)),
		//		Uncle1Coinbase: Era1UncleReward,
		//		Uncle2Coinbase: Era1UncleReward,
		//	},
		//},
		//{
		//	block: big.NewInt(1915001),
		//	rewards: rewards{
		//		WinnerCoinbase: new(big.Int).Add(Era1WinnerReward, new(big.Int).Mul(Era1WinnerUncleReward, common.Big2)),
		//		Uncle1Coinbase: Era1UncleReward,
		//		Uncle2Coinbase: Era1UncleReward,
		//	},
		//},
		//{
		//	block: big.NewInt(4999999),
		//	rewards: rewards{
		//		WinnerCoinbase: new(big.Int).Add(Era1WinnerReward, new(big.Int).Mul(Era1WinnerUncleReward, common.Big2)),
		//		Uncle1Coinbase: Era1UncleReward,
		//		Uncle2Coinbase: Era1UncleReward,
		//	},
		//},
		{
			block: big.NewInt(5000001),
			rewards: rewards{
				WinnerCoinbase: new(big.Int).Add(Era2WinnerReward, new(big.Int).Mul(Era2WinnerUncleReward, common.Big2)),
				Uncle1Coinbase: Era2UncleReward,
				Uncle2Coinbase: Era2UncleReward,
			},
		},
		{
			block: big.NewInt(5000010),
			rewards: rewards{
				WinnerCoinbase: new(big.Int).Add(Era2WinnerReward, new(big.Int).Mul(Era2WinnerUncleReward, common.Big2)),
				Uncle1Coinbase: Era2UncleReward,
				Uncle2Coinbase: Era2UncleReward,
			},
		},
		{
			block: big.NewInt(10000000),
			rewards: rewards{
				WinnerCoinbase: new(big.Int).Add(Era2WinnerReward, new(big.Int).Mul(Era2WinnerUncleReward, common.Big2)),
				Uncle1Coinbase: Era2UncleReward,
				Uncle2Coinbase: Era2UncleReward,
			},
		},
		{
			block: big.NewInt(10000001),
			rewards: rewards{
				WinnerCoinbase: new(big.Int).Add(Era3WinnerReward, new(big.Int).Mul(Era3WinnerUncleReward, common.Big2)),
				Uncle1Coinbase: Era3UncleReward,
				Uncle2Coinbase: Era3UncleReward,
			},
		},
		{
			block: big.NewInt(15000000),
			rewards: rewards{
				WinnerCoinbase: new(big.Int).Add(Era3WinnerReward, new(big.Int).Mul(Era3WinnerUncleReward, common.Big2)),
				Uncle1Coinbase: Era3UncleReward,
				Uncle2Coinbase: Era3UncleReward,
			},
		},
	}

	for _, c := range cases {

		db, _ := ethdb.NewMemDatabase()
		stateDB, err := state.New(common.Hash{}, db)
		if err != nil {
			t.Fatalf("could not open statedb: %v", err)
		}

		var winner *types.Header = &types.Header{
			Number:   c.block,
			Coinbase: WinnerCoinbase,
		}
		var uncles []*types.Header = []*types.Header{{
			Number:   new(big.Int).Sub(c.block, common.Big1),
			Coinbase: Uncle1Coinbase,
		}, {
			Number:   new(big.Int).Sub(c.block, common.Big1),
			Coinbase: Uncle2Coinbase,
		}}

		gotWinnerBalance := stateDB.GetBalance(winner.Coinbase)
		gotUncle1Balance := stateDB.GetBalance(Uncle1Coinbase)
		gotUncle2Balance := stateDB.GetBalance(Uncle2Coinbase)
		r := new(big.Int)
		r.Add(gotWinnerBalance, gotUncle1Balance)
		r.Add(r, gotUncle2Balance)
		if r.Cmp(big.NewInt(0)) != 0 {
			t.Errorf("unexpected: %v", r)
		}

		AccumulateRewards(TestConfig, stateDB, winner, uncles)
		gotWinnerBalance = stateDB.GetBalance(winner.Coinbase)
		gotUncle1Balance = stateDB.GetBalance(Uncle1Coinbase)
		gotUncle2Balance = stateDB.GetBalance(Uncle2Coinbase)

		feat, _, configured := TestConfig.HasFeature("reward")
		if !configured {
			t.Error("unexpected reward feature not configured")
		}
		eraLen, ok := feat.GetBigInt("era")
		if !ok {
			t.Error("unexpected reward length not configured")
		}
		era := GetBlockEra(c.block, eraLen)

		// Check balances.
		if gotWinnerBalance.Cmp(c.rewards[WinnerCoinbase]) != 0 {
			t.Errorf("Era %v: winner balance @ %v, want: %v, got: %v, \n-> diff: %v", era, c.block, c.rewards[WinnerCoinbase], gotWinnerBalance, new(big.Int).Sub(gotWinnerBalance, c.rewards[WinnerCoinbase]))
		}
		if gotUncle1Balance.Cmp(c.rewards[Uncle1Coinbase]) != 0 {
			t.Errorf("Era %v: uncle1 balance @ %v, want: %v, got: %v, \n-> diff: %v", era, c.block, c.rewards[Uncle1Coinbase], gotUncle1Balance, new(big.Int).Sub(gotUncle1Balance, c.rewards[Uncle1Coinbase]))
		}
		if gotUncle2Balance.Cmp(c.rewards[Uncle2Coinbase]) != 0 {
			t.Errorf("Era %v: uncle2 balance @ %v, want: %v, got: %v, \n-> diff: %v", era, c.block, c.rewards[Uncle2Coinbase], gotUncle2Balance, new(big.Int).Sub(gotUncle2Balance, c.rewards[Uncle2Coinbase]))
		}
		db.Close()
	}
}
