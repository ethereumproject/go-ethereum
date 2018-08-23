/*  */ // Copyright 2015 The go-ethereum Authors
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

package tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"bytes"
	"strings"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/common/hexutil"
	"github.com/ethereumproject/go-ethereum/common/math"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/ethdb"
)

// func RunStateTestWithReader(ruleSet RuleSet, r io.Reader, skipTests []string) error {
// 	tests := make(map[string]stTest)
// 	if err := readJSON(r, &tests); err != nil {
// 		return err
// 	}

// 	if err := runStateTests(tests, skipTests); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func RunStateTest(ruleSet RuleSet, p string, skipTests []string) error {
// 	tests := make(map[string]stTest)
// 	if err := readJSONFile(p, &tests); err != nil {
// 		return err
// 	}

// 	if err := runStateTests(tests, skipTests); err != nil {
// 		return err
// 	}

// 	return nil

// }

// func BenchStateTest(ruleSet RuleSet, p string, conf bconf, b *testing.B) error {
// 	tests := make(map[string]stTest)
// 	if err := readJSONFile(p, &tests); err != nil {
// 		return err
// 	}
// 	test, ok := tests[conf.name]
// 	if !ok {
// 		return fmt.Errorf("test not found: %s", conf.name)
// 	}

// 	// XXX Yeah, yeah...
// 	env := make(map[string]string)
// 	env["currentCoinbase"] = test.Env.CurrentCoinbase
// 	env["currentDifficulty"] = test.Env.CurrentDifficulty
// 	env["currentGasLimit"] = test.Env.CurrentGasLimit
// 	env["currentNumber"] = test.Env.CurrentNumber
// 	env["previousHash"] = test.Env.PreviousHash
// 	if n, ok := test.Env.CurrentTimestamp.(float64); ok {
// 		env["currentTimestamp"] = strconv.Itoa(int(n))
// 	} else {
// 		env["currentTimestamp"] = test.Env.CurrentTimestamp.(string)
// 	}

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		benchStateTest(ruleSet, test, env, b)
// 	}

// 	return nil
// }

// func benchStateTest(ruleSet RuleSet, test VmTest, env map[string]string, b *testing.B) {
// 	b.StopTimer()
// 	db, _ := ethdb.NewMemDatabase()
// 	statedb := makePreState(db, test.Pre)
// 	b.StartTimer()

// 	RunState(db, statedb, env, test.Exec)
// }

// func runStateTests(tests map[string]stTest, skipTests []string) error {
// 	skipTest := make(map[string]bool, len(skipTests))
// 	for _, name := range skipTests {
// 		skipTest[name] = true
// 	}

// 	for name, test := range tests {
// 		if skipTest[name] /*|| name != "callcodecallcode_11" */ {
// 			glog.Infoln("Skipping state test", name)
// 			continue
// 		}

// 		//fmt.Println("StateTest:", name)
// 		if err := runStateTest(ruleSet, test); err != nil {
// 			return fmt.Errorf("%s: %s\n", name, err.Error())
// 		}

// 		//glog.Infoln("State test passed: ", name)
// 		//fmt.Println(string(statedb.Dump()))
// 	}
// 	return nil

// }

// func runStateTest(ruleSet RuleSet, test VmTest) error {
// 	db, _ := ethdb.NewMemDatabase()
// 	statedb := makePreState(db, test.Pre)

// 	// XXX Yeah, yeah...
// 	env := make(map[string]string)
// 	env["currentCoinbase"] = test.Env.CurrentCoinbase
// 	env["currentDifficulty"] = test.Env.CurrentDifficulty
// 	env["currentGasLimit"] = test.Env.CurrentGasLimit
// 	env["currentNumber"] = test.Env.CurrentNumber
// 	env["previousHash"] = test.Env.PreviousHash
// 	if n, ok := test.Env.CurrentTimestamp.(float64); ok {
// 		env["currentTimestamp"] = strconv.Itoa(int(n))
// 	} else {
// 		env["currentTimestamp"] = test.Env.CurrentTimestamp.(string)
// 	}

