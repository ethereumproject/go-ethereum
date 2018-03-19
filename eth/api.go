// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ethereumproject/ethash"
	"github.com/ethereumproject/go-ethereum/accounts"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/common/compiler"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/event"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	ethMetrics "github.com/ethereumproject/go-ethereum/metrics"
	"github.com/ethereumproject/go-ethereum/miner"
	"github.com/ethereumproject/go-ethereum/p2p"
	"github.com/ethereumproject/go-ethereum/rlp"
	"github.com/ethereumproject/go-ethereum/rpc"
)

const defaultGas = uint64(90000)

// blockByNumber is a commonly used helper function which retrieves and returns
// the block for the given block number, capable of handling two special blocks:
// rpc.LatestBlockNumber and rpc.PendingBlockNumber. It returns nil when no block
// could be found.
func blockByNumber(m *miner.Miner, bc *core.BlockChain, blockNr rpc.BlockNumber) *types.Block {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, _ := m.Pending()
		return block
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return bc.CurrentBlock()
	}
	return bc.GetBlockByNumber(uint64(blockNr))
}

// stateAndBlockByNumber is a commonly used helper function which retrieves and
// returns the state and containing block for the given block number, capable of
// handling two special states: rpc.LatestBlockNumber and rpc.PendingBlockNumber.
// It returns nil when no block or state could be found.
func stateAndBlockByNumber(m *miner.Miner, bc *core.BlockChain, blockNr rpc.BlockNumber, chainDb ethdb.Database) (*state.StateDB, *types.Block, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := m.Pending()
		return state, block, nil
	}
	// Otherwise resolve the block number and return its state
	block := blockByNumber(m, bc, blockNr)
	if block == nil {
		return nil, nil, nil
	}
	stateDb, err := state.New(block.Root(), chainDb)
	return stateDb, block, err
}

// PublicEthereumAPI provides an API to access Ethereum related information.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicEthereumAPI struct {
	e   *Ethereum
	gpo *GasPriceOracle
}

// NewPublicEthereumAPI creates a new Ethereum protocol API.
func NewPublicEthereumAPI(e *Ethereum) *PublicEthereumAPI {
	return &PublicEthereumAPI{
		e:   e,
		gpo: e.gpo,
	}
}

// GasPrice returns a suggestion for a gas price.
func (s *PublicEthereumAPI) GasPrice() *big.Int {
	return s.gpo.SuggestPrice()
}

// GetCompilers returns the collection of available smart contract compilers
func (s *PublicEthereumAPI) GetCompilers() ([]string, error) {
	solc, err := s.e.Solc()
	if err == nil && solc != nil {
		return []string{"Solidity"}, nil
	}

	return []string{}, nil
}

// CompileSolidity compiles the given solidity source
func (s *PublicEthereumAPI) CompileSolidity(source string) (map[string]*compiler.Contract, error) {
	solc, err := s.e.Solc()
	if err != nil {
		return nil, err
	}

	if solc == nil {
		return nil, errors.New("solc (solidity compiler) not found")
	}

	return solc.Compile(source)
}

// Etherbase is the address that mining rewards will be send to
func (s *PublicEthereumAPI) Etherbase() (common.Address, error) {
	return s.e.Etherbase()
}

// Coinbase is the address that mining rewards will be send to (alias for Etherbase)
func (s *PublicEthereumAPI) Coinbase() (common.Address, error) {
	return s.Etherbase()
}

// ProtocolVersion returns the current Ethereum protocol version this node supports
func (s *PublicEthereumAPI) ProtocolVersion() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.EthVersion())
}

// Hashrate returns the POW hashrate
func (s *PublicEthereumAPI) Hashrate() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.Miner().HashRate())
}

// Syncing returns false in case the node is currently not syncing with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing:
// - startingBlock: block number this node started to synchronise from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (s *PublicEthereumAPI) Syncing() (interface{}, error) {
	origin, current, height, pulled, known := s.e.Downloader().Progress()

	// Return not syncing if the synchronisation already completed
	if current >= height {
		return false, nil
	}
	// Otherwise gather the block sync stats
	return map[string]interface{}{
		"startingBlock": rpc.NewHexNumber(origin),
		"currentBlock":  rpc.NewHexNumber(current),
		"highestBlock":  rpc.NewHexNumber(height),
		"pulledStates":  rpc.NewHexNumber(pulled),
		"knownStates":   rpc.NewHexNumber(known),
	}, nil
}

// ChainId returns the chain-configured value for EIP-155 chain id, used in signing protected txs.
// If EIP-155 is not configured it will return 0.
// Number will be returned as a string in hexadecimal format.
// 61 - Mainnet $((0x3d))
// 62 - Morden $((0x3e))
func (s *PublicEthereumAPI) ChainId() *big.Int {
	return s.e.chainConfig.GetChainID()
}

// PublicMinerAPI provides an API to control the miner.
// It offers only methods that operate on data that pose no security risk when it is publicly accessible.
type PublicMinerAPI struct {
	e     *Ethereum
	agent *miner.RemoteAgent
}

// NewPublicMinerAPI create a new PublicMinerAPI instance.
func NewPublicMinerAPI(e *Ethereum) *PublicMinerAPI {
	agent := miner.NewRemoteAgent()
	e.Miner().Register(agent)

	return &PublicMinerAPI{e, agent}
}

// Mining returns an indication if this node is currently mining.
func (s *PublicMinerAPI) Mining() bool {
	return s.e.IsMining()
}

// SubmitWork can be used by external miner to submit their POW solution. It returns an indication if the work was
// accepted. Note, this is not an indication if the provided work was valid!
func (s *PublicMinerAPI) SubmitWork(nonce rpc.HexNumber, solution, digest common.Hash) bool {
	return s.agent.SubmitWork(nonce.Uint64(), digest, solution)
}

// GetWork returns a work package for external miner. The work package consists of 3 strings
// result[0], 32 bytes hex encoded current block header pow-hash
// result[1], 32 bytes hex encoded seed hash used for DAG
// result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
func (s *PublicMinerAPI) GetWork() (work [3]string, err error) {
	if !s.e.IsMining() {
		if err := s.e.StartMining(0, ""); err != nil {
			return work, err
		}
	}
	if work, err = s.agent.GetWork(); err == nil {
		return
	}
	glog.V(logger.Debug).Infof("%v", err)
	return work, fmt.Errorf("mining not ready")
}

// SubmitHashrate can be used for remote miners to submit their hash rate. This enables the node to report the combined
// hash rate of all miners which submit work through this node. It accepts the miner hash rate and an identifier which
// must be unique between nodes.
func (s *PublicMinerAPI) SubmitHashrate(hashrate rpc.HexNumber, id common.Hash) bool {
	s.agent.SubmitHashrate(id, hashrate.Uint64())
	return true
}

// PrivateMinerAPI provides private RPC methods to control the miner.
// These methods can be abused by external users and must be considered insecure for use by untrusted users.
type PrivateMinerAPI struct {
	e *Ethereum
}

// NewPrivateMinerAPI create a new RPC service which controls the miner of this node.
func NewPrivateMinerAPI(e *Ethereum) *PrivateMinerAPI {
	return &PrivateMinerAPI{e: e}
}

// Start the miner with the given number of threads. If threads is nil the number of
// workers started is equal to the number of logical CPU's that are usable by this process.
func (s *PrivateMinerAPI) Start(threads *rpc.HexNumber) (bool, error) {
	s.e.StartAutoDAG()

	if threads == nil {
		threads = rpc.NewHexNumber(runtime.NumCPU())
	}

	err := s.e.StartMining(threads.Int(), "")
	if err == nil {
		return true, nil
	}
	return false, err
}

