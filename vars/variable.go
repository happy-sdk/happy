// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

// Type of original variable.
func (v Variable) Type() Type {
	return v.val.Type()
}

// Value returns Value of variable.
func (v Variable) Value() Value {
	return v.val
}

// Key returns assigned key for this variable.
func (v Variable) Key() string {
	return v.key
}

// Bool returns boolean representation of the var value.
func (v Variable) Bool() bool {
	return v.val.Bool()
}

// Float32 returns Float32 representation of the var value.
func (v Variable) Float32() float32 {
	return v.val.Float32()
}

// Float64 returns float64 representation of the var value.
func (v Variable) Float64() float64 {
	return v.val.Float64()
}

// Complex64 returns complex64 representation of the var value.
func (v Variable) Complex64() complex64 {
	return v.val.Complex64()
}

// Complex128 returns complex128 representation of the var value.
func (v Variable) Complex128() complex128 {
	return v.val.Complex128()
}

// Int returns int representation of the var value.
func (v Variable) Int() int {
	return v.val.Int()
}

// Int8 returns int8 representation of the var value.
func (v Variable) Int8() int8 {
	return v.val.Int8()
}

// Int16 returns int16 representation of the var value.
func (v Variable) Int16() int16 {
	return v.val.Int16()
}

// Int32 returns int32 representation of the var value.
func (v Variable) Int32() int32 {
	return v.val.Int32()
}

// Int64 returns int64 representation of the var value.
func (v Variable) Int64() int64 {
	return v.val.Int64()
}

// Uint returns uint representation of the var value.
func (v Variable) Uint() uint {
	return v.val.Uint()
}

// Uint8 returns uint8 representation of the var value.
func (v Variable) Uint8() uint8 {
	return v.val.Uint8()
}

// Uint16 returns uint16 representation of the var value.
func (v Variable) Uint16() uint16 {
	return v.val.Uint16()
}

// Uint32 returns uint32 representation of the var value.
func (v Variable) Uint32() uint32 {
	return v.val.Uint32()
}

// Uint64 returns uint64 representation of the var value.
func (v Variable) Uint64() uint64 {
	return v.val.Uint64()
}

// Uintptr returns uintptr representation of the var value.
func (v Variable) Uintptr() uintptr {
	return v.val.Uintptr()
}

// String returns string representation of the var value.
func (v Variable) String() string {
	return v.val.String()
}

// Bytes returns []bytes representation of the var value.
func (v Variable) Bytes() []byte {
	return v.val.Bytes()
}

// Runes returns []rune representation of the var value.
func (v Variable) Runes() []rune {
	return v.val.Runes()
}

// Len returns the length of the string representation of the Value.
func (v Variable) Len() int {
	return v.val.Len()
}

// Empty returns true if this Value is empty.
func (v Variable) Empty() bool {
	return v.val.Empty()
}

// Fields calls strings.Fields on Value string.
func (v Variable) Fields() []string {
	return v.val.Fields()
}
