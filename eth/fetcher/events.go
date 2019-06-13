package fetcher

import "github.com/eth-classic/go-ethereum/core/types"

type FetcherInsertBlockEvent struct {
	Peer  string
	Block *types.Block
}
