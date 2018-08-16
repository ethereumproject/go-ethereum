package core

import (
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

var (
	DefaultConfigMainnet             *SufficientChainConfig
	DefaultConfigMorden              *SufficientChainConfig
	TestConfigFrontier               *SufficientChainConfig
	TestConfigHomestead              *SufficientChainConfig
	TestConfigEIP150                 *SufficientChainConfig
	TestConfigFrontierToHomesteadAt5 *SufficientChainConfig
	TestConfigHomesteadToEIP150At5   *SufficientChainConfig
)

func init() {

	var err error

	DefaultConfigMainnet, err = parseExternalChainConfig("/core/config/mainnet.json", assetsOpen)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
	DefaultConfigMorden, err = parseExternalChainConfig("/core/config/morden.json", assetsOpen)
	if err != nil {
		glog.Fatal("Error parsing morden defaults from JSON:", err)
	}

	// Test configs for tests/BlockchainTests suite (JSON tests)
	TestConfigFrontier, err = parseExternalChainConfig("/core/config/test_frontier.json", assetsOpen)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
	TestConfigHomestead, err = parseExternalChainConfig("/core/config/test_homestead.json", assetsOpen)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
	TestConfigEIP150, err = parseExternalChainConfig("/core/config/test_eip150.json", assetsOpen)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
	TestConfigFrontierToHomesteadAt5, err = parseExternalChainConfig("/core/config/test_frontier_to_homestead_at_5.json", assetsOpen)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
	TestConfigHomesteadToEIP150At5, err = parseExternalChainConfig("/core/config/test_homestead_to_eip150_at_5.json", assetsOpen)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
}
