// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestValueNew(t *testing.T) {
	for _, test := range newTests {
		v, _ := ParseValue(test.val)
		want := fmt.Sprintf("%v", test.val)
		if v.String() != want {
			t.Errorf("New %q, %v: expected value.String to return %q got %q", test.key, test.val, want, v.String())
			continue
		}
	}
}

// func (v Value) Bool() bool
func TestValueBool(t *testing.T) {
	for _, test := range boolTests {
		v := NewValue(test.in)
		if v.Bool() != test.want {
			t.Errorf("TestBool(%s): expected %t got %t", test.in, test.want, v.Bool())
			continue
		}

		vt, err := NewTypedValue(test.in, TypeBool)
		if v.Bool() != vt.Bool() {
			t.Errorf("TestBool(%s): expected New and NewTyped to return same values got: %t %t", test.in, v.Bool(), vt.Bool())
			continue
		}

		if !errors.Is(err, test.err) {
			t.Errorf("TestBool(%s): expected err to equal %#v got %#v", test.in, test.err, err)
			continue
		}

		vp, _ := ParseValue(test.in)
		if vp.Bool() != test.want {
			t.Errorf("TestBool(%s, %s): expected %t got %t", test.key, test.in, test.want, vp.Bool())
			continue
		}
	}
}

// func (v Value) Float32() float32
func TestValueFloat32(t *testing.T) {
	for _, test := range float32Tests {
		v := NewValue(test.in)
		if v.Float32() != test.wantFloat32 {
			t.Errorf("TestFloat32(%s, %s): expected %v got %v", test.key, test.in, test.wantFloat32, v.Float32())
			continue
		}
		vt, err := NewTypedValue(test.in, TypeFloat32)
		if vt.String() != test.wantStr {
			t.Errorf("TestFloat32(%s, %s): expected %q got %q", test.key, test.in, test.wantStr, vt.String())
			continue
		}
		if vt.Float32() != test.wantFloat32 {
			t.Errorf("TestFloat32(%s, %s): expected %q got %q", test.key, test.in, test.wantStr, vt.String())
			continue
		}
		if !errors.Is(err, test.wantErr) {
			t.Errorf("TestFloat32(%s, %s): expected err to equal %#v got %#v", test.key, test.in, test.wantErr, err)
			continue
		}

		vp, _ := ParseValue(test.in)
		if vp.Float32() != test.wantFloat32 {
			t.Errorf("TestFloat32(%s, %s): expected %q got %f", test.key, test.in, test.wantStr, vp.Float32())
			continue
		}
	}
}

// func (v Value) Float64() float64
func TestValueFloat64(t *testing.T) {
	for _, test := range float64Tests {
		v := NewValue(test.in)
		if v.Float64() != test.wantFloat64 {
			if test.wantStr == "NaN" && math.IsNaN(v.Float64()) {
				continue
			}
			t.Errorf("TestFloat64(%s, %s): expected %v got %v", test.key, test.in, test.wantFloat64, v.Float64())
			continue
		}
		vt, err := NewTypedValue(test.in, TypeFloat64)
		if vt.String() != test.wantStr {
			t.Errorf("TestFloat64(%s, %s): expected %q got %q", test.key, test.in, test.wantStr, vt.String())
			continue
		}
		if vt.Float64() != test.wantFloat64 {
			t.Errorf("TestFloat32(%s, %s): expected %q got %q", test.key, test.in, test.wantStr, vt.String())
			continue
		}
		if !errors.Is(err, test.wantErr) {
			t.Errorf("TestFloat64(%s, %s): expected err to equal %v got %v", test.key, test.in, test.wantErr, err)
			continue
		}

		vp, _ := ParseValue(test.in)
		if vp.Float64() != test.wantFloat64 {
			t.Errorf("TestFloat64(%s, %s): expected %q got %f", test.key, test.in, test.wantStr, vp.Float64())
			continue
		}
	}
}

