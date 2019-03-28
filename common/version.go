package common

import (
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
)

var clientSessionIdentity *ClientSessionIdentityT
var SessionID string // global because we use in mlog fns to inject for all data points

func init() {
	initClientSessionIdentity()
}

// ClientSessionIdentityT holds values describing the client, environment, and session.
type ClientSessionIdentityT struct {
	Version   string    `json:"version"`
	Hostname  string    `json:"host"`
	Username  string    `json:"user"`
	MachineID string    `json:"machineid"`
	Goos      string    `json:"goos"`
	Goarch    string    `json:"goarch"`
	Goversion string    `json:"goversion"`
	Pid       int       `json:"pid"`
	SessionID string    `json:"session"`
	StartTime time.Time `json:"start"`
}

// String is the stringer fn for ClientSessionIdentityT
func (s *ClientSessionIdentityT) String() string {
	return fmt.Sprintf("VERSION=%s GO=%s GOOS=%s GOARCH=%s SESSIONID=%s HOSTNAME=%s USER=%s MACHINE=%s PID=%d",
		s.Version, s.Goversion, s.Goos, s.Goarch, s.SessionID, s.Hostname, s.Username, s.MachineID, s.Pid)
}

// Helpers for random sessionid string.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(rng *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rng.Intn(len(letterBytes))]
	}
	return string(b)
}

// initClientSessionIdentity sets the global variable describing details about the client and session.
func initClientSessionIdentity() {
	rng := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	SessionID = randStringBytes(rng, 4)

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
		Version:   "unknown",
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

func SetClientVersion(version string) {
	if clientSessionIdentity != nil {
		clientSessionIdentity.Version = version
	}
}

// GetClientSessionIdentity is the getter fn for a the clientSessionIdentity value.
func GetClientSessionIdentity() *ClientSessionIdentityT {
	return clientSessionIdentity
}
