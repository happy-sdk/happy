// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars_test

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/mkungla/happy/sdk/testutils"
	"github.com/mkungla/happy/vars"
)

func FuzzNewValueInt(f *testing.F) {
	for _, arg := range getIntTests() {
		f.Add(arg.Int)
	}
	f.Fuzz(func(t *testing.T, arg int) {
		v, err := vars.New("int8", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindInt, v.Kind())
		testutils.Equal(t, arg, v.Int())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindInt, vars.KindOf(arg))

		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)
		if u, err := v2.Value().Int(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}
	})
}

func FuzzNewValueInt8(f *testing.F) {
	for _, arg := range getIntTests() {
		f.Add(arg.Int8)
	}
	f.Fuzz(func(t *testing.T, arg int8) {
		v, err := vars.New("int8", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindInt8, v.Kind())
		testutils.Equal(t, arg, v.Int8())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindInt8, vars.KindOf(arg))

		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)
		if u, err := v2.Value().Int8(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}
	})
}

func FuzzNewValueInt16(f *testing.F) {
	for _, arg := range getIntTests() {
		f.Add(arg.Int16)
	}
	f.Fuzz(func(t *testing.T, arg int16) {
		v, err := vars.New("int16", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindInt16, v.Kind())
		testutils.Equal(t, arg, v.Int16())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindInt16, vars.KindOf(arg))

		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)
		if u, err := v2.Value().Int16(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}
	})
}

func FuzzNewValueInt32(f *testing.F) {
	for _, arg := range getIntTests() {
		f.Add(arg.Int32)
	}
	f.Fuzz(func(t *testing.T, arg int32) {
		v, err := vars.New("int32", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindInt32, v.Kind())
		testutils.Equal(t, arg, v.Int32())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindInt32, vars.KindOf(arg))

		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)
		if u, err := v2.Value().Int32(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}
	})
}

func FuzzNewValueInt64(f *testing.F) {
	for _, arg := range getIntTests() {
		f.Add(arg.Int64)
	}
	f.Fuzz(func(t *testing.T, arg int64) {
		v, err := vars.New("int64", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindInt64, v.Kind())
		testutils.Equal(t, arg, v.Int64())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindInt64, vars.KindOf(arg))

		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)
		if u, err := v2.Value().Int64(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}

		for i := 0; i < 32; i++ {
			if i >= 2 {
				for j := 0; j < 64; j++ {
					str := v.Value().FormatInt(i)
					vars.ParseUint(str, i, j)
				}
				continue
			}
			for j := 0; j < 64; j++ {
				vars.ParseInt(v.String(), i, j)
				vars.ParseInt(string('b')+v.String(), i, j)
				vars.ParseInt(string('o')+v.String(), i, j)
				vars.ParseInt(string('x')+v.String(), i, j)
			}
		}
	})
}

func FuzzNewValueUint(f *testing.F) {
	for _, arg := range getUintTests() {
		f.Add(arg.Uint)
	}
	f.Fuzz(func(t *testing.T, arg uint) {
		v, err := vars.New("uint", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindUint, v.Kind())
		testutils.Equal(t, arg, v.Uint())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindUint, vars.KindOf(arg))
		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)
		if u, err := v2.Value().Uint(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}
	})
}

func FuzzNewValueUint8(f *testing.F) {
	for _, arg := range getUintTests() {
		f.Add(arg.Uint8)
	}
	f.Fuzz(func(t *testing.T, arg uint8) {
		v, err := vars.New("uint8", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindUint8, v.Kind())
		testutils.Equal(t, arg, v.Uint8())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindUint8, vars.KindOf(arg))

		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)
		if u, err := v2.Value().Uint8(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}
	})
}

func FuzzNewValueUint16(f *testing.F) {
	for _, arg := range getUintTests() {
		f.Add(arg.Uint16)
	}
	f.Fuzz(func(t *testing.T, arg uint16) {
		v, err := vars.New("uint16", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindUint16, v.Kind())
		testutils.Equal(t, arg, v.Uint16())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindUint16, vars.KindOf(arg))

		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)
		if u, err := v2.Value().Uint16(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}

	})
}

