package common

import (
	"testing"
)

func TestClientSessionIdentityIsNotNilOnInit(t *testing.T) {
	if v := GetClientSessionIdentity(); v == nil {
		t.Error("Expected not-nil client session identity")
	} else {
		t.Log(v)
	}

}

func TestSessionIDIsExpected(t *testing.T) {
	if v := SessionID; v == "" || len(v) != 4 {
		t.Errorf("got: %v, want: randomstring", v)
	}
}

func TestSessionVersionInformation(t *testing.T) {
	v := GetClientSessionIdentity()
	if v == nil {
		t.Error("Expected not-nil client session identity")
	}

	if v.Version != "unknown" {
		t.Errorf("Version: expected 'unknown', got '%v'", v.Version)
	}

	version := "9.8.7-test"
	SetClientVersion(version)
	v = GetClientSessionIdentity()

	if v.Version != version {
		t.Errorf("Version: expected '%s', got '%s'", version, v.Version)
	}
}
