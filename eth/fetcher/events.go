package fetcher

import "github.com/ethereumproject/go-ethereum/core/types"

type FetcherInsertBlockEvent struct {
	Peer  string
	Block *types.Block
}
