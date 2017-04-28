package core

import (
	"math/big"
	"testing"

	"github.com/ethereumproject/go-ethereum/ethdb"
)

func TestConfigErrorProperties(t *testing.T) {
	if IsValidateError(ErrHashKnownBad) {
		t.Error("ErrHashKnownBad is a validation error")
	}
	if !IsValidateError(ErrHashKnownFork) {
		t.Error("ErrHashKnownFork is not a validation error")
	}
}

func TestChainConfig_IsHomestead(t *testing.T) {
	config := DefaultConfig

	if config.IsHomestead(big.NewInt(10000)) {
		t.Errorf("Unexpected for %d", 10000)
	}

	if !config.IsHomestead(big.NewInt(1920000)) {
		t.Errorf("Expected for %d", 1920000)
	}
	if !config.IsHomestead(big.NewInt(2325166)) {
		t.Errorf("Expected for %d", 2325166)
	}
	if !config.IsHomestead(big.NewInt(3000000)) {
		t.Errorf("Expected for %d", 3000000)
	}
	if !config.IsHomestead(big.NewInt(3000001)) {
		t.Errorf("Expected for %d", 3000001)
	}
	if !config.IsHomestead(big.NewInt(4000000)) {
		t.Errorf("Expected for %d", 3000000)
	}
	if !config.IsHomestead(big.NewInt(5000000)) {
		t.Errorf("Expected for %d", 5000000)
	}
	if !config.IsHomestead(big.NewInt(5000001)) {
		t.Errorf("Expected for %d", 5000001)
	}
}

func TestChainConfig_IsDiehard(t *testing.T) {
	config := DefaultConfig

	if config.IsDiehard(big.NewInt(1920000)) {
		t.Errorf("Unexpected for %d", 1920000)
	}

	if config.IsDiehard(big.NewInt(2325166)) {
		t.Errorf("Unexpected for %d", 2325166)
	}

	if !config.IsDiehard(big.NewInt(3000000)) {
		t.Errorf("Expected for %d", 3000000)
	}
	if !config.IsDiehard(big.NewInt(3000001)) {
		t.Errorf("Expected for %d", 3000001)
	}
	if !config.IsDiehard(big.NewInt(4000000)) {
		t.Errorf("Expected for %d", 3000000)
	}

	if !config.IsDiehard(big.NewInt(5000000)) {
		t.Errorf("Expected for %d", 5000000)
	}
	if !config.IsDiehard(big.NewInt(5000001)) {
		t.Errorf("Expected for %d", 5000001)
	}
}

func TestChainConfig_IsExplosion(t *testing.T) {
	config := DefaultConfig

	if config.IsExplosion(big.NewInt(1920000)) {
		t.Errorf("Unexpected for %d", 1920000)
	}

	if config.IsExplosion(big.NewInt(2325166)) {
		t.Errorf("Unexpected for %d", 2325166)
	}

	// Default Diehard block is 3000000
	if config.IsExplosion(big.NewInt(3000000)) {
		t.Errorf("Unxpected for %d", 3000000)
	}
	if config.IsExplosion(big.NewInt(3000001)) {
		t.Errorf("Unxpected for %d", 3000001)
	}
	if config.IsExplosion(big.NewInt(4000000)) {
		t.Errorf("Unxpected for %d", 3000000)
	}

	// Default BombDelay length is 2000000.
	if !config.IsExplosion(big.NewInt(5000000)) {
		t.Errorf("Expected for %d", 5000000)
	}
	if !config.IsExplosion(big.NewInt(5000001)) {
		t.Errorf("Expected for %d", 5000001)
	}

}

func sameGenesisDumpAllocationsBalances(gd1, gd2 *GenesisDump) bool {
	for address, alloc := range gd2.Alloc {
		bal1, _ := new(big.Int).SetString(gd1.Alloc[address].Balance, 0)
		bal2, _ := new(big.Int).SetString(alloc.Balance, 0)
		if bal1.Cmp(bal2) != 0 {
			return false
		}
	}
	return true
}

