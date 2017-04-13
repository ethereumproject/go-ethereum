// Copyright 2016 The go-ethereum Authors
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

import (
	hexlib "encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

var (
	ChainConfigNotFoundErr     = errors.New("ChainConfig not found")
	ChainConfigForkNotFoundErr = errors.New("ChainConfig fork not found")

	ErrHashKnownBad  = errors.New("known bad hash")
	ErrHashKnownFork = validateError("known fork hash mismatch")
)

// #chainconfigi
// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	// Forks holds fork block requirements. See ErrHashKnownFork.
	Forks []*Fork `json:"forks"`

	// BadHashes holds well known blocks with consensus issues. See ErrHashKnownBad.
	BadHashes []*BadHash `json:"bad_hashes"`

	ChainId *big.Int `json:"chain_id"`
}

type BadHash struct {
	Block *big.Int
	Hash  common.Hash
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	if c.Fork("Homestead").Block == nil || num == nil {
		return false
	}
	return num.Cmp(c.Fork("Homestead").Block) >= 0
}

// IsETF returns whether num is equal to the bailout fork.
func (c *ChainConfig) IsETF(num *big.Int) bool {
	if c.Fork("ETF").Block == nil || num == nil {
		return false
	}
	return num.Cmp(c.Fork("ETF").Block) == 0
}


// IsDiehard returns whether num is greater than or equal to the Diehard block, but less than explosion.
func (c *ChainConfig) IsDiehard(num *big.Int) bool {
	fork := c.Fork("Diehard")
	if fork.Block == nil || num == nil {
		return false
	}
	return num.Cmp(fork.Block) >= 0
}

// IsExplosion returns whether num is either equal to the explosion block or greater.
func (c *ChainConfig) IsExplosion(num *big.Int) bool {
	fork := c.Fork("Diehard")
	if fork.Block == nil || fork.Length == nil || num == nil {
		return false
	}
	block := big.NewInt(0).Add(fork.Block, fork.Length)
	return num.Cmp(block) >= 0
}

func (c *ChainConfig) Fork(name string) *Fork {
	for i := range c.Forks {
		if c.Forks[i].Name == name {
			return c.Forks[i]
		}
	}
	return &Fork{}
}

func (c *ChainConfig) HeaderCheck(h *types.Header) error {
	for _, fork := range c.Forks {
		if fork.Block.Cmp(h.Number) != 0 {
			continue
		}
		if common.EmptyHash(fork.RequiredHash) {
			continue
		}
		if fork.RequiredHash != h.Hash() {
			return ErrHashKnownFork
		}
	}

	for _, bad := range c.BadHashes {
		if bad.Block.Cmp(h.Number) != 0 {
			continue
		}
		if bad.Hash == h.Hash() {
			return ErrHashKnownBad
		}
	}

	return nil
}

func (c *ChainConfig) GetSigner(blockNumber *big.Int) types.Signer {
	if c.IsDiehard(blockNumber) {
		return types.NewChainIdSigner(c.ChainId)
	}
	return types.BasicSigner{}
}

// GasTable returns the gas table corresponding to the current fork
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) *vm.GasTable {
	t := &vm.GasTable{
		ExtcodeSize:     big.NewInt(20),
		ExtcodeCopy:     big.NewInt(20),
		Balance:         big.NewInt(20),
		SLoad:           big.NewInt(50),
		Calls:           big.NewInt(40),
		Suicide:         big.NewInt(0),
		ExpByte:         big.NewInt(10),
		CreateBySuicide: nil,
	}

	for _, fork := range c.Forks {
		if fork.Block.Cmp(num) <= 0 {
			if fork.GasTable != nil {
				t = fork.GasTable
			}
		}
	}

	return t
}

type Fork struct {
	Name string
	// For user notification only
	Support      bool
	NetworkSplit bool
	// Block is the block number where the hard-fork commences on
	// the Ethereum network.
	Block *big.Int
	// Length of fork, if limited
	Length *big.Int
	// RequiredHash to assist in avoiding sync issues
	// after network split.
	RequiredHash common.Hash
	// Gas Price table
	GasTable *vm.GasTable
	// TODO Derive Oracle contracts from fork struct (Version, Registrar, Release)
}

// WriteGenesisBlock writes the genesis block to the database as block number 0
func WriteGenesisBlock(chainDb ethdb.Database, genesis *GenesisDump) (*types.Block, error) {
	statedb, err := state.New(common.Hash{}, chainDb)
	if err != nil {
		return nil, err
	}

	for addrHex, account := range genesis.Alloc {
		var addr common.Address
		if err := addrHex.Decode(addr[:]); err != nil {
			return nil, fmt.Errorf("malformed addres %q: %s", addrHex, err)
		}

		balance, ok := new(big.Int).SetString(account.Balance, 0)
		if !ok {
			return nil, fmt.Errorf("malformed account %q balance %q", addrHex, account.Balance)
		}
		statedb.AddBalance(addr, balance)

		code, err := account.Code.Bytes()
		if err != nil {
			return nil, fmt.Errorf("malformed account %q code: %s", addrHex, err)
		}
		statedb.SetCode(addr, code)

		for key, value := range account.Storage {
			var k, v common.Hash
			if err := key.Decode(k[:]); err != nil {
				return nil, fmt.Errorf("malformed account %q key: %s", addrHex, err)
			}
			if err := value.Decode(v[:]); err != nil {
				return nil, fmt.Errorf("malformed account %q value: %s", addrHex, err)
			}
			statedb.SetState(addr, k, v)
		}
	}
	root, stateBatch := statedb.CommitBatch()

	header, err := genesis.Header()
	if err != nil {
		return nil, err
	}
	header.Root = root

	block := types.NewBlock(header, nil, nil, nil)

	if block := GetBlock(chainDb, block.Hash()); block != nil {
		glog.V(logger.Info).Infoln("Genesis block already in chain. Writing canonical number")
		err := WriteCanonicalHash(chainDb, block.Hash(), block.NumberU64())
		if err != nil {
			return nil, err
		}
		return block, nil
	}

	if err := stateBatch.Write(); err != nil {
		return nil, fmt.Errorf("cannot write state: %v", err)
	}
	if err := WriteTd(chainDb, block.Hash(), header.Difficulty); err != nil {
		return nil, err
	}
	if err := WriteBlock(chainDb, block); err != nil {
		return nil, err
	}
	if err := WriteBlockReceipts(chainDb, block.Hash(), nil); err != nil {
		return nil, err
	}
	if err := WriteCanonicalHash(chainDb, block.Hash(), block.NumberU64()); err != nil {
		return nil, err
	}
	if err := WriteHeadBlockHash(chainDb, block.Hash()); err != nil {
		return nil, err
	}

	return block, nil
}

