package accounts

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// These are commented because they take a pretty long time and I'm not that patient.
//
//func benchmarkCacheAccounts(n int, b *testing.B) {
//	// 20000 -> 20k
//	staticKeyFilesResourcePath := strconv.Itoa(n)
//	if strings.HasSuffix(staticKeyFilesResourcePath, "000") {
//		staticKeyFilesResourcePath = strings.TrimSuffix(staticKeyFilesResourcePath, "000")
//		staticKeyFilesResourcePath += "k"
//	}
//
//	staticKeyFilesResourcePath, _ = filepath.Abs(filepath.Join("testdata", "benchmark_keystore"+staticKeyFilesResourcePath))
//
//	start := time.Now()
//	cache := newAddrCache(staticKeyFilesResourcePath)
//	cache.watcher.running = true
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
//func BenchmarkCacheAccounts100(b *testing.B)   { benchmarkCacheAccounts(100, b) }
//func BenchmarkCacheAccounts500(b *testing.B)   { benchmarkCacheAccounts(500, b) }
//func BenchmarkCacheAccounts1000(b *testing.B)  { benchmarkCacheAccounts(1000, b) }
//func BenchmarkCacheAccounts5000(b *testing.B)  { benchmarkCacheAccounts(5000, b) }
//func BenchmarkCacheAccounts10000(b *testing.B) { benchmarkCacheAccounts(10000, b) }
//func BenchmarkCacheAccounts20000(b *testing.B) { benchmarkCacheAccounts(20000, b) }
//func BenchmarkCacheAccounts100000(b *testing.B) { benchmarkCacheAccounts(100000, b) }
//func BenchmarkCacheAccounts200000(b *testing.B) { benchmarkCacheAccounts(200000, b) }
//func BenchmarkCacheAccounts500000(b *testing.B) { benchmarkCacheAccounts(500000, b) }

// ac.add checks ac.all to see if given account already exists in cache,
// iff it doesn't, it adds account to byAddr map.
//
// No accounts added here are existing in cache, so *sort.Search* will iterate through all
// cached accounts _up to_ relevant alphabetizing. This is somewhat redundant to test cache.accounts(),
// except will test sort.Search instead of sort.Sort.
//
// Note that this _does not_ include ac.newAddrCache.
func benchmarkCacheAdd(n int, b *testing.B) {

	// 20000 -> 20k
	staticKeyFilesResourcePath := strconv.Itoa(n)
	if strings.HasSuffix(staticKeyFilesResourcePath, "000") {
		staticKeyFilesResourcePath = strings.TrimSuffix(staticKeyFilesResourcePath, "000")
		staticKeyFilesResourcePath += "k"
	}

	staticKeyFilesResourcePath, _ = filepath.Abs(filepath.Join("testdata", "benchmark_keystore"+staticKeyFilesResourcePath))

	start := time.Now()
	cache := newAddrCache(staticKeyFilesResourcePath)
	cache.watcher.running = true
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

func BenchmarkCacheAdd100(b *testing.B)    { benchmarkCacheAdd(100, b) }
func BenchmarkCacheAdd500(b *testing.B)    { benchmarkCacheAdd(500, b) }
func BenchmarkCacheAdd1000(b *testing.B)   { benchmarkCacheAdd(1000, b) }
func BenchmarkCacheAdd5000(b *testing.B)   { benchmarkCacheAdd(5000, b) }
func BenchmarkCacheAdd10000(b *testing.B)  { benchmarkCacheAdd(10000, b) }
func BenchmarkCacheAdd20000(b *testing.B)  { benchmarkCacheAdd(20000, b) }
func BenchmarkCacheAdd100000(b *testing.B) { benchmarkCacheAdd(100000, b) }
func BenchmarkCacheAdd200000(b *testing.B) { benchmarkCacheAdd(200000, b) }
func BenchmarkCacheAdd500000(b *testing.B) { benchmarkCacheAdd(500000, b) }

// ac.find checks ac.all to see if given account already exists in cache,
// iff it doesn't, it adds account to byAddr map.
//
// 3/4 added here are existing in cache; .find will iterate through ac.all
// cached, breaking only upon a find. There is no sort. method here.
//
// Note that this _does not_ include ac.newAddrCache.
func benchmarkCacheFind(n int, onlyExisting bool, b *testing.B) {

	// 20000 -> 20k
	staticKeyFilesResourcePath := strconv.Itoa(n)
	if strings.HasSuffix(staticKeyFilesResourcePath, "000") {
		staticKeyFilesResourcePath = strings.TrimSuffix(staticKeyFilesResourcePath, "000")
		staticKeyFilesResourcePath += "k"
	}

	staticKeyFilesResourcePath, _ = filepath.Abs(filepath.Join("testdata", "benchmark_keystore"+staticKeyFilesResourcePath))

	if _, err := os.Stat(staticKeyFilesResourcePath); err != nil {
		b.Fatal(err)
	}

	start := time.Now()
	cache := newAddrCache(staticKeyFilesResourcePath)
	elapsed := time.Since(start)
	b.Logf("establishing cache for %v accs: %v", n, elapsed)

	accs := cache.accounts()
	cache.watcher.running = true
	if len(accs) == 0 {
		b.Fatalf("no accounts: keystore: %v", staticKeyFilesResourcePath)
	}

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
			cache.muLock()
			cache.find(a)
			cache.muUnlock()
		}
	}
	cache.close()
}

