// Copyright 2014 The go-ethereum Authors
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
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/params"
)

// WriteGenesisBlock writes the genesis block to the database as block number 0
func WriteGenesisBlock(chainDb ethdb.Database, r io.Reader) (*types.Block, error) {
	var genesis GenesisDump
	if err := json.NewDecoder(r).Decode(&genesis); err != nil {
		return nil, err
	}

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

// GenesisBlockForTesting creates a block in which addr has the given wei balance.
// The state trie of the block is written to db. the passed db needs to contain a state root
func GenesisBlockForTesting(db ethdb.Database, addr common.Address, balance *big.Int) *types.Block {
	statedb, err := state.New(common.Hash{}, db)
	if err != nil {
		panic(err)
	}

	obj := statedb.GetOrNewStateObject(addr)
	obj.SetBalance(balance)
	root, err := statedb.Commit()
	if err != nil {
		panic(fmt.Sprintf("cannot write state: %v", err))
	}

	return types.NewBlock(&types.Header{
		Difficulty: params.GenesisDifficulty,
		GasLimit:   params.GenesisGasLimit,
		Root:       root,
	}, nil, nil, nil)
}

type GenesisAccount struct {
	Address common.Address
	Balance *big.Int
}

func WriteGenesisBlockForTesting(db ethdb.Database, accounts ...GenesisAccount) *types.Block {
	accountJson := "{"
	for i, account := range accounts {
		if i != 0 {
			accountJson += ","
		}
		accountJson += fmt.Sprintf(`"%x":{"balance":"0x%x"}`, account.Address, account.Balance.Bytes())
	}
	accountJson += "}"

	testGenesis := fmt.Sprintf(`{
	"nonce":"0x%x",
	"gasLimit":"0x%x",
	"difficulty":"0x%x",
	"alloc": %s
}`, types.EncodeNonce(0), params.GenesisGasLimit.Bytes(), params.GenesisDifficulty.Bytes(), accountJson)
	block, err := WriteGenesisBlock(db, strings.NewReader(testGenesis))
	if err != nil {
		panic(err)
	}
	return block
}

// WriteDefaultGenesisBlock assembles the official Ethereum genesis block and
// writes it - along with all associated state - into a chain database.
func WriteDefaultGenesisBlock(chainDb ethdb.Database) (*types.Block, error) {
	return WriteGenesisBlock(chainDb, strings.NewReader(DefaultGenesisBlock()))
}

// WriteTestNetGenesisBlock assembles the Morden test network genesis block and
// writes it - along with all associated state - into a chain database.
func WriteTestNetGenesisBlock(chainDb ethdb.Database) (*types.Block, error) {
	return WriteGenesisBlock(chainDb, strings.NewReader(TestNetGenesisBlock()))
}

// DefaultGenesisBlock assembles a JSON string representing the default Ethereum
// genesis block.
func DefaultGenesisBlock() string {
	reader, err := gzip.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(defaultGenesisBlock)))
	if err != nil {
		panic(fmt.Sprintf("failed to access default genesis: %v", err))
	}
	blob, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(fmt.Sprintf("failed to load default genesis: %v", err))
	}
	return string(blob)
}

// OlympicGenesisBlock assembles a JSON string representing the Olympic genesis
// block.
func OlympicGenesisBlock() string {
	return fmt.Sprintf(`{
		"nonce":"0x%x",
		"gasLimit":"0x%x",
		"difficulty":"0x%x",
		"alloc": {
			"0000000000000000000000000000000000000001": {"balance": "1"},
			"0000000000000000000000000000000000000002": {"balance": "1"},
			"0000000000000000000000000000000000000003": {"balance": "1"},
			"0000000000000000000000000000000000000004": {"balance": "1"},
			"dbdbdb2cbd23b783741e8d7fcf51e459b497e4a6": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
			"e4157b34ea9615cfbde6b4fda419828124b70c78": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
			"b9c015918bdaba24b4ff057a92a3873d6eb201be": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
			"6c386a4b26f73c802f34673f7248bb118f97424a": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
			"cd2a3d9f938e13cd947ec05abc7fe734df8dd826": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
			"2ef47100e0787b915105fd5e3f4ff6752079d5cb": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
			"e6716f9544a56c530d868e4bfbacb172315bdead": {"balance": "1606938044258990275541962092341162602522202993782792835301376"},
			"1a26338f0d905e295fccb71fa9ea849ffa12aaf4": {"balance": "1606938044258990275541962092341162602522202993782792835301376"}
		}
	}`, types.EncodeNonce(42), params.GenesisGasLimit.Bytes(), params.GenesisDifficulty.Bytes())
}

// TestNetGenesisBlock assembles a JSON string representing the Morden test net
// genenis block.
func TestNetGenesisBlock() string {
	return fmt.Sprintf(`{
		"nonce": "0x%x",
		"difficulty": "0x2000",
		"mixhash": "0x00000000000000000000000000000000000000647572616c65787365646c6578",
		"coinbase": "0x0000000000000000000000000000000000000000",
		"timestamp": "0x00",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"extraData": "0x",
		"gasLimit": "0x2FEFD8",
		"alloc": {
			"0000000000000000000000000000000000000001": { "balance": "1" },
			"0000000000000000000000000000000000000002": { "balance": "1" },
			"0000000000000000000000000000000000000003": { "balance": "1" },
			"0000000000000000000000000000000000000004": { "balance": "1" },
			"102e61f5d8f9bc71d0ad4a084df4e65e05ce0e1c": { "balance": "1606938044258990275541962092341162602522202993782792835301376" }
		}
	}`, types.EncodeNonce(0x6d6f7264656e))
}