// func (v Value) Complex64() complex64
func TestValueComplex64(t *testing.T) {
	for _, test := range complex64Tests {
		v := NewValue(test.in)
		if v.Complex64() != test.wantComplex64 {
			t.Errorf("TestComplex64(%s, %s): expected %v got %v", test.key, test.in, test.wantComplex64, v.Complex64())
			continue
		}
		vt, err := NewTypedValue(test.in, TypeComplex64)
		if vt.String() != test.wantStr {
			t.Errorf("TestComplex64(%s, %s): expected %q got %q", test.key, test.in, test.wantStr, vt.String())
			continue
		}
		if vt.Complex64() != test.wantComplex64 {
			t.Errorf("TestComplex64(%s, %s): expected %q got %q", test.key, test.in, test.wantStr, vt.String())
			continue
		}

		if !errors.Is(err, test.wantErr) {
			t.Errorf("TestComplex64(%s, %s): expected err to equal %v got %v", test.key, test.in, test.wantErr, err)
			continue
		}

		vp, _ := ParseValue(test.in)
		if vp.Complex64() != test.wantComplex64 {
			t.Errorf("TestComplex64(%s, %s): expected %q got %v", test.key, test.in, test.wantStr, vp.Complex64())
			continue
		}
	}
}

// func (v Value) Complex128() complex128
func TestValueComplex128(t *testing.T) {
	for _, test := range complex128Tests {
		v := NewValue(test.in)
		if v.Complex128() != test.wantComplex128 {
			t.Errorf("TestComplex128(%s, %s): expected %v got %v", test.key, test.in, test.wantComplex128, v.Complex128())
			continue
		}
		vt, err := NewTypedValue(test.in, TypeComplex128)
		if vt.String() != test.wantStr {
			t.Errorf("TestComplex128(%s, %s): expected %q got %q", test.key, test.in, test.wantStr, vt.String())
			continue
		}
		if vt.Complex128() != test.wantComplex128 {
			t.Errorf("TestComplex128(%s, %s): expected %q got %q", test.key, test.in, test.wantStr, vt.String())
			continue
		}

		if !errors.Is(err, test.wantErr) {
			t.Errorf("TestComplex128(%s, %s): expected err to equal %v got %v", test.key, test.in, test.wantErr, err)
			continue
		}

		vp, _ := ParseValue(test.in)
		if vp.Complex128() != test.wantComplex128 {
			t.Errorf("TestComplex128(%s, %s): expected %q got %v", test.key, test.in, test.wantStr, vp.Complex128())
			continue
		}
	}
}