func FuzzNewValueUint32(f *testing.F) {
	for _, arg := range getUintTests() {
		f.Add(arg.Uint32)
	}
	f.Fuzz(func(t *testing.T, arg uint32) {
		v, err := vars.New("uint32", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindUint32, v.Kind())
		testutils.Equal(t, arg, v.Uint32())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindUint32, vars.KindOf(arg))

		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)

		if u, err := v2.Value().Uint32(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}
	})
}

func FuzzNewValueUint64(f *testing.F) {
	for _, arg := range getUintTests() {
		f.Add(arg.Uint64)
	}

	f.Add(uint64(1_00_00_0))
	f.Add(uint64(0_001_0.0_00_0))
	f.Fuzz(func(t *testing.T, arg uint64) {
		v, err := vars.New("uint64", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindUint64, v.Kind())
		testutils.Equal(t, arg, v.Uint64())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindUint64, vars.KindOf(arg))

		u, err := v.Value().Uint64()
		testutils.NoError(t, err)
		testutils.Equal(t, u, arg)

		v2, err := vars.New("key", v.String(), true)
		testutils.NoError(t, err)
		u2, err := v2.Value().Uint64()
		testutils.Equal(t, u2, arg)

		if u, err := v2.Value().Uint64(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}
		if u, err := v2.Value().Uintptr(); testutils.NoError(t, err) {
			vv, err := vars.NewValue(u)
			testutils.NoError(t, err)
			testutils.Equal(t, v.String(), vv.String())
		}

		for i := 0; i < 32; i++ {
			if i >= 2 {
				for j := 0; j < 64; j++ {
					str := v.Value().FormatUint(i)
					vars.ParseUint(str, i, j)
				}
				continue
			}
			for j := 0; j < 64; j++ {
				vars.StrconvParseUint(v.String(), i, j)
				vars.StrconvParseUint(string('b')+v.String(), i, j)
				vars.StrconvParseUint(string('o')+v.String(), i, j)
				vars.StrconvParseUint(string('x')+v.String(), i, j)
			}
		}
	})
}

func FuzzNewValueFloat32(f *testing.F) {
	for _, arg := range getFloat32Tests() {
		f.Add(arg.WantFloat32)
	}
	f.Add(float32(1_00_00_0.0_000_0100))
	f.Add(float32(0_001_00_00_0.0_000_0010))
	f.Fuzz(func(t *testing.T, arg float32) {
		v, err := vars.New("float32", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindFloat32, v.Kind())
		testutils.Equal(t, arg, v.Float32())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindFloat32, vars.KindOf(arg))

		f, err := v.Value().Float32()
		testutils.NoError(t, err)
		testutils.Equal(t, f, arg)

		v2, err := vars.New("float32", v.String(), true)
		testutils.NoError(t, err)
		f2, err := v2.Value().Float32()
		testutils.Equal(t, f2, arg)
		v2.Value().FormatFloat('e', -1, 32)
		v2.Value().FormatFloat('E', -1, 32)
		v2.Value().FormatFloat('f', -1, 32)
		v2.Value().FormatFloat('g', -1, 32)
		v2.Value().FormatFloat('G', -1, 32)
		v2.Value().FormatFloat('x', -1, 32)
		v2.Value().FormatFloat('X', -1, 32)

		f64, err := v.Value().Float64()
		testutils.NoError(t, err)

		v64, err := vars.NewValueAs(f64, vars.KindFloat32)
		testutils.NoError(t, err)
		f642, err := v64.Float32()
		testutils.Equal(t, f642, arg)

	})
}

