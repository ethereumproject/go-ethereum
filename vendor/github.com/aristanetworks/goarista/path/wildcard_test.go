// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package path

import (
	"encoding/json"
	"testing"

	"github.com/aristanetworks/goarista/key"
	"github.com/aristanetworks/goarista/value"
)

type pseudoWildcard struct{}

func (w pseudoWildcard) Key() interface{} {
	return struct{}{}
}

func (w pseudoWildcard) String() string {
	return "*"
}

func (w pseudoWildcard) Equal(other interface{}) bool {
	o, ok := other.(pseudoWildcard)
	return ok && w == o
}

func TestWildcardUniqueness(t *testing.T) {
	if Wildcard.Equal(pseudoWildcard{}) {
		t.Fatal("Wildcard is not unique")
	}
	if Wildcard.Equal(struct{}{}) {
		t.Fatal("Wildcard is not unique")
	}
	if Wildcard.Equal(key.New("*")) {
		t.Fatal("Wildcard is not unique")
	}
}

func TestWildcardTypeIsNotAKey(t *testing.T) {
	var intf interface{} = WildcardType{}
	_, ok := intf.(key.Key)
	if ok {
		t.Error("WildcardType should not implement key.Key")
	}
}

func TestWildcardTypeEqual(t *testing.T) {
	k1 := key.New(WildcardType{})
	k2 := key.New(WildcardType{})
	if !k1.Equal(k2) {
		t.Error("They should be equal")
	}
	if !Wildcard.Equal(k1) {
		t.Error("They should be equal")
	}
}

func TestWildcardTypeAsValue(t *testing.T) {
	var k value.Value = WildcardType{}
	w := WildcardType{}
	if k.ToBuiltin() != w {
		t.Error("WildcardType.ToBuiltin is not correct")
	}
}

func TestWildcardMarshalJSON(t *testing.T) {
	b, err := json.Marshal(Wildcard)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"_wildcard":{}}`
	if string(b) != expected {
		t.Errorf("Invalid Wildcard json representation.\nExpected: %s\nReceived: %s",
			expected, string(b))
	}
}
