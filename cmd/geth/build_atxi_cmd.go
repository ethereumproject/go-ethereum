package main

import (
	"math"

	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
)

var buildAddrTxIndexCommand = cli.Command{
	Action: buildAddrTxIndexCmd,
	Name:   "atxi-build",
	Usage:  "Generate index for transactions by address",
	Description: `
	Builds an index for transactions by address. 
	The command is idempotent; it will not hurt to run multiple times on the same range.
	If run without --start flag, the command makes use of a persistent placeholder, so you can
	run the command on multiple occasions and pick up indexing progress where the last session
	left off.
	To enable address-transaction indexing during block sync and import, use the '--atxi' flag.
			`,
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:  "start",
			Usage: "Block number at which to begin building index",
		},
		cli.IntFlag{
			Name:  "stop",
			Usage: "Block number at which to stop building index",
		},
		cli.IntFlag{
			Name:  "step",
			Usage: "Step increment for batching. Higher number requires more mem, but may be faster",
			Value: 10000,
		},
	},
}

func buildAddrTxIndexCmd(ctx *cli.Context) error {
	// Divide global cache availability equally between chaindata (pre-existing blockdata) and
	// address-transaction database. This ratio is arbitrary and could potentially be optimized or delegated to be user configurable.
	ethdb.SetCacheRatio("chaindata", 0.5)
	ethdb.SetHandleRatio("chaindata", 1)
	ethdb.SetCacheRatio("indexes", 0.5)
	ethdb.SetHandleRatio("indexes", 1)

	var startIndex uint64 = math.MaxUint64
	if ctx.IsSet("start") {
		startIndex = uint64(ctx.Int("start"))
	}
	stopIndex := uint64(ctx.Int("stop"))
	step := uint64(ctx.Int("step"))

	indexDB := MakeIndexDatabase(ctx)
	if indexDB == nil {
		glog.Fatalln("can't open index database")
	}
	defer indexDB.Close()

	bc, chainDB := MakeChain(ctx)
	if bc == nil || chainDB == nil {
		glog.Fatalln("can't open chain database")
	}
	defer chainDB.Close()

	bc.SetAtxi(&core.AtxiT{Db: indexDB, AutoMode: false, Progress: &core.AtxiProgressT{}})
	return core.BuildAddrTxIndex(bc, chainDB, indexDB, startIndex, stopIndex, step)
}