// Stop the miner
func (s *PrivateMinerAPI) Stop() bool {
	s.e.StopMining()
	return true
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (s *PrivateMinerAPI) SetGasPrice(gasPrice rpc.HexNumber) bool {
	s.e.Miner().SetGasPrice(gasPrice.BigInt())
	return true
}

// SetEtherbase sets the etherbase of the miner
func (s *PrivateMinerAPI) SetEtherbase(etherbase common.Address) bool {
	s.e.SetEtherbase(etherbase)
	return true
}

// StartAutoDAG starts auto DAG generation. This will prevent the DAG generating on epoch change
// which will cause the node to stop mining during the generation process.
func (s *PrivateMinerAPI) StartAutoDAG() bool {
	s.e.StartAutoDAG()
	return true
}

// StopAutoDAG stops auto DAG generation
func (s *PrivateMinerAPI) StopAutoDAG() bool {
	s.e.StopAutoDAG()
	return true
}

// MakeDAG creates the new DAG for the given block number
func (s *PrivateMinerAPI) MakeDAG(blockNr rpc.BlockNumber) (bool, error) {
	if err := ethash.MakeDAG(uint64(blockNr.Int64()), ""); err != nil {
		return false, err
	}
	return true, nil
}

// PublicTxPoolAPI offers and API for the transaction pool. It only operates on data that is non confidential.
type PublicTxPoolAPI struct {
	e *Ethereum
}

// NewPublicTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewPublicTxPoolAPI(e *Ethereum) *PublicTxPoolAPI {
	return &PublicTxPoolAPI{e}
}

