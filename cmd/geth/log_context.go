package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gopkg.in/urfave/cli.v1"

	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"os"
	"path/filepath"
)

const defaultStatusLog = "sync=60s"

var isToFileLoggingEnabled = true

// setupLogging sets default
func setupLogging(ctx *cli.Context) error {
	glog.CopyStandardLogTo("INFO")

	// Turn on only file logging, disabling logging(T).toStderr and logging(T).alsoToStdErr
	glog.SetToStderr(glog.DefaultToStdErr)
	glog.SetAlsoToStderr(glog.DefaultAlsoToStdErr)

	glog.SetV(glog.DefaultVerbosity)

	// Set up file logging.
	logDir := ""
	isToFileLoggingEnabled = toFileLoggingEnabled(ctx)

	// If '--log-dir' flag is in use, override the default.
	if ctx.GlobalIsSet(aliasableName(LogDirFlag.Name, ctx)) {
		ld := ctx.GlobalString(aliasableName(LogDirFlag.Name, ctx))
		if ld == "" {
			return fmt.Errorf("--%s cannot be empty", LogDirFlag.Name)
		}
		if isToFileLoggingEnabled {
			ld = expandPath(ld)
			ldAbs, err := filepath.Abs(ld)
			if err != nil {
				return err
			}
			logDir = ldAbs
		} else {
			glog.SetD(0)
			glog.SetToStderr(true)
		}
	} else {
		logDir = filepath.Join(MustMakeChainDataDir(ctx), glog.DefaultLogDirName)
	}

	// Allow to-file logging to be disabled
	if logDir != "" {
		// Ensure log dir exists; mkdir -p <logdir>
		if e := os.MkdirAll(logDir, os.ModePerm); e != nil {
			return e
		}

		// Before glog.SetLogDir is called, logs are saved to system-default temporary directory.
		// If logging is started before this call, the new logDir will be used after file rotation
		// (by default after 1800MB of data per file).
		glog.SetLogDir(logDir)
	}

	// Handle --neckbeard config overrides if set.
	if ctx.GlobalBool(NeckbeardFlag.Name) {
		glog.SetD(0)
		glog.SetV(5)
		glog.SetAlsoToStderr(true)
	}

	// Handle display level configuration.
	if ctx.GlobalIsSet(DisplayFlag.Name) {
		i := ctx.GlobalInt(DisplayFlag.Name)
		if i > 5 {
			return fmt.Errorf("--%s level must be 0 <= i <= 5, got: %d", DisplayFlag.Name, i)
		}
		glog.SetD(i)
	}

	// Manual context configs
	// Global V verbosity
	if ctx.GlobalIsSet(VerbosityFlag.Name) {
		nint := ctx.GlobalInt(VerbosityFlag.Name)
		if nint <= logger.Detail || nint == logger.Ridiculousness {
			glog.SetV(nint)
		}
	}

	// Global Vmodule
	if ctx.GlobalIsSet(VModuleFlag.Name) {
		v := ctx.GlobalString(VModuleFlag.Name)
		glog.GetVModule().Set(v)
	}

	// If --log-status not set, set default 60s interval
	if !ctx.GlobalIsSet(LogStatusFlag.Name) {
		ctx.Set(LogStatusFlag.Name, defaultStatusLog)
	}

	return nil
}

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
		's': uint64(time.Second),
		'm': uint64(time.Minute),
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

func toFileLoggingEnabled(ctx *cli.Context) bool {
	if ctx.GlobalIsSet(aliasableName(LogDirFlag.Name, ctx)) {
		ld := ctx.GlobalString(aliasableName(LogDirFlag.Name, ctx))
		if ld == "off" || ld == "disable" || ld == "disabled" {
			return false
		}
	}
	return true
}

func mustMakeMLogDir(ctx *cli.Context) string {
	if ctx.GlobalIsSet(MLogDirFlag.Name) {
		p := ctx.GlobalString(MLogDirFlag.Name)
		if p == "" {
			glog.Fatalf("Flag %v requires a non-empty argument", MLogDirFlag.Name)
			return ""
		}
		if filepath.IsAbs(p) {
			return p
		}
		ap, e := filepath.Abs(p)
		if e != nil {
			glog.Fatalf("could not establish absolute path for mlog dir: %v", e)
		}
		return ap
	}

	return filepath.Join(MustMakeChainDataDir(ctx), "mlogs")
}

