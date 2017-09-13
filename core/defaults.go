package core

import (
	"math/big"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/core/assets"
)

var (
	DefaultChainConfigID string
	DefaultTestnetChainConfigID string

	DefaultChainConfigName string
	DefaultTestnetChainConfigName string

	DefaultChainConfigChainID *big.Int
	DefaultTestnetChainConfigChainID *big.Int

	DefaultConfig *ChainConfig
	TestConfig *ChainConfig

	TestNetGenesis *GenesisDump
	DefaultGenesis *GenesisDump
)

func init() {

	var err error

	mainnetJSONData, err := assets.DEFAULTS.Open("/core/config/mainnet.json")
	if err != nil {
		glog.Fatal("Error opening mainnet default JSON:", err)
	}
	mordenJSONData, err := assets.DEFAULTS.Open("/core/config/morden.json")
	if err != nil {
		glog.Fatal("Error opening morden default JSON:", err)
	}

	mainnetConfigDefaults, err := parseExternalChainConfig(mainnetJSONData)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
	mordenConfigDefaults, err := parseExternalChainConfig(mordenJSONData)
	if err != nil {
		glog.Fatal("Error parsing morden defaults from JSON:", err)
	}

	DefaultChainConfigID = mainnetConfigDefaults.Identity
	DefaultTestnetChainConfigID = mordenConfigDefaults.Identity

	DefaultChainConfigName = mainnetConfigDefaults.Name
	DefaultTestnetChainConfigName = mordenConfigDefaults.Name

	DefaultChainConfigChainID = mainnetConfigDefaults.ChainConfig.GetChainID()
	DefaultTestnetChainConfigChainID = mordenConfigDefaults.ChainConfig.GetChainID()

	DefaultConfig = mainnetConfigDefaults.ChainConfig
	TestConfig = mordenConfigDefaults.ChainConfig

	DefaultGenesis = mainnetConfigDefaults.Genesis
	TestNetGenesis = mordenConfigDefaults.Genesis
}
