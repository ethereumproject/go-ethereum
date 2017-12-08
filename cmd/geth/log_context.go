package main

import (
	"fmt"
	"time"

	"gopkg.in/urfave/cli.v1"

	"github.com/ethereumproject/go-ethereum/logger/glog"
)

func setupLogRotation(ctx *cli.Context) error {
	// log-rotation options
	maxSize := ctx.GlobalInt(aliasableName(LogMaxSizeFlag.Name, ctx))
	if maxSize < 0 {
		maxSize = 0
	}
	glog.MaxSize = uint64(maxSize)

	minSize := ctx.GlobalInt(aliasableName(LogMinSizeFlag.Name, ctx))
	if minSize < 0 {
		minSize = 0
	}
	glog.MinSize = uint64(minSize)

	maxTotalSize := ctx.GlobalInt(aliasableName(LogMaxTotalSizeFlag.Name, ctx))
	if maxTotalSize < 0 {
		maxTotalSize = 0
	}
	glog.MaxTotalSize = uint64(maxTotalSize)

	glog.Compress = ctx.GlobalBool(aliasableName(LogCompressFlag.Name, ctx))

	interval, err := glog.ParseInterval(ctx.GlobalString(aliasableName(LogIntervalFlag.Name, ctx)))
	if err != nil {
		return fmt.Errorf("invalid log rotation interval: %v", err)
	}
	glog.RotationInterval = interval

	maxAgeString := ctx.GlobalString(aliasableName(LogMaxAgeFlag.Name, ctx))
	maxAge, err := time.ParseDuration(maxAgeString)
	if err != nil {
		return fmt.Errorf("error parsing log max age: %v", err)
	}
	glog.MaxAge = maxAge

	return nil
}
