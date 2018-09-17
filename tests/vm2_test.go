package tests

// This file uses v2 of the JSON tests schema, where 'logs' field is a single string (rlpHash) of the statedb logs, instead of an array of type vm.Logs.

import (
	"path/filepath"
	"testing"
)

// VmTests2 describes the "new" JSON test schema for VM tests.
// See vm_test_util.go help fns for correlated helpers, where those fns are suffixed with 2, eg runVmTest2
type VmTest2 struct {
	Callcreates interface{}
	//Env         map[string]string
	Env           VmEnv
	Exec          map[string]string
	Transaction   map[string]string
	Logs          string
	Gas           string
	Out           string
	Post          map[string]Account
	Pre           map[string]Account
	PostStateRoot string
}

func TestSputnikVMTests(t *testing.T) {
	fns, _ := filepath.Glob(filepath.Join(vmTestDir, "SputnikVM", "vmBitwiseLogicOperation", "*"))
	for _, fn := range fns {
		if err := RunVmTest2(fn, VmSkipTests); err != nil {
			t.Error(err)
		}
	}
}
