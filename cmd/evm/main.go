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

	"github.com/ethereumproject/go-ethereum/params"
	"gopkg.in/urfave/cli.v1"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/state"
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
	statedb.CreateAccount(common.StringToAddress("sender"))
	sender := statedb.GetOrNewStateObject(common.StringToAddress("sender"))

	valueFlag, _ := new(big.Int).SetString(ctx.GlobalString(ValueFlag.Name), 0)
	if valueFlag == nil {
		log.Fatalf("malformed %s flag value %q", ValueFlag.Name, ctx.GlobalString(ValueFlag.Name))
	}

	gasFlag, _ := new(big.Int).SetString(ctx.GlobalString(GasFlag.Name), 0)
	if gasFlag == nil {
		log.Fatalf("malformed %s flag value %q", GasFlag.Name, ctx.GlobalString(GasFlag.Name))
	}
	priceFlag, _ := new(big.Int).SetString(ctx.GlobalString(PriceFlag.Name), 0)
	if priceFlag == nil {
		log.Fatalf("malformed %s flag value %q", PriceFlag.Name, ctx.GlobalString(PriceFlag.Name))
	}

	cant := func(statedb vm.StateDB, a common.Address, i *big.Int) bool { return false }
	trans := func(statedb vm.StateDB, a common.Address, b common.Address, i *big.Int) {}
	gh := func(i uint64) common.Hash { return common.Hash{} }
	t := time.Now().Unix()
	vmctx := vm.Context{
		CanTransfer: cant,
		Transfer:    trans,
		GetHash:     gh,
		Origin:      sender.Address(),
		GasPrice:    priceFlag,
		Coinbase:    sender.Address(),
		GasLimit:    gasFlag.Uint64(),
		BlockNumber: common.Big1,
		Time:        big.NewInt(t),
		Difficulty:  core.CalcDifficulty(params.DefaultConfigMorden.ChainConfig, uint64(t), uint64(t-1), new(big.Int), new(big.Int).SetBytes(common.Hex2Bytes("0x020000"))),
	}

	evm := vm.NewEVM(vmctx, statedb, params.DefaultConfigMorden.ChainConfig, vm.Config{
		Debug:                   true,
		Tracer:                  nil,
		NoRecursion:             false,
		EnablePreimageRecording: false,
	})

	var (
		ret         []byte
		leftoverGas uint64
		err         error
	)

	tstart := time.Now()

	if ctx.GlobalBool(CreateFlag.Name) {
		input := append(common.Hex2Bytes(ctx.GlobalString(CodeFlag.Name)), common.Hex2Bytes(ctx.GlobalString(InputFlag.Name))...)
		ret, _, leftoverGas, err = evm.Create(sender, input, gasFlag.Uint64(), valueFlag)
	} else {
		statedb.CreateAccount(common.StringToAddress("receiver"))
		receiver := statedb.GetOrNewStateObject(common.StringToAddress("receiver"))

		code := common.Hex2Bytes(ctx.GlobalString(CodeFlag.Name))
		receiver.SetCode(crypto.Keccak256Hash(code), code)
		ret, leftoverGas, err = evm.Call(sender, receiver.Address(), common.Hex2Bytes(ctx.GlobalString(InputFlag.Name)), gasFlag.Uint64(), valueFlag)
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
	fmt.Printf("LEFTOVER GAS: %d", leftoverGas)
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
