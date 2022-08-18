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
	"github.com/mkungla/happy/x/pkg/vars/testdata"
	"github.com/stretchr/testify/assert"

	"math"
	"strings"
	"testing"
	"time"
)

func TestParseKeyValue(t *testing.T) {
	tests := testdata.GetKeyValueParseTests()
	for _, test := range tests {
		kv := fmt.Sprintf("%s=%s", test.Key, test.Val)
		t.Run(kv, func(t *testing.T) {
			v, err := vars.ParseKeyValue(kv)

			assert.ErrorIs(t, err, test.Err, kv)
			if err == nil {
				assert.Equal(t, vars.TypeString, v.Type())
				assert.Equalf(t, test.WantVal, v.Underlying(), "val1.Underlying = in(%s)", test.Val)
			} else {
				assert.Equal(t, vars.TypeInvalid, v.Type())
				assert.Equalf(t, nil, v.Underlying(), "val1.Underlying = in(%s)", test.Val)
			}
			assert.Equalf(t, test.WantKey, v.Key(), "key1 = in(%s)", test.Key)
			assert.Equalf(t, test.WantVal, v.String(), "val1.String = in(%s)", test.Val)

			if strings.Contains(test.Key, "=") {
				return
			}
			kvq := fmt.Sprintf("%q=%q", test.Key, test.Val)
			vq, err := vars.ParseKeyValue(kvq)
			assert.ErrorIs(t, err, test.Err, kv)
			if err == nil {
				assert.Equal(t, vars.TypeString, vq.Type())
				assert.Equalf(t, test.WantValq, vq.Underlying(), "val2.Underlying = in(%q)", test.Val)
			} else {
				assert.Equal(t, vars.TypeInvalid, vq.Type())
				assert.Equalf(t, nil, vq.Underlying(), "val2.Underlying = in(%q)", test.Val)
			}
			assert.Equalf(t, test.WantKey, vq.Key(), "key2  in(%q)", test.Key)
			assert.Equalf(t, test.WantValq, vq.String(), "val2.String = in(%q)", test.Val)
		})

	}
	v, err := vars.ParseKeyValue("X=1")
	assert.Equal(t, "X", v.Key())
	assert.False(t, v.Empty())
	assert.Equal(t, 1, v.Int())
	assert.Equal(t, err, nil)
}

func TestParseKeyValueEmpty(t *testing.T) {
	v, err := vars.ParseKeyValue("")
	assert.True(t, v.Empty())
	assert.Error(t, err)
	assert.ErrorIs(t, err, vars.ErrKeyEmpty)
}

func TestParseKeyValueEmptyKey(t *testing.T) {
	_, err := vars.ParseKeyValue("=val")
	assert.Error(t, err)
	assert.ErrorIs(t, err, vars.ErrKeyEmpty)
}

func TestBoolValue(t *testing.T) {
	for _, test := range testdata.GetBoolTests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			assert.Equal(t, test.In, v1.String())
			assert.NoError(t, err)

			b1, err := v1.Bool()
			assert.Equal(t, test.Want, b1)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, false, b1)
				assert.Equal(t, vars.TypeString, v1.Type())
			} else {
				assert.Equal(t, test.Want, b1)
				assert.Equal(t, vars.TypeString, v1.Type())
			}

			v2, err := vars.NewValue(test.Want)
			assert.NoError(t, err)
			b2, err := v2.Bool()
			assert.NoError(t, err)
			assert.Equal(t, test.Want, b2)

			v3, err := vars.ParseTypedValue(test.In, vars.TypeBool)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, vars.TypeInvalid, v3.Type())
			} else {
				b3, err := v3.Bool()
				assert.Equal(t, test.Want, b3)
				assert.ErrorIs(t, err, test.Err)
				assert.Equal(t, vars.TypeBool, v3.Type())
			}
			v4, err := vars.ParseTypedVariable("var", test.In, false, vars.TypeBool)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, vars.TypeInvalid, v4.Type())
			} else {
				assert.Equal(t, vars.TypeBool, v4.Type())
				assert.Equal(t, test.Want, v4.Bool())
			}
			v5, err := vars.NewVariable("var", v4, false)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValue)
				assert.Equal(t, vars.TypeInvalid, v5.Type())
			} else {
				assert.Equal(t, vars.TypeBool, v5.Type())
				assert.Equal(t, test.Want, v5.Bool())
			}
			v6, err := vars.NewValue(v4)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValue)
				assert.Equal(t, vars.TypeInvalid, v6.Type())
			} else {
				assert.Equal(t, vars.TypeBool, v6.Type())
				b6, _ := v6.Bool()
				assert.Equal(t, test.Want, b6)
			}
		})
	}
}

