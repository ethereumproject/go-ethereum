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

package glog

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	stdLog "log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test that shortHostname works as advertised.
func TestShortHostname(t *testing.T) {
	for hostname, expect := range map[string]string{
		"":                "",
		"host":            "host",
		"host.google.com": "host",
	} {
		if got := shortHostname(hostname); expect != got {
			t.Errorf("shortHostname(%q): expected %q, got %q", hostname, expect, got)
		}
	}
}

// flushBuffer wraps a bytes.Buffer to satisfy flushSyncWriter.
type flushBuffer struct {
	bytes.Buffer
}

func (f *flushBuffer) Flush() error {
	return nil
}

func (f *flushBuffer) Sync() error {
	return nil
}

// swapLogging sets the log writers and returns the old array.
func (l *loggingT) swapLogging(writers [numSeverity]flushSyncWriter) (old [numSeverity]flushSyncWriter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	old = l.file
	for i, w := range writers {
		logging.file[i] = w
	}
	return
}

// swapLogging sets the log writers and returns the old array.
func (l *loggingT) swapDisplay(writers [numSeverity]flushSyncWriter) (old [numSeverity]flushSyncWriter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	old = l.file
	for i, w := range writers {
		display.file[i] = w
	}
	return
}

// newLoggingBuffers sets the log writers to all new byte buffers and returns the old array.
func (l *loggingT) newLoggingBuffers() [numSeverity]flushSyncWriter {
	return l.swapLogging([numSeverity]flushSyncWriter{new(flushBuffer), new(flushBuffer), new(flushBuffer), new(flushBuffer)})
}

// newDisplayBuffers sets the log writers to all new byte buffers and returns the old array.
func (l *loggingT) newDisplayBuffers() [numSeverity]flushSyncWriter {
	return l.swapDisplay([numSeverity]flushSyncWriter{new(flushBuffer), new(flushBuffer), new(flushBuffer), new(flushBuffer)})
}

// loggingContents returns the specified log value as a string.
func loggingContents(s severity) string {
	return logging.file[s].(*flushBuffer).String()
}

// displayContents returns the specified log value as a string.
func displayContents(s severity) string {
	return display.file[s].(*flushBuffer).String()
}

// loggingContains reports whether the string is contained in the log.
func loggingContains(s severity, str string, t *testing.T) bool {
	return strings.Contains(loggingContents(s), str)
}

// displayContains reports whether the string is contained in the log.
func displayContains(s severity, str string, t *testing.T) bool {
	return strings.Contains(displayContents(s), str)
}


// setFlags configures the logging flags how the test expects them.
func setFlags() {
	logging.toStderr = false
	display.toStderr = false
}

// Test that Info works as advertised.
func TestInfo(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	Info("test")
	if !loggingContains(infoLog, "I", t) {
		t.Errorf("Info has wrong character: %q", loggingContents(infoLog))
	}
	if !loggingContains(infoLog, "test", t) {
		t.Error("Info failed")
	}
}

func TestInfoDepth(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())

	f := func() { InfoDepth(1, "depth-test1") }

	// The next three lines must stay together
	_, _, wantLine, _ := runtime.Caller(0)
	InfoDepth(0, "depth-test0")
	f()

	msgs := strings.Split(strings.TrimSuffix(loggingContents(infoLog), "\n"), "\n")
	if len(msgs) != 2 {
		t.Fatalf("Got %d lines, expected 2", len(msgs))
	}

	for i, m := range msgs {
		if !strings.HasPrefix(m, "I") {
			t.Errorf("InfoDepth[%d] has wrong character: %q", i, m)
		}
		w := fmt.Sprintf("depth-test%d", i)
		if !strings.Contains(m, w) {
			t.Errorf("InfoDepth[%d] missing %q: %q", i, w, m)
		}

		// pull out the line number (between : and ])
		msg := m[strings.LastIndex(m, ":")+1:]
		x := strings.Index(msg, "]")
		if x < 0 {
			t.Errorf("InfoDepth[%d]: missing ']': %q", i, m)
			continue
		}
		line, err := strconv.Atoi(msg[:x])
		if err != nil {
			t.Errorf("InfoDepth[%d]: bad line number: %q", i, m)
			continue
		}
		wantLine++
		if wantLine != line {
			t.Errorf("InfoDepth[%d]: got line %d, want %d", i, line, wantLine)
		}
	}
}

