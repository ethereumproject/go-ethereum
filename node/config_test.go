// Copyright 2015 The go-ethereum Authors
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

package node

import (
	"bytes"
	"github.com/eth-classic/go-ethereum/crypto"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
)

// Tests that datadirs can be successfully created, be them manually configured
// ones or automatically generated temporary ones.
func TestDatadirCreation(t *testing.T) {
	// Create a temporary data dir and check that it can be used by a node
	memFS := &fs{afero.NewMemMapFs()}

	dir := filepath.Join("path", "to", "datadir")
	err := memFS.MkdirAll(dir, os.ModePerm) // just in mem... heh
	if err != nil {
		t.Fatalf("failed to create manual data dir: %v", err)
	}

	if _, err := New(&Config{DataDir: dir, fs: memFS}); err != nil {
		t.Fatalf("failed to create stack with existing datadir: %v", err)
	}
	// Generate a long non-existing datadir path and check that it gets created by a node
	dir = filepath.Join(dir, "a", "b", "c", "d", "e", "f")
	if _, err := New(&Config{DataDir: dir, fs: memFS}); err != nil {
		t.Fatalf("failed to create stack with creatable datadir: %v", err)
	}
	if _, err := memFS.Stat(dir); err != nil {
		t.Fatalf("freshly created datadir not accessible: %v", err)
	}
	// Verify that an impossible datadir fails creation
	// Impossible here means that it would attempt to be created on top of an already existing file
	osFS := &fs{afero.NewOsFs()} // work around https://github.com/spf13/afero/issues/164
	file, err := afero.TempFile(osFS, "", "")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	defer osFS.Remove(file.Name())

	dir = filepath.Join(file.Name(), "invalid/path")
	if n, err := New(&Config{DataDir: dir, fs: osFS}); err == nil {
		t.Fatalf("protocol stack created with an invalid datadir: %s / %s\nabove file=%s", dir, n.DataDir(), file.Name())
	}
}

// Tests that IPC paths are correctly resolved to valid endpoints of different
// platforms.
func TestIPCPathResolution(t *testing.T) {

	var tests = []struct {
		DataDir  string
		IPCPath  string
		Windows  bool
		Endpoint string
	}{
		{"", "", false, ""},
		{"data", "", false, ""},
		{"", "geth.ipc", false, filepath.Join(os.TempDir(), "geth.ipc")}, // test 2
		{"data", "geth.ipc", false, "data/geth.ipc"},
		{"data", "./geth.ipc", false, "./geth.ipc"},
		{"data", "/geth.ipc", false, "/geth.ipc"},
		{"", "", true, ``},
		{"data", "", true, ``},
		{"", "geth.ipc", true, `\\.\pipe\geth.ipc`},
		{"data", "geth.ipc", true, `\\.\pipe\geth.ipc`},
		{"data", `\\.\pipe\geth.ipc`, true, `\\.\pipe\geth.ipc`},
	}
	for i, test := range tests {
		// Only run when platform/test match
		if (runtime.GOOS == "windows") == test.Windows {
			if gotEndPoint := (&Config{
				DataDir: test.DataDir,
				IPCPath: test.IPCPath,
				fs:      &fs{afero.NewMemMapFs()},
			}).IPCEndpoint(); gotEndPoint != test.Endpoint {
				t.Errorf("test %d: IPC endpoint mismatch: got: %s, want: %s", i, gotEndPoint, test.Endpoint)
			}
		}
	}
}

// Tests that node keys can be correctly created, persisted, loaded and/or made
// ephemeral.
func TestNodeKeyPersistency(t *testing.T) {
	dir := os.TempDir()
	memFS := &fs{afero.NewMemMapFs()}

	if _, err := memFS.Stat(filepath.Join(dir, datadirPrivateKey)); err == nil {
		t.Fatalf("non-created node key already exists")
	}
	// Configure a node with a preset key and ensure it's not persisted
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate one-shot node key: %v", err)
	}
	if _, err := New(&Config{DataDir: dir, PrivateKey: key, fs: memFS}); err != nil {
		t.Fatalf("failed to create empty stack: %v", err)
	}
	if _, err := memFS.Stat(filepath.Join(dir, datadirPrivateKey)); err == nil {
		t.Fatalf("one-shot node key persisted to data directory")
	}
	// Configure a node with no preset key and ensure it is persisted this time
	if _, err := New(&Config{DataDir: dir, fs: memFS}); err != nil {
		t.Fatalf("failed to create newly keyed stack: %v", err)
	}
	if _, err := memFS.Stat(filepath.Join(dir, datadirPrivateKey)); err != nil {
		t.Fatalf("node key not persisted to data directory: %v", err)
	}
	blob1, err := afero.ReadFile(memFS, filepath.Join(dir, datadirPrivateKey))
	if err != nil {
		t.Fatalf("failed to read freshly persisted node key: %v", err)
	}
	f, err := memFS.Open(filepath.Join(dir, datadirPrivateKey))
	if err != nil {
		t.Fatalf("failed to open private key file: %v", err)
	}
	key, err = crypto.LoadECDSA(f)
	if err != nil {
		t.Fatalf("failed to load freshly persisted node key: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close node key: %v", err)
	}
	// Configure a new node and ensure the previously persisted key is loaded
	if _, err := New(&Config{DataDir: dir, fs: memFS}); err != nil {
		t.Fatalf("failed to create previously keyed stack: %v", err)
	}
	blob2, err := afero.ReadFile(memFS, filepath.Join(dir, datadirPrivateKey))
	if err != nil {
		t.Fatalf("failed to read previously persisted node key: %v", err)
	}
	if bytes.Compare(blob1, blob2) != 0 {
		t.Fatalf("persisted node key mismatch: have %x, want %x", blob2, blob1)
	}
	// Configure ephemeral node and ensure no key is dumped locally
	if _, err := New(&Config{DataDir: "", fs: memFS}); err != nil {
		t.Fatalf("failed to create ephemeral stack: %v", err)
	}
	if _, err := memFS.Stat(filepath.Join(".", datadirPrivateKey)); err == nil {
		t.Fatalf("ephemeral node key persisted to disk")
	}
}
