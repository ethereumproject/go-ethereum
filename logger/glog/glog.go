// Go support for leveled logs, analogous to https://code.google.com/p/google-glog/
//
// Copyright 2013 Google Inc. All Rights Reserved.
// Modifications copyright 2017 ETC Dev Team. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package glog implements logging analogous to the Google-internal C++ INFO/ERROR/V setup.
// It provides functions Info, Warning, Error, Fatal, plus formatting variants such as
// Infof. It also provides V-style logging controlled by the -v and -vmodule=file=2 flags.
//
// Basic examples:
//
//	glog.Info("Prepare to repel boarders")
//
//	glog.Fatalf("Initialization failed: %s", err)
//
// See the documentation for the V function for an explanation of these examples:
//
//	if glog.V(2) {
//		glog.Info("Starting transaction...")
//	}
//
//	glog.V(2).Infoln("Processed", nItems, "elements")
//
// Log output is buffered and written periodically using Flush. Programs
// should call Flush before exiting to guarantee all log output is written.
//
// By default, all log statements write to files in a temporary directory.
// This package provides several flags that modify this behavior.
// As a result, flag.Parse must be called before any logging is done.
//
//	-logtostderr=false
//		Logs are written to standard error instead of to files.
//	-alsologtostderr=false
//		Logs are written to standard error as well as to files.
//	-stderrthreshold=ERROR
//		Log events at or above this severity are logged to standard
//		error as well as to files.
//	-log_dir=""
//		Log files will be written to this directory instead of the
//		default temporary directory.
//
//	Other flags provide aids to debugging.
//
//	-log_backtrace_at=""
//		When set to a file and line number holding a logging statement,
//		such as
//			-log_backtrace_at=gopherflakes.go:234
//		a stack trace will be written to the Info log whenever execution
//		hits that statement. (Unlike with -vmodule, the ".go" must be
//		present.)
//	-v=0
//		Enable V-leveled logging at the specified level.
//	-vmodule=""
//		The syntax of the argument is a comma-separated list of pattern=N,
//		where pattern is a literal file name or "glob" pattern matching
//		and N is a V level. For instance,
//
//	-vmodule=gopher.go=3
//		sets the V level to 3 in all Go files named "gopher.go".
//
//	-vmodule=foo=3
//		sets V to 3 in all files of any packages whose import path ends in "foo".
//
//	-vmodule=foo/*=3
//		sets V to 3 in all files of any packages whose import path contains "foo".
//
// This fork of original golang/glog adds log rotation functionality.
// Logs are rotated after reaching file size limit or age limit. Additionally
// limiting total amount of logs is supported (also by both size and age).
// To keep it simple, log-rotation is configured with package-level variables:
//  - MaxSize - maximum file size (in bytes) - default value: 1024 * 1024 * 1800
//  - MinSize - minimum file size (in bytes) - default 0 (even empty file can be rotated)
//  - MaxTotalSize - maximum size of all files (in bytes) - default 0 (do not remove old files)
//  - RotationInterval - how often log should be rotated - default Never
//  - MaxAge - maximum age (time.Duration) of log file - default 0 (do not remove old files)
//  - Compress - whether to GZIP compress rotated logs - default - false
//
// Default values provide backward-compatibility with golang/glog. If compression is used,
// all files except the current one are compressed with GZIP.
//
// Rotation works like this:
//  - if MaxSize or RotationInterval is reached, and file size is > MinSize,
//    current file became old file, and new file is created as a current log file
//  - all log files older than MaxAge are removed
//  - if compression is enabled, the old file is compressed
//  - size of all log files in log_dir is recalculated (to handle external removals of files, etc)
//  - oldest log files are removed until total size of log files doesn't exceed  MaxTotalSize-MaxSize
// For sanity, this action is executed only when current file is needs to be rotated
//
package glog

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	stdLog "log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultVerbosity establishes the default verbosity Level for
// to-file (debug) logging.
var DefaultVerbosity = 5

// DefaultDisplay establishes the default verbosity Level for
// display (stderr) logging.
var DefaultDisplay = 3

// DefaultToStdErr establishes the default bool toggling whether logging
// should be directed ONLY to stderr.
var DefaultToStdErr = false

// DefaultAlsoToStdErr establishes the default bool toggling whether logging
// should be written to BOTH file and stderr.
var DefaultAlsoToStdErr = false

// DefaultLogDirName establishes the default directory name for debug (V) logs.
// Log files will be written inside this dir.
// By default, this directory will be created if it does not exist within the context's chain directory, eg.
// <datadir>/<chain>/log/.
var DefaultLogDirName = "log"

// MinSize is a minimum file size qualifying for rotation. This variable can be used
// to avoid rotation of empty or almost emtpy files.
var MinSize uint64

// MaxTotalSize is a maximum size of all log files.
var MaxTotalSize uint64

// Interval is a type for rotation interval specification
type Interval uint8

// These constants identify the interval for log rotation.
const (
	Never Interval = iota
	Hourly
	Daily
	Weekly
	Monthly
)

func ParseInterval(str string) (Interval, error) {
	mapping := map[string]Interval{
		"never":   Never,
		"hourly":  Hourly,
		"daily":   Daily,
		"weekly":  Weekly,
		"monthly": Monthly,
	}

	interval, ok := mapping[strings.ToLower(str)]
	if !ok {
		return Never, fmt.Errorf("invalid interval value '%s'", str)
	}
	return interval, nil
}

// RotationInterval determines how often log rotation should take place
var RotationInterval = Never

// MaxAge defines the maximum age of the oldest log file. All log files older
// than MaxAge will be removed.
var MaxAge time.Duration

// Compress determines whether to compress rotated logs with GZIP or not.
var Compress bool

// severity identifies the sort of log: info, warning etc. It also implements
// the flag.Value interface. The -stderrthreshold flag is of type severity and
// should be modified only through the flag.Value interface. The values match
// the corresponding constants in C++.
// Severity is determined by the method called upon receiver Verbose,
// eg. glog.V(logger.Debug).Warnf("this log's severity is %v", warningLog)
// eg. glog.V(logger.Error).Infof("This log's severity is %v", infoLog)
type severity int32 // sync/atomic int32

// These constants identify the log levels in order of increasing severity.
// A message written to a high-severity log file is also written to each
// lower-severity log file.
const (
	infoLog severity = iota
	warningLog
	errorLog
	fatalLog
	numSeverity = 4
)

const severityChar = "IWEF"

const severityColorReset = "\x1b[0m"                                        // reset both foreground and background
var severityColor = []string{"\x1b[2m", "\x1b[33m", "\x1b[31m", "\x1b[35m"} // info:dim warn:yellow, error:red, fatal:magenta

var severityName = []string{
	infoLog:    "INFO",
	warningLog: "WARNING",
	errorLog:   "ERROR",
	fatalLog:   "FATAL",
}