// 	var (
// 		ret []byte
// 		// gas  *big.Int
// 		// err  error
// 		stlogs vm.Logs
// 	)

// 	ret, stlogs, _, _ = RunState(ruleSet, db, statedb, env, test.Transaction)

// 	// Compare expected and actual return
// 	rexp := common.FromHex(test.Out)
// 	if !bytes.Equal(rexp, ret) {
// 		return fmt.Errorf("return failed. Expected %x, got %x\n", rexp, ret)
// 	}

// 	// check post state
// 	for addr, account := range test.Post {
// 		obj := statedb.GetAccount(common.HexToAddress(addr))
// 		if obj == nil {
// 			return fmt.Errorf("did not find expected post-state account: %s", addr)
// 		}
// 		// Because vm.Account interface does not have Nonce method, so after
// 		// checking that obj exists, we'll use the StateObject type afterwards
// 		sobj := statedb.GetOrNewStateObject(common.HexToAddress(addr))

// 		if balance, ok := new(big.Int).SetString(account.Balance, 0); !ok {
// 			panic("malformed test account balance")
// 		} else if balance.Cmp(obj.Balance()) != 0 {
// 			return fmt.Errorf("(%x) balance failed. Expected: %v have: %v\n", obj.Address().Bytes()[:4], account.Balance, obj.Balance())
// 		}

// 		if nonce, err := strconv.ParseUint(account.Nonce, 0, 64); err != nil {
// 			return fmt.Errorf("test account %q malformed nonce: %s", addr, err)
// 		} else if sobj.Nonce() != nonce {
// 			return fmt.Errorf("(%x) nonce failed. Expected: %v have: %v\n", obj.Address().Bytes()[:4], account.Nonce, sobj.Nonce())
// 		}

// 		for addr, value := range account.Storage {
// 			v := statedb.GetState(obj.Address(), common.HexToHash(addr))
// 			vexp := common.HexToHash(value)

// 			if v != vexp {
// 				return fmt.Errorf("storage failed:\n%x: %s:\nexpected: %x\nhave:     %x\n(%v %v)\n", obj.Address().Bytes(), addr, vexp, v, vexp.Big(), v.Big())
// 			}
// 		}
// 	}

// 	root, _ := statedb.CommitTo(db, false)
// 	if common.HexToHash(test.PostStateRoot) != root {
// 		return fmt.Errorf("Post state root error. Expected: %s have: %x", test.PostStateRoot, root)
// 	}

// 	// check logs
// 	if test.Logs != "" {
// 		stateLogsHash := rlpHash(statedb.Logs())
// 		stLogsHash := rlpHash(stlogs)
// 		if stateLogsHash != stLogsHash {
// 			return fmt.Errorf("mismatch state/vm logs: state: %v vm: %v", stateLogsHash.Hex(), stLogsHash.Hex())
// 		}
// 		if stateLogsHash.Hex() != test.Logs && stLogsHash.Hex() != test.Logs {
// 			return fmt.Errorf("post state logs hash mismatch; got/state: %v got/vm: %v want: %v", stateLogsHash.Hex(), stLogsHash.Hex(), test.Logs)
// 		}
// 	}

// 	return nil
// }

// Subtests returns all valid subtests of the test.
func (t *StateTest) Subtests() []StateSubtest {
	var sub []StateSubtest
	for fork, pss := range t.json.Post {
		for i := range pss {
			sub = append(sub, StateSubtest{fork, i})
		}
	}
	return sub
}

