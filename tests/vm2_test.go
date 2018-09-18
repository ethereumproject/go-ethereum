package tests

// This file uses v2 of the JSON tests schema, where 'logs' field is a single string (rlpHash) of the statedb logs, instead of an array of type vm.Logs.

import (
	"bytes"
	"fmt"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"math/big"
	"path/filepath"
	"strconv"
	"testing"
)

// VmTests2 describes the "new" JSON test schema for VM tests, described in ethereum/tests,
// and not backwards compatible. The primary difference is that the 'Logs' field is the RLP hash of the state logs,
// instead of an array.
type VmTest2 struct {
	Callcreates interface{}
	//Env         map[string]string
	Env           VmEnv
	Exec          map[string]string
	Transaction   map[string]string
	Logs          string
	Gas           string
	Out           string
	Post          map[string]Account
	Pre           map[string]Account
	PostStateRoot string
}

func TestSputnikVMTests(t *testing.T) {
	fns, _ := filepath.Glob(filepath.Join(vmTestDir, "SputnikVM", "vmBitwiseLogicOperation", "*"))
	for _, fn := range fns {
		if err := RunVmTest2(fn, VmSkipTests); err != nil {
			t.Error(err)
		}
	}
}

// RunVmTest2 reads input JSON and runs associated test.
func RunVmTest2(p string, skipTests []string) error {
	tests := make(map[string]VmTest2)
	err := readJsonFile(p, &tests)
	if err != nil {
		return err
	}

	if err := runVmTests2(tests, skipTests); err != nil {
		return err
	}

	return nil
}

func runVmTests2(tests map[string]VmTest2, skipTests []string) error {
	skipTest := make(map[string]bool, len(skipTests))
	for _, name := range skipTests {
		skipTest[name] = true
	}

	for name, test := range tests {
		if skipTest[name] {
			glog.Infoln("Skipping VM test", name)
			return nil
		}

		if err := runVmTest2(test); err != nil {
			return fmt.Errorf("%s %s", name, err.Error())
		}

		glog.Infoln("VM test passed: ", name)
		//fmt.Println(string(statedb.Dump()))
	}
	return nil
}

func runVmTest2(test VmTest2) error {
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

	ret, vmlogs, gas, err := RunVm2(statedb, env, test.Exec)

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
		obj := statedb.GetAccount(common.HexToAddress(addr))
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
	if test.Logs != "" {
		stateLogsHash := rlpHash(statedb.Logs())
		vmLogsHash := rlpHash(vmlogs)
		if stateLogsHash != vmLogsHash {
			return fmt.Errorf("mismatch state/vm logs: state: %v vm: %v", stateLogsHash.Hex(), vmLogsHash.Hex())
		}
		if stateLogsHash.Hex() != test.Logs && vmLogsHash.Hex() != test.Logs {
			return fmt.Errorf("post state logs hash mismatch; got/state: %v got/vm: %v want: %v", stateLogsHash.Hex(), vmLogsHash.Hex(), test.Logs)
		}
	}

	return nil
}

// RunVm2 is the same as RunVm, except that it configures a RuleSet definition that includes ECIP1045* forks.
// This is an example of terrible code factoring, since the RuleSet (ie ChainConfig) params are _hardcoded_
// inside of the function. Kids, if you're watching at home, don't do this.
func RunVm2(state *state.StateDB, env, exec map[string]string) ([]byte, vm.Logs, *big.Int, error) {
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
	vm.PrecompiledHomestead = make(map[string]*vm.PrecompiledAccount)

	caller := state.GetOrNewStateObject(from)

	vmenv := NewEnvFromMap(RuleSet{
		HomesteadBlock:           big.NewInt(0),
		HomesteadGasRepriceBlock: big.NewInt(0),
		DiehardBlock:             big.NewInt(0),
		ExplosionBlock:           big.NewInt(0),
		ECIP1045BBlock:           big.NewInt(0),
		ECIP1045CBlock:           big.NewInt(0),
	}, state, env, exec)
	vmenv.vmTest = true
	vmenv.skipTransfer = true
	vmenv.initial = true
	ret, err := vmenv.Call(caller, to, data, gas, price, value)

	return ret, vmenv.state.Logs(), gas, err
}
