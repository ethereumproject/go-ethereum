package params

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
)

// GenesisDump is the geth JSON format.
// https://github.com/ethereumproject/wiki/wiki/Ethereum-Chain-Spec-Format#subformat-genesis
type GenesisDump struct {
	Nonce      PrefixedHex `json:"nonce"`
	Timestamp  PrefixedHex `json:"timestamp"`
	ParentHash PrefixedHex `json:"parentHash"`
	ExtraData  PrefixedHex `json:"extraData"`
	GasLimit   PrefixedHex `json:"gasLimit"`
	Difficulty PrefixedHex `json:"difficulty"`
	Mixhash    PrefixedHex `json:"mixhash"`
	Coinbase   PrefixedHex `json:"coinbase"`

	// Alloc maps accounts by their address.
	Alloc map[Hex]*GenesisDumpAlloc `json:"alloc"`
	// Alloc file contains CSV representation of Alloc
	AllocFile string `json:"alloc_file"`
}

// GenesisDumpAlloc is a GenesisDump.Alloc entry.
type GenesisDumpAlloc struct {
	Code    PrefixedHex `json:"-"` // skip field for json encode
	Storage map[Hex]Hex `json:"-"`
	Balance string      `json:"balance"` // decimal string
}

type GenesisAccount struct {
	Address common.Address `json:"address"`
	Balance *big.Int       `json:"balance"`
}