// TestMakeGenesisDump is a unit-ish test for MakeGenesisDump()
func TestMakeGenesisDump(t *testing.T) {
	// setup so we have a genesis block in this test db
	db, _ := ethdb.NewMemDatabase()
	genesisDump := &GenesisDump{
		Nonce:      "0x0000000000000042",
		Timestamp:  "0x0000000000000000000000000000000000000000000000000000000000000000",
		Coinbase:   "0x0000000000000000000000000000000000000000",
		Difficulty: "0x0000000000000000000000000000000000000000000000000000000400000000",
		ExtraData:  "0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa",
		GasLimit:   "0x0000000000000000000000000000000000000000000000000000000000001388",
		Mixhash:    "0x0000000000000000000000000000000000000000000000000000000000000000",
		ParentHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
		Alloc: map[hex]*GenesisDumpAlloc{
			"000d836201318ec6899a67540690382780743280": {Balance: "200000000000000000000"},
			"001762430ea9c3a26e5749afdb70da5f78ddbb8c": {Balance: "200000000000000000000"},
			"001d14804b399c6ef80e64576f657660804fec0b": {Balance: "4200000000000000000000"},
			"0032403587947b9f15622a68d104d54d33dbd1cd": {Balance: "77500000000000000000"},
			"00497e92cdc0e0b963d752b2296acb87da828b24": {Balance: "194800000000000000000"},
			"004bfbe1546bc6c65b5c7eaa55304b38bbfec6d3": {Balance: "2000000000000000000000"},
			"005a9c03f69d17d66cbb8ad721008a9ebbb836fb": {Balance: "2000000000000000000000"},
		},
	}
	gBlock1, err := WriteGenesisBlock(db, genesisDump)
	if err != nil {
		t.Errorf("WriteGenesisBlock could not setup genesisDump: err: %v", err)
	}

	// ensure equivalent genesis dumps in and out
	gotGenesisDump, err := MakeGenesisDump(db)
	if gotGenesisDump.Nonce != genesisDump.Nonce {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump nonce: wanted: %v, got: %v", genesisDump.Nonce, gotGenesisDump.Nonce)
	}
	if gotGenesisDump.Timestamp != genesisDump.Timestamp {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump timestamp: wanted: %v, got: %v", genesisDump.Timestamp, gotGenesisDump.Timestamp)
	}
	if gotGenesisDump.Coinbase != genesisDump.Coinbase {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump coinbase: wanted: %v, got: %v", genesisDump.Coinbase, gotGenesisDump.Coinbase)
	}
	if gotGenesisDump.Difficulty != genesisDump.Difficulty {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump difficulty: wanted: %v, got: %v", genesisDump.Difficulty, gotGenesisDump.Difficulty)
	}
	if gotGenesisDump.ExtraData != genesisDump.ExtraData {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump extra data: wanted: %v, got: %v", genesisDump.ExtraData, gotGenesisDump.ExtraData)
	}
	if gotGenesisDump.GasLimit != genesisDump.GasLimit {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump gaslimit: wanted: %v, got: %v", genesisDump.GasLimit, gotGenesisDump.GasLimit)
	}
	if gotGenesisDump.Mixhash != genesisDump.Mixhash {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump mixhash: wanted: %v, got: %v", genesisDump.Mixhash, gotGenesisDump.Mixhash)
	}
	if gotGenesisDump.ParentHash != genesisDump.ParentHash {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump parenthash: wanted: %v, got: %v", genesisDump.ParentHash, gotGenesisDump.ParentHash)
	}
	if !sameGenesisDumpAllocationsBalances(genesisDump, gotGenesisDump) {
		t.Error("MakeGenesisDump failed to make equivalent genesis dump allocations.")
	}

	// ensure equivalent genesis blocks in and out
	gBlock2, err := WriteGenesisBlock(db, gotGenesisDump)
	if err != nil {
		t.Errorf("WriteGenesisBlock could not setup gotGenesisDump: err: %v", err)
	}

	if gBlock1.Hash() != gBlock2.Hash() {
		t.Errorf("MakeGenesisDump failed to make genesis block with equivalent hashes: wanted: %, got: %v", gBlock1.Hash(), gBlock2.Hash())
	}
	db.Close()
}

func makeTestChainConfig() *ChainConfig {
	return DefaultConfig.SortForks()
}

// Unit-y tests.

// TestChainConfig_GetFeature should be able to get all features described in DefaultConfig.
func TestChainConfig_GetFeature(t *testing.T) {
	c := makeTestChainConfig()
	var dict = make(map[*big.Int][]string)
	for _, fork := range c.Forks {
		for _, feat := range fork.Features {
			dict[fork.Block] = append(dict[fork.Block], feat.ID)
		}
	}
	for block, ids := range dict {
		for _, name := range ids {
			feat, fork, ok := c.GetFeature(block, name)
			if !ok {
				t.Errorf("expected feature exist: feat: %v, fork: %v, block: %v", feat, fork, block)
			}
		}
	}
}

