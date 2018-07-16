package core

import (
	"bytes"
	enchex "encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/common/hexutil"
	"github.com/ethereumproject/go-ethereum/common/math"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/params"
)

/*
	(whilei): The contents of this file are intended, at this point, only for use by the tests/ package.
	Implementation uses GenesisDump struct, which uses raw params.PrefixedHex field types.
*/

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go
//go:generate gencodec -type GenesisAccount -field-override genesisAccountMarshaling -out gen_genesis_account.go

var errGenesisNoConfig = errors.New("genesis has no chain configuration")

// Genesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type Genesis struct {
	Config     *params.ChainConfig `json:"config"`
	Nonce      uint64              `json:"nonce"`
	Timestamp  uint64              `json:"timestamp"`
	ExtraData  []byte              `json:"extraData"`
	GasLimit   uint64              `json:"gasLimit"   gencodec:"required"`
	Difficulty *big.Int            `json:"difficulty" gencodec:"required"`
	Mixhash    common.Hash         `json:"mixHash"`
	Coinbase   common.Address      `json:"coinbase"`
	Alloc      GenesisAlloc        `json:"alloc"      gencodec:"required"`

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number     uint64      `json:"number"`
	GasUsed    uint64      `json:"gasUsed"`
	ParentHash common.Hash `json:"parentHash"`
}

// GenesisAlloc specifies the initial state that is part of the genesis block.
type GenesisAlloc map[common.Address]GenesisAccount

func (ga *GenesisAlloc) UnmarshalJSON(data []byte) error {
	m := make(map[common.UnprefixedAddress]GenesisAccount)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*ga = make(GenesisAlloc)
	for addr, a := range m {
		(*ga)[common.Address(addr)] = a
	}
	return nil
}

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code       []byte                      `json:"code,omitempty"`
	Storage    map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance    *big.Int                    `json:"balance" gencodec:"required"`
	Nonce      uint64                      `json:"nonce,omitempty"`
	PrivateKey []byte                      `json:"secretKey,omitempty"` // for tests
}

// field type overrides for gencodec
type genesisSpecMarshaling struct {
	Nonce      math.HexOrDecimal64
	Timestamp  math.HexOrDecimal64
	ExtraData  hexutil.Bytes
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Number     math.HexOrDecimal64
	Difficulty *math.HexOrDecimal256
	Alloc      map[common.UnprefixedAddress]GenesisAccount
}

type genesisAccountMarshaling struct {
	Code       hexutil.Bytes
	Balance    *math.HexOrDecimal256
	Nonce      math.HexOrDecimal64
	Storage    map[storageJSON]storageJSON
	PrivateKey hexutil.Bytes
}

// storageJSON represents a 256 bit byte array, but allows less than 256 bits when
// unmarshaling from hex.
type storageJSON common.Hash

func (h *storageJSON) UnmarshalText(text []byte) error {
	text = bytes.TrimPrefix(text, []byte("0x"))
	if len(text) > 64 {
		return fmt.Errorf("too many hex characters in storage key/value %q", text)
	}
	offset := len(h) - len(text)/2 // pad on the left
	if _, err := enchex.Decode(h[offset:], text); err != nil {
		fmt.Println(err)
		return fmt.Errorf("invalid hex storage key/value %q", text)
	}
	return nil
}

func (h storageJSON) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New common.Hash
}

func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database already contains an incompatible genesis block (have %x, new %x)", e.Stored[:8], e.New[:8])
}

// ToBlock creates the genesis block and writes state of a genesis specification
// to the given database (or discards it if nil).
func (g *Genesis) ToBlock(db ethdb.Database) *types.Block {
	if db == nil {
		db, _ = ethdb.NewMemDatabase()
	}
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	for addr, account := range g.Alloc {
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	root := statedb.IntermediateRoot(false)
	head := &types.Header{
		Number:     new(big.Int).SetUint64(g.Number),
		Nonce:      types.EncodeNonce(g.Nonce),
		Time:       new(big.Int).SetUint64(g.Timestamp),
		ParentHash: g.ParentHash,
		Extra:      g.ExtraData,
		GasLimit:   g.GasLimit,
		GasUsed:    g.GasUsed,
		Difficulty: g.Difficulty,
		MixDigest:  g.Mixhash,
		Coinbase:   g.Coinbase,
		Root:       root,
	}
	if g.GasLimit == 0 {
		head.GasLimit = params.GenesisGasLimit
	}
	if g.Difficulty == nil {
		head.Difficulty = params.GenesisDifficulty
	}

	_, err := statedb.CommitTo(db, false)
	if err != nil {
		panic("err committo statedb " + err.Error())
	}
	// sdb, err := state.New(root, state.NewDatabase(db))
	// if err != nil {
	// 	panic("err stateNew " + err.Error())
	// }
	// _, err = sdb.CommitTo(db, false)
	// if err != nil {
	// 	panic("sdb committo " + err.Error())
	// }
	// tr, err := statedb.OpenTrie(root)
	// if err != nil {
	// 	panic("err open trie " + err.Error())
	// }
	// _, err = tr.CommitTo(db)
	// if err != nil {
	// 	panic("err commit trie " + err.Error())
	// }
	// statedb.Database().OpenTrie()
	// statedb.Database().TrieDB().Commit(root, true)

	return types.NewBlock(head, nil, nil, nil)
}

// Commit writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func (g *Genesis) Commit(db ethdb.Database) (*types.Block, error) {
	block := g.ToBlock(db)
	if block.Number().Sign() != 0 {
		return nil, fmt.Errorf("can't commit genesis block with number > 0")
	}
	WriteTd(db, block.Hash(), g.Difficulty)
	WriteBlock(db, block)
	WriteBlockReceipts(db, block.Hash(), nil)
	WriteCanonicalHash(db, block.Hash(), block.NumberU64())
	WriteHeadBlockHash(db, block.Hash())
	WriteHeadHeaderHash(db, block.Hash())

	// config := g.Config
	// if config == nil {
	// 	config = params.AllEthashProtocolChanges
	// }
	// rawdb.WriteChainConfig(db, block.Hash(), config)
	return block, nil
}