func init() {
	CopyStandardLogTo("INFO")
}

// Test that CopyStandardLogTo panics on bad input.
func TestCopyStandardLogToPanic(t *testing.T) {
	defer func() {
		if s, ok := recover().(string); !ok || !strings.Contains(s, "LOG") {
			t.Errorf(`CopyStandardLogTo("LOG") should have panicked: %v`, s)
		}
	}()
	CopyStandardLogTo("LOG")
}

// Test that using the standard log package logs to INFO.
func TestStandardLog(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	stdLog.Print("test")
	if !loggingContains(infoLog, "I", t) {
		t.Errorf("Info has wrong character: %q", loggingContents(infoLog))
	}
	if !loggingContains(infoLog, "test", t) {
		t.Error("Info failed")
	}
}

func lineNumber() int {
	_, _, line, _ := runtime.Caller(1)
	return line
}

// Test that the header has the correct format.
func TestHeader1ErrorLog(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	defer func(previous func() time.Time) { timeNow = previous }(timeNow)
	timeNow = func() time.Time {
		return time.Date(2006, 1, 2, 15, 4, 5, .067890e9, time.Local)
	}
	pid = 1234
	Error("test")
	line := lineNumber() - 1 // the line directly above
	format := "E" + "0102 15:04:05.067890 logger/glog/glog_test.go:%d] test\n"
	n, err := fmt.Sscanf(loggingContents(errorLog), format, &line)
	if n != 1 || err != nil {
		t.Errorf("log format error: %d elements, error %s:\n%s", n, err, loggingContents(errorLog))
	}
	// Scanf treats multiple spaces as equivalent to a single space,
	// so check for correct space-padding also.
	want := fmt.Sprintf(format, line)
	if loggingContents(errorLog) != want {
		t.Errorf("log format error: got:\n\t%q\nwant:\t%q", loggingContents(errorLog), want)
	}
}

// Test that the header has the correct format.
func TestHeader2InfoLog(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	defer func(previous func() time.Time) { timeNow = previous }(timeNow)
	timeNow = func() time.Time {
		return time.Date(2006, 1, 2, 15, 4, 5, .067890e9, time.Local)
	}
	s := logging.verbosityTraceThreshold.get()
	logging.verbosityTraceThreshold.set(5) // Use app flag defaults
	defer logging.verbosityTraceThreshold.set(s)
	pid = 1234
	Info("test")
	format := "I" + "0102 15:04:05.067890 logger/glog/glog_test.go:"+strconv.Itoa(lineNumber()-1)+"] test\n"
	n, err := fmt.Sscanf(loggingContents(infoLog), format)
	if err != nil {
		t.Errorf("log format error: %d elements, error %s:\n%s", n, err, loggingContents(infoLog))
	}
	// Scanf treats multiple spaces as equivalent to a single space,
	// so check for correct space-padding also.
	want := fmt.Sprintf(format)
	if loggingContents(infoLog) != want {
		t.Errorf("log format error: got:\n\t%q\nwant:\n\t%q", loggingContents(infoLog), want)
	}
}

// Test that an Error log goes to Warning and Info.
// Even in the Info log, the source character will be E, so the data should
// all be identical.
func TestError(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	defer display.swapDisplay(display.newDisplayBuffers())
	Error("test")
	if !loggingContains(errorLog, "E", t) {
		t.Errorf("Error has wrong character: %q", loggingContents(errorLog))
	}
	if !loggingContains(errorLog, "test", t) {
		t.Error("Error failed")
	}
	str := loggingContents(errorLog)
	if !loggingContains(warningLog, str, t) {
		t.Error("Warning failed")
	}
	if !loggingContains(infoLog, str, t) {
		t.Error("Info failed")
	}
}

