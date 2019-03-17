// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// evm executes EVM code snippets.
package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/urfave/cli.v1"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

// Version is the application revision identifier. It can be set with the linker
// as in: go build -ldflags "-X main.Version="`git describe --tags`
var Version = "unknown"

var (
	DebugFlag = cli.BoolFlag{
		Name:  "debug",
		Usage: "output full trace logs",
	}
	ForceJitFlag = cli.BoolFlag{
		Name:  "forcejit",
		Usage: "forces jit compilation",
	}
	DisableJitFlag = cli.BoolFlag{
		Name:  "nojit",
		Usage: "disabled jit compilation",
	}
	CodeFlag = cli.StringFlag{
		Name:  "code",
		Usage: "EVM code",
	}
	GasFlag = cli.StringFlag{
		Name:  "gas",
		Usage: "gas limit for the evm",
		Value: "10000000000",
	}
	PriceFlag = cli.StringFlag{
		Name:  "price",
		Usage: "price set for the evm",
		Value: "0",
	}
	ValueFlag = cli.StringFlag{
		Name:  "value",
		Usage: "value set for the evm",
		Value: "0",
	}
	DumpFlag = cli.BoolFlag{
		Name:  "dump",
		Usage: "dumps the state after the run",
	}
	InputFlag = cli.StringFlag{
		Name:  "input",
		Usage: "input for the EVM",
	}
	SysStatFlag = cli.BoolFlag{
		Name:  "sysstat",
		Usage: "display system stats",
	}
	VerbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "sets the verbosity level",
	}
	CreateFlag = cli.BoolFlag{
		Name:  "create",
		Usage: "indicates the action should be create rather than call",
	}
)

var app *cli.App

func init() {
	app = cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Version = Version
	app.Usage = "the evm command line interface"
	app.Action = run
	app.Flags = []cli.Flag{
		CreateFlag,
		DebugFlag,
		VerbosityFlag,
		ForceJitFlag,
		DisableJitFlag,
		SysStatFlag,
		CodeFlag,
		GasFlag,
		PriceFlag,
		ValueFlag,
		DumpFlag,
		InputFlag,
	}
}

func run(ctx *cli.Context) error {
	glog.SetToStderr(true)
	glog.SetV(ctx.GlobalInt(VerbosityFlag.Name))

	db, _ := ethdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	sender := statedb.CreateAccount(common.StringToAddress("sender"))

	valueFlag, _ := new(big.Int).SetString(ctx.GlobalString(ValueFlag.Name), 0)
	if valueFlag == nil {
		log.Fatalf("malformed %s flag value %q", ValueFlag.Name, ctx.GlobalString(ValueFlag.Name))
	}
	vmenv := NewEnv(statedb, common.StringToAddress("evmuser"), valueFlag)

	tstart := time.Now()

	var (
		ret []byte
		err error
	)

	gasFlag, _ := new(big.Int).SetString(ctx.GlobalString(GasFlag.Name), 0)
	if gasFlag == nil {
		log.Fatalf("malformed %s flag value %q", GasFlag.Name, ctx.GlobalString(GasFlag.Name))
	}
	priceFlag, _ := new(big.Int).SetString(ctx.GlobalString(PriceFlag.Name), 0)
	if priceFlag == nil {
		log.Fatalf("malformed %s flag value %q", PriceFlag.Name, ctx.GlobalString(PriceFlag.Name))
	}

	if ctx.GlobalBool(CreateFlag.Name) {
		input := append(common.Hex2Bytes(ctx.GlobalString(CodeFlag.Name)), common.Hex2Bytes(ctx.GlobalString(InputFlag.Name))...)
		ret, _, err = vmenv.Create(sender, input, gasFlag, priceFlag, valueFlag)
	} else {
		receiver := statedb.CreateAccount(common.StringToAddress("receiver"))

		code := common.Hex2Bytes(ctx.GlobalString(CodeFlag.Name))
		receiver.SetCode(crypto.Keccak256Hash(code), code)
		ret, err = vmenv.Call(sender, receiver.Address(), common.Hex2Bytes(ctx.GlobalString(InputFlag.Name)), gasFlag, priceFlag, valueFlag)
	}
	vmdone := time.Since(tstart)

	if ctx.GlobalBool(DumpFlag.Name) {
		statedb.CommitTo(db, false)
		fmt.Println(string(statedb.Dump([]common.Address{})))
	}

	if ctx.GlobalBool(SysStatFlag.Name) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		fmt.Printf("vm took %v\n", vmdone)
		fmt.Printf(`alloc:      %d
tot alloc:  %d
no. malloc: %d
heap alloc: %d
heap objs:  %d
num gc:     %d
`, mem.Alloc, mem.TotalAlloc, mem.Mallocs, mem.HeapAlloc, mem.HeapObjects, mem.NumGC)
	}

	fmt.Printf("OUT: 0x%x", ret)
	if err != nil {
		fmt.Printf(" error: %v", err)
	}
	fmt.Println()
	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type VMEnv struct {
	state *state.StateDB
	block *types.Block

	transactor *common.Address
	value      *big.Int

	depth int
	Gas   *big.Int
	time  *big.Int

	evm *vm.EVM
}

