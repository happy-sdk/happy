// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars

import (
	"unsafe"
)

func valueFromPtr[T any](ptr unsafe.Pointer) T {
	return *(*T)(ptr)
}

// That is super unsafe call. Pointer must match with kind.
func (k Kind) valueFromPtr(ptr unsafe.Pointer) (val any) {
	if ptr == nil {
		return nil
	}
	switch k {
	case KindBool:
		val = valueFromPtr[bool](ptr)
	case KindInt:
		val = valueFromPtr[int](ptr)
	case KindInt8:
		val = valueFromPtr[int8](ptr)
	case KindInt16:
		val = valueFromPtr[int16](ptr)
	case KindInt32:
		val = valueFromPtr[int32](ptr)
	case KindInt64:
		val = valueFromPtr[int64](ptr)
	case KindUint:
		val = valueFromPtr[uint](ptr)
	case KindUint8:
		val = valueFromPtr[uint8](ptr)
	case KindUint16:
		val = valueFromPtr[uint16](ptr)
	case KindUint32:
		val = valueFromPtr[uint32](ptr)
	case KindUint64:
		val = valueFromPtr[uint64](ptr)
	case KindUintptr, KindPointer, KindUnsafePointer:
		val = valueFromPtr[uintptr](ptr)
	case KindFloat32:
		val = valueFromPtr[float32](ptr)
	case KindFloat64:
		val = valueFromPtr[float64](ptr)
	case KindComplex64:
		val = valueFromPtr[complex64](ptr)
	case KindComplex128:
		val = valueFromPtr[complex128](ptr)
	case KindString:
		val = valueFromPtr[string](ptr)
	case KindSlice:
		val = valueFromPtr[[]byte](ptr)
	default:
		val = nil
	}
	return val
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

// interface for the header of builtin value
type kindeiface struct {
	kind *kindinfo
	ptr  unsafe.Pointer
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
	f := uint8(Kind(t.kind & ((1 << 5) - 1)))
	if t.kind&(1<<5) == 0 {
		f |= uint8(1 << 7)
		kind = Kind(f & (1<<5 - 1))
	} else {
		kind = Kind(t.kind & ((1 << 5) - 1))
	}

	if !withvalue {
		return nil, kind
	}

	return kind.valueFromPtr(e.ptr), kind
}

// Float32bits returns the IEEE 754 binary representation of f,
// with the sign bit of f and the result in the same bit position.
// Float32bits(Float32frombits(x)) == x.
func mathFloat32bits(f float32) uint32 { return *(*uint32)(unsafe.Pointer(&f)) }

// Float32frombits returns the floating-point number corresponding
// to the IEEE 754 binary representation b, with the sign bit of b
// and the result in the same bit position.
// Float32frombits(Float32bits(x)) == x.
func mathFloat32frombits(b uint32) float32 { return *(*float32)(unsafe.Pointer(&b)) }

// Float64bits returns the IEEE 754 binary representation of f,
// with the sign bit of f and the result in the same bit position,
// and Float64bits(Float64frombits(x)) == x.
func mathFloat64bits(f float64) uint64 { return *(*uint64)(unsafe.Pointer(&f)) }

// Float64frombits returns the floating-point number corresponding
// to the IEEE 754 binary representation b, with the sign bit of b
// and the result in the same bit position.
// Float64frombits(Float64bits(x)) == x.
func mathFloat64frombits(b uint64) float64 { return *(*float64)(unsafe.Pointer(&b)) }
