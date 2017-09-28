package tests

import (
	"testing"
	"path/filepath"
	"fmt"
	"strconv"
	"bytes"
	"math/big"

	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/machine/classic"
	"github.com/ethereumproject/go-ethereum/core/state"
)

type VmEnv struct {
	CurrentCoinbase   string
	CurrentDifficulty string
	CurrentGasLimit   string
	CurrentNumber     string
	CurrentTimestamp  interface{}
	PreviousHash      string
}

type VmTest struct {
	Callcreates interface{}
	Env           VmEnv
	Exec          map[string]string
	Transaction   map[string]string
	Logs          []Log
	Gas           string
	Out           string
	Post          map[string]Account
	Pre           map[string]Account
	PostStateRoot string
}

type RuleSet struct {
	HomesteadBlock           *big.Int
	HomesteadGasRepriceBlock *big.Int
	DiehardBlock             *big.Int
	ExplosionBlock           *big.Int
}

type Env struct {
	ruleSet      RuleSet
	depth        int
	state        *state.StateDB
	skipTransfer bool
	initial      bool
	Gas          *big.Int

	origin   common.Address
	parent   common.Hash
	coinbase common.Address

	number     *big.Int
	time       *big.Int
	difficulty *big.Int
	gasLimit   *big.Int

	vmTest bool

	evm vm.VirtualMachine
}

func (self *Env) Call(me vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	self.evm.Start(me.Address(), addr, data, gas, price, value)
	defer self.evm.Finish()
	for {
		err := self.evm.Fire()
		if err != nil {
			if e, ok := err.(*vm.RequireAccountError); ok {
				acc := self.state.GetAccount(e.Address)
				self.evm.CommitAccount(acc)
			} else if e, ok := err.(*vm.RequireHashError); ok {
				hash := common.Hash{}
				self.evm.CommitBlockhash(e.Number,hash)
			} else if e, ok := err.(*vm.RequireCodeError); ok {

			} else {
				return nil,e
			}
		} else {
			out, err := self.evm.Out()
			return out, err
		}
	}
}

func NewEnv(ruleSet RuleSet, state *state.StateDB) *Env {
	env := &Env{
		ruleSet: ruleSet,
		state:   state,
	}
	return env
}

func NewEnvFromMap(ruleSet RuleSet, state *state.StateDB, envValues map[string]string, exeValues map[string]string) *Env {
	env := NewEnv(ruleSet, state)

	env.origin = common.HexToAddress(exeValues["caller"])
	env.parent = common.HexToHash(envValues["previousHash"])
	env.coinbase = common.HexToAddress(envValues["currentCoinbase"])
	env.number, _ = new(big.Int).SetString(envValues["currentNumber"], 0)
	if env.number == nil {
		panic("malformed current number")
	}
	env.time, _ = new(big.Int).SetString(envValues["currentTimestamp"], 0)
	if env.time == nil {
		panic("malformed current timestamp")
	}
	env.difficulty, _ = new(big.Int).SetString(envValues["currentDifficulty"], 0)
	if env.difficulty == nil {
		panic("malformed current difficulty")
	}
	env.gasLimit, _ = new(big.Int).SetString(envValues["currentGasLimit"], 0)
	if env.gasLimit == nil {
		panic("malformed current gas limit")
	}

	env.Gas = new(big.Int)
	env.evm = classic.AdaptVM()

	return env
}

func RunVmTest(p string, skipTests []string) (err error) {
	tests := make(map[string]VmTest)

	err = readJsonFile(p, &tests)
	if err != nil {
		return err
	}

	err = runVmTests(tests, skipTests)
	return err
}

func runVmTests(tests map[string]VmTest, skipTests []string) error {
	skipTest := make(map[string]bool, len(skipTests))

	for _, name := range skipTests {
		skipTest[name] = true
	}

	for name, test := range tests {
		if skipTest[name] {
			glog.Infoln("Skipping VM test", name)
			continue
		}

		if err := runVmTest(test); err != nil {
			return fmt.Errorf("%s %s", name, err.Error())
		}

		glog.Infoln("VM test passed: ", name)
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
	classic.Precompiled = make(map[string]*classic.PrecompiledAccount)

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

func TestVMArithmetic(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmArithmeticTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestBitwiseLogicOperation(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmBitwiseLogicOperationTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestBlockInfo(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmBlockInfoTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEnvironmentalInfo(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmEnvironmentalInfoTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestFlowOperation(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmIOandFlowOperationsTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestLogTest(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmLogTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestPerformance(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmPerformanceTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestPushDupSwap(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmPushDupSwapTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestVMSha3(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmSha3Test.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestVm(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmtests.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestVmLog(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmLogTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestInputLimits(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmInputLimits.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestInputLimitsLight(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmInputLimitsLight.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestVMRandom(t *testing.T) {
	fns, _ := filepath.Glob(filepath.Join(baseDir, "RandomTests", "*"))
	for _, fn := range fns {
		if err := RunVmTest(fn, VmSkipTests); err != nil {
			t.Error(err)
		}
	}
}
