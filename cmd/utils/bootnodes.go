// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import "github.com/ethereumproject/go-ethereum/p2p/discover"

// FrontierBootNodes are the enode URLs of the P2P bootstrap nodes running on
// the Frontier network.
var FrontierBootNodes = []*discover.Node{
	// ETC Volunteer Go Bootnodes
	discover.MustParseNode("enode://e809c4a2fec7daed400e5e28564e23693b23b2cc5a019b612505631bbe7b9ccf709c1796d2a3d29ef2b045f210caf51e3c4f5b6d3587d43ad5d6397526fa6179@174.112.32.157:30303"), //bar-
	discover.MustParseNode("enode://687be94c3a7beaa3d2fde82fa5046cdeb3e8198354e05b29d6e0d4e276713e3707ac10f784a7904938b06b46c764875c241b0337dd853385a4d8bfcbf8190647@95.183.51.229:30303"),  //tre-
	discover.MustParseNode("enode://6e538e7c1280f0a31ff08b382db5302480f775480b8e68f8febca0ceff81e4b19153c6f8bf60313b93bef2cc34d34e1df41317de0ce613a201d1660a788a03e2@52.206.67.235:30303"),  //ela-
	discover.MustParseNode("enode://217ebe27e89bf4fec8ce06509323ff095b1014378deb75ab2e5f6759a4e875 0a3bd8254b8c6833136e4d5e58230d65ee8ab34a5db5abf0640408c4288af3c8a7@188.138.1.237:30303"), //lec-
}

// TestNetBootNodes are the enode URLs of the P2P bootstrap nodes running on the
// Morden test network.
var TestNetBootNodes = []*discover.Node{
	// !EPROJECT TestNet Bootnodes needed
	// ETH/DEV Go Bootnodes
	discover.MustParseNode("enode://e4533109cc9bd7604e4ff6c095f7a1d807e15b38e9bfeb05d3b7c423ba86af0a9e89abbf40bd9dde4250fef114cd09270fa4e224cbeef8b7bf05a51e8260d6b8@94.242.229.4:40404"),
	discover.MustParseNode("enode://8c336ee6f03e99613ad21274f269479bf4413fb294d697ef15ab897598afb931f56beb8e97af530aee20ce2bcba5776f4a312bc168545de4d43736992c814592@94.242.229.203:30303"),

	// ETH/DEV Cpp Bootnodes
}