func TestFloat32Value(t *testing.T) {
	for _, test := range testdata.GetFloat32Tests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			assert.Equal(t, test.In, v1.String())
			assert.NoError(t, err)

			b1, err := v1.Float32()
			assert.Equal(t, test.WantFloat32, b1)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, test.WantFloat32, b1)
				assert.Equal(t, vars.TypeString, v1.Type())
			} else {
				assert.Equal(t, test.WantFloat32, b1)
				assert.Equal(t, vars.TypeString, v1.Type())
			}

			v2, err := vars.NewValue(test.WantFloat32)
			assert.NoError(t, err)
			b2, err := v2.Float32()
			assert.NoError(t, err)
			assert.Equal(t, test.WantFloat32, b2)

			v3, err := vars.ParseTypedValue(test.In, vars.TypeFloat32)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, vars.TypeInvalid, v3.Type())
			} else {
				b3, err := v3.Float32()
				assert.Equal(t, test.WantFloat32, b3)
				assert.ErrorIs(t, err, test.Err)
				assert.Equal(t, vars.TypeFloat32, v3.Type())
			}

			v4, err := vars.ParseTypedVariable("var", test.In, false, vars.TypeFloat32)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, vars.TypeInvalid, v4.Type())
			} else {
				assert.Equal(t, vars.TypeFloat32, v4.Type())
				assert.Equal(t, test.WantFloat32, v4.Float32())
				assert.ErrorIs(t, err, test.Err)
			}
		})
	}
}

func TestFloat64Value(t *testing.T) {
	for _, test := range testdata.GetFloat64Tests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			assert.Equal(t, test.In, v1.String())
			assert.NoError(t, err)

			b1, err := v1.Float64()
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, test.WantFloat64, b1)
				assert.Equal(t, vars.TypeString, v1.Type())
			} else {
				if !math.IsNaN(test.WantFloat64) {
					assert.Equal(t, test.WantFloat64, b1)
				} else {
					assert.Equal(t, math.IsNaN(test.WantFloat64), math.IsNaN(b1))
				}
				assert.Equal(t, vars.TypeString, v1.Type())
			}

			v2, err := vars.NewValue(test.WantFloat64)
			assert.NoError(t, err)
			b2, err := v2.Float64()
			assert.NoError(t, err)

			if !math.IsNaN(test.WantFloat64) {
				assert.Equal(t, test.WantFloat64, b2)
			} else {
				assert.Equal(t, math.IsNaN(test.WantFloat64), math.IsNaN(b2))
			}

			v3, err := vars.ParseTypedValue(test.In, vars.TypeFloat64)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, vars.TypeInvalid, v3.Type())
			} else {
				b3, err := v3.Float64()
				assert.ErrorIs(t, err, test.Err)
				if err != nil {
					assert.Equal(t, float64(0), b3)
					assert.Equal(t, vars.TypeInvalid, v3.Type())
				} else {
					if !math.IsNaN(test.WantFloat64) {
						assert.Equal(t, test.WantFloat64, b3)
					} else {
						assert.Equal(t, math.IsNaN(test.WantFloat64), math.IsNaN(b3))
					}

					assert.Equal(t, vars.TypeFloat64, v3.Type())
				}
			}

			v4, err := vars.ParseTypedVariable("var", test.In, false, vars.TypeFloat64)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, vars.TypeInvalid, v4.Type())
			} else {
				assert.Equal(t, vars.TypeFloat64, v4.Type())
				if !math.IsNaN(test.WantFloat64) {
					assert.Equal(t, test.WantFloat64, v4.Float64())
				} else {
					assert.Equal(t, math.IsNaN(test.WantFloat64), math.IsNaN(v4.Float64()))
				}
			}
		})
	}
}

