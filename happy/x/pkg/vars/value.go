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
	"fmt"
	"strings"
)

// Value describes the value.
type Value struct {
	typ Type
	str string
	raw any
	// marks that underlying type was custom type.
	// Custom Value may have its str set with Stringer
	// so converion between types must be made from undelying value.
	// Take for e.g time.Duration(123456) which is int64 while
	// it's string value will be 123.456µs which means we can not parse
	// another types from that string.
	isCustom bool
}

func (v Value) AsType(typ Type) (Value, error) {
	if v.raw == nil {
		return EmptyValue, fmt.Errorf("%w: underlying value is <nil>", ErrValue)
	}

	return NewTypedValue(v.raw, typ)
}

// String returns string representation of the Value.
func (v Value) String() string {
	return v.str
}

// Type of value.
func (v Value) Type() Type {
	return v.typ
}

func (v Value) Underlying() any {
	return v.raw
}

// Empty returns true if this Value is empty.
func (v Value) Empty() bool {
	return v.Len() == 0 || v.raw == nil
}

// Len returns the length of the string representation of the Value.
func (v Value) Len() int {
	if v.isCustom {
		return len(fmt.Sprint(v.raw))
	}
	return len(v.str)
}

// Bool returns boolean representation of the var Value.
func (v Value) Bool() (bool, error) {
	if v.typ == TypeBool {
		if vv, ok := v.raw.(bool); ok {
			return vv, nil
		}
	}

	if v.isCustom {
		vv, err := v.AsType(TypeBool)
		if err != nil {
			return false, err
		}
		return vv.Bool()
	}
	val, _, err := parseBool(v.str)
	return val, err
}

// Int returns int representation of the Value.
func (v Value) Int() (int, error) {
	if v.typ == TypeInt {
		if vv, ok := v.raw.(int); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.AsType(TypeInt)
		if err != nil {
			return 0, err
		}
		return vv.Int()
	}
	val, _, err := parseInt(v.str, 10, 0)
	return int(val), err
}

// Int8 returns int8 representation of the Value.
func (v Value) Int8() (int8, error) {
	if v.typ == TypeInt8 {
		if vv, ok := v.raw.(int8); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.AsType(TypeInt8)
		if err != nil {
			return 0, err
		}
		return vv.Int8()
	}
	val, _, err := parseInt(v.str, 10, 8)
	return int8(val), err
}

// Int16 returns int16 representation of the Value.
func (v Value) Int16() (int16, error) {
	if v.typ == TypeInt16 {
		if vv, ok := v.raw.(int16); ok {
			return vv, nil
		}
	}

	var (
		i   int16
		err error
	)

	if v.typ == TypeInt8 {
		var vi int8
		vi, err = v.Int8()
		return int16(vi), err
	} else if v.isCustom {
		vv, err := v.AsType(TypeInt16)
		if err != nil {
			return 0, err
		}
		return vv.Int16()
	}
	var vi int64
	vi, _, err = parseInt(v.str, 10, 16)
	i = int16(vi)
	return i, err
}

// Int32 returns int32 representation of the Value.
func (v Value) Int32() (int32, error) {
	if v.typ == TypeInt32 {
		if vv, ok := v.raw.(int32); ok {
			return vv, nil
		}
	}

	var (
		i   int32
		err error
	)

	switch v.typ {
	case TypeInt:
		var vi int
		vi, err = v.Int()
		i = int32(vi)
	case TypeInt8:
		var vi int8
		vi, err = v.Int8()
		i = int32(vi)
	case TypeInt16:
		var vi int16
		vi, err = v.Int16()
		i = int32(vi)
	default:
		if v.isCustom {
			vv, err := v.AsType(TypeInt32)
			if err != nil {
				return 0, err
			}
			return vv.Int32()
		}
		var vi int64
		vi, _, err = parseInt(v.str, 10, 32)
		i = int32(vi)
	}
	return i, err
}

// Int64 returns int64 representation of the Value.
func (v Value) Int64() (int64, error) {
	if v.typ == TypeInt64 {
		if vv, ok := v.raw.(int64); ok {
			return vv, nil
		}
	}

	var (
		i   int64
		err error
	)

	switch v.typ {
	case TypeInt:
		var vi int
		vi, err = v.Int()
		i = int64(vi)
	case TypeInt8:
		var vi int8
		vi, err = v.Int8()
		i = int64(vi)
	case TypeInt16:
		var vi int16
		vi, err = v.Int16()
		i = int64(vi)
	case TypeInt32:
		var vi int32
		vi, err = v.Int32()
		i = int64(vi)
	default:
		if v.isCustom {
			vv, err := v.AsType(TypeInt64)
			if err != nil {
				return 0, err
			}
			return vv.Int64()
		}
		i, _, err = parseInt(v.str, 10, 64)
	}
	return i, err
}

// Uint returns uint representation of the Value.
func (v Value) Uint() (uint, error) {
	if v.typ == TypeUint {
		if vv, ok := v.raw.(uint); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.AsType(TypeUint)
		if err != nil {
			return 0, err
		}
		return vv.Uint()
	}
	val, _, err := parseUint(v.str, 10, 0)
	return uint(val), err
}

