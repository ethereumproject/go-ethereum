package distip

import (
	"testing"
	"net"
)

func makeTestDistinctNetSet() *DistinctNetSet {
	return &DistinctNetSet{
		Subnet: 24,
		Limit:  1,
	}
}

func TestDistinctNetSet(t *testing.T) {
	set := makeTestDistinctNetSet()
	testip := net.ParseIP("24.207.212.9")

	key := set.key(testip)
	t.Logf("key: %v", key)

	if ok := set.Add(testip); !ok {
		t.Errorf("got: %v, want: %v", ok, true)
	}
	if contains := set.Contains(testip); !contains {
		t.Errorf("got: %v, want: %v", contains, true)
	}
	if len := set.Len(); len != 1 {
		t.Errorf("got: %v, want: %v", len, 1)
	}

	set.Remove(testip)
	if contains := set.Contains(testip); contains {
		t.Errorf("got: %v, want: %v", contains, false)
	}
	if len := set.Len(); len != 0 {
		t.Errorf("got: %v, want: %v", len, 0)
	}
}
