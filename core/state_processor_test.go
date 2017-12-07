package core

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"fmt"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/ethdb"
)

var (
	defaultEraLength *big.Int = big.NewInt(5000000)
)

// Unit tests.

// Use default era length 5,000,000
func TestGetBlockEra1(t *testing.T) {
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
		gotEra := GetBlockEra(bn, defaultEraLength)
		if gotEra.Cmp(expectedEra) != 0 {
			t.Errorf("got: %v, want: %v", gotEra, expectedEra)
		}
	}
}

// Use custom era length 2
func TestGetBlockEra2(t *testing.T) {
	cases := map[*big.Int]*big.Int{
		big.NewInt(0):  big.NewInt(0),
		big.NewInt(1):  big.NewInt(0),
		big.NewInt(2):  big.NewInt(0),
		big.NewInt(3):  big.NewInt(1),
		big.NewInt(4):  big.NewInt(1),
		big.NewInt(5):  big.NewInt(2),
		big.NewInt(6):  big.NewInt(2),
		big.NewInt(7):  big.NewInt(3),
		big.NewInt(8):  big.NewInt(3),
		big.NewInt(9):  big.NewInt(4),
		big.NewInt(10): big.NewInt(4),
		big.NewInt(11): big.NewInt(5),
		big.NewInt(12): big.NewInt(5),
	}

	for bn, expectedEra := range cases {
		gotEra := GetBlockEra(bn, big.NewInt(2))
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
		gotReward := GetBlockWinnerRewardByEra(GetBlockEra(bn, defaultEraLength))
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
	we2.Div(GetBlockWinnerRewardByEra(GetBlockEra(big.NewInt(5000001), defaultEraLength)), big.NewInt(32))
	we3.Div(GetBlockWinnerRewardByEra(GetBlockEra(big.NewInt(10000001), defaultEraLength)), big.NewInt(32))
	we4.Div(GetBlockWinnerRewardByEra(GetBlockEra(big.NewInt(15000001), defaultEraLength)), big.NewInt(32))

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

		era := GetBlockEra(bn, defaultEraLength)

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
		got := GetBlockWinnerRewardForUnclesByEra(GetBlockEra(bn, defaultEraLength), uncleSingle)
		if got.Cmp(want) != 0 {
			t.Errorf("@ %v: want: %v, got: %v", bn, want, got)
		}

		// test double uncle
		got = GetBlockWinnerRewardForUnclesByEra(GetBlockEra(bn, defaultEraLength), uncleDouble)
		dub := new(big.Int)
		if got.Cmp(dub.Mul(want, big.NewInt(2))) != 0 {
			t.Errorf("@ %v: want: %v, got: %v", bn, want, got)
		}
	}
}

// Integration tests.
//
// There are two kinds of integration tests: accumulating and non-accumulation.
// Accumulating tests check simulated accrual of a
// winner and two uncle accounts over the winnings of many mined blocks.
// If ecip1017 feature is not included in the hardcoded mainnet configuration, it will be temporarily
// included and tested in this test.
// This tests not only reward changes, but summations and state tallies over time.
// Non-accumulating tests check the one-off reward structure at any point
// over the specified era period.
// Currently tested eras are 1, 2, 3, and the beginning of 4.
// Both kinds of tests rely on manual calculations of 'want' account balance state,
// and purposely avoid using existing calculation functions in state_processor.go.
// Check points confirming calculations are at and around the 'boundaries' of forks and eras.
//
// Helpers.

// expectedEraForTesting is a 1-indexed version of era number,
// used exclusively for testing.
type expectedEraForTesting int

const (
	era1 expectedEraForTesting = iota + 1
	era2
	era3
	era4
)

type expectedRewards map[common.Address]*big.Int

func calculateExpectedEraRewards(era expectedEraForTesting, numUncles int) expectedRewards {
	wr := new(big.Int)
	wur := new(big.Int)
	ur := new(big.Int)
	switch era {
	case era1:
		wr = Era1WinnerReward
		wur = Era1WinnerUncleReward
		ur = Era1UncleReward
	case era2:
		wr = Era2WinnerReward
		wur = Era2WinnerUncleReward
		ur = Era2UncleReward
	case era3:
		wr = Era3WinnerReward
		wur = Era3WinnerUncleReward
		ur = Era3UncleReward
	case era4:
		wr = Era4WinnerReward
		wur = Era4WinnerUncleReward
		ur = Era4UncleReward
	}
	return expectedRewards{
		WinnerCoinbase: new(big.Int).Add(wr, new(big.Int).Mul(wur, big.NewInt(int64(numUncles)))),
		Uncle1Coinbase: ur,
		Uncle2Coinbase: ur,
	}
}

