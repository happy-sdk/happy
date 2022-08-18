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

type Type uint

const (
	TypeInvalid Type = iota
	TypeBool
	TypeInt
	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt64
	TypeUint
	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint64
	TypeUintptr
	TypeFloat32
	TypeFloat64
	TypeComplex64
	TypeComplex128
	TypeArray
	TypeChan
	TypeFunc
	TypeInterface
	TypeMap
	TypePointer
	TypeSlice
	TypeString
	TypeStruct
	TypeUnsafePointer
)

var kindNames = []string{
	TypeInvalid:       "invalid",
	TypeBool:          "bool",
	TypeInt:           "int",
	TypeInt8:          "int8",
	TypeInt16:         "int16",
	TypeInt32:         "int32",
	TypeInt64:         "int64",
	TypeUint:          "uint",
	TypeUint8:         "uint8",
	TypeUint16:        "uint16",
	TypeUint32:        "uint32",
	TypeUint64:        "uint64",
	TypeUintptr:       "uintptr",
	TypeFloat32:       "float32",
	TypeFloat64:       "float64",
	TypeComplex64:     "complex64",
	TypeComplex128:    "complex128",
	TypeArray:         "array",
	TypeChan:          "chan",
	TypeFunc:          "func",
	TypeInterface:     "interface",
	TypeMap:           "map",
	TypePointer:       "ptr",
	TypeSlice:         "slice",
	TypeString:        "string",
	TypeStruct:        "struct",
	TypeUnsafePointer: "unsafe.Pointer",
}

func (t Type) String() (str string) {
	if uint(t) < uint(len(kindNames)) {
		str = kindNames[uint(t)]
	}
	return
}

func underlyingValueOf(in any, withvalue bool) (val any, typ Type) {
	e := (*typeiface)(unsafe.Pointer(&in))

	// check whether it is really a pointer or not.
	t := e.typ
	if in == nil || t == nil {
		return nil, TypeInvalid
	}

	// there are 27 kinds.
	// check whether t is stored indirectly in an interface value.
	f := uintptr(Type(t.kind & ((1 << 5) - 1)))
	if t.kind&(1<<5) == 0 {
		f |= uintptr(1 << 7)
		typ = Type(f & (1<<5 - 1))
	} else {
		typ = Type(t.kind & ((1 << 5) - 1))
	}

	if !withvalue {
		return nil, typ
	}
	switch typ {
	case TypeBool:
		val = *(*bool)(e.ptr)
	case TypeInt:
		val = *(*int)(e.ptr)
	case TypeInt8:
		val = *(*int8)(e.ptr)
	case TypeInt16:
		val = *(*int16)(e.ptr)
	case TypeInt32:
		val = *(*int32)(e.ptr)
	case TypeInt64:
		val = *(*int64)(e.ptr)
	case TypeUint:
		val = *(*uint)(e.ptr)
	case TypeUint8:
		val = *(*uint8)(e.ptr)
	case TypeUint16:
		val = *(*uint16)(e.ptr)
	case TypeUint32:
		val = *(*uint32)(e.ptr)
	case TypeUint64:
		val = *(*uint64)(e.ptr)
	case TypeUintptr, TypePointer, TypeUnsafePointer:
		val = *(*uintptr)(e.ptr)
	case TypeFloat32:
		val = *(*float32)(e.ptr)
	case TypeFloat64:
		val = *(*float64)(e.ptr)
	case TypeComplex64:
		val = *(*complex64)(e.ptr)
	case TypeComplex128:
		val = *(*complex128)(e.ptr)
	case TypeString:
		val = *(*string)(e.ptr)
	}
	return val, typ
}

func ValueTypeFor(in any) (typ Type) {
	_, typ = underlyingValueOf(in, false)
	return
}

// interface for the header of builtin value
type typeiface struct {
	typ *typeinfo
	ptr unsafe.Pointer
}

// builtin type info
// lint: go-static
type typeinfo struct {
	size       uintptr
	ptrdata    uintptr // number of bytes in the type that can contain pointers
	hash       uint32  // hash of type; avoids computation in hash tables
	tflag      uint8   // extra type information flags
	align      uint8   // alignment of variable with this type
	fieldAlign uint8   // alignment of struct field with this type
	kind       uint8   // enumeration for C
}