// these path prefixes are trimmed for display, but not when
// matching vmodule filters.
var trimPrefixes = []string{
	"/github.com/ethereumproject/go-ethereum",
	"/github.com/ethereumproject/ethash",
}

func trimToImportPath(file string) string {
	if root := strings.LastIndex(file, "src/"); root != 0 {
		file = file[root+3:]
	}
	return file
}

// SetV sets the global verbosity level
func SetV(v int) {
	logging.verbosity.set(Level(v))
}

func SetD(v int) {
	display.verbosity.set(Level(v))
}

// SetToStderr sets the global output style
func SetToStderr(toStderr bool) {
	logging.mu.Lock()
	logging.toStderr = toStderr
	logging.mu.Unlock()
}

// SetAlsoToStderr sets global output option
// for logging to both FS and stderr.
func SetAlsoToStderr(to bool) {
	logging.mu.Lock()

	logging.alsoToStderr = to
	logging.mu.Unlock()
}

// GetTraceLocation returns the global TraceLocation flag.
func GetTraceLocation() *TraceLocation {
	return &logging.traceLocation
}

// GetVModule returns the global verbosity pattern flag.
func GetVModule() *moduleSpec {
	return &logging.vmodule
}

// GetVerbosity returns the global verbosity level flag.
func GetVerbosity() *Level {
	return &logging.verbosity
}

func GetDisplayable() *Level {
	return &display.verbosity
}

// get returns the value of the severity.
func (s *severity) get() severity {
	return severity(atomic.LoadInt32((*int32)(s)))
}

// set sets the value of the severity.
func (s *severity) set(val severity) {
	atomic.StoreInt32((*int32)(s), int32(val))
}

// String is part of the flag.Value interface.
func (s *severity) String() string {
	return strconv.FormatInt(int64(*s), 10)
}

// Get is part of the flag.Value interface.
func (s *severity) Get() interface{} {
	return *s
}

// Set is part of the flag.Value interface.
func (s *severity) Set(value string) error {
	var threshold severity
	// Is it a known name?
	if v, ok := severityByName(value); ok {
		threshold = v
	} else {
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		threshold = severity(v)
	}
	logging.stderrThreshold.set(threshold)
	return nil
}

func severityByName(s string) (severity, bool) {
	s = strings.ToUpper(s)
	for i, name := range severityName {
		if name == s {
			return severity(i), true
		}
	}
	return 0, false
}

// OutputStats tracks the number of output lines and bytes written.
type OutputStats struct {
	lines int64
	bytes int64
}

// Lines returns the number of lines written.
func (s *OutputStats) Lines() int64 {
	return atomic.LoadInt64(&s.lines)
}

// Bytes returns the number of bytes written.
func (s *OutputStats) Bytes() int64 {
	return atomic.LoadInt64(&s.bytes)
}

// Stats tracks the number of lines of output and number of bytes
// per severity level. Values must be read with atomic.LoadInt64.
var Stats struct {
	Info, Warning, Error OutputStats
}

var severityStats = [numSeverity]*OutputStats{
	infoLog:    &Stats.Info,
	warningLog: &Stats.Warning,
	errorLog:   &Stats.Error,
}

// Level is exported because it appears in the arguments to V and is
// the type of the v flag, which can be set programmatically.
// It's a distinct type because we want to discriminate it from logType.
// Variables of type level are only changed under logging.mu.
// The -v flag is read only with atomic ops, so the state of the logging
// module is consistent.

// Level is treated as a sync/atomic int32.

// Level specifies a level of verbosity for V logs. *Level implements
// flag.Value; the -v flag is of type Level and should be modified
// only through the flag.Value interface.
type Level int32

// get returns the value of the Level.
func (l *Level) get() Level {
	return Level(atomic.LoadInt32((*int32)(l)))
}

// set sets the value of the Level.
func (l *Level) set(val Level) {
	atomic.StoreInt32((*int32)(l), int32(val))
}

// String is part of the flag.Value interface.
func (l *Level) String() string {
	return strconv.FormatInt(int64(*l), 10)
}

// Get is part of the flag.Value interface.
func (l *Level) Get() interface{} {
	return *l
}

// Set is part of the flag.Value interface.
func (l *Level) Set(value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	logging.mu.Lock()
	defer logging.mu.Unlock()
	logging.setVState(Level(v), logging.vmodule.filter, false)
	return nil
}

// moduleSpec represents the setting of the -vmodule flag.
type moduleSpec struct {
	filter []modulePat
}

// modulePat contains a filter for the -vmodule flag.
// It holds a verbosity level and a file pattern to match.
type modulePat struct {
	pattern *regexp.Regexp
	level   Level
}

func (m *moduleSpec) String() string {
	// Lock because the type is not atomic. TODO: clean this up.
	logging.mu.Lock()
	defer logging.mu.Unlock()
	var b bytes.Buffer
	for i, f := range m.filter {
		if i > 0 {
			b.WriteRune(',')
		}
		fmt.Fprintf(&b, "%s=%d", f.pattern, f.level)
	}
	return b.String()
}

// Get is part of the (Go 1.2)  flag.Getter interface. It always returns nil for this flag type since the
// struct is not exported.
func (m *moduleSpec) Get() interface{} {
	return nil
}

var errVmoduleSyntax = errors.New("syntax error: expect comma-separated list of filename=N")

// Syntax: -vmodule=recordio=2,file=1,gfs*=3
func (m *moduleSpec) Set(value string) error {
	var filter []modulePat
	for _, pat := range strings.Split(value, ",") {
		if len(pat) == 0 {
			// Empty strings such as from a trailing comma can be ignored.
			continue
		}
		patLev := strings.Split(pat, "=")
		if len(patLev) != 2 || len(patLev[0]) == 0 || len(patLev[1]) == 0 {
			return errVmoduleSyntax
		}
		pattern := patLev[0]
		v, err := strconv.Atoi(patLev[1])
		if err != nil {
			return errors.New("syntax error: expect comma-separated list of filename=N")
		}
		if v < 0 {
			return errors.New("negative value for vmodule level")
		}
		if v == 0 {
			continue // Ignore. It's harmless but no point in paying the overhead.
		}
		// TODO: check syntax of filter?
		re, _ := compileModulePattern(pattern)
		filter = append(filter, modulePat{re, Level(v)})
	}
	logging.mu.Lock()
	defer logging.mu.Unlock()
	logging.setVState(logging.verbosity, filter, true)
	return nil
}

// compiles a vmodule pattern to a regular expression.
func compileModulePattern(pat string) (*regexp.Regexp, error) {
	re := ".*"
	for _, comp := range strings.Split(pat, "/") {
		if comp == "*" {
			re += "(/.*)?"
		} else if comp != "" {
			// TODO: maybe return error if comp contains *
			re += "/" + regexp.QuoteMeta(comp)
		}
	}
	if !strings.HasSuffix(pat, ".go") {
		re += "/[^/]+\\.go"
	}
	return regexp.Compile(re + "$")
}