var allAvailableConfigKeys = []string{
	"difficulty",
	"gastable",
	"eip155",
}

// TestChainConfig_EventuallyGetAllPossibleFeatures should aggregate all available features from previous branches
func TestChainConfig_GetFeature2_EventuallyGetAllPossibleFeatures(t *testing.T) {
	c := makeTestChainConfig()
	for _, id := range allAvailableConfigKeys {
		if _, _, ok := c.GetFeature(big.NewInt(5000000), id); !ok {
			t.Errorf("could not get feature with id: %v, at block: %v", id, big.NewInt(5000000))
		}
	}
}

var unavailableConfigKeys = []string{
	"foo",
	"bar",
	"monkey",
}

// TestChainConfig_NeverGetNonexistantFeatures should never eventually collect features that don't exist
func TestChainConfig_GetFeature3_NeverGetNonexistantFeatures(t *testing.T) {
	c := makeTestChainConfig()
	for _, id := range unavailableConfigKeys {
		if feat, _, ok := c.GetFeature(big.NewInt(5000000), id); ok {
			t.Errorf("found unexpected feature: %v, for name: %v, at block: %v", feat, id, big.NewInt(5000000))
		}
	}
}

func TestChainConfig_GetFeature4_WorkForHighNumbers(t *testing.T) {
	c := makeTestChainConfig()
	highBlock := big.NewInt(99999999999999999)
	if _, _, ok := c.GetFeature(highBlock, "difficulty"); !ok {
		t.Errorf("unexpected unfound difficulty feature for far-future block: %v", highBlock)
	}
}

// Acceptance-y tests.

// Test GetFeature gets expected feature values from default configuration data...

// TestChainConfig_GetFeature_DefaultEIP155 should get the eip155 feature for (only and above) its default implemented block.
func TestChainConfig_GetFeature5_DefaultEIP155(t *testing.T) {
	c := makeTestChainConfig()
	var tables = map[*big.Int]*big.Int{
		big.NewInt(0).Sub(DefaultConfig.ForkByName("Homestead").Block, big.NewInt(1)): nil,
		DefaultConfig.ForkByName("Homestead").Block:                                   nil,
		big.NewInt(0).Add(DefaultConfig.ForkByName("Homestead").Block, big.NewInt(1)): nil,

		big.NewInt(0).Sub(DefaultConfig.ForkByName("GasReprice").Block, big.NewInt(1)): nil,
		DefaultConfig.ForkByName("GasReprice").Block:                                   nil,
		big.NewInt(0).Add(DefaultConfig.ForkByName("GasReprice").Block, big.NewInt(1)): nil,

		big.NewInt(0).Sub(DefaultConfig.ForkByName("Diehard").Block, big.NewInt(1)): nil,
		DefaultConfig.ForkByName("Diehard").Block:                                   big.NewInt(61),
		big.NewInt(0).Add(DefaultConfig.ForkByName("Diehard").Block, big.NewInt(1)): big.NewInt(61),
	}
	for block, expected := range tables {
		feat, fork, ok := c.GetFeature(block, "eip155")
		if expected != nil {
			if !ok {
				t.Errorf("Expected eip155 feature to exist. feat: %v, fork: %v, block: %v", feat, fork, block)
			}
			val, ok := feat.GetBigInt("chainID")
			if !ok {
				t.Errorf("failed to get value for eip155 feature. feat: %v, fork: %v, block: %v", feat, fork, block)
			}
			if val.Cmp(expected) != 0 {
				t.Errorf("want: %v, got: %v", expected, val)
			}
		} else {
			if ok {
				t.Errorf("Unexpected eip155 feature exists. feat: %v, fork: %v, block: %v", feat, fork, block)
			}
		}
	}
}

