package accounts

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// These are commented because they take a pretty long time and I'm not that patient.
//
//func benchmarkCacheDBAccounts(n int, b *testing.B) {
//	// 20000 -> 20k
//	staticKeyFilesResourcePath := strconv.Itoa(n)
//	if strings.HasSuffix(staticKeyFilesResourcePath, "000") {
//		staticKeyFilesResourcePath = strings.TrimSuffix(staticKeyFilesResourcePath, "000")
//		staticKeyFilesResourcePath += "k"
//	}
//
//	staticKeyFilesResourcePath = filepath.Join("testdata", "benchmark_keystore"+staticKeyFilesResourcePath)
//
//	start := time.Now()
//	cache := newCacheDB(staticKeyFilesResourcePath)
//	elapsed := time.Since(start)
//
//	b.Logf("establishing cache for %v accs: %v", n, elapsed)
//
//	b.ResetTimer() // _benchmark_ timer, not setup timer.
//
//	for i := 0; i < b.N; i++ {
//		cache.accounts()
//	}
//	cache.close()
//}
//func BenchmarkCacheDBAccounts100(b *testing.B)    { benchmarkCacheDBAccounts(100, b) }
//func BenchmarkCacheDBAccounts500(b *testing.B)    { benchmarkCacheDBAccounts(500, b) }
//func BenchmarkCacheDBAccounts1000(b *testing.B)   { benchmarkCacheDBAccounts(1000, b) }
//func BenchmarkCacheDBAccounts5000(b *testing.B)   { benchmarkCacheDBAccounts(5000, b) }
//func BenchmarkCacheDBAccounts10000(b *testing.B)  { benchmarkCacheDBAccounts(10000, b) }
//func BenchmarkCacheDBAccounts20000(b *testing.B)  { benchmarkCacheDBAccounts(20000, b) }
//func BenchmarkCacheDBAccounts100000(b *testing.B) { benchmarkCacheDBAccounts(100000, b) }
//func BenchmarkCacheDBAccounts200000(b *testing.B) { benchmarkCacheDBAccounts(200000, b) }
//func BenchmarkCacheDBAccounts500000(b *testing.B) { benchmarkCacheDBAccounts(500000, b) }

// ac.add checks ac.all to see if given account already exists in cache,
// iff it doesn't, it adds account to byAddr map.
//
// No accounts added here are existing in cache, so *sort.Search* will iterate through all
// cached accounts _up to_ relevant alphabetizing. This is somewhat redundant to test cache.accounts(),
// except will test sort.Search instead of sort.Sort.
//
// Note that this _does not_ include ac.newCacheDB.
func benchmarkCacheDBAdd(n int, b *testing.B) {

	// 20000 -> 20k
	staticKeyFilesResourcePath := strconv.Itoa(n)
	if strings.HasSuffix(staticKeyFilesResourcePath, "000") {
		staticKeyFilesResourcePath = strings.TrimSuffix(staticKeyFilesResourcePath, "000")
		staticKeyFilesResourcePath += "k"
	}

	staticKeyFilesResourcePath = filepath.Join("testdata", "benchmark_keystore"+staticKeyFilesResourcePath)

	start := time.Now()
	cache := newCacheDB(staticKeyFilesResourcePath)
	elapsed := time.Since(start)

	b.Logf("establishing cache for %v accs: %v", n, elapsed)

	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		// cacheTestAccounts are constant established in cache_test.go
		for _, a := range cachetestAccounts {
			cache.add(a)
		}
	}
	cache.close()
}

func BenchmarkCacheDBAdd100(b *testing.B)    { benchmarkCacheDBAdd(100, b) }
func BenchmarkCacheDBAdd500(b *testing.B)    { benchmarkCacheDBAdd(500, b) }
func BenchmarkCacheDBAdd1000(b *testing.B)   { benchmarkCacheDBAdd(1000, b) }
func BenchmarkCacheDBAdd5000(b *testing.B)   { benchmarkCacheDBAdd(5000, b) }
func BenchmarkCacheDBAdd10000(b *testing.B)  { benchmarkCacheDBAdd(10000, b) }
func BenchmarkCacheDBAdd20000(b *testing.B)  { benchmarkCacheDBAdd(20000, b) }
func BenchmarkCacheDBAdd100000(b *testing.B) { benchmarkCacheDBAdd(100000, b) }
func BenchmarkCacheDBAdd200000(b *testing.B) { benchmarkCacheDBAdd(200000, b) }
func BenchmarkCacheDBAdd500000(b *testing.B) { benchmarkCacheDBAdd(500000, b) }

