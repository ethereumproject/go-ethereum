// Copyright (c) 2017 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package path

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aristanetworks/goarista/key"
	"github.com/aristanetworks/goarista/test"
)

func accumulator(counter map[int]int) VisitorFunc {
	return func(val interface{}) error {
		counter[val.(int)]++
		return nil
	}
}

func TestMapSet(t *testing.T) {
	m := Map{}
	a := m.Set(key.Path{key.New("foo")}, 0)
	b := m.Set(key.Path{key.New("foo")}, 1)
	if !a || b {
		t.Fatal("Map.Set not working properly")
	}
}

func TestMapVisit(t *testing.T) {
	m := Map{}
	m.Set(key.Path{key.New("foo"), key.New("bar"), key.New("baz")}, 1)
	m.Set(key.Path{Wildcard, key.New("bar"), key.New("baz")}, 2)
	m.Set(key.Path{Wildcard, Wildcard, key.New("baz")}, 3)
	m.Set(key.Path{Wildcard, Wildcard, Wildcard}, 4)
	m.Set(key.Path{key.New("foo"), Wildcard, Wildcard}, 5)
	m.Set(key.Path{key.New("foo"), key.New("bar"), Wildcard}, 6)
	m.Set(key.Path{key.New("foo"), Wildcard, key.New("baz")}, 7)
	m.Set(key.Path{Wildcard, key.New("bar"), Wildcard}, 8)

	m.Set(key.Path{}, 10)

	m.Set(key.Path{Wildcard}, 20)
	m.Set(key.Path{key.New("foo")}, 21)

	m.Set(key.Path{key.New("zap"), key.New("zip")}, 30)
	m.Set(key.Path{key.New("zap"), key.New("zip")}, 31)

	m.Set(key.Path{key.New("zip"), Wildcard}, 40)
	m.Set(key.Path{key.New("zip"), Wildcard}, 41)

	testCases := []struct {
		path     key.Path
		expected map[int]int
	}{{
		path:     key.Path{key.New("foo"), key.New("bar"), key.New("baz")},
		expected: map[int]int{1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1, 7: 1, 8: 1},
	}, {
		path:     key.Path{key.New("qux"), key.New("bar"), key.New("baz")},
		expected: map[int]int{2: 1, 3: 1, 4: 1, 8: 1},
	}, {
		path:     key.Path{key.New("foo"), key.New("qux"), key.New("baz")},
		expected: map[int]int{3: 1, 4: 1, 5: 1, 7: 1},
	}, {
		path:     key.Path{key.New("foo"), key.New("bar"), key.New("qux")},
		expected: map[int]int{4: 1, 5: 1, 6: 1, 8: 1},
	}, {
		path:     key.Path{},
		expected: map[int]int{10: 1},
	}, {
		path:     key.Path{key.New("foo")},
		expected: map[int]int{20: 1, 21: 1},
	}, {
		path:     key.Path{key.New("foo"), key.New("bar")},
		expected: map[int]int{},
	}, {
		path:     key.Path{key.New("zap"), key.New("zip")},
		expected: map[int]int{31: 1},
	}, {
		path:     key.Path{key.New("zip"), key.New("zap")},
		expected: map[int]int{41: 1},
	}}

	for _, tc := range testCases {
		result := make(map[int]int, len(tc.expected))
		m.Visit(tc.path, accumulator(result))
		if diff := test.Diff(tc.expected, result); diff != "" {
			t.Errorf("Test case %v: %s", tc.path, diff)
		}
	}
}

