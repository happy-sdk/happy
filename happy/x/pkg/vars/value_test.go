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
	"fmt"
	"github.com/mkungla/happy/x/pkg/vars"
	"github.com/mkungla/happy/x/pkg/vars/internal/testutils"
	"math"
	"testing"
)

func TestBoolValue(t *testing.T) {
	for _, test := range testutils.GetBoolTests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			testutils.Equal(t, test.In, v1.String())
			testutils.NoError(t, err)

			b1, err := v1.Bool()
			testutils.Equal(t, test.Want, b1)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, false, b1)
				testutils.Equal(t, vars.KindString, v1.Kind())
			} else {
				testutils.Equal(t, test.Want, b1)
				testutils.Equal(t, vars.KindString, v1.Kind())
			}

			v2, err := vars.NewValue(test.Want)
			testutils.NoError(t, err)
			b2, err := v2.Bool()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Want, b2)

			v3, err := vars.ParseValueAs(test.In, vars.KindBool)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, vars.KindInvalid, v3.Kind())
			} else {
				b3, err := v3.Bool()
				testutils.Equal(t, test.Want, b3)
				testutils.ErrorIs(t, err, test.Err)
				testutils.Equal(t, vars.KindBool, v3.Kind())
			}
			v4, err := vars.ParseVariableAs("var", test.In, false, vars.KindBool)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, vars.KindInvalid, v4.Kind())
			} else {
				testutils.Equal(t, vars.KindBool, v4.Kind())
				testutils.Equal(t, test.Want, v4.Bool())
			}
			v5, err := vars.NewVariable("var", v4, false)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValue)
				testutils.Equal(t, vars.KindInvalid, v5.Kind())
			} else {
				testutils.Equal(t, vars.KindBool, v5.Kind())
				testutils.Equal(t, test.Want, v5.Bool())
			}
			v6, err := vars.NewValue(v4)
			if err != nil {
				testutils.ErrorIs(t, err, vars.ErrValue)
				testutils.Equal(t, vars.KindInvalid, v6.Kind())
			} else {
				testutils.Equal(t, vars.KindBool, v6.Kind())
				b6, _ := v6.Bool()
				testutils.Equal(t, test.Want, b6)
			}
		})
	}
}

func TestFloat32Value(t *testing.T) {
	for _, test := range testutils.GetFloat32Tests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			testutils.Equal(t, test.In, v1.String())
			testutils.NoError(t, err)

			b1, err := v1.Float32()
			testutils.Equal(t, test.WantFloat32, b1)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, test.WantFloat32, b1)
				testutils.Equal(t, vars.KindString, v1.Kind())
			} else {
				testutils.Equal(t, test.WantFloat32, b1)
				testutils.Equal(t, vars.KindString, v1.Kind())
			}

			v2, err := vars.NewValue(test.WantFloat32)
			testutils.NoError(t, err)
			b2, err := v2.Float32()
			testutils.NoError(t, err)
			testutils.Equal(t, test.WantFloat32, b2)

			v3, err := vars.ParseValueAs(test.In, vars.KindFloat32)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, vars.KindInvalid, v3.Kind())
			} else {
				b3, err := v3.Float32()
				testutils.Equal(t, test.WantFloat32, b3)
				testutils.ErrorIs(t, err, test.Err)
				testutils.Equal(t, vars.KindFloat32, v3.Kind())
			}

			v4, err := vars.ParseVariableAs("var", test.In, false, vars.KindFloat32)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, vars.KindInvalid, v4.Kind())
			} else {
				testutils.Equal(t, vars.KindFloat32, v4.Kind())
				testutils.Equal(t, test.WantFloat32, v4.Float32())
				testutils.ErrorIs(t, err, test.Err)
			}
		})
	}
}