// Test that a Warning log goes to Info.
// Even in the Info log, the source character will be W, so the data should
// all be identical.
func TestWarningLogging(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	Warning("test")
	if !loggingContains(warningLog, "W", t) {
		t.Errorf("Warning has wrong character: %q", loggingContents(warningLog))
	}
	if !loggingContains(warningLog, "test", t) {
		t.Error("Warning failed")
	}
	str := loggingContents(warningLog)
	if !loggingContains(infoLog, str, t) {
		t.Error("Info failed")
	}
}

func TestWarningDisplay(t *testing.T) {
	setFlags()
	defer display.swapDisplay(display.newDisplayBuffers())
	display.verbosity.Set("3")
	defer display.verbosity.Set("0")
	D(2).Warnln("test")
	if !displayContains(warningLog, "test", t) {
		t.Error("Warning failed")
	}
	str := displayContents(warningLog)
	if !displayContains(infoLog, str, t) {
		t.Error("Info failed")
	}
}

func TestErrorDisplay(t *testing.T) {
	setFlags()
	defer display.swapDisplay(display.newDisplayBuffers())
	display.verbosity.Set("3")
	defer display.verbosity.Set("0")
	D(2).Errorln("test")
	if !displayContains(errorLog, "ERROR", t) {
		t.Errorf("Error has wrong character: %q", displayContents(errorLog))
	}
	if !displayContains(warningLog, "test", t) {
		t.Error("Warning failed")
	}
	str := displayContents(warningLog)
	if !displayContains(infoLog, str, t) {
		t.Error("Info failed")
	}
}

// Test that a V log goes to Info.
func TestV(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	logging.verbosity.Set("2")
	defer logging.verbosity.Set("0")
	V(2).Info("test")
	if !loggingContains(infoLog, "I", t) {
		t.Errorf("Info has wrong character: %q", loggingContents(infoLog))
	}
	if !loggingContains(infoLog, "test", t) {
		t.Error("Info failed")
	}
}

// Test that a V log goes to Info.
func TestD(t *testing.T) {
	setFlags()
	defer display.swapDisplay(display.newDisplayBuffers())
	display.verbosity.Set("2")
	defer display.verbosity.Set("0")
	D(2).Infoln("test")
	if !displayContains(infoLog, ":", t) {
		t.Errorf("Info has wrong character: %q", displayContents(infoLog))
	}
	if displayContains(infoLog, "I", t) {
		t.Errorf("Info has wrong character: %q", displayContents(infoLog))
	}
	if !displayContains(infoLog, "test", t) {
		t.Error("Info failed")
	}
}

// Test that a vmodule enables a log in this file.
func TestVmoduleOn(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	logging.vmodule.Set("glog_test.go=2")
	defer logging.vmodule.Set("")
	if !V(1) {
		t.Error("V not enabled for 1")
	}
	if !V(2) {
		t.Error("V not enabled for 2")
	}
	if V(3) {
		t.Error("V enabled for 3")
	}
	V(2).Info("test")
	if !loggingContains(infoLog, "I", t) {
		t.Errorf("Info has wrong character: %q", loggingContents(infoLog))
	}
	if !loggingContains(infoLog, "test", t) {
		t.Error("Info failed")
	}
}

// Test that a vmodule of another file does not enable a log in this file.
func TestVmoduleOff(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	logging.vmodule.Set("notthisfile=2")
	defer logging.vmodule.Set("")
	for i := 1; i <= 3; i++ {
		if V(Level(i)) {
			t.Errorf("V enabled for %d", i)
		}
	}
	V(2).Info("test")
	if loggingContents(infoLog) != "" {
		t.Error("V logged incorrectly")
	}
}

var patternTests = []struct{ input, want string }{
	{"foo/bar/x.go", ".*/foo/bar/x\\.go$"},
	{"foo/*/x.go", ".*/foo(/.*)?/x\\.go$"},
	{"foo/*", ".*/foo(/.*)?/[^/]+\\.go$"},
}

