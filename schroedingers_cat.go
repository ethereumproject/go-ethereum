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
	"regexp"
	"runtime"
	"strings"
	"time"
)

// tests file should contains only lines or blank lines of the form:
// ./eth/downloader TestCanonicalSynchronisation
//./p2p TestPeerProtoReadMsg
var testsFile string

// allowed times to try to get a nondeterministic test to pass
var trialsAllowed int

var goExecutablePath string
var commandPrefix []string

var errEmptyLine = errors.New("empty line")
var errCommentLine = errors.New("comment line")

type test struct {
	pkg  string
	name string
}

func init() {
	goExecutablePath = getGoPath()
	commandPrefix = getCommandPrefix()
	flag.StringVar(&testsFile, "f", "schroedinger-tests.txt", "file argument")
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
	s = strings.Trim(s, " ")
	if len(s) == 0 {
		return nil, errEmptyLine
	}
	re := regexp.MustCompile(`/^$/`)
	if re.MatchString(s) {
		return nil, errEmptyLine
	}
	re = regexp.MustCompile(`/^#/`)
	if re.MatchString(s) {
		return nil, errCommentLine
	}
	t = parseLinePackageTest(s)
	return t, nil
}

func runTest(t *test) error {
	args := fmt.Sprintf("test %s -run %s", t.pkg, t.name)
	cmd := exec.Command(commandPrefix[0], commandPrefix[1], goExecutablePath+" "+args)
	return cmd.Run()
}

func main() {
	testsFile = filepath.Clean(testsFile)
	testsFile, _ := filepath.Abs(testsFile)

	log.Println("* go executable path:", goExecutablePath)
	log.Println("* command prefix:", strings.Join(commandPrefix, " "))
	log.Println("* tests file:", testsFile)
	log.Println("* trials: ", trialsAllowed)

	file, err := os.Open(testsFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	allstart := time.Now()
	defer log.Printf("FINISHED (%v)", time.Since(allstart))

outer:
	for scanner.Scan() {
		t, err := handleLine(scanner.Text())
		if err != nil {
			if err == errCommentLine {
				log.Println(scanner.Text())
			}
			continue outer
		}
		for i := 0; i < trialsAllowed; i++ {
			log.Println(t.pkg, t.name)
			start := time.Now()
			if e := runTest(t); e == nil {
				log.Printf("- PASS (%v) %d/%d", time.Since(start), i+1, trialsAllowed)
				continue outer
			} else {
				log.Printf("- FAIL (%v) %d/%d: %v", time.Since(start), i+1, trialsAllowed, e)
			}
		}
		log.Fatalln("FAIL", t.pkg, t.name)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
