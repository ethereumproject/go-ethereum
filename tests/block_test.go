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
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereumproject/go-ethereum/logger/glog"
)

func init() {
	glog.SetD(0)
	glog.SetV(0)
}

func TestBlockchainTests(t *testing.T) {
	err := filepath.Walk(blockTestDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			t.Fatalf("%s: FAIL [walk err]=%v", p, err) // debugging, should not happen
			return nil
		}
		if info.IsDir() {
			// t.Logf("%s: SKIP [DIR]", p) // debugging
			return nil
		}
		mil := big.NewInt(1000000)
		if e := RunBlockTest(mil, mil, p, BlockSkipTests); e != nil {
			// if e != nil {
			// 	// Originally our tests had hardcoded fork block parameters. This "softly" ensures that those parameters can be met.
			// 	// Interestingly, however, this appears to never be touched.
			// 	t.Logf("1err=%v", e)
			if e2 := RunBlockTest(new(big.Int), mil, p, BlockSkipTests); e2 != nil {
				t.Errorf("%s: FAIL2 err=%v", p, e2)

			} else {
				t.Logf("%s: PASS2", p)
			}

			// } else {
			t.Errorf("%s: FAIL err=%v", p, e)
		} else {
			t.Logf("%s: PASS", p)
		}
		return nil
	})
	if err != nil {
		panic(err.Error())
	}
}