// MustCommit writes the genesis block and state to db, panicking on error.
// The block is committed as the canonical head block.
func (g *Genesis) MustCommit(db ethdb.Database) *types.Block {
	block, err := g.Commit(db)
	if err != nil {
		panic(err)
	}
	return block
}

// // GenesisBlockForTesting creates and writes a block in which addr has the given wei balance.
// func GenesisBlockForTesting(db ethdb.Database, addr common.Address, balance *big.Int) *types.Block {
// 	g := Genesis{Alloc: GenesisAlloc{addr: {Balance: balance}}}
// 	return g.MustCommit(db)
// }

//
// // DefaultGenesisBlock returns the Ethereum main net genesis block.
// func DefaultGenesisBlock() *Genesis {
// 	return &Genesis{
// 		Config:     params.MainnetChainConfig,
// 		Nonce:      66,
// 		ExtraData:  hexutil.MustDecode("0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa"),
// 		GasLimit:   5000,
// 		Difficulty: big.NewInt(17179869184),
// 		Alloc:      decodePrealloc(mainnetAllocData),
// 	}
// }
//
// // DefaultTestnetGenesisBlock returns the Ropsten network genesis block.
// func DefaultTestnetGenesisBlock() *Genesis {
// 	return &Genesis{
// 		Config:     params.TestnetChainConfig,
// 		Nonce:      66,
// 		ExtraData:  hexutil.MustDecode("0x3535353535353535353535353535353535353535353535353535353535353535"),
// 		GasLimit:   16777216,
// 		Difficulty: big.NewInt(1048576),
// 		Alloc:      decodePrealloc(testnetAllocData),
// 	}
// }
//
// // DefaultRinkebyGenesisBlock returns the Rinkeby network genesis block.
// func DefaultRinkebyGenesisBlock() *Genesis {
// 	return &Genesis{
// 		Config:     params.RinkebyChainConfig,
// 		Timestamp:  1492009146,
// 		ExtraData:  hexutil.MustDecode("0x52657370656374206d7920617574686f7269746168207e452e436172746d616e42eb768f2244c8811c63729a21a3569731535f067ffc57839b00206d1ad20c69a1981b489f772031b279182d99e65703f0076e4812653aab85fca0f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
// 		GasLimit:   4700000,
// 		Difficulty: big.NewInt(1),
// 		Alloc:      decodePrealloc(rinkebyAllocData),
// 	}
// }
//
// // DeveloperGenesisBlock returns the 'geth --dev' genesis block. Note, this must
// // be seeded with the
// func DeveloperGenesisBlock(period uint64, faucet common.Address) *Genesis {
// 	// Override the default period to the user requested one
// 	config := *params.AllCliqueProtocolChanges
// 	config.Clique.Period = period
//
// 	// Assemble and return the genesis with the precompiles and faucet pre-funded
// 	return &Genesis{
// 		Config:     &config,
// 		ExtraData:  append(append(make([]byte, 32), faucet[:]...), make([]byte, 65)...),
// 		GasLimit:   6283185,
// 		Difficulty: big.NewInt(1),
// 		Alloc: map[common.Address]GenesisAccount{
// 			common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // ECRecover
// 			common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
// 			common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
// 			common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
// 			common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
// 			common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
// 			common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
// 			common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
// 			faucet: {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
// 		},
// 	}
// }
//
// func decodePrealloc(data string) GenesisAlloc {
// 	var p []struct{ Addr, Balance *big.Int }
// 	if err := rlp.NewStream(strings.NewReader(data), 0).Decode(&p); err != nil {
// 		panic(err)
// 	}
// 	ga := make(GenesisAlloc, len(p))
// 	for _, account := range p {
// 		ga[common.BigToAddress(account.Addr)] = GenesisAccount{Balance: account.Balance}
// 	}
// 	return ga
// }