// traceLocation represents the setting of the -log_backtrace_at flag.
type TraceLocation struct {
	file string
	line int
}

// isSet reports whether the trace location has been specified.
// logging.mu is held.
func (t *TraceLocation) isSet() bool {
	return t.line > 0
}

// match reports whether the specified file and line matches the trace location.
// The argument file name is the full path, not the basename specified in the flag.
// logging.mu is held.
func (t *TraceLocation) match(file string, line int) bool {
	if t.line != line {
		return false
	}
	if i := strings.LastIndex(file, "/"); i >= 0 {
		file = file[i+1:]
	}
	return t.file == file
}

func (t *TraceLocation) String() string {
	// Lock because the type is not atomic. TODO: clean this up.
	logging.mu.Lock()
	defer logging.mu.Unlock()
	return fmt.Sprintf("%s:%d", t.file, t.line)
}

// Get is part of the (Go 1.2) flag.Getter interface. It always returns nil for this flag type since the
// struct is not exported
func (t *TraceLocation) Get() interface{} {
	return nil
}

var errTraceSyntax = errors.New("syntax error: expect 'file.go:234'")

// Syntax: -log_backtrace_at=gopherflakes.go:234
// Note that unlike vmodule the file extension is included here.
func (t *TraceLocation) Set(value string) error {
	if value == "" {
		// Unset.
		logging.mu.Lock()
		t.line = 0
		t.file = ""
		logging.mu.Unlock()
		return nil
	}

	fields := strings.Split(value, ":")
	if len(fields) != 2 {
		return errTraceSyntax
	}
	file, line := fields[0], fields[1]
	if !strings.Contains(file, ".") {
		return errTraceSyntax
	}
	v, err := strconv.Atoi(line)
	if err != nil {
		return errTraceSyntax
	}
	if v <= 0 {
		return errors.New("negative or zero value for level")
	}
	logging.mu.Lock()
	defer logging.mu.Unlock()
	t.line = v
	t.file = file
	return nil
}

// flushSyncWriter is the interface satisfied by logging destinations.
type flushSyncWriter interface {
	Flush() error
	Sync() error
	io.Writer
}

type logTName string

const (
	fileLog    logTName = "file"
	displayLog logTName = "display"
)

// loggingT collects all the global state of the logging setup.
type loggingT struct {
	logTName
	// Boolean flags. Not handled atomically because the flag.Value interface
	// does not let us avoid the =true, and that shorthand is necessary for
	// compatibility. TODO: does this matter enough to fix? Seems unlikely.
	toStderr     bool // The -logtostderr flag.
	alsoToStderr bool // The -alsologtostderr flag.

	// Level flag. Handled atomically.
	stderrThreshold severity // The -stderrthreshold flag.

	// freeList is a list of byte buffers, maintained under freeListMu.
	freeList *buffer
	// freeListMu maintains the free list. It is separate from the main mutex
	// so buffers can be grabbed and printed to without holding the main lock,
	// for better parallelization.
	freeListMu sync.Mutex

	// mu protects the remaining elements of this structure and is
	// used to synchronize logging.
	mu sync.Mutex
	// file holds writer for each of the log types.
	file [numSeverity]flushSyncWriter
	// pcs is used in V to avoid an allocation when computing the caller's PC.
	pcs [1]uintptr
	// vmap is a cache of the V Level for each V() call site, identified by PC.
	// It is wiped whenever the vmodule flag changes state.
	vmap map[uintptr]Level
	// filterLength stores the length of the vmodule filter chain. If greater
	// than zero, it means vmodule is enabled. It may be read safely
	// using sync.LoadInt32, but is only modified under mu.
	filterLength int32
	// traceLocation is the state of the -log_backtrace_at flag.
	traceLocation TraceLocation
	// These flags are modified only under lock, although verbosity may be fetched
	// safely using atomic.LoadInt32.
	vmodule   moduleSpec // The state of the -vmodule flag.
	verbosity Level      // V logging level, the value of the -v flag/

	// severityTraceThreshold determines the minimum severity at which
	// file traces will be logged in the header. See severity const iota above.
	// Only severities at or above this number will be logged with a trace,
	// eg. at severityTraceThreshold = 2, then only severities errorLog and fatalLog
	// will log with traces.
	severityTraceThreshold severity

	// verbosityTraceThreshold determines the minimum verbosity at which
	// file traces will be logged in the header.
	// Only levels at or above this number will be logged with a trace,
	// eg. at verbosityTraceThreshold = 5, then only verbosities Debug, Detail, and Ridiculousness
	// will log with traces.
	verbosityTraceThreshold Level
}

// buffer holds a byte Buffer for reuse. The zero value is ready for use.
type buffer struct {
	bytes.Buffer
	tmp  [64]byte // temporary byte array for creating headers.
	next *buffer
}

var logging loggingT
var display loggingT

func init() {
	//flag.BoolVar(&logging.toStderr, "logtostderr", false, "log to standard error instead of files")
	//flag.BoolVar(&logging.alsoToStderr, "alsologtostderr", false, "log to standard error as well as files")
	//flag.Var(&logging.verbosity, "v", "log level for V logs")
	//flag.Var(&logging.stderrThreshold, "stderrthreshold", "logs at or above this threshold go to stderr")
	//flag.Var(&logging.vmodule, "vmodule", "comma-separated list of pattern=N settings for file-filtered logging")
	//flag.Var(&logging.traceLocation, "log_backtrace_at", "when logging hits line file:N, emit a stack trace")

	logging.logTName = fileLog
	// Default stderrThreshold is ERROR.
	// This makes V(logger.Error) logs print ALSO to stderr.
	logging.stderrThreshold = errorLog

	// Establish defaults for trace thresholds.
	logging.verbosityTraceThreshold.set(0)
	logging.severityTraceThreshold.set(2)

	// Default for verbosity.
	logging.setVState(Level(DefaultVerbosity), nil, false)
	go logging.flushDaemon()

	display.logTName = displayLog
	// Renders anything at or below (Warn...) level Info to stderr, which
	// is set by default anyway.
	display.stderrThreshold = infoLog

	// toStderr makes it ONLY print to stderr, not to file
	display.toStderr = true

	// Should never reach... unless we get real fancy with D(levels)
	display.verbosityTraceThreshold.set(5)
	// Only includes traces for severity=fatal logs for display.
	// This should never be reached; fatal logs should ALWAYS be logged to file,
	// and they will also be written to stderr (anything Error and above is).
	// Keep in mind severities are "upside-down" from verbosities; so here 3=error, 4=fatal, and 0=info
	// and that here severity>=3 will meet the threshold.
	display.severityTraceThreshold.set(2)
	// Set display verbosity default Info. So it will render
	// all Fatal, Error, Warn, and Info log levels.
	// Please don't use Fatal for display; again, Fatal logs should only go through file logging
	// (they will be printed to stderr anyway).
	display.setVState(Level(DefaultDisplay), nil, false)
	go display.flushDaemon()
}

