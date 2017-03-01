// Copyright 2016 The go-ethereum Authors
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

package core

import (
	"errors"
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/params"
)

var (
	ChainConfigNotFoundErr     = errors.New("ChainConfig not found")
	ChainConfigForkNotFoundErr = errors.New("ChainConfig fork not found")

	ErrHashKnownBad  = errors.New("known bad hash")
	ErrHashKnownFork = validateError("known fork hash mismatch")
)

// DefaultConfig is the Ethereum Classic standard setup.
var DefaultConfig = &ChainConfig{
	Forks: []*Fork{
		{
			Name:         "Homestead",
			Block:        big.NewInt(1150000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     &params.GasTableHomestead,
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
			GasTable:     &params.GasTableHomesteadGasRepriceFork,
		}, {
			Name:         "Diehard",
			Block:        big.NewInt(3000000),
			Length:       big.NewInt(2000000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     &params.GasTableDiehardFork,
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
	ChainId: params.ChainId,
}

// TestConfig is the semi-official setup for testing purposes.
// TODO(pascaldekloe): drop with issue #131
var TestConfig = &ChainConfig{
	Forks: []*Fork{
		{
			Name:         "Homestead",
			Block:        big.NewInt(494000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     &params.GasTableHomestead,
		},
		{
			Name:         "GasReprice",
			Block:        big.NewInt(1783000),
			NetworkSplit: false,
			Support:      true,
			GasTable:     &params.GasTableHomesteadGasRepriceFork,
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
			GasTable:     &params.GasTableDiehardFork,
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
	ChainId: params.TestnetChainId,
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	// Forks holds fork block requirements. See ErrHashKnownFork.
	Forks []*Fork `json:"forks"`

	// BadHashes holds well known blocks with consensus issues. See ErrHashKnownBad.
	BadHashes []*BadHash `json:"bad_hashes"`

	ChainId *big.Int `json:"chain_id"`
}

type BadHash struct {
	Block *big.Int
	Hash  common.Hash
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	if c.Fork("Homestead").Block == nil || num == nil {
		return false
	}
	return num.Cmp(c.Fork("Homestead").Block) >= 0
}

// IsDiehard returns whether num is greater than or equal to the Diehard block, but less than explosion.
func (c *ChainConfig) IsDiehard(num *big.Int) bool {
	fork := c.Fork("Diehard")
	if fork.Block == nil || num == nil {
		return false
	}
	return num.Cmp(fork.Block) >= 0
}

// IsExplosion returns whether num is either equal to the explosion block or greater.
func (c *ChainConfig) IsExplosion(num *big.Int) bool {
	fork := c.Fork("Diehard")
	if fork.Block == nil || fork.Length == nil || num == nil {
		return false
	}
	block := big.NewInt(0).Add(fork.Block, fork.Length)
	return num.Cmp(block) >= 0
}

func (c *ChainConfig) Fork(name string) *Fork {
	for i := range c.Forks {
		if c.Forks[i].Name == name {
			return c.Forks[i]
		}
	}
	return &Fork{}
}

func (c *ChainConfig) HeaderCheck(h *types.Header) error {
	for _, fork := range c.Forks {
		if fork.Block.Cmp(h.Number) != 0 {
			continue
		}
		if common.EmptyHash(fork.RequiredHash) {
			continue
		}
		if fork.RequiredHash != h.Hash() {
			return ErrHashKnownFork
		}
	}

	for _, bad := range c.BadHashes {
		if bad.Block.Cmp(h.Number) != 0 {
			continue
		}
		if bad.Hash == h.Hash() {
			return ErrHashKnownBad
		}
	}

	return nil
}

func (c *ChainConfig) GetSigner(blockNumber *big.Int) types.Signer {
	if c.IsDiehard(blockNumber) {
		return types.NewChainIdSigner(c.ChainId)
	}
	return types.BasicSigner{}
}

// GasTable returns the gas table corresponding to the current fork
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) params.GasTable {
	var gasTable = params.GasTableHomestead
	//TODO avoid loop, remember current fork
	for i := range c.Forks {
		fork := c.Forks[i]
		if fork.Block.Cmp(num) <= 0 {
			if fork.GasTable != nil {
				gasTable = *fork.GasTable
			}
		}
	}
	return gasTable
}

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
