// Copyright 2014 The go-ethereum Authors
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

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkVmAckermann32Tests(b *testing.B) {
	fn := filepath.Join(vmTestDir, "vmPerformanceTest.json")
	if err := BenchVmTest(fn, bconf{"ackermann32", os.Getenv("JITFORCE") == "true", os.Getenv("JITVM") == "true"}, b); err != nil {
		b.Error(err)
	}
}

func BenchmarkVmFibonacci16Tests(b *testing.B) {
	fn := filepath.Join(vmTestDir, "vmPerformanceTest.json")
	if err := BenchVmTest(fn, bconf{"fibonacci16", os.Getenv("JITFORCE") == "true", os.Getenv("JITVM") == "true"}, b); err != nil {
		b.Error(err)
	}
}

func BenchmarkVMTests(b *testing.B) {
	err := filepath.Walk(vmTestDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			b.Logf("walk err=%v", err)
			return nil
		}
		if info.IsDir() {
			b.Logf("%s: SKIP (DIR)", p)
			return nil
		}
		name := filepath.Base(p)
		ext := filepath.Ext(name)
		name = strings.Replace(name, ext, "", -1)
		if err := BenchVmTest(p, bconf{name, os.Getenv("JITFORCE") == "true", os.Getenv("JITVM") == "true"}, b); err != nil {
			b.Error(err)
		}

		return nil
	})
	if err != nil {
		panic(err.Error())
	}
}

func TestVMTests(t *testing.T) {
	err := filepath.Walk(vmTestDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			t.Logf("walk err=%v", err)
			return nil
		}
		if info.IsDir() {
			t.Logf("%s: SKIP (DIR)", p)
			return nil
		}
		if err := RunVmTest(p, VmSkipTests); err != nil {
			t.Error(err)
		} else {
			t.Logf("%s: PASS", p)
		}
		return nil
	})
	if err != nil {
		panic(err.Error())
	}
}
