// Copyright 2015 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereumproject/ethash"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/event"
)

func testChainConfig() *ChainConfig {
	return &ChainConfig{
		Forks: []*Fork{
			{
				Name:  "Homestead",
				Block: big.NewInt(0),
				Features: []*ForkFeature{
					{
						ID: "gastable",
						Options: ChainFeatureConfigOptions{
							"type": "homestead",
						},
					},
					{
						ID: "difficulty",
						Options: ChainFeatureConfigOptions{
							"type": "homestead",
						},
					},
				},
			},
			{
				Name:  "Diehard",
				Block: big.NewInt(5),
				Features: []*ForkFeature{
					{
						ID: "eip155",
						Options: ChainFeatureConfigOptions{
							"chainID": 62,
						},
					},
					{ // ecip1010 bomb delay
						ID: "gastable",
						Options: ChainFeatureConfigOptions{
							"type": "eip160",
						},
					},
					{ // ecip1010 bomb delay
						ID: "difficulty",
						Options: ChainFeatureConfigOptions{
							"type":   "ecip1010",
							"length": 2000000,
						},
					},
				},
			},
		},
	}
}

func proc(t testing.TB) (Validator, *BlockChain) {
	db, err := ethdb.NewMemDatabase()
	if err != nil {
		t.Fatal(err)
	}
	_, err = WriteGenesisBlock(db, TestNetGenesis)
	if err != nil {
		t.Fatal(err)
	}

	pow, err := ethash.NewForTesting()
	if err != nil {
		t.Fatal(err)
	}

	var mux event.TypeMux
	blockchain, err := NewBlockChain(db, testChainConfig(), pow, &mux)
	if err != nil {
		t.Fatal(err)
	}
	return blockchain.validator, blockchain
}

func TestNumber(t *testing.T) {
	_, chain := proc(t)

	statedb, err := state.New(chain.Genesis().Root(), chain.chainDb)
	if err != nil {
		t.Fatal(err)
	}
	header := makeHeader(chain.config, chain.Genesis(), statedb)
	header.Number = big.NewInt(3)
	cfg := testChainConfig()
	err = ValidateHeader(cfg, nil, header, chain.Genesis().Header(), false, false)
	if err != BlockNumberErr {
		t.Errorf("expected block number error, got %q", err)
	}

	header = makeHeader(chain.config, chain.Genesis(), statedb)
	err = ValidateHeader(cfg, nil, header, chain.Genesis().Header(), false, false)
	if err == BlockNumberErr {
		t.Errorf("didn't expect block number error")
	}
}

func TestPutReceipt(t *testing.T) {
	db, err := ethdb.NewMemDatabase()
	if err != nil {
		t.Fatal(err)
	}

	var addr common.Address
	addr[0] = 1
	var hash common.Hash
	hash[0] = 2

	receipt := new(types.Receipt)
	receipt.Logs = vm.Logs{&vm.Log{
		Address:     addr,
		Topics:      []common.Hash{hash},
		Data:        []byte("hi"),
		BlockNumber: 42,
		TxHash:      hash,
		TxIndex:     0,
		BlockHash:   hash,
		Index:       0,
	}}

	WriteReceipts(db, types.Receipts{receipt})
	receipt = GetReceipt(db, common.Hash{})
	if receipt == nil {
		t.Error("expected to get 1 receipt, got none.")
	}
}

func TestDifficultyBombFreeze(t *testing.T) {
	diehardBlock := big.NewInt(3000000)
	var parentTime uint64

	// 20 seconds, parent diff 7654414978364
	parentTime = 1452838500
	act := calcDifficultyDiehard(parentTime+20, parentTime, big.NewInt(7654414978364), diehardBlock)
	exp := big.NewInt(7650945906507)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 5 seconds, parent diff 7654414978364
	act = calcDifficultyDiehard(parentTime+5, parentTime, big.NewInt(7654414978364), diehardBlock)
	exp = big.NewInt(7658420921133)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 80 seconds, parent diff 7654414978364
	act = calcDifficultyDiehard(parentTime+80, parentTime, big.NewInt(7654414978364), diehardBlock)
	exp = big.NewInt(7628520862629)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 76 seconds, parent diff 12657404717367
	parentTime = 1469081721
	act = calcDifficultyDiehard(parentTime+76, parentTime, big.NewInt(12657404717367), diehardBlock)
	exp = big.NewInt(12620590912441)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 1 second, parent diff 12620590912441
	parentTime = 1469164521
	act = calcDifficultyDiehard(parentTime+1, parentTime, big.NewInt(12620590912441), diehardBlock)
	exp = big.NewInt(12627021745803)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 10 seconds, parent diff 12627021745803
	parentTime = 1469164522
	act = calcDifficultyDiehard(parentTime+10, parentTime, big.NewInt(12627021745803), diehardBlock)
	exp = big.NewInt(12627290181259)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

}

