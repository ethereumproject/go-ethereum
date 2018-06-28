package params

import (
	"io"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"path/filepath"

	"reflect"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/types"
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
	config := DefaultConfigMainnet.ChainConfig

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
	config := DefaultConfigMainnet.ChainConfig

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
	config := DefaultConfigMainnet.ChainConfig

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

func getDefaultChainConfigSorted() *ChainConfig {
	return DefaultConfigMainnet.ChainConfig.SortForks()
}

// Unit-y tests.

func TestChainConfig_HasFeature(t *testing.T) {
	c := DefaultConfigMorden.ChainConfig.SortForks()
	for _, id := range allAvailableTestnetConfigKeys {
		if _, _, ok := c.HasFeature(id); !ok {
			t.Errorf("feature not found: %v", id)
		}
	}
	c = getDefaultChainConfigSorted()
	for _, id := range allAvailableDefaultConfigKeys {
		if _, _, ok := c.HasFeature(id); !ok {
			t.Errorf("feature not found: %v", id)
		}
	}

	// never gets unavailable keys
	c = DefaultConfigMorden.ChainConfig.SortForks()
	for _, id := range unavailableConfigKeys {
		if _, _, ok := c.HasFeature(id); ok {
			t.Errorf("nonexisting feature found: %v", id)
		}
	}
	c = getDefaultChainConfigSorted()
	for _, id := range unavailableConfigKeys {
		if _, _, ok := c.HasFeature(id); ok {
			t.Errorf("nonexisting feature found: %v", id)
		}
	}
}

// TestChainConfig_GetFeature should be able to get all features described in DefaultChainConfigMainnet.
func TestChainConfig_GetFeature(t *testing.T) {
	c := getDefaultChainConfigSorted()
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

var allAvailableDefaultConfigKeys = []string{
	"difficulty",
	"gastable",
	"eip155",
}
var allAvailableTestnetConfigKeys = []string{
	"difficulty",
	"gastable",
	"eip155",
	"reward",
}
var unavailableConfigKeys = []string{
	"foo",
	"bar",
	"monkey",
}

// veryHighBlock is a block in the far distant future (so far, in fact, that it will never actually exist)
// Used to test cumulative aggregation functions, ie "eventually".
var veryHighBlock *big.Int = big.NewInt(250000000)

// TestChainConfig_EventuallyGetAllPossibleFeatures should aggregate all available features from previous branches
func TestChainConfig_GetFeature2_EventuallyGetAllPossibleFeatures(t *testing.T) {
	c := getDefaultChainConfigSorted()
	for _, id := range allAvailableDefaultConfigKeys {
		if _, _, ok := c.GetFeature(veryHighBlock, id); !ok {
			t.Errorf("could not get feature with id: %v, at block: %v", id, big.NewInt(5000000))
		}
	}
}

// TestChainConfig_NeverGetNonexistantFeatures should never eventually collect features that don't exist
func TestChainConfig_GetFeature3_NeverGetNonexistantFeatures(t *testing.T) {
	c := getDefaultChainConfigSorted()
	for _, id := range unavailableConfigKeys {
		if feat, _, ok := c.GetFeature(veryHighBlock, id); ok {
			t.Errorf("found unexpected feature: %v, for name: %v, at block: %v", feat, id, big.NewInt(5000000))
		}
	}
}

func TestChainConfig_GetFeature4_WorkForHighNumbers(t *testing.T) {
	c := getDefaultChainConfigSorted()
	ultraHighBlock := big.NewInt(99999999999999999)
	if _, _, ok := c.GetFeature(ultraHighBlock, "difficulty"); !ok {
		t.Errorf("unexpected unfound difficulty feature for far-future block: %v", ultraHighBlock)
	}
}

func TestChainConfig_GetChainID(t *testing.T) {
	// Test default hardcoded configs.
	if DefaultConfigMainnet.ChainConfig.GetChainID().Cmp(DefaultConfigMainnet.ChainConfig.GetChainID()) != 0 {
		t.Errorf("got: %v, want: %v", DefaultConfigMainnet.ChainConfig.GetChainID(), DefaultConfigMainnet.ChainConfig.GetChainID())
	}
	if DefaultConfigMorden.ChainConfig.GetChainID().Cmp(DefaultConfigMorden.ChainConfig.GetChainID()) != 0 {
		t.Errorf("got: %v, want: %v", DefaultConfigMorden.ChainConfig.GetChainID(), DefaultConfigMorden.ChainConfig.GetChainID())
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
		"../params/config/mainnet.json": DefaultConfigMainnet.ChainConfig.GetChainID(),
		"../params/config/morden.json":  DefaultConfigMorden.ChainConfig.GetChainID(),
	}
	for extConfigPath, wantInt := range cases {
		p, e := filepath.Abs(extConfigPath)
		if e != nil {
			t.Fatalf("filepath err: %v", e)
		}
		extConfig, err := ReadExternalChainConfigFromFile(p)
		if err != nil {
			t.Fatalf("could not decode file: %v", err)
		}
		if extConfig.ChainConfig.GetChainID().Cmp(wantInt) != 0 {
			t.Errorf("got: %v, want: %v", extConfig.ChainConfig.GetChainID(), wantInt)
		}
	}
}

// Acceptance-y tests.

// Test GetFeature gets expected feature values from default configuration data...

// TestChainConfig_GetFeature_DefaultEIP155 should get the eip155 feature for (only and above) its default implemented block.
func TestChainConfig_GetFeature5_DefaultEIP155(t *testing.T) {
	c := getDefaultChainConfigSorted()
	var tables = map[*big.Int]*big.Int{
		big.NewInt(0).Sub(DefaultConfigMainnet.ChainConfig.ForkByName("Homestead").Block, big.NewInt(1)): nil,
		DefaultConfigMainnet.ChainConfig.ForkByName("Homestead").Block:                                   nil,
		big.NewInt(0).Add(DefaultConfigMainnet.ChainConfig.ForkByName("Homestead").Block, big.NewInt(1)): nil,

		big.NewInt(0).Sub(DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice").Block, big.NewInt(1)): nil,
		DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice").Block:                                   nil,
		big.NewInt(0).Add(DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice").Block, big.NewInt(1)): nil,

		big.NewInt(0).Sub(DefaultConfigMainnet.ChainConfig.ForkByName("Diehard").Block, big.NewInt(1)): nil,
		DefaultConfigMainnet.ChainConfig.ForkByName("Diehard").Block:                                   big.NewInt(61),
		big.NewInt(0).Add(DefaultConfigMainnet.ChainConfig.ForkByName("Diehard").Block, big.NewInt(1)): big.NewInt(61),
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
	c := getDefaultChainConfigSorted()
	var tables = map[*big.Int]string{
		big.NewInt(0).Sub(DefaultConfigMainnet.ChainConfig.ForkByName("Homestead").Block, big.NewInt(1)): "",
		DefaultConfigMainnet.ChainConfig.ForkByName("Homestead").Block:                                   "homestead",
		big.NewInt(0).Add(DefaultConfigMainnet.ChainConfig.ForkByName("Homestead").Block, big.NewInt(1)): "homestead",

		big.NewInt(0).Sub(DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice").Block, big.NewInt(1)): "homestead",
		DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice").Block:                                   "eip150",
		big.NewInt(0).Add(DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice").Block, big.NewInt(1)): "eip150",

		big.NewInt(0).Sub(DefaultConfigMainnet.ChainConfig.ForkByName("Diehard").Block, big.NewInt(1)): "eip150",
		DefaultConfigMainnet.ChainConfig.ForkByName("Diehard").Block:                                   "eip160",
		big.NewInt(0).Add(DefaultConfigMainnet.ChainConfig.ForkByName("Diehard").Block, big.NewInt(1)): "eip160",
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
	c := getDefaultChainConfigSorted()
	var tables = map[*big.Int]string{
		big.NewInt(0).Sub(DefaultConfigMainnet.ChainConfig.ForkByName("Homestead").Block, big.NewInt(1)): "",
		DefaultConfigMainnet.ChainConfig.ForkByName("Homestead").Block:                                   "homestead",
		big.NewInt(0).Add(DefaultConfigMainnet.ChainConfig.ForkByName("Homestead").Block, big.NewInt(1)): "homestead",

		big.NewInt(0).Sub(DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice").Block, big.NewInt(1)): "homestead",
		DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice").Block:                                   "homestead",
		big.NewInt(0).Add(DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice").Block, big.NewInt(1)): "homestead",

		big.NewInt(0).Sub(DefaultConfigMainnet.ChainConfig.ForkByName("Diehard").Block, big.NewInt(1)): "homestead",
		DefaultConfigMainnet.ChainConfig.ForkByName("Diehard").Block:                                   "ecip1010",
		big.NewInt(0).Add(DefaultConfigMainnet.ChainConfig.ForkByName("Diehard").Block, big.NewInt(1)): "ecip1010",
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
	c := getDefaultChainConfigSorted()
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

func TestChainConfigGetSet(t *testing.T) {
	c := getDefaultChainConfigSorted()
	set := SetCacheChainConfig(&SufficientChainConfig{ChainConfig: c})

	if set == nil {
		t.Fatal("set returned nil")
	}

	got := GetCacheChainConfig()
	if got == nil {
		t.Fatal("get returned nil")
	}

	// create new "checkpoint" fork for testing
	checkpoint := &Fork{
		Name:         "checkpoint",
		Block:        big.NewInt(1930000),
		RequiredHash: common.HexToHash("0xabc65e3a8c0b35089c1d1195081fe7489b528a84b22199c916180db8b28ad123"),
	}
	c.Forks = append(c.Forks, checkpoint)

	got.ChainConfig.Forks = c.Forks
	didSet := SetCacheChainConfig(got)
	if !reflect.DeepEqual(got, didSet) {
		t.Errorf("got: %v, want: %v", didSet, got)
	}

	if f := set.ChainConfig.ForkByName("checkpoint"); f == nil {
		t.Errorf("got: %v, want: %v", f, checkpoint)
	}
}

func TestChainConfig_GetLastRequiredHashFork(t *testing.T) {
	c := getDefaultChainConfigSorted()

	daoFork := c.ForkByName("The DAO Hard Fork")
	got, want := c.GetLatestRequiredHashFork(big.NewInt(1920000)), daoFork

	// sanity check
	if want == nil {
		t.Fatal("nil want hard fork")
	}

	if got == nil {
		t.Fatalf("got: %v, want: %s", got, want.Name)
	}
	if got.RequiredHash.Hex() != "0x94365e3a8c0b35089c1d1195081fe7489b528a84b22199c916180db8b28ade7f" {
		t.Errorf("got: %v, want: %s", got, got.RequiredHash.Hex())
	}
	if got.Block.Cmp(big.NewInt(1920000)) != 0 {
		t.Errorf("got: %d, want: %d", got.Block, 1920000)
	}

	// create new "checkpoint" fork for testing
	checkpoint := &Fork{
		Name:         "checkpoint",
		Block:        big.NewInt(1930000),
		RequiredHash: common.HexToHash("0xabc65e3a8c0b35089c1d1195081fe7489b528a84b22199c916180db8b28ad123"),
	}
	c.Forks = append(c.Forks, checkpoint)

	// Noting that config forks do not have to be sorted for this function to work.
	//c.SortForks()

	got, want = c.GetLatestRequiredHashFork(big.NewInt(1930000)), checkpoint
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// should use dao fork, not checkpoint since block n has not reached checkpoint
	got, want = c.GetLatestRequiredHashFork(big.NewInt(1920001)), daoFork
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestChainConfig_GetSigner(t *testing.T) {
	c := getDefaultChainConfigSorted()
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

func TestResolvePath(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{
			args: []string{"./a/b/config.csv", "."},
			want: filepath.Clean("a/b/config.csv"),
		},
		{
			args: []string{"./a/b/config.csv", ""},
			want: filepath.Clean("a/b/config.csv"),
		},
		{
			args: []string{"./a/b/config.csv", "./a/b/config.json"},
			want: filepath.Clean("a/b/a/b/config.csv"),
		},
		{
			args: []string{"config.csv", "a/b"},
			want: filepath.Clean("a/config.csv"), // since resolvePath expects b|config.csv to be adjacent (neighboring filepaths), ie. 'b' should be a file
		},
		{
			args: []string{"config.csv", "a/b/config.json"},
			want: filepath.Clean("a/b/config.csv"),
		},
		{
			args: []string{"test.txt", "some/dir/conf.json"},
			want: filepath.Clean("some/dir/test.txt"),
		},
		{
			args: []string{"test.txt", "./some/dir/conf.json"},
			want: filepath.Clean("some/dir/test.txt"),
		},
		{
			args: []string{"./test.txt", "some/dir/conf.json"},
			want: filepath.Clean("some/dir/test.txt"),
		},
		{
			args: []string{"./test.txt", "./some/dir/conf.json"},
			want: filepath.Clean("some/dir/test.txt"),
		},
		{
			args: []string{"../test.txt", "some/dir/conf.json"},
			want: filepath.Clean("some/test.txt"),
		},
		{
			args: []string{"../test.txt", "/some/dir/conf.json"},
			want: filepath.Clean("/some/test.txt"),
		},
		{
			args: []string{"../test.txt", "some/dir/.././conf.json"},
			want: filepath.Clean("test.txt"),
		},
		{
			args: []string{"../test.txt", "conf.json"},
			want: filepath.Clean("../test.txt"),
		},
		{
			args: []string{"../../../../a/b/c/d/test.txt", "conf.json"},
			want: filepath.Clean("../../../../a/b/c/d/test.txt"),
		},
		{
			args: []string{"../../../../a/b/c/d/test.txt", "a/b/c/d/conf.json"},
			want: filepath.Clean("a/b/c/d/test.txt"),
		},
	}
	for _, c := range cases {
		t.Run(c.args[0]+"+"+c.args[1], func(t *testing.T) {
			if got := resolvePath(c.args[0], c.args[1]); got != c.want {
				t.Errorf("got: %v, want: %v", got, c.want)
			}
		})
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
	dumps := []*GenesisDump{DefaultConfigMainnet.Genesis, DefaultConfigMorden.Genesis}
	configs := []*ChainConfig{DefaultConfigMainnet.ChainConfig, DefaultConfigMorden.ChainConfig}

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
			scc.Bootstrap = []string{}
			if s, ok := scc.IsValid(); !ok {
				t.Errorf("unexpected notok: %v @ %v/%v", s, i, j)
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

func TestGenesisAllocationError(t *testing.T) {
	_, err := parseExternalChainConfig("testdata/test.json", func(path string) (io.ReadCloser, error) { return os.Open(path) })
	if err == nil {
		t.Error("expected error, got nil")
	}
	want := "\"alloc\" values already set"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("invalid error message: want: '%s' got: '%s'", want, err.Error())
	}
}
