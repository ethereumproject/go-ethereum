package core

import (
	"bytes"
	"math/big"

	"github.com/ethereumproject/go-ethereum/core/types"
)

// EPROJECT Load from genesis file
type Fork struct {
	Name         string
	NetworkSplit bool
	Support      bool
	// EPROJECT Add option to download both branches
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
	// EPROJECT Derive Oracle contracts (Version, Registrar, Release)
}

func (fork *Fork) ValidateForkHeaderExtraData(header *types.Header) error {
	// Short circuit validation if the node doesn't care about the DAO fork
	if fork.MainNetBlock == nil {
		return nil
	}
	// Make sure the block is within the fork's modified extra-data range
	limit := new(big.Int).Add(fork.MainNetBlock, fork.ForkExtraRange)
	if header.Number.Cmp(fork.MainNetBlock) < 0 || header.Number.Cmp(limit) >= 0 {
		return nil
	}
	// Depending whether we support or oppose the fork, validate the extra-data contents
	if fork.Support {
		if bytes.Compare(header.Extra, fork.ForkBlockExtra) != 0 {
			return ValidationError("Fork bad block extra-data: 0x%x", header.Extra)
		}
	} else {
		if bytes.Compare(header.Extra, fork.ForkBlockExtra) == 0 {
			return ValidationError("No-fork bad block extra-data: 0x%x", header.Extra)
		}
	}
	// All ok, header has the same extra-data we expect
	return nil
}
