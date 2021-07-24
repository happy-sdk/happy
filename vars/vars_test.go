// Copyright 2021 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars_test

import (
	"fmt"
	"math"
	"strconv"
	"testing"
	"unicode/utf8"

	"github.com/mkungla/vars/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	for _, test := range newTests {
		v, err := vars.New(test.key, test.val)
		want := fmt.Sprintf("%v", test.val)
		assert.Equal(t, want, v.String())
		assert.Equal(t, test.err, err)
	}
}

func TestNewBool(t *testing.T) {
	for _, test := range boolTests {
		v, err := vars.NewValue(test.in)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, test.in, v.String())
		assert.Equal(t, test.want, v.Bool())

		v1, err := vars.NewTypedValue(test.in, vars.TypeBool)
		assert.Equal(t, test.err, err)
		assert.Equal(t, test.want, v1.Bool())
		if v1.Type() != vars.TypeBool {
			t.Errorf("v.Type() != %#v actual: %v", vars.TypeBool, v1.Type())
		}

		v2, err := vars.NewValue(test.want)
		assert.Equal(t, nil, err)
		assert.Equal(t, test.want, v2.Bool())
		if v2.Type() != vars.TypeBool {
			t.Errorf("v.Type() != %#v actual: %v", vars.TypeBool, v2.Type())
		}

		v3, err := vars.NewTyped(test.key, test.in, vars.TypeBool)
		assert.Equal(t, test.err, err)
		if err == nil {
			assert.Equal(t, test.want, v3.Bool())
			if v3.Type() != vars.TypeBool {
				t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeBool, v3.Type())
			}
		}
	}
}

// func (v Value) Float32() float32
func TestNewFloat32(t *testing.T) {
	for _, test := range float32Tests {
		v, err := vars.NewValue(test.in)
		assert.ErrorIs(t, err, nil)
		assert.Equal(t, test.wantFloat32, v.Float32())

		v1, err := vars.NewTypedValue(test.in, vars.TypeFloat32)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, v1.String(), test.key)
		if v1.Type() != vars.TypeFloat32 {
			t.Errorf("v.Type() != %#v actual: %v", vars.TypeFloat32, v.Type())
		}

		v2, err := vars.NewValue(test.wantFloat32)
		assert.Equal(t, nil, err)
		assert.Equal(t, test.wantStr, v1.String(), test.key)
		assert.Equal(t, test.wantFloat32, v2.Float32())
		if v2.Type() != vars.TypeFloat32 {
			t.Errorf("v.Type() != %#v actual: %v", vars.TypeFloat32, v.Type())
		}

		v3, err := vars.NewTyped(test.key, test.in, vars.TypeFloat32)
		assert.ErrorIs(t, err, test.err)
		if err == nil {
			assert.Equal(t, test.wantFloat32, v3.Float32())
			if v3.Type() != vars.TypeFloat32 {
				t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeFloat32, v3.Type())
			}
		}
	}
}

func TestNewFloat64(t *testing.T) {
	for _, test := range float64Tests {
		v, err := vars.NewValue(test.in)
		assert.ErrorIs(t, err, nil)
		if v.Float64() != test.wantFloat64 {
			if test.wantStr == "NaN" && math.IsNaN(v.Float64()) {
				continue
			}
			assert.Equal(t, test.wantFloat64, v.Float64())
		}

		v1, err := vars.NewTypedValue(test.in, vars.TypeFloat64)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, v1.String(), test.key)
		assert.Equal(t, test.wantFloat64, v1.Float64(), test.key)
		if v1.Type() != vars.TypeFloat64 {
			t.Errorf("v.Type() != %#v actual: %v", vars.TypeFloat64, v.Type())
		}

		v2, err := vars.NewValue(test.wantFloat64)
		assert.Equal(t, nil, err)
		assert.Equal(t, test.wantStr, v1.String(), test.key)
		assert.Equal(t, test.wantFloat64, v2.Float64())
		if v2.Type() != vars.TypeFloat64 {
			t.Errorf("v.Type() != %#v actual: %v", vars.TypeFloat64, v.Type())
		}

		v3, err := vars.NewTyped(test.key, test.in, vars.TypeFloat64)
		assert.ErrorIs(t, err, test.err)
		if err == nil {
			assert.Equal(t, test.wantFloat64, v3.Float64())
			if v3.Type() != vars.TypeFloat64 {
				t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeFloat64, v3.Type())
			}
		}
	}
}

