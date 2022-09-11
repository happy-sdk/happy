// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vars

import (
	"unsafe"
)

// A Kind represents the specific kind of kinde that a Value represents.
// The zero Kind is not a valid kind.
type Kind uint

const (
	KindInvalid Kind = iota
	KindBool
	KindInt
	KindInt8
	KindInt16
	KindInt32
	KindInt64
	KindUint
	KindUint8
	KindUint16
	KindUint32
	KindUint64
	KindUintptr
	KindFloat32
	KindFloat64
	KindComplex64
	KindComplex128
	KindArray
	KindChan
	KindFunc
	KindInterface
	KindMap
	KindPointer
	KindSlice
	KindString
	KindStruct
	KindUnsafePointer
)

var kindNames = []string{
	KindInvalid:       "invalid",
	KindBool:          "bool",
	KindInt:           "int",
	KindInt8:          "int8",
	KindInt16:         "int16",
	KindInt32:         "int32",
	KindInt64:         "int64",
	KindUint:          "uint",
	KindUint8:         "uint8",
	KindUint16:        "uint16",
	KindUint32:        "uint32",
	KindUint64:        "uint64",
	KindUintptr:       "uintptr",
	KindFloat32:       "float32",
	KindFloat64:       "float64",
	KindComplex64:     "complex64",
	KindComplex128:    "complex128",
	KindArray:         "array",
	KindChan:          "chan",
	KindFunc:          "func",
	KindInterface:     "interface",
	KindMap:           "map",
	KindPointer:       "ptr",
	KindSlice:         "slice",
	KindString:        "string",
	KindStruct:        "struct",
	KindUnsafePointer: "unsafe.Pointer",
}

func (k Kind) String() (str string) {
	if uint(k) < uint(len(kindNames)) {
		str = kindNames[uint(k)]
	}
	return
}

func valueFromPtr[T any](ptr unsafe.Pointer, asKind Kind) T {
	return *(*T)(ptr)
}

// That is super unsafe call. Pointer must match with kind.
func (k Kind) valueFromPtr(ptr unsafe.Pointer) (val any) {
	switch k {
	case KindBool:
		val = valueFromPtr[bool](ptr, k)
	case KindInt:
		val = valueFromPtr[int](ptr, k)
	case KindInt8:
		val = valueFromPtr[int8](ptr, k)
	case KindInt16:
		val = valueFromPtr[int16](ptr, k)
	case KindInt32:
		val = valueFromPtr[int32](ptr, k)
	case KindInt64:
		val = valueFromPtr[int64](ptr, k)
	case KindUint:
		val = valueFromPtr[uint](ptr, k)
	case KindUint8:
		val = valueFromPtr[uint8](ptr, k)
	case KindUint16:
		val = valueFromPtr[uint16](ptr, k)
	case KindUint32:
		val = valueFromPtr[uint32](ptr, k)
	case KindUint64:
		val = valueFromPtr[uint64](ptr, k)
	case KindUintptr, KindPointer, KindUnsafePointer:
		val = valueFromPtr[uintptr](ptr, k)
	case KindFloat32:
		val = valueFromPtr[float32](ptr, k)
	case KindFloat64:
		val = valueFromPtr[float64](ptr, k)
	case KindComplex64:
		val = valueFromPtr[complex64](ptr, k)
	case KindComplex128:
		val = valueFromPtr[complex128](ptr, k)
	case KindString:
		val = valueFromPtr[string](ptr, k)
	default:
		val = nil
	}
	return val
}

// interface for the header of builtin value
type kindeiface struct {
	kind *kindinfo
	ptr  unsafe.Pointer
}

// builtin type info
type kindinfo struct {
	size       uintptr
	ptrdata    uintptr // number of bytes in the kinde that can contain pointers
	hash       uint32  // hash of type; avoids computation in hash tables
	tflag      uint8   // extra type information flags
	align      uint8   // alignment of variable with this type
	fieldAlign uint8   // alignment of struct field with this type
	kind       uint8   // enumeration for C
}

func underlyingValueOf(in any, withvalue bool) (val any, kind Kind) {
	e := (*kindeiface)(unsafe.Pointer(&in))

	// check whether it is really a pointer or not.
	t := e.kind
	if in == nil || t == nil {
		return nil, KindInvalid
	}

	// there are 27 kinds.
	// check whether t is stored indirectly in an interface value.
	f := uintptr(Kind(t.kind & ((1 << 5) - 1)))
	if t.kind&(1<<5) == 0 {
		f |= uintptr(1 << 7)
		kind = Kind(f & (1<<5 - 1))
	} else {
		kind = Kind(t.kind & ((1 << 5) - 1))
	}

	if !withvalue {
		return nil, kind
	}

	return kind.valueFromPtr(e.ptr), kind
}
