// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aristanetworks/goarista/key"
	"github.com/aristanetworks/goarista/path"
)

func TestPointer(t *testing.T) {
	p := key.NewPointer(path.New("foo"))
	if expected, actual := path.New("foo"), p.Pointer(); !path.Equal(expected, actual) {
		t.Errorf("Expected %#v but got %#v", expected, actual)
	}
	if expected, actual := "{/foo}", fmt.Sprintf("%s", p); actual != expected {
		t.Errorf("Expected %q but got %q", expected, actual)
	}
	if js, err := json.Marshal(p); err != nil {
		t.Errorf("JSON marshaling failed: %s", err)
	} else if expected, actual := `{"_ptr":"/foo"}`, string(js); actual != expected {
		t.Errorf("Expected %q but got %q", expected, actual)
	}
}

type pointer string

func (ptr pointer) Pointer() key.Path {
	return path.FromString(string(ptr))
}

func (ptr pointer) ToBuiltin() interface{} {
	panic("NOT IMPLEMENTED")
}

func (ptr pointer) String() string {
	panic("NOT IMPLEMENTED")
}

func (ptr pointer) MarshalJSON() ([]byte, error) {
	panic("NOT IMPLEMENTED")
}

func TestPointerEqual(t *testing.T) {
	tests := []struct {
		a      key.Pointer
		b      key.Pointer
		result bool
	}{{
		a:      key.NewPointer(nil),
		b:      key.NewPointer(path.New()),
		result: true,
	}, {
		a:      key.NewPointer(path.New()),
		b:      key.NewPointer(nil),
		result: true,
	}, {
		a:      key.NewPointer(path.New("foo")),
		b:      key.NewPointer(path.New("foo")),
		result: true,
	}, {
		a:      key.NewPointer(path.New("foo")),
		b:      key.NewPointer(path.New("bar")),
		result: false,
	}, {
		a:      key.NewPointer(path.New(true)),
		b:      key.NewPointer(path.New("true")),
		result: false,
	}, {
		a:      key.NewPointer(path.New(int8(0))),
		b:      key.NewPointer(path.New(int16(0))),
		result: false,
	}, {
		a:      key.NewPointer(path.New(path.Wildcard, "bar")),
		b:      key.NewPointer(path.New("foo", path.Wildcard)),
		result: false,
	}, {
		a:      key.NewPointer(path.New(map[string]interface{}{"a": "x", "b": "y"})),
		b:      key.NewPointer(path.New(map[string]interface{}{"b": "y", "a": "x"})),
		result: true,
	}, {
		a:      key.NewPointer(path.New(map[string]interface{}{"a": "x", "b": "y"})),
		b:      key.NewPointer(path.New(map[string]interface{}{"x": "x", "y": "y"})),
		result: false,
	}, {
		a:      key.NewPointer(path.New("foo")),
		b:      pointer("/foo"),
		result: true,
	}, {
		a:      pointer("/foo"),
		b:      key.NewPointer(path.New("foo")),
		result: true,
	}, {
		a:      key.NewPointer(path.New("foo")),
		b:      pointer("/foo/bar"),
		result: false,
	}, {
		a:      pointer("/foo/bar"),
		b:      key.NewPointer(path.New("foo")),
		result: false,
	}}

	for i, tcase := range tests {
		if key.New(tcase.a).Equal(key.New(tcase.b)) != tcase.result {
			t.Errorf("Error in pointer comparison for test %d", i)
		}
	}
}

func TestPointerAsKey(t *testing.T) {
	a := key.NewPointer(path.New("foo", path.Wildcard, map[string]interface{}{
		"bar": map[key.Key]interface{}{
			// Should be able to embed pointer key.
			key.New(key.NewPointer(path.New("baz"))):
			// Should be able to embed pointer value.
			key.NewPointer(path.New("baz")),
		},
	}))
	m := map[key.Key]string{
		key.New(a): "a",
	}
	if s, ok := m[key.New(a)]; !ok {
		t.Error("pointer to path not keyed in map")
	} else if s != "a" {
		t.Errorf("pointer to path not mapped to correct value in map: %s", s)
	}

	// Ensure that we preserve custom pointer implementations.
	b := key.New(pointer("/foo/bar"))
	if _, ok := b.Key().(pointer); !ok {
		t.Errorf("pointer implementation not preserved: %T", b.Key())
	}
}

func BenchmarkPointer(b *testing.B) {
	benchmarks := []key.Path{
		path.New(),
		path.New("foo"),
		path.New("foo", "bar"),
		path.New("foo", "bar", "baz"),
		path.New("foo", "bar", "baz", "qux"),
	}
	for i, benchmark := range benchmarks {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				key.NewPointer(benchmark)
			}
		})
	}
}

func BenchmarkPointerAsKey(b *testing.B) {
	benchmarks := []key.Pointer{
		key.NewPointer(path.New()),
		key.NewPointer(path.New("foo")),
		key.NewPointer(path.New("foo", "bar")),
		key.NewPointer(path.New("foo", "bar", "baz")),
		key.NewPointer(path.New("foo", "bar", "baz", "qux")),
	}
	for i, benchmark := range benchmarks {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				key.New(benchmark)
			}
		})
	}
}

func BenchmarkEmbeddedPointerAsKey(b *testing.B) {
	benchmarks := [][]interface{}{
		[]interface{}{key.NewPointer(path.New())},
		[]interface{}{key.NewPointer(path.New("foo"))},
		[]interface{}{key.NewPointer(path.New("foo", "bar"))},
		[]interface{}{key.NewPointer(path.New("foo", "bar", "baz"))},
		[]interface{}{key.NewPointer(path.New("foo", "bar", "baz", "qux"))},
	}
	for i, benchmark := range benchmarks {
		b.Run(fmt.Sprintf("%d", i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				key.New(benchmark)
			}
		})
	}
}