// func (v Value) Int?n() int?n
func TestValueInt(t *testing.T) {
	for _, test := range intTests {
		t.Run(fmt.Sprintf("%s(int): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt)
			checkIntString(t, int64(test.int), v.String())
			if v.Int() != test.int {
				t.Errorf("expected %d got %d", test.int, v.Int())
			}
			checkErrors(t, test.val, test.errs, errInt, err)

			vu := NewValue(test.val)
			if vu.Int() != test.int {
				t.Errorf("untyped expected %d got %d", test.int, vu.Int())
			}

			vp, _ := ParseValue(test.val)
			if vp.Int() != test.int {
				t.Errorf("untyped expected %d got %d", test.int, vp.Int())
			}
		})

		t.Run(fmt.Sprintf("%s(int8): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt8)
			checkIntString(t, int64(test.int8), v.String())
			if v.Int8() != test.int8 {
				t.Errorf("expected %d got %d", test.int8, v.Int8())
			}
			checkErrors(t, test.val, test.errs, errInt8, err)

			vu := NewValue(test.val)
			if vu.Int8() != test.int8 {
				t.Errorf("untyped expected %d got %d", test.int8, vu.Int8())
			}

			vp, _ := ParseValue(test.val)
			if vp.Int8() != test.int8 {
				t.Errorf("untyped expected %d got %d", test.int8, vp.Int8())
			}
		})

		t.Run(fmt.Sprintf("%s(int16): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt16)
			checkIntString(t, int64(test.int16), v.String())
			if v.Int16() != test.int16 {
				t.Errorf("expected %d got %d", test.int16, v.Int16())
			}
			checkErrors(t, test.val, test.errs, errInt16, err)

			vu := NewValue(test.val)
			if vu.Int16() != test.int16 {
				t.Errorf("untyped expected %d got %d", test.int16, vu.Int16())
			}

			vp, _ := ParseValue(test.val)
			if vp.Int16() != test.int16 {
				t.Errorf("untyped expected %d got %d", test.int16, vp.Int16())
			}
		})

		t.Run(fmt.Sprintf("%s(int32): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt32)
			checkIntString(t, int64(test.int32), v.String())
			if v.Int32() != test.int32 {
				t.Errorf("expected %d got %d", test.int32, v.Int32())
			}
			checkErrors(t, test.val, test.errs, errInt32, err)

			vu := NewValue(test.val)
			if vu.Int32() != test.int32 {
				t.Errorf("untyped expected %d got %d", test.int32, vu.Int32())
			}

			vp, _ := ParseValue(test.val)
			if vp.Int32() != test.int32 {
				t.Errorf("untyped expected %d got %d", test.int32, vp.Int32())
			}
		})
		t.Run(fmt.Sprintf("%s(int64): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeInt64)
			checkIntString(t, test.int64, v.String())
			if v.Int64() != test.int64 {
				t.Errorf("expected %d got %d", test.int64, v.Int64())
			}
			checkErrors(t, test.val, test.errs, errInt64, err)

			vu := NewValue(test.val)
			if vu.Int64() != test.int64 {
				t.Errorf("untyped expected %d got %d", test.int64, vu.Int64())
			}

			vp, _ := ParseValue(test.val)
			if vp.Int64() != test.int64 {
				t.Errorf("untyped expected %d got %d", test.int64, vp.Int64())
			}

			t2 := &testInt{
				key:   test.key,
				val:   test.val,
				int:   test.int,
				int8:  test.int8,
				int16: test.int16,
				int32: test.int32,
				int64: test.int64,
				errs:  test.errs,
			}
			vp1, _ := ParseValue(t2.val)
			if vp1.Int64() != t2.int64 {
				t.Errorf("untyped expected %d got %d", t2.int64, vp1.Int64())
			}
		})
	}
}