func makeMLogFileLogger(ctx *cli.Context) (string, error) {
	now := time.Now()

	mlogdir := mustMakeMLogDir(ctx)
	logger.SetMLogDir(mlogdir)

	_, filename, err := logger.CreateMLogFile(now)
	if err != nil {
		return "", err
	}
	// withTs toggles custom timestamp ISO8601 prefix
	// logger print without timestamp header prefix if json
	withTs := true
	if f := ctx.GlobalString(MLogFlag.Name); logger.MLogStringToFormat[f] == logger.MLOGJSON {
		withTs = false
	}
	logger.BuildNewMLogSystem(mlogdir, filename, 1, 0, withTs) // flags: 0 disables automatic log package time prefix
	return filename, nil
}

func mustRegisterMLogsFromContext(ctx *cli.Context) {
	if e := logger.MLogRegisterComponentsFromContext(ctx.GlobalString(MLogComponentsFlag.Name)); e != nil {
		// print documentation if user enters unavailable mlog component
		var components []string
		for k := range logger.MLogRegistryAvailable {
			components = append(components, string(k))
		}
		glog.V(logger.Error).Errorf("Error: %s", e)
		glog.V(logger.Error).Errorf("Available machine log components: %v", components)
		os.Exit(1)
	}
	// Set the global logger mlog format from context
	if e := logger.SetMLogFormatFromString(ctx.GlobalString(MLogFlag.Name)); e != nil {
		glog.Fatalf("Error setting mlog format: %v, value was: %v", e, ctx.GlobalString(MLogFlag.Name))
	}
	_, e := makeMLogFileLogger(ctx)
	if e != nil {
		glog.Fatalf("Failed to start machine log: %v", e)
	}
	logger.SetMlogEnabled(true)
}

func logLoggingConfiguration(ctx *cli.Context) {
	v := glog.GetVerbosity().String()
	logdir := "off"
	if isToFileLoggingEnabled {
		logdir = glog.GetLogDir()
	}
	vmodule := glog.GetVModule().String()
	// An empty string looks unused, so show * instead, which is equivalent.
	if vmodule == "" {
		vmodule = "*"
	}
	d := glog.GetDisplayable().String()

	statusFeats := []string{}
	for k, v := range availableLogStatusFeatures {
		if v.Seconds() == 0 {
			statusFeats = append(statusFeats, fmt.Sprintf("%s=%s", k, "off"))
			continue
		}
		statusFeats = append(statusFeats, fmt.Sprintf("%s=%v", k, v))
	}
	statusLine := strings.Join(statusFeats, ",")

	glog.V(logger.Warn).Infoln("Debug log configuration", "v=", v, "logdir=", logdir, "vmodule=", vmodule)
	glog.D(logger.Warn).Infof("Debug log config: verbosity=%s log-dir=%s vmodule=%s",
		logger.ColorGreen(v),
		logger.ColorGreen(logdir),
		logger.ColorGreen(vmodule),
	)

	glog.V(logger.Warn).Infoln("Display log configuration", "d=", d, "status=", statusLine)
	glog.D(logger.Warn).Infof("Display log config: display=%s status=%s",
		logger.ColorGreen(d),
		logger.ColorGreen(statusLine),
	)

	if logger.MlogEnabled() {
		glog.V(logger.Warn).Infof("Machine log config: mlog=%s mlog-dir=%s", logger.GetMLogFormat().String(), logger.GetMLogDir())
		glog.D(logger.Warn).Infof("Machine log config: mlog=%s mlog-dir=%s", logger.ColorGreen(logger.GetMLogFormat().String()), logger.ColorGreen(logger.GetMLogDir()))
	} else {
		glog.V(logger.Warn).Warnf("Machine log config: mlog=%s mlog-dir=%s", logger.GetMLogFormat().String(), logger.GetMLogDir())
		glog.D(logger.Warn).Warnf("Machine log config: mlog=%s", logger.ColorYellow("off"))
	}

}
