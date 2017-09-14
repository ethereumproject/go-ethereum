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

	// TODO: check each individual feature

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