// Flush flushes all pending log I/O.
func Flush() {
	logging.lockAndFlushAll()
	display.lockAndFlushAll()
}

// traceThreshold determines the arbitrary level for log lines to be printed
// with caller trace information in the header.
func (l *loggingT) traceThreshold(s severity) bool {
	return s >= l.severityTraceThreshold || l.verbosity >= l.verbosityTraceThreshold
}

// GetVTraceThreshold gets the current verbosity trace threshold for logging.
func GetVTraceThreshold() *Level {
	return &logging.verbosityTraceThreshold
}

// SetVTraceThreshold sets the current verbosity trace threshold for logging.
func SetVTraceThreshold(v int) {
	logging.mu.Lock()
	defer logging.mu.Unlock()

	l := logging.verbosity.get()
	logging.verbosity.set(0)
	logging.verbosityTraceThreshold.set(Level(v))
	logging.verbosity.set(l)
}

// setVState sets a consistent state for V logging.
// l.mu is held.
func (l *loggingT) setVState(verbosity Level, filter []modulePat, setFilter bool) {
	// Turn verbosity off so V will not fire while we are in transition.
	l.verbosity.set(0)
	// Ditto for filter length.
	atomic.StoreInt32(&l.filterLength, 0)

	// Set the new filters and wipe the pc->Level map if the filter has changed.
	if setFilter {
		l.vmodule.filter = filter
		l.vmap = make(map[uintptr]Level)
	}

	// Things are consistent now, so enable filtering and verbosity.
	// They are enabled in order opposite to that in V.
	atomic.StoreInt32(&l.filterLength, int32(len(filter)))
	l.verbosity.set(verbosity)
}

// getBuffer returns a new, ready-to-use buffer.
func (l *loggingT) getBuffer() *buffer {
	l.freeListMu.Lock()
	b := l.freeList
	if b != nil {
		l.freeList = b.next
	}
	l.freeListMu.Unlock()
	if b == nil {
		b = new(buffer)
	} else {
		b.next = nil
		b.Reset()
	}
	return b
}

// putBuffer returns a buffer to the free list.
func (l *loggingT) putBuffer(b *buffer) {
	if b.Len() >= 256 {
		// Let big buffers die a natural death.
		return
	}
	l.freeListMu.Lock()
	b.next = l.freeList
	l.freeList = b
	l.freeListMu.Unlock()
}

var timeNow = time.Now // Stubbed out for testing.

/*
header formats a log header as defined by the C++ implementation.
It returns a buffer containing the formatted header and the user's file and line number.
The depth specifies how many stack frames above lives the source line to be identified in the log message.

Log lines have this form:
	Lmmdd hh:mm:ss.uuuuuu threadid file:line] msg...
where the fields are defined as follows:
	L                A single character, representing the log level (eg 'I' for INFO)
	mm               The month (zero padded; ie May is '05')
	dd               The day (zero padded)
	hh:mm:ss.uuuuuu  Time in hours, minutes and fractional seconds
	threadid         The space-padded thread ID as returned by GetTID()
	file             The file name
	line             The line number
	msg              The user-supplied message
*/
func (l *loggingT) header(s severity, depth int) (*buffer, string, int) {
	_, file, line, ok := runtime.Caller(3 + depth)
	if !ok {
		file = "???"
		line = 1
	} else {
		file = trimToImportPath(file)
		for _, p := range trimPrefixes {
			if strings.HasPrefix(file, p) {
				file = file[len(p):]
				break
			}
		}
		file = file[1:] // drop '/'
	}
	return l.formatHeader(s, file, line), file, line
}

// formatHeader formats a log header using the provided file name and line number.
func (l *loggingT) formatHeader(s severity, file string, line int) *buffer {
	now := timeNow()
	if line < 0 {
		line = 0 // not a real line number, but acceptable to someDigits
	}
	if s > fatalLog {
		s = infoLog // for safety.
	}
	buf := l.getBuffer()

	// Avoid Fprintf, for speed. The format is so simple that we can do it quickly by hand.
	// It's worth about 3X. Fprintf is hard.
	year, month, day := now.Date()
	hour, minute, second := now.Clock()
	// Lmmdd hh:mm:ss.uuuuuu threadid file:line]

	//buf.nDigits(8, 0, severityColor[s],'')

	// If to-file (debuggable) logs.
	if l.logTName == fileLog {
		buf.tmp[0] = severityChar[s]
		buf.Write(buf.tmp[:1])
		buf.twoDigits(0, int(month))
		buf.twoDigits(2, day)
		buf.tmp[4] = ' '
		buf.twoDigits(5, hour)
		buf.tmp[7] = ':'
		buf.twoDigits(8, minute)
		buf.tmp[10] = ':'
		buf.twoDigits(11, second)
		// Only keep nanoseconds for file logs
		buf.tmp[13] = '.'
		buf.nDigits(6, 14, now.Nanosecond()/1000, '0')
		buf.Write(buf.tmp[:20])
		buf.WriteString(" ")

		if l.traceThreshold(s) {
			buf.WriteString(file)
			buf.tmp[0] = ':'
			n := buf.someDigits(1, line)
			buf.tmp[n+1] = ']'
			buf.tmp[n+2] = ' '
			buf.Write(buf.tmp[:n+3])
		}
	} else {
		// Write dim.
		buf.WriteString(severityColor[infoLog])

		buf.nDigits(4, 0, year, '_')
		buf.nDigits(4, 0, year, '_')
		buf.tmp[4] = '-'
		buf.twoDigits(5, int(month))
		buf.tmp[7] = '-'
		buf.twoDigits(8, day)
		buf.tmp[10] = ' '
		buf.twoDigits(11, hour)
		buf.tmp[13] = ':'
		buf.twoDigits(14, minute)
		buf.tmp[16] = ':'
		buf.twoDigits(17, second)
		buf.Write(buf.tmp[:19])

		buf.WriteString(severityColorReset + " ")
		if l.traceThreshold(s) {
			buf.WriteString(severityColor[s])
			buf.Write([]byte{'['})
			buf.WriteString(severityName[s])
			buf.Write([]byte{']'})
			buf.WriteString(severityColorReset)
			buf.Write([]byte{' '})
		}
	}

	return buf
}

// Some custom tiny helper functions to print the log header efficiently.

const digits = "0123456789"

// twoDigits formats a zero-prefixed two-digit integer at buf.tmp[i].
func (buf *buffer) twoDigits(i, d int) {
	buf.tmp[i+1] = digits[d%10]
	d /= 10
	buf.tmp[i] = digits[d%10]
}