// func (v Value) UInt?n() uint?n
func TestValueUint(t *testing.T) {
	for _, test := range uintTests {
		t.Run(fmt.Sprintf("%s(uint): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint)
			checkUintString(t, uint64(test.uint), v.String())
			if v.Uint() != test.uint {
				t.Errorf("expected %d got %d", test.uint, v.Uint())
			}
			checkErrors(t, test.val, test.errs, errUint, err)

			vu := NewValue(test.val)
			if vu.Uint() != test.uint {
				t.Errorf("untyped expected %d got %d", test.uint, vu.Uint())
			}

			vp, _ := ParseValue(test.val)
			if vp.Uint() != test.uint {
				t.Errorf("untyped expected %d got %d", test.uint, vp.Uint())
			}
		})

		t.Run(fmt.Sprintf("%s(uint8): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint8)
			checkUintString(t, uint64(test.uint8), v.String())
			if v.Uint8() != test.uint8 {
				t.Errorf("expected %d got %d", test.uint8, v.Uint8())
			}
			checkErrors(t, test.val, test.errs, errUint8, err)

			vu := NewValue(test.val)
			if vu.Uint8() != test.uint8 {
				t.Errorf("untyped expected %d got %d", test.uint8, vu.Uint8())
			}

			vp, _ := ParseValue(test.val)
			if vp.Uint8() != test.uint8 {
				t.Errorf("untyped expected %d got %d", test.uint8, vp.Uint8())
			}
		})

		t.Run(fmt.Sprintf("%s(uint16): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint16)
			checkUintString(t, uint64(test.uint16), v.String())
			if v.Uint16() != test.uint16 {
				t.Errorf("expected %d got %d", test.uint16, v.Uint16())
			}
			checkErrors(t, test.val, test.errs, errUint16, err)

			vu := NewValue(test.val)
			if vu.Uint16() != test.uint16 {
				t.Errorf("untyped expected %d got %d", test.uint16, vu.Uint16())
			}

			vp, _ := ParseValue(test.val)
			if vp.Uint16() != test.uint16 {
				t.Errorf("untyped expected %d got %d", test.uint16, vp.Uint16())
			}
		})

		t.Run(fmt.Sprintf("%s(uint32): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint32)
			checkUintString(t, uint64(test.uint32), v.String())
			if v.Uint32() != test.uint32 {
				t.Errorf("expected %d got %d", test.uint32, v.Uint32())
			}
			checkErrors(t, test.val, test.errs, errUint32, err)

			vu := NewValue(test.val)
			if vu.Uint32() != test.uint32 {
				t.Errorf("untyped expected %d got %d", test.uint32, vu.Uint32())
			}

			vp, _ := ParseValue(test.val)
			if vp.Uint32() != test.uint32 {
				t.Errorf("untyped expected %d got %d", test.uint32, vp.Uint32())
			}
		})

		t.Run(fmt.Sprintf("%s(uint64): %q", test.key, test.val), func(t *testing.T) {
			v, err := NewTypedValue(test.val, TypeUint64)
			checkUintString(t, test.uint64, v.String())
			if v.Uint64() != test.uint64 {
				t.Errorf("expected %d got %d", test.uint64, v.Uint64())
			}
			checkErrors(t, test.val, test.errs, errUint64, err)

			vu := NewValue(test.val)
			if vu.Uint64() != test.uint64 {
				t.Errorf("untyped expected %d got %d", test.uint64, vu.Uint64())
			}

			vp, _ := ParseValue(test.val)
			if vp.Uint64() != test.uint64 {
				t.Errorf("untyped expected %d got %d", test.uint64, vp.Uint64())
			}
		})
	}
}

// func (v Value) Uintptr() uintptr
func TestValueUintptr(t *testing.T) {
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
		if v.Uintptr() != test.want {
			t.Errorf("expected %d got %d", test.want, v.Uintptr())
		}
		checkErrors(t, test.val, test.errs, errUintptr, err)

		vu := NewValue(test.val)
		if vu.Uintptr() != test.want {
			t.Errorf("untyped expected %d got %d", test.want, vu.Uintptr())
		}

		vp, _ := ParseValue(test.val)
		if vp.Uintptr() != test.want {
			t.Errorf("untyped expected %d got %d", test.want, vp.Uintptr())
		}
	}
}

// func (v Value) String() string {
func TestValueStringParse(t *testing.T) {
	for _, test := range stringTests {
		v, _ := ParseValue(test.val)
		if v.String() != test.val {
			t.Errorf("Parse %q, %q: expected value.String to return %q got %q", test.key, test.val, test.val, v.String())
			continue
		}
	}
}

func TestValueStringNew(t *testing.T) {
	for _, test := range stringTests {
		v := NewValue(test.val)
		if v.String() != test.val {
			t.Errorf("New %q, %q: expected value.String to return %q got %q", test.key, test.val, test.val, v.String())
			continue
		}
	}
}

func TestValueStringType(t *testing.T) {
	for _, test := range stringTests {
		v, _ := NewTypedValue(test.val, TypeString)
		if v.String() != test.val {
			t.Errorf("New %q, %q: expected value.String to return %q got %q", test.key, test.val, test.val, v.String())
			continue
		}
	}
}

// func (v Variable) Bytes() []byte {
// 	return []byte{0}
// }
//
// func (v Variable) Reflect() reflect.Value {
// 	return reflect.ValueOf(v.raw)
// }

