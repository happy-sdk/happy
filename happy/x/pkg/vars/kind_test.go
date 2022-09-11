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

package vars_test

import (
	"github.com/mkungla/happy/x/pkg/vars"
	"github.com/mkungla/happy/x/pkg/vars/internal/testutils"
	"testing"
	"time"
)

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
		typtest    testutils.KindTest
	}{
		{time.Duration(-123456), int64(-123456), vars.KindInt64, "-123.456µs", 11, testutils.KindTest{
			Bool:       false,
			Float32:    -123456,
			Float64:    -123456,
			Complex64:  complex(-123456, 0),
			Complex128: complex(-123456, 0),
			Int:        -123456,
			Int8:       0,
			Int16:      0,
			Int32:      -123456,
			Int64:      -123456,
			Uint:       0,
			Uint8:      0,
			Uint16:     0,
			Uint32:     0,
			Uint64:     0,
			Uintptr:    0,
			String:     "-123456",
		}},
		{time.Duration(123456), int64(123456), vars.KindInt64, "123.456µs", 10, testutils.KindTest{
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
			Uintptr:    123456,
			String:     "123456",
		}},
		{time.Duration(123), int64(123), vars.KindInt64, "123ns", 5, testutils.KindTest{
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
			Uintptr:    123,
			String:     "123",
		}},
		{time.Month(1), int(1), vars.KindInt, "January", 7, testutils.KindTest{
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
		{vars.Kind(26), uint(26), vars.KindUint, "unsafe.Pointer", 14, testutils.KindTest{
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
			cbool(true), bool(true), vars.KindBool, "true", 4, testutils.KindTest{
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
			cbool(false), bool(false), vars.KindBool, "false", 5, testutils.KindTest{
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
			cint(1), int(1), vars.KindInt, "1", 1, testutils.KindTest{
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
			cint8(1), int8(1), vars.KindInt8, "1", 1, testutils.KindTest{
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
			cint16(1), int16(1), vars.KindInt16, "1", 1, testutils.KindTest{
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
			cint32(1), int32(1), vars.KindInt32, "1", 1, testutils.KindTest{
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
			cint64(1), int64(1), vars.KindInt64, "1", 1, testutils.KindTest{
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
			cuint(1), uint(1), vars.KindUint, "1", 1, testutils.KindTest{
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
			cuint8(1), uint8(1), vars.KindUint8, "1", 1, testutils.KindTest{
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
			cuint16(1), uint16(1), vars.KindUint16, "1", 1, testutils.KindTest{
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
			cuint32(1), uint32(1), vars.KindUint32, "1", 1, testutils.KindTest{
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
			cuint64(1), uint64(1), vars.KindUint64, "1", 1, testutils.KindTest{
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
			cuintptr(1), uintptr(1), vars.KindUintptr, "1", 1, testutils.KindTest{
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
			cfloat32(1.5), float32(1.5), vars.KindFloat32, "1.5", 3, testutils.KindTest{
				Bool:       false,
				Float32:    1.5,
				Float64:    1.5,
				Complex64:  complex(1.5, 0),
				Complex128: complex(1.5, 0),
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
				String:     "1.5",
			},
		},
		{
			cfloat64(1.5), float64(1.5), vars.KindFloat64, "1.5", 3, testutils.KindTest{
				Bool:       false,
				Float32:    1.5,
				Float64:    1.5,
				Complex64:  complex(1.5, 0),
				Complex128: complex(1.5, 0),
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
				String:     "1.5",
			},
		},
		{
			ccomplex64(complex(1.1, 2.5)), complex64(complex(1.1, 2.5)), vars.KindComplex64, "(1.1+2.5i)", 10, testutils.KindTest{
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
			ccomplex128(complex(1.1, 2.5)), complex128(complex(1.1, 2.5)), vars.KindComplex128, "(1.1+2.5i)", 10, testutils.KindTest{
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
			cstring("hello"), "hello", vars.KindString, "hello", 5, testutils.KindTest{
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
			cstring2("hello"), "hello", vars.KindString, "hello", 5, testutils.KindTest{
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
			cany(nil), nil, vars.KindInvalid, "<nil>", 5, testutils.KindTest{
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
			testutils.Equal(t, test.typ.String(), v.Kind().String())
			testutils.EqualAny(t, test.underlying, v.Underlying())
			testutils.Equal(t, test.str, v.String())

			t1, err := v.Bool()
			testutils.Equal(t, test.typtest.Bool, t1)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t2, err := v.Float32()
			testutils.Equal(t, test.typtest.Float32, t2)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t3, err := v.Float64()
			testutils.Equal(t, test.typtest.Float64, t3)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t4, err := v.Complex64()
			testutils.Equal(t, test.typtest.Complex64, t4)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t4b, err := v.Complex128()
			testutils.Equal(t, test.typtest.Complex128, t4b)

			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t5, err := v.Int()
			testutils.Equal(t, test.typtest.Int, t5)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t6, err := v.Int8()
			testutils.Equal(t, test.typtest.Int8, t6)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t7, err := v.Int16()
			testutils.Equal(t, test.typtest.Int16, t7)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t8, err := v.Int32()
			testutils.Equal(t, test.typtest.Int32, t8)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t9, err := v.Int64()
			testutils.Equal(t, test.typtest.Int64, t9)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t10, err := v.Uint()
			testutils.Equal(t, test.typtest.Uint, t10)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t11, err := v.Uint8()
			testutils.Equal(t, test.typtest.Uint8, t11)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t12, err := v.Uint16()
			testutils.Equal(t, test.typtest.Uint16, t12)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t13, err := v.Uint32()
			testutils.Equal(t, test.typtest.Uint32, t13)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t14, err := v.Uint64()
			testutils.Equal(t, test.typtest.Uint64, t14)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t15, err := v.Uintptr()
			testutils.Equal(t, test.typtest.Uintptr, t15)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValueConv)
			}

			t16, err := v.CloneAs(vars.KindString)
			testutils.Equal(t, test.typtest.String, t16.String())
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValue)
			}

			vv, err := vars.NewVariableAs("test", test.orig, false, test.typ)
			if err == nil {
				testutils.NoError(t, err)
				testutils.Equal(t, test.typ, vv.Kind())
			}

			testutils.Equal(t, test.len, v.Len())
		})
	}
}

func TestVariableKinds(t *testing.T) {
	for _, test := range testutils.GetKindTests() {
		v, err := vars.NewVariable(test.Key, test.In, false)
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
	for _, test := range testutils.GetKindTests() {
		emsg := testutils.OnErrorMsg(test.Key, test.In)

		v, err := vars.NewVariable("value-types", test.In, false)
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
