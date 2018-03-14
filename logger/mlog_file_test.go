package logger

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
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

func TestAssignDetails(t *testing.T) {
	addr := "sampleAddress"
	id := "sampleId"
	bytes := 123
	mlogExample1T.AssignDetails(addr, id, bytes)

	if mlogExample1T.Details[0].Value != addr {
		t.Errorf("expected: '%s', got: '%s'", addr, mlogExample1T.Details[0].Value)
	}
	if mlogExample1T.Details[1].Value != id {
		t.Errorf("expected: '%s', got: '%s'", id, mlogExample1T.Details[1].Value)
	}
	if mlogExample1T.Details[2].Value != bytes {
		t.Errorf("expected: %d, got: %d", bytes, mlogExample1T.Details[2].Value)
	}

	// assign again...
	addr2 := "anotherAddress"
	id2 := "anotherId"
	bytes2 := 321
	mlogExample1T.AssignDetails(addr2, id2, bytes2)

	if mlogExample1T.Details[0].Value != addr2 {
		t.Errorf("expected: '%s', got: '%s'", addr2, mlogExample1T.Details[0].Value)
	}
	if mlogExample1T.Details[1].Value != id2 {
		t.Errorf("expected: '%s', got: '%s'", id2, mlogExample1T.Details[1].Value)
	}
	if mlogExample1T.Details[2].Value != bytes2 {
		t.Errorf("expected: %d, got: %d", bytes2, mlogExample1T.Details[2].Value)
	}
}

func TestSend(t *testing.T) {
	testLogger := MLogRegisterAvailable("example1", []*MLogT{mlogExample1T, mlogExample2T})
	MLogRegisterActive("example1")

	addr := "sampleAddress"
	id := "sampleId"
	bytes := 123
	mlogExample1T.AssignDetails(addr, id, bytes)

	formats := []string{"plain", "kv", "json"}
	for _, format := range formats {
		SetMLogFormatFromString(format)
		mlogExample1T.Send(testLogger)
		// TODO: add some assertions!
	}

}

func TestFormats(t *testing.T) {
	formats := []struct {
		name  string
		valid bool
	}{
		{"plain", true},
		{"kv", true},
		{"json", true},
		{"invalid", false},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			err := SetMLogFormatFromString(format.name)
			if format.valid {
				if err != nil {
					t.Error("unexpected error")
				}
				if fmt := GetMLogFormat().String(); fmt != format.name {
					t.Errorf("expected: '%s', got: '%s'", format.name, fmt)
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				}
			}
		})
	}
}

func TestInit(t *testing.T) {
	now := time.Now()

	dir, err := ioutil.TempDir("", "mlog_test")
	if err != nil {
		t.Errorf("cannot create temp dir: %v", err)
	}
	defer os.RemoveAll(dir) // clean up

	SetMLogDir(dir)
	_, filename, err := CreateMLogFile(now)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(filename) == 0 {
		t.Errorf("expected non-empty filename")
	}
	if !strings.HasPrefix(filename, dir) {
		t.Errorf("file created in wrong directory, expected: %s, got: %s", dir, filename)
	}
}

func TestDocumentation(t *testing.T) {
	// reset structure to force valid type information and give some esoteric names
	mlogExample1T.Details = []MLogDetailT{
		{"FROM616", "UDP_ADDRESS911", "STRING"},
		{"FROM666", "RANDOMIZED_ID", "STRING"},
		{"NEIGHBORS", "BYTES_TRANSFERRED", "INT"},
	}
	logger := MLogRegisterAvailable("example1", []*MLogT{mlogExample1T})

	docs := mlogExample1T.FormatDocumentation(logger)
	if len(docs) == 0 {
		t.Error("documentation is empty!")
	}
	if !strings.Contains(docs, mlogExample1T.Subject) {
		t.Error("missing information about subject")
	}
	if !strings.Contains(docs, mlogExample1T.Receiver) {
		t.Error("missing information about receiver")
	}
	if !strings.Contains(docs, mlogExample1T.Verb) {
		t.Error("missing information about verb")
	}
	ldocs := strings.ToLower(docs)
	for _, detail := range mlogExample1T.Details {
		if !strings.Contains(ldocs, strings.ToLower(detail.Owner)) {
			t.Error("missing information about detail owner")
		}
		if !strings.Contains(ldocs, strings.ToLower(detail.Key)) {
			t.Error("missing information about detail key")
		}
	}
}
