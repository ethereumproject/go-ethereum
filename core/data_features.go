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
)

var DefaultGasRepriceFeature = &ForkFeature{
	ID: "gasReprice",
	Options: &FeatureOptions{
		GasTable: DefaultGasRepriceGasTable,
	},
}

var DefaultEIP155Feature = &ForkFeature{
	ID: "eip155",
	Options: &FeatureOptions{
		ChainID: big.NewInt(61),
	},
}

var DefaultBombDelayFeature = &ForkFeature{
	ID: "ecip1010", // bomb delay
	Options: &FeatureOptions{
		Length: big.NewInt(2000000),
	},
}

var DefaultDiehardGasRepriceFeature = &ForkFeature{
	ID: "diehardGasprice",
	Options: &FeatureOptions{
		GasTable: DefaultDiehardGasTable,
	},
}
