// Copyright (c) 2015 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package test

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
)

// CopyFile copies a file
func CopyFile(t *testing.T, srcPath, dstPath string) {
	t.Helper()
	src, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.Create(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		t.Fatal(err)
	}
}

// TempDir creates a temporary directory under the default directory for temporary files (see
// os.TempDir) and returns the path of the new directory or fails the test trying.
func TempDir(t *testing.T, dirName string) string {
	t.Helper()
	tempDir, err := ioutil.TempDir("", dirName)
	if err != nil {
		t.Fatal(err)
	}
	return tempDir
}
