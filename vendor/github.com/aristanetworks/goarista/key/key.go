// Copyright (c) 2015 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/aristanetworks/goarista/value"
)

// Key represents the Key in the updates and deletes of the Notification
// objects.  The only reason this exists is that Go won't let us define
// our own hash function for non-hashable types, and unfortunately we
// need to be able to index maps by map[string]interface{} objects
// and slices by []interface{} objects.
type Key interface {
	Key() interface{}
	String() string
	Equal(other interface{}) bool
}

// compositeKey allows storing a map[string]interface{} or []interface{} as a key
// in a Go map. This is useful when the key isn't a fixed data structure known
// at compile time but rather something generic, like a bag of key-value pairs
// or a list of elements. Go does not allow storing a map or slice inside the
// key of a map, because maps and slices are not comparable or hashable, and
// keys in maps and slice elements must be both.  This file is a hack specific
// to the 'gc' implementation of Go (which is the one most people use when they
// use Go), to bypass this check, by abusing reflection to override how Go
// compares compositeKey for equality or how it's hashed. The values allowed in
// this map are only the types whitelisted in New() as well as map[Key]interface{}
// and []interface{}.
//
// See also https://github.com/golang/go/issues/283
type compositeKey struct {
	// This value must always be set to the sentinel constant above.
	sentinel uintptr
	m        map[string]interface{}
	s        []interface{}
}

type interfaceKey struct {
	key interface{}
}

type strKey string

type int8Key int8
type int16Key int16
type int32Key int32
type int64Key int64

type uint8Key int8
type uint16Key int16
type uint32Key int32
type uint64Key int64

type float32Key float32
type float64Key float64

type boolKey bool

type pointerKey compositeKey

type pathKey compositeKey

func pathToSlice(path Path) []interface{} {
	s := make([]interface{}, len(path))
	for i, element := range path {
		s[i] = element.Key()
	}
	return s
}

func sliceToPath(s []interface{}) Path {
	path := make(Path, len(s))
	for i, intf := range s {
		path[i] = New(intf)
	}
	return path
}

func pointerToSlice(ptr Pointer) []interface{} {
	return pathToSlice(ptr.Pointer())
}

func sliceToPointer(s []interface{}) pointer {
	return pointer(sliceToPath(s))
}

// New wraps the given value in a Key.
// This function panics if the value passed in isn't allowed in a Key or
// doesn't implement value.Value.
func New(intf interface{}) Key {
	switch t := intf.(type) {
	case map[string]interface{}:
		return compositeKey{sentinel: sentinel, m: t}
	case []interface{}:
		return compositeKey{sentinel: sentinel, s: t}
	case string:
		return strKey(t)
	case int8:
		return int8Key(t)
	case int16:
		return int16Key(t)
	case int32:
		return int32Key(t)
	case int64:
		return int64Key(t)
	case uint8:
		return uint8Key(t)
	case uint16:
		return uint16Key(t)
	case uint32:
		return uint32Key(t)
	case uint64:
		return uint64Key(t)
	case float32:
		return float32Key(t)
	case float64:
		return float64Key(t)
	case bool:
		return boolKey(t)
	case value.Value:
		return interfaceKey{key: intf}
	case Pointer:
		return pointerKey{sentinel: sentinel, s: pointerToSlice(t)}
	case Path:
		return pathKey{sentinel: sentinel, s: pathToSlice(t)}
	default:
		panic(fmt.Sprintf("Invalid type for key: %T", intf))
	}
}

func (k interfaceKey) Key() interface{} {
	return k.key
}

func (k interfaceKey) String() string {
	return stringify(k.key)
}

func (k interfaceKey) GoString() string {
	return fmt.Sprintf("key.New(%#v)", k.Key())
}

func (k interfaceKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.Key())
}

func (k interfaceKey) Equal(other interface{}) bool {
	o, ok := other.(Key)
	return ok && keyEqual(k.key, o.Key())
}

// Comparable types have an equality-testing method.
type Comparable interface {
	// Equal returns true if this object is equal to the other one.
	Equal(other interface{}) bool
}

func mapStringEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		if bv, ok := b[k]; !ok || !keyEqual(av, bv) {
			return false
		}
	}
	return true
}

func sliceEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if !keyEqual(v, b[i]) {
			return false
		}
	}
	return true
}

