// Copyright 2015 The go-ethereum Authors
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
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

func RunStateTestWithReader(ruleSet RuleSet, r io.Reader, skipTests []string) error {
	tests := make(map[string]VmTest)
	if err := readJson(r, &tests); err != nil {
		return err
	}

	if err := runStateTests(ruleSet, tests, skipTests); err != nil {
		return err
	}

	return nil
}

func RunStateTest(ruleSet RuleSet, p string, skipTests []string) error {
	tests := make(map[string]VmTest)
	if err := readJsonFile(p, &tests); err != nil {
		return err
	}

	if err := runStateTests(ruleSet, tests, skipTests); err != nil {
		return err
	}

	return nil

}

func BenchStateTest(ruleSet RuleSet, p string, conf bconf, b *testing.B) error {
	tests := make(map[string]VmTest)
	if err := readJsonFile(p, &tests); err != nil {
		return err
	}
	test, ok := tests[conf.name]
	if !ok {
		return fmt.Errorf("test not found: %s", conf.name)
	}

	// XXX Yeah, yeah...
	env := make(map[string]string)
	env["currentCoinbase"] = test.Env.CurrentCoinbase
	env["currentDifficulty"] = test.Env.CurrentDifficulty
	env["currentGasLimit"] = test.Env.CurrentGasLimit
	env["currentNumber"] = test.Env.CurrentNumber
	env["previousHash"] = test.Env.PreviousHash
	if n, ok := test.Env.CurrentTimestamp.(float64); ok {
		env["currentTimestamp"] = strconv.Itoa(int(n))
	} else {
		env["currentTimestamp"] = test.Env.CurrentTimestamp.(string)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchStateTest(ruleSet, test, env, b)
	}

	return nil
}

func benchStateTest(ruleSet RuleSet, test VmTest, env map[string]string, b *testing.B) {
	b.StopTimer()
	db, _ := ethdb.NewMemDatabase()
	statedb := makePreState(db, test.Pre)
	b.StartTimer()

	RunState(ruleSet, statedb, env, test.Exec)
}

func runStateTests(ruleSet RuleSet, tests map[string]VmTest, skipTests []string) error {
	skipTest := make(map[string]bool, len(skipTests))
	for _, name := range skipTests {
		skipTest[name] = true
	}

	for name, test := range tests {
		if skipTest[name] /*|| name != "callcodecallcode_11" */ {
			glog.Infoln("Skipping state test", name)
			continue
		}

		//fmt.Println("StateTest:", name)
		if err := runStateTest(ruleSet, test); err != nil {
			return fmt.Errorf("%s: %s\n", name, err.Error())
		}

		//glog.Infoln("State test passed: ", name)
		//fmt.Println(string(statedb.Dump()))
	}
	return nil

}

func runStateTest(ruleSet RuleSet, test VmTest) error {
	db, _ := ethdb.NewMemDatabase()
	statedb := makePreState(db, test.Pre)

	// XXX Yeah, yeah...
	env := make(map[string]string)
	env["currentCoinbase"] = test.Env.CurrentCoinbase
	env["currentDifficulty"] = test.Env.CurrentDifficulty
	env["currentGasLimit"] = test.Env.CurrentGasLimit
	env["currentNumber"] = test.Env.CurrentNumber
	env["previousHash"] = test.Env.PreviousHash
	if n, ok := test.Env.CurrentTimestamp.(float64); ok {
		env["currentTimestamp"] = strconv.Itoa(int(n))
	} else {
		env["currentTimestamp"] = test.Env.CurrentTimestamp.(string)
	}

	var (
		ret []byte
		// gas  *big.Int
		// err  error
		logs vm.Logs
	)

	ret, logs, _, _ = RunState(ruleSet, statedb, env, test.Transaction)

	// Compare expected and actual return
	rexp := common.FromHex(test.Out)
	if bytes.Compare(rexp, ret) != 0 {
		return fmt.Errorf("return failed. Expected %x, got %x\n", rexp, ret)
	}

	// check post state
	for addr, account := range test.Post {
		obj := statedb.GetStateObject(common.HexToAddress(addr))
		if obj == nil {
			return fmt.Errorf("did not find expected post-state account: %s", addr)
		}

		if balance, ok := new(big.Int).SetString(account.Balance, 0); !ok {
			panic("malformed test account balance")
		} else if balance.Cmp(obj.Balance()) != 0 {
			return fmt.Errorf("(%x) balance failed. Expected: %v have: %v\n", obj.Address().Bytes()[:4], account.Balance, obj.Balance())
		}

		if nonce, err := strconv.ParseUint(account.Nonce, 0, 64); err != nil {
			return fmt.Errorf("test account %q malformed nonce: %s", addr, err)
		} else if obj.Nonce() != nonce {
			return fmt.Errorf("(%x) nonce failed. Expected: %v have: %v\n", obj.Address().Bytes()[:4], account.Nonce, obj.Nonce())
		}

		for addr, value := range account.Storage {
			v := statedb.GetState(obj.Address(), common.HexToHash(addr))
			vexp := common.HexToHash(value)

			if v != vexp {
				return fmt.Errorf("storage failed:\n%x: %s:\nexpected: %x\nhave:     %x\n(%v %v)\n", obj.Address().Bytes(), addr, vexp, v, vexp.Big(), v.Big())
			}
		}
	}

	root, _ := statedb.Commit()
	if common.HexToHash(test.PostStateRoot) != root {
		return fmt.Errorf("Post state root error. Expected: %s have: %x", test.PostStateRoot, root)
	}

	// check logs
	if len(test.Logs) > 0 {
		if err := checkLogs(test.Logs, logs); err != nil {
			return err
		}
	}

	return nil
}

func RunState(ruleSet RuleSet, statedb *state.StateDB, env, tx map[string]string) ([]byte, vm.Logs, *big.Int, error) {
	data := common.FromHex(tx["data"])
	gas, _ := new(big.Int).SetString(tx["gasLimit"], 0)
	price, _ := new(big.Int).SetString(tx["gasPrice"], 0)
	value, _ := new(big.Int).SetString(tx["value"], 0)
	if gas == nil || price == nil || value == nil {
		panic("malformed gas, price or value")
	}
	nonce, err := strconv.ParseUint(tx["nonce"], 0, 64)
	if err != nil {
		panic(err)
	}

	var to *common.Address
	if len(tx["to"]) > 2 {
		t := common.HexToAddress(tx["to"])
		to = &t
	}
	// Set pre compiled contracts
	vm.Precompiled = vm.PrecompiledContracts()
	snapshot := statedb.Snapshot()
	currentGasLimit, ok := new(big.Int).SetString(env["currentGasLimit"], 0)
	if !ok {
		panic("malformed currentGasLimit")
	}
	gaspool := new(core.GasPool).AddGas(currentGasLimit)

	key, err := hex.DecodeString(tx["secretKey"])
	if err != nil {
		panic(err)
	}
	addr := crypto.PubkeyToAddress(crypto.ToECDSA(key).PublicKey)
	message := NewMessage(addr, to, data, value, gas, price, nonce)
	vmenv := NewEnvFromMap(ruleSet, statedb, env, tx)
	vmenv.origin = addr
	ret, _, err := core.ApplyMessage(vmenv, message, gaspool)
	if core.IsNonceErr(err) || core.IsInvalidTxErr(err) || core.IsGasLimitErr(err) {
		statedb.RevertToSnapshot(snapshot)
	}
	statedb.Commit()

	return ret, vmenv.state.Logs(), vmenv.Gas, err
}
