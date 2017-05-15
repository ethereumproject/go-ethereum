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
	"fmt"
	"io"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

func RunVmTestWithReader(r io.Reader, skipTests []string) error {
	tests := make(map[string]VmTest)
	err := readJson(r, &tests)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	if err := runVmTests(tests, skipTests); err != nil {
		return err
	}

	return nil
}

type bconf struct {
	name    string
	precomp bool
	jit     bool
}

func BenchVmTest(p string, conf bconf, b *testing.B) error {
	tests := make(map[string]VmTest)
	err := readJsonFile(p, &tests)
	if err != nil {
		return err
	}

	test, ok := tests[conf.name]
	if !ok {
		return fmt.Errorf("test not found: %s", conf.name)
	}

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
		benchVmTest(test, env, b)
	}

	return nil
}

func benchVmTest(test VmTest, env map[string]string, b *testing.B) {
	b.StopTimer()
	db, _ := ethdb.NewMemDatabase()
	statedb := makePreState(db, test.Pre)
	b.StartTimer()

	RunVm(statedb, env, test.Exec)
}

func RunVmTest(p string, skipTests []string) error {
	tests := make(map[string]VmTest)
	err := readJsonFile(p, &tests)
	if err != nil {
		return err
	}

	if err := runVmTests(tests, skipTests); err != nil {
		return err
	}

	return nil
}

func runVmTests(tests map[string]VmTest, skipTests []string) error {
	skipTest := make(map[string]bool, len(skipTests))
	for _, name := range skipTests {
		skipTest[name] = true
	}

	for name, test := range tests {
		if skipTest[name] {
			glog.Infoln("Skipping VM test", name)
			return nil
		}

		if err := runVmTest(test); err != nil {
			return fmt.Errorf("%s %s", name, err.Error())
		}

		glog.Infoln("VM test passed: ", name)
		//fmt.Println(string(statedb.Dump()))
	}
	return nil
}

func runVmTest(test VmTest) error {
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

	ret, logs, gas, err := RunVm(statedb, env, test.Exec)

	// Compare expected and actual return
	rexp := common.FromHex(test.Out)
	if bytes.Compare(rexp, ret) != 0 {
		return fmt.Errorf("return failed. Expected %x, got %x\n", rexp, ret)
	}

	// Check gas usage
	if test.Gas == "" && err == nil {
		return fmt.Errorf("gas unspecified, indicating an error. VM returned (incorrectly) successfull")
	} else {
		want, ok := new(big.Int).SetString(test.Gas, 0)
		if test.Gas == "" {
			want = new(big.Int)
		} else if !ok {
			return fmt.Errorf("malformed test gas %q", test.Gas)
		}
		if want.Cmp(gas) != 0 {
			return fmt.Errorf("gas failed. Expected %v, got %v\n", want, gas)
		}
	}

	// check post state
	for addr, account := range test.Post {
		obj := statedb.GetStateObject(common.HexToAddress(addr))
		if obj == nil {
			continue
		}
		for addr, value := range account.Storage {
			v := statedb.GetState(obj.Address(), common.HexToHash(addr))
			vexp := common.HexToHash(value)
			if v != vexp {
				return fmt.Errorf("(%x: %s) storage failed. Expected %x, got %x (%v %v)\n", obj.Address().Bytes()[0:4], addr, vexp, v, vexp.Big(), v.Big())
			}
		}
	}

	// check logs
	if len(test.Logs) > 0 {
		lerr := checkLogs(test.Logs, logs)
		if lerr != nil {
			return lerr
		}
	}

	return nil
}

func RunVm(state *state.StateDB, env, exec map[string]string) ([]byte, vm.Logs, *big.Int, error) {
	var (
		to       = common.HexToAddress(exec["address"])
		from     = common.HexToAddress(exec["caller"])
		data     = common.FromHex(exec["data"])
		gas, _   = new(big.Int).SetString(exec["gas"], 0)
		price, _ = new(big.Int).SetString(exec["gasPrice"], 0)
		value, _ = new(big.Int).SetString(exec["value"], 0)
	)
	if gas == nil || price == nil || value == nil {
		panic("malformed gas, price or value")
	}
	// Reset the pre-compiled contracts for VM tests.
	vm.Precompiled = make(map[string]*vm.PrecompiledAccount)

	caller := state.GetOrNewStateObject(from)

	vmenv := NewEnvFromMap(RuleSet{
		HomesteadBlock:           big.NewInt(1150000),
		HomesteadGasRepriceBlock: big.NewInt(2500000),
		DiehardBlock:             big.NewInt(3000000),
		ExplosionBlock:           big.NewInt(5000000),
	}, state, env, exec)
	vmenv.vmTest = true
	vmenv.skipTransfer = true
	vmenv.initial = true
	ret, err := vmenv.Call(caller, to, data, gas, price, value)

	return ret, vmenv.state.Logs(), vmenv.Gas, err
}
