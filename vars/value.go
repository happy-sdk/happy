// Copyright 2012 Marko Kungla.
// Source code is provider under MIT License.

package vars

import (
	"strconv"
	"strings"
)

// Value describes the variable value
type Value string

func (v Value) String() string {
	return string(v)
}

// Bool returns vars.Value as bool and error if vars.Value does not represent bool value
func (v Value) Bool() (bool, error) {
	return strconv.ParseBool(v.String())
}

// Float returns vars.Value as float64 and error if vars.Value does not float Float value
func (v Value) Float(bitSize int) (float64, error) {
	return strconv.ParseFloat(v.String(), bitSize)
}

// Int returns vars.Value as int64 and error if vars.Value does not represent int value
func (v Value) Int(base int, bitSize int) (int64, error) {
	return strconv.ParseInt(v.String(), base, bitSize)
}

// AsInt returns vars.Value as int and error if vars.Value does not represent int value
func (v Value) AsInt() (int, error) {
	if _, err := strconv.Atoi(v.String()); err != nil {
		return 0, err
	}
	i64, err := strconv.ParseInt(v.String(), 10, 64)
	return int(i64), err
}

// Uint returns vars.Value as uint64 and error if vars.Value does not represent uint value
func (v Value) Uint(base int, bitSize int) (uint64, error) {
	return strconv.ParseUint(v.String(), base, bitSize)
}

// Uintptr returns vars.Value as uintptr and error if vars.Value does not represent uint value
func (v Value) Uintptr() (uintptr, error) {
	ptrInt, err := strconv.ParseUint(v.String(), 10, 64)
	return uintptr(ptrInt), err
}

// Rune returns rune slice
func (v Value) Rune() []rune {
	return []rune(string(v))
}

// Complex64 tries to split Value to strings.Fields and
// use 2 first fields to return complex64
func (v Value) Complex64() (complex64, error) {
	var err error
	fields := v.ParseFields()
	if len(fields) != 2 {
		return complex64(0), strconv.ErrSyntax
	}

	var f1 float64
	var f2 float64
	if f1, err = strconv.ParseFloat(fields[0], 32); err != nil {
		return complex64(0), err
	}
	if f2, err = strconv.ParseFloat(fields[1], 32); err != nil {
		return complex64(0), err
	}
	return complex64(complex(f1, f2)), nil
}

// Complex128 tries to split Value to strings.Fields and
// use 2 first fields to return complex128
func (v Value) Complex128() (complex128, error) {
	var err error
	fields := v.ParseFields()
	if len(fields) != 2 {
		return complex128(0), strconv.ErrSyntax
	}
	var f1 float64
	var f2 float64
	if f1, err = strconv.ParseFloat(fields[0], 64); err != nil {
		return complex128(0), err
	}
	if f2, err = strconv.ParseFloat(fields[1], 64); err != nil {
		return complex128(0), err
	}
	return complex128(complex(f1, f2)), nil
}

// ParseFields calls strings.Fields on Value string
func (v Value) ParseFields() []string {
	return strings.Fields(v.String())
}

// Len returns the length of the string representation of the Value
func (v Value) Len() int {
	return len(v.String())
}

// Empty returns true if this Value is empty
func (v Value) Empty() bool {
	return v.Len() == 0
}