func TestMapVisitError(t *testing.T) {
	m := Map{}
	m.Set(key.Path{key.New("foo"), key.New("bar")}, 1)
	m.Set(key.Path{Wildcard, key.New("bar")}, 2)

	errTest := errors.New("Test")

	err := m.Visit(key.Path{key.New("foo"), key.New("bar")},
		func(v interface{}) error { return errTest })
	if err != errTest {
		t.Errorf("Unexpected error. Expected: %v, Got: %v", errTest, err)
	}
	err = m.VisitPrefixes(key.Path{key.New("foo"), key.New("bar"), key.New("baz")},
		func(v interface{}) error { return errTest })
	if err != errTest {
		t.Errorf("Unexpected error. Expected: %v, Got: %v", errTest, err)
	}
}

func TestMapGet(t *testing.T) {
	m := Map{}
	m.Set(key.Path{}, 0)
	m.Set(key.Path{key.New("foo"), key.New("bar")}, 1)
	m.Set(key.Path{key.New("foo"), Wildcard}, 2)
	m.Set(key.Path{Wildcard, key.New("bar")}, 3)
	m.Set(key.Path{key.New("zap"), key.New("zip")}, 4)
	m.Set(key.Path{key.New("baz"), key.New("qux")}, nil)

	testCases := []struct {
		path key.Path
		v    interface{}
		ok   bool
	}{{
		path: key.Path{},
		v:    0,
		ok:   true,
	}, {
		path: key.Path{key.New("foo"), key.New("bar")},
		v:    1,
		ok:   true,
	}, {
		path: key.Path{key.New("foo"), Wildcard},
		v:    2,
		ok:   true,
	}, {
		path: key.Path{Wildcard, key.New("bar")},
		v:    3,
		ok:   true,
	}, {
		path: key.Path{key.New("baz"), key.New("qux")},
		v:    nil,
		ok:   true,
	}, {
		path: key.Path{key.New("bar"), key.New("foo")},
		v:    nil,
	}, {
		path: key.Path{key.New("zap"), Wildcard},
		v:    nil,
	}}

	for _, tc := range testCases {
		v, ok := m.Get(tc.path)
		if v != tc.v || ok != tc.ok {
			t.Errorf("Test case %v: Expected (v: %v, ok: %t), Got (v: %v, ok: %t)",
				tc.path, tc.v, tc.ok, v, ok)
		}
	}
}

func countNodes(m *Map) int {
	if m == nil {
		return 0
	}
	count := 1
	count += countNodes(m.wildcard)
	for _, child := range m.children {
		count += countNodes(child)
	}
	return count
}

