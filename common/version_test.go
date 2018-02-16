package common

import (
	"testing"
	"strings"
)

func TestClientSessionIDIsNotNilOnInit(t *testing.T) {
	if v := GetClientSessionIdentity(); v == nil {
		t.Errorf("got: %v, want: notnil instance", v)
	} else {
		t.Log(v)
	}

}

func TestSessionIDIsExpected(t *testing.T) {
	if v := SessionID; v == "" || len(v) != 8 {
		t.Errorf("got: %v, want: randomstring", v)
	}
}

func TestVCRevisionIsExpected(t *testing.T) {
	if v := VCRevision; v == "" || len(v) != 8 {
		t.Errorf("got: %v, want: git hash abbrev.", v)
	}
}

func TestMustSourceBuildVersionFormatted(t *testing.T) {
	if v := MustSourceBuildVersionFormatted(); v == "" || !strings.HasPrefix(v, "source-") {
		t.Errorf("got: %v, want: source-*", v)
	} else {
		t.Log(v)
	}
}
