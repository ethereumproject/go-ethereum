package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const commentPattern = "#"

var errCommentLine = errors.New("comment line")
var errEmptyLine = errors.New("empty line")

// tests file should contains only lines or blank lines of the form:
// ./eth/downloader TestCanonicalSynchronisation
// or
// github.com/ethereumproject/go-ethereum/eth/downloader TestFastCriticalRestarts
var testsFile string

// allowed times to try to get a nondeterministic test to pass
var trialsAllowed int

// string to match to *list tests
var whitelistMatch string
var blacklistMatch string

// different for windows
var goExecutablePath string
var commandPrefix []string

type test struct {
	pkg  string
	name string
}

func init() {
	goExecutablePath = getGoPath()
	commandPrefix = getCommandPrefix()
	flag.StringVar(&testsFile, "f", "schroedinger-tests.txt", "file argument")
	flag.StringVar(&whitelistMatch, "w", "", "whitelist lines containing")
	flag.StringVar(&blacklistMatch, "b", "", "blacklist lines containing")
	flag.IntVar(&trialsAllowed, "t", 3, "allowed trials before nondeterministic test actually fails")
	flag.Parse()
}

func getGoPath() string {
	return filepath.Join(runtime.GOROOT(), "bin", "go")
}

func getCommandPrefix() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C"}
	}
	return []string{"/bin/sh", "-c"}
}

func parseLinePackageTest(s string) *test {
	t := &test{}
	lsep := strings.Split(s, " ")
	t.pkg, t.name = lsep[0], lsep[1]
	t.pkg = strings.Replace(t.pkg, "/", string(filepath.Separator), -1)
	return t
}

func handleLine(s string) (*test, error) {
	var t *test
	ss := strings.Trim(s, " ")
	if len(ss) == 0 {
		return nil, errEmptyLine
	}
	if strings.HasPrefix(ss, commentPattern) {
		return nil, errCommentLine
	}
	if strings.Contains(ss, commentPattern) {
		sss := strings.Split(ss, commentPattern)
		ss = strings.Trim(sss[0], " ")
	}
	t = parseLinePackageTest(ss)
	return t, nil
}

func runTest(t *test) (string, error) {
	args := fmt.Sprintf("test %s -run %s", t.pkg, t.name)
	cmd := exec.Command(commandPrefix[0], commandPrefix[1], goExecutablePath+" "+args)
	out, err := cmd.Output()
	return string(out), err
}

func collectTests(f string) (tests []*test, err error) {
	file, err := os.Open(f)
	if err != nil {
		return tests, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		t, e := handleLine(scanner.Text())
		if e == nil {
			tests = append(tests, t)
		}
	}

	return tests, scanner.Err()
}

func filterTests(tests []*test, allowed func(*test) bool) []*test {
	var out []*test
	for _, t := range tests {
		if allowed(t) {
			out = append(out, t)
		}
	}
	return out
}

func tryTest(t *test, c chan error) {
	for i := 0; i < trialsAllowed; i++ {
		start := time.Now()
		if o, e := runTest(t); e == nil {
			log.Println(t.pkg, t.name)
			log.Printf("- PASS (%v) %d/%d", time.Since(start), i+1, trialsAllowed)
			c <- nil
			return
		} else {
			log.Println(t.pkg, t.name)
			log.Println(o)
			log.Printf("- FAIL (%v) %d/%d: %v", time.Since(start), i+1, trialsAllowed, e)
		}
	}
	c <- fmt.Errorf("FAIL %s %s", t.pkg, t.name)
}

func lineMatchList(line string, whites, blacks []string) bool {
	if blacks != nil && len(blacks) > 0 {
		for _, m := range blacks {
			if strings.Contains(line, m) {
				return false
			}
		}
	}
	if whites != nil && len(whites) > 0 {
		for _, m := range whites {
			if !strings.Contains(line, m) {
				return false
			} else {
				return true
			}
		}
	}
	return true
}

func parseMatchList(list string) []string {
	// eg. "", "downloader,fetcher", "sync"
	if len(list) == 0 {
		return nil
	}
	ll := strings.Trim(list, " ")
	return strings.Split(ll, ",")
}

func main() {
	if (whitelistMatch != "" && blacklistMatch != "") && whitelistMatch == blacklistMatch {
		log.Fatal("whitelist cannot match blacklist")
	}
	whites := parseMatchList(whitelistMatch)
	blacks := parseMatchList(blacklistMatch)

	testsFile = filepath.Clean(testsFile)
	testsFile, _ := filepath.Abs(testsFile)

	allowed := func(t *test) bool {
		return lineMatchList(t.pkg+" "+t.name, whites, blacks)
	}

	alltests, err := collectTests(testsFile)
	if err != nil {
		log.Fatal(err)
	}

	tests := filterTests(alltests, allowed)

	log.Println("* go executable path:", goExecutablePath)
	log.Println("* command prefix:", strings.Join(commandPrefix, " "))
	log.Println("* tests file:", testsFile)
	log.Println("* trials allowed: ", trialsAllowed)
	log.Println("* blacklist: ", blacks)
	log.Println("* whitelist: ", whites)
	log.Printf("* running %d/%d tests", len(tests), len(alltests))

	var results = make(chan error, len(tests))

	allstart := time.Now()
	defer func() {
		log.Printf("FINISHED (%v)", time.Since(allstart))
	}()

	go func() {
		for _, t := range tests {
			tryTest(t, results)
		}
	}()

	for i := 0; i < len(tests); i++ {
		if e := <-results; e != nil {
			log.Fatal(e)
		}
	}

	close(results)
}
