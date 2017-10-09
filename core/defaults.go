package core

import (
	"github.com/ethereumproject/go-ethereum/core/assets"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

var (
	DefaultConfigMainnet *SufficientChainConfig
	DefaultConfigMorden  *SufficientChainConfig
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

	DefaultConfigMainnet, err = parseExternalChainConfig(mainnetJSONData)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
	DefaultConfigMorden, err = parseExternalChainConfig(mordenJSONData)
	if err != nil {
		glog.Fatal("Error parsing morden defaults from JSON:", err)
	}
}
