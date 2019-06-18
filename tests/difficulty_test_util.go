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

package tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/common/hexutil"
	"github.com/eth-classic/go-ethereum/core"
	"github.com/eth-classic/go-ethereum/core/types"
	"github.com/eth-classic/go-ethereum/params"
)

// DifficultyTest is the structure of JSON from test files
type DifficultyTest struct {
	ParentTimestamp    string      `json:"parentTimestamp"`
	ParentDifficulty   string      `json:"parentDifficulty"`
	UncleHash          common.Hash `json:"parentUncles"`
	CurrentTimestamp   string      `json:"currentTimestamp"`
	CurrentBlockNumber string      `json:"currentBlockNumber"`
	CurrentDifficulty  string      `json:"currentDifficulty"`
}

func (test *DifficultyTest) runDifficulty(t *testing.T, config *core.ChainConfig) error {
	currentNumber, _ := hexutil.HexOrDecimalToBigInt(test.CurrentBlockNumber)
	parentNumber := new(big.Int).Sub(currentNumber, big.NewInt(1))
	parentTimestamp, _ := hexutil.HexOrDecimalToBigInt(test.ParentTimestamp)
	parentDifficulty, _ := hexutil.HexOrDecimalToBigInt(test.ParentDifficulty)
	currentTimestamp, _ := hexutil.HexOrDecimalToUint64(test.CurrentTimestamp)

	parent := &types.Header{
		Number:     parentNumber,
		Time:       parentTimestamp,
		Difficulty: parentDifficulty,
		UncleHash:  test.UncleHash,
	}

	if config.IsAtlantis(parentNumber) && parent.Number.Cmp(big.NewInt(3199999)) >= 0 {
		return nil
	}

	// Check to make sure difficulty is above minimum
	if parentDifficulty.Cmp(params.MinimumDifficulty) < 0 {
		t.Skip("difficulty below minimum")
		return nil
	}

	actual := core.CalcDifficulty(config, currentTimestamp, parent)
	exp, _ := hexutil.HexOrDecimalToBigInt(test.CurrentDifficulty)

	if actual.Cmp(exp) != 0 {
		return fmt.Errorf("parent[time %v diff %v unclehash:%x] child[time %v number %v] diff %v != expected %v",
			test.ParentTimestamp, test.ParentDifficulty, test.UncleHash,
			test.CurrentTimestamp, test.CurrentBlockNumber, actual, exp)
	}
	return nil

}
