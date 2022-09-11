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

// import (
// 	"fmt"
// 	"github.com/mkungla/happy/x/pkg/vars"
// 	"github.com/mkungla/happy/x/pkg/vars/testdata"
// 	"github.com/stretchr/testify/assert"
// 	"math"
// 	"strings"
// 	"testing"
// )

// func FuzzNewValueInt(f *testing.F) {
// 	for _, arg := range testdata.GetIntTests() {
// 		f.Add(arg.Int)
// 	}
// 	f.Fuzz(func(t *testing.T, arg int) {
// 		v, err := vars.NewVariable("int", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindInt, v.Kind())
// 		assert.Equal(t, arg, v.Int())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindInt, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueInt8(f *testing.F) {
// 	for _, arg := range testdata.GetIntTests() {
// 		f.Add(arg.Int8)
// 	}
// 	f.Fuzz(func(t *testing.T, arg int8) {
// 		v, err := vars.NewVariable("int8", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindInt8, v.Kind())
// 		assert.Equal(t, arg, v.Int8())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindInt8, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueInt16(f *testing.F) {
// 	for _, arg := range testdata.GetIntTests() {
// 		f.Add(arg.Int16)
// 	}
// 	f.Fuzz(func(t *testing.T, arg int16) {
// 		v, err := vars.NewVariable("int16", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindInt16, v.Kind())
// 		assert.Equal(t, arg, v.Int16())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindInt16, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueInt32(f *testing.F) {
// 	for _, arg := range testdata.GetIntTests() {
// 		f.Add(arg.Int32)
// 	}
// 	f.Fuzz(func(t *testing.T, arg int32) {
// 		v, err := vars.NewVariable("int32", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindInt32, v.Kind())
// 		assert.Equal(t, arg, v.Int32())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindInt32, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueInt64(f *testing.F) {
// 	for _, arg := range testdata.GetIntTests() {
// 		f.Add(arg.Int64)
// 	}
// 	f.Fuzz(func(t *testing.T, arg int64) {
// 		v, err := vars.NewVariable("int64", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindInt64, v.Kind())
// 		assert.Equal(t, arg, v.Int64())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindInt64, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueUint(f *testing.F) {
// 	for _, arg := range testdata.GetUintTests() {
// 		f.Add(arg.Uint)
// 	}
// 	f.Fuzz(func(t *testing.T, arg uint) {
// 		v, err := vars.NewVariable("uint", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindUint, v.Kind())
// 		assert.Equal(t, arg, v.Uint())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindUint, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueUint8(f *testing.F) {
// 	for _, arg := range testdata.GetUintTests() {
// 		f.Add(arg.Uint8)
// 	}
// 	f.Fuzz(func(t *testing.T, arg uint8) {
// 		v, err := vars.NewVariable("uint8", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindUint8, v.Kind())
// 		assert.Equal(t, arg, v.Uint8())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindUint8, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueUint16(f *testing.F) {
// 	for _, arg := range testdata.GetUintTests() {
// 		f.Add(arg.Uint16)
// 	}
// 	f.Fuzz(func(t *testing.T, arg uint16) {
// 		v, err := vars.NewVariable("uint16", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindUint16, v.Kind())
// 		assert.Equal(t, arg, v.Uint16())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindUint16, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueUint32(f *testing.F) {
// 	for _, arg := range testdata.GetUintTests() {
// 		f.Add(arg.Uint32)
// 	}
// 	f.Fuzz(func(t *testing.T, arg uint32) {
// 		v, err := vars.NewVariable("uint32", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindUint32, v.Kind())
// 		assert.Equal(t, arg, v.Uint32())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindUint32, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueUint64(f *testing.F) {
// 	for _, arg := range testdata.GetUintTests() {
// 		f.Add(arg.Uint64)
// 	}
// 	f.Fuzz(func(t *testing.T, arg uint64) {
// 		v, err := vars.NewVariable("uint64", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindUint64, v.Kind())
// 		assert.Equal(t, arg, v.Uint64())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindUint64, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueFloat32(f *testing.F) {
// 	for _, arg := range testdata.GetFloat32Tests() {
// 		f.Add(arg.WantFloat32)
// 	}
// 	f.Fuzz(func(t *testing.T, arg float32) {
// 		v, err := vars.NewVariable("float32", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindFloat32, v.Kind())
// 		assert.Equal(t, arg, v.Float32())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindFloat32, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueFloat64(f *testing.F) {
// 	for _, arg := range testdata.GetFloat64Tests() {
// 		f.Add(arg.WantFloat64)
// 	}
// 	f.Fuzz(func(t *testing.T, arg float64) {
// 		v, err := vars.NewVariable("float64", arg, true)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindFloat64, v.Kind())
// 		if math.IsNaN(arg) {
// 			return
// 		}
// 		assert.Equal(t, arg, v.Float64())
// 		assert.Equal(t, arg, v.Underlying())
// 		assert.Equal(t, fmt.Sprint(arg), v.String())
// 		assert.Equal(t, vars.KindFloat64, vars.ValueKindOf(arg))
// 	})
// }

