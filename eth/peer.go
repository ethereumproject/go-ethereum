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
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/core/types"
	"github.com/eth-classic/go-ethereum/logger"
	"github.com/eth-classic/go-ethereum/logger/glog"
	"github.com/eth-classic/go-ethereum/p2p"
	"github.com/eth-classic/go-ethereum/rlp"
	"gopkg.in/fatih/set.v0"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const (
	maxKnownTxs    = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks = 1024  // Maximum block hashes to keep in the known list (prevent DOS)

	// maxQueuedTxs is the maximum number of transaction lists to queue up before
	// dropping broadcasts. This is a sensitive number as a transaction list might
	// contain a single transaction, or thousands.
	maxQueuedTxs = 128

	// maxQueuedProps is the maximum number of block propagations to queue up before
	// dropping broadcasts. There's not much point in queueing stale blocks, so a few
	// that might cover uncles should be enough.
	maxQueuedProps = 4

	// maxQueuedAnns is the maximum number of block announcements to queue up before
	// dropping broadcasts. Similarly to block propagations, there's no point to queue
	// above some healthy uncle limit, so use that.
	maxQueuedAnns = 4

	handshakeTimeout = 5 * time.Second
)

// PeerInfo represents a short summary of the Ethereum sub-protocol metadata known
// about a connected peer.
type PeerInfo struct {
	Version    int      `json:"version"`    // Ethereum protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

// propEvent is a block propagation, waiting for its turn in the broadcast queue.
type propEvent struct {
	block *types.Block
	td    *big.Int
}

type peer struct {
	id string

	*p2p.Peer
	rw p2p.MsgReadWriter

	version  int         // Protocol version negotiated
	forkDrop *time.Timer // Timed connection dropper if forks aren't validated in time

	head common.Hash
	td   *big.Int
	lock sync.RWMutex

	knownTxs    *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks *set.Set // Set of block hashes known to be known by this peer

	queuedTxs   chan []*types.Transaction // Queue of transactions to broadcast to the peer
	queuedProps chan *propEvent           // Queue of blocks to broadcast to the peer
	queuedAnns  chan *types.Block         // Queue of blocks to announce to the peer
	term        chan struct{}             // Termination channel to stop the broadcaster
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	id := p.ID()

	return &peer{
		Peer:        p,
		rw:          rw,
		version:     version,
		id:          fmt.Sprintf("%x", id[:8]),
		knownTxs:    set.New(),
		knownBlocks: set.New(),
		queuedTxs:   make(chan []*types.Transaction, maxQueuedTxs),
		queuedProps: make(chan *propEvent, maxQueuedProps),
		queuedAnns:  make(chan *types.Block, maxQueuedAnns),
		term:        make(chan struct{}),
	}
}

// broadcast is a write loop that multiplexes block propagations, announcements
// and transaction broadcasts into the remote peer. The goal is to have an async
// writer that does not lock up node internals.
func (p *peer) broadcast() {
	for {
		select {
		case txs := <-p.queuedTxs:
			if err := p.SendTransactions(txs); err != nil {
				return
			}
			glog.V(logger.Detail).Infoln("Broadcast transactions", "count", len(txs))

		case prop := <-p.queuedProps:
			if err := p.SendNewBlock(prop.block, prop.td); err != nil {
				return
			}
			glog.V(logger.Detail).Infoln("Propagated block", "number", prop.block.Number(), "hash", prop.block.Hash().Hex(), "td", prop.td)

		case block := <-p.queuedAnns:
			if err := p.SendNewBlockHashes([]common.Hash{block.Hash()}, []uint64{block.NumberU64()}); err != nil {
				return
			}
			glog.V(logger.Detail).Infoln("Announced block", "number", block.Number(), "hash", block.Hash().Hex())

		case <-p.term:
			return
		}
	}
}

// close signals the broadcast goroutine to terminate.
func (p *peer) close() {
	close(p.term)
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *PeerInfo {
	hash, td := p.Head()

	return &PeerInfo{
		Version:    p.version,
		Difficulty: td,
		Head:       hash.Hex(),
	}
}

// Head retrieves a copy of the current head hash and total difficulty of the
// peer.
func (p *peer) Head() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

// SetHead updates the head hash and total difficulty of the peer.
func (p *peer) SetHead(hash common.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.td.Set(td)
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (p *peer) MarkBlock(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known block hash
	for p.knownBlocks.Size() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash)
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Size() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) SendTransactions(txs types.Transactions) error {
	for _, tx := range txs {
		p.knownTxs.Add(tx.Hash())
	}
	s, e := p2p.Send(p.rw, TxMsg, txs)
	mlogWireDelegate(p, "send", TxMsg, s, txs, nil)
	return e
}

// AsyncSendTransactions queues list of transactions propagation to a remote
// peer. If the peer's broadcast queue is full, the event is silently dropped.
func (p *peer) AsyncSendTransactions(txs []*types.Transaction) {
	select {
	case p.queuedTxs <- txs:
		for _, tx := range txs {
			p.knownTxs.Add(tx.Hash())
		}
	default:
		glog.V(logger.Debug).Infoln("Dropping transaction propagation", "count", len(txs))
	}
}

// SendNewBlockHashes announces the availability of a number of blocks through
// a hash notification.
func (p *peer) SendNewBlockHashes(hashes []common.Hash, numbers []uint64) error {
	for _, hash := range hashes {
		p.knownBlocks.Add(hash)
	}
	request := make(newBlockHashesData, len(hashes))
	for i := 0; i < len(hashes); i++ {
		request[i].Hash = hashes[i]
		request[i].Number = numbers[i]
	}
	s, e := p2p.Send(p.rw, NewBlockHashesMsg, request)
	mlogWireDelegate(p, "send", NewBlockHashesMsg, s, request, nil)
	return e
}

// AsyncSendNewBlockHash queues the availability of a block for propagation to a
// remote peer. If the peer's broadcast queue is full, the event is silently
// dropped.
func (p *peer) AsyncSendNewBlockHash(block *types.Block) {
	select {
	case p.queuedAnns <- block:
		p.knownBlocks.Add(block.Hash())
	default:
		glog.V(logger.Debug).Infoln("Dropping block announcement", "number", block.NumberU64(), "hash", block.Hash())
	}
}

// SendNewBlock propagates an entire block to a remote peer.
func (p *peer) SendNewBlock(block *types.Block, td *big.Int) error {
	p.knownBlocks.Add(block.Hash())
	s, e := p2p.Send(p.rw, NewBlockMsg, []interface{}{block, td})
	mlogWireDelegate(p, "send", NewBlockMsg, s, newBlockData{Block: block, TD: td}, nil) // send slice of len 1
	return e
}

// AsyncSendNewBlock queues an entire block for propagation to a remote peer. If
// the peer's broadcast queue is full, the event is silently dropped.
func (p *peer) AsyncSendNewBlock(block *types.Block, td *big.Int) {
	select {
	case p.queuedProps <- &propEvent{block: block, td: td}:
		p.knownBlocks.Add(block.Hash())
	default:
		glog.V(logger.Debug).Infoln("Dropping block propagation", "number", block.NumberU64(), "hash", block.Hash())
	}
}

// SendBlockHeaders sends a batch of block headers to the remote peer.m
func (p *peer) SendBlockHeaders(headers []*types.Header) error {
	s, e := p2p.Send(p.rw, BlockHeadersMsg, headers)
	mlogWireDelegate(p, "send", BlockHeadersMsg, s, headers, nil)
	return e
}

// SendBlockBodies sends a batch of block contents to the remote peer.
func (p *peer) SendBlockBodies(bodies []*blockBody) error {
	s, e := p2p.Send(p.rw, BlockBodiesMsg, blockBodiesData(bodies))
	mlogWireDelegate(p, "send", BlockBodiesMsg, s, bodies, nil)
	return e
}

// SendBlockBodiesRLP sends a batch of block contents to the remote peer from
// an already RLP encoded format.
func (p *peer) SendBlockBodiesRLP(bodies []rlp.RawValue) error {
	s, e := p2p.Send(p.rw, BlockBodiesMsg, bodies)
	mlogWireDelegate(p, "send", BlockBodiesMsg, s, bodies, nil)
	return e
}

// SendNodeDataRLP sends a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (p *peer) SendNodeData(data [][]byte) error {
	s, e := p2p.Send(p.rw, NodeDataMsg, data)
	mlogWireDelegate(p, "send", NodeDataMsg, s, data, nil)
	return e
}

// SendReceiptsRLP sends a batch of transaction receipts, corresponding to the
// ones requested from an already RLP encoded format.
func (p *peer) SendReceiptsRLP(receipts []rlp.RawValue) error {
	s, e := p2p.Send(p.rw, ReceiptsMsg, receipts)
	mlogWireDelegate(p, "send", ReceiptsMsg, s, receipts, nil)
	return e
}

// RequestHeaders is a wrapper around the header query functions to fetch a
// single header. It is used solely by the fetcher.
func (p *peer) RequestOneHeader(hash common.Hash) error {
	glog.V(logger.Debug).Infof("fetching from: %v req=singleheader hash=%x", p, hash)
	d := &getBlockHeadersData{Origin: hashOrNumber{Hash: hash}, Amount: uint64(1), Skip: uint64(0), Reverse: false}
	s, e := p2p.Send(p.rw, GetBlockHeadersMsg, d)
	mlogWireDelegate(p, "send", GetBlockHeadersMsg, s, d, nil)
	return e
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	glog.V(logger.Debug).Infof("fetching from: %v req=headersbyhash n=%d origin=%x, skipping=%d reverse=%v", p, amount, origin[:4], skip, reverse)
	d := &getBlockHeadersData{Origin: hashOrNumber{Hash: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse}
	s, e := p2p.Send(p.rw, GetBlockHeadersMsg, d)
	mlogWireDelegate(p, "send", GetBlockHeadersMsg, s, d, nil)
	return e
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	glog.V(logger.Debug).Infof("fetching from: %v %d req=headersbynumber n=%d, skipping=%d reverse=%v", p, amount, origin, skip, reverse)
	d := &getBlockHeadersData{Origin: hashOrNumber{Number: origin}, Amount: uint64(amount), Skip: uint64(skip), Reverse: reverse}
	s, e := p2p.Send(p.rw, GetBlockHeadersMsg, d)
	mlogWireDelegate(p, "send", GetBlockHeadersMsg, s, d, nil)
	return e
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *peer) RequestBodies(hashes []common.Hash) error {
	glog.V(logger.Debug).Infof("fetching from: %v req=blockbodies n=%d first=%s", p, len(hashes), hashes[0].Hex())
	s, e := p2p.Send(p.rw, GetBlockBodiesMsg, hashes)
	mlogWireDelegate(p, "send", GetBlockBodiesMsg, s, hashes, nil)
	return e
}

// RequestNodeData fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (p *peer) RequestNodeData(hashes []common.Hash) error {
	glog.V(logger.Debug).Infof("fetching from: %v req=statedata n=%d first=%s", p, len(hashes), hashes[0].Hex())
	s, e := p2p.Send(p.rw, GetNodeDataMsg, hashes)
	mlogWireDelegate(p, "send", GetNodeDataMsg, s, hashes, nil)
	return e
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *peer) RequestReceipts(hashes []common.Hash) error {
	glog.V(logger.Debug).Infof("fetching from: %v req=receipts n=%d first=%s", p, len(hashes), hashes[0].Hex())
	s, e := p2p.Send(p.rw, GetReceiptsMsg, hashes)
	mlogWireDelegate(p, "send", GetReceiptsMsg, s, hashes, nil)
	return e
}

// Handshake executes the eth protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *peer) Handshake(network uint64, td *big.Int, head common.Hash, genesis common.Hash) error {
	// Send out own handshake in a new thread
	sendErrc := make(chan error, 1)
	recErrc := make(chan error, 1)
	var sendErr, recErr error
	var sendSize, recSize int
	var status statusData // safe to read after two values have been received from errc

	d := &statusData{
		ProtocolVersion: uint32(p.version),
		NetworkId:       uint32(network),
		TD:              td,
		CurrentBlock:    head,
		GenesisBlock:    genesis,
	}

	go func() {
		var e error
		sendSize, e = p2p.Send(p.rw, StatusMsg, d)
		sendErrc <- e
	}()
	go func() {
		var e error
		var s uint32
		s, e = p.readStatusReturnSize(network, &status, genesis)
		recSize = int(s)
		recErrc <- e
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()

	defer mlogWireDelegate(p, "receive", StatusMsg, recSize, &status, recErr)
	defer mlogWireDelegate(p, "send", StatusMsg, sendSize, d, sendErr)

	for i := 0; i < 2; i++ {
		select {
		case err := <-sendErrc:
			if err != nil {
				sendErr = err
				return err
			}
		case err := <-recErrc:
			if err != nil {
				recErr = err
				return err
			}
		case <-timeout.C:
			recErr = p2p.DiscReadTimeout
			return p2p.DiscReadTimeout
		}
	}
	p.td, p.head = status.TD, status.CurrentBlock
	return nil
}

func (p *peer) readStatusReturnSize(network uint64, status *statusData, genesis common.Hash) (size uint32, err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return msg.Size, err
	}
	if msg.Code != StatusMsg {
		return msg.Size, errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return msg.Size, errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return msg.Size, errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return msg.Size, errResp(ErrGenesisBlockMismatch, "%x (!= %x…)", status.GenesisBlock, genesis.Bytes()[:8])
	}
	if status.NetworkId != uint32(network) {
		return msg.Size, errResp(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, network)
	}
	if int(status.ProtocolVersion) != p.version {
		return msg.Size, errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	return msg.Size, nil
}

func (p *peer) readStatus(network uint64, status *statusData, genesis common.Hash) (err error) {
	_, err = p.readStatusReturnSize(network, status, genesis)
	return
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	// id is %x[:8]
	return fmt.Sprintf("peer:%s@[%s] eth/%d", p.id, p.Name(), p.version)
}
