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
	// "regexp"
	"testing"
)

func TestState(t *testing.T) {
	t.Parallel()
	st := new(testMatcher)
	// Long tests:
	st.skipShortMode(`^stQuadraticComplexityTest/`)
	// Broken tests:
	st.skipLoad(`^stTransactionTest/OverflowGasRequire\.json`) // gasLimit > 256 bits
	st.skipLoad(`^stTransactionTest/zeroSigTransa[^/]*\.json`) // EIP-86 is not supported yet
	// Expected failures:
	st.fails(`^stRevertTest/RevertPrecompiledTouch\.json/EIP158`, "bug in test")
	st.fails(`^stRevertTest/RevertPrecompiledTouch\.json/Byzantium`, "bug in test")

	st.walk(t, stateTestDir, func(t *testing.T, name string, test *StateTest) {
		for _, subtest := range test.Subtests() {
			subtest := subtest
			key := fmt.Sprintf("%s/%d", subtest.Fork, subtest.Index)
			name := name + "/" + key
			t.Run(key, func(t *testing.T) {
				for _, f := range []string{"EIP158", "Constantinople", "HomesteadToDaoAt5", "EIP158ToByzantiumAt5"} {
					if subtest.Fork == f {
						t.Skipf("fork=%s not supported", subtest.Fork)
					}
				}

				rs, ok := Rules[subtest.Fork]
				if !ok {
					t.Skipf("WARNING fork=%s not supported", subtest.Fork)
				}

				_, err := test.Run(test, subtest, rs)
				if e := st.checkFailure(t, name, err); e != nil {
					t.Fatal(err)
				}
			})
		}
	})
}