func BenchmarkCacheFind100(b *testing.B)                { benchmarkCacheFind(100, false, b) }
func BenchmarkCacheFind500(b *testing.B)                { benchmarkCacheFind(500, false, b) }
func BenchmarkCacheFind1000(b *testing.B)               { benchmarkCacheFind(1000, false, b) }
func BenchmarkCacheFind5000(b *testing.B)               { benchmarkCacheFind(5000, false, b) }
func BenchmarkCacheFind10000(b *testing.B)              { benchmarkCacheFind(10000, false, b) }
func BenchmarkCacheFind20000(b *testing.B)              { benchmarkCacheFind(20000, false, b) }
func BenchmarkCacheFind100000(b *testing.B)             { benchmarkCacheFind(100000, false, b) }
func BenchmarkCacheFind200000(b *testing.B)             { benchmarkCacheFind(200000, false, b) }
func BenchmarkCacheFind500000(b *testing.B)             { benchmarkCacheFind(500000, false, b) }
func BenchmarkCacheFind100OnlyExisting(b *testing.B)    { benchmarkCacheFind(100, true, b) }
func BenchmarkCacheFind500OnlyExisting(b *testing.B)    { benchmarkCacheFind(500, true, b) }
func BenchmarkCacheFind1000OnlyExisting(b *testing.B)   { benchmarkCacheFind(1000, true, b) }
func BenchmarkCacheFind5000OnlyExisting(b *testing.B)   { benchmarkCacheFind(5000, true, b) }
func BenchmarkCacheFind10000OnlyExisting(b *testing.B)  { benchmarkCacheFind(10000, true, b) }
func BenchmarkCacheFind20000OnlyExisting(b *testing.B)  { benchmarkCacheFind(20000, true, b) }
func BenchmarkCacheFind100000OnlyExisting(b *testing.B) { benchmarkCacheFind(100000, true, b) }
func BenchmarkCacheFind200000OnlyExisting(b *testing.B) { benchmarkCacheFind(200000, true, b) }
func BenchmarkCacheFind500000OnlyExisting(b *testing.B) { benchmarkCacheFind(500000, true, b) }

// https://gist.github.com/m4ng0squ4sh/92462b38df26839a3ca324697c8cba04
// CopyFile copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file. The file mode will be copied from the source and
// the copied data is synced/flushed to stable storage.
func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist.
// Symlinks are ignored and skipped.
func CopyDir(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if skipKeyFile(entry) {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}