// expectedEraFromBlockNumber is similar to GetBlockEra, but it
// returns a 1-indexed version of the number of type expectedEraForTesting
func expectedEraFromBlockNumber(i, eralen *big.Int, t *testing.T) expectedEraForTesting {
	e := GetBlockEra(i, eralen)
	ePlusOne := new(big.Int).Add(e, big.NewInt(1)) // since expectedEraForTesting is not 0-indexed; iota + 1
	ei := ePlusOne.Int64()
	expEra := int(ei)
	if expEra > 4 || expEra < 1 {
		t.Fatalf("Unexpected era value, want 1 < e < 5, got: %d", expEra)
	}
	return expectedEraForTesting(expEra)
}

type expectedRewardCase struct {
	eraNum  expectedEraForTesting
	block   *big.Int
	rewards expectedRewards
}

// String implements stringer interface for expectedRewards
// Useful for logging tests for visual confirmation.
func (r expectedRewards) String() string {
	return fmt.Sprintf("w: %d, u1: %d, u2: %d", r[WinnerCoinbase], r[Uncle1Coinbase], r[Uncle2Coinbase])
}

// String implements stringer interface for expectedRewardCase --
// useful for double-checking test cases with t.Log
// to visually ensure getting all desired test cases.
func (c *expectedRewardCase) String() string {
	return fmt.Sprintf("block=%d era=%d rewards=%s", c.block, c.eraNum, c.rewards)
}

// makeExpectedRewardCasesForConfig makes an array of expectedRewardCases.
// It checks boundary cases for era length and fork numbers.
//
// An example of output:
// ----
//	{
//		// mainnet
//		{
//			block:   big.NewInt(2),
//			rewards: calculateExpectedEraRewards(era1, 1),
//		},
// ...
//		{
//			block:   big.NewInt(20000000),
//			rewards: calculateExpectedEraRewards(era4, 1),
//		},
//	},
func makeExpectedRewardCasesForConfig(c *ChainConfig, numUncles int, t *testing.T) []expectedRewardCase {
	erasToTest := []expectedEraForTesting{era1, era2, era3}
	eraLen := new(big.Int)
	feat, _, configured := c.HasFeature("reward")
	if !configured {
		eraLen = defaultEraLength
	} else {
		elen, ok := feat.GetBigInt("era")
		if !ok {
			t.Error("unexpected reward length not configured")
		} else {
			eraLen = elen
		}
	}

	var cases []expectedRewardCase
	var boundaryDiffs = []int64{-2, -1, 0, 1, 2}

	// Include trivial initial early block values.
	for _, i := range []*big.Int{big.NewInt(2), big.NewInt(13)} {
		cases = append(cases, expectedRewardCase{
			eraNum:  era1,
			block:   i,
			rewards: calculateExpectedEraRewards(era1, numUncles),
		})
	}

	// Test boundaries of forks.
	for _, f := range c.Forks {
		fn := f.Block
		for _, d := range boundaryDiffs {
			fnb := new(big.Int).Add(fn, big.NewInt(d))
			if fnb.Sign() < 1 {
				t.Fatalf("unexpected 0 or neg block number: %d", fnb)
			}
			expEra := expectedEraFromBlockNumber(fnb, eraLen, t)

			cases = append(cases, expectedRewardCase{
				eraNum:  expEra,
				block:   fnb,
				rewards: calculateExpectedEraRewards(expEra, numUncles),
			})
		}
	}

	// Test boundaries of era.
	for _, e := range erasToTest {
		for _, d := range boundaryDiffs {
			eb := big.NewInt(int64(e))
			eraBoundary := new(big.Int).Mul(eb, eraLen)
			bn := new(big.Int).Add(eraBoundary, big.NewInt(d))
			if bn.Sign() < 1 {
				t.Fatalf("unexpected 0 or neg block number: %d", bn)
			}
			era := expectedEraFromBlockNumber(bn, eraLen, t)
			cases = append(cases, expectedRewardCase{
				eraNum:  era,
				block:   bn,
				rewards: calculateExpectedEraRewards(era, numUncles),
			})
		}
	}

	return cases
}

