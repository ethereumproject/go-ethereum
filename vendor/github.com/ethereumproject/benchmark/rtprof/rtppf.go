package rtppf

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/ethereumproject/benchmark/rtprof/pprof/driver"
	"github.com/ethereumproject/benchmark/rtprof/pprof/measurement"
	"github.com/ethereumproject/benchmark/rtprof/pprof/profile"
	"net"
	"net/http"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

const defaultProfile = "@"

func isLocalhost(host string) bool {
	for _, v := range []string{"localhost", "127.0.0.1", "[::1]", "::1"} {
		if host == v {
			return true
		}
	}
	return false
}

type Profile struct {
	bf   bytes.Buffer
	prof *profile.Profile
	mu   sync.Mutex
	stop chan interface{}
	serv *http.Server
}

func (f *Profile) Stop() {
	close(f.stop)
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	f.serv.Shutdown(ctx)
}

func (f *Profile) Serve(args *driver.HTTPServerArgs) error {
	ln, err := net.Listen("tcp", args.Hostport)
	if err != nil {
		return err
	}
	isLocal := isLocalhost(args.Host)
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if isLocal {
			// Only allow local clients
			host, _, err := net.SplitHostPort(req.RemoteAddr)
			if err != nil || !isLocalhost(host) {
				http.Error(w, "permission denied", http.StatusForbidden)
				return
			}
		}
		h := args.Handlers[req.URL.Path]
		if h == nil {
			// Fall back to default behavior
			h = http.DefaultServeMux
		}
		h.ServeHTTP(w, req)
	})
	f.serv = &http.Server{Handler: handler}
	return f.serv.Serve(ln)
}

func (f *Profile) Fetch(src string, duration, timeout time.Duration) (*profile.Profile, string, error) {
	if src == defaultProfile {
		f.mu.Lock()
		for f.prof == nil {
			f.mu.Unlock()
			<-time.After(100 * time.Millisecond)
			f.mu.Lock()
		}
		f.mu.Unlock()
		return f.prof, "", nil
	}
	return nil, "", fmt.Errorf("unknown source %s", src)
}

func (f *Profile) Update(b []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, err := profile.ParseData(b); err != nil {
		return err
	} else {
		if f.prof == nil {
			f.prof = p
		} else {
			pfs := []*profile.Profile{f.prof, p}
			if err := measurement.ScaleProfiles(pfs); err != nil {
				return err
			}
			if p, err := profile.Merge(pfs); err != nil {
				return err
			} else {
				p.RemoveUninteresting()
				if err := p.CheckValid(); err != nil {
					return err
				}
				f.prof = p
			}
		}
	}
	return nil
}

func (f *Profile) UpdateLoop(interval time.Duration) {
	pprof.StartCPUProfile(&f.bf)
	for {
		select {
		case <-f.stop:
			return
		case <-time.After(interval):
			pprof.StopCPUProfile()
			if err := f.Update(f.bf.Bytes()); err != nil {
				fmt.Fprintf(os.Stderr, "failed to merge profiles: %s", err.Error())
			}
			f.bf.Reset()
			pprof.StartCPUProfile(&f.bf)
		}
	}
}

func (*Profile) ReadLine(prompt string) (string, error) {
	return "quit", nil
}

func (*Profile) PrintErr(a ...interface{})                    {}
func (*Profile) Print(a ...interface{})                       {}
func (*Profile) IsTerminal() bool                             { return false }
func (*Profile) SetAutoComplete(complete func(string) string) {}

var globalProfile *Profile

func Start(interval time.Duration, port int) {
	if globalProfile == nil {
		globalProfile := &Profile{stop: make(chan interface{})}
		go globalProfile.UpdateLoop(interval)
		go driver.PProf(&driver.Options{
			Fetch: globalProfile,
			UI:    globalProfile,
			HTTPServer: func(args *driver.HTTPServerArgs) error {
				return globalProfile.Serve(args)
			},
			Flagset: &FlagSet{
				FlagSet: flag.NewFlagSet("rtppf", flag.ContinueOnError),
				Args:    []string{fmt.Sprintf("--http=:%d", port), defaultProfile},
			},
		})
	}
}

func Stop() {
	if globalProfile != nil {
		globalProfile.Stop()
		globalProfile = nil
	}
}
