package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

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
	maxAge, err := parseDuration(maxAgeString)
	if err != nil {
		return fmt.Errorf("error parsing log max age: %v", err)
	}
	glog.MaxAge = maxAge

	return nil
}

func parseDuration(str string) (time.Duration, error) {
	mapping := map[rune]uint64{
		0:   uint64(time.Second), // no-suffix means value in seconds
		'h': uint64(time.Hour),
		'd': uint64(24 * time.Hour),
		'w': uint64(7 * 24 * time.Hour),
	}
	value, err := parseWithSuffix(str, mapping)
	if err != nil {
		return 0, err
	}
	return time.Duration(value), nil
}

// reinventing the wheel to avoid external dependency
func parseSize(str string) (uint64, error) {
	mapping := map[rune]uint64{
		0:   1, // no-suffix means multiply by 1
		'k': 1024,
		'm': 1024 * 1024,
		'g': 1024 * 1024 * 1024,
	}
	return parseWithSuffix(str, mapping)
}

func parseWithSuffix(str string, mapping map[rune]uint64) (uint64, error) {
	trim := " \t"
	number := strings.ToLower(strings.TrimLeft(str, trim))

	for k, _ := range mapping {
		trim += string(k)
	}
	suffix := rune(0)
	number = strings.TrimRightFunc(number, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		if suffix == 0 && strings.ContainsRune(trim, r) {
			suffix = r
			return true
		}
		return false
	})

	value, err := strconv.ParseUint(number, 10, 64)

	if err != nil {
		return 0, fmt.Errorf("invalid value: '%v', natural number expected", number)
	}

	return value * mapping[suffix], nil
}