func TestComplex64Value(t *testing.T) {
	for _, test := range testdata.GetComplex64Tests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			assert.Equal(t, test.In, v1.String())
			assert.NoError(t, err)

			b1, err := v1.Complex64()
			assert.Equal(t, test.WantComplex64, b1)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, test.WantComplex64, b1)
				assert.Equal(t, vars.TypeString, v1.Type())
			} else {
				assert.Equal(t, test.WantComplex64, b1)
				assert.Equal(t, vars.TypeString, v1.Type())
			}

			v2, err := vars.NewValue(test.WantComplex64)
			assert.NoError(t, err)
			b2, err := v2.Complex64()
			assert.NoError(t, err)
			assert.Equal(t, test.WantComplex64, b2)

			v3, err := vars.ParseTypedValue(test.In, vars.TypeComplex64)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, vars.TypeInvalid, v3.Type())
			} else {
				b3, err := v3.Complex64()
				assert.Equal(t, test.WantComplex64, b3)
				assert.ErrorIs(t, err, test.Err)
				if err != nil {
					assert.Equal(t, 0, b3)
					assert.Equal(t, vars.TypeInvalid, v3.Type())
				} else {
					assert.Equal(t, test.WantComplex64, b3)
					assert.Equal(t, vars.TypeComplex64, v3.Type())
				}
			}
		})
	}
}

func TestComplex128Value(t *testing.T) {
	for _, test := range testdata.GetComplex128Tests() {
		t.Run(test.Key, func(t *testing.T) {
			v1, err := vars.NewValue(test.In)
			assert.Equal(t, test.In, v1.String())
			assert.NoError(t, err)

			b1, err := v1.Complex128()
			assert.Equal(t, test.WantComplex128, b1)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, test.WantComplex128, b1)
				assert.Equal(t, vars.TypeString, v1.Type())
			} else {
				assert.Equal(t, test.WantComplex128, b1)
				assert.Equal(t, vars.TypeString, v1.Type())
			}

			v2, err := vars.NewValue(test.WantComplex128)
			assert.NoError(t, err)
			b2, err := v2.Complex128()
			assert.NoError(t, err)
			assert.Equal(t, test.WantComplex128, b2)

			v3, err := vars.ParseTypedValue(test.In, vars.TypeComplex128)
			assert.ErrorIs(t, err, test.Err)
			if err != nil {
				assert.Equal(t, vars.TypeInvalid, v3.Type())
			} else {
				b3, err := v3.Complex128()
				assert.Equal(t, test.WantComplex128, b3)
				assert.ErrorIs(t, err, test.Err)
				if err != nil {
					assert.Equal(t, 0, b3)
					assert.Equal(t, vars.TypeInvalid, v3.Type())
				} else {
					assert.Equal(t, test.WantComplex128, b3)
					assert.Equal(t, vars.TypeComplex128, v3.Type())
				}
			}
		})
	}
}

