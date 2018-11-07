// Copyright (c) 2017 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package path

import (
	"fmt"
	"testing"

	"github.com/aristanetworks/goarista/key"
	"github.com/aristanetworks/goarista/value"
)

func TestNew(t *testing.T) {
	tcases := []struct {
		in  []interface{}
		out key.Path
	}{
		{
			in:  nil,
			out: key.Path{},
		}, {
			in:  []interface{}{},
			out: key.Path{},
		}, {
			in:  []interface{}{"foo", key.New("bar"), true},
			out: key.Path{key.New("foo"), key.New("bar"), key.New(true)},
		}, {
			in: []interface{}{int8(5), int16(5), int32(5), int64(5)},
			out: key.Path{key.New(int8(5)), key.New(int16(5)), key.New(int32(5)),
				key.New(int64(5))},
		}, {
			in: []interface{}{uint8(5), uint16(5), uint32(5), uint64(5)},
			out: key.Path{key.New(uint8(5)), key.New(uint16(5)), key.New(uint32(5)),
				key.New(uint64(5))},
		}, {
			in:  []interface{}{float32(5), float64(5)},
			out: key.Path{key.New(float32(5)), key.New(float64(5))},
		}, {
			in:  []interface{}{customKey{i: &a}, map[string]interface{}{}},
			out: key.Path{key.New(customKey{i: &a}), key.New(map[string]interface{}{})},
		},
	}
	for i, tcase := range tcases {
		if p := New(tcase.in...); !Equal(p, tcase.out) {
			t.Fatalf("Test %d failed: %#v != %#v", i, p, tcase.out)
		}
	}
}

func TestClone(t *testing.T) {
	if !Equal(Clone(key.Path{}), key.Path{}) {
		t.Error("Clone(key.Path{}) != key.Path{}")
	}
	a := key.Path{key.New("foo"), key.New("bar")}
	b, c := Clone(a), Clone(a)
	b[1] = key.New("baz")
	if Equal(a, b) || !Equal(a, c) {
		t.Error("Clone is not making a copied path")
	}
}

func TestAppend(t *testing.T) {
	tcases := []struct {
		a      key.Path
		b      []interface{}
		result key.Path
	}{
		{
			a:      key.Path{},
			b:      []interface{}{},
			result: key.Path{},
		}, {
			a:      key.Path{key.New("foo")},
			b:      []interface{}{},
			result: key.Path{key.New("foo")},
		}, {
			a:      key.Path{},
			b:      []interface{}{"foo", key.New("bar")},
			result: key.Path{key.New("foo"), key.New("bar")},
		}, {
			a:      key.Path{key.New("foo")},
			b:      []interface{}{int64(0), key.New("bar")},
			result: key.Path{key.New("foo"), key.New(int64(0)), key.New("bar")},
		},
	}
	for i, tcase := range tcases {
		if p := Append(tcase.a, tcase.b...); !Equal(p, tcase.result) {
			t.Fatalf("Test %d failed: %#v != %#v", i, p, tcase.result)
		}
	}
}

func TestJoin(t *testing.T) {
	tcases := []struct {
		paths  []key.Path
		result key.Path
	}{
		{
			paths:  nil,
			result: nil,
		}, {
			paths:  []key.Path{},
			result: nil,
		}, {
			paths:  []key.Path{key.Path{}},
			result: nil,
		}, {
			paths:  []key.Path{key.Path{key.New(true)}, key.Path{}},
			result: key.Path{key.New(true)},
		}, {
			paths:  []key.Path{key.Path{}, key.Path{key.New(true)}},
			result: key.Path{key.New(true)},
		}, {
			paths:  []key.Path{key.Path{key.New("foo")}, key.Path{key.New("bar")}},
			result: key.Path{key.New("foo"), key.New("bar")},
		}, {
			paths:  []key.Path{key.Path{key.New("bar")}, key.Path{key.New("foo")}},
			result: key.Path{key.New("bar"), key.New("foo")},
		}, {
			paths: []key.Path{
				key.Path{key.New(uint32(0)), key.New(uint64(0))},
				key.Path{key.New(int8(0))},
				key.Path{key.New(int16(0)), key.New(int32(0))},
				key.Path{key.New(int64(0)), key.New(uint8(0)), key.New(uint16(0))},
			},
			result: key.Path{
				key.New(uint32(0)), key.New(uint64(0)),
				key.New(int8(0)), key.New(int16(0)),
				key.New(int32(0)), key.New(int64(0)),
				key.New(uint8(0)), key.New(uint16(0)),
			},
		},
	}
	for i, tcase := range tcases {
		if p := Join(tcase.paths...); !Equal(p, tcase.result) {
			t.Fatalf("Test %d failed: %#v != %#v", i, p, tcase.result)
		}
	}
}

