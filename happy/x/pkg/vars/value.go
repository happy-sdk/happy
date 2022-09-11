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

type (
	// Value describes the arbitrary value.
	// Value can be typed when value Kind is detected or forced when parsing.
	// All composite types and type alias values are converted into basic type.
	// Since Value holds its raw value and cached sting representation then.
	// All non basic types which successfully convert into basic type and
	// implement fmt.Stringer will have their string value set to value returned
	// by val.String().
	// Value is not modifiable after created and is safe for concurrent use.
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

	// ValueIface is minimal interface for Value to implement by thirtparty libraries.
	ValueIface interface {
		// String MUST return string value Value
		String() string
		// Underlying MUST return original value from what this
		// Value was created.
		Underlying() any
	}
)

// CloneAs takes argument Kind and tries to create new typed value from this value.
// Error returned would be same as calling NewTypedValue(v.Underlying())
func (v Value) CloneAs(kind Kind) (Value, error) {
	return NewValueAs(v.raw, kind)
}

// String returns string representation of the Value.
func (v Value) String() string {
	return v.str
}

// Kind of value is reporting current Values Kind and may not reflect
// original underlying values type.
func (v Value) Kind() Kind {
	return v.kind
}

// Underlying returns value from what this Value was created.
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
		return len(v.str)
	}
	return len(v.str)
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
	if v.kind == KindInt64 {
		if vv, ok := v.raw.(int64); ok {
			return vv, nil
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

// Fields is like calling strings.Fields on Value.String().
// It returns slice of strings (words) found in Value string representation.
func (v Value) Fields() []string {
	return stringsFields(v.str)
}

var valueAsciiStripAround = [256]uint8{
	'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1,
}

var asciiQuotes = [256]uint8{
	'"': 1, '\'': 1, '`': 1, '\\': 1,
}

func trimAndUnquoteValue(str string) string {
	// Fast path for ASCII: look for the first ASCII allwed byte.
	start := 0

	for ; start < len(str); start++ {
		c := str[start]
		if c >= utf8RuneSelf {
			// If we run into a non-ASCII byte, fall back to the
			// slower unicode-aware method on the remaining bytes
			return stringsTrimFunc(str[start:], valueUnicodeIsNotAllowedAround)
		}
		if asciiQuotes[c] == 1 {
			start++
			break
		}
		if c > 32 && valueAsciiStripAround[c] == 0 {
			break
		}
	}

	// Now look for the first ASCII allwed byte from the end.
	stop := len(str)
	for ; stop > start; stop-- {
		c := str[stop-1]
		if c >= utf8RuneSelf {
			// start has been already trimmed above, should trim end only
			return stringsTrimRightFunc(str[start:stop], valueUnicodeIsNotAllowedAround)
		}
		if asciiQuotes[c] == 1 {
			stop--
			c2 := str[stop-1]
			if c2 == '\\' {
				stop--
			}
			break
		}
		if c > 32 && valueAsciiStripAround[c] == 0 {
			break
		}
	}

	return str[start:stop]
}

func valueUnicodeIsNotAllowedAround(r rune) bool {
	// This property isn't the same as Z; special-case it.
	if uint32(r) <= unicodeMaxLatin1 {
		switch r {
		// spaces
		case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		case '\\':
		case '"', '\'', '`':
			return true
		}
		return false
	}
	return unicodeIsExcludingLatin(unicodeWhiteSpace, r)
}