func TestFloat64Value(t *testing.T) {
	for _, test := range testutils.GetFloat64Tests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			testutils.Equal(t, test.In, v1.String())
			testutils.NoError(t, err)

			b1, err := v1.Float64()
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, test.WantFloat64, b1)
				testutils.Equal(t, vars.KindString, v1.Kind())
			} else {
				if !math.IsNaN(test.WantFloat64) {
					testutils.Equal(t, test.WantFloat64, b1)
				} else {
					testutils.Equal(t, math.IsNaN(test.WantFloat64), math.IsNaN(b1))
				}
				testutils.Equal(t, vars.KindString, v1.Kind())
			}

			v2, err := vars.NewValue(test.WantFloat64)
			testutils.NoError(t, err)
			b2, err := v2.Float64()
			testutils.NoError(t, err)

			if !math.IsNaN(test.WantFloat64) {
				testutils.Equal(t, test.WantFloat64, b2)
			} else {
				testutils.Equal(t, math.IsNaN(test.WantFloat64), math.IsNaN(b2))
			}

			v3, err := vars.ParseValueAs(test.In, vars.KindFloat64)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, vars.KindInvalid, v3.Kind())
			} else {
				b3, err := v3.Float64()
				testutils.ErrorIs(t, err, test.Err)
				if err != nil {
					testutils.Equal(t, float64(0), b3)
					testutils.Equal(t, vars.KindInvalid, v3.Kind())
				} else {
					if !math.IsNaN(test.WantFloat64) {
						testutils.Equal(t, test.WantFloat64, b3)
					} else {
						testutils.Equal(t, math.IsNaN(test.WantFloat64), math.IsNaN(b3))
					}

					testutils.Equal(t, vars.KindFloat64, v3.Kind())
				}
			}

			v4, err := vars.ParseVariableAs("var", test.In, false, vars.KindFloat64)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, vars.KindInvalid, v4.Kind())
			} else {
				testutils.Equal(t, vars.KindFloat64, v4.Kind())
				if !math.IsNaN(test.WantFloat64) {
					testutils.Equal(t, test.WantFloat64, v4.Float64())
				} else {
					testutils.Equal(t, math.IsNaN(test.WantFloat64), math.IsNaN(v4.Float64()))
				}
			}
		})
	}
}

func TestComplex64Value(t *testing.T) {
	for _, test := range testutils.GetComplex64Tests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			testutils.Equal(t, test.In, v1.String())
			testutils.NoError(t, err)

			b1, err := v1.Complex64()
			testutils.Equal(t, test.WantComplex64, b1)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, test.WantComplex64, b1)
				testutils.Equal(t, vars.KindString, v1.Kind())
			} else {
				testutils.Equal(t, test.WantComplex64, b1)
				testutils.Equal(t, vars.KindString, v1.Kind())
			}

			v2, err := vars.NewValue(test.WantComplex64)
			testutils.NoError(t, err)
			b2, err := v2.Complex64()
			testutils.NoError(t, err)
			testutils.Equal(t, test.WantComplex64, b2)

			v3, err := vars.ParseValueAs(test.In, vars.KindComplex64)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, vars.KindInvalid, v3.Kind())
			} else {
				b3, err := v3.Complex64()
				testutils.Equal(t, test.WantComplex64, b3)
				testutils.ErrorIs(t, err, test.Err)
				if err != nil {
					testutils.Equal(t, 0, b3)
					testutils.Equal(t, vars.KindInvalid, v3.Kind())
				} else {
					testutils.Equal(t, test.WantComplex64, b3)
					testutils.Equal(t, vars.KindComplex64, v3.Kind())
				}
			}
		})
	}
}

