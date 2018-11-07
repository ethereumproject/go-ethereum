// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// json2test reformats 'go test -json' output as text as if the -json
// flag were not passed to go test. It is useful if you want to
// analyze go test -json output, but still want a human readable test
// log.
//
// Usage:
//
//  go test -json > out.txt; <analysis program> out.txt; cat out.txt | json2test
//
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

var errTestFailure = errors.New("testfailure")

func main() {
	err := writeTestOutput(os.Stdin, os.Stdout)
	if err == errTestFailure {
		os.Exit(1)
	} else if err != nil {
		log.Fatal(err)
	}
}

type testEvent struct {
	Time    time.Time // encodes as an RFC3339-format string
	Action  string
	Package string
	Test    string
	Elapsed float64 // seconds
	Output  string
}

type test struct {
	pkg  string
	test string
}

type outputBuffer struct {
	output []string
}

func (o *outputBuffer) push(s string) {
	o.output = append(o.output, s)
}

type testFailure struct {
	t test
	o outputBuffer
}

func writeTestOutput(in io.Reader, out io.Writer) error {
	testOutputBuffer := map[test]*outputBuffer{}
	var failures []testFailure
	d := json.NewDecoder(in)

	buf := bufio.NewWriter(out)
	defer buf.Flush()
	for {
		var e testEvent
		if err := d.Decode(&e); err != nil {
			break
		}

		switch e.Action {
		default:
			continue
		case "run":
			testOutputBuffer[test{pkg: e.Package, test: e.Test}] = new(outputBuffer)
		case "pass":
			// Don't hold onto text for passing
			delete(testOutputBuffer, test{pkg: e.Package, test: e.Test})
		case "fail":
			// fail may be for a package, which won't have an entry in
			// testOutputBuffer because packages don't have a "run"
			// action.
			t := test{pkg: e.Package, test: e.Test}
			if o, ok := testOutputBuffer[t]; ok {
				f := testFailure{t: t, o: *o}
				delete(testOutputBuffer, t)
				failures = append(failures, f)
			}
		case "output":
			buf.WriteString(e.Output)
			// output may be for a package, which won't have an entry
			// in testOutputBuffer because packages don't have a "run"
			// action.
			if o, ok := testOutputBuffer[test{pkg: e.Package, test: e.Test}]; ok {
				o.push(e.Output)
			}
		}
	}
	if len(failures) == 0 {
		return nil
	}
	buf.WriteString("\nTest failures:\n")
	for i, f := range failures {
		fmt.Fprintf(buf, "[%d] %s.%s\n", i+1, f.t.pkg, f.t.test)
		for _, s := range f.o.output {
			buf.WriteString(s)
		}
		if i < len(failures)-1 {
			buf.WriteByte('\n')
		}
	}
	return errTestFailure
}
