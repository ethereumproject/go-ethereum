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
	"path/filepath"
	"strings"
	"testing"
)

func TestETHDifficulty(t *testing.T) {
	fileNames, _ := filepath.Glob(filepath.Join(ethBasicTestDir, "*"))

	supportedTests := map[string]bool{
		// "difficulty.json":          true, // Testing ETH mainnet config
		"difficultyFrontier.json":  true,
		"difficultyHomestead.json": true,
		"difficultyByzantium.json": true,
	}

	// Loop through each file
	for _, fn := range fileNames {
		fileName := fn[strings.LastIndex(fn, "/")+1 : len(fn)]

		if !supportedTests[fileName] {
			continue
		}

		t.Run(fileName, func(t *testing.T) {
			config := ChainConfigs[fileName]
			tests := make(map[string]DifficultyTest)

			if err := readJsonFile(fn, &tests); err != nil {
				t.Error(err)
			}

			// Loop through each test in file
			for key, test := range tests {
				// Subtest within the JSON file
				t.Run(key, func(t *testing.T) {
					if err := test.runDifficulty(t, &config); err != nil {
						t.Error(err)
					}
				})

			}
		})
	}
}