// func FuzzNewValueString(f *testing.F) {
// 	testargs := []string{
// 		"",
// 		"<nil>",
// 		"1",
// 		"0",
// 		"-0",
// 		"-1",
// 		"abc",
// 	}
// 	for _, arg := range testargs {
// 		f.Add(arg)
// 	}
// 	f.Fuzz(func(t *testing.T, arg string) {
// 		v, err := vars.NewValue(arg)
// 		assert.NoError(t, err)
// 		assert.Equal(t, vars.KindString, v.Kind())
// 		assert.Equal(t, arg, v.String())
// 		assert.Equal(t, arg, v.Underlying())
// 	})
// }

// func FuzzParseKeyValue(f *testing.F) {
// 	tests := testdata.GetKeyValueParseTests()
// 	for _, test := range tests {
// 		if test.Fuzz {
// 			f.Add(test.Key, test.Val)
// 		}
// 	}
// 	f.Fuzz(func(t *testing.T, key, val string) {
// 		if strings.Contains(key, "=") || val == "=" {
// 			return
// 		}

// 		kv := fmt.Sprintf("%s=%s", key, val)
// 		v, err := vars.ParseVariableFromString(kv)

// 		expkey, _ := vars.ParseKey(key)
// 		expval := testdata.NormalizeExpValue(val)
// 		if err == nil {
// 			assert.Equal(t, vars.KindString, v.Kind())
// 			assert.Equalf(t, expval, v.Underlying(), "val1.Underlying -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, expval, v.String(), "val1.String -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, expkey, v.Key(), "key1 -> key(%s) val(%s)", key, val)
// 		} else {
// 			assert.Equal(t, vars.KindInvalid, v.Kind())
// 			assert.Equalf(t, nil, v.Underlying(), "val1.Underlying -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, "", v.String(), "val1.String -> key(%s) val(%s)", key, val)
// 		}

// 		// exceptions and special cases we can not test there
// 		// and should be in TestParseKeyValue
// 		if strings.ContainsRune(val, '"') {
// 			return
// 		}
// 		keyq := fmt.Sprintf("%q", key)
// 		valq := fmt.Sprintf("%q", val)
// 		if strings.Contains(keyq, "\\") || strings.Contains(valq, "\\") {
// 			return
// 		}
// 		kvq := fmt.Sprintf("%s=%s", keyq, valq)
// 		vq, err2 := vars.ParseVariableFromString(kvq)
// 		if err2 == nil {
// 			assert.Equal(t, vars.KindString, vq.Kind())
// 			assert.Equalf(t, val, vq.Underlying(), "val2.Underlying -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, val, vq.String(), "val2.String -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, expkey, vq.Key(), "key2 -> key(%s) val(%s)", key, val)
// 		} else {
// 			assert.Equal(t, vars.KindInvalid, vq.Kind())
// 			assert.Equalf(t, nil, vq.Underlying(), "val2.Underlying -> key(%s) val(%s)", key, val)
// 			assert.Equalf(t, "", vq.String(), "val2.String -> key(%s) val(%s)", key, val)
// 		}

// 	})
// }
