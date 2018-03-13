package logger

import (
	"testing"
)

var mlogExample1T = &MLogT{
	Description: `Struct for testing mlog structs.`,
	Receiver:    "TESTER",
	Verb:        "TESTING",
	Subject:     "MLOG",
	Details: []MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"NEIGHBORS", "BYTES_TRANSFERRED", "INT"},
	},
}

var mlogExample2T = &MLogT{
	Description: `Yet another struct for testing mlog structs.`,
	Receiver:    "TESTER",
	Verb:        "TESTING",
	Subject:     "MLOG",
	Details: []MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"NEIGHBORS", "BYTES_TRANSFERRED", "INT"},
	},
}

func BenchmarkSetDetailValues(b *testing.B) {
	vals := []interface{}{"hello", "kitty", 42}
	for i := 0; i < b.N; i++ {
		mlogExample1T.AssignDetails(vals...)
	}
}

func TestEnabling(t *testing.T) {
	SetMlogEnabled(false)
	if MlogEnabled() != false {
		t.Error("expected: false, got: true")
	}

	SetMlogEnabled(true)
	if MlogEnabled() != true {
		t.Error("expected: true, got: false")
	}
}

func TestRegisterAvailable(t *testing.T) {
	MLogRegisterAvailable("example1", []*MLogT{mlogExample1T})

	avail := GetMLogRegistryAvailable()
	if len(avail) != 1 {
		t.Errorf("expected: 1, got: %d", len(avail))
	}
	if _, ok := avail["example1"]; !ok {
		t.Error("expected key 'example1' not found")
	}
	if l := len(avail["example1"]); l != 1 {
		t.Errorf("expected: 1, got: %d", l)
	}

	MLogRegisterAvailable("example1", []*MLogT{mlogExample2T})
	avail = GetMLogRegistryAvailable()
	if len(avail) != 1 {
		t.Errorf("expected: 1, got: %d", len(avail))
	}
	if _, ok := avail["example1"]; !ok {
		t.Error("expected key 'example1' not found")
	}
	if l := len(avail["example1"]); l != 1 {
		t.Errorf("expected: 1, got: %d", l)
	}

	MLogRegisterAvailable("example1", []*MLogT{mlogExample1T, mlogExample2T})
	avail = GetMLogRegistryAvailable()
	if len(avail) != 1 {
		t.Errorf("expected: 1, got: %d", len(avail))
	}
	if _, ok := avail["example1"]; !ok {
		t.Error("expected key 'example1' not found")
	}
	if l := len(avail["example1"]); l != 2 {
		t.Errorf("expected: 2, got: %d", l)
	}

	MLogRegisterAvailable("example2", []*MLogT{mlogExample2T})
	avail = GetMLogRegistryAvailable()
	if len(avail) != 2 {
		t.Errorf("expected: 2, got: %d", len(avail))
	}
	if _, ok := avail["example1"]; !ok {
		t.Error("expected key 'example1' not found")
	}
	if l := len(avail["example1"]); l != 2 {
		t.Errorf("expected: 2, got: %d", l)
	}
	if _, ok := avail["example2"]; !ok {
		t.Error("expected key 'example1' not found")
	}
	if l := len(avail["example2"]); l != 1 {
		t.Errorf("expected: 1, got: %d", l)
	}
}

func TestRegisterFromContext(t *testing.T) {
	setupRegister()

	err := MLogRegisterComponentsFromContext("example1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	active := GetMLogRegistryActive()
	if l := len(active); l != 1 {
		t.Errorf("expected: 1, got: %d", l)
	}
	if _, ok := active["example1"]; !ok {
		t.Error("expected key 'example1' not found")
	}
}

func TestRegisterFromContextMany(t *testing.T) {
	setupRegister()

	err := MLogRegisterComponentsFromContext("example3,example2")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	active := GetMLogRegistryActive()
	if l := len(active); l != 2 {
		t.Errorf("expected: 2, got: %d", l)
	}
	if _, ok := active["example2"]; !ok {
		t.Error("expected key 'example2' not found")
	}
	if _, ok := active["example3"]; !ok {
		t.Error("expected key 'example3' not found")
	}
}

func TestRegisterFromNegativeContext(t *testing.T) {
	setupRegister()

	err := MLogRegisterComponentsFromContext("!example2")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	active := GetMLogRegistryActive()
	if l := len(active); l != 2 {
		t.Errorf("expected: 2, got: %d", l)
	}
	if _, ok := active["example1"]; !ok {
		t.Error("expected key 'example1' not found")
	}
	if _, ok := active["example3"]; !ok {
		t.Error("expected key 'example3' not found")
	}
}

func TestRegisterFromNegativeContextMany(t *testing.T) {
	setupRegister()

	err := MLogRegisterComponentsFromContext("!example2,example1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	active := GetMLogRegistryActive()
	if l := len(active); l != 1 {
		t.Errorf("expected: 1, got: %d", l)
	}
	if _, ok := active["example3"]; !ok {
		t.Error("expected key 'example3' not found")
	}
}

func TestRegisterFromWrongContext(t *testing.T) {
	setupRegister()

	err := MLogRegisterComponentsFromContext("wrongOne")
	if err == nil {
		t.Error("expected error, got nil")
	}

	err = MLogRegisterComponentsFromContext("example1,wrongOne")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func setupRegister() {
	MLogRegisterAvailable("example1", []*MLogT{mlogExample1T, mlogExample2T})
	MLogRegisterAvailable("example2", []*MLogT{mlogExample1T, mlogExample2T})
	MLogRegisterAvailable("example3", []*MLogT{mlogExample1T, mlogExample2T})

	// clean the global state
	MLogRegisterComponentsFromContext("!example1,example2,example3")
}