func TestIntValue(t *testing.T) {
	for _, test := range testdata.GetIntTests() {
		t.Run(fmt.Sprintf("%s(int): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Int()
			assert.Equal(t, test.Int, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeInt)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt, err))
			i2, err := v2.Int()
			assert.NoError(t, err)
			assert.Equal(t, test.Int, i2)
		})

		t.Run(fmt.Sprintf("%s(int8): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Int8()
			assert.Equal(t, test.Int8, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt8, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeInt8)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt8, err))
			i2, err := v2.Int8()
			assert.NoError(t, err)
			assert.Equal(t, test.Int8, i2)
		})

		t.Run(fmt.Sprintf("%s(int16): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Int16()
			assert.Equal(t, test.Int16, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt16, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeInt16)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt16, err))
			i2, err := v2.Int16()
			assert.NoError(t, err)
			assert.Equal(t, test.Int16, i2)
		})

		t.Run(fmt.Sprintf("%s(int32): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Int32()
			assert.Equal(t, test.Int32, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt32, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeInt32)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt32, err))
			i2, err := v2.Int32()
			assert.NoError(t, err)
			assert.Equal(t, test.Int32, i2)
		})

		t.Run(fmt.Sprintf("%s(int64): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Int64()
			assert.Equal(t, test.Int64, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt64, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeInt64)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrInt64, err))
			i2, err := v2.Int64()
			assert.NoError(t, err)
			assert.Equal(t, test.Int64, i2)
		})
	}
}

func TestUintValue(t *testing.T) {
	for _, test := range testdata.GetUintTests() {
		t.Run(fmt.Sprintf("%s(uint): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint()
			assert.Equal(t, test.Uint, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeUint)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint, err))
			i2, err := v2.Uint()
			assert.NoError(t, err)
			assert.Equal(t, test.Uint, i2)
		})

		t.Run(fmt.Sprintf("%s(uint8): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint8()
			assert.Equal(t, test.Uint8, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint8, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeUint8)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint8, err))
			i2, err := v2.Uint8()
			assert.NoError(t, err)
			assert.Equal(t, test.Uint8, i2)
		})

		t.Run(fmt.Sprintf("%s(uint16): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint16()
			assert.Equal(t, test.Uint16, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint16, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeUint16)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint16, err))
			i2, err := v2.Uint16()
			assert.NoError(t, err)
			assert.Equal(t, test.Uint16, i2)
		})

		t.Run(fmt.Sprintf("%s(uint32): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint32()
			assert.Equal(t, test.Uint32, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint32, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeUint32)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint32, err))
			i2, err := v2.Uint32()
			assert.NoError(t, err)
			assert.Equal(t, test.Uint32, i2)
		})

		t.Run(fmt.Sprintf("%s(uint64): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			assert.NoError(t, err)
			assert.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint64()
			assert.Equal(t, test.Uint64, i1)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint64, err))

			v2, err := vars.ParseTypedValue(test.Val, vars.TypeUint64)
			assert.NoError(t, testdata.CheckIntErrors(test.Val, test.Errs, testdata.ErrUint64, err))
			i2, err := v2.Uint64()
			assert.NoError(t, err)
			assert.Equal(t, test.Uint64, i2)
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
		assert.NoError(t, err)
		assert.Equal(t, test.val, v1.String())
		i1, err := v1.Uintptr()
		assert.Equal(t, test.want, i1)
		assert.NoError(t, testdata.CheckIntErrors(test.val, test.errs, testdata.ErrUint64, err))

		v2, err := vars.ParseTypedValue(test.val, vars.TypeUintptr)
		assert.NoError(t, testdata.CheckIntErrors(test.val, test.errs, testdata.ErrUintptr, err))
		i2, err := v2.Uintptr()
		assert.NoError(t, err)
		assert.Equal(t, test.want, i2)
	}
}

func TestNewValueString(t *testing.T) {
	for _, test := range testdata.GetStringTests() {
		v1, err := vars.NewValue(test.Val)
		assert.NoError(t, err)
		assert.Equal(t, test.Val, v1.String())

		v2, err := vars.ParseTypedValue(test.Val, vars.TypeString)
		assert.NoError(t, err)
		assert.Equal(t, test.Val, v2.String())

		v3, err := vars.NewValue(vars.EmptyValue)
		assert.ErrorIs(t, err, vars.ErrValue)
		assert.Equal(t, v3.Type(), vars.TypeInvalid)
	}
}

