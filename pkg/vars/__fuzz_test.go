// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars_test

// import (
// 	"fmt"
// 	"math"
// 	"strconv"
// 	"strings"
// 	"testing"

// 	"github.com/happy-sdk/happy/pkg/devel/testutils"
// 	"github.com/happy-sdk/happy/pkg/vars"
// )

// func FuzzParseKeyValue(f *testing.F) {
// 	tests := getKeyValueParseTests()
// 	for _, test := range tests {
// 		if test.Fuzz {
// 			f.Add(test.Key, test.Val)
// 		}
// 		f.Add("key", "_1.0_1_0")
// 		f.Add("key", "_1.0_1_0.0")
// 		f.Add("key", "_10.0_1_0.01")
// 		f.Add("key", "_10.0_1_01")
// 		f.Add("key", "INF")
// 		f.Add("key", "INFI")
// 		f.Add("key", "INFI")
// 		f.Add("key", "_________________________________")
// 		f.Add("key", "10.0_1_01"+strings.Repeat("0", 500))
// 		f.Add("key", "A")
// 	}
// 	f.Fuzz(func(t *testing.T, key, val string) {
// 		parseKeyValueTest(t, key, val)
// 		vars.SetOptimize(false)
// 		parseKeyValueTest(t, key, val)
// 		vars.SetOptimize(true)

// 		vars.SetHost32bit()
// 		parseKeyValueTest(t, key, val)
// 		vars.SetOptimize(false)
// 		parseKeyValueTest(t, key, val)
// 		vars.SetOptimize(true)
// 		vars.RestoreHost32bit()
// 	})
// }

// func parseKeyValueTest(t *testing.T, key, arg string) {
// 	clean := strings.TrimSpace(arg)
// 	if strings.Contains(key, "=") || arg == "=" || clean == "\"\"" {
// 		return
// 	}

// 	kv := fmt.Sprintf("%s=%s", key, arg)
// 	v, err := vars.ParseVariableFromString(kv)
// 	if err != nil {
// 		testutils.Equal(t, vars.KindInvalid, v.Kind())
// 		return
// 	}

// 	testutils.Equal(t, vars.KindString, v.Kind())

// 	expkey, _ := vars.ParseKey(key)
// 	testutils.Equal(t, expkey, v.Name(), "key1 -> key(%s) val(%s)", key, arg)

// 	expval := parseValueStd(arg)

// 	if !testutils.Equal(t, expval, v.String(), "in = %q expval = %q v.String() = %q", arg, expval, v.String()) {
// 		testutils.EqualAny(t, expval, v.Any(), ".String -> expval(%q) val(%q) %#v %#v", expval, v.String(), []byte(expval), []byte(v.String()))
// 	}

// 	if f64, err := strconv.ParseFloat(expval, 64); err == nil {
// 		vv, err := vars.NewValue(f64)
// 		testutils.NoError(t, err)

// 		str := strconv.FormatFloat(f64, 'g', -1, 64)
// 		testutils.Equal(t, str, vv.String())
// 		if f32, err := v.Value().Float32(); err == nil {
// 			vv, err := vars.NewValue(f32)
// 			testutils.NoError(t, err)

// 			str := strconv.FormatFloat(float64(f32), 'g', -1, 32)
// 			testutils.Equal(t, str, vv.String())
// 		}

// 		val, err := vars.NewValueAs(f64, vars.KindString)
// 		testutils.NoError(t, err)
// 		for _, fmt := range []byte{'e', 'E', 'f', 'g', 'G', 'x', 'X'} {
// 			for prec := -1; prec < 69; prec++ {
// 				val32 := val.FormatFloat(fmt, prec, 32)
// 				str32 := strconv.FormatFloat(f64, fmt, prec, 32)
// 				testutils.Equal(t, str32, val32)
// 				if _, err := vars.NewAs(key, str32, false, vars.KindFloat32); err != nil {
// 					testutils.NoError(t, err)
// 				}

// 				val64 := val.FormatFloat(fmt, prec, 64)
// 				str64 := strconv.FormatFloat(f64, fmt, prec, 64)
// 				testutils.Equal(t, val64, str64)
// 				if _, err := vars.NewAs(key, str64, false, vars.KindFloat64); err != nil {
// 					testutils.NoError(t, err)
// 				}
// 			}
// 		}

