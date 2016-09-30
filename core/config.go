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

	"github.com/ethereumproject/go-ethereum/core/vm"
)

var (
	ChainConfigNotFoundErr     = errors.New("ChainConfig not found")
	ChainConfigForkNotFoundErr = errors.New("ChainConfig fork not found")
)

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	VmConfig vm.Config `json:"-"`
	// ForkConfig fork.Config
	Forks []*Fork `json:"forks"`
}

func NewChainConfig() *ChainConfig {
	return &ChainConfig{
		Forks: LoadForks(),
	}
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	if c.Fork("Homestead").Block == nil || num == nil {
		return false
	}
	return num.Cmp(c.Fork("Homestead").Block) >= 0
}

// IsGotham returns whether num is greater than or equal to the gotham block, but less than explosion.
func (c *ChainConfig) IsGotham(num *big.Int) bool {
	if c.Fork("Gotham").Block == nil || num == nil {
		return false
	}
	return num.Cmp(c.Fork("Gotham").Block) >= 0 && num.Cmp(c.Fork("Explosion").Block) < 0
}

// IsExplosion returns whether num is either equal to the explosion block or greater.
func (c *ChainConfig) IsExplosion(num *big.Int) bool {
	if c.Fork("Explosion").Block == nil || num == nil {
		return false
	}
	return num.Cmp(c.Fork("Explosion").Block) >= 0
}

func (c *ChainConfig) Fork(name string) *Fork {
	for i := range c.Forks {
		if c.Forks[i].Name == name {
			return c.Forks[i]
		}
	}
	return &Fork{}
}

func (c *ChainConfig) LoadForkConfig() {
	c.Forks = LoadForks()
}