// nDigits formats an n-digit integer at buf.tmp[i],
// padding with pad on the left.
// It assumes d >= 0.
func (buf *buffer) nDigits(n, i, d int, pad byte) {
	j := n - 1
	for ; j >= 0 && d > 0; j-- {
		buf.tmp[i+j] = digits[d%10]
		d /= 10
	}
	for ; j >= 0; j-- {
		buf.tmp[i+j] = pad
	}
}

// someDigits formats a zero-prefixed variable-width integer at buf.tmp[i].
func (buf *buffer) someDigits(i, d int) int {
	// Print into the top, then copy down. We know there's space for at least
	// a 10-digit number.
	j := len(buf.tmp)
	for {
		j--
		buf.tmp[j] = digits[d%10]
		d /= 10
		if d == 0 {
			break
		}
	}
	return copy(buf.tmp[i:], buf.tmp[j:])
}

func (l *loggingT) println(s severity, args ...interface{}) {
	buf, file, line := l.header(s, 0)
	fmt.Fprintln(buf, args...)
	l.output(s, buf, file, line, false)
}

func (l *loggingT) print(s severity, args ...interface{}) {
	l.printDepth(s, 1, args...)
}

func (l *loggingT) printDepth(s severity, depth int, args ...interface{}) {
	buf, file, line := l.header(s, depth)
	fmt.Fprint(buf, args...)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	l.output(s, buf, file, line, false)
}

func (l *loggingT) printfmt(s severity, format string, args ...interface{}) {
	buf, file, line := l.header(s, 0)
	fmt.Fprintf(buf, format, args...)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	l.output(s, buf, file, line, false)
}

// printWithFileLine behaves like print but uses the provided file and line number.  If
// alsoLogToStderr is true, the log message always appears on standard error; it
// will also appear in the log file unless --logtostderr is set.
func (l *loggingT) printWithFileLine(s severity, file string, line int, alsoToStderr bool, args ...interface{}) {
	buf := l.formatHeader(s, file, line)
	fmt.Fprint(buf, args...)
	if buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
	l.output(s, buf, file, line, alsoToStderr)
}

// output writes the data to the log files and releases the buffer.
func (l *loggingT) output(s severity, buf *buffer, file string, line int, alsoToStderr bool) {
	l.mu.Lock()
	if l.traceLocation.isSet() {
		if l.traceLocation.match(file, line) {
			buf.Write(stacks(false))
		}
	}
	data := buf.Bytes()
	if l.toStderr {
		os.Stderr.Write(data)
	} else {
		if alsoToStderr || l.alsoToStderr || s >= l.stderrThreshold.get() {
			os.Stderr.Write(data)
		}
		if l.file[s] == nil {
			if err := l.createFiles(s); err != nil {
				os.Stderr.Write(data) // Make sure the message appears somewhere.
				l.exit(err)
			}
		}
		switch s {
		case fatalLog:
			l.file[fatalLog].Write(data)
			fallthrough
		case errorLog:
			l.file[errorLog].Write(data)
			fallthrough
		case warningLog:
			l.file[warningLog].Write(data)
			fallthrough
		case infoLog:
			l.file[infoLog].Write(data)
		}
	}
	if s == fatalLog {
		// If we got here via Exit rather than Fatal, print no stacks.
		if atomic.LoadUint32(&fatalNoStacks) > 0 {
			l.mu.Unlock()
			timeoutFlush(10 * time.Second)
			os.Exit(1)
		}
		// Dump all goroutine stacks before exiting.
		// First, make sure we see the trace for the current goroutine on standard error.
		// If -logtostderr has been specified, the loop below will do that anyway
		// as the first stack in the full dump.
		if !l.toStderr {
			os.Stderr.Write(stacks(false))
		}
		// Write the stack trace for all goroutines to the files.
		trace := stacks(true)
		logExitFunc = func(error) {} // If we get a write error, we'll still exit below.
		for log := fatalLog; log >= infoLog; log-- {
			if f := l.file[log]; f != nil { // Can be nil if -logtostderr is set.
				f.Write(trace)
			}
		}
		l.mu.Unlock()
		timeoutFlush(10 * time.Second)
		os.Exit(255) // C++ uses -1, which is silly because it's anded with 255 anyway.
	}
	l.putBuffer(buf)
	l.mu.Unlock()
	if stats := severityStats[s]; stats != nil {
		atomic.AddInt64(&stats.lines, 1)
		atomic.AddInt64(&stats.bytes, int64(len(data)))
	}
}

// timeoutFlush calls Flush and returns when it completes or after timeout
// elapses, whichever happens first.  This is needed because the hooks invoked
// by Flush may deadlock when glog.Fatal is called from a hook that holds
// a lock.
func timeoutFlush(timeout time.Duration) {
	done := make(chan bool, 1)
	go func() {
		Flush() // calls logging.lockAndFlushAll()
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		fmt.Fprintln(os.Stderr, "glog: Flush took longer than", timeout)
	}
}

// stacks is a wrapper for runtime.Stack that attempts to recover the data for all goroutines.
func stacks(all bool) []byte {
	// We don't know how big the traces are, so grow a few times if they don't fit. Start large, though.
	n := 10000
	if all {
		n = 100000
	}
	var trace []byte
	for i := 0; i < 5; i++ {
		trace = make([]byte, n)
		nbytes := runtime.Stack(trace, all)
		if nbytes < len(trace) {
			return trace[:nbytes]
		}
		n *= 2
	}
	return trace
}

// logExitFunc provides a simple mechanism to override the default behavior
// of exiting on error. Used in testing and to guarantee we reach a required exit
// for fatal logs. Instead, exit could be a function rather than a method but that
// would make its use clumsier.
var logExitFunc func(error)

// exit is called if there is trouble creating or writing log files.
// It flushes the logs and exits the program; there's no point in hanging around.
// l.mu is held.
func (l *loggingT) exit(err error) {
	fmt.Fprintf(os.Stderr, "log: exiting because of error: %s\n", err)
	// If logExitFunc is set, we do that instead of exiting.
	if logExitFunc != nil {
		logExitFunc(err)
		return
	}
	l.flushAll()
	os.Exit(2)
}

// syncBuffer joins a bufio.Writer to its underlying file, providing access to the
// file's Sync method and providing a wrapper for the Write method that provides log
// file rotation. There are conflicting methods, so the file cannot be embedded.
// l.mu is held for all its methods.
type syncBuffer struct {
	logger *loggingT
	*bufio.Writer
	file   *os.File
	time   time.Time
	sev    severity
	nbytes uint64 // The number of bytes written to this file
}

func (sb *syncBuffer) Sync() error {
	return sb.file.Sync()
}

func (sb *syncBuffer) Write(p []byte) (n int, err error) {
	now := time.Now()
	if sb.shouldRotate(len(p), now) {
		if err := sb.rotateCurrent(now); err != nil {
			sb.logger.exit(err)
		}
		go sb.rotateOld(now)
	}
	n, err = sb.Writer.Write(p)
	sb.nbytes += uint64(n)
	if err != nil {
		sb.logger.exit(err)
	}
	return
}

