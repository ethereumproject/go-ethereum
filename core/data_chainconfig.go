// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// This file contains configuration literals.

package core

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/vm"
)

// DefaultConfig is the Ethereum Classic standard setup.
var DefaultConfig = &ChainConfig{
	Forks: []*Fork{
		{
			Name:         "Homestead",
			Block:        big.NewInt(1150000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     HomeSteadGasTable,
		}, {
			Name:         "ETF",
			Block:        big.NewInt(1920000),
			NetworkSplit: true,
			Support:      false,
			RequiredHash: common.HexToHash("94365e3a8c0b35089c1d1195081fe7489b528a84b22199c916180db8b28ade7f"),
		}, {
			Name:         "GasReprice",
			Block:        big.NewInt(2500000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     GasRepriceGasTable,
		}, {
			Name:         "Diehard",
			Block:        big.NewInt(3000000),
			Length:       big.NewInt(2000000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     DiehardGasTable,
		},
	},
	BadHashes: []*BadHash{
		{
			// consensus issue that occurred on the Frontier network at block 116,522, mined on 2015-08-20 at 14:59:16+02:00
			// https://blog.ethereum.org/2015/08/20/security-alert-consensus-issue
			Block: big.NewInt(116522),
			Hash:  common.HexToHash("05bef30ef572270f654746da22639a7a0c97dd97a7050b9e252391996aaeb689"),
		},
	},
	ChainId: big.NewInt(61),
}

// TestConfig is the semi-official setup for testing purposes.
var TestConfig = &ChainConfig{
	Forks: []*Fork{
		{
			Name:         "Homestead",
			Block:        big.NewInt(494000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     HomeSteadGasTable,
		},
		{
			Name:         "GasReprice",
			Block:        big.NewInt(1783000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     GasRepriceGasTable,
		},
		{
			Name:         "ETF",
			Block:        big.NewInt(1885000),
			NetworkSplit: true,
			Support:      false,
			RequiredHash: common.HexToHash("2206f94b53bd0a4d2b828b6b1a63e576de7abc1c106aafbfc91d9a60f13cb740"),
		},
		{
			Name:         "Diehard",
			Block:        big.NewInt(1915000),
			Length:       big.NewInt(1500000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     DiehardGasTable,
		},
	},
	BadHashes: []*BadHash{
		{
			// consensus issue at Testnet #383792
			// http://ethereum.stackexchange.com/questions/10183/upgraded-to-geth-1-5-0-bad-block-383792
			Block: big.NewInt(383792),
			Hash:  common.HexToHash("9690db54968a760704d99b8118bf79d565711669cefad24b51b5b1013d827808"),
		},
		{
			// chain followed by non-diehard testnet
			Block: big.NewInt(1915277),
			Hash:  common.HexToHash("3bef9997340acebc85b84948d849ceeff74384ddf512a20676d424e972a3c3c4"),
		},
	},
	ChainId: big.NewInt(62),
}
