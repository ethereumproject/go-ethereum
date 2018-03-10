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

// This file contains some shares testing functionality, common to  multiple
// different files and modules being tested.

package eth

import (
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"
	"sync"
	"testing"

	"github.com/ellaism/go-ellaism/common"
	"github.com/ellaism/go-ellaism/core"
	"github.com/ellaism/go-ellaism/core/types"
	"github.com/ellaism/go-ellaism/crypto"
	"github.com/ellaism/go-ellaism/ethdb"
	"github.com/ellaism/go-ellaism/event"
	"github.com/ellaism/go-ellaism/p2p"
	"github.com/ellaism/go-ellaism/p2p/discover"
)

var (
	testBankKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testBank       = core.GenesisAccount{
		Address: crypto.PubkeyToAddress(testBankKey.PublicKey),
		Balance: big.NewInt(1000000),
	}
)

// newTestProtocolManager creates a new protocol manager for testing purposes,
// with the given number of blocks already known, and potential notification
// channels for different events.
func newTestProtocolManager(fastSync bool, blocks int, generator func(int, *core.BlockGen), newtx chan<- []*types.Transaction) (*ProtocolManager, error) {
	var (
		evmux       = new(event.TypeMux)
		pow         = new(core.FakePow)
		db, _       = ethdb.NewMemDatabase()
		genesis     = core.WriteGenesisBlockForTesting(db, testBank)
		chainConfig = &core.ChainConfig{
			Forks: []*core.Fork{
				{
					Name:  "Homestead",
					Block: big.NewInt(0),
				},
			},
		}
		blockchain, _ = core.NewBlockChain(db, chainConfig, pow, evmux)
	)

	chain, _ := core.GenerateChain(core.DefaultConfigMorden.ChainConfig, genesis, db, blocks, generator)
	if _, err := blockchain.InsertChain(chain); err != nil {
		panic(err)
	}

	pm, err := NewProtocolManager(chainConfig, fastSync, NetworkId, evmux, &testTxPool{added: newtx}, pow, blockchain, db)
	if err != nil {
		return nil, err
	}
	pm.Start()
	return pm, nil
}

// newTestProtocolManagerMust creates a new protocol manager for testing purposes,
// with the given number of blocks already known, and potential notification
// channels for different events. In case of an error, the constructor force-
// fails the test.
func newTestProtocolManagerMust(t *testing.T, fastSync bool, blocks int, generator func(int, *core.BlockGen), newtx chan<- []*types.Transaction) *ProtocolManager {
	pm, err := newTestProtocolManager(fastSync, blocks, generator, newtx)
	if err != nil {
		t.Fatalf("Failed to create protocol manager: %v", err)
	}
	return pm
}

// testTxPool is a fake, helper transaction pool for testing purposes
type testTxPool struct {
	pool  []*types.Transaction        // Collection of all transactions
	added chan<- []*types.Transaction // Notification channel for new transactions

	lock sync.RWMutex // Protects the transaction pool
}

// AddTransactions appends a batch of transactions to the pool, and notifies any
// listeners if the addition channel is non nil
func (p *testTxPool) AddTransactions(txs []*types.Transaction) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.pool = append(p.pool, txs...)
	if p.added != nil {
		p.added <- txs
	}
}

// GetTransactions returns all the transactions known to the pool
func (p *testTxPool) GetTransactions() types.Transactions {
	p.lock.RLock()
	defer p.lock.RUnlock()

	txs := make([]*types.Transaction, len(p.pool))
	copy(txs, p.pool)

	return txs
}

// newTestTransaction create a new dummy transaction.
func newTestTransaction(from *ecdsa.PrivateKey, nonce uint64, datasize int) *types.Transaction {
	tx := types.NewTransaction(nonce, common.Address{}, big.NewInt(0), big.NewInt(100000), big.NewInt(0), make([]byte, datasize))
	tx, _ = tx.SignECDSA(from)
	return tx
}

// testPeer is a simulated peer to allow testing direct network calls.
type testPeer struct {
	net p2p.MsgReadWriter // Network layer reader/writer to simulate remote messaging
	app *p2p.MsgPipeRW    // Application layer reader/writer to simulate the local side
	*peer
}

// newTestPeer creates a new peer registered at the given protocol manager.
func newTestPeer(name string, version int, pm *ProtocolManager, shake bool) (*testPeer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id discover.NodeID
	rand.Read(id[:])

	peer := pm.newPeer(version, p2p.NewPeer(id, name, nil), net)

	// Start the peer on a new thread
	errc := make(chan error, 1)
	go func() {
		select {
		case pm.newPeerCh <- peer:
			errc <- pm.handle(peer)
		case <-pm.quitSync:
			errc <- p2p.DiscQuitting
		}
	}()
	tp := &testPeer{app: app, net: net, peer: peer}
	// Execute any implicitly requested handshakes and return
	if shake {
		td, head, genesis := pm.blockchain.Status()
		tp.handshake(nil, td, head, genesis)
	}
	return tp, errc
}

// handshake simulates a trivial handshake that expects the same state from the
// remote side as we are simulating locally.
func (p *testPeer) handshake(t *testing.T, td *big.Int, head common.Hash, genesis common.Hash) {
	msg := &statusData{
		ProtocolVersion: uint32(p.version),
		NetworkId:       uint32(NetworkId),
		TD:              td,
		CurrentBlock:    head,
		GenesisBlock:    genesis,
	}
	if err := p2p.ExpectMsg(p.app, StatusMsg, msg); err != nil {
		t.Fatalf("status recv: %v", err)
	}
	if err := p2p.Send(p.app, StatusMsg, msg); err != nil {
		t.Fatalf("status send: %v", err)
	}
}

// close terminates the local side of the peer, notifying the remote protocol
// manager of termination.
func (p *testPeer) close() {
	p.app.Close()
}
