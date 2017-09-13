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

	DefaultSufficientConfigMainnet *SufficientChainConfig
	DefaultSufficientConfigMorden  *SufficientChainConfig

	// TODO: refactor ChainConfigs and Genesis's to use attributes of SufficentChainConfigs instead of own vars
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

	DefaultSufficientConfigMainnet, err = parseExternalChainConfig(mainnetJSONData)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
	DefaultSufficientConfigMorden, err = parseExternalChainConfig(mordenJSONData)
	if err != nil {
		glog.Fatal("Error parsing morden defaults from JSON:", err)
	}

	DefaultChainConfigID = DefaultSufficientConfigMainnet.Identity
	DefaultTestnetChainConfigID = DefaultSufficientConfigMorden.Identity

	DefaultChainConfigName = DefaultSufficientConfigMainnet.Name
	DefaultTestnetChainConfigName = DefaultSufficientConfigMorden.Name

	DefaultChainConfigChainID = DefaultSufficientConfigMainnet.ChainConfig.GetChainID()
	DefaultTestnetChainConfigChainID = DefaultSufficientConfigMorden.ChainConfig.GetChainID()

	DefaultConfig = DefaultSufficientConfigMainnet.ChainConfig
	TestConfig = DefaultSufficientConfigMorden.ChainConfig

	DefaultGenesis = DefaultSufficientConfigMainnet.Genesis
	TestNetGenesis = DefaultSufficientConfigMorden.Genesis
}
