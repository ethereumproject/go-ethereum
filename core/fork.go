package core

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/params"
)

type Fork struct {
	Name string
	// For user notification only
	Support      bool
	NetworkSplit bool
	// Block is the block number where the hard-fork commences on
	// the Ethereum network.
	Block *big.Int
	// Length of fork, if limited
	Length *big.Int
	// RequiredHash to assist in avoiding sync issues
	// after network split.
	RequiredHash common.Hash
	// Gas Price table
	GasTable *params.GasTable
	// TODO Derive Oracle contracts from fork struct (Version, Registrar, Release)
}

// TODO Migrate hardcoded fork config into a json file
func LoadForks() []*Fork {
	return []*Fork{
		&Fork{
			Name:         "Homestead",
			Block:        big.NewInt(1150000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     &params.GasTableHomestead,
		},
		&Fork{
			Name:         "ETF",
			Block:        big.NewInt(1920000),
			NetworkSplit: true,
			Support:      false,
			RequiredHash: common.HexToHash("94365e3a8c0b35089c1d1195081fe7489b528a84b22199c916180db8b28ade7f"),
		},
		&Fork{
			Name:         "GasReprice",
			Block:        big.NewInt(2500000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     &params.GasTableHomesteadGasRepriceFork,
		},
		&Fork{
			Name:         "Diehard",
			Block:        big.NewInt(3000000),
			Length:       big.NewInt(2000000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     &params.GasTableDiehardFork,
		},
	}
}

func LoadTestnet() []*Fork {
	return []*Fork{
		&Fork{
			Name:         "Homestead",
			Block:        big.NewInt(494000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     &params.GasTableHomestead,
		},
		&Fork{
			Name:         "GasReprice",
			Block:        big.NewInt(1783000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     &params.GasTableHomesteadGasRepriceFork,
		},
		&Fork{
			Name:         "ETF",
			Block:        big.NewInt(1885000),
			NetworkSplit: true,
			Support:      false,
			RequiredHash: common.HexToHash("2206f94b53bd0a4d2b828b6b1a63e576de7abc1c106aafbfc91d9a60f13cb740"),
		},
	}
}
