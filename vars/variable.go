// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import "strings"

// Variable is universl representation of key val pair
type Variable struct {
	key   string
	str   string
	raw   interface{}
	vtype uint
}

// Key returns assigned key for this variable
func (v Variable) Key() string {
	return v.key
}

// Bool returns boolean representation of the var value
func (v Variable) Bool() bool {
	if v.vtype == TypeBool {
		return v.raw.(bool)
	}
	val, _, _ := parseBool(v.str)
	return val
}

// Float32 returns Float32 representation of the var value
func (v Variable) Float32() float32 {
	if v.vtype == TypeFloat32 {
		return v.raw.(float32)
	}
	val, _, _ := parseFloat(v.str, 32)
	return float32(val)
}

// Float64 returns float64 representation of the var value
func (v Variable) Float64() float64 {
	if v.vtype == TypeFloat64 {
		return v.raw.(float64)
	}
	val, _, _ := parseFloat(v.str, 64)
	return val
}

// Complex64 returns complex64 representation of the var value
func (v Variable) Complex64() complex64 {
	if v.vtype == TypeComplex64 {
		return v.raw.(complex64)
	}
	val, _, _ := parseComplex64(v.str)
	return val
}

// Complex128 returns complex128 representation of the var value
func (v Variable) Complex128() complex128 {
	if v.vtype == TypeComplex128 {
		return v.raw.(complex128)
	}
	val, _, _ := parseComplex128(v.str)
	return val
}

// Int returns int representation of the var value
func (v Variable) Int() int {
	if v.vtype == TypeInt {
		return v.raw.(int)
	}
	val, _, _ := parseInt(v.str, 10, 0)
	return int(val)
}

// Int8 returns int8 representation of the var value
func (v Variable) Int8() int8 {
	if v.vtype == TypeInt8 {
		return v.raw.(int8)
	}
	val, _, _ := parseInt(v.str, 10, 8)
	return int8(val)
}

// Int16 returns int16 representation of the var value
func (v Variable) Int16() int16 {
	if v.vtype == TypeInt16 {
		return v.raw.(int16)
	}
	val, _, _ := parseInt(v.str, 10, 16)
	return int16(val)
}

// Int32 returns int32 representation of the var value
func (v Variable) Int32() int32 {
	if v.vtype == TypeInt32 {
		return v.raw.(int32)
	}
	val, _, _ := parseInt(v.str, 10, 32)
	return int32(val)
}

// Int64 returns int64 representation of the var value
func (v Variable) Int64() int64 {
	if v.vtype == TypeInt64 {
		return v.raw.(int64)
	}
	val, _, _ := parseInt(v.str, 10, 64)
	return int64(val)
}

// Uint returns uint representation of the var value
func (v Variable) Uint() uint {
	if v.vtype == TypeUint {
		return v.raw.(uint)
	}
	val, _, _ := parseUint(v.str, 10, 0)
	return uint(val)
}

// Uint8 returns uint8 representation of the var value
func (v Variable) Uint8() uint8 {
	if v.vtype == TypeUint8 {
		return v.raw.(uint8)
	}
	val, _, _ := parseUint(v.str, 10, 8)
	return uint8(val)
}

// Uint16 returns uint16 representation of the var value
func (v Variable) Uint16() uint16 {
	if v.vtype == TypeUint16 {
		return v.raw.(uint16)
	}
	val, _, _ := parseUint(v.str, 10, 16)
	return uint16(val)
}

// Uint32 returns uint32 representation of the var value
func (v Variable) Uint32() uint32 {
	if v.vtype == TypeUint32 {
		return v.raw.(uint32)
	}
	val, _, _ := parseUint(v.str, 10, 32)
	return uint32(val)
}

// Uint64 returns uint64 representation of the var value
func (v Variable) Uint64() uint64 {
	if v.vtype == TypeUint64 {
		return v.raw.(uint64)
	}
	val, _, _ := parseUint(v.str, 10, 64)
	return uint64(val)
}

// Uintptr returns uintptr representation of the var value
func (v Variable) Uintptr() uintptr {
	if v.vtype == TypeUintptr {
		return v.raw.(uintptr)
	}
	val, _, _ := parseUint(v.str, 10, 64)
	return uintptr(val)
}

// String returns string representation of the var value
func (v Variable) String() string {
	return v.str
}

// Bytes returns []bytes representation of the var value
func (v Variable) Bytes() []byte {
	return []byte(v.str)
}

// Runes returns []rune representation of the var value
func (v Variable) Runes() []rune {
	return []rune(v.str)
}

// Len returns the length of the string representation of the Value
func (v Variable) Len() int {
	return len(v.str)
}

// Empty returns true if this Value is empty
func (v Variable) Empty() bool {
	return v.Len() == 0 || v.raw == nil
}

// ParseFields calls strings.Fields on Variable string
func (v Variable) ParseFields() []string {
	return strings.Fields(string(v.str))
}