func TestParent(t *testing.T) {
	if Parent(key.Path{}) != nil {
		t.Fatal("Parent of empty key.Path should be nil")
	}
	tcases := []struct {
		in  key.Path
		out key.Path
	}{
		{
			in:  key.Path{key.New("foo")},
			out: key.Path{},
		}, {
			in:  key.Path{key.New("foo"), key.New("bar")},
			out: key.Path{key.New("foo")},
		}, {
			in:  key.Path{key.New("foo"), key.New("bar"), key.New("baz")},
			out: key.Path{key.New("foo"), key.New("bar")},
		},
	}
	for _, tcase := range tcases {
		if !Equal(Parent(tcase.in), tcase.out) {
			t.Fatalf("Parent of %#v != %#v", tcase.in, tcase.out)
		}
	}
}

func TestBase(t *testing.T) {
	if Base(key.Path{}) != nil {
		t.Fatal("Base of empty key.Path should be nil")
	}
	tcases := []struct {
		in  key.Path
		out key.Key
	}{
		{
			in:  key.Path{key.New("foo")},
			out: key.New("foo"),
		}, {
			in:  key.Path{key.New("foo"), key.New("bar")},
			out: key.New("bar"),
		},
	}
	for _, tcase := range tcases {
		if !Base(tcase.in).Equal(tcase.out) {
			t.Fatalf("Base of %#v != %#v", tcase.in, tcase.out)
		}
	}
}

type customKey struct {
	i *int
}

func (c customKey) String() string {
	return fmt.Sprintf("customKey=%d", *c.i)
}

func (c customKey) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (c customKey) ToBuiltin() interface{} {
	return nil
}

func (c customKey) Equal(other interface{}) bool {
	o, ok := other.(customKey)
	return ok && *c.i == *o.i
}

var (
	_ value.Value    = customKey{}
	_ key.Comparable = customKey{}
	a                = 1
	b                = 1
)

func TestEqual(t *testing.T) {
	tcases := []struct {
		a      key.Path
		b      key.Path
		result bool
	}{
		{
			a:      nil,
			b:      nil,
			result: true,
		}, {
			a:      nil,
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{},
			b:      nil,
			result: true,
		}, {
			a:      key.Path{},
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{},
			b:      key.Path{key.New("")},
			result: false,
		}, {
			a:      key.Path{Wildcard},
			b:      key.Path{key.New("foo")},
			result: false,
		}, {
			a:      key.Path{Wildcard},
			b:      key.Path{Wildcard},
			result: true,
		}, {
			a:      key.Path{key.New("foo")},
			b:      key.Path{key.New("foo")},
			result: true,
		}, {
			a:      key.Path{key.New(true)},
			b:      key.Path{key.New(false)},
			result: false,
		}, {
			a:      key.Path{key.New(int32(5))},
			b:      key.Path{key.New(int64(5))},
			result: false,
		}, {
			a:      key.Path{key.New("foo")},
			b:      key.Path{key.New("foo"), key.New("bar")},
			result: false,
		}, {
			a:      key.Path{key.New("foo"), key.New("bar")},
			b:      key.Path{key.New("foo")},
			result: false,
		}, {
			a:      key.Path{key.New(uint8(0)), key.New(int8(0))},
			b:      key.Path{key.New(int8(0)), key.New(uint8(0))},
			result: false,
		},
		// Ensure that we check deep equality.
		{
			a:      key.Path{key.New(map[string]interface{}{})},
			b:      key.Path{key.New(map[string]interface{}{})},
			result: true,
		}, {
			a:      key.Path{key.New(customKey{i: &a})},
			b:      key.Path{key.New(customKey{i: &b})},
			result: true,
		},
	}
	for i, tcase := range tcases {
		if result := Equal(tcase.a, tcase.b); result != tcase.result {
			t.Fatalf("Test %d failed: a: %#v; b: %#v, result: %t",
				i, tcase.a, tcase.b, tcase.result)
		}
	}
}

