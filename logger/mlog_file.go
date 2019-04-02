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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/logger/glog"
)

type mlogFormatT uint

const (
	mLOGPlain mlogFormatT = iota + 1
	mLOGKV
	MLOGJSON
)

var (
	// If non-empty, overrides the choice of directory in which to write logs.
	// See createLogDirs for the full list of possible destinations.
	mLogDir    = new(string)
	mLogFormat = MLOGJSON

	errMLogComponentUnavailable = errors.New("provided component name is unavailable")
	ErrUnkownMLogFormat         = errors.New("unknown mlog format")

	// MLogRegistryAvailable contains all available mlog components submitted by any package
	// with MLogRegisterAvailable.
	mLogRegistryAvailable = make(map[mlogComponent][]*MLogT)
	// MLogRegistryActive contains all registered mlog component and their respective loggers.
	mLogRegistryActive = make(map[mlogComponent]*Logger)
	mlogRegLock        sync.RWMutex

	// Abstract literals (for documentation examples, labels)
	mlogInterfaceExamples = map[string]interface{}{
		"INT":            int(0),
		"BIGINT":         new(big.Int),
		"STRING":         "string",
		"BOOL":           true,
		"QUOTEDSTRING":   "string with spaces",
		"STRING_OR_NULL": nil,
		"DURATION":       time.Minute + time.Second*3 + time.Millisecond*42,
		"OBJECT":         common.GetClientSessionIdentity(),
	}

	MLogStringToFormat = map[string]mlogFormatT{
		"plain": mLOGPlain,
		"kv":    mLOGKV,
		"json":  MLOGJSON,
	}

	// Global var set to false if "--mlog=off", used to simply/
	// speed-up checks to avoid performance penalty if mlog is
	// off.
	isMlogEnabled bool
)

func (f mlogFormatT) String() string {
	switch f {
	case MLOGJSON:
		return "json"
	case mLOGKV:
		return "kv"
	case mLOGPlain:
		return "plain"
	}
	panic(ErrUnkownMLogFormat)
}

// MLogT defines an mlog LINE
type MLogT struct {
	sync.Mutex
	// TODO: can remove these json tags, since we have a custom MarshalJSON fn
	Description string        `json:"-"`
	Receiver    string        `json:"receiver"`
	Verb        string        `json:"verb"`
	Subject     string        `json:"subject"`
	Details     []MLogDetailT `json:"details"`
}

