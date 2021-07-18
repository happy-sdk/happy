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

func TestNewBool(t *testing.T) {
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

    v3, err := NewTyped(test.key, test.in, TypeBool)
    assert.Equal(t, test.err, err)
    if err == nil {
      assert.Equal(t, test.want, v3.Bool())
      if v3.Type() != TypeBool {
        t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeBool, v3.val.vtype)
      }
    }
	}
}

// func (v Value) Float32() float32
func TestNewFloat32(t *testing.T) {
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

    v3, err := NewTyped(test.key, test.in, TypeFloat32)
    assert.ErrorIs(t, err, test.err)
    if err == nil {
      assert.Equal(t, test.wantFloat32, v3.Float32())
      if v3.Type() != TypeFloat32 {
        t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeFloat32, v3.val.vtype)
      }
    }
	}
}

func TestNewFloat64(t *testing.T) {
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

    v2, err := NewValue(test.wantFloat64)
    assert.Equal(t, nil, err)
    assert.Equal(t, test.wantStr, v1.String(), test.key)
    assert.Equal(t, test.wantFloat64, v2.Float64())
    if v2.Type() != TypeFloat64 {
      t.Errorf("v.Type() != %#v actual: %v", TypeFloat64, v.Type())
    }

    v3, err := NewTyped(test.key, test.in, TypeFloat64)
    assert.ErrorIs(t, err, test.err)
    if err == nil {
      assert.Equal(t, test.wantFloat64, v3.Float64())
      if v3.Type() != TypeFloat64 {
        t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeFloat64, v3.val.vtype)
      }
    }
	}
}

func TestNewValueComplex64(t *testing.T) {
	for _, test := range complex64Tests {
		v, err := NewValue(test.in)
		assert.ErrorIs(t, err, nil, test.key)
		assert.Equal(t, test.wantComplex64, v.Complex64(), test.key)

		v2, err := NewTypedValue(test.in, TypeComplex64)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, v2.String(), test.key)
		assert.Equal(t, test.wantComplex64, v2.Complex64(), test.key)

    v3, err := NewTyped(test.key, test.in, TypeComplex64)
    assert.ErrorIs(t, err, test.err)
    if err == nil {
      assert.Equal(t, test.wantComplex64, v3.Complex64())
      if v3.Type() != TypeComplex64 {
        t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeComplex64, v3.val.vtype)
      }
    }
	}
}

func TestNewValueComplex128(t *testing.T) {
	for _, test := range complex128Tests {
		v, err := NewValue(test.in)
		assert.ErrorIs(t, err, nil, test.key)
		assert.Equal(t, test.wantComplex128, v.Complex128(), test.key)

		v2, err := NewTypedValue(test.in, TypeComplex128)
		assert.ErrorIs(t, err, test.err, test.key)
		assert.Equal(t, test.wantStr, v2.String(), test.key)
		assert.Equal(t, test.wantComplex128, v2.Complex128(), test.key)

    v3, err := NewTyped(test.key, test.in, TypeComplex128)
    assert.ErrorIs(t, err, test.err)
    if err == nil {
      assert.Equal(t, test.wantComplex128, v3.Complex128())
      if v3.Type() != TypeComplex128 {
        t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeComplex128, v3.val.vtype)
      }
    }
	}
}