func TestMatch(t *testing.T) {
	tcases := []struct {
		a      key.Path
		b      key.Path
		result bool
	}{
		{
			a:      nil,
			b:      nil,
			result: true,
		}, {
			a:      nil,
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{},
			b:      nil,
			result: true,
		}, {
			a:      key.Path{},
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{},
			b:      key.Path{key.New("foo")},
			result: false,
		}, {
			a:      key.Path{Wildcard},
			b:      key.Path{key.New("foo")},
			result: true,
		}, {
			a:      key.Path{key.New("foo")},
			b:      key.Path{Wildcard},
			result: false,
		}, {
			a:      key.Path{Wildcard},
			b:      key.Path{key.New("foo"), key.New("bar")},
			result: false,
		}, {
			a:      key.Path{Wildcard, Wildcard},
			b:      key.Path{key.New(int64(0))},
			result: false,
		}, {
			a:      key.Path{Wildcard, Wildcard},
			b:      key.Path{key.New(int64(0)), key.New(int32(0))},
			result: true,
		}, {
			a:      key.Path{Wildcard, key.New(false)},
			b:      key.Path{key.New(true), Wildcard},
			result: false,
		},
	}
	for i, tcase := range tcases {
		if result := Match(tcase.a, tcase.b); result != tcase.result {
			t.Fatalf("Test %d failed: a: %#v; b: %#v, result: %t",
				i, tcase.a, tcase.b, tcase.result)
		}
	}
}

func TestHasElement(t *testing.T) {
	tcases := []struct {
		a      key.Path
		b      key.Key
		result bool
	}{
		{
			a:      nil,
			b:      nil,
			result: false,
		}, {
			a:      nil,
			b:      key.New("foo"),
			result: false,
		}, {
			a:      key.Path{},
			b:      nil,
			result: false,
		}, {
			a:      key.Path{key.New("foo")},
			b:      nil,
			result: false,
		}, {
			a:      key.Path{key.New("foo")},
			b:      key.New("foo"),
			result: true,
		}, {
			a:      key.Path{key.New(true)},
			b:      key.New("true"),
			result: false,
		}, {
			a:      key.Path{key.New("foo"), key.New("bar")},
			b:      key.New("bar"),
			result: true,
		}, {
			a:      key.Path{key.New(map[string]interface{}{})},
			b:      key.New(map[string]interface{}{}),
			result: true,
		}, {
			a:      key.Path{key.New(map[string]interface{}{"foo": "a"})},
			b:      key.New(map[string]interface{}{"bar": "a"}),
			result: false,
		},
	}
	for i, tcase := range tcases {
		if result := HasElement(tcase.a, tcase.b); result != tcase.result {
			t.Errorf("Test %d failed: a: %#v; b: %#v, result: %t, expected: %t",
				i, tcase.a, tcase.b, result, tcase.result)
		}
	}
}