func TestDifficultyBombFreezeTestnet(t *testing.T) {
	diehardBlock := big.NewInt(1915000)

	act := calcDifficultyDiehard(1481895639, 1481850947, big.NewInt(28670444), diehardBlock)
	exp := big.NewInt(27415615)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}
}

func TestDifficultyBombExplode(t *testing.T) {
	diehardBlock := big.NewInt(3000000)
	explosionBlock := big.NewInt(5000000)

	var parentTime uint64

	// 6 seconds, blocks in 5m
	num := big.NewInt(5000102)
	parentTime = 1513175023
	act := calcDifficultyExplosion(parentTime+6, parentTime, num, big.NewInt(22627021745803), diehardBlock, explosionBlock)
	exp := big.NewInt(22638338531720)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(5001105)
	parentTime = 1513189406
	act = calcDifficultyExplosion(parentTime+29, parentTime, num, big.NewInt(22727021745803), diehardBlock, explosionBlock)
	exp = big.NewInt(22716193002673)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(5100123)
	parentTime = 1514609324
	act = calcDifficultyExplosion(parentTime+41, parentTime, num, big.NewInt(22893437765583), diehardBlock, explosionBlock)
	exp = big.NewInt(22860439327271)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(6150001)
	parentTime = 1529664575
	act = calcDifficultyExplosion(parentTime+105, parentTime, num, big.NewInt(53134780363303), diehardBlock, explosionBlock)
	exp = big.NewInt(53451033724425)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(6550001)
	parentTime = 1535431724
	act = calcDifficultyExplosion(parentTime+60, parentTime, num, big.NewInt(82893437765583), diehardBlock, explosionBlock)
	exp = big.NewInt(91487154230751)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(7000000)
	parentTime = 1535431724
	act = calcDifficultyExplosion(parentTime+180, parentTime, num, big.NewInt(304334879167015), diehardBlock, explosionBlock)
	exp = big.NewInt(583283638618965)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(8000000)
	parentTime = 1535431724
	act = calcDifficultyExplosion(parentTime+420, parentTime, num, big.NewInt(78825323605416810), diehardBlock, explosionBlock)
	exp = big.NewInt(365477653727918567)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(9000000)
	parentTime = 1535431724
	act = calcDifficultyExplosion(parentTime+2040, parentTime, num, big.NewInt(288253236054168103), diehardBlock, explosionBlock)
	exp = big.NewInt(0)
	exp.SetString("295422224299015703633", 10)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}
}

func TestDifficultyBombExplodeTestnet(t *testing.T) {
	diehardBlock := big.NewInt(1915000)
	explosionBlock := big.NewInt(3415000)

	var parentTime uint64
	parentTime = 1513175023
	act := calcDifficultyExplosion(parentTime+20, parentTime, big.NewInt(5200000), big.NewInt(28670444), diehardBlock, explosionBlock)
	exp := big.NewInt(34388394813)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}
}