// shouldRotate checks if we need to rotate the current log file
func (sb *syncBuffer) shouldRotate(len int, now time.Time) bool {
	newLen := sb.nbytes + uint64(len)
	if newLen <= MinSize {
		return false
	} else if MaxSize > 0 && newLen >= MaxSize {
		return true
	}

	switch RotationInterval {
	case Never:
		return false
	case Hourly:
		return sb.time.Hour() != now.Hour()
	case Daily:
		return sb.time.Day() != now.Day()
	case Weekly:
		yearLog, weekLog := sb.time.ISOWeek()
		yearNow, weekNow := now.ISOWeek()
		return !(yearLog == yearNow && weekLog == weekNow)
	case Monthly:
		return sb.time.Month() != now.Month()
	}
	return false
}

// rotateCurrent closes the syncBuffer's file and starts a new one.
func (sb *syncBuffer) rotateCurrent(now time.Time) error {
	if sb.file != nil {
		sb.Flush()
		sb.file.Close()
	}
	var err error
	sb.file, _, err = create(severityName[sb.sev], now)
	sb.nbytes = 0
	sb.time = time.Now()
	if err != nil {
		return err
	}

	sb.Writer = bufio.NewWriterSize(sb.file, bufferSize)

	// Write header.
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Log file created at: %s\n", now.Format("2006/01/02 15:04:05"))
	fmt.Fprintf(&buf, "Running on machine: %s\n", host)
	fmt.Fprintf(&buf, "Binary: Built with %s %s for %s/%s\n", runtime.Compiler, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&buf, "Log line format: [IWEF]mmdd hh:mm:ss.uuuuuu threadid file:line] msg\n")
	n, err := sb.file.Write(buf.Bytes())
	sb.nbytes += uint64(n)
	return err
}

// converts plain log file to gzipped log file. New file is created
func gzipFile(name string) error {
	gzipped, err := os.Create(name + ".gz")
	defer gzipped.Close()
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(gzipped)
	gzipWriter := gzip.NewWriter(writer)

	plain, err := os.Open(name)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(plain)

	// copy from plain text file to gzipped output
	_, err = io.Copy(gzipWriter, reader)
	_ = plain.Close()
	if err != nil {
		return err
	}
	if err = gzipWriter.Close(); err != nil {
		return err
	}
	if err = writer.Flush(); err != nil {
		return err
	}
	if err = gzipped.Sync(); err != nil {
		return err
	}
	if err = gzipped.Close(); err != nil {
		return err
	}

	return os.Remove(name)
}

var rotationTime int64

func (sb *syncBuffer) rotateOld(now time.Time) {
	nanos := now.UnixNano()
	if atomic.CompareAndSwapInt64(&rotationTime, 0, nanos) {
		logs, err := getLogFiles()
		if err != nil {
			Fatal(err)
		}

		logs = sb.excludeActive(logs)

		logs, err = removeOutdated(logs, now)
		if err != nil {
			Fatal(err)
		}

		logs, err = compressOrphans(logs)
		if err != nil {
			Fatal(err)
		}

		if MaxTotalSize > MaxSize {
			totalSize := getTotalSize(logs)
			for i := 0; i < len(logs) && totalSize > MaxTotalSize-MaxSize; i++ {
				err := os.Remove(filepath.Join(logs[i].dir, logs[i].name))
				if err != nil {
					Fatal(err)
				}
				totalSize -= logs[i].size
			}
		}

		if current := atomic.SwapInt64(&rotationTime, 0); current > nanos {
			go sb.rotateOld(time.Unix(0, current))
		}
	} else {
		atomic.StoreInt64(&rotationTime, nanos)
	}
}

type logFile struct {
	dir       string
	name      string
	size      uint64
	timestamp string
}

// getLogFiles returns log files, ordered from oldest to newest
func getLogFiles() (logFiles []logFile, err error) {
	prefix := fmt.Sprintf("%s.%s.%s.log.", program, host, userName)
	for _, logDir := range logDirs {
		files, err := ioutil.ReadDir(logDir)
		if err == nil {
			files = filterLogFiles(files, prefix)
			for _, file := range files {
				logFiles = append(logFiles, logFile{
					dir:       logDir,
					name:      file.Name(),
					size:      uint64(file.Size()),
					timestamp: extractTimestamp(file.Name(), prefix),
				})
			}
			sort.Slice(logFiles, func(i, j int) bool {
				return logFiles[i].timestamp < logFiles[j].timestamp
			})
			return logFiles, nil
		}
	}
	return nil, errors.New("log: no log dirs")
}

