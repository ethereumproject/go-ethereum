// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package path

import "github.com/aristanetworks/goarista/key"

// Wildcard is a special element in a path that is used by Map
// and the Match* functions to match any other element.
var Wildcard = key.New(WildcardType{})

// WildcardType is the type used to construct a Wildcard. It
// implements the value.Value interface so it can be used as
// a key.Key.
type WildcardType struct{}

func (w WildcardType) String() string {
	return "*"
}

// Equal implements the key.Comparable interface.
func (w WildcardType) Equal(other interface{}) bool {
	_, ok := other.(WildcardType)
	return ok
}

// ToBuiltin implements the value.Value interface.
func (w WildcardType) ToBuiltin() interface{} {
	return WildcardType{}
}

// MarshalJSON implements the value.Value interface.
func (w WildcardType) MarshalJSON() ([]byte, error) {
	return []byte(`{"_wildcard":{}}`), nil
}
