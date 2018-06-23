package core

import (
	"fmt"
	"math/big"

	hexlib "encoding/hex"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/params"
)

// MakeGenesisDump makes a genesis dump
func MakeGenesisDump(chaindb ethdb.Database) (*params.GenesisDump, error) {
	genesis := GetBlock(chaindb, GetCanonicalHash(chaindb, 0))
	if genesis == nil {
		return nil, nil
	}
	// Settings.

	genesisHeader := genesis.Header()
	nonce := fmt.Sprintf(`0x%x`, genesisHeader.Nonce)
	time := common.BigToHash(genesisHeader.Time).Hex()
	parentHash := genesisHeader.ParentHash.Hex()
	gasLimit := common.BigToHash(new(big.Int).SetUint64(genesisHeader.GasLimit)).Hex()
	difficulty := common.BigToHash(genesisHeader.Difficulty).Hex()
	mixHash := genesisHeader.MixDigest.Hex()
	coinbase := genesisHeader.Coinbase.Hex()
	var dump = &params.GenesisDump{
		Nonce:      params.PrefixedHex(nonce), // common.ToHex(n)), // common.ToHex(
		Timestamp:  params.PrefixedHex(time),
		ParentHash: params.PrefixedHex(parentHash),
		//ExtraData:  params.PrefixedHex(extra),
		GasLimit:   params.PrefixedHex(gasLimit),
		Difficulty: params.PrefixedHex(difficulty),
		Mixhash:    params.PrefixedHex(mixHash),
		Coinbase:   params.PrefixedHex(coinbase),
		//Alloc: ,
	}
	if genesisHeader.Extra != nil && len(genesisHeader.Extra) > 0 {
		dump.ExtraData = params.PrefixedHex(common.ToHex(genesisHeader.Extra))
	}
	// State allocations.
	genState, err := state.New(genesis.Root(), state.NewDatabase(chaindb))
	if err != nil {
		return nil, err
	}
	stateDump := genState.RawDump([]common.Address{})
	stateAccounts := stateDump.Accounts
	dump.Alloc = make(map[params.Hex]*params.GenesisDumpAlloc, len(stateAccounts))
	for address, acct := range stateAccounts {
		if common.IsHexAddress(address) {
			dump.Alloc[params.Hex(address)] = &params.GenesisDumpAlloc{
				Balance: acct.Balance,
			}
		} else {
			return nil, fmt.Errorf("Invalid address in genesis state: %v", address)
		}
	}
	return dump, nil
}

// WriteGenesisBlock writes the genesis block to the database as block number 0
func WriteGenesisBlock(chainDb ethdb.Database, genesis *params.GenesisDump) (*types.Block, error) {
	statedb, err := state.New(common.Hash{}, state.NewDatabase(chainDb))
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
	root, err := statedb.CommitTo(chainDb, false)
	if err != nil {
		return nil, err
	}

	header, err := genesis.Header()
	if err != nil {
		return nil, err
	}
	header.Root = root

	gblock := types.NewBlock(header, nil, nil, nil)

	if block := GetBlock(chainDb, gblock.Hash()); block != nil {
		glog.V(logger.Debug).Infof("Genesis block %s already exists in chain -- writing canonical number", block.Hash().Hex())
		err := WriteCanonicalHash(chainDb, block.Hash(), block.NumberU64())
		if err != nil {
			return nil, err
		}
		return block, nil
	}

	//if err := stateBatch.Write(); err != nil {
	//	return nil, fmt.Errorf("cannot write state: %v", err)
	//}
	if err := WriteTd(chainDb, gblock.Hash(), header.Difficulty); err != nil {
		return nil, err
	}
	if err := WriteBlock(chainDb, gblock); err != nil {
		return nil, err
	}
	if err := WriteBlockReceipts(chainDb, gblock.Hash(), nil); err != nil {
		return nil, err
	}
	if err := WriteCanonicalHash(chainDb, gblock.Hash(), gblock.NumberU64()); err != nil {
		return nil, err
	}
	if err := WriteHeadBlockHash(chainDb, gblock.Hash()); err != nil {
		return nil, err
	}

	return gblock, nil
}

func WriteGenesisBlockForTesting(db ethdb.Database, accounts ...params.GenesisAccount) *types.Block {
	dump := params.GenesisDump{
		GasLimit:   "0x47E7C4",
		Difficulty: "0x020000",
		Alloc:      make(map[params.Hex]*params.GenesisDumpAlloc, len(accounts)),
	}

	for _, a := range accounts {
		dump.Alloc[params.Hex(hexlib.EncodeToString(a.Address[:]))] = &params.GenesisDumpAlloc{
			Balance: a.Balance.String(),
		}
	}

	block, err := WriteGenesisBlock(db, &dump)
	if err != nil {
		panic(err)
	}
	return block
}

// Makeparams.GenesisDump makes a genesis dump
func GenesisDump(chaindb ethdb.Database) (*params.GenesisDump, error) {

	genesis := GetBlock(chaindb, GetCanonicalHash(chaindb, 0))
	if genesis == nil {
		return nil, nil
	}

	// Settings.
	genesisHeader := genesis.Header()
	nonce := fmt.Sprintf(`0x%x`, genesisHeader.Nonce)
	time := common.BigToHash(genesisHeader.Time).Hex()
	parentHash := genesisHeader.ParentHash.Hex()
	gasLimit := fmt.Sprintf("0x%x", genesisHeader.GasLimit)
	difficulty := common.BigToHash(genesisHeader.Difficulty).Hex()
	mixHash := genesisHeader.MixDigest.Hex()
	coinbase := genesisHeader.Coinbase.Hex()

	var dump = &params.GenesisDump{
		Nonce:      params.PrefixedHex(nonce), // common.ToHex(n)), // common.ToHex(
		Timestamp:  params.PrefixedHex(time),
		ParentHash: params.PrefixedHex(parentHash),
		//ExtraData:  params.PrefixedHex(extra),
		GasLimit:   params.PrefixedHex(gasLimit),
		Difficulty: params.PrefixedHex(difficulty),
		Mixhash:    params.PrefixedHex(mixHash),
		Coinbase:   params.PrefixedHex(coinbase),
		//Alloc: ,
	}
	if genesisHeader.Extra != nil && len(genesisHeader.Extra) > 0 {
		dump.ExtraData = params.PrefixedHex(common.ToHex(genesisHeader.Extra))
	}

	// State allocations.
	genState, err := state.New(genesis.Root(), state.NewDatabase(chaindb))
	if err != nil {
		return nil, err
	}
	stateDump := genState.RawDump([]common.Address{})

	stateAccounts := stateDump.Accounts
	dump.Alloc = make(map[params.Hex]*params.GenesisDumpAlloc, len(stateAccounts))

	for address, acct := range stateAccounts {
		if common.IsHexAddress(address) {
			dump.Alloc[params.Hex(address)] = &params.GenesisDumpAlloc{
				Balance: acct.Balance,
			}
		} else {
			return nil, fmt.Errorf("Invalid address in genesis state: %v", address)
		}
	}
	return dump, nil
}
