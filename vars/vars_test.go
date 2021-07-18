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
		assert.Equal(t, want, v.String())
		assert.Equal(t, test.err, err)
	}
}

func TestNewValueBool(t *testing.T) {
	for _, test := range boolTests {
		v, err := NewValue(test.in)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, test.in, v.String())
    assert.Equal(t, test.want, v.Bool())

		v1, err := NewTypedValue(test.in, TypeBool)
    assert.Equal(t, test.err, err)
    assert.Equal(t, test.want, v1.Bool())
    if v1.Type() != TypeBool {
      t.Errorf("v.Type() != %#v actual: %v", TypeBool, v.Type())
    }

    v2, err := NewValue(test.want)
    assert.Equal(t, nil, err)
    assert.Equal(t, test.want, v2.Bool())
    if v2.Type() != TypeBool {
      t.Errorf("v.Type() != %#v actual: %v", TypeBool, v.Type())
    }
	}
}

// func (v Value) Float32() float32
func TestNewValueFloat32(t *testing.T) {
	for _, test := range float32Tests {
		v, err := NewValue(test.in)
		assert.ErrorIs(t, err, nil)
		assert.Equal(t, test.wantFloat32, v.Float32())

		v1, err := NewTypedValue(test.in, TypeFloat32)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, v1.String(), test.key)
    if v1.Type() != TypeFloat32 {
      t.Errorf("v.Type() != %#v actual: %v", TypeFloat32, v.Type())
    }

    v2, err := NewValue(test.wantFloat32)
    assert.Equal(t, nil, err)
    assert.Equal(t, test.wantStr, v1.String(), test.key)
    assert.Equal(t, test.wantFloat32, v2.Float32())
    if v2.Type() != TypeFloat32 {
      t.Errorf("v.Type() != %#v actual: %v", TypeFloat32, v.Type())
    }
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
			assert.Equal(t, test.wantFloat64, v.Float64())
		}

		v1, err := NewTypedValue(test.in, TypeFloat64)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, v1.String(), test.key)
		assert.Equal(t, test.wantFloat64, v1.Float64(), test.key)
    if v1.Type() != TypeFloat64 {
      t.Errorf("v.Type() != %#v actual: %v", TypeFloat64, v.Type())
    }
	}
}

func TestNewValueComplex64(t *testing.T) {
	for _, test := range complex64Tests {
		v, err := NewValue(test.in)
		assert.ErrorIs(t, err, nil, test.key)
		assert.Equal(t, test.wantComplex64, v.Complex64(), test.key)

		vt, err := NewTypedValue(test.in, TypeComplex64)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, vt.String(), test.key)
		assert.Equal(t, test.wantComplex64, vt.Complex64(), test.key)
	}
}

func TestNewValueComplex128(t *testing.T) {
	for _, test := range complex128Tests {
		v, err := NewValue(test.in)
		assert.ErrorIs(t, err, nil, test.key)
		assert.Equal(t, test.wantComplex128, v.Complex128(), test.key)

		vt, err := NewTypedValue(test.in, TypeComplex128)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, vt.String(), test.key)
		assert.Equal(t, test.wantComplex128, vt.Complex128(), test.key)
	}
}

