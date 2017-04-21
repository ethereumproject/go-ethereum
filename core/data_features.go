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

// DefaultHomesteadFeature is the default homestead fork feature configuration
var DefaultHomesteadFeature = &ForkFeature{
	ID: "homestead",
	Options: &FeatureOptions{
		GasTable:     HomeSteadGasTable,
	},
}

// DefaultETFFeature is the default etf fork feature configuration
var DefaultETFFeature = &ForkFeature{
	ID: "etf",
}

// DefaultGasRepriceFeature is the default gasreprice fork feature configuration
var DefaultGasRepriceFeature = &ForkFeature{
	ID: "gasreprice",
	Options: &FeatureOptions{
		GasTable:     GasRepriceGasTable,
	},
}

// DefaultDiehardFeature is the default diehard fork feature configuration
var DefaultDiehardFeature = &ForkFeature{
	ID: "diehard",
	Options: &FeatureOptions{
		Length:       big.NewInt(2000000),
		GasTable:     DiehardGasTable,
	},
}
