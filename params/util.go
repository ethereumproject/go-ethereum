// Copyright 2015 The go-ethereum Authors
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

package params

import "math/big"

var (
	// Legacy
	TestNetHomesteadBlock = big.NewInt(494000)  // Testnet homestead block
	MainNetHomesteadBlock = big.NewInt(1150000) // Mainnet homestead block
	// Initialize Forks
	DAOFork = Fork{TestNetBlock: big.NewInt(494000),
		MainNetBlock: big.NewInt(1150000),
	}
	DAOFork = Fork{TestNetBlock: nil,
		MainNetBlock:   big.NewInt(1920000),
		ForkBlockExtra: common.FromHex("0x64616f2d686172642d666f726b"),
		ForkExtraRange: big.NewInt(10),
	}
)

type Fork struct {
	// TestNetBlock is the block number where a hard-fork commences on
	// the Ethereum test network. It's enforced nil since it was decided not to do a
	// testnet transition.
	TestNetBlock *big.Int
	// MainNetBlock is the block number where the hard-fork commences on
	// the Ethereum main network.
	MainNetBlock *big.Int
	// ForkBlockExtra is the block header extra-data field to set for a fork
	// point and a number of consecutive blocks to allow fast/light syncers to correctly
	ForkBlockExtra []byte
	// ForkExtraRange is the number of consecutive blocks from the fork point
	// to override the extra-data in to prevent no-fork attacks.
	ForkExtraRange *big.Int
}