// Uint8 returns uint8 representation of the Value.
func (v Value) Uint8() (uint8, error) {
	if v.typ == TypeUint8 {
		if vv, ok := v.raw.(uint8); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.AsType(TypeUint8)
		if err != nil {
			return 0, err
		}
		return vv.Uint8()
	}
	val, _, err := parseUint(v.str, 10, 8)
	return uint8(val), err
}

// Uint16 returns uint16 representation of the Value.
func (v Value) Uint16() (uint16, error) {
	if v.typ == TypeUint16 {
		if vv, ok := v.raw.(uint16); ok {
			return vv, nil
		}
	}

	var (
		i   uint16
		err error
	)

	if v.typ == TypeInt8 {
		var vi uint8
		vi, err = v.Uint8()
		return uint16(vi), err
	} else if v.isCustom {
		vv, err := v.AsType(TypeUint16)
		if err != nil {
			return 0, err
		}
		return vv.Uint16()
	}
	var vi uint64
	vi, _, err = parseUint(v.str, 10, 16)
	i = uint16(vi)
	return i, err
}

// Uint32 returns uint32 representation of the Value.
func (v Value) Uint32() (uint32, error) {
	if v.typ == TypeUint32 {
		if vv, ok := v.raw.(uint32); ok {
			return vv, nil
		}
	}

	var (
		i   uint32
		err error
	)

	switch v.typ {
	case TypeInt:
		var vi uint
		vi, err = v.Uint()
		i = uint32(vi)
	case TypeInt8:
		var vi uint8
		vi, err = v.Uint8()
		i = uint32(vi)
	case TypeInt16:
		var vi uint16
		vi, err = v.Uint16()
		i = uint32(vi)
	default:
		if v.isCustom {
			vv, err := v.AsType(TypeUint32)
			if err != nil {
				return 0, err
			}
			return vv.Uint32()
		}
		var vi uint64
		vi, _, err = parseUint(v.str, 10, 32)
		i = uint32(vi)
	}
	return i, err
}

// Uint64 returns uint64 representation of the Value.
func (v Value) Uint64() (uint64, error) {
	if v.typ == TypeUint64 {
		if vv, ok := v.raw.(uint64); ok {
			return vv, nil
		}
	}
	var (
		i   uint64
		err error
	)

	switch v.typ {
	case TypeInt:
		var vi uint
		vi, err = v.Uint()
		i = uint64(vi)
	case TypeInt8:
		var vi uint8
		vi, err = v.Uint8()
		i = uint64(vi)
	case TypeInt16:
		var vi uint16
		vi, err = v.Uint16()
		i = uint64(vi)
	case TypeInt32:
		var vi uint32
		vi, err = v.Uint32()
		i = uint64(vi)
	default:
		if v.isCustom {
			vv, err := v.AsType(TypeUint64)
			if err != nil {
				return 0, err
			}
			return vv.Uint64()
		}
		i, _, err = parseUint(v.str, 10, 64)
	}
	return i, err
}

// Float32 returns Float32 representation of the Value.
func (v Value) Float32() (float32, error) {
	if v.typ == TypeFloat32 {
		if vv, ok := v.raw.(float32); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.AsType(TypeFloat32)
		if err != nil {
			return 0, err
		}
		return vv.Float32()
	}
	val, _, err := parseFloat(v.str, 32)
	return float32(val), err
}

// Float64 returns float64 representation of Value.
func (v Value) Float64() (float64, error) {
	if v.typ == TypeFloat64 {
		if vv, ok := v.raw.(float64); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.AsType(TypeFloat64)
		if err != nil {
			return 0, err
		}
		return vv.Float64()
	}
	val, _, err := parseFloat(v.str, 64)
	return val, err
}

// Complex64 returns complex64 representation of the Value.
func (v Value) Complex64() (complex64, error) {
	if v.typ == TypeComplex64 {
		if vv, ok := v.raw.(complex64); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.AsType(TypeComplex64)
		if err != nil {
			return 0, err
		}
		return vv.Complex64()
	}
	val, _, err := parseComplex64(v.str)
	return val, err
}

// Complex128 returns complex128 representation of the Value.
func (v Value) Complex128() (complex128, error) {
	if v.typ == TypeComplex128 {
		if vv, ok := v.raw.(complex128); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.AsType(TypeComplex128)
		if err != nil {
			return 0, err
		}
		return vv.Complex128()
	}
	val, _, err := parseComplex128(v.str)
	return val, err
}

// Uintptr returns uintptr representation of the Value.
func (v Value) Uintptr() (uintptr, error) {
	if v.typ == TypeUintptr {
		if vv, ok := v.raw.(uintptr); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.AsType(TypeUintptr)
		if err != nil {
			return 0, err
		}
		return vv.Uintptr()
	}
	val, _, err := parseUint(v.str, 10, 64)
	return uintptr(val), err
}

// Fields calls strings.Fields on Value string.
func (v Value) Fields() []string {
	return strings.Fields(v.str)
}