func (sb *syncBuffer) excludeActive(logs []logFile) []logFile {
	filtered := logs[:0]
	current := sb.getCurrentLogs()
	for _, log := range logs {
		active := false
		fullName := filepath.Join(log.dir, log.name)
		for _, latest := range current {
			if fullName == latest {
				active = true
			}
		}
		if !active {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// getCurrentLogs returns list of log files currently in use by the syncBuffer
func (sb *syncBuffer) getCurrentLogs() (logs []string) {
	if sb.logger == nil {
		return nil
	}
	for _, buffer := range sb.logger.file {
		if buffer != nil && buffer.(*syncBuffer).file != nil {
			path, err := filepath.Abs(buffer.(*syncBuffer).file.Name())
			if err == nil {
				logs = append(logs, path)
			}
		}
	}
	return logs
}

func extractTimestamp(logFile, prefix string) string {
	if len(logFile) <= len(prefix) {
		return ""
	}
	splits := strings.SplitN(logFile[len(prefix):], ".", 3)
	if len(splits) == 3 {
		return splits[1]
	} else {
		return ""
	}
}

func filterLogFiles(files []os.FileInfo, prefix string) []os.FileInfo {
	filtered := files[:0]
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), prefix) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func removeOutdated(logs []logFile, now time.Time) ([]logFile, error) {
	if MaxAge == 0 {
		return logs, nil
	}
	t := now.Add(-1 * MaxAge)
	timestamp := fmt.Sprintf("%04d%02d%02d-%02d%02d%02d",
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
	)

	remaining := logs[:0]
	for _, log := range logs {
		if log.timestamp <= timestamp {
			if err := os.Remove(filepath.Join(log.dir, log.name)); err != nil {
				return nil, err
			}
		} else {
			remaining = append(remaining, log)
		}
	}
	return remaining, nil
}

// compress all uncompressed log files, except the currently used log file
func compressOrphans(logs []logFile) ([]logFile, error) {
	for i, log := range logs {
		fullName := filepath.Join(log.dir, log.name)
		if !strings.HasSuffix(log.name, ".gz") {
			if err := gzipFile(fullName); err != nil {
				return nil, err
			}
			logs[i].name += ".gz"
		}
	}
	return logs, nil
}

func getTotalSize(logs []logFile) (size uint64) {
	for _, log := range logs {
		size += log.size
	}
	return
}

// bufferSize sizes the buffer associated with each log file. It's large
// so that log records can accumulate without the logging thread blocking
// on disk I/O. The flushDaemon will block instead.
const bufferSize = 256 * 1024

// createFiles creates all the log files for severity from sev down to infoLog.
// l.mu is held.
func (l *loggingT) createFiles(sev severity) error {
	now := time.Now()
	// Files are created in decreasing severity order, so as soon as we find one
	// has already been created, we can stop.
	for s := sev; s >= infoLog && l.file[s] == nil; s-- {
		sb := &syncBuffer{
			logger: l,
			sev:    s,
		}
		if err := sb.rotateCurrent(now); err != nil {
			return err
		}
		l.file[s] = sb
	}
	return nil
}

const flushInterval = 5 * time.Second

// flushDaemon periodically flushes the log file buffers.
func (l *loggingT) flushDaemon() {
	for range time.NewTicker(flushInterval).C {
		l.lockAndFlushAll()
	}
}

// lockAndFlushAll is like flushAll but locks l.mu first.
func (l *loggingT) lockAndFlushAll() {
	l.mu.Lock()
	l.flushAll()
	l.mu.Unlock()
}

// flushAll flushes all the logs and attempts to "sync" their data to disk.
// l.mu is held.
func (l *loggingT) flushAll() {
	// Flush from fatal down, in case there's trouble flushing.
	for s := fatalLog; s >= infoLog; s-- {
		file := l.file[s]
		if file != nil {
			// if e := file.Flush(); e != nil {
			// 	stdLog.Fatalln(e)
			// }
			// if e := file.Sync(); e != nil {
			// 	stdLog.Fatalln(e)
			// }
			file.Flush() // ignore error
			file.Sync()  // ignore error
		}
	}
}

// CopyStandardLogTo arranges for messages written to the Go "log" package's
// default logs to also appear in the Google logs for the named and lower
// severities.  Subsequent changes to the standard log's default output location
// or format may break this behavior.
//
// Valid names are "INFO", "WARNING", "ERROR", and "FATAL".  If the name is not
// recognized, CopyStandardLogTo panics.
func CopyStandardLogTo(name string) {
	sev, ok := severityByName(name)
	if !ok {
		panic(fmt.Sprintf("log.CopyStandardLogTo(%q): unrecognized severity name", name))
	}
	// Set a log format that captures the user's file and line:
	//   d.go:23: message
	stdLog.SetFlags(stdLog.Lshortfile)
	stdLog.SetOutput(logBridge(sev))
}

// logBridge provides the Write method that enables CopyStandardLogTo to connect
// Go's standard logs to the logs provided by this package.
type logBridge severity

// Write parses the standard logging line and passes its components to the
// logger for severity(lb).
func (lb logBridge) Write(b []byte) (n int, err error) {
	var (
		file = "???"
		line = 1
		text string
	)
	// Split "d.go:23: message" into "d.go", "23", and "message".
	if parts := bytes.SplitN(b, []byte{':'}, 3); len(parts) != 3 || len(parts[0]) < 1 || len(parts[2]) < 1 {
		text = fmt.Sprintf("bad log format: %s", b)
	} else {
		file = string(parts[0])
		text = string(parts[2][1:]) // skip leading space
		line, err = strconv.Atoi(string(parts[1]))
		if err != nil {
			text = fmt.Sprintf("bad line number: %s", b)
			line = 1
		}
	}
	// printWithFileLine with alsoToStderr=true, so standard log messages
	// always appear on standard error.

	logging.printWithFileLine(severity(lb), file, line, true, text)
	return len(b), nil
}

// setV computes and remembers the V level for a given PC
// when vmodule is enabled.
// File pattern matching takes the basename of the file, stripped
// of its .go suffix, and uses filepath.Match, which is a little more
// general than the *? matching used in C++.
// l.mu is held.
func (l *loggingT) setV(pc uintptr) Level {
	fn := runtime.FuncForPC(pc)
	file, _ := fn.FileLine(pc)
	file = trimToImportPath(file)
	for _, filter := range l.vmodule.filter {
		if filter.pattern.MatchString(file) {
			l.vmap[pc] = filter.level
			return filter.level
		}
	}
	l.vmap[pc] = 0
	return 0
}

// Verbose is a boolean type that implements Infof (like Printf) etc.
// See the documentation of V for more information.
type Verbose bool
type Displayable bool

// V reports whether verbosity at the call site is at least the requested level.
// The returned value is a boolean of type Verbose, which implements Info, Infoln
// and Infof. These methods will write to the Info log if called.
// Thus, one may write either
//	if glog.V(2) { glog.Info("log this") }
// or
//	glog.V(2).Info("log this")
// The second form is shorter but the first is cheaper if logging is off because it does
// not evaluate its arguments.
//
// Whether an individual call to V generates a log record depends on the setting of
// the -v and --vmodule flags; both are off by default. If the level in the call to
// V is at least the value of -v, or of -vmodule for the source file containing the
// call, the V call will log.
func V(level Level) Verbose {
	// This function tries hard to be cheap unless there's work to do.
	// The fast path is two atomic loads and compares.

	// Here is a cheap but safe test to see if V logging is enabled globally.
	if logging.verbosity.get() >= level {
		return Verbose(true)
	}
	// It's off globally but it vmodule may still be set.
	// Here is another cheap but safe test to see if vmodule is enabled.
	if atomic.LoadInt32(&logging.filterLength) > 0 {
		// Now we need a proper lock to use the logging structure. The pcs field
		// is shared so we must lock before accessing it. This is fairly expensive,
		// but if V logging is enabled we're slow anyway.
		logging.mu.Lock()
		defer logging.mu.Unlock()
		if runtime.Callers(2, logging.pcs[:]) == 0 {
			return Verbose(false)
		}
		v, ok := logging.vmap[logging.pcs[0]]
		if !ok {
			v = logging.setV(logging.pcs[0])
		}
		return Verbose(v >= level)
	}
	return Verbose(false)
}

func D(level Level) Displayable {
	// This function tries hard to be cheap unless there's work to do.
	// The fast path is two atomic loads and compares.

	// Here is a cheap but safe test to see if V logging is enabled globally.
	if display.verbosity.get() >= level {
		return Displayable(true)
	}
	// It's off globally but it vmodule may still be set.
	// Here is another cheap but safe test to see if vmodule is enabled.
	if atomic.LoadInt32(&display.filterLength) > 0 {
		// Now we need a proper lock to use the logging structure. The pcs field
		// is shared so we must lock before accessing it. This is fairly expensive,
		// but if V logging is enabled we're slow anyway.
		display.mu.Lock()
		defer display.mu.Unlock()
		if runtime.Callers(2, display.pcs[:]) == 0 {
			return Displayable(false)
		}
		v, ok := display.vmap[logging.pcs[0]]
		if !ok {
			v = display.setV(logging.pcs[0])
		}
		return Displayable(v >= level)
	}
	return Displayable(false)
}

func (d Displayable) Infoln(args ...interface{}) {
	if d {
		display.println(infoLog, args...)
	}
}

func (d Displayable) Infof(format string, args ...interface{}) {
	if d {
		display.printfmt(infoLog, format, args...)
	}
}

func (d Displayable) Warnln(args ...interface{}) {
	if d {
		display.println(warningLog, args...)
	}
}

func (d Displayable) Warnf(format string, args ...interface{}) {
	if d {
		display.printfmt(warningLog, format, args...)
	}
}

func (d Displayable) Errorln(args ...interface{}) {
	if d {
		display.println(errorLog, args...)
	}
}

func (d Displayable) Errorf(format string, args ...interface{}) {
	if d {
		display.printfmt(errorLog, format, args...)
	}
}

// INFO
// Info is equivalent to the global Info function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Info(args ...interface{}) {
	if v {
		logging.print(infoLog, args...)
	}
}

// Infoln is equivalent to the global Infoln function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Infoln(args ...interface{}) {
	if v {
		logging.println(infoLog, args...)
	}
}