// ac.find checks ac.all to see if given account already exists in cache,
// iff it doesn't, it adds account to byAddr map.
//
// 3/4 added here are existing in cache; .find will iterate through ac.all
// cached, breaking only upon a find. There is no sort. method here.
//
// Note that this _does not_ include ac.newCacheDB.
func benchmarkCacheDBFind(n int, onlyExisting bool, b *testing.B) {

	// 20000 -> 20k
	staticKeyFilesResourcePath := strconv.Itoa(n)
	if strings.HasSuffix(staticKeyFilesResourcePath, "000") {
		staticKeyFilesResourcePath = strings.TrimSuffix(staticKeyFilesResourcePath, "000")
		staticKeyFilesResourcePath += "k"
	}

	staticKeyFilesResourcePath = filepath.Join("testdata", "benchmark_keystore"+staticKeyFilesResourcePath)

	start := time.Now()
	cache := newCacheDB(staticKeyFilesResourcePath)
	elapsed := time.Since(start)
	b.Logf("establishing cache for %v accs: %v", n, elapsed)

	accs := cache.accounts()

	// Set up 1 DNE and 3 existing accs.
	// Using the last accs because they should take the longest to iterate to.
	var findAccounts []Account

	if !onlyExisting {
		findAccounts = append(cachetestAccounts[(len(cachetestAccounts)-1):], accs[(len(accs)-3):]...)
		if len(findAccounts) != 4 {
			b.Fatalf("wrong number find accs: got: %v, want: 4", len(findAccounts))
		}
	} else {
		findAccounts = accs[(len(accs) - 4):]
	}
	accs = nil // clear mem?

	b.ResetTimer() // _benchmark_ timer, not setup timer.

	for i := 0; i < b.N; i++ {
		for _, a := range findAccounts {
			cache.find(a)
		}
	}
	cache.close()
}

func BenchmarkCacheDBFind100(b *testing.B)                { benchmarkCacheDBFind(100, false, b) }
func BenchmarkCacheDBFind500(b *testing.B)                { benchmarkCacheDBFind(500, false, b) }
func BenchmarkCacheDBFind1000(b *testing.B)               { benchmarkCacheDBFind(1000, false, b) }
func BenchmarkCacheDBFind5000(b *testing.B)               { benchmarkCacheDBFind(5000, false, b) }
func BenchmarkCacheDBFind10000(b *testing.B)              { benchmarkCacheDBFind(10000, false, b) }
func BenchmarkCacheDBFind20000(b *testing.B)              { benchmarkCacheDBFind(20000, false, b) }
func BenchmarkCacheDBFind100000(b *testing.B)             { benchmarkCacheDBFind(100000, false, b) }
func BenchmarkCacheDBFind200000(b *testing.B)             { benchmarkCacheDBFind(200000, false, b) }
func BenchmarkCacheDBFind500000(b *testing.B)             { benchmarkCacheDBFind(500000, false, b) }
func BenchmarkCacheDBFind100OnlyExisting(b *testing.B)    { benchmarkCacheDBFind(100, true, b) }
func BenchmarkCacheDBFind500OnlyExisting(b *testing.B)    { benchmarkCacheDBFind(500, true, b) }
func BenchmarkCacheDBFind1000OnlyExisting(b *testing.B)   { benchmarkCacheDBFind(1000, true, b) }
func BenchmarkCacheDBFind5000OnlyExisting(b *testing.B)   { benchmarkCacheDBFind(5000, true, b) }
func BenchmarkCacheDBFind10000OnlyExisting(b *testing.B)  { benchmarkCacheDBFind(10000, true, b) }
func BenchmarkCacheDBFind20000OnlyExisting(b *testing.B)  { benchmarkCacheDBFind(20000, true, b) }
func BenchmarkCacheDBFind100000OnlyExisting(b *testing.B) { benchmarkCacheDBFind(100000, true, b) }
func BenchmarkCacheDBFind200000OnlyExisting(b *testing.B) { benchmarkCacheDBFind(200000, true, b) }
func BenchmarkCacheDBFind500000OnlyExisting(b *testing.B) { benchmarkCacheDBFind(500000, true, b) }
