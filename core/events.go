// Copyright 2014 The go-ethereum Authors
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

package core

import (
	"math/big"

	"time"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
)

// TxPreEvent is posted when a transaction enters the transaction pool.
type TxPreEvent struct{ Tx *types.Transaction }

// TxPostEvent is posted when a transaction has been processed.
type TxPostEvent struct{ Tx *types.Transaction }

// PendingLogsEvent is posted pre mining and notifies of pending logs.
type PendingLogsEvent struct {
	Logs []*types.Log
}

// PendingStateEvent is posted pre mining and notifies of pending state changes.
type PendingStateEvent struct{}

// NewBlockEvent is posted when a block has been imported.
type NewBlockEvent struct{ Block *types.Block }

// NewMinedBlockEvent is posted when a block has been imported.
type NewMinedBlockEvent struct{ Block *types.Block }

// RemovedTransactionEvent is posted when a reorg happens
type RemovedTransactionEvent struct{ Txs types.Transactions }

// RemovedLogEvent is posted when a reorg happens
type RemovedLogsEvent struct{ Logs []*types.Log }

// ChainSplit is posted when a new head is detected
type ChainSplitEvent struct {
	Block *types.Block
	Logs  []*types.Log
}

type ChainEvent struct {
	Block *types.Block
	Hash  common.Hash
	Logs  []*types.Log
}

type ChainSideEvent struct {
	Block *types.Block
	Logs  []*types.Log
}

// TODO: no usages found in project files
type PendingBlockEvent struct {
	Block *types.Block
	Logs  []*types.Log
}

type ChainInsertEvent struct {
	Processed       int
	Queued          int
	Ignored         int
	TxCount         int
	LastNumber      uint64
	LastHash        common.Hash
	Elasped         time.Duration
	LatestBlockTime time.Time
}

type ReceiptChainInsertEvent struct {
	Processed         int
	Ignored           int
	FirstNumber       uint64
	FirstHash         common.Hash
	LastNumber        uint64
	LastHash          common.Hash
	Elasped           time.Duration
	LatestReceiptTime time.Time
}

type HeaderChainInsertEvent struct {
	Processed  int
	Ignored    int
	LastNumber uint64
	LastHash   common.Hash
	Elasped    time.Duration
}

// TODO: no usages found in project files
type ChainUncleEvent struct {
	Block *types.Block
}

type ChainHeadEvent struct{ Block *types.Block }

type GasPriceChanged struct{ Price *big.Int }

// Mining operation events
type StartMining struct{}
type StopMining struct{}
