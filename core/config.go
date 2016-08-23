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

var ChainConfigNotFoundErr = errors.New("ChainConfig not found") // general config not found error

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	Forks []Fork

	VmConfig vm.Config `json:"-"`
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return c.IsForkIndex(0, num)
}

func (c *ChainConfig) IsETF(num *big.Int) bool {
	return c.IsForkIndex(1, num)
}

func (c *ChainConfig) IsForkIndex(i int, num *big.Int) bool {
	if len(c.Forks) > 0 {
		fork := c.Forks[i]
		if fork.MainNetBlock == nil || num == nil {
			return false
		}
		return num.Cmp(fork.MainNetBlock) >= num.Cmp(num)
	}
	return false
}

func (c *ChainConfig) Fork(name string) (fork Fork) {
	for i := range c.Forks {
		if c.Forks[i].Name == name {
			fork = c.Forks[i]
		}
	}
	return fork
}
