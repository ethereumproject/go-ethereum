// +build sputnikvm

package core

import (
	"math/big"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/params"
	"github.com/ethereumproject/sputnikvm-ffi/go/sputnikvm"
)

const SputnikVMExists = true

var UseSputnikVM = false

// Apply a transaction using the SputnikVM processor with the given
// chain config and state. Note that we use the name of the chain
// config to determine which hard fork to use so ClassicVM's gas table
// would not be used.
func ApplyMultiVmTransaction(config *params.ChainConfig, bc *BlockChain, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, totalUsedGas *uint64) (*types.Receipt, []*types.Log, uint64, error) {
	tx.SetSigner(config.GetSigner(header.Number))

	from, err := tx.From()
	if err != nil {
		return nil, nil, 0, err
	}
	vmtx := sputnikvm.Transaction{
		Caller:   from,
		GasPrice: tx.GasPrice(),
		GasLimit: tx.Gas(),
		Address:  tx.To(),
		Value:    tx.Value(),
		Input:    tx.Data(),
		Nonce:    new(big.Int).SetUint64(tx.Nonce()),
	}
	vmheader := sputnikvm.HeaderParams{
		Beneficiary: header.Coinbase,
		Timestamp:   header.Time.Uint64(),
		Number:      header.Number,
		Difficulty:  header.Difficulty,
		GasLimit:    new(big.Int).SetUint64(header.GasLimit),
	}

	currentNumber := header.Number
	homesteadFork := config.ForkByName("Homestead")
	eip150Fork := config.ForkByName("GasReprice")
	eip160Fork := config.ForkByName("Diehard")

	var vm *sputnikvm.VM
	if state.StartingNonce == 0 {
		if eip160Fork.Block != nil && currentNumber.Cmp(eip160Fork.Block) >= 0 {
			vm = sputnikvm.NewEIP160(&vmtx, &vmheader)
		} else if eip150Fork.Block != nil && currentNumber.Cmp(eip150Fork.Block) >= 0 {
			vm = sputnikvm.NewEIP150(&vmtx, &vmheader)
		} else if homesteadFork.Block != nil && currentNumber.Cmp(homesteadFork.Block) >= 0 {
			vm = sputnikvm.NewHomestead(&vmtx, &vmheader)
		} else {
			vm = sputnikvm.NewFrontier(&vmtx, &vmheader)
		}
	} else if state.StartingNonce == 1048576 {
		if eip160Fork.Block != nil && currentNumber.Cmp(eip160Fork.Block) >= 0 {
			vm = sputnikvm.NewMordenEIP160(&vmtx, &vmheader)
		} else if eip150Fork.Block != nil && currentNumber.Cmp(eip150Fork.Block) >= 0 {
			vm = sputnikvm.NewMordenEIP150(&vmtx, &vmheader)
		} else if homesteadFork.Block != nil && currentNumber.Cmp(homesteadFork.Block) >= 0 {
			vm = sputnikvm.NewMordenHomestead(&vmtx, &vmheader)
		} else {
			vm = sputnikvm.NewMordenFrontier(&vmtx, &vmheader)
		}
	} else {
		sputnikvm.SetCustomInitialNonce(big.NewInt(int64(state.StartingNonce)))
		if eip160Fork.Block != nil && currentNumber.Cmp(eip160Fork.Block) >= 0 {
			vm = sputnikvm.NewCustomEIP160(&vmtx, &vmheader)
		} else if eip150Fork.Block != nil && currentNumber.Cmp(eip150Fork.Block) >= 0 {
			vm = sputnikvm.NewCustomEIP150(&vmtx, &vmheader)
		} else if homesteadFork.Block != nil && currentNumber.Cmp(homesteadFork.Block) >= 0 {
			vm = sputnikvm.NewCustomHomestead(&vmtx, &vmheader)
		} else {
			vm = sputnikvm.NewCustomFrontier(&vmtx, &vmheader)
		}
	}

Loop:
	for {
		ret := vm.Fire()
		switch ret.Typ() {
		case sputnikvm.RequireNone:
			break Loop
		case sputnikvm.RequireAccount:
			address := ret.Address()
			if statedb.Exist(address) {
				vm.CommitAccount(address, new(big.Int).SetUint64(statedb.GetNonce(address)),
					statedb.GetBalance(address), statedb.GetCode(address))
				break
			}
			vm.CommitNonexist(address)
		case sputnikvm.RequireAccountCode:
			address := ret.Address()
			if statedb.Exist(address) {
				vm.CommitAccountCode(address, statedb.GetCode(address))
				break
			}
			vm.CommitNonexist(address)
		case sputnikvm.RequireAccountStorage:
			address := ret.Address()
			key := common.BigToHash(ret.StorageKey())
			if statedb.Exist(address) {
				value := statedb.GetState(address, key).Big()
				sKey := ret.StorageKey()
				vm.CommitAccountStorage(address, sKey, value)
				break
			}
			vm.CommitNonexist(address)
		case sputnikvm.RequireBlockhash:
			number := ret.BlockNumber()
			hash := common.Hash{}
			if block := bc.GetBlockByNumber(number.Uint64()); block != nil && block.Number().Cmp(number) == 0 {
				hash = block.Hash()
			}
			vm.CommitBlockhash(number, hash)
		}
	}

	// VM execution is finished at this point. We apply changes to the statedb.

	for _, account := range vm.AccountChanges() {
		switch account.Typ() {
		case sputnikvm.AccountChangeIncreaseBalance:
			address := account.Address()
			amount := account.ChangedAmount()
			statedb.AddBalance(address, amount)
		case sputnikvm.AccountChangeDecreaseBalance:
			address := account.Address()
			amount := account.ChangedAmount()
			balance := new(big.Int).Sub(statedb.GetBalance(address), amount)
			statedb.SetBalance(address, balance)
		case sputnikvm.AccountChangeRemoved:
			address := account.Address()
			statedb.Suicide(address)
		case sputnikvm.AccountChangeFull:
			address := account.Address()
			code := account.Code()
			nonce := account.Nonce()
			balance := account.Balance()
			statedb.SetBalance(address, balance)
			statedb.SetNonce(address, nonce.Uint64())
			statedb.SetCode(address, code)
			for _, item := range account.ChangedStorage() {
				statedb.SetState(address, common.BigToHash(item.Key), common.BigToHash(item.Value))
			}
		case sputnikvm.AccountChangeCreate:
			address := account.Address()
			code := account.Code()
			nonce := account.Nonce()
			balance := account.Balance()
			statedb.SetBalance(address, balance)
			statedb.SetNonce(address, nonce.Uint64())
			statedb.SetCode(address, code)
			for _, item := range account.Storage() {
				statedb.SetState(address, common.BigToHash(item.Key), common.BigToHash(item.Value))
			}
		default:
			panic("unreachable")
		}
	}
	for _, log := range vm.Logs() {
		statelog := &types.Log{
			Address:     log.Address,
			Topics:      log.Topics,
			Data:        log.Data,
			BlockNumber: header.Number.Uint64(),
			TxHash:      tx.Hash(),
		}
		// (, log.Topics, log.Data, header.Number.Uint64())
		statedb.AddLog(statelog)
	}
	usedGas := vm.UsedGas()
	*totalUsedGas += usedGas.Uint64()

	receipt := types.NewReceipt(statedb.IntermediateRoot(false).Bytes(), false, *totalUsedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = usedGas.Uint64()
	if tx.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(from, tx.Nonce())
	}

	logs := statedb.GetLogs(tx.Hash())
	receipt.Logs = logs
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	glog.V(logger.Debug).Infoln(receipt)

	vm.Free()
	return receipt, logs, usedGas.Uint64(), nil
}