func TestComplex128Value(t *testing.T) {
	for _, test := range testutils.GetComplex128Tests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			testutils.Equal(t, test.In, v1.String())
			testutils.NoError(t, err)

			b1, err := v1.Complex128()
			testutils.Equal(t, test.WantComplex128, b1)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, test.WantComplex128, b1)
				testutils.Equal(t, vars.KindString, v1.Kind())
			} else {
				testutils.Equal(t, test.WantComplex128, b1)
				testutils.Equal(t, vars.KindString, v1.Kind())
			}

			v2, err := vars.NewValue(test.WantComplex128)
			testutils.NoError(t, err)
			b2, err := v2.Complex128()
			testutils.NoError(t, err)
			testutils.Equal(t, test.WantComplex128, b2)

			v3, err := vars.ParseValueAs(test.In, vars.KindComplex128)
			testutils.ErrorIs(t, err, test.Err)
			if err != nil {
				testutils.Equal(t, vars.KindInvalid, v3.Kind())
			} else {
				b3, err := v3.Complex128()
				testutils.Equal(t, test.WantComplex128, b3)
				testutils.ErrorIs(t, err, test.Err)
				if err != nil {
					testutils.Equal(t, 0, b3)
					testutils.Equal(t, vars.KindInvalid, v3.Kind())
				} else {
					testutils.Equal(t, test.WantComplex128, b3)
					testutils.Equal(t, vars.KindComplex128, v3.Kind())
				}
			}
		})
	}
}

func TestIntValue(t *testing.T) {
	for _, test := range testutils.GetIntTests() {
		t.Run(fmt.Sprintf("%s(int): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int()
			testutils.Equal(t, test.Int, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt, err))
			i2, err := v2.Int()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Int, i2)
		})

		t.Run(fmt.Sprintf("%s(int8): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int8()
			testutils.Equal(t, test.Int8, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt8, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt8)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt8, err))
			i2, err := v2.Int8()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Int8, i2)
		})

		t.Run(fmt.Sprintf("%s(int16): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int16()
			testutils.Equal(t, test.Int16, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt16, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt16)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt16, err))
			i2, err := v2.Int16()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Int16, i2)
		})

		t.Run(fmt.Sprintf("%s(int32): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int32()
			testutils.Equal(t, test.Int32, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt32, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt32)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt32, err))
			i2, err := v2.Int32()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Int32, i2)
		})

		t.Run(fmt.Sprintf("%s(int64): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int64()
			testutils.Equal(t, test.Int64, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt64, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt64)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrInt64, err))
			i2, err := v2.Int64()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Int64, i2)
		})
	}
}

func TestUintValue(t *testing.T) {
	for _, test := range testutils.GetUintTests() {
		t.Run(fmt.Sprintf("%s(uint): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint()
			testutils.Equal(t, test.Uint, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint, err))
			i2, err := v2.Uint()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Uint, i2)
		})

		t.Run(fmt.Sprintf("%s(uint8): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint8()
			testutils.Equal(t, test.Uint8, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint8, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint8)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint8, err))
			i2, err := v2.Uint8()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Uint8, i2)
		})

		t.Run(fmt.Sprintf("%s(uint16): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint16()
			testutils.Equal(t, test.Uint16, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint16, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint16)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint16, err))
			i2, err := v2.Uint16()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Uint16, i2)
		})

		t.Run(fmt.Sprintf("%s(uint32): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint32()
			testutils.Equal(t, test.Uint32, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint32, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint32)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint32, err))
			i2, err := v2.Uint32()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Uint32, i2)
		})

		t.Run(fmt.Sprintf("%s(uint64): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint64()
			testutils.Equal(t, test.Uint64, i1)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint64, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint64)
			testutils.NoError(t, testutils.CheckIntErrors(test.Val, test.Errs, testutils.ErrUint64, err))
			i2, err := v2.Uint64()
			testutils.NoError(t, err)
			testutils.Equal(t, test.Uint64, i2)
		})
	}
}

func TestUintptrValue(t *testing.T) {
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
		v1, err := vars.NewValue(test.val)
		testutils.NoError(t, err)
		testutils.Equal(t, test.val, v1.String())
		i1, err := v1.Uintptr()
		testutils.Equal(t, test.want, i1)
		testutils.NoError(t, testutils.CheckIntErrors(test.val, test.errs, testutils.ErrUint64, err))

		v2, err := vars.ParseValueAs(test.val, vars.KindUintptr)
		testutils.NoError(t, testutils.CheckIntErrors(test.val, test.errs, testutils.ErrUintptr, err))
		i2, err := v2.Uintptr()
		testutils.NoError(t, err)
		testutils.Equal(t, test.want, i2)
	}
}