// Accruing over block cases simulates miner account winning many times.
// Uses maps of running sums for winner & 2 uncles to keep tally.
func TestAccumulateRewards1(t *testing.T) {
	configs := []*ChainConfig{DefaultConfigMainnet.ChainConfig, DefaultConfigMorden.ChainConfig}
	cases := [][]expectedRewardCase{}
	for _, c := range configs {
		cases = append(cases, makeExpectedRewardCasesForConfig(c, 2, t))
	}

	// t.Logf("Accruing balances over cases. 2 uncles. Configs mainnet=0, morden=1")
	for i, config := range configs {
		// Set up era len by chain configurations.
		feat, _, exists := config.HasFeature("reward")
		eraLen := new(big.Int)
		if !exists {
			// t.Logf("No ecip1017 feature installed for config=%d, setting up a placeholder ecip1017 feature for testing.", i)
			dhFork := config.ForkByName("Diehard")
			dhFork.Features = append(dhFork.Features, &ForkFeature{
				ID: "reward",
				Options: ChainFeatureConfigOptions{
					"type": "ecip1017",
					"era":  5000000, // for mainnet will be 5m
				},
			})
			feat, _, exists = config.HasFeature("reward")
			if !exists {
				t.Fatal("no expected feature installed")
			}
		}
		eraLen, ok := feat.GetBigInt("era")
		if !ok {
			t.Error("No era length configured, is required.")
		}

		db, _ := ethdb.NewMemDatabase()

		stateDB, err := state.New(common.Hash{}, db)
		if err != nil {
			t.Fatalf("could not open statedb: %v", err)
		}

		var header *types.Header = &types.Header{}
		var uncles []*types.Header = []*types.Header{{}, {}}

		if i == 0 {
			header.Coinbase = common.StringToAddress("000d836201318ec6899a67540690382780743280")
			uncles[0].Coinbase = common.StringToAddress("001762430ea9c3a26e5749afdb70da5f78ddbb8c")
			uncles[1].Coinbase = common.StringToAddress("001d14804b399c6ef80e64576f657660804fec0b")
		} else {
			header.Coinbase = common.StringToAddress("0000000000000000000000000000000000000001")
			uncles[0].Coinbase = common.StringToAddress("0000000000000000000000000000000000000002")
			uncles[1].Coinbase = common.StringToAddress("0000000000000000000000000000000000000003")
		}

		// Manual tallies for reward accumulation.
		winnerB, totalB := new(big.Int), new(big.Int)
		unclesB := []*big.Int{new(big.Int), new(big.Int)}

		winnerB = stateDB.GetBalance(header.Coinbase)
		unclesB[0] = stateDB.GetBalance(uncles[0].Coinbase)
		unclesB[1] = stateDB.GetBalance(uncles[1].Coinbase)

		totalB.Add(totalB, winnerB)
		totalB.Add(totalB, unclesB[0])
		totalB.Add(totalB, unclesB[1])

		if totalB.Cmp(big.NewInt(0)) != 0 {
			t.Errorf("unexpected: %v", totalB)
		}

		for _, c := range cases[i] {
			bn := c.block
			era := GetBlockEra(bn, eraLen)

			header.Number = bn

			for i, uncle := range uncles {

				// Randomize uncle numbers with bound ( n-1 <= uncleNum <= n-7 ), where n is current head number
				// See yellowpaper@11.1 for ommer validation reference. I expect n-7 is 6th-generation ommer.
				// Note that ommer nth-generation impacts reward only for "Era 1".
				rand.Seed(time.Now().UTC().UnixNano())

				// 1 + [0..rand..7) == 1 + 0, 1 + 1, ... 1 + 6
				un := new(big.Int).Add(big.NewInt(1), big.NewInt(int64(rand.Int31n(int32(7)))))
				uncle.Number = new(big.Int).Sub(header.Number, un) // n - un

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
			//t.Logf("config=%d block=%d era=%d w:%d u1:%d u2:%d", i, bn, new(big.Int).Add(era, big.NewInt(1)), winnerB, unclesB[0], unclesB[1])
			if wb := stateDB.GetBalance(header.Coinbase); wb.Cmp(winnerB) != 0 {
				t.Errorf("winner balance @ %v, want: %v, got: %v (config: %v)", bn, winnerB, wb, i)
			}
			if uB0 := stateDB.GetBalance(uncles[0].Coinbase); unclesB[0].Cmp(uB0) != 0 {
				t.Errorf("uncle1 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[0], uB0, i)
			}
			if uB1 := stateDB.GetBalance(uncles[1].Coinbase); unclesB[1].Cmp(uB1) != 0 {
				t.Errorf("uncle2 balance @ %v, want: %v, got: %v (config: %v)", bn, unclesB[1], uB1, i)
			}
		}
		db.Close()
	}
}

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

	Era3WinnerReward      = new(big.Int).Mul(new(big.Int).Div(Era2WinnerReward, big.NewInt(5)), big.NewInt(4))
	Era3WinnerUncleReward = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Div(Era2WinnerReward, big.NewInt(5)), big.NewInt(4)), big32)
	Era3UncleReward       = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Div(Era2WinnerReward, big.NewInt(5)), big.NewInt(4)), big32)

	Era4WinnerReward      = new(big.Int).Mul(new(big.Int).Div(Era3WinnerReward, big.NewInt(5)), big.NewInt(4))
	Era4WinnerUncleReward = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Div(Era3WinnerReward, big.NewInt(5)), big.NewInt(4)), big32)
	Era4UncleReward       = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Div(Era3WinnerReward, big.NewInt(5)), big.NewInt(4)), big32)
)

