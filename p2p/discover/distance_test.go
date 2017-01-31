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

package discover

import (
	"math/big"
	"testing"
	"testing/quick"

	"github.com/ethereumproject/go-ethereum/common"
)

func TestDistcmp(t *testing.T) {
	distcmpBig := func(target, a, b common.Hash) int {
		tbig := new(big.Int).SetBytes(target[:])
		abig := new(big.Int).SetBytes(a[:])
		bbig := new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(tbig, abig).Cmp(new(big.Int).Xor(tbig, bbig))
	}
	if err := quick.CheckEqual(distcmp, distcmpBig, quickcfg()); err != nil {
		t.Error(err)
	}
}

// the random tests is likely to miss the case where they're equal.
func TestDistcmpEqual(t *testing.T) {
	base := common.Hash{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	x := common.Hash{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	if distcmp(base, x, x) != 0 {
		t.Errorf("distcmp(base, x, x) != 0")
	}
}

func TestLogdist(t *testing.T) {
	logdistBig := func(a, b common.Hash) int {
		abig, bbig := new(big.Int).SetBytes(a[:]), new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(abig, bbig).BitLen()
	}
	if err := quick.CheckEqual(logdist, logdistBig, quickcfg()); err != nil {
		t.Error(err)
	}
}

// the random tests is likely to miss the case where they're equal.
func TestLogdistEqual(t *testing.T) {
	x := common.Hash{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	if logdist(x, x) != 0 {
		t.Errorf("logdist(x, x) != 0")
	}
}
