// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars

import "time"

type (
	// Value describes an arbitrary value. When the Kind of the value is detected
	// or forced during parsing, the Value can be typed. All composite types and
	// type alias values are converted into basic types. The Value holds its raw
	// value and cached string representation. Non-basic types that successfully
	// convert into basic types and implement fmt.Stringer will have their string
	// value set to the value returned by val.String().
	//
	// The Value is not modifiable after it is created
	// and is safe for concurrent use.
	Value struct {
		// kind represents underlying value type. It is KindInvalid when parsing
		// initial value fails or source value is not one of the supported Kind's.
		kind Kind

		// str holds cached string representation for underlying value.
		str string

		// raw is original value type. It can only be builtin type.
		// That side effect should be taken into account, that .Underlying()
		// may return builtin type of original instead of value
		// what was initially provided.
		raw any

		// isCustom marks that underlying type was custom type.
		// Custom Value may have its str set with Stringer
		// so converion between types must be made from undelying value.
		// Take for e.g time.Duration(123456) which is int64 while
		// it's string value will be 123.456µs which means we can not parse
		// another types from that string.
		isCustom bool
	}
)

// String returns string representation of the Value.
func (v Value) String() string {
	return v.str
}

// Any returns underlying value from what this Value was created.
func (v Value) Any() any {
	return v.raw
}

// Kind of value is reporting current Values Kind
// and may not reflect original underlying values type.
func (v Value) Kind() Kind {
	return v.kind
}

// Empty returns true if this Value is empty.
func (v Value) Empty() bool {
	return v.Len() == 0 || v.raw == nil
}

// Len returns the length of the string representation of the Value.
func (v Value) Len() int {
	return len(v.str)
}

// CloneAs takes argument Kind and tries to create new typed value from this value.
// Error returned would be same as calling NewTypedValue(v.Underlying())
func (v Value) CloneAs(kind Kind) (Value, error) {
	return NewValueAs(v.raw, kind)
}