// Infof is equivalent to the global Infof function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Infof(format string, args ...interface{}) {
	if v {
		logging.printfmt(infoLog, format, args...)
	}
}

// WARN
// Warn is equivalent to the global Warn function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Warn(args ...interface{}) {
	if v {
		logging.print(warningLog, args...)
	}
}

// Warnln is equivalent to the global Warnln function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Warnln(args ...interface{}) {
	if v {
		logging.println(warningLog, args...)
	}
}

// Warnf is equivalent to the global Warnf function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Warnf(format string, args ...interface{}) {
	if v {
		logging.printfmt(warningLog, format, args...)
	}
}

// ERROR
// Error is equivalent to the global Error function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Error(args ...interface{}) {
	if v {
		logging.print(errorLog, args...)
	}
}

// Errorln is equivalent to the global Errorln function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Errorln(args ...interface{}) {
	if v {
		logging.println(errorLog, args...)
	}
}

// Errorf is equivalent to the global Errorf function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Errorf(format string, args ...interface{}) {
	if v {
		logging.printfmt(errorLog, format, args...)
	}
}

// Separator creates a line, ie ---------------------------------
func Separator(iterable string) string {
	return strings.Repeat(iterable, 110)
}

// Info logs to the INFO log.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Info(args ...interface{}) {
	logging.print(infoLog, args...)
}

// InfoDepth acts as Info but uses depth to determine which call frame to log.
// InfoDepth(0, "msg") is the same as Info("msg").
func InfoDepth(depth int, args ...interface{}) {
	logging.printDepth(infoLog, depth, args...)
}

// Infoln logs to the INFO log.
// Arguments are handled in the manner of fmt.Println; a newline is appended if missing.
func Infoln(args ...interface{}) {
	logging.print(infoLog, args...)
}

// Infof logs to the INFO log.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Infof(format string, args ...interface{}) {
	logging.printfmt(infoLog, format, args...)
}

// Warning logs to the WARNING and INFO logs.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Warning(args ...interface{}) {
	logging.print(warningLog, args...)
}

// WarningDepth acts as Warning but uses depth to determine which call frame to log.
// WarningDepth(0, "msg") is the same as Warning("msg").
func WarningDepth(depth int, args ...interface{}) {
	logging.printDepth(warningLog, depth, args...)
}

// Warningln logs to the WARNING and INFO logs.
// Arguments are handled in the manner of fmt.Println; a newline is appended if missing.
func Warningln(args ...interface{}) {
	logging.println(warningLog, args...)
}

// Warningf logs to the WARNING and INFO logs.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Warningf(format string, args ...interface{}) {
	logging.printfmt(warningLog, format, args...)
}

// Error logs to the ERROR, WARNING, and INFO logs.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Error(args ...interface{}) {
	logging.print(errorLog, args...)
}

// ErrorDepth acts as Error but uses depth to determine which call frame to log.
// ErrorDepth(0, "msg") is the same as Error("msg").
func ErrorDepth(depth int, args ...interface{}) {
	logging.printDepth(errorLog, depth, args...)
}

// Errorln logs to the ERROR, WARNING, and INFO logs.
// Arguments are handled in the manner of fmt.Println; a newline is appended if missing.
func Errorln(args ...interface{}) {
	logging.println(errorLog, args...)
}

// Errorf logs to the ERROR, WARNING, and INFO logs.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Errorf(format string, args ...interface{}) {
	logging.printfmt(errorLog, format, args...)
}

// Fatal logs to the FATAL, ERROR, WARNING, and INFO logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Fatal(args ...interface{}) {
	logging.print(fatalLog, args...)
}

// FatalDepth acts as Fatal but uses depth to determine which call frame to log.
// FatalDepth(0, "msg") is the same as Fatal("msg").
func FatalDepth(depth int, args ...interface{}) {
	logging.printDepth(fatalLog, depth, args...)
}

// Fatalln logs to the FATAL, ERROR, WARNING, and INFO logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Println; a newline is appended if missing.
func Fatalln(args ...interface{}) {
	logging.println(fatalLog, args...)
}

// Fatalf logs to the FATAL, ERROR, WARNING, and INFO logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Fatalf(format string, args ...interface{}) {
	logging.printfmt(fatalLog, format, args...)
}

// fatalNoStacks is non-zero if we are to exit without dumping goroutine stacks.
// It allows Exit and relatives to use the Fatal logs.
var fatalNoStacks uint32

// Exit logs to the FATAL, ERROR, WARNING, and INFO logs, then calls os.Exit(1).
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Exit(args ...interface{}) {
	atomic.StoreUint32(&fatalNoStacks, 1)
	logging.print(fatalLog, args...)
}

// ExitDepth acts as Exit but uses depth to determine which call frame to log.
// ExitDepth(0, "msg") is the same as Exit("msg").
func ExitDepth(depth int, args ...interface{}) {
	atomic.StoreUint32(&fatalNoStacks, 1)
	logging.printDepth(fatalLog, depth, args...)
}

// Exitln logs to the FATAL, ERROR, WARNING, and INFO logs, then calls os.Exit(1).
func Exitln(args ...interface{}) {
	atomic.StoreUint32(&fatalNoStacks, 1)
	logging.println(fatalLog, args...)
}

// Exitf logs to the FATAL, ERROR, WARNING, and INFO logs, then calls os.Exit(1).
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Exitf(format string, args ...interface{}) {
	atomic.StoreUint32(&fatalNoStacks, 1)
	logging.printfmt(fatalLog, format, args...)
}