func TestNewValueComplex64(t *testing.T) {
	for _, test := range complex64Tests {
		v, err := vars.NewValue(test.in)
		assert.ErrorIs(t, err, nil, test.key)
		assert.Equal(t, test.wantComplex64, v.Complex64(), test.key)

		v2, err := vars.NewTypedValue(test.in, vars.TypeComplex64)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, v2.String(), test.key)
		assert.Equal(t, test.wantComplex64, v2.Complex64(), test.key)

		v3, err := vars.NewTyped(test.key, test.in, vars.TypeComplex64)
		assert.ErrorIs(t, err, test.err)
		if err == nil {
			assert.Equal(t, test.wantComplex64, v3.Complex64())
			if v3.Type() != vars.TypeComplex64 {
				t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeComplex64, v3.Type())
			}
		}
	}
}

func TestNewValueComplex128(t *testing.T) {
	for _, test := range complex128Tests {
		v, err := vars.NewValue(test.in)
		assert.ErrorIs(t, err, nil, test.key)
		assert.Equal(t, test.wantComplex128, v.Complex128(), test.key)

		v2, err := vars.NewTypedValue(test.in, vars.TypeComplex128)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, v2.String(), test.key)
		assert.Equal(t, test.wantComplex128, v2.Complex128(), test.key)

		v3, err := vars.NewTyped(test.key, test.in, vars.TypeComplex128)
		assert.ErrorIs(t, err, test.err)
		if err == nil {
			assert.Equal(t, test.wantComplex128, v3.Complex128())
			if v3.Type() != vars.TypeComplex128 {
				t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeComplex128, v3.Type())
			}
		}
	}
}

