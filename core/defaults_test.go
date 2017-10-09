package core

import (
	"math/big"
	"testing"
)

// Implement chain config defaults tests, ensure all existing
// features are correctly in place (as to not break backwards compatibility).
// New chain configurations should be accompanied by relevant associated tests.
// Tests are designed to ensure that compiled JSON default configs
// are up-to-date and accurate.

func TestDefaultChainConfigurationVariablesExist(t *testing.T) {

	if DefaultConfigMainnet.Identity != "mainnet" {
		t.Errorf("got: %v, want: %v", DefaultConfigMainnet.Identity, "mainnet")
	}
	if DefaultConfigMorden.Identity != "morden" {
		t.Errorf("got: %v, want: %v", DefaultConfigMorden.Identity, "mainnet")
	}

	if DefaultConfigMainnet.Name != "Ethereum Classic Mainnet" {
		t.Errorf("got: %v, want: %v", DefaultConfigMainnet.Name, "Ethereum Classic Mainnet")
	}
	if DefaultConfigMorden.Name != "Morden Testnet" {
		t.Errorf("got: %v, want: %v", DefaultConfigMorden.Name, "Morden Testnet")
	}

	if DefaultConfigMainnet.ChainConfig.GetChainID().Cmp(big.NewInt(61)) != 0 {
		t.Errorf("got: %v, want: %v", DefaultConfigMainnet.ChainConfig.GetChainID(), big.NewInt(61))
	}
	if DefaultConfigMorden.ChainConfig.GetChainID().Cmp(big.NewInt(62)) != 0 {
		t.Errorf("got: %v, want: %v", DefaultConfigMorden.ChainConfig.GetChainID(), big.NewInt(62))
	}

	// Test forks existence and block numbers
	// Homestead
	if fork := DefaultConfigMainnet.ChainConfig.ForkByName("Homestead"); fork.Block.Cmp(big.NewInt(1150000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := DefaultConfigMorden.ChainConfig.ForkByName("Homestead"); fork.Block.Cmp(big.NewInt(494000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	// The DAO Hard Fork
	if fork := DefaultConfigMainnet.ChainConfig.ForkByName("The DAO Hard Fork"); fork.Block.Cmp(big.NewInt(1920000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := DefaultConfigMorden.ChainConfig.ForkByName("The DAO Hard Fork"); fork.Block.Cmp(big.NewInt(1885000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	// GasReprice
	if fork := DefaultConfigMainnet.ChainConfig.ForkByName("GasReprice"); fork.Block.Cmp(big.NewInt(2500000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := DefaultConfigMorden.ChainConfig.ForkByName("GasReprice"); fork.Block.Cmp(big.NewInt(1783000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	// Diehard
	if fork := DefaultConfigMainnet.ChainConfig.ForkByName("Diehard"); fork.Block.Cmp(big.NewInt(3000000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := DefaultConfigMorden.ChainConfig.ForkByName("Diehard"); fork.Block.Cmp(big.NewInt(1915000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	// Gotham
	if fork := DefaultConfigMainnet.ChainConfig.ForkByName("Gotham"); fork.Block.Cmp(big.NewInt(5000000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := DefaultConfigMorden.ChainConfig.ForkByName("Gotham"); fork.Block.Cmp(big.NewInt(2000000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}

	checks := []struct {
		Config   *SufficientChainConfig
		Block    *big.Int
		Name     string
		Features []*ForkFeature
	}{
		{
			Config: DefaultConfigMorden,
			Block:  big.NewInt(494000),
			Name:   "Homestead",
			Features: []*ForkFeature{
				{
					ID: "difficulty",
					Options: ChainFeatureConfigOptions{
						"type": "homestead",
					},
				},
				{
					ID: "gastable",
					Options: ChainFeatureConfigOptions{
						"type": "homestead",
					},
				},
			},
		},
		{
			Config: DefaultConfigMainnet,
			Block:  big.NewInt(1150000),
			Name:   "Homestead",
			Features: []*ForkFeature{
				{
					ID: "difficulty",
					Options: ChainFeatureConfigOptions{
						"type": "homestead",
					},
				},
				{
					ID: "gastable",
					Options: ChainFeatureConfigOptions{
						"type": "homestead",
					},
				},
			},
		},
		{
			Config: DefaultConfigMorden,
			Block:  big.NewInt(1885000),
			Name:   "The DAO Hard Fork",
		},
		{
			Config: DefaultConfigMainnet,
			Block:  big.NewInt(1920000),
			Name:   "The DAO Hard Fork",
		},
		{
			Config: DefaultConfigMorden,
			Block:  big.NewInt(1783000),
			Name:   "GasReprice",
			Features: []*ForkFeature{
				{
					ID: "gastable",
					Options: ChainFeatureConfigOptions{
						"type": "eip150",
					},
				},
			},
		},
		{
			Config: DefaultConfigMainnet,
			Block:  big.NewInt(2500000),
			Name:   "GasReprice",
			Features: []*ForkFeature{
				{
					ID: "gastable",
					Options: ChainFeatureConfigOptions{
						"type": "eip150",
					},
				},
			},
		},
		{
			Config: DefaultConfigMorden,
			Block:  big.NewInt(1915000),
			Name:   "Diehard",
			Features: []*ForkFeature{
				{
					ID: "gastable",
					Options: ChainFeatureConfigOptions{
						"type": "eip160",
					},
				},
				{
					ID: "eip155",
					Options: ChainFeatureConfigOptions{
						"chainID": big.NewInt(62),
					},
				},
				{
					ID: "difficulty",
					Options: ChainFeatureConfigOptions{
						"length": big.NewInt(2000000),
						"type":   "ecip1010",
					},
				},
			},
		},
		{
			Config: DefaultConfigMainnet,
			Block:  big.NewInt(3000000),
			Name:   "Diehard",
			Features: []*ForkFeature{
				{
					ID: "gastable",
					Options: ChainFeatureConfigOptions{
						"type": "eip160",
					},
				},
				{
					ID: "eip155",
					Options: ChainFeatureConfigOptions{
						"chainID": big.NewInt(61),
					},
				},
				{
					ID: "difficulty",
					Options: ChainFeatureConfigOptions{
						"length": big.NewInt(2000000),
						"type":   "ecip1010",
					},
				},
			},
		},
		{
			Config: DefaultConfigMorden,
			Block:  big.NewInt(2000000),
			Name:   "Gotham",
			Features: []*ForkFeature{
				{
					ID: "reward",
					Options: ChainFeatureConfigOptions{
						"era":  big.NewInt(2000000),
						"type": "ecip1017",
					},
				},
			},
		},
		{
			Config: DefaultConfigMainnet,
			Block:  big.NewInt(5000000),
			Name:   "Gotham",
			Features: []*ForkFeature{
				{
					ID: "reward",
					Options: ChainFeatureConfigOptions{
						"era":  big.NewInt(5000000),
						"type": "ecip1017",
					},
				},
			},
		},
	}
	for _, check := range checks {
		// Ensure fork exists at correct block
		if fork := check.Config.ChainConfig.ForkByName(check.Name); fork.Block.Cmp(check.Block) != 0 {
			t.Errorf("got: %v, want: %v", fork.Block, check.Block)
		}
		for _, feat := range check.Features {
			ff, f, ok := check.Config.ChainConfig.GetFeature(check.Block, feat.ID)
			if !ok {
				t.Errorf("unfound fork feat: %s", feat.ID)
			}
			for k, v := range ff.Options {
				switch v.(type) {
				case *big.Int:
					if big.NewInt(feat.Options[k].(int64)).Cmp(big.NewInt(v.(int64))) != 0 {
						t.Errorf("mismatch for feature options: got: %v/%v, want: %v/%v", k, feat.Options[k], k, v)
					}
				case string:
					if feat.Options[k] != v {
						t.Errorf("mismatch for feature options: got: %v/%v, want: %v/%v", k, feat.Options[k], k, v)
					}
				}

			}
			if f.Block.Cmp(check.Block) != 0 {
				t.Errorf("feature fork block wrong: got: %v, want: %v", f.Block, check.Block)
			}
		}
	}

	// Number of bootstrap nodes
	if l := len(DefaultConfigMainnet.ParsedBootstrap); l != 10 {
		t.Errorf("got: %v, want: %v", l, 10)
	}
	if l := len(DefaultConfigMorden.ParsedBootstrap); l != 7 {
		t.Errorf("got: %v, want: %v", l, 7)
	}

	// Config validity checks.
	if s, ok := DefaultConfigMainnet.IsValid(); !ok {
		t.Errorf("Unexpected invalid default chain config: %s", s)
	}
	if s, ok := DefaultConfigMorden.IsValid(); !ok {
		t.Errorf("Unexpected invalid default chain config: %s", s)
	}

}
