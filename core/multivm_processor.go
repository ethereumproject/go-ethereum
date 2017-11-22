package core

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
)

type MultiVmFactory interface {
	Create(config *ChainConfig, header *types.Header, tx *types.Transaction) MultiVm
}

type MultiVm interface {
	CommitAccount(address common.Address, nonce uint64, balance *big.Int) error
	CommitAccountCode(address common.Address, code []byte) error
	CommitAccountStorage(address common.Address, key *big.Int, value *big.Int) error
	CommitBlockhash(number uint64, hash common.Hash) error
	Fire() Require
	Status() byte
	Accounts() []AccountChange
	Logs() []vm.Logs
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
	Storage map[*big.Int]*big.Int
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
	Key *big.Int
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