func TestNewValueInt(t *testing.T) {
	for _, test := range intTests {
		t.Run(fmt.Sprintf("%s(int): %q", test.key, test.val), func(t *testing.T) {
			v, err := vars.NewTypedValue(test.val, vars.TypeInt)
			assert.Equal(t, fmt.Sprint(test.int), v.String(), err)
			assert.Equal(t, test.int, v.Int(), test.key)
			checkErrors(t, test.val, test.errs, errInt, err)

			vu, err := vars.NewValue(test.val)
			assert.Equal(t, test.int, vu.Int(), test.key)
			assert.NoError(t, err, test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeInt)
			if err == nil {
				assert.Equal(t, test.int, v3.Int())
				if v3.Type() != vars.TypeInt {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeInt, v3.Type())
				}
			}
		})

		t.Run(fmt.Sprintf("%s(int8): %q", test.key, test.val), func(t *testing.T) {
			v, _ := vars.NewTypedValue(test.val, vars.TypeInt8)
			assert.Equal(t, test.int8, v.Int8(), test.key)
			checkIntString(t, int64(test.int8), v.String())

			vu, _ := vars.NewValue(test.val)
			assert.Equal(t, test.int8, vu.Int8(), test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeInt8)
			if err == nil {
				assert.Equal(t, test.int8, v3.Int8())
				if v3.Type() != vars.TypeInt8 {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeInt8, v3.Type())
				}
			}
		})

		t.Run(fmt.Sprintf("%s(int16): %q", test.key, test.val), func(t *testing.T) {
			v, err := vars.NewTypedValue(test.val, vars.TypeInt16)
			checkIntString(t, int64(test.int16), v.String())
			checkErrors(t, test.val, test.errs, errInt16, err)

			assert.Equal(t, fmt.Sprint(test.int16), v.String(), err)
			assert.Equal(t, test.int16, v.Int16(), test.key)

			vu, err := vars.NewValue(test.val)
			assert.Equal(t, test.int16, vu.Int16(), test.key)
			assert.NoError(t, err, test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeInt16)
			if err == nil {
				assert.Equal(t, test.int16, v3.Int16())
				if v3.Type() != vars.TypeInt16 {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeInt16, v3.Type())
				}
			}
		})

		t.Run(fmt.Sprintf("%s(int32): %q", test.key, test.val), func(t *testing.T) {
			v, err := vars.NewTypedValue(test.val, vars.TypeInt32)
			checkIntString(t, int64(test.int32), v.String())
			checkErrors(t, test.val, test.errs, errInt32, err)

			assert.Equal(t, fmt.Sprint(test.int32), v.String(), err)
			assert.Equal(t, test.int32, v.Int32(), test.key)

			vu, err := vars.NewValue(test.val)
			assert.Equal(t, test.int32, vu.Int32(), test.key)
			assert.NoError(t, err, test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeInt32)
			if err == nil {
				assert.Equal(t, test.int32, v3.Int32())
				if v3.Type() != vars.TypeInt32 {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeInt32, v3.Type())
				}
			}
		})

		t.Run(fmt.Sprintf("%s(int64): %q", test.key, test.val), func(t *testing.T) {
			v, err := vars.NewTypedValue(test.val, vars.TypeInt64)
			checkIntString(t, test.int64, v.String())
			checkErrors(t, test.val, test.errs, errInt64, err)

			assert.Equal(t, fmt.Sprint(test.int64), v.String(), err)
			assert.Equal(t, test.int64, v.Int64(), test.key)

			vu, err := vars.NewValue(test.val)
			assert.Equal(t, test.int64, vu.Int64(), test.key)
			assert.NoError(t, err, test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeInt64)
			if err == nil {
				assert.Equal(t, test.int64, v3.Int64())
				if v3.Type() != vars.TypeInt64 {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeInt64, v3.Type())
				}
			}
		})
	}
}