func keyEqual(a, b interface{}) bool {
	switch a := a.(type) {
	case map[string]interface{}:
		b, ok := b.(map[string]interface{})
		return ok && mapStringEqual(a, b)
	case map[Key]interface{}:
		b, ok := b.(map[Key]interface{})
		if !ok || len(a) != len(b) {
			return false
		}
		for k, av := range a {
			if bv, ok := b[k]; !ok || !keyEqual(av, bv) {
				return false
			}
		}
		return true
	case []interface{}:
		b, ok := b.([]interface{})
		return ok && sliceEqual(a, b)
	case Comparable:
		return a.Equal(b)
	case Pointer:
		b, ok := b.(Pointer)
		return ok && pointerEqual(a, b)
	case Path:
		b, ok := b.(Path)
		return ok && pathEqual(a, b)
	}

	return a == b
}

// Key interface implementation for map[string]interface{} and []interface{}
func (k compositeKey) Key() interface{} {
	if k.m != nil {
		return k.m
	}
	return k.s
}

func (k compositeKey) String() string {
	return stringify(k.Key())
}

func (k compositeKey) GoString() string {
	return fmt.Sprintf("key.New(%#v)", k.Key())
}

func (k compositeKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.Key())
}

func (k compositeKey) Equal(other interface{}) bool {
	o, ok := other.(compositeKey)
	if k.m != nil {
		return ok && mapStringEqual(k.m, o.m)
	}
	return ok && sliceEqual(k.s, o.s)
}

func (k strKey) Key() interface{} {
	return string(k)
}

func (k strKey) String() string {
	return string(k)
}

func (k strKey) GoString() string {
	return fmt.Sprintf("key.New(%q)", string(k))
}

func (k strKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(escape(string(k)))
}

func (k strKey) Equal(other interface{}) bool {
	o, ok := other.(strKey)
	return ok && k == o
}

// Key interface implementation for int8
func (k int8Key) Key() interface{} {
	return int8(k)
}

func (k int8Key) String() string {
	return strconv.FormatInt(int64(k), 10)
}

func (k int8Key) GoString() string {
	return fmt.Sprintf("key.New(int8(%d))", int8(k))
}

func (k int8Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(k), 10)), nil
}

func (k int8Key) Equal(other interface{}) bool {
	o, ok := other.(int8Key)
	return ok && k == o
}

// Key interface implementation for int16
func (k int16Key) Key() interface{} {
	return int16(k)
}

func (k int16Key) String() string {
	return strconv.FormatInt(int64(k), 10)
}

func (k int16Key) GoString() string {
	return fmt.Sprintf("key.New(int16(%d))", int16(k))
}

func (k int16Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(k), 10)), nil
}

func (k int16Key) Equal(other interface{}) bool {
	o, ok := other.(int16Key)
	return ok && k == o
}

// Key interface implementation for int32
func (k int32Key) Key() interface{} {
	return int32(k)
}

func (k int32Key) String() string {
	return strconv.FormatInt(int64(k), 10)
}

func (k int32Key) GoString() string {
	return fmt.Sprintf("key.New(int32(%d))", int32(k))
}

func (k int32Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(k), 10)), nil
}

func (k int32Key) Equal(other interface{}) bool {
	o, ok := other.(int32Key)
	return ok && k == o
}

// Key interface implementation for int64
func (k int64Key) Key() interface{} {
	return int64(k)
}

func (k int64Key) String() string {
	return strconv.FormatInt(int64(k), 10)
}

func (k int64Key) GoString() string {
	return fmt.Sprintf("key.New(int64(%d))", int64(k))
}

func (k int64Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(int64(k), 10)), nil
}

func (k int64Key) Equal(other interface{}) bool {
	o, ok := other.(int64Key)
	return ok && k == o
}

// Key interface implementation for uint8
func (k uint8Key) Key() interface{} {
	return uint8(k)
}

func (k uint8Key) String() string {
	return strconv.FormatUint(uint64(k), 10)
}

func (k uint8Key) GoString() string {
	return fmt.Sprintf("key.New(uint8(%d))", uint8(k))
}

func (k uint8Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(k), 10)), nil
}

func (k uint8Key) Equal(other interface{}) bool {
	o, ok := other.(uint8Key)
	return ok && k == o
}

// Key interface implementation for uint16
func (k uint16Key) Key() interface{} {
	return uint16(k)
}

func (k uint16Key) String() string {
	return strconv.FormatUint(uint64(k), 10)
}