// Compare expected difficulties on edges of forks.
func TestCalcDifficulty1(t *testing.T) {
	configs := []*ChainConfig{DefaultConfig, TestConfig}
	for i, config := range configs {

		parentTime := uint64(1513175023)
		time := parentTime + 20
		parentDiff := big.NewInt(28670444)

		dhB := config.ForkByName("Diehard").Block
		if dhB == nil {
			t.Error("missing Diehard fork block")
		}

		feat, dhFork, configured := config.GetFeature(dhB, "difficulty")
		if !configured {
			t.Errorf("difficulty not configured for diehard block: %v", dhB)
		}
		if val, ok := feat.GetString("type"); !ok || val != "ecip1010" {
			t.Errorf("ecip1010 not configured as difficulty for diehard block: %v", dhB)
		}
		delay, ok := feat.GetBigInt("length")
		if !ok {
			t.Error("ecip1010 bomb delay length not configured")
		}

		explosionBlock := big.NewInt(0).Add(dhB, delay)

		//parentBlocks to compare with expected equality
		table := map[*big.Int]*big.Int{

			big.NewInt(0).Add(config.ForkByName("Homestead").Block, big.NewInt(-2)): calcDifficultyFrontier(time, parentTime, big.NewInt(0).Add(config.ForkByName("Homestead").Block, big.NewInt(-2)), parentDiff),
			big.NewInt(0).Add(config.ForkByName("Homestead").Block, big.NewInt(-1)): calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("Homestead").Block, big.NewInt(-1)), parentDiff),
			big.NewInt(0).Add(config.ForkByName("Homestead").Block, big.NewInt(0)):  calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("Homestead").Block, big.NewInt(0)), parentDiff),
			big.NewInt(0).Add(config.ForkByName("Homestead").Block, big.NewInt(1)):  calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("Homestead").Block, big.NewInt(1)), parentDiff),

			big.NewInt(0).Add(config.ForkByName("The DAO Hard Fork").Block, big.NewInt(-2)): calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("The DAO Hard Fork").Block, big.NewInt(-2)), parentDiff),
			big.NewInt(0).Add(config.ForkByName("The DAO Hard Fork").Block, big.NewInt(-1)): calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("The DAO Hard Fork").Block, big.NewInt(-1)), parentDiff),
			big.NewInt(0).Add(config.ForkByName("The DAO Hard Fork").Block, big.NewInt(0)):  calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("The DAO Hard Fork").Block, big.NewInt(0)), parentDiff),
			big.NewInt(0).Add(config.ForkByName("The DAO Hard Fork").Block, big.NewInt(1)):  calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("The DAO Hard Fork").Block, big.NewInt(1)), parentDiff),

			big.NewInt(0).Add(config.ForkByName("GasReprice").Block, big.NewInt(-2)): calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("GasReprice").Block, big.NewInt(-2)), parentDiff),
			big.NewInt(0).Add(config.ForkByName("GasReprice").Block, big.NewInt(-1)): calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("GasReprice").Block, big.NewInt(-1)), parentDiff),
			big.NewInt(0).Add(config.ForkByName("GasReprice").Block, big.NewInt(0)):  calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("GasReprice").Block, big.NewInt(0)), parentDiff),
			big.NewInt(0).Add(config.ForkByName("GasReprice").Block, big.NewInt(1)):  calcDifficultyHomestead(time, parentTime, big.NewInt(0).Add(config.ForkByName("GasReprice").Block, big.NewInt(1)), parentDiff),

			big.NewInt(0).Add(dhB, big.NewInt(-1)): calcDifficultyDiehard(time, parentTime, parentDiff, dhFork.Block), // 2999999
			big.NewInt(0).Add(dhB, big.NewInt(0)):  calcDifficultyDiehard(time, parentTime, parentDiff, dhFork.Block), // 3000000
			big.NewInt(0).Add(dhB, big.NewInt(1)):  calcDifficultyDiehard(time, parentTime, parentDiff, dhFork.Block), // 3000001
			big.NewInt(-2).Add(dhB, delay):         calcDifficultyDiehard(time, parentTime, parentDiff, dhFork.Block), // 4999998

			big.NewInt(-1).Add(dhB, delay): calcDifficultyExplosion(time, parentTime, big.NewInt(-1).Add(dhB, delay), parentDiff, dhFork.Block, explosionBlock), // 4999999
			big.NewInt(0).Add(dhB, delay):  calcDifficultyExplosion(time, parentTime, big.NewInt(0).Add(dhB, delay), parentDiff, dhFork.Block, explosionBlock),  // 5000000
			big.NewInt(1).Add(dhB, delay):  calcDifficultyExplosion(time, parentTime, big.NewInt(1).Add(dhB, delay), parentDiff, dhFork.Block, explosionBlock),  // 5000001

			big.NewInt(10000000): calcDifficultyExplosion(time, parentTime, big.NewInt(10000000), parentDiff, dhFork.Block, explosionBlock),
		}

		for parentNum, expected := range table {
			difficulty := CalcDifficulty(config, time, parentTime, parentNum, parentDiff)
			if difficulty.Cmp(expected) != 0 {
				t.Errorf("config: %v, got: %v, want: %v, with parentBlock: %v", i, difficulty, expected, parentNum)
			}
		}
	}
}