// Content returns the transactions contained within the transaction pool.
func (s *PublicTxPoolAPI) Content() map[string]map[string]map[string][]*RPCTransaction {
	content := map[string]map[string]map[string][]*RPCTransaction{
		"pending": make(map[string]map[string][]*RPCTransaction),
		"queued":  make(map[string]map[string][]*RPCTransaction),
	}
	pending, queue := s.e.TxPool().Content()

	// Flatten the pending transactions
	for account, batches := range pending {
		dump := make(map[string][]*RPCTransaction)
		for nonce, txs := range batches {
			nonce := fmt.Sprintf("%d", nonce)
			for _, tx := range txs {
				dump[nonce] = append(dump[nonce], newRPCPendingTransaction(tx))
			}
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, batches := range queue {
		dump := make(map[string][]*RPCTransaction)
		for nonce, txs := range batches {
			nonce := fmt.Sprintf("%d", nonce)
			for _, tx := range txs {
				dump[nonce] = append(dump[nonce], newRPCPendingTransaction(tx))
			}
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (s *PublicTxPoolAPI) Status() map[string]*rpc.HexNumber {
	pending, queue := s.e.TxPool().Stats()
	return map[string]*rpc.HexNumber{
		"pending": rpc.NewHexNumber(pending),
		"queued":  rpc.NewHexNumber(queue),
	}
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (s *PublicTxPoolAPI) Inspect() map[string]map[string]map[string][]string {
	content := map[string]map[string]map[string][]string{
		"pending": make(map[string]map[string][]string),
		"queued":  make(map[string]map[string][]string),
	}
	pending, queue := s.e.TxPool().Content()

	// Define a formatter to flatten a transaction into a string
	var format = func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%s: %v wei + %v × %v gas", tx.To().Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v wei + %v × %v gas", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	// Flatten the pending transactions
	for account, batches := range pending {
		dump := make(map[string][]string)
		for nonce, txs := range batches {
			nonce := fmt.Sprintf("%d", nonce)
			for _, tx := range txs {
				dump[nonce] = append(dump[nonce], format(tx))
			}
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, batches := range queue {
		dump := make(map[string][]string)
		for nonce, txs := range batches {
			nonce := fmt.Sprintf("%d", nonce)
			for _, tx := range txs {
				dump[nonce] = append(dump[nonce], format(tx))
			}
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}

// PublicAccountAPI provides an API to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type PublicAccountAPI struct {
	am *accounts.Manager
}

// NewPublicAccountAPI creates a new PublicAccountAPI.
func NewPublicAccountAPI(am *accounts.Manager) *PublicAccountAPI {
	return &PublicAccountAPI{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *PublicAccountAPI) Accounts() []accounts.Account {
	return s.am.Accounts()
}

// PrivateAccountAPI provides an API to access accounts managed by this node.
// It offers methods to create, (un)lock en list accounts. Some methods accept
// passwords and are therefore considered private by default.
type PrivateAccountAPI struct {
	bc     *core.BlockChain
	am     *accounts.Manager
	txPool *core.TxPool
	txMu   *sync.Mutex
	gpo    *GasPriceOracle
}

// NewPrivateAccountAPI create a new PrivateAccountAPI.
func NewPrivateAccountAPI(e *Ethereum) *PrivateAccountAPI {
	return &PrivateAccountAPI{
		bc:     e.blockchain,
		am:     e.accountManager,
		txPool: e.txPool,
		txMu:   &e.txMu,
		gpo:    e.gpo,
	}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAccountAPI) ListAccounts() []common.Address {
	accounts := s.am.Accounts()
	addresses := make([]common.Address, len(accounts))
	for i, acc := range accounts {
		addresses[i] = acc.Address
	}
	return addresses
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PrivateAccountAPI) NewAccount(password string) (common.Address, error) {
	acc, err := s.am.NewAccount(password)
	if err == nil {
		return acc.Address, nil
	}
	return common.Address{}, err
}

// ImportRawKey stores the given hex encoded ECDSA key into the key directory,
// encrypting it with the passphrase.
func (s *PrivateAccountAPI) ImportRawKey(privkey string, password string) (common.Address, error) {
	hexkey, err := hex.DecodeString(privkey)
	if err != nil {
		return common.Address{}, err
	}

	acc, err := s.am.ImportECDSA(crypto.ToECDSA(hexkey), password)
	return acc.Address, err
}

// UnlockAccount will unlock the account associated with the given address with
// the given password for duration seconds. If duration is nil it will use a
// default of 300 seconds. It returns an indication if the account was unlocked.
func (s *PrivateAccountAPI) UnlockAccount(addr common.Address, password string, duration *rpc.HexNumber) (bool, error) {
	if duration == nil {
		duration = rpc.NewHexNumber(300)
	}
	a := accounts.Account{Address: addr}
	d := time.Duration(duration.Int64()) * time.Second
	if err := s.am.TimedUnlock(a, password, d); err != nil {
		return false, err
	}
	return true, nil
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PrivateAccountAPI) LockAccount(addr common.Address) bool {
	return s.am.Lock(addr) == nil
}

// Sign calculates an Ethereum ECDSA signature for:
// keccack256("\x19Ethereum Signed Message:\n" + len(message) + message))
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The key used to calculate the signature is decrypted with the given password.
//
// https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_sign
func (s *PrivateAccountAPI) Sign(data []byte, addr common.Address, passwd string) (string, error) {
	signature, err := s.am.SignWithPassphrase(addr, passwd, signHash(data))
	if err == nil {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return common.ToHex(signature), nil
}

// SendTransaction will create a transaction from the given arguments and
// tries to sign it with the key associated with args.To. If the given passwd isn't
// able to decrypt the key it fails.
func (s *PrivateAccountAPI) SendTransaction(args SendTxArgs, passwd string) (common.Hash, error) {
	args = prepareSendTxArgs(args, s.gpo)

	s.txMu.Lock()
	defer s.txMu.Unlock()

	if args.Nonce == nil {
		args.Nonce = rpc.NewHexNumber(s.txPool.State().GetNonce(args.From))
	}

	var tx *types.Transaction
	if args.To == nil {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), *args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	tx.SetSigner(s.bc.Config().GetSigner(s.bc.CurrentBlock().Number()))

	signature, err := s.am.SignWithPassphrase(args.From, passwd, tx.SigHash().Bytes())
	if err != nil {
		return common.Hash{}, err
	}

	return submitTransaction(s.bc, s.txPool, tx, signature)
}

// SignAndSendTransaction was renamed to SendTransaction. This method is deprecated
// and will be removed in the future. It primary goal is to give clients time to update.
func (s *PrivateAccountAPI) SignAndSendTransaction(args SendTxArgs, passwd string) (common.Hash, error) {
	return s.SignAndSendTransaction(args, passwd)
}

// PublicBlockChainAPI provides an API to access the Ethereum blockchain.
// It offers only methods that operate on public data that is freely available to anyone.
type PublicBlockChainAPI struct {
	config                  *core.ChainConfig
	bc                      *core.BlockChain
	chainDb                 ethdb.Database
	eventMux                *event.TypeMux
	muNewBlockSubscriptions sync.Mutex                             // protects newBlocksSubscriptions
	newBlockSubscriptions   map[string]func(core.ChainEvent) error // callbacks for new block subscriptions
	am                      *accounts.Manager
	miner                   *miner.Miner
	gpo                     *GasPriceOracle
}

// NewPublicBlockChainAPI creates a new Etheruem blockchain API.
func NewPublicBlockChainAPI(config *core.ChainConfig, bc *core.BlockChain, m *miner.Miner, chainDb ethdb.Database, gpo *GasPriceOracle, eventMux *event.TypeMux, am *accounts.Manager) *PublicBlockChainAPI {
	api := &PublicBlockChainAPI{
		config:   config,
		bc:       bc,
		miner:    m,
		chainDb:  chainDb,
		eventMux: eventMux,
		am:       am,
		newBlockSubscriptions: make(map[string]func(core.ChainEvent) error),
		gpo: gpo,
	}

	go api.subscriptionLoop()

	return api
}

// subscriptionLoop reads events from the global event mux and creates notifications for the matched subscriptions.
func (s *PublicBlockChainAPI) subscriptionLoop() {
	sub := s.eventMux.Subscribe(core.ChainEvent{})
	for event := range sub.Chan() {
		if chainEvent, ok := event.Data.(core.ChainEvent); ok {
			s.muNewBlockSubscriptions.Lock()
			for id, notifyOf := range s.newBlockSubscriptions {
				if notifyOf(chainEvent) == rpc.ErrNotificationNotFound {
					delete(s.newBlockSubscriptions, id)
				}
			}
			s.muNewBlockSubscriptions.Unlock()
		}
	}
}

// BlockNumber returns the block number of the chain head.
func (s *PublicBlockChainAPI) BlockNumber() *big.Int {
	return s.bc.CurrentHeader().Number
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (s *PublicBlockChainAPI) GetBalance(address common.Address, blockNr rpc.BlockNumber) (*big.Int, error) {
	state, _, err := stateAndBlockByNumber(s.miner, s.bc, blockNr, s.chainDb)
	if state == nil || err != nil {
		return nil, err
	}
	return state.GetBalance(address), nil
}

// GetBlockByNumber returns the requested block. When blockNr is -1 the chain head is returned. When fullTx is true all
// transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByNumber(blockNr rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		response, err := s.rpcOutputBlock(block, true, fullTx)
		if err == nil && blockNr == rpc.PendingBlockNumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, nil
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetBlockByHash(blockHash common.Hash, fullTx bool) (map[string]interface{}, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return s.rpcOutputBlock(block, true, fullTx)
	}
	return nil, nil
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockNumberAndIndex(blockNr rpc.BlockNumber, index rpc.HexNumber) (map[string]interface{}, error) {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		uncles := block.Uncles()
		if index.Int() < 0 || index.Int() >= len(uncles) {
			glog.V(logger.Debug).Infof("uncle block on index %d not found for block #%d", index.Int(), blockNr)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index.Int()])
		return s.rpcOutputBlock(block, false, false)
	}
	return nil, nil
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index. When fullTx is true
// all transactions in the block are returned in full detail, otherwise only the transaction hash is returned.
func (s *PublicBlockChainAPI) GetUncleByBlockHashAndIndex(blockHash common.Hash, index rpc.HexNumber) (map[string]interface{}, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		uncles := block.Uncles()
		if index.Int() < 0 || index.Int() >= len(uncles) {
			glog.V(logger.Debug).Infof("uncle block on index %d not found for block %s", index.Int(), blockHash.Hex())
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index.Int()])
		return s.rpcOutputBlock(block, false, false)
	}
	return nil, nil
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
func (s *PublicBlockChainAPI) GetUncleCountByBlockNumber(blockNr rpc.BlockNumber) *rpc.HexNumber {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Uncles()))
	}
	return nil
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (s *PublicBlockChainAPI) GetUncleCountByBlockHash(blockHash common.Hash) *rpc.HexNumber {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return rpc.NewHexNumber(len(block.Uncles()))
	}
	return nil
}

// NewBlocksArgs allows the user to specify if the returned block should include transactions and in which format.
type NewBlocksArgs struct {
	IncludeTransactions bool `json:"includeTransactions"`
	TransactionDetails  bool `json:"transactionDetails"`
}

// NewBlocks triggers a new block event each time a block is appended to the chain. It accepts an argument which allows
// the caller to specify whether the output should contain transactions and in what format.
func (s *PublicBlockChainAPI) NewBlocks(ctx context.Context, args NewBlocksArgs) (rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	// create a subscription that will remove itself when unsubscribed/cancelled
	subscription, err := notifier.NewSubscription(func(subId string) {
		s.muNewBlockSubscriptions.Lock()
		delete(s.newBlockSubscriptions, subId)
		s.muNewBlockSubscriptions.Unlock()
	})

	if err != nil {
		return nil, err
	}

	// add a callback that is called on chain events which will format the block and notify the client
	s.muNewBlockSubscriptions.Lock()
	s.newBlockSubscriptions[subscription.ID()] = func(e core.ChainEvent) error {
		notification, err := s.rpcOutputBlock(e.Block, args.IncludeTransactions, args.TransactionDetails)
		if err == nil {
			return subscription.Notify(notification)
		}
		glog.V(logger.Warn).Info("unable to format block %v\n", err)
		return nil
	}
	s.muNewBlockSubscriptions.Unlock()
	return subscription, nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *PublicBlockChainAPI) GetCode(address common.Address, blockNr rpc.BlockNumber) (string, error) {
	state, _, err := stateAndBlockByNumber(s.miner, s.bc, blockNr, s.chainDb)
	if state == nil || err != nil {
		return "", err
	}
	res := state.GetCode(address)
	if len(res) == 0 { // backwards compatibility
		return "0x", nil
	}
	return common.ToHex(res), nil
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (s *PublicBlockChainAPI) GetStorageAt(address common.Address, key string, blockNr rpc.BlockNumber) (string, error) {
	state, _, err := stateAndBlockByNumber(s.miner, s.bc, blockNr, s.chainDb)
	if state == nil || err != nil {
		return "0x", err
	}
	return state.GetState(address, common.HexToHash(key)).Hex(), nil
}

// callmsg is the message type used for call transactions.
type callmsg struct {
	from          *state.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error)         { return m.from.Address(), nil }
func (m callmsg) FromFrontier() (common.Address, error) { return m.from.Address(), nil }
func (m callmsg) Nonce() uint64                         { return m.from.Nonce() }
func (m callmsg) To() *common.Address                   { return m.to }
func (m callmsg) GasPrice() *big.Int                    { return m.gasPrice }
func (m callmsg) Gas() *big.Int                         { return m.gas }
func (m callmsg) Value() *big.Int                       { return m.value }
func (m callmsg) Data() []byte                          { return m.data }

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Value    rpc.HexNumber   `json:"value"`
	Data     string          `json:"data"`
}

func (s *PublicBlockChainAPI) doCall(args CallArgs, blockNr rpc.BlockNumber) (string, *big.Int, error) {
	// Fetch the state associated with the block number
	stateDb, block, err := stateAndBlockByNumber(s.miner, s.bc, blockNr, s.chainDb)
	if stateDb == nil || err != nil {
		return "0x", nil, err
	}
	stateDb = stateDb.Copy()

	// Retrieve the account state object to interact with
	var from *state.StateObject
	if args.From == (common.Address{}) {
		accounts := s.am.Accounts()
		if len(accounts) == 0 {
			from = stateDb.GetOrNewStateObject(common.Address{})
		} else {
			from = stateDb.GetOrNewStateObject(accounts[0].Address)
		}
	} else {
		from = stateDb.GetOrNewStateObject(args.From)
	}
	from.SetBalance(common.MaxBig)

	// Assemble the CALL invocation
	msg := callmsg{
		from:     from,
		to:       args.To,
		gas:      args.Gas.BigInt(),
		gasPrice: args.GasPrice.BigInt(),
		value:    args.Value.BigInt(),
		data:     common.FromHex(args.Data),
	}
	if msg.gas == nil {
		msg.gas = big.NewInt(50000000)
	}
	if msg.gasPrice == nil {
		msg.gasPrice = s.gpo.SuggestPrice()
	}

	// Execute the call and return
	vmenv := core.NewEnv(stateDb, s.config, s.bc, msg, block.Header())
	gp := new(core.GasPool).AddGas(common.MaxBig)

	res, requiredGas, _, err := core.NewStateTransition(vmenv, msg, gp).TransitionDb()
	if len(res) == 0 { // backwards compatibility
		return "0x", requiredGas, err
	}
	return common.ToHex(res), requiredGas, err
}

// Call executes the given transaction on the state for the given block number.
// It doesn't make and changes in the state/blockchain and is useful to execute and retrieve values.
func (s *PublicBlockChainAPI) Call(args CallArgs, blockNr rpc.BlockNumber) (string, error) {
	result, _, err := s.doCall(args, blockNr)
	return result, err
}

// EstimateGas returns an estimate of the amount of gas needed to execute the given transaction.
func (s *PublicBlockChainAPI) EstimateGas(args CallArgs) (*rpc.HexNumber, error) {
	_, gas, err := s.doCall(args, rpc.PendingBlockNumber)
	return rpc.NewHexNumber(gas), err
}

// rpcOutputBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func (s *PublicBlockChainAPI) rpcOutputBlock(b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	fields := map[string]interface{}{
		"number":           rpc.NewHexNumber(b.Number()),
		"hash":             b.Hash(),
		"parentHash":       b.ParentHash(),
		"nonce":            b.Header().Nonce,
		"sha3Uncles":       b.UncleHash(),
		"logsBloom":        b.Bloom(),
		"stateRoot":        b.Root(),
		"miner":            b.Coinbase(),
		"difficulty":       rpc.NewHexNumber(b.Difficulty()),
		"totalDifficulty":  rpc.NewHexNumber(s.bc.GetTd(b.Hash())),
		"extraData":        fmt.Sprintf("0x%x", b.Extra()),
		"size":             rpc.NewHexNumber(b.Size().Int64()),
		"gasLimit":         rpc.NewHexNumber(b.GasLimit()),
		"gasUsed":          rpc.NewHexNumber(b.GasUsed()),
		"timestamp":        rpc.NewHexNumber(b.Time()),
		"transactionsRoot": b.TxHash(),
		"receiptsRoot":     b.ReceiptHash(),
	}

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}

		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				if tx.Protected() {
					tx.SetSigner(types.NewChainIdSigner(s.bc.Config().GetChainID()))
				}
				return newRPCTransaction(b, tx.Hash())
			}
		}

		txs := b.Transactions()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range b.Transactions() {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		fields["transactions"] = transactions
	}

	uncles := b.Uncles()
	uncleHashes := make([]common.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields["uncles"] = uncleHashes

	return fields, nil
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        common.Hash     `json:"blockHash"`
	BlockNumber      *rpc.HexNumber  `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              *rpc.HexNumber  `json:"gas"`
	GasPrice         *rpc.HexNumber  `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            string          `json:"input"`
	Nonce            *rpc.HexNumber  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *rpc.HexNumber  `json:"transactionIndex"`
	Value            *rpc.HexNumber  `json:"value"`
	ReplayProtected  bool            `json:"replayProtected"`
	ChainId          *big.Int        `json:"chainId,omitempty"`
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction) *RPCTransaction {
	from, _ := tx.From()

	var protected bool
	var chainId *big.Int
	if tx.Protected() {
		protected = true
		chainId = tx.ChainId()
	}

	return &RPCTransaction{
		From:            from,
		Gas:             rpc.NewHexNumber(tx.Gas()),
		GasPrice:        rpc.NewHexNumber(tx.GasPrice()),
		Hash:            tx.Hash(),
		Input:           fmt.Sprintf("0x%x", tx.Data()),
		Nonce:           rpc.NewHexNumber(tx.Nonce()),
		To:              tx.To(),
		Value:           rpc.NewHexNumber(tx.Value()),
		ReplayProtected: protected,
		ChainId:         chainId,
	}
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *types.Block, txIndex int) (*RPCTransaction, error) {
	if txIndex >= 0 && txIndex < len(b.Transactions()) {
		tx := b.Transactions()[txIndex]
		var signer types.Signer = types.BasicSigner{}
		var protected bool
		var chainId *big.Int
		if tx.Protected() {
			signer = types.NewChainIdSigner(tx.ChainId())
			protected = true
			chainId = tx.ChainId()
		}
		from, _ := types.Sender(signer, tx)

		return &RPCTransaction{
			BlockHash:        b.Hash(),
			BlockNumber:      rpc.NewHexNumber(b.Number()),
			From:             from,
			Gas:              rpc.NewHexNumber(tx.Gas()),
			GasPrice:         rpc.NewHexNumber(tx.GasPrice()),
			Hash:             tx.Hash(),
			Input:            fmt.Sprintf("0x%x", tx.Data()),
			Nonce:            rpc.NewHexNumber(tx.Nonce()),
			To:               tx.To(),
			TransactionIndex: rpc.NewHexNumber(txIndex),
			Value:            rpc.NewHexNumber(tx.Value()),
			ReplayProtected:  protected,
			ChainId:          chainId,
		}, nil
	}

	return nil, nil
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation.
func newRPCTransaction(b *types.Block, txHash common.Hash) (*RPCTransaction, error) {
	for idx, tx := range b.Transactions() {
		if tx.Hash() == txHash {
			return newRPCTransactionFromBlockIndex(b, idx)
		}
	}

	return nil, nil
}

// PublicTransactionPoolAPI exposes methods for the RPC interface
type PublicTransactionPoolAPI struct {
	eventMux        *event.TypeMux
	chainDb         ethdb.Database
	gpo             *GasPriceOracle
	bc              *core.BlockChain
	miner           *miner.Miner
	am              *accounts.Manager
	txPool          *core.TxPool
	txMu            *sync.Mutex
	muPendingTxSubs sync.Mutex
	pendingTxSubs   map[string]rpc.Subscription
}

// NewPublicTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewPublicTransactionPoolAPI(e *Ethereum) *PublicTransactionPoolAPI {
	api := &PublicTransactionPoolAPI{
		eventMux:      e.eventMux,
		gpo:           e.gpo,
		chainDb:       e.chainDb,
		bc:            e.blockchain,
		am:            e.accountManager,
		txPool:        e.txPool,
		txMu:          &e.txMu,
		miner:         e.miner,
		pendingTxSubs: make(map[string]rpc.Subscription),
	}
	go api.subscriptionLoop()

	return api
}

// subscriptionLoop listens for events on the global event mux and creates notifications for subscriptions.
func (s *PublicTransactionPoolAPI) subscriptionLoop() {
	sub := s.eventMux.Subscribe(core.TxPreEvent{})
	for event := range sub.Chan() {
		tx := event.Data.(core.TxPreEvent)
		if from, err := tx.Tx.From(); err == nil {
			if s.am.HasAddress(from) {
				s.muPendingTxSubs.Lock()
				for id, sub := range s.pendingTxSubs {
					if sub.Notify(tx.Tx.Hash()) == rpc.ErrNotificationNotFound {
						delete(s.pendingTxSubs, id)
					}
				}
				s.muPendingTxSubs.Unlock()
			}
		}
	}
}

func getTransaction(chainDb ethdb.Database, txPool *core.TxPool, txHash common.Hash) (*types.Transaction, bool, error) {
	txData, err := chainDb.Get(txHash.Bytes())
	isPending := false
	tx := new(types.Transaction)

	if err == nil && len(txData) > 0 {
		if err := rlp.DecodeBytes(txData, tx); err != nil {
			return nil, isPending, err
		}
	} else {
		// pending transaction?
		tx = txPool.GetTransaction(txHash)
		isPending = true
	}

	return tx, isPending, nil
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByNumber(blockNr rpc.BlockNumber) *rpc.HexNumber {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}
	return nil
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (s *PublicTransactionPoolAPI) GetBlockTransactionCountByHash(blockHash common.Hash) *rpc.HexNumber {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return rpc.NewHexNumber(len(block.Transactions()))
	}
	return nil
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockNumberAndIndex(blockNr rpc.BlockNumber, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := blockByNumber(s.miner, s.bc, blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (s *PublicTransactionPoolAPI) GetTransactionByBlockHashAndIndex(blockHash common.Hash, index rpc.HexNumber) (*RPCTransaction, error) {
	if block := s.bc.GetBlock(blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, index.Int())
	}
	return nil, nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (s *PublicTransactionPoolAPI) GetTransactionCount(address common.Address, blockNr rpc.BlockNumber) (*rpc.HexNumber, error) {
	state, _, err := stateAndBlockByNumber(s.miner, s.bc, blockNr, s.chainDb)
	if state == nil || err != nil {
		return nil, err
	}
	return rpc.NewHexNumber(state.GetNonce(address)), nil
}

// getTransactionBlockData fetches the meta data for the given transaction from the chain database. This is useful to
// retrieve block information for a hash. It returns the block hash, block index and transaction index.
func getTransactionBlockData(chainDb ethdb.Database, txHash common.Hash) (common.Hash, uint64, uint64, error) {
	var txBlock struct {
		BlockHash  common.Hash
		BlockIndex uint64
		Index      uint64
	}

	blockData, err := chainDb.Get(append(txHash.Bytes(), 0x0001))
	if err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	reader := bytes.NewReader(blockData)
	if err = rlp.Decode(reader, &txBlock); err != nil {
		return common.Hash{}, uint64(0), uint64(0), err
	}

	return txBlock.BlockHash, txBlock.BlockIndex, txBlock.Index, nil
}

// GetTransactionByHash returns the transaction for the given hash
func (s *PublicTransactionPoolAPI) GetTransactionByHash(txHash common.Hash) (*RPCTransaction, error) {
	var tx *types.Transaction
	var isPending bool
	var err error

	if tx, isPending, err = getTransaction(s.chainDb, s.txPool, txHash); err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	} else if tx == nil {
		return nil, nil
	}

	if isPending {
		return newRPCPendingTransaction(tx), nil
	}

	blockHash, _, _, err := getTransactionBlockData(s.chainDb, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	if block := s.bc.GetBlock(blockHash); block != nil {
		return newRPCTransaction(block, txHash)
	}

	return nil, nil
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *PublicTransactionPoolAPI) GetTransactionReceipt(txHash common.Hash) (map[string]interface{}, error) {
	receipt := core.GetReceipt(s.chainDb, txHash)
	if receipt == nil {
		glog.V(logger.Debug).Infof("receipt not found for transaction %s", txHash.Hex())
		return nil, nil
	}

	tx, _, err := getTransaction(s.chainDb, s.txPool, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	txBlock, blockIndex, index, err := getTransactionBlockData(s.chainDb, txHash)
	if err != nil {
		glog.V(logger.Debug).Infof("%v\n", err)
		return nil, nil
	}

	var signer types.Signer = types.BasicSigner{}
	if tx.Protected() {
		signer = types.NewChainIdSigner(tx.ChainId())
	}
	from, _ := types.Sender(signer, tx)

	fields := map[string]interface{}{
		"root":              common.Bytes2Hex(receipt.PostState),
		"blockHash":         txBlock,
		"blockNumber":       rpc.NewHexNumber(blockIndex),
		"transactionHash":   txHash,
		"transactionIndex":  rpc.NewHexNumber(index),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           rpc.NewHexNumber(receipt.GasUsed),
		"cumulativeGasUsed": rpc.NewHexNumber(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
	}

	if receipt.Logs == nil {
		fields["logs"] = []vm.Logs{}
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if bytes.Compare(receipt.ContractAddress.Bytes(), bytes.Repeat([]byte{0}, 20)) != 0 {
		fields["contractAddress"] = receipt.ContractAddress
	}

	return fields, nil
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (s *PublicTransactionPoolAPI) sign(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	signer := s.bc.Config().GetSigner(s.bc.CurrentBlock().Number())

	signature, err := s.am.Sign(addr, signer.Hash(tx).Bytes())
	if err != nil {
		return nil, err
	}
	return tx.WithSigner(signer).WithSignature(signature)
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Value    *rpc.HexNumber  `json:"value"`
	Data     string          `json:"data"`
	Nonce    *rpc.HexNumber  `json:"nonce"`
}

// prepareSendTxArgs is a helper function that fills in default values for unspecified tx fields.
func prepareSendTxArgs(args SendTxArgs, gpo *GasPriceOracle) SendTxArgs {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		args.GasPrice = rpc.NewHexNumber(gpo.SuggestPrice())
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}
	return args
}

// submitTransaction is a helper function that submits tx to txPool and creates a log entry.
func submitTransaction(bc *core.BlockChain, txPool *core.TxPool, tx *types.Transaction, signature []byte) (common.Hash, error) {
	signer := bc.Config().GetSigner(bc.CurrentBlock().Number())

	signedTx, err := tx.WithSigner(signer).WithSignature(signature)
	if err != nil {
		return common.Hash{}, err
	}

	txPool.SetLocal(signedTx)
	if err := txPool.Add(signedTx); err != nil {
		return common.Hash{}, err
	}

	if signedTx.To() == nil {
		from, _ := signedTx.From()
		addr := crypto.CreateAddress(from, signedTx.Nonce())
		glog.V(logger.Info).Infof("Tx(%s) created: %s\n", signedTx.Hash().Hex(), addr.Hex())
	} else {
		glog.V(logger.Info).Infof("Tx(%s) to: %s\n", signedTx.Hash().Hex(), tx.To().Hex())
	}

	return signedTx.Hash(), nil
}

// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
func (s *PublicTransactionPoolAPI) SendTransaction(args SendTxArgs) (common.Hash, error) {
	args = prepareSendTxArgs(args, s.gpo)

	s.txMu.Lock()
	defer s.txMu.Unlock()

	if args.Nonce == nil {
		args.Nonce = rpc.NewHexNumber(s.txPool.State().GetNonce(args.From))
	}

	var tx *types.Transaction
	if args.To == nil {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), *args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	signer := s.bc.Config().GetSigner(s.bc.CurrentBlock().Number())
	tx.SetSigner(signer)

	signature, err := s.am.Sign(args.From, signer.Hash(tx).Bytes())
	if err != nil {
		return common.Hash{}, err
	}

	return submitTransaction(s.bc, s.txPool, tx, signature)
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (s *PublicTransactionPoolAPI) SendRawTransaction(encodedTx string) (string, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(common.FromHex(encodedTx), tx); err != nil {
		return "", err
	}

	s.txPool.SetLocal(tx)
	if err := s.txPool.Add(tx); err != nil {
		return "", err
	}

	if tx.To() == nil {
		from, err := tx.From()
		if err != nil {
			return "", err
		}
		addr := crypto.CreateAddress(from, tx.Nonce())
		glog.V(logger.Info).Infof("Tx(%x) created: %x\n", tx.Hash(), addr)
	} else {
		glog.V(logger.Info).Infof("Tx(%x) to: %x\n", tx.Hash(), tx.To())
	}

	return tx.Hash().Hex(), nil
}

// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calculated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

// Sign signs the given hash using the key that matches the address. The key must be
// unlocked in order to sign the hash.
func (s *PublicTransactionPoolAPI) Sign(addr common.Address, data []byte) (string, error) {
	signature, err := s.am.Sign(addr, signHash(data))
	if err == nil {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return common.ToHex(signature), err
}

// SignTransactionArgs represents the arguments to sign a transaction.
type SignTransactionArgs struct {
	From     common.Address
	To       *common.Address
	Nonce    *rpc.HexNumber
	Value    *rpc.HexNumber
	Gas      *rpc.HexNumber
	GasPrice *rpc.HexNumber
	Data     string

	BlockNumber int64
}

// Tx is a helper object for argument and return values
type Tx struct {
	tx *types.Transaction

	To       *common.Address `json:"to"`
	From     common.Address  `json:"from"`
	Nonce    *rpc.HexNumber  `json:"nonce"`
	Value    *rpc.HexNumber  `json:"value"`
	Data     string          `json:"data"`
	GasLimit *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Hash     common.Hash     `json:"hash"`
}

// UnmarshalJSON parses JSON data into tx.
func (tx *Tx) UnmarshalJSON(b []byte) (err error) {
	req := struct {
		To       *common.Address `json:"to"`
		From     common.Address  `json:"from"`
		Nonce    *rpc.HexNumber  `json:"nonce"`
		Value    *rpc.HexNumber  `json:"value"`
		Data     string          `json:"data"`
		GasLimit *rpc.HexNumber  `json:"gas"`
		GasPrice *rpc.HexNumber  `json:"gasPrice"`
		Hash     common.Hash     `json:"hash"`
	}{}

	if err := json.Unmarshal(b, &req); err != nil {
		return err
	}

	tx.To = req.To
	tx.From = req.From
	tx.Nonce = req.Nonce
	tx.Value = req.Value
	tx.Data = req.Data
	tx.GasLimit = req.GasLimit
	tx.GasPrice = req.GasPrice
	tx.Hash = req.Hash

	data := common.Hex2Bytes(tx.Data)

	if tx.Nonce == nil {
		return fmt.Errorf("need nonce")
	}
	if tx.Value == nil {
		tx.Value = rpc.NewHexNumber(0)
	}
	if tx.GasLimit == nil {
		tx.GasLimit = rpc.NewHexNumber(0)
	}
	if tx.GasPrice == nil {
		tx.GasPrice = rpc.NewHexNumber(int64(50000000000))
	}

	if req.To == nil {
		tx.tx = types.NewContractCreation(tx.Nonce.Uint64(), tx.Value.BigInt(), tx.GasLimit.BigInt(), tx.GasPrice.BigInt(), data)
	} else {
		tx.tx = types.NewTransaction(tx.Nonce.Uint64(), *tx.To, tx.Value.BigInt(), tx.GasLimit.BigInt(), tx.GasPrice.BigInt(), data)
	}

	return nil
}

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw string `json:"raw"`
	Tx  *Tx    `json:"tx"`
}

func newTx(t *types.Transaction) *Tx {
	var signer types.Signer = types.BasicSigner{}
	if t.Protected() {
		signer = types.NewChainIdSigner(t.ChainId())
	}
	from, _ := types.Sender(signer, t)
	return &Tx{
		tx:       t,
		To:       t.To(),
		From:     from,
		Value:    rpc.NewHexNumber(t.Value()),
		Nonce:    rpc.NewHexNumber(t.Nonce()),
		Data:     "0x" + common.Bytes2Hex(t.Data()),
		GasLimit: rpc.NewHexNumber(t.Gas()),
		GasPrice: rpc.NewHexNumber(t.GasPrice()),
		Hash:     t.Hash(),
	}
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (s *PublicTransactionPoolAPI) SignTransaction(args SignTransactionArgs) (*SignTransactionResult, error) {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		args.GasPrice = rpc.NewHexNumber(s.gpo.SuggestPrice())
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}

	s.txMu.Lock()
	defer s.txMu.Unlock()

	if args.Nonce == nil {
		args.Nonce = rpc.NewHexNumber(s.txPool.State().GetNonce(args.From))
	}

	var tx *types.Transaction
	if args.To == nil {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), *args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	signedTx, err := s.sign(args.From, tx)
	if err != nil {
		return nil, err
	}

	data, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, err
	}

	return &SignTransactionResult{"0x" + common.Bytes2Hex(data), newTx(signedTx)}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool and have a from address that is one of
// the accounts this node manages.
func (s *PublicTransactionPoolAPI) PendingTransactions() []*RPCTransaction {
	pending := s.txPool.GetTransactions()
	transactions := make([]*RPCTransaction, 0, len(pending))
	for _, tx := range pending {
		var signer types.Signer = types.BasicSigner{}
		if tx.Protected() {
			signer = types.NewChainIdSigner(tx.ChainId())
		}
		from, _ := types.Sender(signer, tx)
		if s.am.HasAddress(from) {
			transactions = append(transactions, newRPCPendingTransaction(tx))
		}
	}
	return transactions
}

// NewPendingTransactions creates a subscription that is triggered each time a transaction enters the transaction pool
// and is send from one of the transactions this nodes manages.
func (s *PublicTransactionPoolAPI) NewPendingTransactions(ctx context.Context) (rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	subscription, err := notifier.NewSubscription(func(id string) {
		s.muPendingTxSubs.Lock()
		delete(s.pendingTxSubs, id)
		s.muPendingTxSubs.Unlock()
	})

	if err != nil {
		return nil, err
	}

	s.muPendingTxSubs.Lock()
	s.pendingTxSubs[subscription.ID()] = subscription
	s.muPendingTxSubs.Unlock()

	return subscription, nil
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove the given transaction from the
// pool and reinsert it with the new gas price and limit.
func (s *PublicTransactionPoolAPI) Resend(tx Tx, gasPrice, gasLimit *rpc.HexNumber) (common.Hash, error) {

	pending := s.txPool.GetTransactions()
	for _, p := range pending {
		var signer types.Signer = types.BasicSigner{}
		if p.Protected() {
			signer = types.NewChainIdSigner(p.ChainId())
		}

		if pFrom, err := types.Sender(signer, p); err == nil && pFrom == tx.From && signer.Hash(p) == signer.Hash(tx.tx) {
			if gasPrice == nil {
				gasPrice = rpc.NewHexNumber(tx.tx.GasPrice())
			}
			if gasLimit == nil {
				gasLimit = rpc.NewHexNumber(tx.tx.Gas())
			}

			var newTx *types.Transaction
			if tx.tx.To() == nil {
				newTx = types.NewContractCreation(tx.tx.Nonce(), tx.tx.Value(), gasPrice.BigInt(), gasLimit.BigInt(), tx.tx.Data())
			} else {
				newTx = types.NewTransaction(tx.tx.Nonce(), *tx.tx.To(), tx.tx.Value(), gasPrice.BigInt(), gasLimit.BigInt(), tx.tx.Data())
			}

			signedTx, err := s.sign(tx.From, newTx)
			if err != nil {
				return common.Hash{}, err
			}

			s.txPool.RemoveTx(tx.Hash)
			if err = s.txPool.Add(signedTx); err != nil {
				return common.Hash{}, err
			}

			return signedTx.Hash(), nil
		}
	}

	return common.Hash{}, fmt.Errorf("Transaction %#x not found", tx.Hash)
}

// PrivateAdminAPI is the collection of Etheruem APIs exposed over the private
// admin endpoint.
type PrivateAdminAPI struct {
	eth *Ethereum
}

// NewPrivateAdminAPI creates a new API definition for the private admin methods
// of the Ethereum service.
func NewPrivateAdminAPI(eth *Ethereum) *PrivateAdminAPI {
	return &PrivateAdminAPI{eth: eth}
}

// SetSolc sets the Solidity compiler path to be used by the node.
func (api *PrivateAdminAPI) SetSolc(path string) (string, error) {
	solc, err := api.eth.SetSolc(path)
	if err != nil {
		return "", err
	}
	return solc.Info(), nil
}

// ExportChain exports the current blockchain into a local file.
func (api *PrivateAdminAPI) ExportChain(file string) (bool, error) {
	// Make sure we can create the file to export into
	out, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return false, err
	}
	defer out.Close()

	// Export the blockchain
	if err := api.eth.BlockChain().Export(out); err != nil {
		return false, err
	}
	return true, nil
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash()) {
			return false
		}
	}

	return true
}

// ImportChain imports a blockchain from a local file.
func (api *PrivateAdminAPI) ImportChain(file string) (bool, error) {
	// Make sure the can access the file to import
	in, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer in.Close()

	// Run actual the import in pre-configured batches
	stream := rlp.NewStream(in, 0)

	blocks, index := make([]*types.Block, 0, 2500), 0
	for batch := 0; ; batch++ {
		// Load a batch of blocks from the input file
		for len(blocks) < cap(blocks) {
			block := new(types.Block)
			if err := stream.Decode(block); err == io.EOF {
				break
			} else if err != nil {
				return false, fmt.Errorf("block %d: failed to parse: %v", index, err)
			}
			blocks = append(blocks, block)
			index++
		}
		if len(blocks) == 0 {
			break
		}

		if hasAllBlocks(api.eth.BlockChain(), blocks) {
			blocks = blocks[:0]
			continue
		}
		// Import the batch and reset the buffer
		if _, err := api.eth.BlockChain().InsertChain(blocks); err != nil {
			return false, fmt.Errorf("batch %d: failed to insert: %v", batch, err)
		}
		blocks = blocks[:0]
	}
	return true, nil
}

// PublicDebugAPI is the collection of Etheruem APIs exposed over the public
// debugging endpoint.
type PublicDebugAPI struct {
	eth *Ethereum
}

// NewPublicDebugAPI creates a new API definition for the public debug methods
// of the Ethereum service.
func NewPublicDebugAPI(eth *Ethereum) *PublicDebugAPI {
	return &PublicDebugAPI{eth: eth}
}

// DumpBlock retrieves the entire state of the database at a given block.
// TODO: update to be able to dump for specific addresses?
func (api *PublicDebugAPI) DumpBlock(number uint64) (state.Dump, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return state.Dump{}, fmt.Errorf("block #%d not found", number)
	}
	stateDb, err := api.eth.BlockChain().StateAt(block.Root())
	if err != nil {
		return state.Dump{}, err
	}
	return stateDb.RawDump([]common.Address{}), nil
}

// AccountExist checks whether an address is considered exists at a given block.
func (api *PublicDebugAPI) AccountExist(address common.Address, number uint64) (bool, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return false, fmt.Errorf("block #%d not found", number)
	}
	stateDb, err := api.eth.BlockChain().StateAt(block.Root())
	if err != nil {
		return false, err
	}
	return stateDb.Exist(address), nil
}

// GetBlockRlp retrieves the RLP encoded for of a single block.
func (api *PublicDebugAPI) GetBlockRlp(number uint64) (string, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", encoded), nil
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *PublicDebugAPI) PrintBlock(number uint64) (string, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return fmt.Sprintf("%s", block), nil
}

// SeedHash retrieves the seed hash of a block.
func (api *PublicDebugAPI) SeedHash(number uint64) (string, error) {
	block := api.eth.BlockChain().GetBlockByNumber(number)
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	hash, err := ethash.GetSeedHash(number)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%x", hash), nil
}

// Metrics return all available registered metrics for the client.
// See https://github.com/ethereumproject/go-ethereum/wiki/Metrics-and-Monitoring for prophetic documentation.
func (api *PublicDebugAPI) Metrics(raw bool) (map[string]interface{}, error) {

	// Create a rate formatter
	units := []string{"", "K", "M", "G", "T", "E", "P"}
	round := func(value float64, prec int) string {
		unit := 0
		for value >= 1000 {
			unit, value, prec = unit+1, value/1000, 2
		}
		return fmt.Sprintf(fmt.Sprintf("%%.%df%s", prec, units[unit]), value)
	}
	format := func(total float64, rate float64) string {
		return fmt.Sprintf("%s (%s/s)", round(total, 0), round(rate, 2))
	}

	var b []byte
	var err error

	b, err = ethMetrics.CollectToJSON()
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{})
	err = json.Unmarshal(b, &out)
	if err != nil {
		return nil, err
	}

	// human friendly if specified
	if !raw {
		rout := out
		for k, v := range out {
			if m, ok := v.(map[string]interface{}); ok {
				// This is one way of differentiating Meters, Timers, and Gauges.
				// whilei: (having some trouble with *metrics.StandardTimer(etc) type switches)
				// If new types metrics are introduced, they should have their relevant values added here.
				if _, ok := m["mean.rate"]; ok {
					h1m := format(m["1m.rate"].(float64)*60, m["1m.rate"].(float64))
					h5m := format(m["5m.rate"].(float64)*300, m["5m.rate"].(float64))
					h15m := format(m["15m.rate"].(float64)*900, m["15m.rate"].(float64))
					hmr := format(m["mean.rate"].(float64), m["mean.rate"].(float64))
					rout[k] = map[string]interface{}{
						"1m.rate":   h1m,
						"5m.rate":   h5m,
						"15m.rate":  h15m,
						"mean.rate": hmr,
						"count":     fmt.Sprintf("%v", m["count"]),
					}
				} else if _, ok := m["value"]; ok {
					rout[k] = map[string]interface{}{
						"value": fmt.Sprintf("%v", m["value"]),
					}
				}
			}
		}
		return rout, nil
	}

	return out, nil
}

// Verbosity implements api method debug_verbosity, enabling setting
// global logging verbosity on the fly.
// Note that it will NOT allow setting verbosity '0', which is effectively 'off'.
// In place of ability to receive 0 as a functional parameter, debug.verbosity() -> debug.verbosity(0) -> glog.GetVerbosity().
// creates a shorthand/convenience method as a "getter" function returning the current value
// of the verbosity setting.
func (api *PublicDebugAPI) Verbosity(n uint64) (int, error) {
	nint := int(n)
	if nint == 0 {
		return int(*glog.GetVerbosity()), nil
	}
	if nint <= logger.Detail || nint == logger.Ridiculousness {
		glog.SetV(nint)
		i := int(*glog.GetVerbosity())
		glog.V(logger.Warn).Infof("Set verbosity: %d", i)
		glog.D(logger.Warn).Warnf("Set verbosity: %d", i)
		return i, nil
	}
	e := errors.New("invalid logging level")
	glog.V(logger.Warn).Errorf("Set verbosity failed: %d (%v)", nint, e)
	glog.D(logger.Error).Errorf("Set verbosity failed: %d (%v)", nint, e)
	return -1, e
}

// Vmodule implements api method debug_vmodule, enabling setting
// glog per-module verbosity settings on the fly.
func (api *PublicDebugAPI) Vmodule(s string) (string, error) {
	if s == "" {
		return glog.GetVModule().String(), nil
	}
	err := glog.GetVModule().Set(s)
	ns := glog.GetVModule().String()
	if err != nil {
		glog.V(logger.Warn).Infof("Set vmodule: '%s'", ns)
		glog.D(logger.Warn).Warnf("Set vmodule: '%s'", ns)
	} else {
		glog.V(logger.Warn).Infof("Set vmodule failed: '%s' (%v)", ns, err)
		glog.D(logger.Warn).Warnf("Set vmodule: '%s' (%v)", ns, err)
	}
	return ns, err
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as the amount of
// gas used and the return value
type ExecutionResult struct {
	Gas         *big.Int `json:"gas"`
	ReturnValue string   `json:"returnValue"`
}

// TraceCall executes a call and returns the amount of gas and optionally returned values.
func (s *PublicBlockChainAPI) TraceCall(args CallArgs, blockNr rpc.BlockNumber) (*ExecutionResult, error) {
	// Fetch the state associated with the block number
	stateDb, block, err := stateAndBlockByNumber(s.miner, s.bc, blockNr, s.chainDb)
	if stateDb == nil || err != nil {
		return nil, err
	}
	stateDb = stateDb.Copy()

	// Retrieve the account state object to interact with
	var from *state.StateObject
	if args.From == (common.Address{}) {
		accounts := s.am.Accounts()
		if len(accounts) == 0 {
			from = stateDb.GetOrNewStateObject(common.Address{})
		} else {
			from = stateDb.GetOrNewStateObject(accounts[0].Address)
		}
	} else {
		from = stateDb.GetOrNewStateObject(args.From)
	}
	from.SetBalance(common.MaxBig)

	// Assemble the CALL invocation
	msg := callmsg{
		from:     from,
		to:       args.To,
		gas:      args.Gas.BigInt(),
		gasPrice: args.GasPrice.BigInt(),
		value:    args.Value.BigInt(),
		data:     common.FromHex(args.Data),
	}
	if msg.gas.Sign() == 0 {
		msg.gas = big.NewInt(50000000)
	}
	if msg.gasPrice.Sign() == 0 {
		msg.gasPrice = new(big.Int).Mul(big.NewInt(50), common.Shannon)
	}

	// Execute the call and return
	vmenv := core.NewEnv(stateDb, s.config, s.bc, msg, block.Header())
	gp := new(core.GasPool).AddGas(common.MaxBig)

	ret, gas, err := core.ApplyMessage(vmenv, msg, gp)
	return &ExecutionResult{
		Gas:         gas,
		ReturnValue: fmt.Sprintf("%x", ret),
	}, nil
}

// TraceTransaction returns the amount of gas and execution result of the given transaction.
func (s *PublicDebugAPI) TraceTransaction(txHash common.Hash) (*ExecutionResult, error) {
	var result *ExecutionResult
	tx, blockHash, _, txIndex := core.GetTransaction(s.eth.ChainDb(), txHash)
	if tx == nil {
		return result, fmt.Errorf("tx '%x' not found", txHash)
	}

	msg, vmenv, err := s.computeTxEnv(blockHash, int(txIndex))
	if err != nil {
		return nil, err
	}

	gp := new(core.GasPool).AddGas(tx.Gas())
	ret, gas, err := core.ApplyMessage(vmenv, msg, gp)
	return &ExecutionResult{
		Gas:         gas,
		ReturnValue: fmt.Sprintf("%x", ret),
	}, nil
}

// computeTxEnv returns the execution environment of a certain transaction.
func (s *PublicDebugAPI) computeTxEnv(blockHash common.Hash, txIndex int) (core.Message, *core.VMEnv, error) {

	// Create the parent state.
	block := s.eth.BlockChain().GetBlock(blockHash)
	if block == nil {
		return nil, nil, fmt.Errorf("block %x not found", blockHash)
	}
	parent := s.eth.BlockChain().GetBlock(block.ParentHash())
	if parent == nil {
		return nil, nil, fmt.Errorf("block parent %x not found", block.ParentHash())
	}
	statedb, err := s.eth.BlockChain().StateAt(parent.Root())
	if err != nil {
		return nil, nil, err
	}
	txs := block.Transactions()

	// Recompute transactions up to the target index.
	for idx, tx := range txs {
		// Assemble the transaction call message
		// Retrieve the account state object to interact with
		var from *state.StateObject
		fromAddress, e := tx.From()
		if e != nil {
			return nil, nil, e
		}
		if fromAddress == (common.Address{}) {
			from = statedb.GetOrNewStateObject(common.Address{})
		} else {
			from = statedb.GetOrNewStateObject(fromAddress)
		}

		msg := callmsg{
			from:     from,
			to:       tx.To(),
			gas:      tx.Gas(),
			gasPrice: tx.GasPrice(),
			value:    tx.Value(),
			data:     tx.Data(),
		}

		vmenv := core.NewEnv(statedb, s.eth.chainConfig, s.eth.BlockChain(), msg, block.Header())
		if idx == txIndex {
			return msg, vmenv, nil
		}

		gp := new(core.GasPool).AddGas(tx.Gas())
		_, _, err := core.ApplyMessage(vmenv, msg, gp)
		if err != nil {
			return nil, nil, fmt.Errorf("tx %x failed: %v", tx.Hash(), err)
		}
		statedb.DeleteSuicides()
	}
	return nil, nil, fmt.Errorf("tx index %d out of range for block %x", txIndex, blockHash)
}

// PublicNetAPI offers network related RPC methods
type PublicNetAPI struct {
	net            *p2p.Server
	networkVersion int
}

// NewPublicNetAPI creates a new net API instance.
func NewPublicNetAPI(net *p2p.Server, networkVersion int) *PublicNetAPI {
	return &PublicNetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *PublicNetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (s *PublicNetAPI) PeerCount() *rpc.HexNumber {
	return rpc.NewHexNumber(s.net.PeerCount())
}

// Version returns the current ethereum protocol version.
func (s *PublicNetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}