func (k uint16Key) GoString() string {
	return fmt.Sprintf("key.New(uint16(%d))", uint16(k))
}

func (k uint16Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(k), 10)), nil
}

func (k uint16Key) Equal(other interface{}) bool {
	o, ok := other.(uint16Key)
	return ok && k == o
}

// Key interface implementation for uint32
func (k uint32Key) Key() interface{} {
	return uint32(k)
}

func (k uint32Key) String() string {
	return strconv.FormatUint(uint64(k), 10)
}

func (k uint32Key) GoString() string {
	return fmt.Sprintf("key.New(uint32(%d))", uint32(k))
}

func (k uint32Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(k), 10)), nil
}

func (k uint32Key) Equal(other interface{}) bool {
	o, ok := other.(uint32Key)
	return ok && k == o
}

// Key interface implementation for uint64
func (k uint64Key) Key() interface{} {
	return uint64(k)
}

func (k uint64Key) String() string {
	return strconv.FormatUint(uint64(k), 10)
}

func (k uint64Key) GoString() string {
	return fmt.Sprintf("key.New(uint64(%d))", uint64(k))
}

func (k uint64Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatUint(uint64(k), 10)), nil
}

func (k uint64Key) Equal(other interface{}) bool {
	o, ok := other.(uint64Key)
	return ok && k == o
}

// Key interface implementation for float32
func (k float32Key) Key() interface{} {
	return float32(k)
}

func (k float32Key) String() string {
	return "f" + strconv.FormatInt(int64(math.Float32bits(float32(k))), 10)
}

func (k float32Key) GoString() string {
	return fmt.Sprintf("key.New(float32(%v))", float32(k))
}

func (k float32Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(k), 'g', -1, 32)), nil
}

func (k float32Key) Equal(other interface{}) bool {
	o, ok := other.(float32Key)
	return ok && k == o
}

// Key interface implementation for float64
func (k float64Key) Key() interface{} {
	return float64(k)
}

func (k float64Key) String() string {
	return "f" + strconv.FormatInt(int64(math.Float64bits(float64(k))), 10)
}

func (k float64Key) GoString() string {
	return fmt.Sprintf("key.New(float64(%v))", float64(k))
}

func (k float64Key) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(k), 'g', -1, 64)), nil
}

func (k float64Key) Equal(other interface{}) bool {
	o, ok := other.(float64Key)
	return ok && k == o
}

// Key interface implementation for bool
func (k boolKey) Key() interface{} {
	return bool(k)
}

func (k boolKey) String() string {
	return strconv.FormatBool(bool(k))
}

func (k boolKey) GoString() string {
	return fmt.Sprintf("key.New(%v)", bool(k))
}

func (k boolKey) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatBool(bool(k))), nil
}

func (k boolKey) Equal(other interface{}) bool {
	o, ok := other.(boolKey)
	return ok && k == o
}

// Key interface implementation for Pointer
func (k pointerKey) Key() interface{} {
	return sliceToPointer(k.s)
}

func (k pointerKey) String() string {
	return sliceToPointer(k.s).String()
}

func (k pointerKey) GoString() string {
	return fmt.Sprintf("key.New(%#v)", k.s)
}

func (k pointerKey) MarshalJSON() ([]byte, error) {
	return sliceToPointer(k.s).MarshalJSON()
}

func (k pointerKey) Equal(other interface{}) bool {
	if o, ok := other.(pointerKey); ok {
		return sliceEqual(k.s, o.s)
	}
	key, ok := other.(Key)
	if !ok {
		return false
	}
	return ok && sliceToPointer(k.s).Equal(key.Key())
}

// Key interface implementation for Path
func (k pathKey) Key() interface{} {
	return sliceToPath(k.s)
}

func (k pathKey) String() string {
	return sliceToPath(k.s).String()
}

func (k pathKey) GoString() string {
	return fmt.Sprintf("key.New(%#v)", k.s)
}

func (k pathKey) MarshalJSON() ([]byte, error) {
	return sliceToPath(k.s).MarshalJSON()
}

func (k pathKey) Equal(other interface{}) bool {
	if o, ok := other.(pathKey); ok {
		return sliceEqual(k.s, o.s)
	}
	key, ok := other.(Key)
	if !ok {
		return false
	}
	return ok && sliceToPath(k.s).Equal(key.Key())
}
