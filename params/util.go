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

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/fork"
)

var (
	// Init
	HomesteadFork = fork.Fork{Name: "Homestead",
		NetworkSplit: false,
		Support:      true,
		TestNetBlock: big.NewInt(494000),
		MainNetBlock: big.NewInt(1150000),
	}
	ETFork = fork.Fork{Name: "ETF",
		NetworkSplit:   true,
		Support:        false,
		TestNetBlock:   big.NewInt(137),
		MainNetBlock:   big.NewInt(1920000),
		ForkBlockExtra: common.FromHex("0x64616f2d686172642d666f726b"),
		ForkExtraRange: big.NewInt(10),
	}
)
