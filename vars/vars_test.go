// Copyright 2021 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	for _, test := range newTests {
		v, err := New(test.key, test.val)
		want := fmt.Sprintf("%v", test.val)
		assert.Equal(t, v.String(), want)
		assert.Equal(t, err, test.err)
	}
}

func TestNewValueBool(t *testing.T) {
	for _, test := range boolTests {
		v, err := NewValue(test.in)
		assert.Equal(t, err, nil)
		assert.Equal(t, v.String(), test.in)

		vt, err := NewTypedValue(test.in, TypeBool)
		assert.ErrorIs(t, err, test.err)
		if v.Bool() != vt.Bool() {
			t.Errorf("TestBool(%s): expected New and NewTyped to return same values got: %t %t", test.in, v.Bool(), vt.Bool())
			continue
		}
	}
}

// func (v Value) Float32() float32
func TestNewValueFloat32(t *testing.T) {
	for _, test := range float32Tests {
		v, err := NewValue(test.in)
		assert.ErrorIs(t, err, nil)
		assert.Equal(t, v.Float32(), test.wantFloat32)

		vt, err := NewTypedValue(test.in, TypeFloat32)
		assert.ErrorIs(t, err, test.err)
		assert.Equal(t, vt.String(), test.wantStr)
	}
}

func TestNewValueFloat64(t *testing.T) {
	for _, test := range float64Tests {
		v, err := NewValue(test.in)
		assert.ErrorIs(t, err, nil)
		if v.Float64() != test.wantFloat64 {
			if test.wantStr == "NaN" && math.IsNaN(v.Float64()) {
				continue
			}
			assert.Equal(t, v.Float64(), test.wantFloat64)
		}

		vt, err := NewTypedValue(test.in, TypeFloat64)
		assert.ErrorIs(t, err, test.err)
		assert.Equal(t, vt.String(), test.wantStr)
		assert.Equal(t, vt.Float64(), test.wantFloat64)
	}
}

func TestNewValueComplex64(t *testing.T) {
	for _, test := range complex64Tests {
		v, err := NewValue(test.in)
		assert.ErrorIs(t, err, nil)
		assert.Equal(t, v.Complex64(), test.wantComplex64)

		vt, err := NewTypedValue(test.in, TypeComplex64)
		assert.ErrorIs(t, err, test.err)
		assert.Equal(t, vt.String(), test.wantStr)
		assert.Equal(t, vt.Complex64(), test.wantComplex64)
	}
}

func TestNewValueComplex128(t *testing.T) {
	for _, test := range complex128Tests {
		v, err := NewValue(test.in)
		assert.ErrorIs(t, err, nil)
		assert.Equal(t, v.Complex128(), test.wantComplex128)

		vt, err := NewTypedValue(test.in, TypeComplex128)
		assert.ErrorIs(t, err, test.err)
		assert.Equal(t, vt.String(), test.wantStr)
		assert.Equal(t, vt.Complex128(), test.wantComplex128)
	}
}

func TestNewValueInt(t *testing.T) {
	for _, test := range intTests {
		t.Run(fmt.Sprintf("%s(uint): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt)
			assert.Equal(t, fmt.Sprint(test.int), v.String(), err)
			assert.Equal(t, v.Int(), test.int)
			checkErrors(t, test.val, test.errs, errInt, err)

			vu, err := NewValue(test.val)
			assert.Equal(t, vu.Int(), test.int)
			assert.NoError(t, err)
		})

		t.Run(fmt.Sprintf("%s(int8): %q", test.key, test.val), func(t *testing.T) {
			v, _ := NewTypedValue(test.val, TypeInt8)
			assert.Equal(t, v.Int8(), test.int8)
			checkIntString(t, int64(test.int8), v.String())

			vu, _ := NewValue(test.val)
			assert.Equal(t, vu.Int8(), test.int8)
		})

		t.Run(fmt.Sprintf("%s(int16): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt16)
			checkIntString(t, int64(test.int16), v.String())
			checkErrors(t, test.val, test.errs, errInt16, err)

			assert.Equal(t, fmt.Sprint(test.int16), v.String(), err)
			assert.Equal(t, v.Int16(), test.int16)

			vu, err := NewValue(test.val)
			assert.Equal(t, vu.Int16(), test.int16)
			assert.NoError(t, err)
		})

		t.Run(fmt.Sprintf("%s(int32): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt32)
			checkIntString(t, int64(test.int32), v.String())
			checkErrors(t, test.val, test.errs, errInt32, err)

			assert.Equal(t, fmt.Sprint(test.int32), v.String(), err)
			assert.Equal(t, v.Int32(), test.int32)

			vu, err := NewValue(test.val)
			assert.Equal(t, vu.Int32(), test.int32)
			assert.NoError(t, err)
		})

		t.Run(fmt.Sprintf("%s(int64): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt64)
			checkIntString(t, test.int64, v.String())
			checkErrors(t, test.val, test.errs, errInt64, err)

			assert.Equal(t, fmt.Sprint(test.int64), v.String(), err)
			assert.Equal(t, v.Int64(), test.int64)

			vu, err := NewValue(test.val)
			assert.Equal(t, vu.Int64(), test.int64)
			assert.NoError(t, err)
		})
	}
}