// Bool returns boolean representation of the Value.
func (v Value) Bool() (bool, error) {
	if v.kind == KindBool {
		if vv, ok := v.raw.(bool); ok {
			return vv, nil
		}
	}

	if v.isCustom {
		vv, err := v.CloneAs(KindBool)
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
	if v.kind == KindInt {
		if vv, ok := v.raw.(int); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindInt)
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
	if v.kind == KindInt8 {
		if vv, ok := v.raw.(int8); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindInt8)
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
	if v.kind == KindInt16 {
		if vv, ok := v.raw.(int16); ok {
			return vv, nil
		}
	}

	var (
		i   int16
		err error
	)

	if v.kind == KindInt8 {
		var vi int8
		vi, err = v.Int8()
		return int16(vi), err
	} else if v.isCustom {
		vv, err := v.CloneAs(KindInt16)
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
func (v Value) Duration() (time.Duration, error) {
	if v.kind == KindDuration {
		if vv, ok := v.raw.(time.Duration); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindDuration)
		if err != nil {
			return 0, err
		}
		return vv.Duration()
	}

	val, err := time.ParseDuration(v.str)
	return val, err
}

// Int32 returns int32 representation of the Value.
func (v Value) Int32() (int32, error) {
	if v.kind == KindInt32 {
		if vv, ok := v.raw.(int32); ok {
			return vv, nil
		}
	}

	var (
		i   int32
		err error
	)

	switch v.kind {
	case KindInt:
		var vi int
		vi, err = v.Int()
		i = int32(vi)
	case KindInt8:
		var vi int8
		vi, err = v.Int8()
		i = int32(vi)
	case KindInt16:
		var vi int16
		vi, err = v.Int16()
		i = int32(vi)
	default:
		if v.isCustom {
			vv, err := v.CloneAs(KindInt32)
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
	if v.kind == KindInt64 || v.kind == KindDuration {
		if vv, ok := v.raw.(int64); ok {
			return vv, nil
		}
	}
	if v.kind == KindDuration {
		if vv, ok := v.raw.(time.Duration); ok {
			return int64(vv), nil
		}
	}

	var (
		i   int64
		err error
	)

	switch v.kind {
	case KindInt:
		var vi int
		vi, err = v.Int()
		i = int64(vi)
	case KindInt8:
		var vi int8
		vi, err = v.Int8()
		i = int64(vi)
	case KindInt16:
		var vi int16
		vi, err = v.Int16()
		i = int64(vi)
	case KindInt32:
		var vi int32
		vi, err = v.Int32()
		i = int64(vi)
	default:
		if v.isCustom {
			vv, err := v.CloneAs(KindInt64)
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
	if v.kind == KindUint {
		if vv, ok := v.raw.(uint); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindUint)
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
	if v.kind == KindUint8 {
		if vv, ok := v.raw.(uint8); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindUint8)
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
	if v.kind == KindUint16 {
		if vv, ok := v.raw.(uint16); ok {
			return vv, nil
		}
	}

	var (
		i   uint16
		err error
	)

	if v.kind == KindInt8 {
		var vi uint8
		vi, err = v.Uint8()
		return uint16(vi), err
	} else if v.isCustom {
		vv, err := v.CloneAs(KindUint16)
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
	if v.kind == KindUint32 {
		if vv, ok := v.raw.(uint32); ok {
			return vv, nil
		}
	}

	var (
		i   uint32
		err error
	)

	switch v.kind {
	case KindInt:
		var vi uint
		vi, err = v.Uint()
		i = uint32(vi)
	case KindInt8:
		var vi uint8
		vi, err = v.Uint8()
		i = uint32(vi)
	case KindInt16:
		var vi uint16
		vi, err = v.Uint16()
		i = uint32(vi)
	default:
		if v.isCustom {
			vv, err := v.CloneAs(KindUint32)
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
	if v.kind == KindUint64 {
		if vv, ok := v.raw.(uint64); ok {
			return vv, nil
		}
	}
	var (
		i   uint64
		err error
	)

	switch v.kind {
	case KindInt:
		var vi uint
		vi, err = v.Uint()
		i = uint64(vi)
	case KindInt8:
		var vi uint8
		vi, err = v.Uint8()
		i = uint64(vi)
	case KindInt16:
		var vi uint16
		vi, err = v.Uint16()
		i = uint64(vi)
	case KindInt32:
		var vi uint32
		vi, err = v.Uint32()
		i = uint64(vi)
	default:
		if v.isCustom {
			vv, err := v.CloneAs(KindUint64)
			if err != nil {
				return 0, err
			}
			return vv.Uint64()
		}
		i, _, err = parseUint(v.str, 10, 64)
	}
	return i, err
}

// Float32 returns float32 representation of the Value.
func (v Value) Float32() (float32, error) {
	if v.kind == KindFloat32 {
		if vv, ok := v.raw.(float32); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindFloat32)
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
	if v.kind == KindFloat64 {
		if vv, ok := v.raw.(float64); ok {
			return vv, nil
		}
	}
	if v.kind == KindFloat32 {
		if vv, ok := v.raw.(float32); ok {
			return float64(vv), nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindFloat64)
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
	if v.kind == KindComplex64 {
		if vv, ok := v.raw.(complex64); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindComplex64)
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
	if v.kind == KindComplex128 {
		if vv, ok := v.raw.(complex128); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindComplex128)
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
	if v.kind == KindUintptr {
		if vv, ok := v.raw.(uintptr); ok {
			return vv, nil
		}
	}
	if v.isCustom {
		vv, err := v.CloneAs(KindUintptr)
		if err != nil {
			return 0, err
		}
		return vv.Uintptr()
	}
	val, _, err := parseUint(v.str, 10, 64)
	return uintptr(val), err
}

// FormatInt returns the string representation of i in the given base,
// for 2 <= base <= 36. The result uses the lower-case letters 'a' to 'z'
// for digit values >= 10.
func (v Value) FormatInt(base int) string {
	i, _ := v.Int64()
	return formatIntFast(i, base)
}

// FormatUint returns the string representation of i in the given base,
// for 2 <= base <= 36. The result uses the lower-case letters 'a' to 'z'
// for digit values >= 10.
func (v Value) FormatUint(base int) string {
	u, _ := v.Uint64()
	return formatUintFast(u, base)
}

// FormatFloat converts the floating-point number f to a string,
// according to the format fmt and precision prec. It rounds the
// result assuming that the original was obtained from a floating-point
// value of bitSize bits (32 for float32, 64 for float64).
//
// The format fmt is one of
// 'b' (-ddddp±ddd, a binary exponent),
// 'e' (-d.dddde±dd, a decimal exponent),
// 'E' (-d.ddddE±dd, a decimal exponent),
// 'f' (-ddd.dddd, no exponent),
// 'g' ('e' for large exponents, 'f' otherwise),
// 'G' ('E' for large exponents, 'f' otherwise),
// 'x' (-0xd.ddddp±ddd, a hexadecimal fraction and binary exponent), or
// 'X' (-0Xd.ddddP±ddd, a hexadecimal fraction and binary exponent).
//
// The precision prec controls the number of digits (excluding the exponent)
// printed by the 'e', 'E', 'f', 'g', 'G', 'x', and 'X' formats.
// For 'e', 'E', 'f', 'x', and 'X', it is the number of digits after the decimal point.
// For 'g' and 'G' it is the maximum number of significant digits (trailing
// zeros are removed).
// The special precision -1 uses the smallest number of digits
func (v Value) FormatFloat(fmt byte, prec, bitSize int) string {
	f, _ := v.Float64()
	return string(fastFtoa(make([]byte, 0, max(prec+4, 24)), f, fmt, prec, bitSize))
}

// Fields is like calling strings.Fields on Value.String().
// It returns slice of strings (words) found in Value string representation.
func (v Value) Fields() []string {
	return stringsFields(v.str)
}

// ValueIface is minimal interface for Value to implement by thirtparty libraries.
type ValueIface interface {
	// String MUST return string value Value
	String() string
	// Underlying MUST return original value from what this
	// Value was created.
	Any() any
	Len() int
	Bool() (bool, error)
	Int() (int, error)
	Int8() (int8, error)
	Int16() (int16, error)
	Int32() (int32, error)
	Int64() (int64, error)
	Uint() (uint, error)
	Uint8() (uint8, error)
	Uint16() (uint16, error)
	Uint32() (uint32, error)
	Uint64() (uint64, error)
	Float32() (float32, error)
	Float64() (float64, error)
	Complex64() (complex64, error)
	Complex128() (complex128, error)
	Uintptr() (uintptr, error)
	Fields() []string
}