// Non-accruing over block cases simulates instance,
// ie. a miner wins once at different blocks.
//
// Tests winner includes 2 ommer headers.
func TestAccumulateRewards2_2Uncles(t *testing.T) {

	// Order matters here; expected cases must be ordered the same.
	// Will uses indexes to match expectations -> test outcomes.
	configs := []*ChainConfig{DefaultConfigMainnet.ChainConfig, DefaultConfigMorden.ChainConfig}
	cases := [][]expectedRewardCase{}
	for _, c := range configs {
		cases = append(cases, makeExpectedRewardCasesForConfig(c, 2, t))
	}
	// t.Logf("Non-accruing balances over cases. 2 uncles. Configs mainnet=0, morden=1")
	for i, config := range configs {
		// Here's where cases slice is assign according to config slice.
		for _, c := range cases[i] {
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
				Number:   new(big.Int).Sub(c.block, common.Big1), // use 1st-generation ommer, since random n-[1,7) is tested by accrual above
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

			AccumulateRewards(config, stateDB, winner, uncles)
			gotWinnerBalance = stateDB.GetBalance(winner.Coinbase)
			gotUncle1Balance = stateDB.GetBalance(Uncle1Coinbase)
			gotUncle2Balance = stateDB.GetBalance(Uncle2Coinbase)

			// Use config if possible. Currently installed on testnet only.
			// If not configured, assume default and still test it.
			eraLen := new(big.Int)
			feat, _, configured := config.HasFeature("reward")
			if !configured {
				eraLen = defaultEraLength
			} else {
				elen, ok := feat.GetBigInt("era")
				if !ok {
					t.Error("unexpected reward length not configured")
				} else {
					eraLen = elen
				}
			}
			era := GetBlockEra(c.block, eraLen)

			// Check we have expected era number.
			indexed1EraNum := new(big.Int).Add(era, big.NewInt(1))
			if indexed1EraNum.Cmp(big.NewInt(int64(c.eraNum))) != 0 {
				t.Errorf("era num mismatch, want: %v, got %v", c.eraNum, indexed1EraNum)
			}

			// Check balances.
			// t.Logf("config=%d block=%d era=%d w:%d u1:%d u2:%d", i, c.block, c.eraNum, gotWinnerBalance, gotUncle1Balance, gotUncle2Balance)
			if configured {
				if gotWinnerBalance.Cmp(c.rewards[WinnerCoinbase]) != 0 {
					t.Errorf("Config: %v | Era %v: winner balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, c.rewards[WinnerCoinbase], gotWinnerBalance, new(big.Int).Sub(gotWinnerBalance, c.rewards[WinnerCoinbase]))
				}
				if gotUncle1Balance.Cmp(c.rewards[Uncle1Coinbase]) != 0 {
					t.Errorf("Config: %v | Era %v: uncle1 balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, c.rewards[Uncle1Coinbase], gotUncle1Balance, new(big.Int).Sub(gotUncle1Balance, c.rewards[Uncle1Coinbase]))
				}
				if gotUncle2Balance.Cmp(c.rewards[Uncle2Coinbase]) != 0 {
					t.Errorf("Config: %v | Era %v: uncle2 balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, c.rewards[Uncle2Coinbase], gotUncle2Balance, new(big.Int).Sub(gotUncle2Balance, c.rewards[Uncle2Coinbase]))
				}
			} else {
				if gotWinnerBalance.Cmp(new(big.Int).Add(Era1WinnerReward, new(big.Int).Mul(Era1WinnerUncleReward, big.NewInt(2)))) != 0 {
					t.Errorf("Config: %v | Era %v: winner balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, new(big.Int).Add(Era1WinnerReward, new(big.Int).Mul(Era1WinnerUncleReward, big.NewInt(2))), gotWinnerBalance, new(big.Int).Sub(gotWinnerBalance, c.rewards[WinnerCoinbase]))
				}
				if gotUncle1Balance.Cmp(Era1UncleReward) != 0 {
					t.Errorf("Config: %v | Era %v: uncle1 balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, Era1UncleReward, gotUncle1Balance, new(big.Int).Sub(gotUncle1Balance, c.rewards[Uncle1Coinbase]))
				}
				if gotUncle2Balance.Cmp(Era1UncleReward) != 0 {
					t.Errorf("Config: %v | Era %v: uncle2 balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, Era1UncleReward, gotUncle2Balance, new(big.Int).Sub(gotUncle2Balance, c.rewards[Uncle2Coinbase]))
				}
			}
			db.Close()
		}
	}
}

// Non-accruing over block cases simulates instance,
// ie. a miner wins once at different blocks.
//
// Tests winner includes 1 ommer header.
func TestAccumulateRewards3_1Uncle(t *testing.T) {

	configs := []*ChainConfig{DefaultConfigMainnet.ChainConfig, DefaultConfigMorden.ChainConfig}
	cases := [][]expectedRewardCase{}
	for _, c := range configs {
		cases = append(cases, makeExpectedRewardCasesForConfig(c, 1, t))
	}
	// t.Logf("Non-accruing balances over cases. 1 uncle. Configs mainnet=0, morden=1")
	for i, config := range configs {
		for _, c := range cases[i] {

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
				Number:   new(big.Int).Sub(c.block, common.Big1), // use 1st-generation ommer, since random n-[1,7) is tested by accrual above
				Coinbase: Uncle1Coinbase,
			}}

			gotWinnerBalance := stateDB.GetBalance(winner.Coinbase)
			gotUncle1Balance := stateDB.GetBalance(Uncle1Coinbase)
			r := new(big.Int)
			r.Add(gotWinnerBalance, gotUncle1Balance)
			if r.Cmp(big.NewInt(0)) != 0 {
				t.Errorf("unexpected: %v", r)
			}

			AccumulateRewards(config, stateDB, winner, uncles)
			gotWinnerBalance = stateDB.GetBalance(winner.Coinbase)
			gotUncle1Balance = stateDB.GetBalance(Uncle1Coinbase)

			// Use config if possible. Currently on testnet only.
			eraLen := new(big.Int)
			feat, _, configured := config.HasFeature("reward")
			if !configured {
				eraLen = defaultEraLength
			} else {
				elen, ok := feat.GetBigInt("era")
				if !ok {
					t.Error("unexpected reward length not configured")
				} else {
					eraLen = elen
				}
			}
			era := GetBlockEra(c.block, eraLen)

			// Check we have expected era number.
			indexed1EraNum := new(big.Int).Add(era, big.NewInt(1))
			if indexed1EraNum.Cmp(big.NewInt(int64(c.eraNum))) != 0 {
				t.Errorf("era num mismatch, want: %v, got %v", c.eraNum, indexed1EraNum)
			}

			// Check balances.
			// t.Logf("config=%d block=%d era=%d w:%d u1:%d", i, c.block, c.eraNum, gotWinnerBalance, gotUncle1Balance)
			if configured {
				if gotWinnerBalance.Cmp(c.rewards[WinnerCoinbase]) != 0 {
					t.Errorf("Config: %v | Era %v: winner balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, c.rewards[WinnerCoinbase], gotWinnerBalance, new(big.Int).Sub(gotWinnerBalance, c.rewards[WinnerCoinbase]))
				}
				if gotUncle1Balance.Cmp(c.rewards[Uncle1Coinbase]) != 0 {
					t.Errorf("Config: %v | Era %v: uncle1 balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, c.rewards[Uncle1Coinbase], gotUncle1Balance, new(big.Int).Sub(gotUncle1Balance, c.rewards[Uncle1Coinbase]))
				}
			} else {
				if gotWinnerBalance.Cmp(new(big.Int).Add(Era1WinnerReward, new(big.Int).Mul(Era1WinnerUncleReward, big.NewInt(1)))) != 0 {
					t.Errorf("Config: %v | Era %v: winner balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, new(big.Int).Add(Era1WinnerReward, new(big.Int).Mul(Era1WinnerUncleReward, big.NewInt(1))), gotWinnerBalance, new(big.Int).Sub(gotWinnerBalance, c.rewards[WinnerCoinbase]))
				}
				if gotUncle1Balance.Cmp(Era1UncleReward) != 0 {
					t.Errorf("Config: %v | Era %v: uncle1 balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, Era1UncleReward, gotUncle1Balance, new(big.Int).Sub(gotUncle1Balance, c.rewards[Uncle1Coinbase]))
				}
			}

			db.Close()
		}
	}
}

// Non-accruing over block cases simulates instance,
// ie. a miner wins once at different blocks.
//
// Tests winner includes 0 ommer headers.
func TestAccumulateRewards4_0Uncles(t *testing.T) {

	configs := []*ChainConfig{DefaultConfigMainnet.ChainConfig, DefaultConfigMorden.ChainConfig}
	cases := [][]expectedRewardCase{}
	for _, c := range configs {
		cases = append(cases, makeExpectedRewardCasesForConfig(c, 0, t))
	}
	// t.Logf("Non-accruing balances over cases. 0 uncles. Configs mainnet=0, morden=1")
	for i, config := range configs {
		for _, c := range cases[i] {

			db, _ := ethdb.NewMemDatabase()
			stateDB, err := state.New(common.Hash{}, db)
			if err != nil {
				t.Fatalf("could not open statedb: %v", err)
			}

			var winner *types.Header = &types.Header{
				Number:   c.block,
				Coinbase: WinnerCoinbase,
			}
			var uncles []*types.Header = []*types.Header{}

			gotWinnerBalance := stateDB.GetBalance(winner.Coinbase)
			if gotWinnerBalance.Cmp(big.NewInt(0)) != 0 {
				t.Errorf("unexpected: %v", gotWinnerBalance)
			}

			AccumulateRewards(config, stateDB, winner, uncles)
			gotWinnerBalance = stateDB.GetBalance(winner.Coinbase)

			// Use config if possible. Currently on testnet only.
			eraLen := new(big.Int)
			feat, _, configured := config.HasFeature("reward")
			if !configured {
				eraLen = defaultEraLength
			} else {
				elen, ok := feat.GetBigInt("era")
				if !ok {
					t.Error("unexpected reward length not configured")
				} else {
					eraLen = elen
				}
			}
			era := GetBlockEra(c.block, eraLen)

			// Check balances.
			// t.Logf("config=%d block=%d era=%d w:%d", i, c.block, c.eraNum, gotWinnerBalance)
			if configured {
				if gotWinnerBalance.Cmp(c.rewards[WinnerCoinbase]) != 0 {
					t.Errorf("Config: %v | Era %v: winner balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, c.rewards[WinnerCoinbase], gotWinnerBalance, new(big.Int).Sub(gotWinnerBalance, c.rewards[WinnerCoinbase]))
				}
			} else {
				if gotWinnerBalance.Cmp(Era1WinnerReward) != 0 {
					t.Errorf("Config: %v | Era %v: winner balance @ %v, want: %v, got: %v, \n-> diff: %v", i, era, c.block, Era1WinnerReward, gotWinnerBalance, new(big.Int).Sub(gotWinnerBalance, c.rewards[WinnerCoinbase]))
				}
			}

			db.Close()
		}
	}
}