// Run executes a specific subtest.
func (t *StateTest) Run(test *StateTest, subtest StateSubtest, ruleset RuleSet) (*state.StateDB, error) {
	db, _ := ethdb.NewMemDatabase()
	statedb := makePreState(db, t.json.Pre)

	env := NewEnv(ruleset, statedb)
	env.coinbase = test.json.Env.Coinbase
	pb, err := test.json.Env.PreviousHash.MarshalText()
	if err != nil {
		panic(err.Error())
	}
	env.parent = common.BytesToHash(pb)
	env.number = new(big.Int).SetUint64(test.json.Env.Number)
	env.time = new(big.Int).SetUint64(test.json.Env.Timestamp)
	env.difficulty = test.json.Env.Difficulty
	env.gasLimit = new(big.Int).SetUint64(test.json.Env.GasLimit)
	env.Gas = new(big.Int)
	env.evm = vm.New(env)

	post := t.json.Post[subtest.Fork][subtest.Index]

	// Derive sender from private key if present.
	var from common.Address
	if len(test.json.Tx.PrivateKey) > 0 {
		key := crypto.ToECDSA(test.json.Tx.PrivateKey)
		from = crypto.PubkeyToAddress(key.PublicKey)
	}
	// Parse recipient if present.
	var to *common.Address
	if test.json.Tx.To != "" {
		to = new(common.Address)
		if err := to.UnmarshalText([]byte(test.json.Tx.To)); err != nil {
			return nil, fmt.Errorf("invalid to address: %v", err)
		}
	}

	// Get values specific to this post state.
	if post.Indexes.Data > len(test.json.Tx.Data) {
		return nil, fmt.Errorf("tx data index %d out of bounds", post.Indexes.Data)
	}
	if post.Indexes.Value > len(test.json.Tx.Value) {
		return nil, fmt.Errorf("tx value index %d out of bounds", post.Indexes.Value)
	}
	if post.Indexes.Gas > len(test.json.Tx.GasLimit) {
		return nil, fmt.Errorf("tx gas limit index %d out of bounds", post.Indexes.Gas)
	}
	dataHex := test.json.Tx.Data[post.Indexes.Data]
	valueHex := test.json.Tx.Value[post.Indexes.Value]
	gasLimit := test.json.Tx.GasLimit[post.Indexes.Gas]
	// Value, Data hex encoding is messy: https://github.com/ethereum/tests/issues/203
	value := new(big.Int)
	if valueHex != "0x" {
		v, ok := math.ParseBig256(valueHex)
		if !ok {
			return nil, fmt.Errorf("invalid tx value %q", valueHex)
		}
		value = v
	}
	data, err := hex.DecodeString(strings.TrimPrefix(dataHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid tx data %q", dataHex)
	}

	// Set pre compiled contracts
	vm.Precompiled = vm.PrecompiledContracts()

	gaspool := new(core.GasPool)
	gaspool.AddGas(new(big.Int).SetUint64(test.json.Tx.GasLimit[post.Indexes.Gas]))
	snapshot := statedb.Snapshot()
	message := NewMessage(from, to, data, value, new(big.Int).SetUint64(gasLimit), test.json.Tx.GasPrice, test.json.Tx.Nonce)
	if _, _, _, err := core.ApplyMessage(env, message, gaspool); err != nil {
		statedb.RevertToSnapshot(snapshot)
	}
	if logs := rlpHash(statedb.Logs()); logs != common.Hash(post.Logs) {
		return statedb, fmt.Errorf("post state logs hash mismatch: got %x, want %x", logs, post.Logs)
	}
	root, _ := statedb.CommitTo(db, false)
	if root != common.Hash(post.Root) {
		return statedb, fmt.Errorf("post state root mismatch: got %x, want %x", root, post.Root)
	}
	return statedb, nil
}

func (t *StateTest) gasLimit(subtest StateSubtest) uint64 {
	return t.json.Tx.GasLimit[t.json.Post[subtest.Fork][subtest.Index].Indexes.Gas]
}

var Rules = map[string]RuleSet{
	"Frontier": {
		HomesteadBlock: big.NewInt(1000000),
	},
	"Homestead": {
		HomesteadBlock: new(big.Int),
	},
	"EIP150": {
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: new(big.Int),
	},
	"Byzantium": {
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: new(big.Int),
		DiehardBlock:             new(big.Int),
		ECIP1045Block:            new(big.Int),
	},
	"FrontierToHomesteadAt5": {
		HomesteadBlock: big.NewInt(5),
	},
	"HomesteadToEIP150At5": {
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(5),
	},
}

// UnsupportedForkError is returned when a test requests a fork that isn't implemented.
type UnsupportedForkError struct {
	Name string
}

func (e UnsupportedForkError) Error() string {
	return fmt.Sprintf("unsupported fork %q", e.Name)
}

// StateTest checks transaction processing without block context.
// See https://github.com/ethereum/EIPs/issues/176 for the test format specification.
type StateTest struct {
	json stJSON
}

// StateSubtest selects a specific configuration of a General State Test.
type StateSubtest struct {
	Fork  string
	Index int
}

func (t *StateTest) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.json)
}

