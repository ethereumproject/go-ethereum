// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.


// File I/O and registry for mlogs.

package logger

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"bytes"
	"runtime"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"sync"
)

var (
	// If non-empty, overrides the choice of directory in which to write logs.
	// See createLogDirs for the full list of possible destinations.
	mLogDir *string = new(string)

	errMLogComponentUnavailable = errors.New("provided component name is unavailable")

	// MLogRegistryAvailable contains all available mlog components submitted by any package
	// with MLogRegisterAvailable.
	MLogRegistryAvailable = make(map[mlogComponent][]MLogT)
	// MLogRegistryActive contains all registered mlog component and their respective loggers.
	MLogRegistryActive = make(map[mlogComponent]*Logger)
	mlogRegLock        sync.RWMutex
)

// mlogComponent is used as a golang receiver type that can call Send(logLine).
type mlogComponent string

// The following vars and init() essentially duplicate those found in glog_file;
// the reason for the non-DRYness of that is that this allows us flexibility
// as we finalize the spec and format for the mlog lines, allowing customization
// of the establish system if desired, without exporting the vars from glog.
var (
	pid      = os.Getpid()
	program  = filepath.Base(os.Args[0])
	host     = "unknownhost"
	userName = "unknownuser"
)

func init() {
	h, err := os.Hostname()
	if err == nil {
		host = shortHostname(h)
	}

	current, err := user.Current()
	if err == nil {
		userName = current.Username
	}

	// Sanitize userName since it may contain filepath separators on Windows.
	userName = strings.Replace(userName, `\`, "_", -1)
}

// MLogRegisterAvailable is called for each log component variable from a package/mlog.go file
// as they set up their mlog vars.
// It registers an mlog component as Available.
func MLogRegisterAvailable(name string, lines []MLogT) mlogComponent {
	c := mlogComponent(name)
	mlogRegLock.Lock()
	MLogRegistryAvailable[c] = lines
	mlogRegLock.Unlock()
	return c
}

// MLogRegisterComponentsFromContext receives a comma-separated string of
// desired mlog components.
// It returns an error if the specified mlog component is unavailable.
// For each available component, the desires mlog components are registered as active,
// creating new loggers for each.
func MLogRegisterComponentsFromContext(s string) error {
	ss := strings.Split(s, ",")
	for _, c := range ss {
		ct := strings.TrimSpace(c)
		if MLogRegistryAvailable[mlogComponent(ct)] != nil {
			MLogRegisterActive(mlogComponent(ct))
			continue
		}
		return fmt.Errorf("%v: '%s'", errMLogComponentUnavailable, ct)
	}
	return nil
}

// MLogRegisterActive registers a component for mlogging.
// Only registered loggers will write to mlog file.
func MLogRegisterActive(component mlogComponent) {
	mlogRegLock.Lock()
	MLogRegistryActive[component] = NewLogger(string(component))
	mlogRegLock.Unlock()
}

// SendMLog writes enabled component mlogs to file if the component is registered active.
func (c mlogComponent) Send(logLine string) {
	mlogRegLock.RLock()
	if l := MLogRegistryActive[c]; l != nil {
		l.Sendf(1, logLine)
	}
	mlogRegLock.RUnlock()
}

// SetMLogDir sets the mlog directory, into which one mlog file per session
// will be written.
func SetMLogDir(str string) {
	*mLogDir = str
}

func createLogDirs() error {
	if *mLogDir != "" {
		return os.MkdirAll(*mLogDir, os.ModePerm)
	}
	return errors.New("createLogDirs received empty string")
}

// shortHostname returns its argument, truncating at the first period.
// For instance, given "www.google.com" it returns "www".
func shortHostname(hostname string) string {
	if i := strings.Index(hostname, "."); i >= 0 {
		return hostname[:i]
	}
	return hostname
}

// logName returns a new log file name containing tag, with start time t, and
// the name for the symlink for tag.
func logName(t time.Time) (name, link string) {
	name = fmt.Sprintf("%s.%s.%s.mlog.%04d%02d%02d-%02d%02d%02d.%d",
		program,
		host,
		userName,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		pid)
	return name, program + ".log"
}

// CreateMLogFile creates a new log file and returns the file and its filename, which
// contains tag ("INFO", "FATAL", etc.) and t.  If the file is created
// successfully, create also attempts to update the symlink for that tag, ignoring
// errors.
func CreateMLogFile(t time.Time) (f *os.File, filename string, err error) {

	if e := createLogDirs(); e != nil {
		return nil, "", e
	}

	name, link := logName(t)
	fname := filepath.Join(*mLogDir, name)

	f, e := os.Create(fname)
	if e != nil {
		err = e
		return nil, fname, err
	}

	symlink := filepath.Join(*mLogDir, link)
	os.Remove(symlink)        // ignore err
	os.Symlink(name, symlink) // ignore err

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Log file created at: %s\n", t.Format("2006/01/02 15:04:05"))
	fmt.Fprintf(&buf, "Running on machine: %s\n", host)
	fmt.Fprintf(&buf, "Binary: Built with %s %s for %s/%s\n", runtime.Compiler, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	cmps := []string{}
	for k := range MLogRegistryActive {
		cmps = append(cmps, string(k))
	}
	fmt.Fprintf(&buf, "Registered components: %v\n", cmps) // no need for fancy formatting
	fmt.Fprintln(&buf, glog.Separator("-"))
	f.Write(buf.Bytes())

	return f, fname, nil
}

// MLogT defines an mlog LINE
type MLogT struct {
	Description string
	Receiver string
	Verb string
	Subject string
	Details []MLogDetailT
}

// MLogDetailT defines an mlog LINE DETAILS
type MLogDetailT struct {
	Owner string
	Key string
	Value interface{}
}

// SetDetailValues is a setter function for setting values for pre-existing details.
// It accepts a variadic number of empty interfaces.
// If the number of arguments does not match  the number of established details
// for the receiving MLogT, it will fatal error.
// Arguments MUST be provided in the order in which they should be applied to the
// slice of existing details.
func (m MLogT) SetDetailValues(detailVals ...interface{}) MLogT {

	// Check for congruence between argument length and registered details.
	if len(detailVals) != len(m.Details) {
		glog.Fatal("mlog: wrong number of details set, want: ", len(m.Details), "got:", len(detailVals))
	}

	for i, detailval := range detailVals {
		m.Details[i].Value = detailval
	}

	return m
}

// String implements the 'stringer' interface for
// an MLogT struct.
// eg. $RECEIVER $SUBJECT $VERB $RECEIVER:DETAIL $RECEIVER:DETAIL $SUBJECT:DETAIL $SUBJECT:DETAIL
func (m MLogT) String(documentation ...bool) string {
	placeholderEmpty := "-"
	if m.Receiver == "" {
		m.Receiver = placeholderEmpty
	}
	if m.Subject == "" {
		m.Subject = placeholderEmpty
	}
	if m.Verb == "" {
		m.Verb = placeholderEmpty
	}
	out := fmt.Sprintf("%s %s %s", m.Receiver, m.Verb, m.Subject)
	for _, d := range m.Details {
		out += " " + d.String(documentation...)
	}
	if documentation != nil && len(documentation) > 0 && documentation[0] {
		out += fmt.Sprintf("\n    %s", m.Description)
	}
	return out
}

// String implements the stringer interface for mlog details.
// It can used to provide raw mlog-formatted strings, or
// strings formatted for self-documentation.
func (d MLogDetailT) String(documentation ...bool) string {
	if documentation != nil && len(documentation) > 0 && documentation[0] {
		return fmt.Sprintf("$%s:%s:%s", d.Owner, d.Key, d.Value)
	}
	return fmt.Sprintf("[%v]", d.Value)
}