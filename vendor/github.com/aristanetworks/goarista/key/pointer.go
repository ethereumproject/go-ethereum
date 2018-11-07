// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key

import (
	"fmt"
)

// Pointer is a pointer to a path.
type Pointer interface {
	Pointer() Path
}

// NewPointer creates a new pointer to a path.
func NewPointer(path Path) Pointer {
	return pointer(path)
}

// This is the type returned by pointerKey.Key. Returning this is a
// lot faster than having pointerKey implement Pointer, since it is
// a compositeKey and thus would require reconstructing a Path from
// []interface{} any time the Pointer method is called.
type pointer Path

func (ptr pointer) Pointer() Path {
	return Path(ptr)
}

func (ptr pointer) String() string {
	return "{" + ptr.Pointer().String() + "}"
}

func (ptr pointer) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"_ptr":%q}`, ptr.Pointer().String())), nil
}

func (ptr pointer) Equal(other interface{}) bool {
	o, ok := other.(Pointer)
	return ok && pointerEqual(ptr, o)
}

func pointerEqual(a, b Pointer) bool {
	return pathEqual(a.Pointer(), b.Pointer())
}