type GenesisAccount struct {
	Address common.Address
	Balance *big.Int
}

func WriteGenesisBlockForTesting(db ethdb.Database, accounts ...GenesisAccount) *types.Block {
	dump := GenesisDump{
		GasLimit:   "0x47E7C4",
		Difficulty: "0x020000",
		Alloc:      make(map[hex]*GenesisDumpAlloc, len(accounts)),
	}

	for _, a := range accounts {
		dump.Alloc[hex(hexlib.EncodeToString(a.Address[:]))] = &GenesisDumpAlloc{
			Balance: a.Balance.String(),
		}
	}

	block, err := WriteGenesisBlock(db, &dump)
	if err != nil {
		panic(err)
	}
	return block
}

// GenesisDump is the geth JSON format.
// https://github.com/ethereumproject/wiki/wiki/Ethereum-Chain-Spec-Format#subformat-genesis
type GenesisDump struct {
	Nonce      prefixedHex
	Timestamp  prefixedHex
	ParentHash prefixedHex
	ExtraData  prefixedHex
	GasLimit   prefixedHex
	Difficulty prefixedHex
	Mixhash    prefixedHex
	Coinbase   prefixedHex

	// Alloc maps accounts by their address.
	Alloc map[hex]*GenesisDumpAlloc
}

// GenesisDumpAlloc is a GenesisDump.Alloc entry.
type GenesisDumpAlloc struct {
	Code    prefixedHex
	Storage map[hex]hex
	Balance string // decimal string
}

// Header returns the mapping.
func (g *GenesisDump) Header() (*types.Header, error) {
	var h types.Header

	var err error
	if err = g.Nonce.Decode(h.Nonce[:]); err != nil {
		return nil, fmt.Errorf("malformed nonce: %s", err)
	}
	if h.Time, err = g.Timestamp.Int(); err != nil {
		return nil, fmt.Errorf("malformed timestamp: %s", err)
	}
	if err = g.ParentHash.Decode(h.ParentHash[:]); err != nil {
		return nil, fmt.Errorf("malformed parentHash: %s", err)
	}
	if h.Extra, err = g.ExtraData.Bytes(); err != nil {
		return nil, fmt.Errorf("malformed extraData: %s", err)
	}
	if h.GasLimit, err = g.GasLimit.Int(); err != nil {
		return nil, fmt.Errorf("malformed gasLimit: %s", err)
	}
	if h.Difficulty, err = g.Difficulty.Int(); err != nil {
		return nil, fmt.Errorf("malformed difficulty: %s", err)
	}
	if err = g.Mixhash.Decode(h.MixDigest[:]); err != nil {
		return nil, fmt.Errorf("malformed mixhash: %s", err)
	}
	if err := g.Coinbase.Decode(h.Coinbase[:]); err != nil {
		return nil, fmt.Errorf("malformed coinbase: %s", err)
	}

	return &h, nil
}

// hex is a hexadecimal string.
type hex string

// Decode fills buf when h is not empty.
func (h hex) Decode(buf []byte) error {
	if len(h) != 2*len(buf) {
		return fmt.Errorf("want %d hexadecimals", 2*len(buf))
	}

	_, err := hexlib.Decode(buf, []byte(h))
	return err
}

// prefixedHex is a hexadecimal string with an "0x" prefix.
type prefixedHex string

var errNoHexPrefix = errors.New("want 0x prefix")

// Decode fills buf when h is not empty.
func (h prefixedHex) Decode(buf []byte) error {
	i := len(h)
	if i == 0 {
		return nil
	}
	if i == 1 || h[0] != '0' || h[1] != 'x' {
		return errNoHexPrefix
	}
	if i == 2 {
		return nil
	}
	if i != 2*len(buf)+2 {
		return fmt.Errorf("want %d hexadecimals with 0x prefix", 2*len(buf))
	}

	_, err := hexlib.Decode(buf, []byte(h[2:]))
	return err
}

func (h prefixedHex) Bytes() ([]byte, error) {
	l := len(h)
	if l == 0 {
		return nil, nil
	}
	if l == 1 || h[0] != '0' || h[1] != 'x' {
		return nil, errNoHexPrefix
	}
	if l == 2 {
		return nil, nil
	}

	bytes := make([]byte, l/2-1)
	_, err := hexlib.Decode(bytes, []byte(h[2:]))
	return bytes, err
}

func (h prefixedHex) Int() (*big.Int, error) {
	bytes, err := h.Bytes()
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(bytes), nil
}
