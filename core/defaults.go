package core

import (
	"fmt"
	"io"

	"github.com/ethereumproject/go-ethereum/core/assets"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

var (
	DefaultConfigMainnet *SufficientChainConfig
	DefaultConfigMorden  *SufficientChainConfig
)

func init() {

	var err error

	open := func(path string) (io.ReadCloser, error) {
		file, err := assets.DEFAULTS.Open(path)
		if err != nil {
			err = fmt.Errorf("Error opening '%s' default JSON: %v", path, err)
		}
		return file, err
	}
	DefaultConfigMainnet, err = parseExternalChainConfig("/core/config/mainnet.json", open)
	if err != nil {
		glog.Fatal("Error parsing mainnet defaults from JSON:", err)
	}
	DefaultConfigMorden, err = parseExternalChainConfig("/core/config/morden.json", open)
	if err != nil {
		glog.Fatal("Error parsing morden defaults from JSON:", err)
	}
}
