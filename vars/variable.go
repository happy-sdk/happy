// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import "strings"

// Type of original variable
func (v Variable) Type() Type {
	return v.val.vtype
}

// Key returns assigned key for this variable
func (v Variable) Key() string {
	return v.key
}

// Bool returns boolean representation of the var value
func (v Variable) Bool() bool {
	if v.val.vtype == TypeBool {
		return v.val.raw.(bool)
	}
	val, _, _ := parseBool(v.val.str)
	return val
}

// Float32 returns Float32 representation of the var value
func (v Variable) Float32() float32 {
	if v.val.vtype == TypeFloat32 {
		return v.val.raw.(float32)
	}
	val, _, _ := parseFloat(v.val.str, 32)
	return float32(val)
}

// Float64 returns float64 representation of the var value
func (v Variable) Float64() float64 {
	if v.val.vtype == TypeFloat64 {
		return v.val.raw.(float64)
	}
	val, _, _ := parseFloat(v.val.str, 64)
	return val
}

// Complex64 returns complex64 representation of the var value
func (v Variable) Complex64() complex64 {
	if v.val.vtype == TypeComplex64 {
		return v.val.raw.(complex64)
	}
	val, _, _ := parseComplex64(v.val.str)
	return val
}

// Complex128 returns complex128 representation of the var value
func (v Variable) Complex128() complex128 {
	if v.val.vtype == TypeComplex128 {
		return v.val.raw.(complex128)
	}
	val, _, _ := parseComplex128(v.val.str)
	return val
}

// Int returns int representation of the var value
func (v Variable) Int() int {
	if v.val.vtype == TypeInt {
		return v.val.raw.(int)
	}
	val, _, _ := parseInt(v.val.str, 10, 0)
	return int(val)
}

// Int8 returns int8 representation of the var value
func (v Variable) Int8() int8 {
	if v.val.vtype == TypeInt8 {
		return v.val.raw.(int8)
	}
	val, _, _ := parseInt(v.val.str, 10, 8)
	return int8(val)
}

// Int16 returns int16 representation of the var value
func (v Variable) Int16() int16 {
	if v.val.vtype == TypeInt16 {
		return v.val.raw.(int16)
	}
	val, _, _ := parseInt(v.val.str, 10, 16)
	return int16(val)
}

// Int32 returns int32 representation of the var value
func (v Variable) Int32() int32 {
	if v.val.vtype == TypeInt32 {
		return v.val.raw.(int32)
	}
	val, _, _ := parseInt(v.val.str, 10, 32)
	return int32(val)
}

// Int64 returns int64 representation of the var value
func (v Variable) Int64() int64 {
	if v.val.vtype == TypeInt64 {
		return v.val.raw.(int64)
	}
	val, _, _ := parseInt(v.val.str, 10, 64)
	return int64(val)
}

// Uint returns uint representation of the var value
func (v Variable) Uint() uint {
	if v.val.vtype == TypeUint {
		return v.val.raw.(uint)
	}
	val, _, _ := parseUint(v.val.str, 10, 0)
	return uint(val)
}

// Uint8 returns uint8 representation of the var value
func (v Variable) Uint8() uint8 {
	if v.val.vtype == TypeUint8 {
		return v.val.raw.(uint8)
	}
	val, _, _ := parseUint(v.val.str, 10, 8)
	return uint8(val)
}

// Uint16 returns uint16 representation of the var value
func (v Variable) Uint16() uint16 {
	if v.val.vtype == TypeUint16 {
		return v.val.raw.(uint16)
	}
	val, _, _ := parseUint(v.val.str, 10, 16)
	return uint16(val)
}

// Uint32 returns uint32 representation of the var value
func (v Variable) Uint32() uint32 {
	if v.val.vtype == TypeUint32 {
		return v.val.raw.(uint32)
	}
	val, _, _ := parseUint(v.val.str, 10, 32)
	return uint32(val)
}

// Uint64 returns uint64 representation of the var value
func (v Variable) Uint64() uint64 {
	if v.val.vtype == TypeUint64 {
		return v.val.raw.(uint64)
	}
	val, _, _ := parseUint(v.val.str, 10, 64)
	return uint64(val)
}

// Uintptr returns uintptr representation of the var value
func (v Variable) Uintptr() uintptr {
	if v.val.vtype == TypeUintptr {
		return v.val.raw.(uintptr)
	}
	val, _, _ := parseUint(v.val.str, 10, 64)
	return uintptr(val)
}

// String returns string representation of the var value
func (v Variable) String() string {
	return v.val.str
}

// Bytes returns []bytes representation of the var value
func (v Variable) Bytes() []byte {
	return []byte(v.val.str)
}

// Runes returns []rune representation of the var value
func (v Variable) Runes() []rune {
	return []rune(v.val.str)
}

// Len returns the length of the string representation of the Value
func (v Variable) Len() int {
	return len(v.val.str)
}

// Empty returns true if this Value is empty
func (v Variable) Empty() bool {
	return v.Len() == 0 || v.val.raw == nil
}

// ParseFields calls strings.Fields on Variable string
func (v Variable) ParseFields() []string {
	return strings.Fields(string(v.val.str))
}
