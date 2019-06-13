package core

import (
	"github.com/eth-classic/go-ethereum/logger/glog"
)

var (
	DefaultConfigMainnet *SufficientChainConfig
	DefaultConfigMorden  *SufficientChainConfig
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
}