func TestNewValueUint(t *testing.T) {
	for _, test := range uintTests {
		t.Run(fmt.Sprintf("%s(uint): %q", test.key, test.val), func(t *testing.T) {
			v, err := vars.NewTypedValue(test.val, vars.TypeUint)
			checkUintString(t, uint64(test.uint), v.String())
			checkErrors(t, test.val, test.errs, errUint, err)

			assert.Equal(t, fmt.Sprint(test.uint), v.String(), err)
			assert.Equal(t, test.uint, v.Uint(), test.key)

			vu, err := vars.NewValue(test.val)
			assert.Equal(t, test.uint, vu.Uint(), test.key)
			assert.NoError(t, err, test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeUint)
			if err == nil {
				assert.Equal(t, test.uint, v3.Uint())
				if v3.Type() != vars.TypeUint {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeUint, v3.Type())
				}
			}
		})

		t.Run(fmt.Sprintf("%s(uint8): %q", test.key, test.val), func(t *testing.T) {
			v, err := vars.NewTypedValue(test.val, vars.TypeUint8)
			checkUintString(t, uint64(test.uint8), v.String())
			checkErrors(t, test.val, test.errs, errUint8, err)

			assert.Equal(t, fmt.Sprint(test.uint8), v.String(), err)
			assert.Equal(t, test.uint8, v.Uint8(), test.key)

			vu, err := vars.NewValue(test.val)
			assert.Equal(t, test.uint8, vu.Uint8(), test.key)
			assert.NoError(t, err, test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeUint8)
			if err == nil {
				assert.Equal(t, test.uint8, v3.Uint8())
				if v3.Type() != vars.TypeUint8 {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeUint8, v3.Type())
				}
			}
		})

		t.Run(fmt.Sprintf("%s(uint16): %q", test.key, test.val), func(t *testing.T) {
			v, err := vars.NewTypedValue(test.val, vars.TypeUint16)
			checkUintString(t, uint64(test.uint16), v.String())
			checkErrors(t, test.val, test.errs, errUint16, err)

			assert.Equal(t, fmt.Sprint(test.uint16), v.String(), err)
			assert.Equal(t, test.uint16, v.Uint16(), test.key)

			vu, err := vars.NewValue(test.val)
			assert.Equal(t, test.uint16, vu.Uint16(), test.key)
			assert.NoError(t, err, test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeUint16)
			if err == nil {
				assert.Equal(t, test.uint16, v3.Uint16())
				if v3.Type() != vars.TypeUint16 {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeUint16, v3.Type())
				}
			}
		})

		t.Run(fmt.Sprintf("%s(uint32): %q", test.key, test.val), func(t *testing.T) {
			v, err := vars.NewTypedValue(test.val, vars.TypeUint32)
			checkUintString(t, uint64(test.uint32), v.String())
			checkErrors(t, test.val, test.errs, errUint32, err)

			assert.Equal(t, fmt.Sprint(test.uint32), v.String(), err)
			assert.Equal(t, test.uint32, v.Uint32(), test.key)

			vu, err := vars.NewValue(test.val)
			assert.Equal(t, test.uint32, vu.Uint32(), test.key)
			assert.NoError(t, err, test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeUint32)
			if err == nil {
				assert.Equal(t, test.uint32, v3.Uint32())
				if v3.Type() != vars.TypeUint32 {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeUint32, v3.Type())
				}
			}
		})

		t.Run(fmt.Sprintf("%s(uint64): %q", test.key, test.val), func(t *testing.T) {
			v, err := vars.NewTypedValue(test.val, vars.TypeUint64)
			checkUintString(t, test.uint64, v.String())
			checkErrors(t, test.val, test.errs, errUint64, err)

			assert.Equal(t, fmt.Sprint(test.uint64), v.String(), err)
			assert.Equal(t, test.uint64, v.Uint64(), test.key)

			vu, err := vars.NewValue(test.val)
			assert.Equal(t, test.uint64, vu.Uint64(), test.key)
			assert.NoError(t, err, test.key)

			v3, err := vars.NewTyped(test.key, test.val, vars.TypeUint64)
			if err == nil {
				assert.Equal(t, test.uint64, v3.Uint64())
				if v3.Type() != vars.TypeUint64 {
					t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeUint64, v3.Type())
				}
			}
		})
	}
}

func TestNewValueUintptr(t *testing.T) {
	tests := []struct {
		key  string
		val  string
		want uintptr
		errs uint
	}{
		{"UINTPTR_1", "1", 1, 0},
		{"UINTPTR_2", "2", 2, 0},
		{"UINTPTR_3", "9000000000000000000", 9000000000000000000, 0},
	}
	for _, test := range tests {
		v, err := vars.NewTypedValue(test.val, vars.TypeUintptr)
		checkUintString(t, uint64(test.want), v.String())
		assert.Equal(t, test.want, v.Uintptr())
		checkErrors(t, test.val, test.errs, errUintptr, err)

		vu, err := vars.NewValue(test.val)
		assert.Equal(t, test.want, vu.Uintptr(), test.key)
		assert.NoError(t, err, test.key)

		v3, err := vars.NewTyped(test.key, test.val, vars.TypeUintptr)
		if err == nil {
			assert.Equal(t, test.want, v3.Uintptr())
			if v3.Type() != vars.TypeUintptr {
				t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeUintptr, v3.Type())
			}
		}
	}
}

func TestNewValueString(t *testing.T) {
	for _, test := range stringTests {
		v, _ := vars.NewValue(test.val)
		assert.Equal(t, test.val, v.String())

		v1, _ := vars.NewTypedValue(test.val, vars.TypeString)
		assert.Equal(t, test.val, v1.String(), test.key)

		v3, err := vars.NewTyped(test.key, test.val, vars.TypeString)
		if err == nil {
			assert.Equal(t, test.val, v3.String())
			if v3.Type() != vars.TypeString {
				t.Errorf("v.Type() != vars.Type(%v) actual: %v", vars.TypeString, v3.Type())
			}
		}
	}
}