// Genesis structs here only for tests
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
// type genesisSpecMarshaling struct {
// 	Nonce      math.HexOrDecimal64
// 	Timestamp  math.HexOrDecimal64
// 	ExtraData  hexutil.Bytes
// 	GasLimit   math.HexOrDecimal64
// 	GasUsed    math.HexOrDecimal64
// 	Number     math.HexOrDecimal64
// 	Difficulty *math.HexOrDecimal256
// 	Alloc      map[common.UnprefixedAddress]GenesisAccount
// }

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
	if _, err := hex.Decode(h[offset:], text); err != nil {
		fmt.Println(err)
		return fmt.Errorf("invalid hex storage key/value %q", text)
	}
	return nil
}

func (h storageJSON) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

type stJSON struct {
	Env  stEnv                    `json:"env"`
	Pre  GenesisAlloc             `json:"pre"`
	Tx   stTransaction            `json:"transaction"`
	Out  hexutil.Bytes            `json:"out"`
	Post map[string][]stPostState `json:"post"`
}

type stPostState struct {
	Root    common.UnprefixedHash `json:"hash"`
	Logs    common.UnprefixedHash `json:"logs"`
	Indexes struct {
		Data  int `json:"data"`
		Gas   int `json:"gas"`
		Value int `json:"value"`
	}
}

//go:generate gencodec -type stEnv -field-override stEnvMarshaling -out gen_stenv.go

type stEnv struct {
	PreviousHash common.UnprefixedHash `json:"previousHash"`
	Coinbase     common.Address        `json:"currentCoinbase"   gencodec:"required"`
	Difficulty   *big.Int              `json:"currentDifficulty" gencodec:"required"`
	GasLimit     uint64                `json:"currentGasLimit"   gencodec:"required"`
	Number       uint64                `json:"currentNumber"     gencodec:"required"`
	Timestamp    uint64                `json:"currentTimestamp"  gencodec:"required"`
}

type stEnvMarshaling struct {
	Coinbase   common.UnprefixedAddress
	Difficulty *math.HexOrDecimal256
	GasLimit   math.HexOrDecimal64
	Number     math.HexOrDecimal64
	Timestamp  math.HexOrDecimal64
}

//go:generate gencodec -type stTransaction -field-override stTransactionMarshaling -out gen_sttransaction.go

type stTransaction struct {
	GasPrice   *big.Int `json:"gasPrice"`
	Nonce      uint64   `json:"nonce"`
	To         string   `json:"to"`
	Data       []string `json:"data"`
	GasLimit   []uint64 `json:"gasLimit"`
	Value      []string `json:"value"`
	PrivateKey []byte   `json:"secretKey"`
}

type stTransactionMarshaling struct {
	GasPrice   *math.HexOrDecimal256
	Nonce      math.HexOrDecimal64
	GasLimit   []math.HexOrDecimal64
	PrivateKey hexutil.Bytes
}
