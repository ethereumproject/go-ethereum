package fork

import (
	"math/big"
)

// EPROJECT Load from genesis file
type Fork struct {
	Name         string
	NetworkSplit bool
	Support      bool
	// TestNetBlock is the block number where a hard-fork commences on
	// the Ethereum test network. It's enforced nil since it was decided not to do a
	// testnet transition.
	TestNetBlock *big.Int
	// MainNetBlock is the block number where the hard-fork commences on
	// the Ethereum main network.
	MainNetBlock *big.Int

	// EPROJECT Derive bad blocks from fork struct
	// ETC MainNetBlock+1 "94365e3a8c0b35089c1d1195081fe7489b528a84b22199c916180db8b28ade7f"
	// ETF MainNetBlock+1 "4985f5ca3d2afbec36529aa96f74de3cc10a2a4a6c44f2157a57d2c6059a11bb"

	// ForkBlockExtra is the block header extra-data field to set for a fork
	// point and a number of consecutive blocks to allow fast/light syncers to correctly
	ForkBlockExtra []byte
	// ForkExtraRange is the number of consecutive blocks from the fork point
	// to override the extra-data in to prevent no-fork attacks.
	ForkExtraRange *big.Int
}
