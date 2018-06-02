package downloader

import (
	"github.com/ethereumproject/go-ethereum/common"
	"time"
)

type InsertChainEvent struct {
	Processed       int
	Queued          int
	Ignored         int
	TxCount         int
	LastNumber      uint64
	LastHash        common.Hash
	Elasped         time.Duration
	LatestBlockTime time.Time
}
