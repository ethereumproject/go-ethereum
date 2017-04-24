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

import "math/big"

// DefaultBombDelayLength is the default delay length for the "explosion" algorithm according
// to ecip1010
var DefaultBombDelayLength = big.NewInt(2000000) // 1500000 ???

var DefaultGasRepriceFeature = &ForkFeature{
	ID: "gasReprice",
	Options: ChainFeatureConfigOptions{
		"gastable": "DefaultGasRepriceGasTable",
	},
}

var DefaultEIP155Feature = &ForkFeature{
	ID: "eip155",
	Options: ChainFeatureConfigOptions{
		"chainid": 61,
	},
}

var DefaultDiehardGasRepriceFeature = &ForkFeature{
	ID: "diehardGasprice",
	Options: ChainFeatureConfigOptions{
		// This is just an example of the arbitrariness of they key-value config.
		"gastable": `{
			"extcodesize:      700,
			"extcodecopy":     700,
			"balance":         400,
			"sload":           200,
			"calls":           700,
			"suicide":         5000,
			"expbyte":         50,
			"createbysuicide": 25000,
		}`,
	},
}
