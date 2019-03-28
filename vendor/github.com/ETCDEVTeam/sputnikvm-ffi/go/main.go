package main

import (
	"fmt"
	"math/big"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/sputnikvm-ffi/go/sputnikvm"
)

func main() {
	goint := big.NewInt(100000000000000)
	cint := sputnikvm.ToCU256(goint)
	fmt.Printf("%v\n", cint)
	sputnikvm.PrintCU256(cint)

	transaction := sputnikvm.Transaction {
		Caller: *new(common.Address),
		GasPrice: new(big.Int),
		GasLimit: new(big.Int).SetUint64(1000000000),
		Address: new(common.Address),
		Value: new(big.Int),
		Input: []byte{1, 2, 3, 4, 5},
		Nonce: new(big.Int),
	}

	header := sputnikvm.HeaderParams {
		Beneficiary: *new(common.Address),
		Timestamp: 0,
		Number: new(big.Int),
		Difficulty: new(big.Int),
		GasLimit: new(big.Int),
	}

	vm := sputnikvm.NewFrontier(&transaction, &header)

Loop:
	for {
		require := vm.Fire()
		fmt.Printf("%v\n", require)
		switch require.Typ() {
		case sputnikvm.RequireNone:
			break Loop
		case sputnikvm.RequireAccount, sputnikvm.RequireAccountCode:
			vm.CommitNonexist(require.Address())
		case sputnikvm.RequireAccountStorage:
			panic("unreachable")
		case sputnikvm.RequireBlockhash:
			vm.CommitBlockhash(require.BlockNumber(), *new(common.Hash))
		default:
			panic("unreachable")
		}
	}
	fmt.Printf("%v\n", vm.UsedGas())
	fmt.Printf("%v\n", vm.Logs())
	fmt.Printf("%v\n", vm.AccountChanges())
	vm.Free()
}