func TestHasPrefix(t *testing.T) {
	tcases := []struct {
		a      key.Path
		b      key.Path
		result bool
	}{
		{
			a:      nil,
			b:      nil,
			result: true,
		}, {
			a:      nil,
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{},
			b:      nil,
			result: true,
		}, {
			a:      key.Path{},
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{},
			b:      key.Path{key.New("foo")},
			result: false,
		}, {
			a:      key.Path{key.New("foo")},
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{key.New(true)},
			b:      key.Path{key.New(false)},
			result: false,
		}, {
			a:      key.Path{key.New("foo"), key.New("bar")},
			b:      key.Path{key.New("bar"), key.New("foo")},
			result: false,
		}, {
			a:      key.Path{key.New(int8(0)), key.New(uint8(0))},
			b:      key.Path{key.New(uint8(0)), key.New(uint8(0))},
			result: false,
		}, {
			a:      key.Path{key.New(true), key.New(true)},
			b:      key.Path{key.New(true), key.New(true), key.New(true)},
			result: false,
		}, {
			a:      key.Path{key.New(true), key.New(true), key.New(true)},
			b:      key.Path{key.New(true), key.New(true)},
			result: true,
		}, {
			a:      key.Path{Wildcard, key.New(int32(0)), Wildcard},
			b:      key.Path{key.New(int64(0)), Wildcard},
			result: false,
		},
	}
	for i, tcase := range tcases {
		if result := HasPrefix(tcase.a, tcase.b); result != tcase.result {
			t.Fatalf("Test %d failed: a: %#v; b: %#v, result: %t",
				i, tcase.a, tcase.b, tcase.result)
		}
	}
}

func TestMatchPrefix(t *testing.T) {
	tcases := []struct {
		a      key.Path
		b      key.Path
		result bool
	}{
		{
			a:      nil,
			b:      nil,
			result: true,
		}, {
			a:      nil,
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{},
			b:      nil,
			result: true,
		}, {
			a:      key.Path{},
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{},
			b:      key.Path{key.New("foo")},
			result: false,
		}, {
			a:      key.Path{key.New("foo")},
			b:      key.Path{},
			result: true,
		}, {
			a:      key.Path{key.New("foo")},
			b:      key.Path{Wildcard},
			result: false,
		}, {
			a:      key.Path{Wildcard},
			b:      key.Path{key.New("foo")},
			result: true,
		}, {
			a:      key.Path{Wildcard},
			b:      key.Path{key.New("foo"), key.New("bar")},
			result: false,
		}, {
			a:      key.Path{Wildcard, key.New(true)},
			b:      key.Path{key.New(false), Wildcard},
			result: false,
		}, {
			a:      key.Path{Wildcard, key.New(int32(0)), key.New(int16(0))},
			b:      key.Path{key.New(int64(0)), key.New(int32(0))},
			result: true,
		},
	}
	for i, tcase := range tcases {
		if result := MatchPrefix(tcase.a, tcase.b); result != tcase.result {
			t.Fatalf("Test %d failed: a: %#v; b: %#v, result: %t",
				i, tcase.a, tcase.b, tcase.result)
		}
	}
}