func TestCompileModulePattern(t *testing.T) {
	for _, test := range patternTests {
		re, err := compileModulePattern(test.input)
		if err != nil {
			t.Fatalf("%s: %v", test.input, err)
		}
		if re.String() != test.want {
			t.Errorf("mismatch for %q: got %q, want %q", test.input, re.String(), test.want)
		}
	}
}

// vGlobs are patterns that match/don't match this file at V=2.
var vGlobs = map[string]bool{
	// Easy to test the numeric match here.
	"glog_test.go=1": false, // If -vmodule sets V to 1, V(2) will fail.
	"glog_test.go=2": true,
	"glog_test.go=3": true, // If -vmodule sets V to 1, V(3) will succeed.

	// Import path prefix matching
	"logger/glog=1": false,
	"logger/glog=2": true,
	"logger/glog=3": true,

	// Import path glob matching
	"logger/*=1": false,
	"logger/*=2": true,
	"logger/*=3": true,

	// These all use 2 and check the patterns.
	"*=2": true,
}

// Test that vmodule globbing works as advertised.
func testVmoduleGlob(pat string, match bool, t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	defer logging.vmodule.Set("")
	logging.vmodule.Set(pat)
	if V(2) != Verbose(match) {
		t.Errorf("incorrect match for %q: got %t expected %t", pat, V(2), match)
	}
}

// Test that a vmodule globbing works as advertised.
func TestVmoduleGlob(t *testing.T) {
	for glob, match := range vGlobs {
		testVmoduleGlob(glob, match, t)
	}
}

func TestRollover(t *testing.T) {
	setFlags()
	var err error
	defer func(previous func(error)) { logExitFunc = previous }(logExitFunc)
	logExitFunc = func(e error) {
		err = e
	}
	defer func(previous uint64) { MaxSize = previous }(MaxSize)
	MaxSize = 512

	Info("x") // Be sure we have a file.
	info, ok := logging.file[infoLog].(*syncBuffer)
	if !ok {
		t.Fatal("info wasn't created")
	}
	if err != nil {
		t.Fatalf("info has initial error: %v", err)
	}
	fname0 := info.file.Name()
	Info(strings.Repeat("x", int(MaxSize))) // force a rollover
	if err != nil {
		t.Fatalf("info has error after big write: %v", err)
	}

	// Make sure the next log file gets a file name with a different
	// time stamp.
	//
	// TODO: determine whether we need to support subsecond log
	// rotation.  C++ does not appear to handle this case (nor does it
	// handle Daylight Savings Time properly).
	time.Sleep(1 * time.Second)

	Info("x") // create a new file
	if err != nil {
		t.Fatalf("error after rotation: %v", err)
	}
	fname1 := info.file.Name()
	if fname0 == fname1 {
		t.Errorf("info.f.Name did not change: %v", fname0)
	}
	if info.nbytes >= MaxSize {
		t.Errorf("file size was not reset: %d", info.nbytes)
	}
}

func TestLogBacktraceAt(t *testing.T) {
	setFlags()
	defer logging.swapLogging(logging.newLoggingBuffers())
	// The peculiar style of this code simplifies line counting and maintenance of the
	// tracing block below.
	var infoLine string
	setTraceLocation := func(file string, line int, ok bool, delta int) {
		if !ok {
			t.Fatal("could not get file:line")
		}
		_, file = filepath.Split(file)
		infoLine = fmt.Sprintf("%s:%d", file, line+delta)
		err := logging.traceLocation.Set(infoLine)
		if err != nil {
			t.Fatal("error setting log_backtrace_at: ", err)
		}
	}
	{
		// Start of tracing block. These lines know about each other's relative position.
		_, file, line, ok := runtime.Caller(0)
		setTraceLocation(file, line, ok, +2) // Two lines between Caller and Info calls.
		Info("we want a stack trace here")
	}
	numAppearances := strings.Count(loggingContents(infoLog), infoLine)
	if numAppearances < 2 {
		// Need 2 appearances, one in the log header and one in the trace:
		//   log_test.go:281: I0511 16:36:06.952398 02238 log_test.go:280] we want a stack trace here
		//   ...
		//   github.com/glog/glog_test.go:280 (0x41ba91)
		//   ...
		// We could be more precise but that would require knowing the details
		// of the traceback format, which may not be dependable.
		t.Fatal("got no trace back; log is ", loggingContents(infoLog))
	}
}

