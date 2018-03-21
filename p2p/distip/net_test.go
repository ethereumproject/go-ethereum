package distip

import (
	"net"
	"testing"
)

func parseIP(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		panic("invalid " + s)
	}
	return ip
}

func checkContains(t *testing.T, fn func(net.IP) bool, inc, exc []string) {
	for _, s := range inc {
		if !fn(parseIP(s)) {
			t.Error("returned false for included address", s)
		}
	}
	for _, s := range exc {
		if fn(parseIP(s)) {
			t.Error("returned true for excluded address", s)
		}
	}
}

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

func TestIsLAN(t *testing.T) {
	checkContains(t, IsLAN,
		[]string{ // included
			"0.0.0.0",
			"0.2.0.8",
			"127.0.0.1",
			"10.0.1.1",
			"10.22.0.3",
			"172.31.252.251",
			"192.168.1.4",
			"fe80::f4a1:8eff:fec5:9d9d",
			"febf::ab32:2233",
			"fc00::4",
		},
		[]string{ // excluded
			"192.0.2.1",
			"1.0.0.0",
			"172.32.0.1",
			"fec0::2233",
		},
	)
}
