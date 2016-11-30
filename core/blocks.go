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

package core

import "github.com/ethereumproject/go-ethereum/common"

// Set of manually tracked bad hashes (usually hard forks)
var BadHashes = map[common.Hash]bool{
	// consensus issue that occurred on the Frontier network at block 116,522, mined on 2015-08-20 at 14:59:16+02:00
	// https://blog.ethereum.org/2015/08/20/security-alert-consensus-issue
	common.HexToHash("05bef30ef572270f654746da22639a7a0c97dd97a7050b9e252391996aaeb689"): true,
	// ETFork #1920000 Block Hash
	common.HexToHash("4985f5ca3d2afbec36529aa96f74de3cc10a2a4a6c44f2157a57d2c6059a11bb"): true,
	// consensus issue at Testnet #383792
	common.HexToHash("9690db54968a760704d99b8118bf79d565711669cefad24b51b5b1013d827808"): true,
}

func LoadForkHashes() {
	c := NewChainConfig()
	for i := range c.Forks {
		if c.Forks[i].NetworkSplit {
			if c.Forks[i].Support {
				BadHashes[common.HexToHash(c.Forks[i].OrigSplitHash)] = true
			} else {
				BadHashes[common.HexToHash(c.Forks[i].ForkSplitHash)] = true
			}
		} else {
			if c.Forks[i].OrigSplitHash != "" {
				BadHashes[common.HexToHash(c.Forks[i].OrigSplitHash)] = true
			}
		}
	}
}
