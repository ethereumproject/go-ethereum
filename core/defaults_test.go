package core

import (
	"testing"
	"math/big"
)

// Implement chain config defaults tests, ensure all existing
// features are correctly in place (as to not break backwards compatibility).
// New chain configurations should be accompanied by relevant associated tests.
// Tests are designed to ensure that compiled JSON default configs
// are up-to-date and accurate.

func TestDefaultChainConfigurationVariablesExist(t *testing.T) {

	if DefaultChainConfigID != "mainnet" {
		t.Errorf("got: %v, want: %v", DefaultChainConfigID, "mainnet")
	}
	if DefaultTestnetChainConfigID != "morden" {
		t.Errorf("got: %v, want: %v", DefaultTestnetChainConfigID, "mainnet")
	}

	if DefaultChainConfigName != "Ethereum Classic Mainnet" {
		t.Errorf("got: %v, want: %v", DefaultChainConfigName, "Ethereum Classic Mainnet")
	}
	if DefaultTestnetChainConfigName != "Morden Testnet" {
		t.Errorf("got: %v, want: %v", DefaultTestnetChainConfigName, "Morden Testnet")
	}

	if DefaultChainConfigChainID.Cmp(big.NewInt(61)) != 0 {
		t.Errorf("got: %v, want: %v", DefaultChainConfigChainID, big.NewInt(61))
	}
	if DefaultTestnetChainConfigChainID.Cmp(big.NewInt(62)) != 0 {
		t.Errorf("got: %v, want: %v", DefaultTestnetChainConfigID, big.NewInt(62))
	}

	// Test forks existence and block numbers
	// Homestead
	if fork := DefaultConfig.ForkByName("Homestead"); fork.Block.Cmp(big.NewInt(1150000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := TestConfig.ForkByName("Homestead"); fork.Block.Cmp(big.NewInt(494000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	// The DAO Hard Fork
	if fork := DefaultConfig.ForkByName("The DAO Hard Fork"); fork.Block.Cmp(big.NewInt(1920000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := TestConfig.ForkByName("The DAO Hard Fork"); fork.Block.Cmp(big.NewInt(1885000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	// GasReprice
	if fork := DefaultConfig.ForkByName("GasReprice"); fork.Block.Cmp(big.NewInt(2500000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := TestConfig.ForkByName("GasReprice"); fork.Block.Cmp(big.NewInt(1783000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	// Diehard
	if fork := DefaultConfig.ForkByName("Diehard"); fork.Block.Cmp(big.NewInt(3000000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := TestConfig.ForkByName("Diehard"); fork.Block.Cmp(big.NewInt(1915000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	// Gotham
	if fork := DefaultConfig.ForkByName("Gotham"); fork.Block.Cmp(big.NewInt(5000000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}
	if fork := TestConfig.ForkByName("Gotham"); fork.Block.Cmp(big.NewInt(2000000)) != 0 {
		t.Errorf("Unexpected fork: %v", fork)
	}

	// TODO: check each individual feature

	// Number of bootstrap nodes
	if l := len(DefaultSufficientConfigMainnet.ParsedBootstrap); l != 10 {
		t.Errorf("got: %v, want: %v", l, 10)
	}
	if l := len(DefaultSufficientConfigMorden.ParsedBootstrap); l != 7 {
		t.Errorf("got: %v, want: %v", l, 7)
	}

	// Config validity checks.
	if s, ok := DefaultSufficientConfigMainnet.IsValid(); !ok {
		t.Errorf("Unexpected invalid default chain config: %s", s)
	}
	if s, ok := DefaultSufficientConfigMorden.IsValid(); !ok {
		t.Errorf("Unexpected invalid default chain config: %s", s)
	}

}