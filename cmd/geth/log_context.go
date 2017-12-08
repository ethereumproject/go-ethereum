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
	var err error
	glog.MaxSize, err = getSizeFlagValue(&LogMaxSizeFlag, ctx)
	if err != nil {
		return err
	}

	glog.MinSize, err = getSizeFlagValue(&LogMinSizeFlag, ctx)
	if err != nil {
		return err
	}

	glog.MaxTotalSize, err = getSizeFlagValue(&LogMaxTotalSizeFlag, ctx)
	if err != nil {
		return err
	}

	glog.Compress = ctx.GlobalBool(aliasableName(LogCompressFlag.Name, ctx))

	interval, err := glog.ParseInterval(ctx.GlobalString(aliasableName(LogIntervalFlag.Name, ctx)))
	if err != nil {
		return fmt.Errorf("invalid log rotation interval: %v", err)
	}
	glog.RotationInterval = interval

	maxAge, err := parseDuration(ctx.GlobalString(aliasableName(LogMaxAgeFlag.Name, ctx)))
	if err != nil {
		return fmt.Errorf("error parsing log max age: %v", err)
	}
	glog.MaxAge = maxAge

	return nil
}

func getSizeFlagValue(flag *cli.StringFlag, ctx *cli.Context) (uint64, error) {
	strVal := ctx.GlobalString(aliasableName(flag.Name, ctx))
	size, err := parseSize(strVal)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid value '%s': %v", flag.Name, strVal, err)
	}
	return size, nil
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
	number := strings.ToLower(strings.TrimLeftFunc(str, unicode.IsSpace))

	trim := ""
	for k, _ := range mapping {
		if k != 0 {
			trim += string(k)
		}
	}
	suffix := rune(0)
	number = strings.TrimRightFunc(number, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		if unicode.IsDigit(r) {
			return false
		}
		if suffix == 0 {
			suffix = r
			return true
		}
		return false
	})

	if suffix != 0 && !strings.ContainsRune(trim, suffix) {
		return 0, fmt.Errorf("invalid suffix '%v', expected one of %v", string(suffix), strings.Split(trim, ""))
	}

	value, err := strconv.ParseUint(number, 10, 64)

	if err != nil {
		return 0, fmt.Errorf("invalid value '%v': natural number expected", number)
	}

	return value * mapping[suffix], nil
}
