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
	"os"
	"os/signal"
	"syscall"
	"runtime"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/core"
)

func init() {
	sigChan := make(chan os.Signal)
	go func() {
		stacktrace := make([]byte, 8192)
		for _ = range sigChan {
			length := runtime.Stack(stacktrace, true)
			fmt.Println(string(stacktrace[:length]))
			syscall.Exit(-1)
		}
	}()
	signal.Notify(sigChan, syscall.SIGINT)
}

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

func (r RuleSet) IsHomestead(n *big.Int) bool {
	return n.Cmp(r.HomesteadBlock) >= 0
}
func (r RuleSet) GasTable(num *big.Int) *vm.GasTable {
	if r.HomesteadGasRepriceBlock == nil || num == nil || num.Cmp(r.HomesteadGasRepriceBlock) < 0 {
		return &vm.GasTable{
			ExtcodeSize:     big.NewInt(20),
			ExtcodeCopy:     big.NewInt(20),
			Balance:         big.NewInt(20),
			SLoad:           big.NewInt(50),
			Calls:           big.NewInt(40),
			Suicide:         big.NewInt(0),
			ExpByte:         big.NewInt(10),
			CreateBySuicide: nil,
		}
	}
	if r.DiehardBlock == nil || num == nil || num.Cmp(r.DiehardBlock) < 0 {
		return &vm.GasTable{
			ExtcodeSize:     big.NewInt(700),
			ExtcodeCopy:     big.NewInt(700),
			Balance:         big.NewInt(400),
			SLoad:           big.NewInt(200),
			Calls:           big.NewInt(700),
			Suicide:         big.NewInt(5000),
			ExpByte:         big.NewInt(10),
			CreateBySuicide: big.NewInt(25000),
		}
	}

	return &vm.GasTable{
		ExtcodeSize:     big.NewInt(700),
		ExtcodeCopy:     big.NewInt(700),
		Balance:         big.NewInt(400),
		SLoad:           big.NewInt(200),
		Calls:           big.NewInt(700),
		Suicide:         big.NewInt(5000),
		ExpByte:         big.NewInt(50),
		CreateBySuicide: big.NewInt(25000),
	}
}

func NewEnvFromMap(ruleSet RuleSet, db *state.StateDB, envValues map[string]string, exeValues map[string]string) *core.VmEnv {

	number, _ := new(big.Int).SetString(envValues["currentNumber"], 0)
	if number == nil {
		panic("malformed current number")
	}
	time, _ := new(big.Int).SetString(envValues["currentTimestamp"], 0)
	if time == nil {
		panic("malformed current timestamp")
	}
	difficulty, _ := new(big.Int).SetString(envValues["currentDifficulty"], 0)
	if difficulty == nil {
		panic("malformed current difficulty")
	}
	gasLimit, _ := new(big.Int).SetString(envValues["currentGasLimit"], 0)
	if gasLimit == nil {
		panic("malformed current gas limit")
	}
	machine, _ := core.VmManager.ConnectMachine()
	if machine == nil {
		panic("could not connect machine")
	}

	fork := vm.Frontier
	if ruleSet.IsHomestead(number) {
		fork = vm.Homestead
	}

	funcFn := func(n uint64) common.Hash {
		return common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(n)).String())))
	}

	env := &core.VmEnv{
		common.HexToAddress(envValues["currentCoinbase"]),
		number,
		difficulty,
		gasLimit,
		time,
		db,
		funcFn,
		fork,
		ruleSet,
		machine,
	}

	//env.origin = common.HexToAddress(exeValues["caller"])
	//env.parent = common.HexToHash(envValues["previousHash"])
	//env.coinbase = common.HexToAddress(envValues["currentCoinbase"])

	machine.SetTestFeatures(vm.AllTestFeatures)

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

func RunVm(state *state.StateDB, env, exec map[string]string) ([]byte, state.Logs, *big.Int, error) {
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

	ret, err := vmenv.Call(caller.Address(), to, data, gas, price, value)
	return ret, vmenv.Db.Logs(), gas, err
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
