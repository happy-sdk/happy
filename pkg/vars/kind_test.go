// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/vars"
)

func TestKindOf(t *testing.T) {
	testutils.Equal(t, vars.KindInvalid, vars.KindOf(nil))
	var str string

	testutils.Equal(t, vars.KindPointer.String(), vars.KindOf(&str).String())
	var vstruct vars.VariableIface[vars.Value]
	testutils.Equal(t, vars.KindInvalid.String(), vars.KindOf(vstruct).String())
	// testutils.Equal(t, vars.KindStruct.String(), vars.ValueKindOf(vstruct).String())
	var viface fmt.Stringer
	testutils.Equal(t, vars.KindInvalid.String(), vars.KindOf(viface).String())
}

func TestNewValueKind(t *testing.T) {
	for _, test := range getKindTests() {
		t.Run("bool: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Bool, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindBool, v.Kind(), test.Key)
		})

		t.Run("float32: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Float32, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindFloat32, v.Kind(), test.Key)
		})

		t.Run("float64: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Float64, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindFloat64, v.Kind(), test.Key)
		})

		t.Run("complex64: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Complex64, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindComplex64, v.Kind(), test.Key)
		})

		t.Run("complex128: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Complex128, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindComplex128, v.Kind(), test.Key)
		})

		t.Run("int: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Int, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt, v.Kind(), test.Key)
		})

		t.Run("int8: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Int8, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt8, v.Kind(), test.Key)
		})

		t.Run("int16: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Int16, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt16, v.Kind(), test.Key)
		})

		t.Run("int32: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Int32, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt32, v.Kind(), test.Key)
		})

		t.Run("int64: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Int64, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt64, v.Kind(), test.Key)
		})

		t.Run("uint: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Uint, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint, v.Kind(), test.Key)
		})

		t.Run("uint8: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Uint8, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint8, v.Kind(), test.Key)
		})

		t.Run("uint16: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Uint16, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint16, v.Kind(), test.Key)
		})

		t.Run("uint32: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Uint32, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint32, v.Kind(), test.Key)
		})

		t.Run("uint64: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Uint64, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint64, v.Kind(), test.Key)
		})

		t.Run("uintptr: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.Uintptr, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUintptr, v.Kind(), test.Key)
		})

		t.Run("string: "+test.Key, func(t *testing.T) {
			v, err := vars.New(test.Key, test.String, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindString, v.Kind(), test.Key)
		})

		t.Run("KindUnsafePointer: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseVariableAs(test.Key, test.String, false, vars.KindUnsafePointer)
			testutils.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("KindStruct: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseVariableAs(test.Key, test.String, false, vars.KindStruct)
			testutils.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("KindSlice: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseVariableAs(test.Key, test.String, false, vars.KindSlice)
			testutils.NoError(t, err)
		})
		t.Run("KindMap: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseVariableAs(test.Key, test.String, false, vars.KindMap)
			testutils.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("KindInterface: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseVariableAs(test.Key, test.String, false, vars.KindInterface)
			testutils.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("KindFunc: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseVariableAs(test.Key, test.String, false, vars.KindFunc)
			testutils.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("KindChan: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseVariableAs(test.Key, test.String, false, vars.KindChan)
			testutils.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("KindArray: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseVariableAs(test.Key, test.String, false, vars.KindArray)
			testutils.ErrorIs(t, err, vars.ErrValue)
		})
	}
}

type kindTest struct {
	Key string
	In  string
	// expected
	Bool       bool
	Float32    float32
	Float64    float64
	Complex64  complex64
	Complex128 complex128
	Int        int
	Int8       int8
	Int16      int16
	Int32      int32
	Int64      int64
	Uint       uint
	Uint8      uint8
	Uint16     uint16
	Uint32     uint32
	Uint64     uint64
	Uintptr    uintptr
	String     string
	Bytes      []byte
	Runes      []rune
}

func getKindTests() []kindTest {
	return []kindTest{
		{"INT_1", "1", true, 1, 1, complex(1, 0i), complex(1, 0i), 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, "1", []byte{49}, []rune{49}},
		{"INT_2", "2147483647", false, 2.1474836e+09, 2.147483647e+09, complex(2.147483647e+09, 0i), complex(2.147483647e+09, 0i), 2147483647, 127, 32767, 2147483647, 2147483647, 2147483647, 255, 65535, 2147483647, 2147483647, 2147483647, "2147483647", []byte{50, 49, 52, 55, 52, 56, 51, 54, 52, 55}, []rune{'2', '1', '4', '7', '4', '8', '3', '6', '4', '7'}},
		{"STRING_1", "asdf", false, 0, 0, complex(0, 0i), complex(0, 0i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "asdf", []byte{97, 115, 100, 102}, []rune{'a', 's', 'd', 'f'}},
		{"FLOAT_1", "2." + strings.Repeat("2", 40) + "e+1", false, 22.222221, 22.22222222222222, complex(22.22222222222222, 0i), complex(22.22222222222222, 0i), 22, 22, 22, 22, 22, 0, 0, 0, 0, 0, 0, "2.2222222222222222222222222222222222222222e+1", []byte("2.2222222222222222222222222222222222222222e+1"), []rune("2.2222222222222222222222222222222222222222e+1")},
		{"COMPLEX128_1", "123456700 1e-100", false, 0, 0, complex(1.234567e+08, 0i), complex(1.234567e+08, 1e-100), 0, 127, 32767, 0, 0, 0, 255, 65535, 0, 0, 0, "123456700 1e-100", []byte("123456700 1e-100"), []rune("123456700 1e-100")},
	}
}

type cbool bool
type cint int
type cint8 int8
type cint16 int16
type cint32 int32
type cint64 int64
type cuint uint
type cuint8 uint8
type cuint16 uint16
type cuint32 uint32
type cuint64 uint64
type cuintptr uintptr
type cfloat32 float32
type cfloat64 float64
type ccomplex64 complex64
type ccomplex128 complex128
type cstring string
type cstring2 string

func (c cstring) String() string {
	return string(c)
}

type cany any

func TestCustomKinds(t *testing.T) {
	var ctyps = []struct {
		orig       any
		underlying any
		typ        vars.Kind
		str        string
		len        int
		typtest    kindTest
	}{
		{time.Duration(-123456), time.Duration(-123456), vars.KindDuration, "-123.456µs", 11, kindTest{
			Bool:       false,
			Float32:    -123456,
			Float64:    -123456,
			Complex64:  complex(-123456, 0),
			Complex128: complex(-123456, 0),
			Int:        -123456,
			Int8:       -64,
			Int16:      7616,
			Int32:      -123456,
			Int64:      -123456,
			Uint:       18446744073709428160,
			Uint8:      192,
			Uint16:     7616,
			Uint32:     4294843840,
			Uint64:     18446744073709428160,
			Uintptr:    0,
			String:     "-123.456µs",
		}},
		{time.Duration(123456), time.Duration(123456), vars.KindDuration, "123.456µs", 10, kindTest{
			Bool:       false,
			Float32:    123456,
			Float64:    123456,
			Complex64:  complex(123456, 0),
			Complex128: complex(123456, 0),
			Int:        123456,
			Int8:       0,
			Int16:      0,
			Int32:      123456,
			Int64:      123456,
			Uint:       123456,
			Uint8:      0,
			Uint16:     0,
			Uint32:     123456,
			Uint64:     123456,
			Uintptr:    0,
			String:     "123.456µs",
		}},
		{time.Duration(123), time.Duration(123), vars.KindDuration, "123ns", 5, kindTest{
			Bool:       false,
			Float32:    123,
			Float64:    123,
			Complex64:  complex(123, 0),
			Complex128: complex(123, 0),
			Int:        123,
			Int8:       123,
			Int16:      123,
			Int32:      123,
			Int64:      123,
			Uint:       123,
			Uint8:      123,
			Uint16:     123,
			Uint32:     123,
			Uint64:     123,
			Uintptr:    0,
			String:     "123ns",
		}},
		{time.Month(1), int(1), vars.KindInt, "January", 7, kindTest{
			Bool:       true,
			Float32:    1,
			Float64:    1,
			Complex64:  complex(1, 0),
			Complex128: complex(1, 0),
			Int:        1,
			Int8:       1,
			Int16:      1,
			Int32:      1,
			Int64:      1,
			Uint:       1,
			Uint8:      1,
			Uint16:     1,
			Uint32:     1,
			Uint64:     1,
			Uintptr:    1,
			String:     "1",
		}},
		{vars.Kind(26), uint(26), vars.KindUint, "unsafe.Pointer", 14, kindTest{
			Bool:       false,
			Float32:    26,
			Float64:    26,
			Complex64:  complex(26, 0),
			Complex128: complex(26, 0),
			Int:        26,
			Int8:       26,
			Int16:      26,
			Int32:      26,
			Int64:      26,
			Uint:       26,
			Uint8:      26,
			Uint16:     26,
			Uint32:     26,
			Uint64:     26,
			Uintptr:    26,
			String:     "26",
		}},
		{
			cbool(true), bool(true), vars.KindBool, "true", 4, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "true",
			},
		},
		{
			cbool(false), bool(false), vars.KindBool, "false", 5, kindTest{
				Bool:       false,
				Float32:    0,
				Float64:    0,
				Complex64:  complex(0, 0),
				Complex128: complex(0, 0),
				Int:        0,
				Int8:       0,
				Int16:      0,
				Int32:      0,
				Int64:      0,
				Uint:       0,
				Uint8:      0,
				Uint16:     0,
				Uint32:     0,
				Uint64:     0,
				Uintptr:    0,
				String:     "false",
			},
		},
		{
			cint(1), int(1), vars.KindInt, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cint8(1), int8(1), vars.KindInt8, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cint16(1), int16(1), vars.KindInt16, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cint32(1), int32(1), vars.KindInt32, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cint64(1), int64(1), vars.KindInt64, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cuint(1), uint(1), vars.KindUint, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cuint8(1), uint8(1), vars.KindUint8, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cuint16(1), uint16(1), vars.KindUint16, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cuint32(1), uint32(1), vars.KindUint32, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cuint64(1), uint64(1), vars.KindUint64, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cuintptr(1), uintptr(1), vars.KindUintptr, "1", 1, kindTest{
				Bool:       true,
				Float32:    1,
				Float64:    1,
				Complex64:  complex(1, 0),
				Complex128: complex(1, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       1,
				Uint8:      1,
				Uint16:     1,
				Uint32:     1,
				Uint64:     1,
				Uintptr:    1,
				String:     "1",
			},
		},
		{
			cfloat32(1.5), float32(1.5), vars.KindFloat32, "1.5", 3, kindTest{
				Bool:       false,
				Float32:    1.5,
				Float64:    1.5,
				Complex64:  complex(1.5, 0),
				Complex128: complex(1.5, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       0,
				Uint8:      0,
				Uint16:     0,
				Uint32:     0,
				Uint64:     0,
				Uintptr:    0,
				String:     "1.5",
			},
		},
		{
			cfloat64(1.5), float64(1.5), vars.KindFloat64, "1.5", 3, kindTest{
				Bool:       false,
				Float32:    1.5,
				Float64:    1.5,
				Complex64:  complex(1.5, 0),
				Complex128: complex(1.5, 0),
				Int:        1,
				Int8:       1,
				Int16:      1,
				Int32:      1,
				Int64:      1,
				Uint:       0,
				Uint8:      0,
				Uint16:     0,
				Uint32:     0,
				Uint64:     0,
				Uintptr:    0,
				String:     "1.5",
			},
		},
		{
			ccomplex64(complex(1.1, 2.5)), complex64(complex(1.1, 2.5)), vars.KindComplex64, "(1.1+2.5i)", 10, kindTest{
				Bool:       false,
				Float32:    0,
				Float64:    0,
				Complex64:  complex64(complex(1.1, 2.5)),
				Complex128: complex128(complex(1.1, 2.5)),
				Int:        0,
				Int8:       0,
				Int16:      0,
				Int32:      0,
				Int64:      0,
				Uint:       0,
				Uint8:      0,
				Uint16:     0,
				Uint32:     0,
				Uint64:     0,
				Uintptr:    0,
				String:     "(1.1+2.5i)",
			},
		},
		{
			ccomplex128(complex(1.1, 2.5)), complex128(complex(1.1, 2.5)), vars.KindComplex128, "(1.1+2.5i)", 10, kindTest{
				Bool:       false,
				Float32:    0,
				Float64:    0,
				Complex64:  complex64(complex(1.1, 2.5)),
				Complex128: complex128(complex(1.1, 2.5)),
				Int:        0,
				Int8:       0,
				Int16:      0,
				Int32:      0,
				Int64:      0,
				Uint:       0,
				Uint8:      0,
				Uint16:     0,
				Uint32:     0,
				Uint64:     0,
				Uintptr:    0,
				String:     "(1.1+2.5i)",
			},
		},
		{
			cstring("hello"), "hello", vars.KindString, "hello", 5, kindTest{
				Bool:       false,
				Float32:    0,
				Float64:    0,
				Complex64:  complex64(0),
				Complex128: complex128(0),
				Int:        0,
				Int8:       0,
				Int16:      0,
				Int32:      0,
				Int64:      0,
				Uint:       0,
				Uint8:      0,
				Uint16:     0,
				Uint32:     0,
				Uint64:     0,
				Uintptr:    0,
				String:     "hello",
			},
		},
		{
			cstring2("hello"), "hello", vars.KindString, "hello", 5, kindTest{
				Bool:       false,
				Float32:    0,
				Float64:    0,
				Complex64:  complex64(0),
				Complex128: complex128(0),
				Int:        0,
				Int8:       0,
				Int16:      0,
				Int32:      0,
				Int64:      0,
				Uint:       0,
				Uint8:      0,
				Uint16:     0,
				Uint32:     0,
				Uint64:     0,
				Uintptr:    0,
				String:     "hello",
			},
		},
		{
			cany(nil), nil, vars.KindInvalid, "<nil>", 5, kindTest{
				Bool:       false,
				Float32:    0,
				Float64:    0,
				Complex64:  complex64(0),
				Complex128: complex128(0),
				Int:        0,
				Int8:       0,
				Int16:      0,
				Int32:      0,
				Int64:      0,
				Uint:       0,
				Uint8:      0,
				Uint16:     0,
				Uint32:     0,
				Uint64:     0,
				Uintptr:    0,
				String:     "",
			},
		},
	}
	for _, test := range ctyps {
		t.Run(test.str, func(t *testing.T) {
			v, err := vars.NewValue(test.orig)
			if test.orig != nil {
				testutils.NoError(t, err)
			}
			testutils.Equal(t, test.typ.String(), v.Kind().String(), "type string")
			testutils.EqualAny(t, test.underlying, v.Any())
			testutils.Equal(t, test.str, v.String())

			t1, err := v.Bool()
			testutils.Equal(t, test.typtest.Bool, t1, "Bool")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t2, err := v.Float32()
			testutils.Equal(t, test.typtest.Float32, t2, "Float32")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t3, err := v.Float64()
			testutils.Equal(t, test.typtest.Float64, t3, "Float64")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t4, err := v.Complex64()
			testutils.Equal(t, test.typtest.Complex64, t4, "Complex64")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t4b, err := v.Complex128()
			testutils.Equal(t, test.typtest.Complex128, t4b, "Complex128")

			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t5, err := v.Int()
			testutils.Equal(t, test.typtest.Int, t5, "Int")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t6, err := v.Int8()
			testutils.Equal(t, test.typtest.Int8, t6, "Int8")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t7, err := v.Int16()
			testutils.Equal(t, test.typtest.Int16, t7, "Int16")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t8, err := v.Int32()
			testutils.Equal(t, test.typtest.Int32, t8, "Int32")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t9, err := v.Int64()
			testutils.Equal(t, test.typtest.Int64, t9, "Int64")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t10, err := v.Uint()
			testutils.Equal(t, test.typtest.Uint, t10, "Uint")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t11, err := v.Uint8()
			testutils.Equal(t, test.typtest.Uint8, t11, "Uint8")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t12, err := v.Uint16()
			testutils.Equal(t, test.typtest.Uint16, t12, "Uint16")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t13, err := v.Uint32()
			testutils.Equal(t, test.typtest.Uint32, t13, "Uint32")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t14, err := v.Uint64()
			testutils.Equal(t, test.typtest.Uint64, t14, "Uint64")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t15, err := v.Uintptr()
			testutils.Equal(t, test.typtest.Uintptr, t15, "Uintptr")
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t16, err := v.CloneAs(vars.KindString)
			testutils.Equal(t, test.typtest.String, t16.String())
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValue)
			}

			vv, err := vars.NewAs("test", test.orig, false, test.typ)
			if err == nil {
				testutils.NoError(t, err)
				testutils.Equal(t, test.typ, vv.Kind())
			}

			testutils.Equal(t, test.len, v.Len())
		})
	}
}
func TestVariableKinds(t *testing.T) {
	for _, test := range getKindTests() {
		v, err := vars.New(test.Key, test.In, false)
		testutils.NoError(t, err)
		testutils.Equal(t, test.Bool, v.Bool(), test.Key)
		testutils.Equal(t, test.Float32, v.Float32(), test.Key)
		testutils.Equal(t, test.Float64, v.Float64(), test.Key)
		testutils.Equal(t, test.Complex64, v.Complex64(), test.Key)
		testutils.Equal(t, test.Complex128, v.Complex128(), test.Key)
		testutils.Equal(t, test.Int, v.Int(), test.Key)
		testutils.Equal(t, test.Int8, v.Int8(), test.Key)
		testutils.Equal(t, test.Int16, v.Int16(), test.Key)
		testutils.Equal(t, test.Int32, v.Int32(), test.Key)
		testutils.Equal(t, test.Int64, v.Int64(), test.Key)
		testutils.Equal(t, test.Uint, v.Uint(), test.Key)
		testutils.Equal(t, test.Uint8, v.Uint8(), test.Key)
		testutils.Equal(t, test.Uint16, v.Uint16(), test.Key)
		testutils.Equal(t, test.Uint32, v.Uint32(), test.Key)
		testutils.Equal(t, test.Uint64, v.Uint64(), test.Key)
		testutils.Equal(t, test.Uintptr, v.Uintptr(), test.Key)
		testutils.Equal(t, test.String, v.String(), test.Key)
		// testutils.Equal(t, test.bytes, v.Bytes(), test.key)
		// testutils.Equal(t, test.runes, v.Runes(), test.key)
	}
}

func TestValueKinds(t *testing.T) {
	for _, test := range getKindTests() {
		emsg := testutils.KeyValErrorMsg(test.Key, test.In)

		v, err := vars.New("value-types", test.In, false)
		testutils.NoError(t, err, emsg)
		testutils.False(t, v.ReadOnly())
		vBool, err := v.Value().Bool()
		testutils.True(t, testutils.Equal(t, test.Bool, vBool, emsg) || err != nil)
		testutils.True(t, testutils.Equal(t, test.Bool, vBool, emsg) || err != nil)

		vFloat32, err := v.Value().Float32()
		testutils.True(t, testutils.Equal(t, test.Float32, vFloat32, emsg) || err != nil)

		vfFloat64, err := v.Value().Float64()
		testutils.True(t, testutils.Equal(t, test.Float64, vfFloat64, emsg) || err != nil)

		vComplex64, err := v.Value().Complex64()
		testutils.True(t, testutils.Equal(t, test.Complex64, vComplex64, emsg) || err != nil)

		vComplex128, err := v.Value().Complex128()
		testutils.True(t, testutils.Equal(t, test.Complex128, vComplex128, emsg) || err != nil)

		vInt, err := v.Value().Int()
		testutils.True(t, testutils.Equal(t, test.Int, vInt, emsg) || err != nil)

		vInt8, err := v.Value().Int8()
		testutils.True(t, testutils.Equal(t, test.Int8, vInt8, emsg) || err != nil)

		vInt16, err := v.Value().Int16()
		testutils.True(t, testutils.Equal(t, test.Int16, vInt16, emsg) || err != nil)

		vInt32, err := v.Value().Int32()
		testutils.True(t, testutils.Equal(t, test.Int32, vInt32, emsg) || err != nil)

		vInt64, err := v.Value().Int64()
		testutils.True(t, testutils.Equal(t, test.Int64, vInt64, emsg) || err != nil)

		vUint, err := v.Value().Uint()
		testutils.True(t, testutils.Equal(t, test.Uint, vUint, emsg) || err != nil)

		vUint8, err := v.Value().Uint8()
		testutils.True(t, testutils.Equal(t, test.Uint8, vUint8, emsg) || err != nil)

		vUint16, err := v.Value().Uint16()
		testutils.True(t, testutils.Equal(t, test.Uint16, vUint16, emsg) || err != nil)

		vUint32, err := v.Value().Uint32()
		testutils.True(t, testutils.Equal(t, test.Uint32, vUint32, emsg) || err != nil)

		vUint64, err := v.Value().Uint64()
		testutils.True(t, testutils.Equal(t, test.Uint64, vUint64, emsg) || err != nil)

		vUintptr, err := v.Value().Uintptr()
		testutils.True(t, testutils.Equal(t, test.Uintptr, vUintptr, emsg) || err != nil)

		testutils.Equal(t, test.String, v.String(), emsg)
		// testutils.Equal(t, test.bytes, v.Bytes(), test.key)
		// testutils.Equal(t, test.runes, v.Runes(), test.key)
	}
}

func TestInvalidKindString(t *testing.T) {
	testutils.Equal(t, "Kind(100)", vars.Kind(100).String())
}