func TestNewValueUint(t *testing.T) {
	for _, test := range uintTests {
		t.Run(fmt.Sprintf("%s(uint): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint)
			checkUintString(t, uint64(test.uint), v.String())
			checkErrors(t, test.val, test.errs, errUint, err)

			assert.Equal(t, fmt.Sprint(test.uint), v.String(), err)
			assert.Equal(t, v.Uint(), test.uint)

			vu, err := NewValue(test.val)
			assert.Equal(t, vu.Uint(), test.uint)
			assert.NoError(t, err)
		})

		t.Run(fmt.Sprintf("%s(uint8): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint8)
			checkUintString(t, uint64(test.uint8), v.String())
			checkErrors(t, test.val, test.errs, errUint8, err)

			assert.Equal(t, fmt.Sprint(test.uint8), v.String(), err)
			assert.Equal(t, v.Uint8(), test.uint8)

			vu, err := NewValue(test.val)
			assert.Equal(t, vu.Uint8(), test.uint8)
			assert.NoError(t, err)
		})

		t.Run(fmt.Sprintf("%s(uint16): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint16)
			checkUintString(t, uint64(test.uint16), v.String())
			checkErrors(t, test.val, test.errs, errUint16, err)

			assert.Equal(t, fmt.Sprint(test.uint16), v.String(), err)
			assert.Equal(t, v.Uint16(), test.uint16)

			vu, err := NewValue(test.val)
			assert.Equal(t, vu.Uint16(), test.uint16)
			assert.NoError(t, err)
		})

		t.Run(fmt.Sprintf("%s(uint32): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint32)
			checkUintString(t, uint64(test.uint32), v.String())
			checkErrors(t, test.val, test.errs, errUint32, err)

			assert.Equal(t, fmt.Sprint(test.uint32), v.String(), err)
			assert.Equal(t, v.Uint32(), test.uint32)

			vu, err := NewValue(test.val)
			assert.Equal(t, vu.Uint32(), test.uint32)
			assert.NoError(t, err)
		})

		t.Run(fmt.Sprintf("%s(uint64): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint64)
			checkUintString(t, test.uint64, v.String())
			checkErrors(t, test.val, test.errs, errUint64, err)

			assert.Equal(t, fmt.Sprint(test.uint64), v.String(), err)
			assert.Equal(t, v.Uint64(), test.uint64)

			vu, err := NewValue(test.val)
			assert.Equal(t, vu.Uint64(), test.uint64)
			assert.NoError(t, err)
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
		v, err := NewTypedValue(test.val, TypeUintptr)
		checkUintString(t, uint64(test.want), v.String())
		assert.Equal(t, v.Uintptr(), test.want)
		checkErrors(t, test.val, test.errs, errUintptr, err)

		vu, err := NewValue(test.val)
		assert.Equal(t, vu.Uintptr(), test.want)
		assert.NoError(t, err)
	}
}

func TestNewValueString(t *testing.T) {
	for _, test := range stringTests {
		v, _ := NewValue(test.val)
		assert.Equal(t, v.String(), test.val)

		v1, _ := NewTypedValue(test.val, TypeString)
		assert.Equal(t, v1.String(), test.val)
	}
}

func TestNewFromKeyVal(t *testing.T) {
	v, err := NewFromKeyVal("X=1")
	assert.Equal(t, v.Key(), "X")
	assert.False(t, v.Empty())
	assert.Equal(t, v.Int(), 1)
	assert.Equal(t, err, nil)
}

func TestNewFromKeyValEmpty(t *testing.T) {
	v, err := NewFromKeyVal("")
	assert.True(t, v.Empty())
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrVariableKeyEmpty)
}

func TestNewFromKeyValEmptyKey(t *testing.T) {
	_, err := NewFromKeyVal("=val")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrVariableKeyEmpty)
}