func TestNewVariableTypes(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.in)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, test.bool, v.Bool(), test.key)
		assert.Equal(t, test.float32, v.Float32(), test.key)
		assert.Equal(t, test.float64, v.Float64(), test.key)
		assert.Equal(t, test.complex64, v.Complex64(), test.key)
		assert.Equal(t, test.complex128, v.Complex128(), test.key)
		assert.Equal(t, test.int, v.Int(), test.key)
		assert.Equal(t, test.int8, v.Int8(), test.key)
		assert.Equal(t, test.int16, v.Int16(), test.key)
		assert.Equal(t, test.int32, v.Int32(), test.key)
		assert.Equal(t, test.int64, v.Int64(), test.key)
		assert.Equal(t, test.uint, v.Uint(), test.key)
		assert.Equal(t, test.uint8, v.Uint8(), test.key)
		assert.Equal(t, test.uint16, v.Uint16(), test.key)
		assert.Equal(t, test.uint32, v.Uint32(), test.key)
		assert.Equal(t, test.uint64, v.Uint64(), test.key)
		assert.Equal(t, test.uintptr, v.Uintptr(), test.key)
		assert.Equal(t, test.string, v.String(), test.key)
		assert.Equal(t, test.bytes, v.Bytes(), test.key)
		assert.Equal(t, test.runes, v.Runes(), test.key)
	}
}

func TestNewVariableValueTypeBool(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.bool)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeBool, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeFloat32(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.float32)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeFloat32, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeFloat64(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.float64)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeFloat64, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeComplex64(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.complex64)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeComplex64, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeComplex128(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.complex128)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeComplex128, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeInt(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.int)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeInt, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeInt8(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.int8)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeInt8, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeInt16(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.int16)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeInt16, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeInt32(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.int32)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeInt32, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeInt64(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.int64)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeInt64, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeUint(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.uint)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeUint, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeUint8(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.uint8)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeUint8, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeUint16(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.uint16)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeUint16, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeUint32(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.uint32)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeUint32, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeUint64(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.uint64)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeUint64, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeUintptr(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.uintptr)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeUintptr, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeString(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.string)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeString, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeBytes(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.bytes)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeBytes, v.Type(), test.key)
	}
}

func TestNewVariableValueTypeRunes(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.New(test.key, test.runes)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, vars.TypeRunes, v.Type(), test.key)
		assert.Equal(t, test.runes, v.Runes(), test.key)
	}

	v2, err := vars.New("runes", []rune{utf8.RuneSelf + 1})
	assert.Equal(t, err, nil)
	assert.Equal(t, vars.TypeRunes, v2.Type())

	v3, err := vars.New("runes", []rune{utf8.UTFMax + 1})
	assert.Equal(t, err, nil)
	assert.Equal(t, vars.TypeRunes, v3.Type())
}

func TestValueTypes(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.NewValue(test.in)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, test.bool, v.Bool(), test.key)
		assert.Equal(t, test.float32, v.Float32(), test.key)
		assert.Equal(t, test.complex64, v.Complex64(), test.key)
		assert.Equal(t, test.complex128, v.Complex128(), test.key)
		assert.Equal(t, test.int, v.Int(), test.key)
		assert.Equal(t, test.int8, v.Int8(), test.key)
		assert.Equal(t, test.int16, v.Int16(), test.key)
		assert.Equal(t, test.int32, v.Int32(), test.key)
		assert.Equal(t, test.int64, v.Int64(), test.key)
		assert.Equal(t, test.uint, v.Uint(), test.key)
		assert.Equal(t, test.uint8, v.Uint8(), test.key)
		assert.Equal(t, test.uint16, v.Uint16(), test.key)
		assert.Equal(t, test.uint32, v.Uint32(), test.key)
		assert.Equal(t, test.uint64, v.Uint64(), test.key)
		assert.Equal(t, test.uintptr, v.Uintptr(), test.key)
		assert.Equal(t, test.string, v.String(), test.key)
		assert.Equal(t, test.bytes, v.Bytes(), test.key)
		assert.Equal(t, test.runes, v.Runes(), test.key)
	}
}

