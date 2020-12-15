// Copyright 2012 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import "strings"

// Value describes the variable value
type Value string

// Bool returns boolean representation of the var Value
func (v Value) Bool() bool {
	switch v {
	case "1", "t", "T", "true", "TRUE", "True":
		return true
	}
	return false
}

// Float32 returns Float32 representation of the Value
func (v Value) Float32() float32 {
	val, _, _ := parseFloat(string(v), 32)
	return float32(val)
}

// Float64 returns float64 representation of Value
func (v Value) Float64() float64 {
	val, _, _ := parseFloat(string(v), 64)
	return val
}

// Complex64 returns complex64 representation of the Value
func (v Value) Complex64() complex64 {
	val, _, _ := parseComplex64(string(v))
	return val
}

// Complex128 returns complex128 representation of the Value
func (v Value) Complex128() complex128 {
	val, _, _ := parseComplex128(string(v))
	return val
}

// Int returns int representation of the Value
func (v Value) Int() int {
	val, _, _ := parseInt(string(v), 10, 0)
	return int(val)
}

// Int8 returns int8 representation of the Value
func (v Value) Int8() int8 {
	val, _, _ := parseInt(string(v), 10, 8)
	return int8(val)
}

// Int16 returns int16 representation of the Value
func (v Value) Int16() int16 {
	val, _, _ := parseInt(string(v), 10, 16)
	return int16(val)
}

// Int32 returns int32 representation of the Value
func (v Value) Int32() int32 {
	val, _, _ := parseInt(string(v), 10, 32)
	return int32(val)
}

// Int64 returns int64 representation of the Value
func (v Value) Int64() int64 {
	val, _, _ := parseInt(string(v), 10, 64)
	return int64(val)
}

// Uint returns uint representation of the Value
func (v Value) Uint() uint {
	val, _, _ := parseUint(string(v), 10, 0)
	return uint(val)
}

// Uint8 returns uint8 representation of the Value
func (v Value) Uint8() uint8 {
	val, _, _ := parseUint(string(v), 10, 8)
	return uint8(val)
}

// Uint16 returns uint16 representation of the Value
func (v Value) Uint16() uint16 {
	val, _, _ := parseUint(string(v), 10, 16)
	return uint16(val)
}

// Uint32 returns uint32 representation of the Value
func (v Value) Uint32() uint32 {
	val, _, _ := parseUint(string(v), 10, 32)
	return uint32(val)
}

// Uint64 returns uint64 representation of the Value
func (v Value) Uint64() uint64 {
	val, _, _ := parseUint(string(v), 10, 64)
	return uint64(val)
}

// Uintptr returns uintptr representation of the Value
func (v Value) Uintptr() uintptr {
	val, _, _ := parseUint(string(v), 10, 64)
	return uintptr(val)
}

// String returns string representation of the Value
func (v Value) String() string {
	return string(v)
}

// Bytes returns []bytes representation of the Value
func (v Value) Bytes() []byte {
	return []byte(v)
}

// Runes returns []rune representation of the var value
func (v Value) Runes() []rune {
	return []rune(v)
}

// Len returns the length of the string representation of the Value
func (v Value) Len() int {
	return len(string(v))
}

// Empty returns true if this Value is empty
func (v Value) Empty() bool {
	return v.Len() == 0
}

// ParseFields calls strings.Fields on Value string
func (v Value) ParseFields() []string {
	return strings.Fields(string(v))
}