func NewEnv(state *state.StateDB, transactor common.Address, value *big.Int) *VMEnv {
	env := &VMEnv{
		state:      state,
		transactor: &transactor,
		value:      value,
		time:       big.NewInt(time.Now().Unix()),
	}

	env.evm = vm.New(env)
	return env
}

// ruleSet implements vm.RuleSet and will always default to the homestead rule set.
type ruleSet struct{}

func (ruleSet) IsHomestead(*big.Int) bool { return true }

func (ruleSet) GasTable(*big.Int) *vm.GasTable {
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

func (self *VMEnv) RuleSet() vm.RuleSet       { return ruleSet{} }
func (self *VMEnv) Vm() vm.Vm                 { return self.evm }
func (self *VMEnv) Db() vm.Database           { return self.state }
func (self *VMEnv) SnapshotDatabase() int     { return self.state.Snapshot() }
func (self *VMEnv) RevertToSnapshot(snap int) { self.state.RevertToSnapshot(snap) }
func (self *VMEnv) Origin() common.Address    { return *self.transactor }
func (self *VMEnv) BlockNumber() *big.Int     { return new(big.Int) }
func (self *VMEnv) Coinbase() common.Address  { return *self.transactor }
func (self *VMEnv) Time() *big.Int            { return self.time }
func (self *VMEnv) Difficulty() *big.Int      { return common.Big1 }
func (self *VMEnv) BlockHash() []byte         { return make([]byte, 32) }
func (self *VMEnv) Value() *big.Int           { return self.value }
func (self *VMEnv) GasLimit() *big.Int        { return big.NewInt(1000000000) }
func (self *VMEnv) VmType() vm.Type           { return vm.StdVmTy }
func (self *VMEnv) Depth() int                { return 0 }
func (self *VMEnv) SetDepth(i int)            { self.depth = i }
func (self *VMEnv) GetHash(n uint64) common.Hash {
	if self.block.Number().Cmp(big.NewInt(int64(n))) == 0 {
		return self.block.Hash()
	}
	return common.Hash{}
}
func (self *VMEnv) AddLog(log *vm.Log) {
	self.state.AddLog(*log)
}
func (self *VMEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	return self.state.GetBalance(from).Cmp(balance) >= 0
}
func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) {
	core.Transfer(from, to, amount)
}

func (self *VMEnv) Call(caller vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	self.Gas = gas
	return core.Call(self, caller, addr, data, gas, price, value)
}

func (self *VMEnv) CallCode(caller vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return core.CallCode(self, caller, addr, data, gas, price, value)
}

func (self *VMEnv) DelegateCall(caller vm.ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return core.DelegateCall(self, caller, addr, data, gas, price)
}

func (self *VMEnv) Create(caller vm.ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return core.Create(self, caller, data, gas, price, value)
}