// TestChainConfig_GetFeature_DefaultGasTables sets that GetFeatures gets expected feature values for default fork configs.
func TestChainConfig_GetFeature6_DefaultGasTables(t *testing.T) {
	c := makeTestChainConfig()
	var tables = map[*big.Int]string{
		big.NewInt(0).Sub(DefaultConfig.ForkByName("Homestead").Block, big.NewInt(1)): "",
		DefaultConfig.ForkByName("Homestead").Block:                                   "homestead",
		big.NewInt(0).Add(DefaultConfig.ForkByName("Homestead").Block, big.NewInt(1)): "homestead",

		big.NewInt(0).Sub(DefaultConfig.ForkByName("GasReprice").Block, big.NewInt(1)): "homestead",
		DefaultConfig.ForkByName("GasReprice").Block:                                   "eip150",
		big.NewInt(0).Add(DefaultConfig.ForkByName("GasReprice").Block, big.NewInt(1)): "eip150",

		big.NewInt(0).Sub(DefaultConfig.ForkByName("Diehard").Block, big.NewInt(1)): "eip150",
		DefaultConfig.ForkByName("Diehard").Block:                                   "eip160",
		big.NewInt(0).Add(DefaultConfig.ForkByName("Diehard").Block, big.NewInt(1)): "eip160",
	}
	for block, expected := range tables {
		feat, fork, ok := c.GetFeature(block, "gastable")
		if expected != "" {
			if !ok {
				t.Errorf("Expected gastable feature to exist. feat: %v, fork: %v, block: %v", feat, fork, block)
			}
			val, ok := feat.GetStringOptions("type")
			if !ok {
				t.Errorf("failed to get value for gastable feature. feat: %v, fork: %v, block: %v", feat, fork, block)
			}
			if val != expected {
				t.Errorf("want: %v, got: %v", expected, val)
			}
		} else {
			if ok {
				t.Errorf("Unexpected gastable feature exists. feat: %v, fork: %v, block: %v", feat, fork, block)
			}
		}
	}
}

// TestChainConfig_GetFeature_DefaultGasTables sets that GetFeatures gets expected feature values for default fork configs.
func TestChainConfig_GetFeature7_DefaultDifficulty(t *testing.T) {
	c := makeTestChainConfig()
	var tables = map[*big.Int]string{
		big.NewInt(0).Sub(DefaultConfig.ForkByName("Homestead").Block, big.NewInt(1)): "",
		DefaultConfig.ForkByName("Homestead").Block:                                   "homestead",
		big.NewInt(0).Add(DefaultConfig.ForkByName("Homestead").Block, big.NewInt(1)): "homestead",

		big.NewInt(0).Sub(DefaultConfig.ForkByName("GasReprice").Block, big.NewInt(1)): "homestead",
		DefaultConfig.ForkByName("GasReprice").Block:                                   "homestead",
		big.NewInt(0).Add(DefaultConfig.ForkByName("GasReprice").Block, big.NewInt(1)): "homestead",

		big.NewInt(0).Sub(DefaultConfig.ForkByName("Diehard").Block, big.NewInt(1)): "homestead",
		DefaultConfig.ForkByName("Diehard").Block:                                   "ecip1010",
		big.NewInt(0).Add(DefaultConfig.ForkByName("Diehard").Block, big.NewInt(1)): "ecip1010",
	}
	for block, expected := range tables {
		feat, fork, ok := c.GetFeature(block, "difficulty")
		if expected != "" {
			if !ok {
				t.Errorf("Expected difficulty feature to exist. feat: %v, fork: %v, block: %v", feat, fork, block)
			}
			val, ok := feat.GetStringOptions("type")
			if !ok {
				t.Errorf("failed to get value for difficulty feature. feat: %v, fork: %v, block: %v", feat, fork, block)
			}
			if val != expected {
				t.Errorf("want: %v, got: %v", expected, val)
			}
		} else {
			if ok {
				t.Errorf("Unexpected difficulty feature exists. feat: %v, fork: %v, block: %v", feat, fork, block)
			}
		}
	}
}

func TestChainConfig_SortForks(t *testing.T) {
	// check code data default
	c := makeTestChainConfig()
	n := big.NewInt(0)
	for _, fork := range c.Forks {
		if n.Cmp(fork.Block) > 0 { t.Errorf("unexpected fork block: %v is greater than: %v", fork.Block, n) }
		n = fork.Block
	}

	// introduce disorder
	f := &Fork{}
	f.Block = big.NewInt(0).Sub(c.Forks[0].Block, big.NewInt(1))
	c.Forks = append(c.Forks, f) // last fork should be out of order

	c.SortForks()
	n = big.NewInt(0)
	for _, fork := range c.Forks {
		if n.Cmp(fork.Block) > 0 { t.Errorf("unexpected fork block: %v is greater than: %v", fork.Block, n) }
		n = fork.Block
	}
}