func TestExtractTimestamp(t *testing.T) {
	preffix := fmt.Sprintf("%s.%s.%s.log.", "geth_test", "sampleHost", "sampleUser")
	cases := []struct {
		name     string
		fileName string
		expected string
	}{
		{"valid INFO", preffix + "INFO.20171202-132113.2841", "20171202-132113"},
		{"valid WARNIG", preffix + "WARNING.20171202-210922.13848", "20171202-210922"},
		{"valid gzipped", preffix + "WARNING.20171202-210922.13848.gz", "20171202-210922"},
		{"extra long filename", preffix + "WARNING.20171202-210922.13848.gz.bak", "20171202-210922"},
		{"too short filename", preffix + "WARNING.20171202-21092", ""},
		{"no preffix", "WARNING.20171202-21092", ""},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			actual := extractTimestamp(test.fileName, preffix)
			if test.expected != actual {
				t.Errorf("Expected: '%s', actual: '%s'", test.expected, actual)
			}
		})
	}
}

func TestShouldRotate(t *testing.T) {
	// fixed date, to make tests stable, 04.12.2017 is Monday
	start := time.Date(2017, time.December, 4, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		nbytes   uint64
		len      int
		now      time.Time
		minSize  uint64
		maxSize  uint64
		interval Interval
		expected bool
	}{
		{"empty log no rotation", 0, 123, start, 0, 0, Never, false},
		{"empty log with hourly rotation", 0, 123, start, 0, 0, Hourly, false},
		{"empty log with size rotation", 0, 123, start, 0, 1024 * 1024, Never, false},
		{"log with hourly rotation after less than hour", 1234, 123, start.Add(45 * time.Minute), 0, 0, Hourly, false},
		{"log with hourly rotation after more than hour", 1234, 123, start.Add(65 * time.Minute), 0, 0, Hourly, true},
		{"log with size rotation below MinSize", 1024, 123, start, 512 * 1024, 1024 * 1024, Never, false},
		{"log with size rotation between MinSize and MaxSize", 765 * 1024, 123, start, 512 * 1024, 1024 * 1024, Never, false},
		{"log with size rotation above MaxSize", 1024*1024 - 100, 123, start, 512 * 1024, 1024 * 1024, Never, true},
		{"log with daily rotation after less than day", 1234, 123, start.Add(23 * time.Hour), 0, 0, Daily, false},
		{"log with daily rotation after more than day", 1234, 123, start.Add(25 * time.Hour), 0, 0, Daily, true},
		{"log with weekly rotation after less than week", 1234, 123, start.Add((6*24 + 23) * time.Hour), 0, 0, Weekly, false},
		{"log with weekly rotation after more than week", 1234, 123, start.Add((7*24 + 1) * time.Hour), 0, 0, Weekly, true},
		{"log with monthly rotation after less than month", 1234, 123, start.Add(14 * 24 * time.Hour), 0, 0, Monthly, false},
		{"log with monthly rotation after more than month", 1234, 123, start.Add(30 * 24 * time.Hour), 0, 0, Monthly, true},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			sb := &syncBuffer{nbytes: test.nbytes, time: start}
			MinSize = test.minSize
			MaxSize = test.maxSize
			RotationInterval = test.interval
			actual := sb.shouldRotate(test.len, test.now)
			if test.expected != actual {
				t.Errorf("Expected: '%v', actual: '%v'", test.expected, actual)
			}
		})
	}
}

func TestGzipFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "sample.log")
	data := strings.Repeat("lorem ipsum dolor sit amet", 4096)
	ioutil.WriteFile(file, []byte(data), 0600)

	err = gzipFile(file)
	if err != nil {
		t.Fatal(err)
	}

	gzipped := file + ".gz"

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file in directory, found %d", len(files))
	}
	if files[0].Size() == 0 || files[0].Size() >= int64(len(data)) {
		t.Errorf("expected: 0 < file size < %d [bytes], actual: %d [bytes]", len(data), files[0].Size())
	}
	if filepath.Join(dir, files[0].Name()) != gzipped {
		t.Errorf("expected filename: %s; actual filename: %s", gzipped, files[0].Name())
	}

	input, err := os.Open(filepath.Join(dir, files[0].Name()))
	if err != nil {
		t.Error(err)
	}
	defer input.Close()

	reader, err := gzip.NewReader(input)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	result, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Error(err)
	}
	if string(result) != data {
		t.Errorf("contents of gzip file are invalid")
	}
}

// there are always 10 interesting files, 10*1024b each
// some of them are gzipped
// there are WARN and INFO logs
// there are also 3 other files, that shoudn't be touched
// there are 2 "current files" (pointed by symlinks)
// global configuration options should be applied
// after running rotation, assertions about resulting data are checked
func testRotation(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	start := time.Date(2017, time.December, 06, 0, 0, 0, 0, time.UTC)

	data := []byte(strings.Repeat(".", 10*1024))
	logDate := start
	// generate files
	for i := 0; i < 5; i++ {
		infoF, infoL := logName("INFO", logDate.Add(1*time.Second))
		warnF, warnL := logName("WARNING", logDate.Add(10*time.Second).Add(1*time.Second))

		ioutil.WriteFile(filepath.Join(dir, infoF), data, 0600)
		ioutil.WriteFile(filepath.Join(dir, warnF), data, 0600)

		infoSL := filepath.Join(dir, infoL)
		os.Remove(infoSL)                             // ignore err
		os.Symlink(filepath.Join(dir, infoF), infoSL) // ignore err

		warnSL := filepath.Join(dir, warnL)
		os.Remove(warnSL)                             // ignore err
		os.Symlink(filepath.Join(dir, warnF), warnSL) // ignore err

		logDate = logDate.Add(24 * time.Hour)
	}
	dummy1 := "and_now_for_something_completely_different.log"
	ioutil.WriteFile(filepath.Join(dir, dummy1), data, 0600)
	dummy2, _ := logName("ERROR", start.Add(-10*24*time.Hour))
	dummy2 = "keep." + dummy2
	ioutil.WriteFile(filepath.Join(dir, dummy2), data, 0600)
	dummy3, _ := logName("INFO", start.Add(+10*24*time.Hour))
	dummy3 = strings.Replace(dummy3, userName, "differentUser", 1)
	ioutil.WriteFile(filepath.Join(dir, dummy3), data, 0600)

	// prepare environment
	logDirs = nil
	SetLogDir(dir)
	createLogDirs()

	// execute rotation
	sb := &syncBuffer{}
	sb.rotateOld(logDate)

	// make assertions
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that 3rd party files are intact
	dummy1ok := false
	dummy2ok := false
	dummy3ok := false
	totalSize := uint64(0)
	nGzipped := 0
	preffix := fmt.Sprintf("%s.%s.%s.log.", program, host, userName)
	maxTimestamp := ""
	if MaxAge > 0 {
		t := logDate.Add(-1 * MaxAge)
		maxTimestamp = fmt.Sprintf("%04d%02d%02d-%02d%02d%02d",
			t.Year(),
			t.Month(),
			t.Day(),
			t.Hour(),
			t.Minute(),
			t.Second(),
		)
	}
	for _, file := range files {
		sameSize := file.Size() == int64(len(data))
		switch file.Name() {
		case dummy1:
			dummy1ok = sameSize
		case dummy2:
			dummy2ok = sameSize
		case dummy3:
			dummy3ok = sameSize
		default:
			totalSize += uint64(file.Size())
			if maxTimestamp != "" && file.Mode().IsRegular() {
				timestamp := extractTimestamp(file.Name(), preffix)
				if strings.Compare(timestamp, maxTimestamp) < 0 {
					t.Errorf("Old file not removed properly: %s\n", file.Name())
				}
			}
		}
		if strings.HasSuffix(file.Name(), ".gz") {
			nGzipped++
		}

	}

	if !dummy1ok || !dummy2ok || !dummy3ok {
		t.Error("Some 3rd party files removed or modified!")
	}

	if MaxTotalSize > 0 && totalSize >= MaxTotalSize {
		t.Error("MaxTotalSize constraint violated!")
	}

	// 3 x 3rd-party files, 2 x symlink, 2 x current log files
	if Compress && nGzipped == len(files)-3-2-2 {
		t.Error("Some files not compressed!")
	}
}

