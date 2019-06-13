// +build sputnikvm

package core

import (
	"math/big"

	"github.com/ETCDEVTeam/sputnikvm-ffi/go/sputnikvm"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	evm "github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

const SputnikVMExists = true

// UseSputnikVM determines whether the VM will be Sputnik or Geth's native one.
// Awkward though it is to use a string variable, go's -ldflags relies on it being a constant string in order to be settable via -X from the command line,
// eg. -ldflags "-X core.UseSputnikVM=true".
var UseSputnikVM string = "false"

// Apply a transaction using the SputnikVM processor with the given
// chain config and state. Note that we use the name of the chain
// config to determine which hard fork to use so ClassicVM's gas table
// would not be used.
func ApplyMultiVmTransaction(config *ChainConfig, bc *BlockChain, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, totalUsedGas *big.Int) (*types.Receipt, evm.Logs, *big.Int, error) {
	tx.SetSigner(config.GetSigner(header.Number))

	from, err := tx.From()
	if err != nil {
		return nil, nil, nil, err
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
		GasLimit:    header.GasLimit,
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
		statelog := evm.NewLog(log.Address, log.Topics, log.Data, header.Number.Uint64())
		statedb.AddLog(*statelog)
	}
	usedGas := vm.UsedGas()
	totalUsedGas.Add(totalUsedGas, usedGas)

	receipt := types.NewReceipt(statedb.IntermediateRoot(false).Bytes(), totalUsedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = new(big.Int).Set(totalUsedGas)
	if vm.Failed() {
		receipt.Status = types.TxFailure
	} else {
		receipt.Status = types.TxSuccess
	}
	if MessageCreatesContract(tx) {
		receipt.ContractAddress = crypto.CreateAddress(from, tx.Nonce())
	}

	logs := statedb.GetLogs(tx.Hash())
	receipt.Logs = logs
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	glog.V(logger.Debug).Infoln(receipt)

	vm.Free()
	return receipt, logs, totalUsedGas, nil
}
