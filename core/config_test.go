package core

import (
	"math/big"
	"testing"

	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"path/filepath"
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
		if gd1.Alloc[address] != nil {
			bal1, _ := new(big.Int).SetString(gd1.Alloc[address].Balance, 0)
			bal2, _ := new(big.Int).SetString(alloc.Balance, 0)
			if bal1.Cmp(bal2) != 0 {
				return false
			}
		} else if alloc.Balance != "" {
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
		t.Errorf("MakeGenesisDump failed to make genesis block with equivalent hashes: wanted: %v, got: %v", gBlock1.Hash(), gBlock2.Hash())
	}
	db.Close()
}

func TestMakeGenesisDump2(t *testing.T) {
	// setup so we have a genesis block in this test db
	for i, gen := range []*GenesisDump{DefaultGenesis, TestNetGenesis} {
		db, _ := ethdb.NewMemDatabase()
		genesisDump := gen
		gBlock1, err := WriteGenesisBlock(db, genesisDump)
		if err != nil {
			t.Errorf("WriteGenesisBlock could not setup initial genesisDump: %v, err: %v", i, err)
		}

		// ensure equivalent genesis dumps in and out
		gotGenesisDump, err := MakeGenesisDump(db)

		// ensure equivalent genesis blocks in and out
		gBlock2, err := WriteGenesisBlock(db, gotGenesisDump)
		if err != nil {
			t.Errorf("WriteGenesisBlock could not setup gotGenesisDump: %v, err: %v", i, err)
		}

		if !sameGenesisDumpAllocationsBalances(genesisDump, gotGenesisDump) {
			t.Error("MakeGenesisDump failed to make equivalent genesis dump allocations.")
		}

		if gBlock1.Hash() != gBlock2.Hash() {
			t.Errorf(`MakeGenesisDump failed to make genesis block with equivalent hashes: %v: wanted: %v, got: %v
			WANTED:
			%v

			GOT:
			%v`, i, gBlock1.Hash().Hex(), gBlock2.Hash().Hex(), gBlock1.String(), gBlock2.String())

		}
		db.Close()
	}
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

func TestChainConfig_GetChainID(t *testing.T) {
	// Test default hardcoded configs.
	if DefaultConfig.GetChainID().Cmp(DefaultChainConfigChainID) != 0 {
		t.Error("got: %v, want: %v", DefaultConfig.GetChainID(), DefaultTestnetChainConfigChainID)
	}
	if TestConfig.GetChainID().Cmp(DefaultTestnetChainConfigChainID) != 0 {
		t.Error("got: %v, want: %v", TestConfig.GetChainID(), DefaultTestnetChainConfigChainID)
	}

	// If no chainID (config is empty) returns 0.
	c := &ChainConfig{}
	cid := c.GetChainID()
	// check is zero
	if cid.Cmp(new(big.Int)) != 0 {
		t.Errorf("got: %v, want: %v", cid, new(big.Int))
	}

	// Test parsing default external mainnet config.
	cases := map[string]*big.Int{
		"../cmd/geth/config/mainnet.json": DefaultChainConfigChainID,
		"../cmd/geth/config/testnet.json": DefaultTestnetChainConfigChainID,
	}
	for extConfigPath, wantInt := range cases {
		p, e := filepath.Abs(extConfigPath)
		if e != nil {
			t.Fatalf("filepath err: %v", e)
		}
		extConfig, err := ReadExternalChainConfig(p)
		if err != nil {
			t.Fatalf("could not decode file: %v", err)
		}
		if extConfig.ChainConfig.GetChainID().Cmp(wantInt) != 0 {
			t.Error("got: %v, want: %v", extConfig.ChainConfig.GetChainID(), wantInt)
		}
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
			val, ok := feat.GetString("type")
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
			val, ok := feat.GetString("type")
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
		if n.Cmp(fork.Block) > 0 {
			t.Errorf("unexpected fork block: %v is greater than: %v", fork.Block, n)
		}
		n = fork.Block
	}

	// introduce disorder
	f := &Fork{}
	f.Block = big.NewInt(0).Sub(c.Forks[0].Block, big.NewInt(1))
	c.Forks = append(c.Forks, f) // last fork should be out of order

	c.SortForks()
	n = big.NewInt(0)
	for _, fork := range c.Forks {
		if n.Cmp(fork.Block) > 0 {
			t.Errorf("unexpected fork block: %v is greater than: %v", fork.Block, n)
		}
		n = fork.Block
	}
}

func TestChainConfig_GetSigner(t *testing.T) {
	c := makeTestChainConfig()
	var forkBlocks []*big.Int
	for _, fork := range c.Forks {
		forkBlocks = append(forkBlocks, fork.Block)
	}

	blockMinus := big.NewInt(-2)
	blockPlus := big.NewInt(2)

	for _, block := range forkBlocks {
		bottom := big.NewInt(0).Add(block, blockMinus)
		top := big.NewInt(0).Add(block, blockPlus)
		current := bottom
		for current.Cmp(top) <= 0 {
			signer := c.GetSigner(current)
			feat, _, configured := c.GetFeature(current, "eip155")
			if !configured {
				if !signer.Equal(types.BasicSigner{}) {
					t.Errorf("expected basic signer, block: %v", current)
				}
			} else {
				cid, ok := feat.GetBigInt("chainID")
				if !ok {
					t.Errorf("unexpected missing eip155 chainid, block: %v", current)
				}
				shouldb := types.NewChainIdSigner(cid)
				if !signer.Equal(shouldb) {
					t.Errorf("want: %v, got: %v", shouldb, current)
				}
			}
			current = big.NewInt(0).Add(current, big.NewInt(1))
		}
	}

}

func makeOKSufficientChainConfig(dump *GenesisDump, config *ChainConfig) *SufficientChainConfig {
	// Setup.
	whole := &SufficientChainConfig{}
	whole.Identity = "testID"
	whole.Network = 3
	whole.Name = "testable"
	whole.Consensus = "ethash"
	whole.Genesis = dump
	whole.ChainConfig = config
	whole.Bootstrap = []string{
		"enode://e809c4a2fec7daed400e5e28564e23693b23b2cc5a019b612505631bbe7b9ccf709c1796d2a3d29ef2b045f210caf51e3c4f5b6d3587d43ad5d6397526fa6179@174.112.32.157:30303",
		"enode://6e538e7c1280f0a31ff08b382db5302480f775480b8e68f8febca0ceff81e4b19153c6f8bf60313b93bef2cc34d34e1df41317de0ce613a201d1660a788a03e2@52.206.67.235:30303",
		"enode://5fbfb426fbb46f8b8c1bd3dd140f5b511da558cd37d60844b525909ab82e13a25ee722293c829e52cb65c2305b1637fa9a2ea4d6634a224d5f400bfe244ac0de@162.243.55.45:30303",
		"enode://42d8f29d1db5f4b2947cd5c3d76c6d0d3697e6b9b3430c3d41e46b4bb77655433aeedc25d4b4ea9d8214b6a43008ba67199374a9b53633301bca0cd20c6928ab@104.155.176.151:30303",
		"enode://814920f1ec9510aa9ea1c8f79d8b6e6a462045f09caa2ae4055b0f34f7416fca6facd3dd45f1cf1673c0209e0503f02776b8ff94020e98b6679a0dc561b4eba0@104.154.136.117:30303",
		"enode://72e445f4e89c0f476d404bc40478b0df83a5b500d2d2e850e08eb1af0cd464ab86db6160d0fde64bd77d5f0d33507ae19035671b3c74fec126d6e28787669740@104.198.71.200:30303",
		"enode://5cd218959f8263bc3721d7789070806b0adff1a0ed3f95ec886fb469f9362c7507e3b32b256550b9a7964a23a938e8d42d45a0c34b332bfebc54b29081e83b93@35.187.57.94:30303",
		"enode://39abab9d2a41f53298c0c9dc6bbca57b0840c3ba9dccf42aa27316addc1b7e56ade32a0a9f7f52d6c5db4fe74d8824bcedfeaecf1a4e533cacb71cf8100a9442@144.76.238.49:30303",
		"enode://f50e675a34f471af2438b921914b5f06499c7438f3146f6b8936f1faeb50b8a91d0d0c24fb05a66f05865cd58c24da3e664d0def806172ddd0d4c5bdbf37747e@144.76.238.49:30306",
	}
	return whole
}

// TestSufficientChainConfig_IsValid tests against defaulty dumps and chainconfigs.
func TestSufficientChainConfig_IsValid(t *testing.T) {
	dumps := []*GenesisDump{DefaultGenesis, TestNetGenesis}
	configs := []*ChainConfig{DefaultConfig, TestConfig}

	for i, dump := range dumps {
		for j, config := range configs {
			// Make sure initial ok config is ok.
			scc := makeOKSufficientChainConfig(dump, config)
			if s, ok := scc.IsValid(); !ok {
				t.Errorf("unexpected notok: %v @ %v/%v", s, i, j)
			}

			// Remove each required field and ensure is NOT ok.
			o1 := scc.Identity
			scc.Identity = ""
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Identity = o1

			o2 := scc.Network
			scc.Network = 0
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Network = o2

			o3 := scc.Consensus
			scc.Consensus = "asdf"
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Consensus = o3

			o4 := scc.Consensus
			scc.Consensus = ""
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Consensus = o4

			o := scc.Genesis
			scc.Genesis = nil
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Genesis = o

			oo := scc.ChainConfig
			scc.ChainConfig = nil
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.ChainConfig = oo

			ooo := scc.Bootstrap
			scc.Bootstrap = nil
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Bootstrap = ooo

			oooo := scc.Genesis.Nonce
			scc.Genesis.Nonce = ""
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Genesis.Nonce = oooo

			ooooo := scc.Genesis.Nonce
			scc.Genesis.Nonce = "0xasdf"
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Genesis.Nonce = ooooo

			oooooo := scc.Genesis.GasLimit
			scc.Genesis.GasLimit = "0xasdf"
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Genesis.GasLimit = oooooo

			ooooooo0 := scc.Genesis.Difficulty
			scc.Genesis.Difficulty = "0xasdf"
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.Genesis.Difficulty = ooooooo0

			ooooooo00 := scc.ChainConfig.Forks
			scc.ChainConfig.Forks = []*Fork{}
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.ChainConfig.Forks = ooooooo00

			ooooooo1 := scc.ChainConfig.Forks
			scc.ChainConfig.Forks = nil
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
			scc.ChainConfig.Forks = ooooooo1

			scc = &SufficientChainConfig{}
			if s, ok := scc.IsValid(); ok {
				t.Errorf("unexpected ok: %v @ %v/%v", s, i, j)
			}
		}
	}
}
