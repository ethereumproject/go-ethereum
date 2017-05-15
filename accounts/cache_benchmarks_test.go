package accounts

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func benchmarkCacheAccounts(n int, b *testing.B) {

	start := time.Now()
	dir, err := ioutil.TempDir("", "eth-acctcache-test")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	am, err := NewManager(dir, veryLightScryptN, veryLightScryptP)
	if err != nil {
		b.Fatal(err)
	}

	for len(am.Accounts()) < n {
		if e := createTestAccount(am, dir); e != nil {
			b.Fatalf("error setting up acount: %v", e)
		}
	}
	elapsed := time.Since(start)
	b.Logf("setting up %v accounts took %v", n, elapsed)

	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		cache := newAddrCache(dir)
		as := cache.accounts()
		if len(as) != n {
			b.Errorf("missing or extra accounts in cache: got: %v, want: %v", len(as), n)
		}
	}
}

func BenchmarkCacheAccounts100(b *testing.B)   { benchmarkCacheAccounts(100, b) }
func BenchmarkCacheAccounts500(b *testing.B)   { benchmarkCacheAccounts(500, b) }
func BenchmarkCacheAccounts1000(b *testing.B)  { benchmarkCacheAccounts(1000, b) }
func BenchmarkCacheAccounts5000(b *testing.B)  { benchmarkCacheAccounts(5000, b) }
func BenchmarkCacheAccounts10000(b *testing.B) { benchmarkCacheAccounts(10000, b) }
func BenchmarkCacheAccounts20000(b *testing.B) { benchmarkCacheAccounts(20000, b) }

// ac.add checks ac.all to see if given account already exists in cache,
// iff it doesn't, it adds account to byAddr map.
//
// No accounts added here are existing in cache, so *sort.Search* will iterate through all
// cached accounts _up to_ relevant alphabetizing. This is somewhat redundant to test cache.accounts(),
// except will test sort.Search instead of sort.Sort.
//
// Note that this _does not_ include ac.newAddrCache.
func benchmarkCacheAdd(n int, b *testing.B) {

	start := time.Now()
	dir, err := ioutil.TempDir("", "eth-acctcache-test")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	am, err := NewManager(dir, veryLightScryptN, veryLightScryptP)
	if err != nil {
		b.Fatal(err)
	}

	for len(am.Accounts()) < n {
		if e := createTestAccount(am, dir); e != nil {
			b.Fatalf("error setting up acount: %v", e)
		}
	}
	elapsed := time.Since(start)
	b.Logf("setting up %v accounts took %v", n, elapsed)

	cache := newAddrCache(dir)
	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		// cacheTestAccounts are constant established in cache_test.go
		for _, a := range cachetestAccounts {
			cache.add(a)
		}
	}
}

func BenchmarkCacheAdd100(b *testing.B)   { benchmarkCacheAdd(100, b) }
func BenchmarkCacheAdd500(b *testing.B)   { benchmarkCacheAdd(500, b) }
func BenchmarkCacheAdd1000(b *testing.B)  { benchmarkCacheAdd(1000, b) }
func BenchmarkCacheAdd5000(b *testing.B)  { benchmarkCacheAdd(5000, b) }
func BenchmarkCacheAdd10000(b *testing.B) { benchmarkCacheAdd(10000, b) }
func BenchmarkCacheAdd20000(b *testing.B) { benchmarkCacheAdd(20000, b) }

// ac.find checks ac.all to see if given account already exists in cache,
// iff it doesn't, it adds account to byAddr map.
//
// 3/4 added here are existing in cache; .find will iterate through ac.all
// cached, breaking only upon a find. There is no sort. method here.
//
// Note that this _does not_ include ac.newAddrCache.
func benchmarkCacheFind(n int, b *testing.B) {

	start := time.Now()
	dir, err := ioutil.TempDir("", "eth-acctcache-test")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	am, err := NewManager(dir, veryLightScryptN, veryLightScryptP)
	if err != nil {
		b.Fatal(err)
	}

	for len(am.Accounts()) < n {
		if e := createTestAccount(am, dir); e != nil {
			b.Fatalf("error setting up acount: %v", e)
		}
	}
	elapsed := time.Since(start)
	b.Logf("setting up %v accounts took %v", n, elapsed)

	cache := newAddrCache(dir)

	// Set up 1 DNE and 3 existing accounts.
	// Using the last accounts because they should take the longest to iterate to.
	findAccounts := append(cachetestAccounts[(len(cachetestAccounts)-1):], am.Accounts()[(len(am.Accounts()) - 3):]...)
	if len(findAccounts) != 4 {
		b.Fatalf("wrong number find accounts: got: %v, want: 4", len(findAccounts))
	}

	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		for _, a := range findAccounts {
			cache.find(a)
		}
	}
}

func BenchmarkCacheFind100(b *testing.B)   { benchmarkCacheFind(100, b) }
func BenchmarkCacheFind500(b *testing.B)   { benchmarkCacheFind(500, b) }
func BenchmarkCacheFind1000(b *testing.B)  { benchmarkCacheFind(1000, b) }
func BenchmarkCacheFind5000(b *testing.B)  { benchmarkCacheFind(5000, b) }
func BenchmarkCacheFind10000(b *testing.B) { benchmarkCacheFind(10000, b) }
func BenchmarkCacheFind20000(b *testing.B) { benchmarkCacheFind(20000, b) }
