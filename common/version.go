package common

import (
	"path/filepath"
	"os/exec"
	"regexp"
	"runtime"
	"os"
	"strings"
	"os/user"
	"github.com/denisbrodbeck/machineid"
	"strconv"
	"math/rand"
	"time"
)

var clientSessionIdentity *ClientSessionIdentityT
var VCRevision = mustGetGitHeadHash(8) // global because we'll use it to name log files
var SessionID string // global because we use in mlog fns to inject for all data points

func init() {
	rand.Seed(time.Now().UnixNano())
	SessionID = randStringBytes(8)
	SetClientSessionIdentity()
}

// ClientSessionIdentityT holds values describing the client, environment, and session.
type ClientSessionIdentityT struct {
	Revision string `json:"head"`
	Version string `json:"version"`
	Hostname string `json:"host"`
	Username string `json:"user"`
	MachineID string `json:"machineid"`
	Goos string `json:"goos"`
	Goarch string `json:"goarch"`
	Goversion string `json:"goversion"`
	Pid int `json:"pid"`
	SessionID string `json:"session"`
	StartTime time.Time `json:"start"`
}

// String is the stringer fn for ClientSessionIdentityT
// TODO/PTAL: meh...
func (s *ClientSessionIdentityT) String() string {
	sep := func(sep string, args ...string) string {
		var s string
		for i, v := range args {
			if i != len(args)-1 {
				s += v + sep
			} else {
				s += v
			}
		}
		return s
	}
	// Use "_" because it's unlikely to conflict with semver or other data delimiters
	// The order is is intended to move from 'client'/os/software identifiers, then to session, host, pid granular identifiers, ie macro->micro ids.
	return sep("_", s.Revision, s.Goos, s.Goarch, s.SessionID, s.Hostname, s.Username, s.MachineID, strconv.Itoa(s.Pid))
}


// Helpers for random sessionid string.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// SetClientSessionIdentity sets the global variable describing details about the client and session.
// It should only be called once per session.
func SetClientSessionIdentity() {
	var hostname, userName string
	var err error

	hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	current, err := user.Current()
	if err == nil {
		userName = current.Username
	} else {
		userName = "unknown"
	}
	// Sanitize userName since it may contain filepath separators on Windows.
	userName = strings.Replace(userName, `\`, "_", -1)

	var mid string
	var e error
	mid, e = machineid.ID()
	if e == nil {
		mid, e = machineid.ProtectedID(mid)
	}
	if e != nil {
		mid = hostname + "." + userName
	}

	clientSessionIdentity = &ClientSessionIdentityT{
		Revision:  VCRevision,
		Hostname:  hostname,
		Username:  userName,
		MachineID: mid[:8], // because we don't care that much
		Goos:      runtime.GOOS,
		Goarch:    runtime.GOARCH,
		Goversion: runtime.Version(),
		Pid:       os.Getpid(),
		SessionID: SessionID,
		StartTime: time.Now(),
	}
}

// GetClientSessionIdentity is the getter fn for a the clientSessionIdentity value.
func GetClientSessionIdentity() *ClientSessionIdentityT {
	return clientSessionIdentity
}

// mustGetGitValue uses the relative path of this 'caller' file to determine the expected location of the go-ethereum project's
// working directory. If it encounters an error a blank string will be returned.
func mustGetGitValue(gitArgs ...string) string {
	// Get the path of this file
	_, f, _, ok := runtime.Caller(1)
	if ok {
		d := filepath.Dir(f) // .../cmd/geth
		// Derive git project dir
		d = filepath.Join(d, "..", ".git")
		// Ignore error
		gitArgsPrefixed := []string{"--git-dir", d}
		gitArgsPrefixed = append(gitArgsPrefixed, gitArgs...)
		if o, err := exec.Command("git", gitArgsPrefixed...).Output(); err == nil {
			// Remove newline
			re := regexp.MustCompile(`\r?\n`) // Handle both Windows carriage returns and *nix newlines
			return re.ReplaceAllString(string(o), "")
		}
	}
	return ""
}

// mustGetGitHeadHash returns the head hash of the go-ethereum/ project working dir, or "" if it encounters an error.
// It returns a string equal to or shorter than argued int.
// TODO: maybe include a way to show if the wd is not on the bespoke HEAD, ie if the wd is dirty.
func mustGetGitHeadHash(abbrvN int) string {
	if v := mustGetGitValue("rev-parse", "HEAD"); v != "" {
		if len(v) >= abbrvN {
			return v[:abbrvN]
		}
		return v
	}
	return ""
}

// MustSourceBuildVersionFormatted returns a string which will be used to augment cases when the '-ldflags' are not used
// to set the client main.Version. Note that this value 'Version' should not be confused with the 'Revision' value
// field of clientSessionIdentity.
func MustSourceBuildVersionFormatted() string {
	if v := mustGetGitValue("describe", "--tags"); v != "" {
		return "source-" + v
	} else {
		return "source-unknown"
	}
}