func TestMapDelete(t *testing.T) {
	m := Map{}
	m.Set(key.Path{}, 0)
	m.Set(key.Path{Wildcard}, 1)
	m.Set(key.Path{key.New("foo"), key.New("bar")}, 2)
	m.Set(key.Path{key.New("foo"), Wildcard}, 3)
	m.Set(key.Path{key.New("foo")}, 4)

	n := countNodes(&m)
	if n != 5 {
		t.Errorf("Initial count wrong. Expected: 5, Got: %d", n)
	}

	testCases := []struct {
		del      key.Path    // key.Path to delete
		expected bool        // expected return value of Delete
		visit    key.Path    // key.Path to Visit
		before   map[int]int // Expected to find items before deletion
		after    map[int]int // Expected to find items after deletion
		count    int         // Count of nodes
	}{{
		del:      key.Path{key.New("zap")}, // A no-op Delete
		expected: false,
		visit:    key.Path{key.New("foo"), key.New("bar")},
		before:   map[int]int{2: 1, 3: 1},
		after:    map[int]int{2: 1, 3: 1},
		count:    5,
	}, {
		del:      key.Path{key.New("foo"), key.New("bar")},
		expected: true,
		visit:    key.Path{key.New("foo"), key.New("bar")},
		before:   map[int]int{2: 1, 3: 1},
		after:    map[int]int{3: 1},
		count:    4,
	}, {
		del:      key.Path{key.New("foo")},
		expected: true,
		visit:    key.Path{key.New("foo")},
		before:   map[int]int{1: 1, 4: 1},
		after:    map[int]int{1: 1},
		count:    4,
	}, {
		del:      key.Path{key.New("foo")},
		expected: false,
		visit:    key.Path{key.New("foo")},
		before:   map[int]int{1: 1},
		after:    map[int]int{1: 1},
		count:    4,
	}, {
		del:      key.Path{Wildcard},
		expected: true,
		visit:    key.Path{key.New("foo")},
		before:   map[int]int{1: 1},
		after:    map[int]int{},
		count:    3,
	}, {
		del:      key.Path{Wildcard},
		expected: false,
		visit:    key.Path{key.New("foo")},
		before:   map[int]int{},
		after:    map[int]int{},
		count:    3,
	}, {
		del:      key.Path{key.New("foo"), Wildcard},
		expected: true,
		visit:    key.Path{key.New("foo"), key.New("bar")},
		before:   map[int]int{3: 1},
		after:    map[int]int{},
		count:    1, // Should have deleted "foo" and "bar" nodes
	}, {
		del:      key.Path{},
		expected: true,
		visit:    key.Path{},
		before:   map[int]int{0: 1},
		after:    map[int]int{},
		count:    1, // Root node can't be deleted
	}}

	for i, tc := range testCases {
		beforeResult := make(map[int]int, len(tc.before))
		m.Visit(tc.visit, accumulator(beforeResult))
		if diff := test.Diff(tc.before, beforeResult); diff != "" {
			t.Errorf("Test case %d (%v): %s", i, tc.del, diff)
		}

		if got := m.Delete(tc.del); got != tc.expected {
			t.Errorf("Test case %d (%v): Unexpected return. Expected %t, Got: %t",
				i, tc.del, tc.expected, got)
		}

		afterResult := make(map[int]int, len(tc.after))
		m.Visit(tc.visit, accumulator(afterResult))
		if diff := test.Diff(tc.after, afterResult); diff != "" {
			t.Errorf("Test case %d (%v): %s", i, tc.del, diff)
		}
	}
}

func TestMapVisitPrefixes(t *testing.T) {
	m := Map{}
	m.Set(key.Path{}, 0)
	m.Set(key.Path{key.New("foo")}, 1)
	m.Set(key.Path{key.New("foo"), key.New("bar")}, 2)
	m.Set(key.Path{key.New("foo"), key.New("bar"), key.New("baz")}, 3)
	m.Set(key.Path{key.New("foo"), key.New("bar"), key.New("baz"), key.New("quux")}, 4)
	m.Set(key.Path{key.New("quux"), key.New("bar")}, 5)
	m.Set(key.Path{key.New("foo"), key.New("quux")}, 6)
	m.Set(key.Path{Wildcard}, 7)
	m.Set(key.Path{key.New("foo"), Wildcard}, 8)
	m.Set(key.Path{Wildcard, key.New("bar")}, 9)
	m.Set(key.Path{Wildcard, key.New("quux")}, 10)
	m.Set(key.Path{key.New("quux"), key.New("quux"), key.New("quux"), key.New("quux")}, 11)

	testCases := []struct {
		path     key.Path
		expected map[int]int
	}{{
		path:     key.Path{key.New("foo"), key.New("bar"), key.New("baz")},
		expected: map[int]int{0: 1, 1: 1, 2: 1, 3: 1, 7: 1, 8: 1, 9: 1},
	}, {
		path:     key.Path{key.New("zip"), key.New("zap")},
		expected: map[int]int{0: 1, 7: 1},
	}, {
		path:     key.Path{key.New("foo"), key.New("zap")},
		expected: map[int]int{0: 1, 1: 1, 8: 1, 7: 1},
	}, {
		path:     key.Path{key.New("quux"), key.New("quux"), key.New("quux")},
		expected: map[int]int{0: 1, 7: 1, 10: 1},
	}}

	for _, tc := range testCases {
		result := make(map[int]int, len(tc.expected))
		m.VisitPrefixes(tc.path, accumulator(result))
		if diff := test.Diff(tc.expected, result); diff != "" {
			t.Errorf("Test case %v: %s", tc.path, diff)
		}
	}
}

