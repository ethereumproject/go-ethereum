package core

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/params"
)

type Fork struct {
	Name         string
	Support      bool
	NetworkSplit bool
	// Block is the block number where the hard-fork commences on
	// the Ethereum network.
	Block *big.Int
	// Length of fork, if limited
	Length *big.Int
	// SplitHash to derive BadHashes to assist in avoiding sync issues
	// after network split.
	OrigSplitHash string
	ForkSplitHash string
	// ForkBlockExtra is the block header extra-data field to set for a fork
	// point and a number of consecutive blocks to allow fast/light syncers to correctly
	BlockExtra []byte
	// ForkExtraRange is the number of consecutive blocks from the fork point
	// to override the extra-data in to prevent no-fork attacks.
	ExtraRange *big.Int
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
			BlockExtra:   common.FromHex("0x64616f2d686172642d666f726b"),
			ExtraRange:   big.NewInt(10),
			// ETC Block+1
			OrigSplitHash: "94365e3a8c0b35089c1d1195081fe7489b528a84b22199c916180db8b28ade7f",
			// ETF Block+1
			ForkSplitHash: "4985f5ca3d2afbec36529aa96f74de3cc10a2a4a6c44f2157a57d2c6059a11bb",
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
			Name:          "ETF",
			Block:         big.NewInt(1885000),
			NetworkSplit:  true,
			Support:       false,
			OrigSplitHash: "2206f94b53bd0a4d2b828b6b1a63e576de7abc1c106aafbfc91d9a60f13cb740",
		},
	}
}