func TestNewValueInt(t *testing.T) {
	for _, test := range intTests {
		t.Run(fmt.Sprintf("%s(uint): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt)
			assert.Equal(t, fmt.Sprint(test.int), v.String(), err)
			assert.Equal(t, test.int, v.Int(), test.key)
			checkErrors(t, test.val, test.errs, errInt, err)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.int, vu.Int(), test.key)
			assert.NoError(t, err, test.key)
		})

		t.Run(fmt.Sprintf("%s(int8): %q", test.key, test.val), func(t *testing.T) {
			v, _ := NewTypedValue(test.val, TypeInt8)
			assert.Equal(t, test.int8, v.Int8(), test.key)
			checkIntString(t, int64(test.int8), v.String())

			vu, _ := NewValue(test.val)
			assert.Equal(t, test.int8, vu.Int8(), test.key)
		})

		t.Run(fmt.Sprintf("%s(int16): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt16)
			checkIntString(t, int64(test.int16), v.String())
			checkErrors(t, test.val, test.errs, errInt16, err)

			assert.Equal(t, fmt.Sprint(test.int16), v.String(), err)
			assert.Equal(t, test.int16, v.Int16(), test.key)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.int16, vu.Int16(), test.key)
			assert.NoError(t, err, test.key)
		})

		t.Run(fmt.Sprintf("%s(int32): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt32)
			checkIntString(t, int64(test.int32), v.String())
			checkErrors(t, test.val, test.errs, errInt32, err)

			assert.Equal(t, fmt.Sprint(test.int32), v.String(), err)
			assert.Equal(t, test.int32, v.Int32(), test.key)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.int32, vu.Int32(), test.key)
			assert.NoError(t, err, test.key)
		})

		t.Run(fmt.Sprintf("%s(int64): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt64)
			checkIntString(t, test.int64, v.String())
			checkErrors(t, test.val, test.errs, errInt64, err)

			assert.Equal(t, fmt.Sprint(test.int64), v.String(), err)
			assert.Equal(t, test.int64, v.Int64(), test.key)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.int64, vu.Int64(), test.key)
			assert.NoError(t, err, test.key)
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
			assert.Equal(t, test.uint, v.Uint(), test.key)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.uint, vu.Uint(), test.key)
			assert.NoError(t, err, test.key)
		})

		t.Run(fmt.Sprintf("%s(uint8): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint8)
			checkUintString(t, uint64(test.uint8), v.String())
			checkErrors(t, test.val, test.errs, errUint8, err)

			assert.Equal(t, fmt.Sprint(test.uint8), v.String(), err)
			assert.Equal(t, test.uint8, v.Uint8(), test.key)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.uint8, vu.Uint8(), test.key)
			assert.NoError(t, err, test.key)
		})

		t.Run(fmt.Sprintf("%s(uint16): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint16)
			checkUintString(t, uint64(test.uint16), v.String())
			checkErrors(t, test.val, test.errs, errUint16, err)

			assert.Equal(t, fmt.Sprint(test.uint16), v.String(), err)
			assert.Equal(t, test.uint16, v.Uint16(), test.key)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.uint16, vu.Uint16(), test.key)
			assert.NoError(t, err, test.key)
		})

		t.Run(fmt.Sprintf("%s(uint32): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint32)
			checkUintString(t, uint64(test.uint32), v.String())
			checkErrors(t, test.val, test.errs, errUint32, err)

			assert.Equal(t, fmt.Sprint(test.uint32), v.String(), err)
			assert.Equal(t, test.uint32, v.Uint32(), test.key)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.uint32, vu.Uint32(), test.key)
			assert.NoError(t, err, test.key)
		})

		t.Run(fmt.Sprintf("%s(uint64): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint64)
			checkUintString(t, test.uint64, v.String())
			checkErrors(t, test.val, test.errs, errUint64, err)

			assert.Equal(t, fmt.Sprint(test.uint64), v.String(), err)
			assert.Equal(t, test.uint64, v.Uint64(), test.key)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.uint64, vu.Uint64(), test.key)
			assert.NoError(t, err, test.key)
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
		assert.Equal(t, test.want, v.Uintptr())
		checkErrors(t, test.val, test.errs, errUintptr, err)

		vu, err := NewValue(test.val)
		assert.Equal(t, test.want, vu.Uintptr(), test.key)
		assert.NoError(t, err, test.key)
	}
}

func TestNewValueString(t *testing.T) {
	for _, test := range stringTests {
		v, _ := NewValue(test.val)
		assert.Equal(t, test.val, v.String())

		v1, _ := NewTypedValue(test.val, TypeString)
		assert.Equal(t, test.val, v1.String(), test.key)
	}
}

func TestNewVariableTypes(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.in)
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

func TestValueTypes(t *testing.T) {
  for _, test := range typeTests {
    v, err := NewValue(test.in)
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
	v, err := NewFromKeyVal("X=1")
	assert.Equal(t, "X", v.Key())
	assert.False(t, v.Empty())
	assert.Equal(t, 1, v.Int())
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

func TestValueLen(t *testing.T) {
	for _, test := range typeTests {
		v, err := NewValue(test.in)
		assert.Equal(t, err, nil, test.key)
		assert.Equal(t, len(v.String()), len(test.in), test.key)
		assert.Equal(t, v.Len(), len(test.in), test.key)
	}
}

func TestFlags(t *testing.T) {
  f := parserFmtFlags{}
	assert.False(t, f.widPresent)
	assert.False(t, f.precPresent)
	assert.False(t, f.minus)
	assert.False(t, f.plus)
	assert.False(t, f.sharp)
	assert.False(t, f.space)
	assert.False(t, f.zero)
	assert.False(t, f.plusV)
	assert.False(t, f.sharpV)
	assert.False(t, f.sharpV)
}
