package fork

import (
	"math/big"
)

type Fork struct {
	Name string
	// TestNetBlock is the block number where a hard-fork commences on
	// the Ethereum test network. It's enforced nil since it was decided not to do a
	// testnet transition.
	TestNetBlock *big.Int
	// MainNetBlock is the block number where the hard-fork commences on
	// the Ethereum main network.
	MainNetBlock *big.Int
	// ForkBlockExtra is the block header extra-data field to set for a fork
	// point and a number of consecutive blocks to allow fast/light syncers to correctly
	ForkBlockExtra []byte
	// ForkExtraRange is the number of consecutive blocks from the fork point
	// to override the extra-data in to prevent no-fork attacks.
	ForkExtraRange *big.Int
}