func TestNewValueInt(t *testing.T) {
	for _, test := range intTests {
		t.Run(fmt.Sprintf("%s(int): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt)
			assert.Equal(t, fmt.Sprint(test.int), v.String(), err)
			assert.Equal(t, test.int, v.Int(), test.key)
			checkErrors(t, test.val, test.errs, errInt, err)

			vu, err := NewValue(test.val)
			assert.Equal(t, test.int, vu.Int(), test.key)
			assert.NoError(t, err, test.key)

      v3, err := NewTyped(test.key, test.val, TypeInt)
      if err == nil {
        assert.Equal(t, test.int, v3.Int())
        if v3.Type() != TypeInt {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeInt, v3.val.vtype)
        }
      }
		})

		t.Run(fmt.Sprintf("%s(int8): %q", test.key, test.val), func(t *testing.T) {
			v, _ := NewTypedValue(test.val, TypeInt8)
			assert.Equal(t, test.int8, v.Int8(), test.key)
			checkIntString(t, int64(test.int8), v.String())

			vu, _ := NewValue(test.val)
			assert.Equal(t, test.int8, vu.Int8(), test.key)

      v3, err := NewTyped(test.key, test.val, TypeInt8)
      if err == nil {
        assert.Equal(t, test.int8, v3.Int8())
        if v3.Type() != TypeInt8 {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeInt8, v3.val.vtype)
        }
      }
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

      v3, err := NewTyped(test.key, test.val, TypeInt16)
      if err == nil {
        assert.Equal(t, test.int16, v3.Int16())
        if v3.Type() != TypeInt16 {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeInt16, v3.val.vtype)
        }
      }
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

      v3, err := NewTyped(test.key, test.val, TypeInt32)
      if err == nil {
        assert.Equal(t, test.int32, v3.Int32())
        if v3.Type() != TypeInt32 {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeInt32, v3.val.vtype)
        }
      }
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

      v3, err := NewTyped(test.key, test.val, TypeInt64)
      if err == nil {
        assert.Equal(t, test.int64, v3.Int64())
        if v3.Type() != TypeInt64 {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeInt64, v3.val.vtype)
        }
      }
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

      v3, err := NewTyped(test.key, test.val, TypeUint)
      if err == nil {
        assert.Equal(t, test.uint, v3.Uint())
        if v3.Type() != TypeUint {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeUint, v3.val.vtype)
        }
      }
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

      v3, err := NewTyped(test.key, test.val, TypeUint8)
      if err == nil {
        assert.Equal(t, test.uint8, v3.Uint8())
        if v3.Type() != TypeUint8 {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeUint8, v3.val.vtype)
        }
      }
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

      v3, err := NewTyped(test.key, test.val, TypeUint8)
      if err == nil {
        assert.Equal(t, test.uint16, v3.Uint16())
        if v3.Type() != TypeUint8 {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeUint8, v3.val.vtype)
        }
      }
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

      v3, err := NewTyped(test.key, test.val, TypeUint32)
      if err == nil {
        assert.Equal(t, test.uint32, v3.Uint32())
        if v3.Type() != TypeUint32 {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeUint32, v3.val.vtype)
        }
      }
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

      v3, err := NewTyped(test.key, test.val, TypeUint64)
      if err == nil {
        assert.Equal(t, test.uint64, v3.Uint64())
        if v3.Type() != TypeUint64 {
          t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeUint64, v3.val.vtype)
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
		v, err := NewTypedValue(test.val, TypeUintptr)
		checkUintString(t, uint64(test.want), v.String())
		assert.Equal(t, test.want, v.Uintptr())
		checkErrors(t, test.val, test.errs, errUintptr, err)

		vu, err := NewValue(test.val)
		assert.Equal(t, test.want, vu.Uintptr(), test.key)
		assert.NoError(t, err, test.key)

    v3, err := NewTyped(test.key, test.val, TypeUintptr)
    if err == nil {
      assert.Equal(t, test.want, v3.Uintptr())
      if v3.Type() != TypeUintptr {
        t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeUintptr, v3.val.vtype)
      }
    }
	}
}

func TestNewValueString(t *testing.T) {
	for _, test := range stringTests {
		v, _ := NewValue(test.val)
		assert.Equal(t, test.val, v.String())

		v1, _ := NewTypedValue(test.val, TypeString)
		assert.Equal(t, test.val, v1.String(), test.key)

    v3, err := NewTyped(test.key, test.val, TypeString)
    if err == nil {
      assert.Equal(t, test.val, v3.String())
      if v3.Type() != TypeString {
        t.Errorf("v.Type() != vars.Type(%v) actual: %v", TypeString, v3.val.vtype)
      }
    }
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

func TestNewVariableValueTypeBool(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.bool)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeBool, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeFloat32(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.float32)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeFloat32, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeFloat64(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.float64)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeFloat64, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeComplex64(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.complex64)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeComplex64, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeComplex128(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.complex128)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeComplex128, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeInt(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.int)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeInt, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeInt8(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.int8)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeInt8, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeInt16(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.int16)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeInt16, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeInt32(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.int32)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeInt32, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeInt64(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.int64)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeInt64, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeUint(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.uint)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeUint, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeUint8(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.uint8)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeUint8, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeUint16(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.uint16)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeUint16, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeUint32(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.uint32)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeUint32, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeUint64(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.uint64)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeUint64, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeUintptr(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.uintptr)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeUintptr, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeString(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.string)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeString, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeBytes(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.bytes)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeBytes, v.Type(), test.key)
  }
}

func TestNewVariableValueTypeRunes(t *testing.T) {
  for _, test := range typeTests {
    v, err := New(test.key, test.runes)
    assert.Equal(t, err, nil, test.key)
    assert.Equal(t, TypeRunes, v.Type(), test.key)
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
		assert.Equal(t, nil, err, test.key)
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

func TestValueFields(t *testing.T) {
  v, err := NewValue("word1 word2 word3")
  assert.Equal(t, nil, err)
  if len(v.Fields()) != 3 {
    t.Error("len of fields should be 3")
  }
}