// 		if !math.IsNaN(f64) {
// 			cmplx, err := val.Complex128()
// 			testutils.NoError(t, err)
// 			testutils.Equal(t, f64, real(cmplx))
// 			testutils.Equal(t, 0, imag(cmplx))
// 		}
// 	}
// 	if f32, err := strconv.ParseFloat(expval, 32); err == nil {
// 		vv, err := vars.NewValue(f32)
// 		testutils.NoError(t, err)

// 		str := strconv.FormatFloat(f32, 'g', -1, 64)
// 		testutils.Equal(t, str, vv.String())
// 		if f32, err := v.Value().Float32(); err == nil {
// 			vv, err := vars.NewValue(f32)
// 			testutils.NoError(t, err)

// 			str := strconv.FormatFloat(float64(f32), 'g', -1, 32)
// 			testutils.Equal(t, str, vv.String())
// 		}

// 		val, err := vars.NewValueAs(f32, vars.KindString)
// 		testutils.NoError(t, err)
// 		for _, fmt := range []byte{'e', 'E', 'f', 'g', 'G', 'x', 'X'} {
// 			for prec := -1; prec < 69; prec++ {
// 				val32 := val.FormatFloat(fmt, prec, 32)
// 				str32 := strconv.FormatFloat(float64(f32), fmt, prec, 32)
// 				testutils.Equal(t, str32, val32)
// 				if _, err := vars.NewAs(key, str32, false, vars.KindFloat32); err != nil {
// 					testutils.NoError(t, err)
// 				}

// 				val64 := val.FormatFloat(fmt, prec, 64)
// 				str64 := strconv.FormatFloat(float64(f32), fmt, prec, 64)
// 				testutils.Equal(t, val64, str64)
// 				if _, err := vars.NewAs(key, str64, false, vars.KindFloat64); err != nil {
// 					testutils.NoError(t, err)
// 				}
// 			}
// 		}
// 		if !math.IsNaN(f32) {
// 			cmplx, err := val.Complex64()
// 			testutils.NoError(t, err)
// 			testutils.Equal(t, f32, real(complex128(cmplx)))
// 			testutils.Equal(t, 0, imag(complex128(cmplx)))
// 		}

// 	}

// 	if _, err := strconv.ParseUint(expval, 10, 64); err == nil {
// 		for base := 2; base <= 36; base++ {
// 			if u64, err := strconv.ParseUint(expval, base, 64); err == nil {
// 				vu64, _, err := vars.ParseUint(expval, base, 64)
// 				testutils.NoError(t, err)
// 				testutils.Equal(t, u64, vu64)

// 				vvv, err := vars.NewValue(vu64)
// 				testutils.NoError(t, err)
// 				str64 := strconv.FormatUint(vu64, base)
// 				testutils.Equal(t, str64, vvv.FormatUint(base))
// 				if _, err := vars.NewAs(key, str64, false, vars.KindUint64); err != nil {
// 					testutils.NoError(t, err)
// 				}
// 			}
// 		}
// 	}
// 	if _, err := strconv.ParseInt(expval, 10, 64); err == nil {
// 		for base := 2; base <= 36; base++ {
// 			if i64, err := strconv.ParseInt(expval, base, 64); err == nil {
// 				vi64, s, err := vars.ParseInt(expval, base, 64)
// 				testutils.NoError(t, err)
// 				testutils.Equal(t, i64, vi64)

// 				vvv, err := vars.NewValue(s)
// 				testutils.NoError(t, err)
// 				str64 := strconv.FormatInt(vi64, base)
// 				testutils.Equal(t, str64, vvv.FormatInt(base))
// 				if _, err := vars.NewAs(key, str64, false, vars.KindInt64); err != nil {
// 					testutils.NoError(t, err)
// 				}
// 			}
// 		}
// 	}

// 	// if _, err := vars.NewAs(key, arg, false, vars.KindBool); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindInt); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindInt8); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindInt16); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindUint); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindUint8); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindUint16); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindUint32); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindUintptr); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindComplex64); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// 	// if _, err := vars.NewAs(key, arg, false, vars.KindComplex128); err != nil {
// 	// 	testutils.NoError(t, err)
// 	// }
// }