func TestValueTypes(t *testing.T) {
	for _, test := range typeTests {
		v, _ := ParseValue(test.in)
		if v.Bool() != test.bool {
			t.Errorf("(%s).Bool: expected %v got %v", test.key, test.bool, v.Bool())
			continue
		}
		if v.Float32() != test.float32 {
			t.Errorf("(%s).Float32: expected %v got %v", test.key, test.float32, v.Float32())
			continue
		}
		if v.Float64() != test.float64 {
			t.Errorf("(%s).Float64: expected %v got %v", test.key, test.float64, v.Float64())
			continue
		}
		if v.Complex64() != test.complex64 {
			t.Errorf("(%s).Complex64: expected %v got %v", test.key, test.complex64, v.Complex64())
			continue
		}
		if v.Complex128() != test.complex128 {
			t.Errorf("(%s).Complex128: expected %v got %v", test.key, test.complex128, v.Complex128())
			continue
		}
		if v.Int() != test.int {
			t.Errorf("(%s).Int: expected %v got %v", test.key, test.int, v.Int())
			continue
		}
		if v.Int8() != test.int8 {
			t.Errorf("(%s).Int8: expected %v got %v", test.key, test.int8, v.Int8())
			continue
		}
		if v.Int16() != test.int16 {
			t.Errorf("(%s).Int16: expected %v got %v", test.key, test.int16, v.Int16())
			continue
		}
		if v.Int32() != test.int32 {
			t.Errorf("(%s).Int32: expected %v got %v", test.key, test.int32, v.Int32())
			continue
		}
		if v.Int64() != test.int64 {
			t.Errorf("(%s).Int64: expected %v got %v", test.key, test.int64, v.Int64())
			continue
		}
		if v.Uint() != test.uint {
			t.Errorf("(%s).Uint: expected %v got %v", test.key, test.uint, v.Uint())
			continue
		}
		if v.Uint8() != test.uint8 {
			t.Errorf("(%s).Uint8: expected %v got %v", test.key, test.uint8, v.Uint8())
			continue
		}
		if v.Uint16() != test.uint16 {
			t.Errorf("(%s).Uint16: expected %v got %v", test.key, test.uint16, v.Uint16())
			continue
		}
		if v.Uint32() != test.uint32 {
			t.Errorf("(%s).Uint32: expected %v got %v", test.key, test.uint32, v.Uint32())
			continue
		}
		if v.Uint64() != test.uint64 {
			t.Errorf("(%s).Uint64: expected %v got %v", test.key, test.uint64, v.Uint64())
			continue
		}
		if v.Uintptr() != test.uintptr {
			t.Errorf("(%s).Uintptr: expected %v got %v", test.key, test.uintptr, v.Uintptr())
			continue
		}
		if v.String() != test.string {
			t.Errorf("(%s).String: expected %v got %v", test.key, test.string, v.String())
			continue
		}
		if !eqBytes(v.Bytes(), test.bytes) {
			t.Errorf("(%s).Bytes: expected %v got %v", test.key, test.bytes, v.Bytes())
			continue
		}
		if !eqRunes(v.Runes(), test.runes) {
			t.Errorf("(%s).Runes: expected %v got %v", test.key, test.runes, v.Runes())
			continue
		}
	}
}

func TestValueLen(t *testing.T) {
	collection := ParseKeyValSlice([]string{})
	tests := []struct {
		k       string
		defVal  string
		wantLen int
	}{
		{"STRING", "one two", 2},
		{"STRING", "one two three four ", 4},
		{"STRING", " one two three four ", 4},
		{"STRING", "1 2 3 4 5 6 7 8.1", 8},
		{"STRING", "", 0},
	}
	for _, tt := range tests {
		val := collection.Get(tt.k, tt.defVal)
		actual := len(val.String())
		if actual != val.Len() {
			t.Errorf("Value.(%q).Len() len = %d, want %d", tt.k, actual, tt.wantLen)
		}
		if tt.defVal == "" && !val.Empty() {
			t.Errorf("Value.(%q).Empty() = %t for value(%q), want true", tt.k, val.Empty(), val.String())
		}
		if tt.defVal != "" && val.Empty() {
			t.Errorf("Value.(%q).Empty() = %t for value(%q), want true", tt.k, val.Empty(), val.String())
		}
	}
}