func TestRotateOldFiles(t *testing.T) {
	MinSize = 0
	MaxSize = 1024
	RotationInterval = Never

	cases := []struct {
		name         string
		maxAge       time.Duration
		maxTotalSize uint64
		compress     bool
	}{
		{"no limits", 0, 0, false},
		{"no limits with compression", 0, 0, true},
		{"with age limit", 2 * 24 * time.Hour, 0, false},
		{"with size limit", 0, 5 * 1024, false},
		{"with both limits", 4 * 24 * time.Hour, 5 * 1024, false},
		{"with age limit and compression", 2 * 24 * time.Hour, 0, true},
		{"with size limit and compression", 0, 5 * 1024, true},
		{"with both limits and compression", 4 * 24 * time.Hour, 5 * 1024, true},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			MaxAge = test.maxAge
			MaxTotalSize = test.maxTotalSize
			Compress = test.compress
			testRotation(t)
		})
	}
}

func TestParseInterval(t *testing.T) {
	cases := []struct {
		value    string
		expected Interval
		err      bool
	}{
		{"never", Never, false},
		{"NeVeR", Never, false},
		{"daily", Daily, false},
		{"Daily", Daily, false},
		{"weekly", Weekly, false},
		{"weekLY", Weekly, false},
		{"monthly", Monthly, false},
		{"mONThLy", Monthly, false},
		{"invalid", Never, true},
		{"daily weekly", Never, true},
		{"none", Never, true},
	}

	for _, test := range cases {
		t.Run(test.value, func(t *testing.T) {
			interval, err := ParseInterval(test.value)
			if test.expected != interval {
				t.Error("Invalid interval value")
			}
			if test.err != (err != nil) {
				t.Error("Invalid error value")
			}
		})
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// See: https://stackoverflow.com/a/31832326/3474438
func randStringBytesMaskImprSrc(n int, src rand.Source) []byte {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return b
}

func TestLongRunningRotateOld(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	start := time.Date(2017, time.December, 06, 0, 0, 0, 0, time.UTC)

	fileSize := 512 * 1024

	logDate := start
	// generate files
	var src = rand.NewSource(time.Now().UnixNano())
	for i := 0; i < 64; i++ {
		infoF, infoL := logName("INFO", logDate.Add(1*time.Second))

		ioutil.WriteFile(filepath.Join(dir, infoF), randStringBytesMaskImprSrc(fileSize, src), 0600)

		infoSL := filepath.Join(dir, infoL)
		os.Remove(infoSL)                             // ignore err
		os.Symlink(filepath.Join(dir, infoF), infoSL) // ignore err

		logDate = logDate.Add(24 * time.Hour)
	}

	// prepare environment
	logDirs = nil
	SetLogDir(dir)
	createLogDirs()

	// execute rotation
	sb := &syncBuffer{}

	wg := sync.WaitGroup{}

	run := func() {
		wg.Add(1)
		defer wg.Done()
		sb.rotateOld(time.Now())
	}

	go run()
	for i := 0; i < 64; i++ {
		time.Sleep(32 * time.Millisecond)
		go run()
	}

	wg.Wait()
}

func BenchmarkHeaderLogging(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf, _, _ := logging.header(infoLog, 0)
		logging.putBuffer(buf)
	}
}

func BenchmarkHeaderDisplay(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf, _, _ := display.header(infoLog, 0)
		display.putBuffer(buf)
	}
}