func TestVariableTypes(t *testing.T) {
	for _, test := range testdata.GetTypeTests() {
		v, err := vars.NewVariable(test.Key, test.In, false)
		assert.NoError(t, err)
		assert.Equal(t, test.Bool, v.Bool(), test.Key)
		assert.Equal(t, test.Float32, v.Float32(), test.Key)
		assert.Equal(t, test.Float64, v.Float64(), test.Key)
		assert.Equal(t, test.Complex64, v.Complex64(), test.Key)
		assert.Equal(t, test.Complex128, v.Complex128(), test.Key)
		assert.Equal(t, test.Int, v.Int(), test.Key)
		assert.Equal(t, test.Int8, v.Int8(), test.Key)
		assert.Equal(t, test.Int16, v.Int16(), test.Key)
		assert.Equal(t, test.Int32, v.Int32(), test.Key)
		assert.Equal(t, test.Int64, v.Int64(), test.Key)
		assert.Equal(t, test.Uint, v.Uint(), test.Key)
		assert.Equal(t, test.Uint8, v.Uint8(), test.Key)
		assert.Equal(t, test.Uint16, v.Uint16(), test.Key)
		assert.Equal(t, test.Uint32, v.Uint32(), test.Key)
		assert.Equal(t, test.Uint64, v.Uint64(), test.Key)
		assert.Equal(t, test.Uintptr, v.Uintptr(), test.Key)
		assert.Equal(t, test.String, v.String(), test.Key)
		// assert.Equal(t, test.bytes, v.Bytes(), test.key)
		// assert.Equal(t, test.runes, v.Runes(), test.key)
	}
}

func TestValueTypes(t *testing.T) {
	for _, test := range testdata.GetTypeTests() {
		emsg := testdata.OnErrorMsg(test.Key, test.In)

		v, err := vars.NewVariable("value-types", test.In, false)
		assert.NoError(t, err, emsg)
		assert.False(t, v.ReadOnly())
		vBool, err := v.Value().Bool()
		assert.True(t, assert.Equal(t, test.Bool, vBool, emsg) || err != nil)
		assert.True(t, assert.Equal(t, test.Bool, vBool, emsg) || err != nil)

		vFloat32, err := v.Value().Float32()
		assert.True(t, assert.Equal(t, test.Float32, vFloat32, emsg) || err != nil)

		vfFloat64, err := v.Value().Float64()
		assert.True(t, assert.Equal(t, test.Float64, vfFloat64, emsg) || err != nil)

		vComplex64, err := v.Value().Complex64()
		assert.True(t, assert.Equal(t, test.Complex64, vComplex64, emsg) || err != nil)

		vComplex128, err := v.Value().Complex128()
		assert.True(t, assert.Equal(t, test.Complex128, vComplex128, emsg) || err != nil)

		vInt, err := v.Value().Int()
		assert.True(t, assert.Equal(t, test.Int, vInt, emsg) || err != nil)

		vInt8, err := v.Value().Int8()
		assert.True(t, assert.Equal(t, test.Int8, vInt8, emsg) || err != nil)

		vInt16, err := v.Value().Int16()
		assert.True(t, assert.Equal(t, test.Int16, vInt16, emsg) || err != nil)

		vInt32, err := v.Value().Int32()
		assert.True(t, assert.Equal(t, test.Int32, vInt32, emsg) || err != nil)

		vInt64, err := v.Value().Int64()
		assert.True(t, assert.Equal(t, test.Int64, vInt64, emsg) || err != nil)

		vUint, err := v.Value().Uint()
		assert.True(t, assert.Equal(t, test.Uint, vUint, emsg) || err != nil)

		vUint8, err := v.Value().Uint8()
		assert.True(t, assert.Equal(t, test.Uint8, vUint8, emsg) || err != nil)

		vUint16, err := v.Value().Uint16()
		assert.True(t, assert.Equal(t, test.Uint16, vUint16, emsg) || err != nil)

		vUint32, err := v.Value().Uint32()
		assert.True(t, assert.Equal(t, test.Uint32, vUint32, emsg) || err != nil)

		vUint64, err := v.Value().Uint64()
		assert.True(t, assert.Equal(t, test.Uint64, vUint64, emsg) || err != nil)

		vUintptr, err := v.Value().Uintptr()
		assert.True(t, assert.Equal(t, test.Uintptr, vUintptr, emsg) || err != nil)

		assert.Equal(t, test.String, v.String(), emsg)
		// assert.Equal(t, test.bytes, v.Bytes(), test.key)
		// assert.Equal(t, test.runes, v.Runes(), test.key)
	}
}