func FuzzNewValueFloat64(f *testing.F) {
	for _, arg := range getFloat64Tests() {
		f.Add(arg.WantFloat64)
	}
	f.Add(float64(1_00_00_0.0_000_0010))
	f.Add(float64(0_001_00_00_0.0_000_0100))
	f.Fuzz(func(t *testing.T, arg float64) {
		v, err := vars.New("float64", arg, true)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindFloat64, v.Kind())
		if math.IsNaN(arg) {
			return
		}
		testutils.Equal(t, arg, v.Float64())
		testutils.EqualAny(t, arg, v.Any())
		testutils.Equal(t, fmt.Sprint(arg), v.String())
		testutils.Equal(t, vars.KindFloat64, vars.KindOf(arg))

		f, err := v.Value().Float64()
		testutils.NoError(t, err)
		testutils.Equal(t, f, arg)

		v2, err := vars.New("float64", v.String(), true)
		testutils.NoError(t, err)

		v2.Value().FormatFloat('e', -1, 64)
		v2.Value().FormatFloat('E', -1, 64)
		v2.Value().FormatFloat('f', -1, 64)
		v2.Value().FormatFloat('g', -1, 64)
		v2.Value().FormatFloat('G', -1, 64)
		v2.Value().FormatFloat('x', -1, 64)
		v2.Value().FormatFloat('X', -1, 64)

		f2, err := v2.Value().Float64()
		testutils.Equal(t, f2, arg)

	})
}

func FuzzNewValueString(f *testing.F) {
	testargs := []string{
		"",
		"<nil>",
		"1",
		"0",
		"01010101",
		"-0",
		"-1",
		"abc",
	}
	for _, arg := range testargs {
		f.Add(arg)
	}
	f.Fuzz(func(t *testing.T, arg string) {
		v, err := vars.NewValue(arg)
		testutils.NoError(t, err)
		testutils.Equal(t, vars.KindString, v.Kind())
		testutils.Equal(t, arg, v.String())
		testutils.EqualAny(t, arg, v.Any())
	})
}

func FuzzParseKeyValue(f *testing.F) {

	tests := getKeyValueParseTests()
	for _, test := range tests {
		if test.Fuzz {
			f.Add(test.Key, test.Val)
		}
		f.Add("key", "_1.0_1_0")
		f.Add("key", "_1.0_1_0.0")
		f.Add("key", "_10.0_1_0.01")
		f.Add("key", "_10.0_1_01")
		f.Add("key", "INF")
		f.Add("key", "INFI")
		f.Add("key", "INFI")
		f.Add("key", "_________________________________")
		f.Add("key", "10.0_1_01"+strings.Repeat("0", 500))
		f.Add("key", "A")
	}
	f.Fuzz(func(t *testing.T, key, val string) {
		clean := strings.TrimSpace(val)
		if strings.Contains(key, "=") || val == "=" || clean == "\"\"" {
			return
		}

		kv := fmt.Sprintf("%s=%s", key, val)
		v, err := vars.ParseVariableFromString(kv)
		if err != nil {
			testutils.Equal(t, vars.KindInvalid, v.Kind())
			return
		}

		testutils.Equal(t, vars.KindString, v.Kind())

		expkey, _ := vars.ParseKey(key)
		testutils.Equal(t, expkey, v.Name(), "key1 -> key(%s) val(%s)", key, val)

		expval := parseValueStd(val)

		if !testutils.Equal(t, expval, v.String(), "in = %q expval = %q v.String() = %q", val, expval, v.String()) {
			testutils.EqualAny(t, expval, v.Any(), ".String -> expval(%q) val(%q) %#v %#v", expval, v.String(), []byte(expval), []byte(v.String()))
		}

		if f64, err := strconv.ParseFloat(expval, 64); err == nil {
			vv, err := vars.NewValue(f64)
			testutils.NoError(t, err)

			str := strconv.FormatFloat(f64, 'g', -1, 64)
			testutils.Equal(t, str, vv.String())
			if f32, err := v.Value().Float32(); err == nil {
				vv, err := vars.NewValue(f32)
				testutils.NoError(t, err)

				str := strconv.FormatFloat(float64(f32), 'g', -1, 32)
				testutils.Equal(t, str, vv.String())
			}
		}

		u, err := strconv.ParseUint(expval, 10, 64)
		vv, err := vars.NewValue(u)
		testutils.NoError(t, err)

		str := strconv.FormatUint(uint64(u), 10)
		testutils.Equal(t, str, vv.String())

		i, err := strconv.ParseInt(expval, 10, 64)
		vvv, err := vars.NewValue(i)
		testutils.NoError(t, err)

		strs := strconv.FormatInt(int64(i), 10)
		testutils.Equal(t, strs, vvv.String())
	})
}
