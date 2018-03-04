package common

import (
	"testing"
)

func TestClientSessionIdentityIsNotNilOnInit(t *testing.T) {
	if v := GetClientSessionIdentity(); v == nil {
		t.Errorf("got: %v, want: notnil instance", v)
	} else {
		t.Log(v)
	}

}

func TestSessionIDIsExpected(t *testing.T) {
	if v := SessionID; v == "" || len(v) != 4 {
		t.Errorf("got: %v, want: randomstring", v)
	}
}
