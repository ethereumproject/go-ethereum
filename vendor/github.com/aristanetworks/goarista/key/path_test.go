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

func TestPath(t *testing.T) {
	expected := path.New("leet")
	k := key.New(expected)

	keyPath, ok := k.Key().(key.Path)
	if !ok {
		t.Fatalf("key.Key() did not return a key.Path")
	}
	if !path.Equal(keyPath, expected) {
		t.Errorf("expected path from key is %#v, got %#v", expected, keyPath)
	}

	if expected, actual := "/leet", fmt.Sprint(k); actual != expected {
		t.Errorf("expected string from path key %q, got %q", expected, actual)
	}

	if js, err := json.Marshal(k); err != nil {
		t.Errorf("JSON marshalling error: %v", err)
	} else if expected, actual := `{"_path":"/leet"}`, string(js); actual != expected {
		t.Errorf("expected json output %q, got %q", expected, actual)
	}
}

func newPathKey(e ...interface{}) key.Key {
	return key.New(path.New(e...))
}

type customPath string

func (p customPath) Key() interface{} {
	return path.FromString(string(p))
}

func (p customPath) Equal(other interface{}) bool {
	o, ok := other.(key.Key)
	return ok && o.Equal(p)
}

func (p customPath) String() string               { return string(p) }
func (p customPath) ToBuiltin() interface{}       { panic("not impl") }
func (p customPath) MarshalJSON() ([]byte, error) { panic("not impl") }

func TestPathEqual(t *testing.T) {
	tests := []struct {
		a      key.Key
		b      key.Key
		result bool
	}{{
		a:      newPathKey(),
		b:      nil,
		result: false,
	}, {
		a:      newPathKey(),
		b:      newPathKey(),
		result: true,
	}, {
		a:      newPathKey(),
		b:      newPathKey("foo"),
		result: false,
	}, {
		a:      newPathKey("foo"),
		b:      newPathKey(),
		result: false,
	}, {
		a:      newPathKey(int16(1337)),
		b:      newPathKey(int64(1337)),
		result: false,
	}, {
		a:      newPathKey(path.Wildcard, "bar"),
		b:      newPathKey("foo", path.Wildcard),
		result: false,
	}, {
		a:      newPathKey(map[string]interface{}{"a": "x", "b": "y"}),
		b:      newPathKey(map[string]interface{}{"b": "y", "a": "x"}),
		result: true,
	}, {
		a:      newPathKey(map[string]interface{}{"a": "x", "b": "y"}),
		b:      newPathKey(map[string]interface{}{"x": "x", "y": "y"}),
		result: false,
	}, {
		a:      newPathKey("foo", "bar"),
		b:      customPath("/foo/bar"),
		result: true,
	}, {
		a:      customPath("/foo/bar"),
		b:      newPathKey("foo", "bar"),
		result: true,
	}, {
		a:      newPathKey("foo"),
		b:      key.New(customPath("/bar")),
		result: false,
	}, {
		a:      key.New(customPath("/foo")),
		b:      newPathKey("foo", "bar"),
		result: false,
	}}

	for i, tc := range tests {
		if a, b := tc.a, tc.b; a.Equal(b) != tc.result {
			t.Errorf("result not as expected for test case %d", i)
		}
	}
}

func TestPathAsKey(t *testing.T) {
	a := newPathKey("foo", path.Wildcard, map[string]interface{}{
		"bar": map[key.Key]interface{}{
			// Should be able to embed a path key and value
			newPathKey("path", "to", "something"): path.New("else"),
		},
	})
	m := map[key.Key]string{
		a: "thats a complex key!",
	}
	if s, ok := m[a]; !ok {
		t.Error("complex key not found in map")
	} else if s != "thats a complex key!" {
		t.Errorf("incorrect value in map: %s", s)
	}

	// preserve custom path implementations
	b := key.New(customPath("/foo/bar"))
	if _, ok := b.Key().(customPath); !ok {
		t.Errorf("customPath implementation not preserved: %T", b.Key())
	}
}
