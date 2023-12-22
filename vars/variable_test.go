// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars_test

import (
	"fmt"
	"testing"

	"github.com/happy-sdk/happy-go/devel/testutils"
	"github.com/happy-sdk/happy-go/vars"
)

type newTest struct {
	Key string
	Val any
}

func getNewTests() []newTest {
	return []newTest{
		{"key", "<nil>"},
		{"key", nil},
		{"key", "val"},
		{"bool", true},
		{"float32", float32(32)},
		{"float64", float64(64)},
		{"complex64", complex64(complex(64, 64))},
		{"complex128", complex128(complex(128, 128))},
		{"int", int(1)},
		{"int8", int8(8)},
		{"int16", int16(16)},
		{"int32", int32(32)},
		{"int64", int64(64)},
		{"uint", uint(1)},
		{"uint8", uint8(8)},
		{"uint16", uint16(16)},
		{"uint32", uint32(32)},
		{"uint64", uint64(64)},
		{"uintptr", uintptr(10)},
		{"string", "string"},
		// {"byte_arr", []byte{1, 2, 3}},
	}
}

func TestNew(t *testing.T) {
	for _, test := range getNewTests() {
		v, err := vars.New(test.Key, test.Val, false)
		if test.Val != nil {
			testutils.NoError(t, err)
		}
		want := fmt.Sprintf("%v", test.Val)
		testutils.Equal(t, want, v.String())
	}

}

func TestEmptyVariable(t *testing.T) {
	v, err := vars.EmptyNamedVariable("vars")
	testutils.NoError(t, err)
	testutils.Equal(t, "vars", v.Name())

	v2, err := vars.EmptyNamedVariable("$vars")
	testutils.Error(t, err)
	testutils.Equal(t, "", v2.Name())
}

func TestLen(t *testing.T) {
	for _, test := range getKindTests() {
		v, err := vars.New(test.Key, test.In, false)
		testutils.NoError(t, err)

		testutils.Equal(t, len(v.String()), len(test.In), test.Key)
		testutils.Equal(t, v.Len(), len(test.In), test.Key)
	}
}

func TestVariableIFields(t *testing.T) {
	v, err := vars.New("fields", "word1 word2 word3", false)
	testutils.NoError(t, err)
	if len(v.Fields()) != 3 {
		t.Error("len of fields should be 3")
	}
}

func TestVariableIface(t *testing.T) {
	v := vars.AsVariable[vars.VariableIface[vars.Value], vars.Value](vars.EmptyVariable)
	testutils.Equal(t, "", v.Name())
	testutils.Equal(t, false, v.ReadOnly())
	testutils.Equal(t, "", v.String())
	testutils.Equal(t, "", v.Value().String())

	vv := vars.ValueOf(v)
	testutils.Equal(t, "", vv.String())

	variable2, _ := vars.New("key", "value", true)
	v2 := vars.AsVariable[vars.VariableIface[vars.Value], vars.Value](variable2)
	testutils.Equal(t, "key", v2.Name())
	testutils.Equal(t, true, v2.ReadOnly())
	testutils.Equal(t, "value", v2.String())
	testutils.Equal(t, "value", v2.Value().String())
	testutils.EqualAny(t, "value", v2.Any())
	testutils.Equal(t, 5, v2.Len())
	testutils.Equal(t, false, v2.Bool())

	variable3, _ := vars.New("key", "36", true)
	v3 := vars.AsVariable[vars.VariableIface[vars.Value], vars.Value](variable3)
	testutils.Equal(t, 36, v3.Int())
	testutils.Equal(t, 36, v3.Int8())
	testutils.Equal(t, 36, v3.Int16())
	testutils.Equal(t, 36, v3.Int32())
	testutils.Equal(t, 36, v3.Int64())
	testutils.Equal(t, 36, v3.Uint())
	testutils.Equal(t, 36, v3.Uint8())
	testutils.Equal(t, 36, v3.Uint16())
	testutils.Equal(t, 36, v3.Uint32())
	testutils.Equal(t, 36, v3.Uint64())
	testutils.Equal(t, 36, v3.Float32())
	testutils.Equal(t, 36, v3.Float64())
	testutils.Equal(t, complex64(36+0i), v3.Complex64())
	testutils.Equal(t, complex128(36+0i), v3.Complex128())
	testutils.Equal(t, 36, v3.Uintptr())
	testutils.EqualAny(t, []string{"36"}, v3.Fields())
}