func TestNewVariable(t *testing.T) {
	for _, test := range testdata.GetNewTests() {
		v, err := vars.NewVariable(test.Key, test.Val, false)
		if test.Val != nil {
			assert.NoError(t, err)
		}
		want := fmt.Sprintf("%v", test.Val)
		assert.Equal(t, want, v.String())
	}
}

func TestNewVariableValueType(t *testing.T) {
	for _, test := range testdata.GetTypeTests() {
		t.Run("bool: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Bool, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeBool, v.Type(), test.Key)
		})

		t.Run("float32: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Float32, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeFloat32, v.Type(), test.Key)
		})

		t.Run("float64: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Float64, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeFloat64, v.Type(), test.Key)
		})

		t.Run("complex64: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Complex64, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeComplex64, v.Type(), test.Key)
		})

		t.Run("complex128: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Complex128, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeComplex128, v.Type(), test.Key)
		})

		t.Run("int: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeInt, v.Type(), test.Key)
		})

		t.Run("int8: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int8, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeInt8, v.Type(), test.Key)
		})

		t.Run("int16: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int16, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeInt16, v.Type(), test.Key)
		})

		t.Run("int32: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int32, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeInt32, v.Type(), test.Key)
		})

		t.Run("int64: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int64, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeInt64, v.Type(), test.Key)
		})

		t.Run("uint: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeUint, v.Type(), test.Key)
		})

		t.Run("uint8: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint8, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeUint8, v.Type(), test.Key)
		})

		t.Run("uint16: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint16, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeUint16, v.Type(), test.Key)
		})

		t.Run("uint32: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint32, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeUint32, v.Type(), test.Key)
		})

		t.Run("uint64: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint64, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeUint64, v.Type(), test.Key)
		})

		t.Run("uintptr: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uintptr, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeUintptr, v.Type(), test.Key)
		})

		t.Run("string: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.String, false)
			assert.NoError(t, err)
			assert.Equal(t, vars.TypeString, v.Type(), test.Key)
		})

		// t.Run("[]byte]: "+test.Key, func(t *testing.T) {
		// 	v, err := vars.NewVariable(test.Key, test.Bytes, false)
		// 	assert.NoError(t, err)
		// 	assert.Equal(t, vars.TypeBytes, v.Type(), test.Key)
		// })
		// t.Run("[]rune]: "+test.Key, func(t *testing.T) {
		// 	v, err := vars.NewVariable(test.Key, test.Runes, false)
		// 	assert.NoError(t, err)
		// 	assert.Equal(t, vars.TypeRunes, v.Type(), test.Key)
		// })
		// v2 := vars.New("runes", []rune{utf8.RuneSelf + 1})
		// assert.Equal(t, vars.TypeRunes, v2.Type())

		// v3 := vars.New("runes", []rune{utf8.UTFMax + 1})
		// assert.Equal(t, vars.TypeRunes, v3.Type())

		t.Run("TypeUnsafePointer: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseTypedVariable(test.Key, test.String, false, vars.TypeUnsafePointer)
			assert.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("TypeStruct: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseTypedVariable(test.Key, test.String, false, vars.TypeStruct)
			assert.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("TypeSlice: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseTypedVariable(test.Key, test.String, false, vars.TypeSlice)
			assert.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("TypeMap: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseTypedVariable(test.Key, test.String, false, vars.TypeMap)
			assert.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("TypeInterface: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseTypedVariable(test.Key, test.String, false, vars.TypeInterface)
			assert.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("TypeFunc: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseTypedVariable(test.Key, test.String, false, vars.TypeFunc)
			assert.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("TypeChan: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseTypedVariable(test.Key, test.String, false, vars.TypeChan)
			assert.ErrorIs(t, err, vars.ErrValue)
		})
		t.Run("TypeArray: "+test.Key, func(t *testing.T) {
			_, err := vars.ParseTypedVariable(test.Key, test.String, false, vars.TypeArray)
			assert.ErrorIs(t, err, vars.ErrValue)
		})
	}
}

func TestLen(t *testing.T) {
	for _, test := range testdata.GetTypeTests() {
		v, err := vars.NewVariable(test.Key, test.In, false)
		assert.NoError(t, err)

		assert.Equal(t, len(v.String()), len(test.In), test.Key)
		assert.Equal(t, v.Len(), len(test.In), test.Key)
	}
}

func TestValueFields(t *testing.T) {
	v, err := vars.NewVariable("fields", "word1 word2 word3", false)
	assert.NoError(t, err)
	if len(v.Fields()) != 3 {
		t.Error("len of fields should be 3")
	}
}

func TestErrors(t *testing.T) {
	_, err := vars.ParseTypedVariable("", "", false, vars.TypeString)
	assert.ErrorIs(t, err, vars.ErrKey)
}

func TestValueTypeFor(t *testing.T) {
	assert.Equal(t, vars.TypeInvalid, vars.ValueTypeFor(nil))
	var str string

	assert.Equal(t, vars.TypePointer.String(), vars.ValueTypeFor(&str).String())
	var vstruct vars.VariableIface[vars.Value]
	assert.Equal(t, vars.TypeStruct.String(), vars.ValueTypeFor(vstruct).String())
	var viface fmt.Stringer
	assert.Equal(t, vars.TypeInvalid.String(), vars.ValueTypeFor(viface).String())
}

func TestVariableIface(t *testing.T) {
	v := vars.As[vars.Value](vars.EmptyVariable)
	assert.Equal(t, "", v.Key())
	assert.Equal(t, false, v.ReadOnly())
	assert.Equal(t, "", v.String())
	assert.Equal(t, "", v.Value().String())

	vv := vars.ValueOf(v)
	assert.Equal(t, "", vv.String())
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

func TestCustomTypes(t *testing.T) {
	var ctyps = []struct {
		orig       any
		underlying any
		typ        vars.Type
		str        string
		len        int
		typtest    testdata.TypeTest
	}{
		{time.Duration(-123456), int64(-123456), vars.TypeInt64, "-123.456µs", 7, testdata.TypeTest{
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
		{time.Duration(123456), int64(123456), vars.TypeInt64, "123.456µs", 6, testdata.TypeTest{
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
		{time.Duration(123), int64(123), vars.TypeInt64, "123ns", 3, testdata.TypeTest{
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
		{time.Month(1), int(1), vars.TypeInt, "January", 1, testdata.TypeTest{
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
		{vars.Type(26), uint(26), vars.TypeUint, "unsafe.Pointer", 2, testdata.TypeTest{
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
			cbool(true), bool(true), vars.TypeBool, "true", 4, testdata.TypeTest{
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
			cbool(false), bool(false), vars.TypeBool, "false", 5, testdata.TypeTest{
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
			cint(1), int(1), vars.TypeInt, "1", 1, testdata.TypeTest{
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
			cint8(1), int8(1), vars.TypeInt8, "1", 1, testdata.TypeTest{
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
			cint16(1), int16(1), vars.TypeInt16, "1", 1, testdata.TypeTest{
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
			cint32(1), int32(1), vars.TypeInt32, "1", 1, testdata.TypeTest{
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
			cint64(1), int64(1), vars.TypeInt64, "1", 1, testdata.TypeTest{
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
			cuint(1), uint(1), vars.TypeUint, "1", 1, testdata.TypeTest{
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
			cuint8(1), uint8(1), vars.TypeUint8, "1", 1, testdata.TypeTest{
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
			cuint16(1), uint16(1), vars.TypeUint16, "1", 1, testdata.TypeTest{
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
			cuint32(1), uint32(1), vars.TypeUint32, "1", 1, testdata.TypeTest{
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
			cuint64(1), uint64(1), vars.TypeUint64, "1", 1, testdata.TypeTest{
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
			cuintptr(1), uintptr(1), vars.TypeUintptr, "1", 1, testdata.TypeTest{
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
			cfloat32(1.5), float32(1.5), vars.TypeFloat32, "1.5", 3, testdata.TypeTest{
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
			cfloat64(1.5), float64(1.5), vars.TypeFloat64, "1.5", 3, testdata.TypeTest{
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
			ccomplex64(complex(1.1, 2.5)), complex64(complex(1.1, 2.5)), vars.TypeComplex64, "(1.1+2.5i)", 10, testdata.TypeTest{
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
			ccomplex128(complex(1.1, 2.5)), complex128(complex(1.1, 2.5)), vars.TypeComplex128, "(1.1+2.5i)", 10, testdata.TypeTest{
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
			cstring("hello"), "hello", vars.TypeString, "hello", 5, testdata.TypeTest{
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
			cstring2("hello"), "hello", vars.TypeString, "hello", 5, testdata.TypeTest{
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
			cany(nil), nil, vars.TypeInvalid, "<nil>", 5, testdata.TypeTest{
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
				assert.NoError(t, err)
			}
			assert.Equal(t, test.typ.String(), v.Type().String())
			assert.Equal(t, test.underlying, v.Underlying())
			assert.Equal(t, test.str, v.String())

			t1, err := v.Bool()
			assert.Equal(t, test.typtest.Bool, t1)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t2, err := v.Float32()
			assert.Equal(t, test.typtest.Float32, t2)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t3, err := v.Float64()
			assert.Equal(t, test.typtest.Float64, t3)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t4, err := v.Complex64()
			assert.Equal(t, test.typtest.Complex64, t4)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t4b, err := v.Complex128()
			assert.Equal(t, test.typtest.Complex128, t4b)

			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t5, err := v.Int()
			assert.Equal(t, test.typtest.Int, t5)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t6, err := v.Int8()
			assert.Equal(t, test.typtest.Int8, t6)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t7, err := v.Int16()
			assert.Equal(t, test.typtest.Int16, t7)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t8, err := v.Int32()
			assert.Equal(t, test.typtest.Int32, t8)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t9, err := v.Int64()
			assert.Equal(t, test.typtest.Int64, t9)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t10, err := v.Uint()
			assert.Equal(t, test.typtest.Uint, t10)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t11, err := v.Uint8()
			assert.Equal(t, test.typtest.Uint8, t11)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t12, err := v.Uint16()
			assert.Equal(t, test.typtest.Uint16, t12)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t13, err := v.Uint32()
			assert.Equal(t, test.typtest.Uint32, t13)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t14, err := v.Uint64()
			assert.Equal(t, test.typtest.Uint64, t14)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t15, err := v.Uintptr()
			assert.Equal(t, test.typtest.Uintptr, t15)
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValueConv)
			}

			t16, err := v.AsType(vars.TypeString)
			assert.Equal(t, test.typtest.String, t16.String())
			if err != nil {
				assert.ErrorIs(t, err, vars.ErrValue)
			}

			vv, err := vars.NewTypedVariable("test", test.orig, false, test.typ)
			if err == nil {
				assert.NoError(t, err)
				assert.Equal(t, test.typ, vv.Type())
			}

			assert.Equal(t, test.len, v.Len())
		})
	}
}