// MLogDetailT defines an mlog LINE DETAILS
type MLogDetailT struct {
	Owner string      `json:"owner"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

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

func SetMlogEnabled(b bool) {
	isMlogEnabled = b
}

func MlogEnabled() bool {
	return isMlogEnabled
}

// MLogRegisterAvailable is called for each log component variable from a package/mlog.go file
// as they set up their mlog vars.
// It registers an mlog component as Available.
func MLogRegisterAvailable(name string, lines []*MLogT) mlogComponent {
	c := mlogComponent(name)
	mlogRegLock.Lock()
	mLogRegistryAvailable[c] = lines
	mlogRegLock.Unlock()
	return c
}

// GetMlogRegistryAvailable returns copy of all registered components mapping
func GetMLogRegistryAvailable() map[mlogComponent][]*MLogT {
	mlogRegLock.RLock()
	defer mlogRegLock.RUnlock()

	ret := make(map[mlogComponent][]*MLogT)
	for k, v := range mLogRegistryAvailable {
		ret[k] = make([]*MLogT, len(v))
		copy(ret[k], v)
	}
	return ret
}

// GetMlogRegistryActive returns copy of all active components mapping
func GetMLogRegistryActive() map[mlogComponent]*Logger {
	mlogRegLock.RLock()
	defer mlogRegLock.RUnlock()

	ret := make(map[mlogComponent]*Logger)
	for k, v := range mLogRegistryActive {
		ret[k] = v
	}
	return ret
}

// MLogRegisterComponentsFromContext receives a comma-separated string of
// desired mlog components.
// It returns an error if the specified mlog component is unavailable.
// For each available component, the desires mlog components are registered as active,
// creating new loggers for each.
// If the string begins with '!', the function will remove the following components from the
// default list
func MLogRegisterComponentsFromContext(s string) error {
	// negation
	var negation bool
	if strings.HasPrefix(s, "!") {
		negation = true
		s = strings.TrimPrefix(s, "!")
	}
	ss := strings.Split(s, ",")

	registry := GetMLogRegistryAvailable()

	if !negation {
		for _, c := range ss {
			ct := strings.TrimSpace(c)
			if _, ok := registry[mlogComponent(ct)]; !ok {
				return fmt.Errorf("%v: '%s'", errMLogComponentUnavailable, ct)
			}
			MLogRegisterActive(mlogComponent(ct))
		}
		return nil
	}
	// Register all
	for c := range registry {
		MLogRegisterActive(c)
	}
	// then remove
	for _, u := range ss {
		ct := strings.TrimSpace(u)
		mlogRegisterInactive(mlogComponent(ct))
	}
	return nil
}

// MLogRegisterActive registers a component for mlogging.
// Only registered loggers will write to mlog file.
func MLogRegisterActive(component mlogComponent) {
	mlogRegLock.Lock()
	mLogRegistryActive[component] = NewLogger(string(component))
	mlogRegLock.Unlock()
}

func mlogRegisterInactive(component mlogComponent) {
	mlogRegLock.Lock()
	delete(mLogRegistryActive, component) // noop if nil
	mlogRegLock.Unlock()
}

// SendMLog writes enabled component mlogs to file if the component is registered active.
func (msg *MLogT) Send(c mlogComponent) {
	mlogRegLock.RLock()
	if l, exists := mLogRegistryActive[c]; exists {
		l.SendFormatted(GetMLogFormat(), 1, msg, c)
	}
	mlogRegLock.RUnlock()
}

func (l *Logger) SendFormatted(format mlogFormatT, level LogLevel, msg *MLogT, c mlogComponent) {
	switch format {
	case mLOGKV:
		l.Sendln(level, msg.FormatKV())
	case MLOGJSON:
		logMessageC <- stdMsg{level, string(msg.FormatJSON(c))}
	case mLOGPlain:
		l.Sendln(level, string(msg.FormatPlain()))
	//case MLOGDocumentation:
	// don't handle this because this is just for one-off help/usage printing documentation
	default:
		glog.Fatalf("Unknown mlog format: %v", format)
	}
}

// SetMLogDir sets the mlog directory, into which one mlog file per session
// will be written.
func SetMLogDir(str string) {
	*mLogDir = str
}

func GetMLogDir() string {
	m := *mLogDir
	return m
}

func SetMLogFormat(format mlogFormatT) {
	mLogFormat = format
}

func GetMLogFormat() mlogFormatT {
	return mLogFormat
}

func SetMLogFormatFromString(formatString string) error {
	if f := MLogStringToFormat[formatString]; f < 1 {
		return ErrUnkownMLogFormat
	} else {
		SetMLogFormat(f)
	}
	return nil
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
	name = fmt.Sprintf("%s.mlog.%s.%04d%02d%02d-%02d%02d%02d.%d.log",
		program,
		common.SessionID,
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

	return f, fname, nil
}

func (m *MLogT) placeholderize() {
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
}

func (m *MLogT) FormatJSON(c mlogComponent) []byte {
	b, _ := m.MarshalJSON(c)
	return b
}

func (m *MLogT) FormatKV() (out string) {
	m.Lock()
	defer m.Unlock()
	m.placeholderize()
	out = fmt.Sprintf("%s %s %s session=%s", m.Receiver, m.Verb, m.Subject, common.SessionID)
	for _, d := range m.Details {
		v := fmt.Sprintf("%v", d.Value)
		// quote strings which contains spaces
		if strings.Contains(v, " ") {
			v = fmt.Sprintf(`"%v"`, d.Value)
		}
		out += fmt.Sprintf(" %s=%v", d.EventName(), v)
	}
	return out
}

func (m *MLogT) FormatPlain() (out string) {
	m.Lock()
	defer m.Unlock()
	m.placeholderize()
	out = fmt.Sprintf("%s %s %s %s", m.Receiver, m.Verb, m.Subject, common.SessionID)
	for _, d := range m.Details {
		v := fmt.Sprintf("%v", d.Value)
		// quote strings which contains spaces
		if strings.Contains(v, " ") {
			v = fmt.Sprintf(`"%v"`, d.Value)
		}
		out += fmt.Sprintf(" %v", v)
	}
	return out
}

func (m *MLogT) MarshalJSON(c mlogComponent) ([]byte, error) {
	m.Lock()
	defer m.Unlock()
	var obj = make(map[string]interface{})
	obj["event"] = m.EventName()
	obj["ts"] = time.Now()
	obj["session"] = common.SessionID
	obj["component"] = string(c)
	for _, d := range m.Details {
		obj[d.EventName()] = d.Value
	}
	return json.Marshal(obj)
}

func (m *MLogT) FormatJSONExample(c mlogComponent) []byte {
	mm := &MLogT{
		Receiver: m.Receiver,
		Verb:     m.Verb,
		Subject:  m.Subject,
	}
	var dets []MLogDetailT
	for _, d := range m.Details {
		ex := mlogInterfaceExamples[d.Value.(string)]
		// Type of var not matched to interface example
		if ex == "" {
			continue
		}
		dets = append(dets, MLogDetailT{
			Owner: d.Owner,
			Key:   d.Key,
			Value: ex,
		})
	}
	mm.Details = dets
	b, _ := mm.MarshalJSON(c)
	return b
}

// FormatDocumentation prints wiki-ready documentation for all available component mlog LINES.
// Output should be in markdown.
func (m *MLogT) FormatDocumentation(cmp mlogComponent) (out string) {

	// Get the json example before converting to abstract literal format, eg STRING -> $STRING
	// This keeps the interface example dictionary as a separate concern.
	exJSON := string(m.FormatJSONExample(cmp))

	// Set up arbitrary documentation abstract literal format
	docDetails := []MLogDetailT{}
	for _, d := range m.Details {
		dd := d.AsDocumentation()
		docDetails = append(docDetails, *dd)
	}
	m.Details = docDetails

	exPlain := m.FormatPlain()
	exKV := m.FormatKV()

	t := time.Now()
	lStandardHeaderDateTime := fmt.Sprintf("%4d/%02d/%02d %02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	cmpS := fmt.Sprintf("[%s]", cmp)

	out += fmt.Sprintf(`
#### %s %s %s
%s

__Key value__:
`+"```"+`
%s %s %s
`+"```"+`

__JSON__:
`+"```json"+`
%s
`+"```"+`

__Plain__:
`+"```"+`
%s %s %s
`+"```"+`

_%d detail values_:

`, m.Receiver, m.Verb, m.Subject,
		m.Description,
		lStandardHeaderDateTime,
		cmpS,
		exKV,
		exJSON,
		lStandardHeaderDateTime,
		cmpS,
		exPlain,
		len(m.Details))

	var details string
	for _, d := range m.Details {
		details += fmt.Sprintf("- `%s`: %s\n", d.EventName(), d.Value)
	}
	details += "\n"

	out += details
	return out
}

// EventName implements the JsonMsg interface in case wanting to use existing half-established json logging system
func (m *MLogT) EventName() string {
	r := strings.ToLower(m.Receiver)
	v := strings.ToLower(m.Verb)
	s := strings.ToLower(m.Subject)
	return strings.Join([]string{r, v, s}, ".")
}

func (m *MLogDetailT) EventName() string {
	o := strings.ToLower(m.Owner)
	k := strings.ToLower(m.Key)
	return strings.Join([]string{o, k}, ".")
}

func (m *MLogDetailT) AsDocumentation() *MLogDetailT {
	m.Value = fmt.Sprintf("$%s", m.Value)
	return m
}

// AssignDetails is a setter function for setting values for pre-existing details.
// It accepts a variadic number of empty interfaces.
// If the number of arguments does not match  the number of established details
// for the receiving MLogT, it will fatal error.
// Arguments MUST be provided in the order in which they should be applied to the
// slice of existing details.
func (m *MLogT) AssignDetails(detailVals ...interface{}) *MLogT {
	// Check for congruence between argument length and registered details.
	if len(detailVals) != len(m.Details) {
		glog.Fatal(m.EventName(), "wrong number of details set, want: ", len(m.Details), "got:", len(detailVals))
	}

	m.Lock()
	for i, detailval := range detailVals {
		m.Details[i].Value = detailval
	}
	m.Unlock()

	return m
}
