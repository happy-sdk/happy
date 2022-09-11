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
	"strings"
	"testing"
)

func TestNewVariableValueKind(t *testing.T) {
	for _, test := range testutils.GetKindTests() {
		t.Run("bool: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Bool, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindBool, v.Kind(), test.Key)
		})

		t.Run("float32: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Float32, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindFloat32, v.Kind(), test.Key)
		})

		t.Run("float64: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Float64, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindFloat64, v.Kind(), test.Key)
		})

		t.Run("complex64: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Complex64, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindComplex64, v.Kind(), test.Key)
		})

		t.Run("complex128: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Complex128, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindComplex128, v.Kind(), test.Key)
		})

		t.Run("int: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt, v.Kind(), test.Key)
		})

		t.Run("int8: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int8, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt8, v.Kind(), test.Key)
		})

		t.Run("int16: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int16, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt16, v.Kind(), test.Key)
		})

		t.Run("int32: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int32, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt32, v.Kind(), test.Key)
		})

		t.Run("int64: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Int64, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindInt64, v.Kind(), test.Key)
		})

		t.Run("uint: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint, v.Kind(), test.Key)
		})

		t.Run("uint8: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint8, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint8, v.Kind(), test.Key)
		})

		t.Run("uint16: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint16, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint16, v.Kind(), test.Key)
		})

		t.Run("uint32: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint32, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint32, v.Kind(), test.Key)
		})

		t.Run("uint64: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uint64, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUint64, v.Kind(), test.Key)
		})

		t.Run("uintptr: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.Uintptr, false)
			testutils.NoError(t, err)
			testutils.Equal(t, vars.KindUintptr, v.Kind(), test.Key)
		})

		t.Run("string: "+test.Key, func(t *testing.T) {
			v, err := vars.NewVariable(test.Key, test.String, false)
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
			testutils.ErrorIs(t, err, vars.ErrValue)
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

func TestValueKindOf(t *testing.T) {
	testutils.Equal(t, vars.KindInvalid, vars.ValueKindOf(nil))
	var str string

	testutils.Equal(t, vars.KindPointer.String(), vars.ValueKindOf(&str).String())
	var vstruct vars.VariableIface[vars.Value]
	testutils.Equal(t, vars.KindStruct.String(), vars.ValueKindOf(vstruct).String())
	var viface fmt.Stringer
	testutils.Equal(t, vars.KindInvalid.String(), vars.ValueKindOf(viface).String())
}

func TestLen(t *testing.T) {
	for _, test := range testutils.GetKindTests() {
		v, err := vars.NewVariable(test.Key, test.In, false)
		testutils.NoError(t, err)

		testutils.Equal(t, len(v.String()), len(test.In), test.Key)
		testutils.Equal(t, v.Len(), len(test.In), test.Key)
	}
}

func TestValueFields(t *testing.T) {
	v, err := vars.NewVariable("fields", "word1 word2 word3", false)
	testutils.NoError(t, err)
	if len(v.Fields()) != 3 {
		t.Error("len of fields should be 3")
	}
}

func TestErrors(t *testing.T) {
	_, err := vars.ParseVariableAs("", "", false, vars.KindString)
	testutils.ErrorIs(t, err, vars.ErrKey)
}

func TestVariableIface(t *testing.T) {
	v := vars.As[vars.Value](vars.EmptyVariable)
	testutils.Equal(t, "", v.Key())
	testutils.Equal(t, false, v.ReadOnly())
	testutils.Equal(t, "", v.String())
	testutils.Equal(t, "", v.Value().String())

	vv := vars.ValueOf(v)
	testutils.Equal(t, "", vv.String())
}

func TestNewVariable(t *testing.T) {
	for _, test := range testutils.GetNewTests() {
		v, err := vars.NewVariable(test.Key, test.Val, false)
		if test.Val != nil {
			testutils.NoError(t, err)
		}
		want := fmt.Sprintf("%v", test.Val)
		testutils.Equal(t, want, v.String())
	}
}

func TestParseVariableFromString(t *testing.T) {
	tests := testutils.GetKeyValueParseTests()
	for _, test := range tests {
		kv := fmt.Sprintf("%s=%s", test.Key, test.Val)
		t.Run(kv, func(t *testing.T) {
			v, err := vars.ParseVariableFromString(kv)

			testutils.ErrorIs(t, err, test.Err, kv)
			if err == nil {
				testutils.Equal(t, vars.KindString, v.Kind())
				testutils.EqualAnyf(t, test.WantVal, v.Underlying(), "val1.Underlying = in(%s)", test.Val)
			} else {
				testutils.Equal(t, vars.KindInvalid, v.Kind())
				testutils.EqualAnyf(t, nil, v.Underlying(), "val1.Underlying = in(%s)", test.Val)
			}
			testutils.Equalf(t, test.WantKey, v.Key(), "key1 = in(%s)", test.Key)
			testutils.Equalf(t, test.WantVal, v.String(), "val1.String = in(%s)", test.Val)

			if strings.Contains(test.Key, "=") {
				return
			}
			kvq := fmt.Sprintf("%q=%q", test.Key, test.Val)
			vq, err := vars.ParseVariableFromString(kvq)
			testutils.ErrorIs(t, err, test.Err, kv)
			if err == nil {
				testutils.Equal(t, vars.KindString, vq.Kind())
				testutils.EqualAnyf(t, test.WantValq, vq.Underlying(), "val2.Underlying = in(%q)", test.Val)
			} else {
				testutils.Equal(t, vars.KindInvalid, vq.Kind())
				testutils.EqualAnyf(t, nil, vq.Underlying(), "val2.Underlying = in(%q)", test.Val)
			}
			testutils.Equalf(t, test.WantKey, vq.Key(), "key2  in(%q)", test.Key)
			testutils.Equalf(t, test.WantValq, vq.String(), "val2.String = in(%q)", test.Val)
		})

	}
	v, err := vars.ParseVariableFromString("X=1")
	testutils.Equal(t, "X", v.Key())
	testutils.False(t, v.Empty())
	testutils.Equal(t, 1, v.Int())
	testutils.EqualAny(t, err, nil)
}

func TestParseVariableFromStringEmpty(t *testing.T) {
	v, err := vars.ParseVariableFromString("")
	testutils.True(t, v.Empty())
	testutils.Error(t, err)
	testutils.ErrorIs(t, err, vars.ErrKey)
}

func TestParseVariableFromStringEmptyKey(t *testing.T) {
	_, err := vars.ParseVariableFromString("=val")
	testutils.Error(t, err)
	testutils.ErrorIs(t, err, vars.ErrKey)
}

func TestNewValueString(t *testing.T) {
	for _, test := range testutils.GetStringTests() {
		v1, err := vars.NewValue(test.Val)
		testutils.NoError(t, err)
		testutils.Equal(t, test.Val, v1.String())

		v2, err := vars.ParseValueAs(test.Val, vars.KindString)
		testutils.NoError(t, err)
		testutils.Equal(t, test.Val, v2.String())

		v3, err := vars.NewValue(vars.EmptyValue)
		testutils.ErrorIs(t, err, vars.ErrValue)
		testutils.Equal(t, v3.Kind(), vars.KindInvalid)
	}
}
