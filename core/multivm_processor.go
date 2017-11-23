package core

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

type MultiVmFactory interface {
	Create(config *ChainConfig, header *types.Header, tx *types.Transaction) MultiVm
}

type MultiVm interface {
	CommitAccount(address common.Address, code []byte, nonce uint64, balance *big.Int)
	CommitAccountNonexist(address common.Address)
	CommitAccountCode(address common.Address, code []byte)
	CommitAccountStorage(address common.Address, key common.Hash, value common.Hash)
	CommitBlockhash(number uint64, hash common.Hash)
	Fire() *Require
	Status() byte
	Accounts() []AccountChange
	Logs() vm.Logs
	GasUsed() *big.Int
}

type IncreaseBalance struct {
	Address common.Address
	Balance *big.Int
}

type DecreaseBalance struct {
	Address common.Address
	Balance *big.Int
}

type Account struct {
	Address common.Address
	Nonce uint64
	Balance *big.Int
	Code []byte
	Storage map[common.Hash]common.Hash
	IsFullStorage bool
}

const (
	AccountChangeIncreaseBalance = iota
	AccountChangeDecreaseBalance
	AccountChangeAccount
	AccountChangeRemoved
)

type AccountChange struct {
	Type byte
	Value interface{}
}

type RequireAccount struct {
	Address common.Address
}

type RequireAccountStorage struct {
	Address common.Address
	Key common.Hash
}

type RequireBlockhash struct {
	Number uint64
}

const (
	RequireRequireAccount = iota
	RequireRequireAccountCode
	RequireRequireAccountStorage
	RequireRequireBlockhash
)

type Require struct {
	Type byte
	Value interface{}
}

const (
	StatusExitedOk = iota
	StatusExitedErr
	StatusExitedNotSupported
	StatusRunning
)

func ApplyMultiVmTransaction(factory *MultiVmFactory, config *ChainConfig, bc *BlockChain, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction) (*types.Receipt, vm.Logs, *big.Int, error) {
	tx.SetSigner(config.GetSigner(header.Number))

	vm := (*factory).Create(config, header, tx)
	for {
		ret := vm.Fire()
		if ret == nil {
			break
		}
		switch ret.Type {
		case RequireRequireAccount:
			value := ret.Value.(RequireAccount)
			if statedb.Exist(value.Address) {
				vm.CommitAccount(value.Address, statedb.GetCode(value.Address),
					statedb.GetNonce(value.Address), statedb.GetBalance(value.Address))
			} else {
				vm.CommitAccountNonexist(value.Address)
			}
		case RequireRequireAccountCode:
			value := ret.Value.(RequireAccount)
			if statedb.Exist(value.Address) {
				vm.CommitAccountCode(value.Address, statedb.GetCode(value.Address))
			} else {
				vm.CommitAccountNonexist(value.Address)
			}
		case RequireRequireAccountStorage:
			value := ret.Value.(RequireAccountStorage)
			if statedb.Exist(value.Address) {
				storageValue := statedb.GetState(value.Address, value.Key)
				vm.CommitAccountStorage(value.Address, value.Key, storageValue)
			} else {
				vm.CommitAccountNonexist(value.Address)
			}
		case RequireRequireBlockhash:
			value := ret.Value.(RequireBlockhash)
			block := bc.GetBlockByNumber(value.Number)
			vm.CommitBlockhash(value.Number, block.Header().Hash())
		}
	}

	// VM execution is finished at this point. We apply changes to the statedb.

	for _, account := range vm.Accounts() {
		switch account.Type {
		case AccountChangeIncreaseBalance:
			value := account.Value.(IncreaseBalance)
			statedb.AddBalance(value.Address, value.Balance)
		case AccountChangeDecreaseBalance:
			value := account.Value.(DecreaseBalance)
			balance := new(big.Int).Sub(statedb.GetBalance(value.Address), value.Balance)
			statedb.SetBalance(value.Address, balance)
		case AccountChangeRemoved:
			value := account.Value.(common.Address)
			statedb.Suicide(value)
		case AccountChangeAccount:
			value := account.Value.(Account)
			statedb.SetBalance(value.Address, value.Balance)
			statedb.SetNonce(value.Address, value.Nonce)
			statedb.SetCode(value.Address, value.Code)
			for storageKey, storageValue := range value.Storage {
				statedb.SetState(value.Address, storageKey, storageValue)
			}
		}
	}
	for _, log := range vm.Logs() {
		statedb.AddLog(log)
	}
	usedGas := vm.GasUsed()

	receipt := types.NewReceipt(statedb.IntermediateRoot().Bytes(), usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = new(big.Int).Set(usedGas)
	if MessageCreatesContract(tx) {
		from, _ := tx.From()
		receipt.ContractAddress = crypto.CreateAddress(from, tx.Nonce())
	}

	logs := statedb.GetLogs(tx.Hash())
	receipt.Logs = logs
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	glog.V(logger.Debug).Infoln(receipt)

	return receipt, logs, usedGas, nil
}
