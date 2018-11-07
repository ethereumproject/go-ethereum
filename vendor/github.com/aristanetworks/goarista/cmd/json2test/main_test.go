// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestWriteTestOutput(t *testing.T) {
	input, err := os.Open("testdata/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := writeTestOutput(input, &out); err != errTestFailure {
		t.Error("expected test failure")
	}

	gold, err := os.Open("testdata/gold.txt")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := ioutil.ReadAll(gold)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out.Bytes(), expected) {
		t.Error("output does not match gold.txt")
		fmt.Println("Expected:")
		fmt.Println(string(expected))
		fmt.Println("Got:")
		fmt.Println(out.String())
	}
}