func TestFromString(t *testing.T) {
	tcases := []struct {
		in  string
		out key.Path
	}{
		{
			in:  "",
			out: key.Path{},
		}, {
			in:  "/",
			out: key.Path{},
		}, {
			in:  "//",
			out: key.Path{key.New(""), key.New("")},
		}, {
			in:  "foo",
			out: key.Path{key.New("foo")},
		}, {
			in:  "/foo",
			out: key.Path{key.New("foo")},
		}, {
			in:  "foo/bar",
			out: key.Path{key.New("foo"), key.New("bar")},
		}, {
			in:  "/foo/bar",
			out: key.Path{key.New("foo"), key.New("bar")},
		}, {
			in:  "foo/bar/baz",
			out: key.Path{key.New("foo"), key.New("bar"), key.New("baz")},
		}, {
			in:  "/foo/bar/baz",
			out: key.Path{key.New("foo"), key.New("bar"), key.New("baz")},
		}, {
			in:  "0/123/456/789",
			out: key.Path{key.New("0"), key.New("123"), key.New("456"), key.New("789")},
		}, {
			in:  "/0/123/456/789",
			out: key.Path{key.New("0"), key.New("123"), key.New("456"), key.New("789")},
		}, {
			in:  "`~!@#$%^&*()_+{}\\/|[];':\"<>?,./",
			out: key.Path{key.New("`~!@#$%^&*()_+{}\\"), key.New("|[];':\"<>?,."), key.New("")},
		}, {
			in:  "/`~!@#$%^&*()_+{}\\/|[];':\"<>?,./",
			out: key.Path{key.New("`~!@#$%^&*()_+{}\\"), key.New("|[];':\"<>?,."), key.New("")},
		},
	}
	for i, tcase := range tcases {
		if p := FromString(tcase.in); !Equal(p, tcase.out) {
			t.Fatalf("Test %d failed: %#v != %#v", i, p, tcase.out)
		}
	}
}

func TestString(t *testing.T) {
	tcases := []struct {
		in  key.Path
		out string
	}{
		{
			in:  key.Path{},
			out: "/",
		}, {
			in:  key.Path{key.New("")},
			out: "/",
		}, {
			in:  key.Path{key.New("foo")},
			out: "/foo",
		}, {
			in:  key.Path{key.New("foo"), key.New("bar")},
			out: "/foo/bar",
		}, {
			in:  key.Path{key.New("/foo"), key.New("bar")},
			out: "//foo/bar",
		}, {
			in:  key.Path{key.New("foo"), key.New("bar/")},
			out: "/foo/bar/",
		}, {
			in:  key.Path{key.New(""), key.New("foo"), key.New("bar")},
			out: "//foo/bar",
		}, {
			in:  key.Path{key.New("foo"), key.New("bar"), key.New("")},
			out: "/foo/bar/",
		}, {
			in:  key.Path{key.New("/"), key.New("foo"), key.New("bar")},
			out: "///foo/bar",
		}, {
			in:  key.Path{key.New("foo"), key.New("bar"), key.New("/")},
			out: "/foo/bar//",
		},
	}
	for i, tcase := range tcases {
		if s := tcase.in.String(); s != tcase.out {
			t.Fatalf("Test %d failed: %s != %s", i, s, tcase.out)
		}
	}
}

func BenchmarkJoin(b *testing.B) {
	generate := func(n int) []key.Path {
		paths := make([]key.Path, 0, n)
		for i := 0; i < n; i++ {
			paths = append(paths, key.Path{key.New("foo")})
		}
		return paths
	}
	benchmarks := map[string][]key.Path{
		"10 key.Paths":    generate(10),
		"100 key.Paths":   generate(100),
		"1000 key.Paths":  generate(1000),
		"10000 key.Paths": generate(10000),
	}
	for name, benchmark := range benchmarks {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				Join(benchmark...)
			}
		})
	}
}

func BenchmarkHasElement(b *testing.B) {
	element := key.New("waldo")
	generate := func(n, loc int) key.Path {
		path := make(key.Path, n)
		for i := 0; i < n; i++ {
			if i == loc {
				path[i] = element
			} else {
				path[i] = key.New(int8(0))
			}
		}
		return path
	}
	benchmarks := map[string]key.Path{
		"10 Elements Index 0":     generate(10, 0),
		"10 Elements Index 4":     generate(10, 4),
		"10 Elements Index 9":     generate(10, 9),
		"100 Elements Index 0":    generate(100, 0),
		"100 Elements Index 49":   generate(100, 49),
		"100 Elements Index 99":   generate(100, 99),
		"1000 Elements Index 0":   generate(1000, 0),
		"1000 Elements Index 499": generate(1000, 499),
		"1000 Elements Index 999": generate(1000, 999),
	}
	for name, benchmark := range benchmarks {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				HasElement(benchmark, element)
			}
		})
	}
}