func TestNewFromKeyVal(t *testing.T) {
	v, err := vars.NewFromKeyVal("X=1")
	assert.Equal(t, "X", v.Key())
	assert.False(t, v.Empty())
	assert.Equal(t, 1, v.Int())
	assert.Equal(t, err, nil)
}

func TestNewFromKeyValEmpty(t *testing.T) {
	v, err := vars.NewFromKeyVal("")
	assert.True(t, v.Empty())
	assert.Error(t, err)
	assert.ErrorIs(t, err, vars.ErrVariableKeyEmpty)
}

func TestNewFromKeyValEmptyKey(t *testing.T) {
	_, err := vars.NewFromKeyVal("=val")
	assert.Error(t, err)
	assert.ErrorIs(t, err, vars.ErrVariableKeyEmpty)
}

func TestLen(t *testing.T) {
	for _, test := range typeTests {
		v, err := vars.NewValue(test.in)
		assert.Equal(t, nil, err, test.key)
		assert.Equal(t, len(v.String()), len(test.in), test.key)
		assert.Equal(t, v.Len(), len(test.in), test.key)

		v2, err := vars.New(test.key, test.in)
		assert.Equal(t, nil, err, test.key)
		assert.Equal(t, len(v2.String()), len(test.in), test.key)
		assert.Equal(t, v2.Len(), len(test.in), test.key)
	}
}

func TestValueFields(t *testing.T) {
	v, err := vars.NewValue("word1 word2 word3")
	assert.Equal(t, nil, err)
	if len(v.Fields()) != 3 {
		t.Error("len of fields should be 3")
	}
	v2, err := vars.New("fields", "word1 word2 word3")
	assert.Equal(t, nil, err)
	if len(v2.Fields()) != 3 {
		t.Error("len of fields should be 3")
	}
}

func TestArbitraryValues(t *testing.T) {
	v1, err := vars.New("map", make(map[int]string))
	assert.Equal(t, nil, err)
	assert.Equal(t, "map[]", v1.String())

	list := make(map[int]string)
	list[1] = "line1"
	v2, err := vars.New("map", list)
	assert.Equal(t, nil, err)
	assert.Equal(t, "map[1:line1]", v2.String())

	v3, err := vars.New("bytes", []byte{})
	assert.Equal(t, nil, err)
	assert.Equal(t, "[]", v3.String())

	v4, err := vars.New("bytes", []byte{1, 2, 3})
	assert.Equal(t, nil, err)
	assert.Equal(t, "[1 2 3]", v4.String())

	v5, err := vars.NewTyped("bytes", "[1 2 3]", vars.TypeBytes)
	assert.Equal(t, nil, err)
	assert.Equal(t, []byte{1, 2, 3}, v5.Bytes())

	_, err = vars.NewTyped("bytes", "[100 200 300]", vars.TypeBytes)
	assert.ErrorIs(t, err, strconv.ErrRange)

	v6, err := vars.New("struct", struct{}{})
	assert.Equal(t, nil, err)
	assert.Equal(t, "{}", v6.String())

	strfn := func() string { return "str" }
	v7, err := vars.New("map", strfn)
	assert.Equal(t, nil, err)
	assert.NotEmpty(t, v7.String())
	assert.Equal(t, vars.TypeUnknown, v7.Type())
}

func TestErrors(t *testing.T) {
	_, err := vars.NewTyped("", "", vars.TypeString)
	assert.ErrorIs(t, err, vars.ErrVariableKeyEmpty)
}

func TestRune(t *testing.T) {
	_, err := vars.New("rune", rune(0x81))
	assert.ErrorIs(t, err, nil)
}