func TestMapVisitPrefixed(t *testing.T) {
	m := Map{}
	m.Set(key.Path{}, 0)
	m.Set(key.Path{key.New("qux")}, 1)
	m.Set(key.Path{key.New("foo")}, 2)
	m.Set(key.Path{key.New("foo"), key.New("qux")}, 3)
	m.Set(key.Path{key.New("foo"), key.New("bar")}, 4)
	m.Set(key.Path{Wildcard, key.New("bar")}, 5)
	m.Set(key.Path{key.New("foo"), Wildcard}, 6)
	m.Set(key.Path{key.New("qux"), key.New("foo"), key.New("bar")}, 7)

	testCases := []struct {
		in  key.Path
		out map[int]int
	}{{
		in:  key.Path{},
		out: map[int]int{0: 1, 1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1, 7: 1},
	}, {
		in:  key.Path{key.New("qux")},
		out: map[int]int{1: 1, 5: 1, 7: 1},
	}, {
		in:  key.Path{key.New("foo")},
		out: map[int]int{2: 1, 3: 1, 4: 1, 5: 1, 6: 1},
	}, {
		in:  key.Path{key.New("foo"), key.New("qux")},
		out: map[int]int{3: 1, 6: 1},
	}, {
		in:  key.Path{key.New("foo"), key.New("bar")},
		out: map[int]int{4: 1, 5: 1, 6: 1},
	}, {
		in:  key.Path{key.New(int64(0))},
		out: map[int]int{5: 1},
	}, {
		in:  key.Path{Wildcard},
		out: map[int]int{5: 1},
	}, {
		in:  key.Path{Wildcard, Wildcard},
		out: map[int]int{},
	}}

	for _, tc := range testCases {
		out := make(map[int]int, len(tc.out))
		m.VisitPrefixed(tc.in, accumulator(out))
		if diff := test.Diff(tc.out, out); diff != "" {
			t.Errorf("Test case %v: %s", tc.out, diff)
		}
	}
}

func TestMapString(t *testing.T) {
	m := Map{}
	m.Set(key.Path{}, 0)
	m.Set(key.Path{key.New("foo"), key.New("bar")}, 1)
	m.Set(key.Path{key.New("foo"), key.New("quux")}, 2)
	m.Set(key.Path{key.New("foo"), Wildcard}, 3)

	expected := `Val: 0
Child "foo":
  Child "*":
    Val: 3
  Child "bar":
    Val: 1
  Child "quux":
    Val: 2
`
	got := fmt.Sprint(&m)

	if expected != got {
		t.Errorf("Unexpected string. Expected:\n\n%s\n\nGot:\n\n%s", expected, got)
	}
}

func genWords(count, wordLength int) key.Path {
	chars := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	if count+wordLength > len(chars) {
		panic("need more chars")
	}
	result := make(key.Path, count)
	for i := 0; i < count; i++ {
		result[i] = key.New(string(chars[i : i+wordLength]))
	}
	return result
}

func benchmarkPathMap(pathLength, pathDepth int, b *testing.B) {
	// Push pathDepth paths, each of length pathLength
	path := genWords(pathLength, 10)
	words := genWords(pathDepth, 10)
	m := &Map{}
	for _, element := range path {
		m.children = map[key.Key]*Map{}
		for _, word := range words {
			m.children[word] = &Map{}
		}
		m = m.children[element]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Visit(path, func(v interface{}) error { return nil })
	}
}

func BenchmarkPathMap1x25(b *testing.B)  { benchmarkPathMap(1, 25, b) }
func BenchmarkPathMap10x50(b *testing.B) { benchmarkPathMap(10, 25, b) }
func BenchmarkPathMap20x50(b *testing.B) { benchmarkPathMap(20, 25, b) }
