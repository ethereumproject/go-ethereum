// Copyright (c) 2016 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key

import (
	"reflect"
	"unsafe"

	"github.com/aristanetworks/goarista/areflect"
)

func hashInterface(v interface{}) uintptr {
	switch v := v.(type) {
	case map[string]interface{}:
		return hashMapString(v)
	case map[Key]interface{}:
		return hashMapKey(v)
	case []interface{}:
		return hashSlice(v)
	case Pointer:
		// This case applies to pointers used
		// as values in maps or slices (i.e.
		// not wrapped in a key).
		return hashSlice(pointerToSlice(v))
	case Path:
		// This case applies to paths used
		// as values in maps or slices (i.e
		// not wrapped in a kay).
		return hashSlice(pathToSlice(v))
	default:
		return _nilinterhash(v)
	}
}

func hashMapString(m map[string]interface{}) uintptr {
	h := uintptr(31 * (len(m) + 1))
	for k, v := range m {
		// Use addition so that the order of iteration doesn't matter.
		h += _strhash(k)
		h += hashInterface(v)
	}
	return h
}

func hashMapKey(m map[Key]interface{}) uintptr {
	h := uintptr(31 * (len(m) + 1))
	for k, v := range m {
		// Use addition so that the order of iteration doesn't matter.
		switch k := k.(type) {
		case interfaceKey:
			h += _nilinterhash(k.key)
		case compositeKey:
			h += hashMapString(k.m)
		}
		h += hashInterface(v)
	}
	return h
}

func hashSlice(s []interface{}) uintptr {
	h := uintptr(31 * (len(s) + 1))
	for _, v := range s {
		h += hashInterface(v)
	}
	return h
}

func hash(p unsafe.Pointer, seed uintptr) uintptr {
	ck := *(*compositeKey)(p)
	if ck.sentinel != sentinel {
		panic("use of unhashable type in a map")
	}
	if ck.m != nil {
		return seed ^ hashMapString(ck.m)
	}
	return seed ^ hashSlice(ck.s)
}

func equal(a unsafe.Pointer, b unsafe.Pointer) bool {
	ca := (*compositeKey)(a)
	cb := (*compositeKey)(b)
	if ca.sentinel != sentinel {
		panic("use of uncomparable type on the lhs of ==")
	}
	if cb.sentinel != sentinel {
		panic("use of uncomparable type on the rhs of ==")
	}
	if ca.m != nil {
		return mapStringEqual(ca.m, cb.m)
	}
	return sliceEqual(ca.s, cb.s)
}

func init() {
	typ := reflect.TypeOf(compositeKey{})
	alg := reflect.ValueOf(typ).Elem().FieldByName("alg").Elem()
	// Pretty certain that doing this voids your warranty.
	// This overwrites the typeAlg of either alg_NOEQ64 (on 32-bit platforms)
	// or alg_NOEQ128 (on 64-bit platforms), which means that all unhashable
	// types that were using this typeAlg are now suddenly hashable and will
	// attempt to use our equal/hash functions, which will lead to undefined
	// behaviors.  But then these types shouldn't have been hashable in the
	// first place, so no one should have attempted to use them as keys in a
	// map.  The compiler will emit an error if it catches someone trying to
	// do this, but if they do it through a map that uses an interface type as
	// the key, then the compiler can't catch it.
	// To prevent this we could instead override the alg pointer in the type,
	// but it's in a read-only data section in the binary (it's put there by
	// dcommontype() in gc/reflect.go), so changing it is also not without
	// perils.  Basically: Here Be Dragons.
	areflect.ForceExport(alg.FieldByName("hash")).Set(reflect.ValueOf(hash))
	areflect.ForceExport(alg.FieldByName("equal")).Set(reflect.ValueOf(equal))
}
