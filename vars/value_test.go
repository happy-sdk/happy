// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars_test

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/happy-sdk/testutils"
	"github.com/happy-sdk/vars"
	"golang.org/x/text/unicode/norm"
)

const (
	errNone = 1 << iota
	errInt
	errInt8
	errInt16
	errInt32
	errInt64
	errUint
	errUint8
	errUint16
	errUint32
	errUint64
	errUintptr
)

// parseValueStd is std library equivalent for value parser from string
func parseValueStd(val any) string {
	str := fmt.Sprint(val)
	str = norm.NFC.String(str)
	str = strings.TrimSpace(str)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	return str
}

func TestParseValueStd(t *testing.T) {
	for _, test := range getStringTests() {
		stdStr := parseValueStd(test.Val)
		varsStr, err := vars.NewValue(test.Val)
		testutils.NoError(t, err)
		testutils.Equal(t, stdStr, varsStr.String())
	}
}

type boolTest struct {
	Key  string
	In   string
	Want bool
	Err  error
}

func getBoolTests() []boolTest {
	return []boolTest{
		{"ATOB_1", "", false, vars.ErrValueConv},
		{"ATOB_2", "asdf", false, vars.ErrValueConv},
		{"ATOB_3", "false1", false, vars.ErrValueConv},
		{"ATOB_4", "0", false, nil},
		{"ATOB_5", "f", false, nil},
		{"ATOB_6", "F", false, nil},
		{"ATOB_7", "FALSE", false, nil},
		{"ATOB_8", "false", false, nil},
		{"ATOB_9", "False", false, nil},
		{"ATOB_10", "true1", false, vars.ErrValueConv},
		{"ATOB_11", "1", true, nil},
		{"ATOB_12", "t", true, nil},
		{"ATOB_13", "T", true, nil},
		{"ATOB_14", "TRUE", true, nil},
		{"ATOB_15", "true", true, nil},
		{"ATOB_16", "True", true, nil},
	}
}

func TestBoolValue(t *testing.T) {
	for _, test := range getBoolTests() {
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
			v5, err := vars.New("var", v4, false)
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

type float32Test struct {
	Key         string
	In          string
	WantStr     string
	WantFloat32 float32
	Err         error
}

func getFloat32Tests() []float32Test {
	return []float32Test{
		{"FLOAT_1", "1.000000059604644775390625", "1", 1, nil},
		{"FLOAT_2", "1.000000059604644775390624", "1", 1, nil},
		{"FLOAT_3", "1.000000059604644775390626", "1.0000001", 1.0000001, nil},
		{"FLOAT_3", "+1.000000059604644775390626", "1.0000001", 1.00_0_0001, nil},
		{"FLOAT_4", "1.000000059604644775390625" + strings.Repeat("0", 10000) + "1", "1.0000001", 1.0000001, nil},
		{"FLOAT_5", "340282346638528859811704183484516925440", "3.4028235e+38", 3.4028235e+38, nil},
		{"FLOAT_6", "-340282346638528859811704183484516925440", "-3.4028235e+38", -3.4028235e+38, nil},
		{"FLOAT_7", "3.4028236e38", "+Inf", float32(math.Inf(1)), vars.ErrValueConv},
		{"FLOAT_8", "-3.4028236e38", "-Inf", float32(math.Inf(-1)), vars.ErrValueConv},
		{"FLOAT_9", "3.402823567e38", "3.4028235e+38", 3.4028235e+38, nil},
		{"FLOAT_10", "-3.402823567e38", "-3.4028235e+38", -3.4028235e+38, nil},
		{"FLOAT_11", "3.4028235678e38", "+Inf", float32(math.Inf(1)), vars.ErrValueConv},
		{"FLOAT_12", "-3.4028235678e38", "-Inf", float32(math.Inf(-1)), vars.ErrValueConv},
		{"FLOAT_13", "1e-38", "1e-38", 1e-38, nil},
		{"FLOAT_14", "1e-39", "1e-39", 1e-39, nil},
		{"FLOAT_15", "1e-40", "1e-40", 1e-40, nil},
		{"FLOAT_16", "1e-41", "1e-41", 1e-41, nil},
		{"FLOAT_17", "1e-42", "1e-42", 1e-42, nil},
		{"FLOAT_18", "1e-43", "1e-43", 1e-43, nil},
		{"FLOAT_20", "1e-44", "1e-44", 1e-44, nil},
		{"FLOAT_21", "6e-45", "6e-45", 6e-45, nil},
		{"FLOAT_22", "5e-45", "6e-45", 6e-45, nil},
		{"FLOAT_23", "1e-45", "1e-45", 1e-45, nil},
		{"FLOAT_24", "2e-45", "1e-45", 1e-45, nil},
		{"FLOAT_25", "4951760157141521099596496896", "4.9517602e+27", 4.9517602e+27, nil},
		{"FLOAT_26", "200", "200", 200, nil},
		{"FLOAT_27", "16", "16", 5 ^ 21, nil},
		{"FLOAT_28", "5987654321.1234578", "5987654321.1234578", 5987654321.123456789123456789123456789123456789123456789123456789123456789123456789123456789123456789, nil},
		{"FLOAT_29", "-Inf", "-Inf", float32(math.Inf(-1)), nil},
	}
}

func TestFloat32Value(t *testing.T) {
	for _, test := range getFloat32Tests() {
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

type float64Test struct {
	Key         string
	In          string
	WantStr     string
	WantFloat64 float64
	Err         error
}

func getFloat64Tests() []float64Test {
	return []float64Test{
		{"FLOAT_1", "", "0", 0, vars.ErrValueConv},
		{"FLOAT_2", "1", "1", 1, nil},
		{"FLOAT_3", "+1", "1", 1, nil},
		{"FLOAT_4", "1x", "0", 0, vars.ErrValueConv},
		{"FLOAT_5", "1.1.", "0", 0, vars.ErrValueConv},
		{"FLOAT_6", "1e23", "1e+23", 1e+23, nil},
		{"FLOAT_7", "1E23", "1e+23", 1e+23, nil},
		{"FLOAT_8", "100000000000000000000000", "1e+23", 1e+23, nil},
		{"FLOAT_9", "1e-100", "1e-100", 1e-100, nil},
		{"FLOAT_10", "123456700", "1.234567e+08", 1.234567e+08, nil},
		{"FLOAT_11", "99999999999999974834176", "9.999999999999997e+22", 9.999_999_999_999997e+22, nil},
		{"FLOAT_12", "100000000000000000000001", "1.0000000000000001e+23", 1.0000000000000001e+23, nil},
		{"FLOAT_13", "100000000000000008388608", "1.0000000000000001e+23", 1.0000000000000001e+23, nil},
		{"FLOAT_14", "100000000000000016777215", "1.0000000000000001e+23", 1.0000000000000001e+23, nil},
		{"FLOAT_15", "100000000000000016777216", "1.0000000000000003e+23", 1.0000000000000003e+23, nil},
		{"FLOAT_16", "-1", "-1", -1, nil},
		{"FLOAT_17", "-0.1", "-0.1", -0.1, nil},
		{"FLOAT_17.1", "+0.1", "0.1", 0.1, nil},
		{"FLOAT_18", "-0", "-0", 0, nil},
		{"FLOAT_19", "1e-20", "1e-20", 1e-20, nil},
		{"FLOAT_20", "625e-3", "0.625", 0.625, nil},
		{"FLOAT_21", "0", "0", 0, nil},
		{"FLOAT_22", "0e0", "0", 0, nil},
		{"FLOAT_24", "-0e0", "-0", 0, nil},
		{"FLOAT_25", "+0e0", "0", 0, nil},
		{"FLOAT_26", "0e-0", "0", 0, nil},
		{"FLOAT_27", "-0e-0", "-0", 0, nil},
		{"FLOAT_28", "+0e-0", "0", 0, nil},
		{"FLOAT_29", "0e+0", "0", 0, nil},
		{"FLOAT_30", "-0e+0", "-0", 0, nil},
		{"FLOAT_31", "+0e+0", "0", 0, nil},
		{"FLOAT_32", "0e+01234567890123456789", "0", 0, nil},
		{"FLOAT_33", "0.00e-01234567890123456789", "0", 0, nil},
		{"FLOAT_34", "-0e+01234567890123456789", "-0", 0, nil},
		{"FLOAT_35", "-0.00e-01234567890123456789", "-0", 0, nil},
		{"FLOAT_36", "0e291", "0", 0, nil},
		{"FLOAT_37", "0e292", "0", 0, nil},
		{"FLOAT_38", "0e347", "0", 0, nil},
		{"FLOAT_39", "0e348", "0", 0, nil},
		{"FLOAT_40", "-0e291", "-0", 0, nil},
		{"FLOAT_41", "-0e292", "-0", 0, nil},
		{"FLOAT_42", "-0e347", "-0", 0, nil},
		{"FLOAT_43", "-0e348", "-0", 0, nil},
		{"FLOAT_44", "nan", "NaN", math.NaN(), nil},
		{"FLOAT_45", "NaN", "NaN", math.NaN(), nil},
		{"FLOAT_46", "NAN", "NaN", math.NaN(), nil},
		{"FLOAT_47", "inf", "+Inf", math.Inf(1), nil},
		{"FLOAT_48", "-Inf", "-Inf", math.Inf(-1), nil},
		{"FLOAT_49", "+INF", "+Inf", math.Inf(1), nil},
		{"FLOAT_50", "-Infinity", "-Inf", math.Inf(-1), nil},
		{"FLOAT_51", "+INFINITY", "+Inf", math.Inf(1), nil},
		{"FLOAT_52", "Infinity", "+Inf", math.Inf(1), nil},
		{"FLOAT_53", "1.7976931348623157e308", "1.7976931348623157e+308", 1.7976931348623157e+308, nil},
		{"FLOAT_54", "-1.7976931348623157e308", "-1.7976931348623157e+308", -1.7976931348623157e+308, nil},
		{"FLOAT_55", "1.7976931348623159e308", "+Inf", math.Inf(1), vars.ErrValueConv},
		{"FLOAT_56", "-1.7976931348623159e308", "-Inf", math.Inf(-1), vars.ErrValueConv},
		{"FLOAT_57", "1.7976931348623158e308", "1.7976931348623157e+308", 1.7976931348623157e+308, nil},
		{"FLOAT_58", "-1.7976931348623158e308", "-1.7976931348623157e+308", -1.7976931348623157e+308, nil},
		{"FLOAT_59", "1.797693134862315808e308", "+Inf", math.Inf(1), vars.ErrValueConv},
		{"FLOAT_60", "-1.797693134862315808e308", "-Inf", math.Inf(-1), vars.ErrValueConv},
		{"FLOAT_61", "1e308", "1e+308", 1e+308, nil},
		{"FLOAT_62", "2e308", "+Inf", math.Inf(1), vars.ErrValueConv},
		{"FLOAT_63", "1e309", "+Inf", math.Inf(1), vars.ErrValueConv},
		{"FLOAT_64", "1e310", "+Inf", math.Inf(1), vars.ErrValueConv},
		{"FLOAT_65", "-1e310", "-Inf", math.Inf(-1), vars.ErrValueConv},
		{"FLOAT_66", "1e400", "+Inf", math.Inf(1), vars.ErrValueConv},
		{"FLOAT_67", "-1e400", "-Inf", math.Inf(-1), vars.ErrValueConv},
		{"FLOAT_68", "1e400000", "+Inf", math.Inf(1), vars.ErrValueConv},
		{"FLOAT_69", "-1e400000", "-Inf", math.Inf(-1), vars.ErrValueConv},
		{"FLOAT_70", "1e-305", "1e-305", 1e-305, nil},
		{"FLOAT_71", "1e-306", "1e-306", 1e-306, nil},
		{"FLOAT_72", "1e-307", "1e-307", 1e-307, nil},
		{"FLOAT_73", "1e-308", "1e-308", 1e-308, nil},
		{"FLOAT_74", "1e-309", "1e-309", 1e-309, nil},
		{"FLOAT_75", "1e-310", "1e-310", 1e-310, nil},
		{"FLOAT_76", "1e-322", "1e-322", 1e-322, nil},
		{"FLOAT_77", "5e-324", "5e-324", 5e-324, nil},
		{"FLOAT_78", "4e-324", "5e-324", 5e-324, nil},
		{"FLOAT_79", "3e-324", "5e-324", 5e-324, nil},
		{"FLOAT_80", "2e-324", "0", 0, nil},
		{"FLOAT_81", "1e-350", "0", 0, nil},
		{"FLOAT_82", "1e-400000", "0", 0, nil},
		{"FLOAT_83", "1e-4294967296", "0", 0, nil},
		{"FLOAT_84", "1e+4294967296", "+Inf", math.Inf(1), vars.ErrValueConv},
		{"FLOAT_85", "1e-18446744073709551616", "0", 0, nil},
		{"FLOAT_86", "1e+18446744073709551616", "+Inf", math.Inf(1), vars.ErrValueConv},
		{"FLOAT_87", "1e", "0", 0, vars.ErrValueConv},
		{"FLOAT_88", "1e-", "0", 0, vars.ErrValueConv},
		{"FLOAT_89", ".e-1", "0", 0, vars.ErrValueConv},
		{"FLOAT_90", "1\x00.2", "0", 0, vars.ErrValueConv},
		{"FLOAT_91", "2.2250738585072012e-308", "2.2250738585072014e-308", 2.2250738585072014e-308, nil},
		{"FLOAT_92", "2.2250738585072011e-308", "2.225073858507201e-308", 2.225073858507201e-308, nil},
		{"FLOAT_93", "4.630813248087435e+307", "4.630813248087435e+307", 4.630813248087435e+307, nil},
		{"FLOAT_94", "22.222222222222222", "22.22222222222222", 22.22222222222222, nil},
		{"FLOAT_95", "2." + strings.Repeat("2", 40) + "e+1", "22.22222222222222", 22.22222222222222, nil},
		{"FLOAT_96", "1.00000000000000011102230246251565404236316680908203125", "1", 1, nil},
		{"FLOAT_97", "1.00000000000000011102230246251565404236316680908203124", "1", 1, nil},
		{"FLOAT_98", "1.00000000000000011102230246251565404236316680908203126", "1.0000000000000002", 1.0000000000000002, nil},
		{"FLOAT_99", "1.00000000000000011102230246251565404236316680908203125" + strings.Repeat("0", 10000) + "1", "1.0000000000000002", 1.0000000000000002, nil},
		{"FLOAT_100", "NaN", "NaN", -math.NaN(), nil},
		{"FLOAT_101", "NaN", "NaN", +math.NaN(), nil},
		{"FLOAT_102", "NaN", "NaN", math.NaN(), nil},
	}
}

func TestFloat64Value(t *testing.T) {
	for _, test := range getFloat64Tests() {
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

type Complex64Test struct {
	Key           string
	In            string
	WantStr       string
	WantComplex64 complex64
	Err           error
}

func getComplex64Tests() []Complex64Test {
	return []Complex64Test{
		{"COMPLEX64_1", "1.000000059604644775390625 1.000000059604644775390624", "1 1", complex64(complex(1.000000059604644775390625, 1.000000059604644775390624)), nil},
		{"COMPLEX64_2", "1", "(1+0i)", complex64(1), nil},
		{"COMPLEX64_3", "1.000000059604644775390626 2", "1.0000001 2", complex(1.0000001, 2), nil},
		{"COMPLEX64_4", "1x -0", "(0+0i)", complex64(0), vars.ErrValueConv},
		{"COMPLEX64_5", "-0 1x", "(0+0i)", complex64(0), vars.ErrValueConv},
	}
}

func TestComplex64Value(t *testing.T) {
	for _, test := range getComplex64Tests() {
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

type complex128Test struct {
	Key            string
	In             string
	WantStr        string
	WantComplex128 complex128
	Err            error
}

func getComplex128Tests() []complex128Test {
	return []complex128Test{
		{"COMPLEX128_1", " 1", "(0+0i)", complex128(0), vars.ErrValueConv},
		{"COMPLEX128_2", "+1 -1", "1 -1", complex128(complex(1, -1)), nil},
		{"COMPLEX128_3", "1x -0", "(0+0i)", complex128(0), vars.ErrValueConv},
		{"COMPLEX128_3", "-0 1x", "(0+0i)", complex128(0), vars.ErrValueConv},
		{"COMPLEX128_4", "1.1. 0", "(0+0i)", complex128(0), vars.ErrValueConv},
		{"COMPLEX128_5", "1e23 1E23", "1e+23 1e+23", complex128(complex(1e+23, 1e+23)), nil},
		{"COMPLEX128_6", "100000000000000000000000 1e-100", "1e+23 1e-100", complex128(complex(1e+23, 1e-100)), nil},
		{"COMPLEX128_7", "123456700 1e-100", "1.234567e+08 1e-100", complex128(complex(1.234567e+08, 1e-100)), nil},
		{"COMPLEX128_8", "99999999999999974834176 100000000000000000000001", "9.999999999999997e+22 1.0000000000000001e+23", complex128(complex(9.999999999999997e+22, 1.0000000000000001e+23)), nil},
		{"COMPLEX128_9", "100000000000000008388608 100000000000000016777215", "1.0000000000000001e+23 1.0000000000000001e+23", complex128(complex(1.0000000000000001e+23, 1.0000000000000001e+23)), nil},
		{"COMPLEX128_10", "1e-20 625e-3", "1e-20 0.625", complex128(complex(1e-20, 0.625)), nil},
	}
}

func TestComplex128Value(t *testing.T) {
	for _, test := range getComplex128Tests() {
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

type intTest struct {
	Key   string
	Val   string
	Int   int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64
	Errs  uint
}

func checkIntErrors(val string, flags uint, flag uint, err error) error {
	if (flags&errNone != 0 || flags&flag == 0) && err != nil {
		return fmt.Errorf("%s: did not expect error got %#v", val, err)
	}

	if flags&flag != 0 && !errors.Is(err, vars.ErrValueConv) {
		return fmt.Errorf("%s: expected vars.ErrValueConv got %#v", val, err)
	}
	return nil
}

func getIntTests() []intTest {
	return []intTest{
		{"INT_1", "", 0, 0, 0, 0, 0, errInt | errInt8 | errInt16 | errInt32 | errInt64},
		{"INT_2", "0", 0, 0, 0, 0, 0, errNone},
		{"INT_3", "-0", 0, 0, 0, 0, 0, errNone},
		{"INT_4", "1", 1, 1, 1, 1, 1, errNone},
		{"INT_5", "-1", -1, -1, -1, -1, -1, errNone},
		{"INT_6", "12345", 12345, 127, 12345, 12345, 12345, errInt8},
		{"INT_6", "12345", 12_345, 12_7, 12_345, 12_345, 12_345, errInt8},
		{"INT_6", "+12345", 12345, 127, 12345, 12345, 12345, errInt8},
		{"INT_7", "-12345", -12345, -128, -12345, -12345, -12345, errInt8},
		{"INT_8", "012345", 12345, 127, 12345, 12345, 12345, errInt8},
		{"INT_9", "-012345", -12345, -128, -12345, -12345, -12345, errInt8},
		{"INT_10", "32767", 32767, 127, 32767, 32767, 32767, errInt8},
		{"INT_11", "-32768", -32768, -128, -32768, -32768, -32768, errInt8},
		{"INT_12", "32768", 32768, 127, 32767, 32768, 32768, errInt8 | errInt16},
		{"INT_13", "-32769", -32769, -128, -32768, -32769, -32769, errInt8 | errInt16},
		{"INT_14", "2147483647", 2147483647, 127, 32767, 2147483647, 2147483647, errInt8 | errInt16},
		{"INT_15", "2147483648", 2147483648, 127, 32767, 2147483647, 2147483648, errInt8 | errInt16 | errInt32},
		{"INT_16", "2147483647", 1<<31 - 1, 127, 32767, 1<<31 - 1, 1<<31 - 1, errInt8 | errInt16},
		{"INT_17", "98765432100", 98765432100, 127, 32767, 2147483647, 98765432100, errInt8 | errInt16 | errInt32},
		{"INT_18", "-98765432100", -98765432100, -128, -32768, -2147483648, -98765432100, errInt8 | errInt16 | errInt32},
		{"INT_19", "127x", 0, 0, 0, 0, 0, errInt | errInt8 | errInt16 | errInt32 | errInt64},
		{"INT_20", "1271", 1271, 127, 1271, 1271, 1271, errInt8},
		{"INT_21", "32768x", 0, 127, 0, 0, 0, errInt | errInt8 | errInt16 | errInt32 | errInt64},
		{"INT_22", "327681x", 0, 127, 32767, 0, 0, errInt | errInt8 | errInt16 | errInt32 | errInt64},
		{"INT_23", "2147483647x", 0, 127, 32767, 0, 0, errInt | errInt8 | errInt16 | errInt32 | errInt64},
		{"INT_24", "21474836471x", 0, 127, 32767, 2147483647, 0, errInt | errInt8 | errInt16 | errInt32 | errInt64},
		{"INT_25", "2147483647", 1<<31 - 1, 127, 32767, 1<<31 - 1, 1<<31 - 1, errInt8 | errInt16},
		{"INT_26", "-2147483647", -(1<<31 - 1), -128, -32768, -(1<<31 - 1), -(1<<31 - 1), errInt8 | errInt16},
		{"INT_27", strconv.FormatInt(math.MaxInt64, 10), int(math.MaxInt64), math.MaxInt8, math.MaxInt16, math.MaxInt32, math.MaxInt64, errInt8 | errInt16 | errInt32},
	}
}

func TestIntValue(t *testing.T) {
	for _, test := range getIntTests() {
		t.Run(fmt.Sprintf("%s(int): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int()
			testutils.Equal(t, test.Int, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt, err))
			i2, err := v2.Int()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt, err))
			if err == nil {
				testutils.Equal(t, test.Int, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(int8): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int8()
			testutils.Equal(t, test.Int8, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt8, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt8)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt8, err))
			i2, err := v2.Int8()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt8, err))
			if err == nil {
				testutils.Equal(t, test.Int8, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(int16): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int16()
			testutils.Equal(t, test.Int16, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt16, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt16)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt16, err))
			i2, err := v2.Int16()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt16, err))
			if err == nil {
				testutils.Equal(t, test.Int16, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(int32): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int32()
			testutils.Equal(t, test.Int32, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt32, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt32)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt32, err))
			i2, err := v2.Int32()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt32, err))
			if err == nil {
				testutils.Equal(t, test.Int32, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(int64): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int64()
			testutils.Equal(t, test.Int64, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt64, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt64)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt64, err))
			i2, err := v2.Int64()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt64, err))
			if err == nil {
				testutils.Equal(t, test.Int64, i2)
			}
		})
	}
}

func TestIntValue32(t *testing.T) {
	if vars.IsHost32bit() {
		t.Skip("curretly testing on 32bit")
		return
	}
	vars.SetHost32bit()
	defer vars.RestoreHost32bit()

	for _, test := range getIntTests() {
		t.Run(fmt.Sprintf("%s(int): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int()
			testutils.Equal(t, test.Int, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt, err))
			i2, err := v2.Int()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt, err))
			if err == nil {
				testutils.Equal(t, test.Int, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(int8): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int8()
			testutils.Equal(t, test.Int8, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt8, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt8)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt8, err))
			i2, err := v2.Int8()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt8, err))
			if err == nil {
				testutils.Equal(t, test.Int8, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(int16): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int16()
			testutils.Equal(t, test.Int16, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt16, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt16)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt16, err))
			i2, err := v2.Int16()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt16, err))
			if err == nil {
				testutils.Equal(t, test.Int16, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(int32): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int32()
			testutils.Equal(t, test.Int32, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt32, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt32)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt32, err))
			i2, err := v2.Int32()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt32, err))
			if err == nil {
				testutils.Equal(t, test.Int32, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(int64): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Int64()
			testutils.Equal(t, test.Int64, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt64, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindInt64)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt64, err))
			i2, err := v2.Int64()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errInt64, err))
			if err == nil {
				testutils.Equal(t, test.Int64, i2)
			}
		})
	}
}

type uintTest struct {
	Key    string
	Val    string
	Uint   uint
	Uint8  uint8
	Uint16 uint16
	Uint32 uint32
	Uint64 uint64
	Errs   uint
}

func getUintTests() []uintTest {
	return []uintTest{
		{"UINT_1", "", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_2", "0", 0, 0, 0, 0, 0, errNone},
		{"UINT_3", "-0", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_4", "1", 1, 1, 1, 1, 1, errNone},
		{"UINT_5", "-1", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_6", "12345", 12345, 255, 12345, 12345, 12345, errUint8},
		{"UINT_6", "12345", 12345, 255, 12345, 12345, 12345, errUint8},
		{"UINT_7", "-12345", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_8", "012345", 12345, 255, 12345, 12345, 12345, errUint8},
		{"UINT_9", "-012345", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_10", "32767", 32767, 255, 32767, 32767, 32767, errUint8},
		{"UINT_11", "-32768", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_12", "32768", 32768, 255, 32768, 32768, 32768, errUint8},
		{"UINT_13", "-32769", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_14", "2147483647", 2147483647, 255, 65535, 2147483647, 2147483647, errUint8 | errUint16},
		{"UINT_15", "2147483648", 2147483648, 255, 65535, 2147483648, 2147483648, errUint8 | errUint16},
		{"UINT_16", "2147483647", 1<<31 - 1, 255, 65535, 1<<31 - 1, 1<<31 - 1, errUint8 | errUint16},
		{"UINT_17", "98765432100", 98765432100, 255, 65535, 4294967295, 98765432100, errUint8 | errUint16 | errUint32},
		{"UINT_18", "-98765432100", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_19", "127x", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_20", "1271", 1271, 255, 1271, 1271, 1271, errUint8},
		{"UINT_21", "32768x", 0, 255, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_22", "327681x", 0, 255, 65535, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_23", "2147483647x", 0, 255, 65535, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_24", "21474836471x", 0, 255, 65535, 4294967295, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_25", "2147483647", 1<<31 - 1, 255, 65535, 1<<31 - 1, 1<<31 - 1, errUint8 | errUint16},
		{"UINT_26", "-2147483647", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_27", strconv.FormatUint(math.MaxUint64, 10), uint(math.MaxUint64), math.MaxUint8, math.MaxUint16, math.MaxUint32, math.MaxUint64, errUint8 | errUint16 | errUint32},
		{"UINT_28", "\x80", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
		{"UINT_28", "\x88", 0, 0, 0, 0, 0, errUint | errUint8 | errUint16 | errUint32 | errUint64},
	}
}

func TestUintValue(t *testing.T) {
	for _, test := range getUintTests() {
		t.Run(fmt.Sprintf("%s(uint): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint()
			testutils.Equal(t, test.Uint, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint, err))
			i2, err := v2.Uint()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint, err))
			testutils.Equal(t, test.Uint, i2)
		})

		t.Run(fmt.Sprintf("%s(uint8): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint8()
			testutils.Equal(t, test.Uint8, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint8, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint8)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint8, err))
			i2, err := v2.Uint8()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint8, err))
			if err == nil {
				testutils.Equal(t, test.Uint8, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(uint16): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint16()
			testutils.Equal(t, test.Uint16, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint16, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint16)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint16, err))
			i2, err := v2.Uint16()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint16, err))
			if err == nil {
				testutils.Equal(t, test.Uint16, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(uint32): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint32()
			testutils.Equal(t, test.Uint32, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint32, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint32)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint32, err))
			i2, err := v2.Uint32()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint32, err))
			if err == nil {
				testutils.Equal(t, test.Uint32, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(uint64): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint64()
			testutils.Equal(t, test.Uint64, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint64, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint64)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint64, err))
			i2, err := v2.Uint64()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint64, err))
			testutils.Equal(t, test.Uint64, i2)
		})
	}
}

func TestUintValue32(t *testing.T) {
	if vars.IsHost32bit() {
		t.Skip("curretly testing on 32bit")
		return
	}
	vars.SetHost32bit()
	defer vars.RestoreHost32bit()

	for _, test := range getUintTests() {
		t.Run(fmt.Sprintf("%s(uint): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint()
			testutils.Equal(t, test.Uint, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint, err))
			i2, err := v2.Uint()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint, err))
			testutils.Equal(t, test.Uint, i2)
		})

		t.Run(fmt.Sprintf("%s(uint8): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint8()
			testutils.Equal(t, test.Uint8, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint8, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint8)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint8, err))
			i2, err := v2.Uint8()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint8, err))
			if err == nil {
				testutils.Equal(t, test.Uint8, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(uint16): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint16()
			testutils.Equal(t, test.Uint16, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint16, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint16)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint16, err))
			i2, err := v2.Uint16()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint16, err))
			if err == nil {
				testutils.Equal(t, test.Uint16, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(uint32): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint32()
			testutils.Equal(t, test.Uint32, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint32, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint32)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint32, err))
			i2, err := v2.Uint32()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint32, err))
			if err == nil {
				testutils.Equal(t, test.Uint32, i2)
			}
		})

		t.Run(fmt.Sprintf("%s(uint64): %q", test.Key, test.Val), func(t *testing.T) {
			v1, err := vars.NewValue(test.Val)
			testutils.NoError(t, err)
			testutils.Equal(t, test.Val, v1.String())
			i1, err := v1.Uint64()
			testutils.Equal(t, test.Uint64, i1)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint64, err))

			v2, err := vars.ParseValueAs(test.Val, vars.KindUint64)
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint64, err))
			i2, err := v2.Uint64()
			testutils.NoError(t, checkIntErrors(test.Val, test.Errs, errUint64, err))
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
		testutils.NoError(t, checkIntErrors(test.val, test.errs, errUint64, err))

		v2, err := vars.ParseValueAs(test.val, vars.KindUintptr)
		testutils.NoError(t, checkIntErrors(test.val, test.errs, errUintptr, err))
		i2, err := v2.Uintptr()
		testutils.NoError(t, err)
		testutils.Equal(t, test.want, i2)
	}
}

type stringTest struct {
	Key string
	Val string
}

func getStringTests() []stringTest {
	return []stringTest{
		{"key", "val"},
		{"GOARCH", "amd64"},
		{"GOHOSTARCH", "amd"},
		{"GOHOSTOS", "linux"},
		{"GOOS", "linux"},
		{"GOPATH", "/go-workspace"},
		{"GOROOT", "/usr/lib/golang"},
		{"GOTOOLDIR", "/usr/lib/golang/pkg/tool/linux_amd64"},
		{"GCCGO", "gccgo"},
		{"CC", "gcc"},
		{"GOGCCFLAGS", "-fPIC -m64 -pthread -fmessage-length=0"},
		{"CXX", "g++"},
		{"PKG_CONFIG", "pkg-config"},
		{"CGO_ENABLED", "1"},
		{"CGO_CFLAGS", "-g -O2"},
		{"CGO_CPPFLAGS", ""},
		{"CGO_CXXFLAGS", "-g -O2"},
		{"CGO_FFLAGS", "-g -O2"},
		{"CGO_LDFLAGS", "-g -O2"},
		{"STRING_1", "some-string"},
		{"STRING_4", "1234567"},
	}
}

func TestNewValueString(t *testing.T) {
	for _, test := range getStringTests() {
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

func TestStringsFieldsFunc(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{input: "", want: []string{}},
		{input: " ", want: []string{}},
		{input: "  ", want: []string{}},
		{input: "a b c", want: []string{"a", "b", "c"}},
		{input: "a  b  c", want: []string{"a", "b", "c"}},
		{input: " a b c ", want: []string{"a", "b", "c"}},
		{input: " a  b  c ", want: []string{"a", "b", "c"}},
		{input: "a\tb\tc", want: []string{"a", "b", "c"}},
		{input: "a\tb  c", want: []string{"a", "b", "c"}},
		{input: "a \tb\tc", want: []string{"a", "b", "c"}},
		{input: "a \tb  c", want: []string{"a", "b", "c"}},
		{input: "a\tb\tc ", want: []string{"a", "b", "c"}},
		{input: "a\tb  c ", want: []string{"a", "b", "c"}},
		{input: "a \tb\tc ", want: []string{"a", "b", "c"}},
		{input: "a \tb  c ", want: []string{"a", "b", "c"}},
		{input: "a b\tc", want: []string{"a", "b", "c"}},
		{input: "a b \tc", want: []string{"a", "b", "c"}},
		{input: "a b\tc ", want: []string{"a", "b", "c"}},
		{input: "a b \tc ", want: []string{"a", "b", "c"}},
		{input: "a\nb\nc", want: []string{"a", "b", "c"}},
		{input: "a\nb  c", want: []string{"a", "b", "c"}},
		{input: "a\n b\nc", want: []string{"a", "b", "c"}},
		{input: "a\n b  c", want: []string{"a", "b", "c"}},
		{input: "a\nb\nc ", want: []string{"a", "b", "c"}},
		{input: "a\nb  c ", want: []string{"a", "b", "c"}},
		{input: "a\n b\nc ", want: []string{"a", "b", "c"}},
		{input: "a\n b  c ", want: []string{"a", "b", "c"}},
		{input: "a b\nc", want: []string{"a", "b", "c"}},
		{input: "a b \nc", want: []string{"a", "b", "c"}},
		{input: "a b\nc ", want: []string{"a", "b", "c"}},
		{input: "a b \nc ", want: []string{"a", "b", "c"}},
		{input: "a\rb\rc", want: []string{"a", "b", "c"}},
		{input: "a\rb  c", want: []string{"a", "b", "c"}},
		{input: "a\r b\rc", want: []string{"a", "b", "c"}},
		{input: "a\r b  c", want: []string{"a", "b", "c"}},
		{input: "a\rb\rc ", want: []string{"a", "b", "c"}},
		{input: "a\rb  c ", want: []string{"a", "b", "c"}},
		{input: "a\r b\rc ", want: []string{"a", "b", "c"}},
		{input: "a\r b  c ", want: []string{"a", "b", "c"}},
		{input: "a b\rc", want: []string{"a", "b", "c"}},
		{input: "a b \rc", want: []string{"a", "b", "c"}},
		{input: "a b\rc ", want: []string{"a", "b", "c"}},
		{input: "a b \rc ", want: []string{"a", "b", "c"}},
		{input: "a\nb\rc", want: []string{"a", "b", "c"}},
		{input: "a\nb  c", want: []string{"a", "b", "c"}},
		{input: "a\n b\rc", want: []string{"a", "b", "c"}},
		{input: "a\n b  c", want: []string{"a", "b", "c"}},
		{input: "a\nb\rc ", want: []string{"a", "b", "c"}},
		{input: "a\nb  c ", want: []string{"a", "b", "c"}},
		{input: "a\n b\rc ", want: []string{"a", "b", "c"}},
		{input: "a\n b  c ", want: []string{"a", "b", "c"}},
		{input: "a b\n\nc", want: []string{"a", "b", "c"}},
		{input: "a b \n\nc", want: []string{"a", "b", "c"}},
		{input: "a b\n\nc ", want: []string{"a", "b", "c"}},
		{input: "a b \n\nc ", want: []string{"a", "b", "c"}},
		{input: "a\tb\tc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\tb\tc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\tb\tc \u00A0", want: []string{"a", "b", "c"}},
		{input: "a\tb\tc \u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\nb\nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\nb\nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\nb\nc \u00A0", want: []string{"a", "b", "c"}},
		{input: "a\nb\nc \u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\rb\rc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\rb\rc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\rb\rc \u00A0", want: []string{"a", "b", "c"}},
		{input: "a\rb\rc \u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a b\nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a b\nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a b\nc \u00A0", want: []string{"a", "b", "c"}},
		{input: "a b\nc \u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\nb\rc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\nb\rc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\nb\rc \u00A0", want: []string{"a", "b", "c"}},
		{input: "a\nb\rc \u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a b\n\nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a b\n\nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a b\n\nc \u00A0", want: []string{"a", "b", "c"}},
		{input: "a b\n\nc \u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\u00A0c", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b  c", want: []string{"a", "b", "c"}},
		{input: "a \u00A0b\u00A0c", want: []string{"a", "b", "c"}},
		{input: "a \u00A0b  c", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\u00A0c ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b  c ", want: []string{"a", "b", "c"}},
		{input: "a \u00A0b\u00A0c ", want: []string{"a", "b", "c"}},
		{input: "a \u00A0b  c ", want: []string{"a", "b", "c"}},
		{input: "a\nb\u00A0c", want: []string{"a", "b", "c"}},
		{input: "a\nb \u00A0c", want: []string{"a", "b", "c"}},
		{input: "a\nb\u00A0c ", want: []string{"a", "b", "c"}},
		{input: "a\nb \u00A0c ", want: []string{"a", "b", "c"}},
		{input: "a\n\nb\u00A0c", want: []string{"a", "b", "c"}},
		{input: "a\n\nb \u00A0c", want: []string{"a", "b", "c"}},
		{input: "a\n\nb\u00A0c ", want: []string{"a", "b", "c"}},
		{input: "a\n\nb \u00A0c ", want: []string{"a", "b", "c"}},
		{input: "a b\u00A0c", want: []string{"a", "b", "c"}},
		{input: "a b \u00A0c", want: []string{"a", "b", "c"}},
		{input: "a b\u00A0c ", want: []string{"a", "b", "c"}},
		{input: "a b \u00A0c ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\nc", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\nc", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \nc", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\nc ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \nc ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\nc", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \nc", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\nc ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \nc ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\rc", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \rc", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\rc ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \rc ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\rc", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \rc", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\rc ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \rc ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\n\nc", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \n\nc", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\n\nc ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \n\nc ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\n\nc", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \n\nc", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\n\nc ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\u00A0c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b  c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a \u00A0b\u00A0c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a \u00A0b  c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\u00A0c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b  c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a \u00A0b\u00A0c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a \u00A0b  c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\nb\u00A0c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\nb \u00A0c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\nb\u00A0c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\nb \u00A0c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\nb\u00A0c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\nb \u00A0c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\nb\u00A0c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\nb \u00A0c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a b\u00A0c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a b \u00A0c\u00A0", want: []string{"a", "b", "c"}},
		{input: "a b\u00A0c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a b \u00A0c\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\rc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \rc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\rc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \rc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\rc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \rc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\rc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \rc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\n\nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \n\nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\n\nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \n\nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\n\nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \n\nc\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\n\nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \n\nc\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a b\u00A0c\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a b \u00A0c\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a b\u00A0c\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a b \u00A0c\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\nc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \nc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\nc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \nc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\nc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \nc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \nc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\rc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \rc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\rc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \rc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\rc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \rc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\rc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \rc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\n\nc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \n\nc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b\n\nc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\u00A0b \n\nc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\n\nc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \n\nc\u00A0\u00A0", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b\n\nc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
		{input: "a\n\u00A0b \n\nc\u00A0\u00A0 ", want: []string{"a", "b", "c"}},
	}

	for _, test := range tests {
		v, err := vars.NewValue(test.input)
		testutils.NoError(t, err, test.input)
		testutils.EqualAny(t, v.Fields(), test.want)
	}
}

type itob64Test struct {
	in   int64
	base int
	out  string
}

var itob64tests = []itob64Test{
	{0, 10, "0"},
	{1, 10, "1"},
	{-1, 10, "-1"},
	{12345678, 10, "12345678"},
	{-987654321, 10, "-987654321"},
	{1<<31 - 1, 10, "2147483647"},
	{-1<<31 + 1, 10, "-2147483647"},
	{1 << 31, 10, "2147483648"},
	{-1 << 31, 10, "-2147483648"},
	{1<<31 + 1, 10, "2147483649"},
	{-1<<31 - 1, 10, "-2147483649"},
	{1<<32 - 1, 10, "4294967295"},
	{-1<<32 + 1, 10, "-4294967295"},
	{1 << 32, 10, "4294967296"},
	{-1 << 32, 10, "-4294967296"},
	{1<<32 + 1, 10, "4294967297"},
	{-1<<32 - 1, 10, "-4294967297"},
	{1 << 50, 10, "1125899906842624"},
	{1<<63 - 1, 10, "9223372036854775807"},
	{-1<<63 + 1, 10, "-9223372036854775807"},
	{-1 << 63, 10, "-9223372036854775808"},

	{0, 2, "0"},
	{10, 2, "1010"},
	{-1, 2, "-1"},
	{1 << 15, 2, "1000000000000000"},

	{-8, 8, "-10"},
	{057635436545, 8, "57635436545"},
	{1 << 24, 8, "100000000"},

	{16, 16, "10"},
	{-0x123456789abcdef, 16, "-123456789abcdef"},
	{1<<63 - 1, 16, "7fffffffffffffff"},
	{1<<63 - 1, 2, "111111111111111111111111111111111111111111111111111111111111111"},
	{-1 << 63, 2, "-1000000000000000000000000000000000000000000000000000000000000000"},

	{16, 17, "g"},
	{25, 25, "10"},
	{(((((17*35+24)*35+21)*35+34)*35+12)*35+24)*35 + 32, 35, "holycow"},
	{(((((17*36+24)*36+21)*36+34)*36+12)*36+24)*36 + 32, 36, "holycow"},
}

func TestItoa(t *testing.T) {
	for _, test := range itob64tests {
		v, err := vars.NewValue(test.in)
		testutils.NoError(t, err, test.in)
		if test.base == 10 {
			testutils.Equal(t, test.out, v.String())
		}
		s := v.FormatInt(test.base)
		if s != test.out {
			t.Errorf("FormatInt(%v, %v) = %v want %v",
				test.in, test.base, s, test.out)
		}

		s2 := strconv.FormatInt(test.in, test.base)
		testutils.Equal(t, s2, s, "strconv.FormatInt and v.FormatInt mismatch")

		if test.in >= 0 {
			s := v.FormatUint(test.base)
			if s != test.out {
				t.Errorf("FormatUint(%v, %v) = %v want %v",
					test.in, test.base, s, test.out)
			}
		}

	}
}

type uitob64Test struct {
	in   uint64
	base int
	out  string
}

var uitob64tests = []uitob64Test{
	{1<<63 - 1, 10, "9223372036854775807"},
	{1 << 63, 10, "9223372036854775808"},
	{1<<63 + 1, 10, "9223372036854775809"},
	{1<<64 - 2, 10, "18446744073709551614"},
	{1<<64 - 1, 10, "18446744073709551615"},
	{1<<64 - 1, 2, "1111111111111111111111111111111111111111111111111111111111111111"},
}

func TestUitoa(t *testing.T) {
	for _, test := range uitob64tests {
		v, err := vars.NewValue(test.in)
		testutils.NoError(t, err, test.in)

		if test.base == 10 {
			testutils.Equal(t, test.out, v.String())
		}
		s := v.FormatUint(test.base)
		if s != test.out {
			t.Errorf("FormatUint(%v, %v) = %v want %v",
				test.in, test.base, s, test.out)
		}

		s2 := strconv.FormatUint(test.in, test.base)
		testutils.Equal(t, s2, s, "strconv.FormatUint and v.FormatUint mismatch")
	}
}

var varlenUints = []struct {
	in  uint64
	out string
}{
	{1, "1"},
	{12, "12"},
	{123, "123"},
	{1234, "1234"},
	{12345, "12345"},
	{123456, "123456"},
	{1234567, "1234567"},
	{12345678, "12345678"},
	{123456789, "123456789"},
	{1234567890, "1234567890"},
	{12345678901, "12345678901"},
	{123456789012, "123456789012"},
	{1234567890123, "1234567890123"},
	{12345678901234, "12345678901234"},
	{123456789012345, "123456789012345"},
	{1234567890123456, "1234567890123456"},
	{12345678901234567, "12345678901234567"},
	{123456789012345678, "123456789012345678"},
	{1234567890123456789, "1234567890123456789"},
	{12345678901234567890, "12345678901234567890"},
}

func TestFormatUintVarlen(t *testing.T) {
	for _, test := range varlenUints {
		v, err := vars.NewValue(test.in)
		testutils.NoError(t, err, test.in)
		testutils.Equal(t, test.out, v.String())
		s := v.FormatUint(10)
		if s != test.out {
			t.Errorf("FormatUint(%v, 10) = %v want %v", test.in, s, test.out)
		}
	}
}

type ftoaTest struct {
	f    float64
	fmt  byte
	prec int
	s    string
}

func fdiv(a, b float64) float64 { return a / b }

const (
	below1e23 = 99999999999999974834176
	above1e23 = 100000000000000008388608
)

var ftoatests = []ftoaTest{
	{1, 'e', 5, "1.00000e+00"},
	{1, 'f', 5, "1.00000"},
	{1, 'g', 5, "1"},
	{1, 'g', -1, "1"},
	{1, 'x', -1, "0x1p+00"},
	{1, 'x', 5, "0x1.00000p+00"},
	{20, 'g', -1, "20"},
	{20, 'x', -1, "0x1.4p+04"},
	{1234567.8, 'g', -1, "1.2345678e+06"},
	{1234567.8, 'x', -1, "0x1.2d687cccccccdp+20"},
	{200000, 'g', -1, "200000"},
	{200000, 'x', -1, "0x1.86ap+17"},
	{200000, 'X', -1, "0X1.86AP+17"},
	{2000000, 'g', -1, "2e+06"},
	{1e10, 'g', -1, "1e+10"},

	// g conversion and zero suppression
	{400, 'g', 2, "4e+02"},
	{40, 'g', 2, "40"},
	{4, 'g', 2, "4"},
	{.4, 'g', 2, "0.4"},
	{.04, 'g', 2, "0.04"},
	{.004, 'g', 2, "0.004"},
	{.0004, 'g', 2, "0.0004"},
	{.00004, 'g', 2, "4e-05"},
	{.000004, 'g', 2, "4e-06"},

	{0, 'e', 5, "0.00000e+00"},
	{0, 'f', 5, "0.00000"},
	{0, 'g', 5, "0"},
	{0, 'g', -1, "0"},
	{0, 'x', 5, "0x0.00000p+00"},

	{-1, 'e', 5, "-1.00000e+00"},
	{-1, 'f', 5, "-1.00000"},
	{-1, 'g', 5, "-1"},
	{-1, 'g', -1, "-1"},

	{12, 'e', 5, "1.20000e+01"},
	{12, 'f', 5, "12.00000"},
	{12, 'g', 5, "12"},
	{12, 'g', -1, "12"},

	{123456700, 'e', 5, "1.23457e+08"},
	{123456700, 'f', 5, "123456700.00000"},
	{123456700, 'g', 5, "1.2346e+08"},
	{123456700, 'g', -1, "1.234567e+08"},

	{1.2345e6, 'e', 5, "1.23450e+06"},
	{1.2345e6, 'f', 5, "1234500.00000"},
	{1.2345e6, 'g', 5, "1.2345e+06"},

	// Round to even
	{1.2345e6, 'e', 3, "1.234e+06"},
	{1.2355e6, 'e', 3, "1.236e+06"},
	{1.2345, 'f', 3, "1.234"},
	{1.2355, 'f', 3, "1.236"},
	{1234567890123456.5, 'e', 15, "1.234567890123456e+15"},
	{1234567890123457.5, 'e', 15, "1.234567890123458e+15"},
	{108678236358137.625, 'g', -1, "1.0867823635813762e+14"},

	{1e23, 'e', 17, "9.99999999999999916e+22"},
	{1e23, 'f', 17, "99999999999999991611392.00000000000000000"},
	{1e23, 'g', 17, "9.9999999999999992e+22"},

	{1e23, 'e', -1, "1e+23"},
	{1e23, 'f', -1, "100000000000000000000000"},
	{1e23, 'g', -1, "1e+23"},

	{below1e23, 'e', 17, "9.99999999999999748e+22"},
	{below1e23, 'f', 17, "99999999999999974834176.00000000000000000"},
	{below1e23, 'g', 17, "9.9999999999999975e+22"},

	{below1e23, 'e', -1, "9.999999999999997e+22"},
	{below1e23, 'f', -1, "99999999999999970000000"},
	{below1e23, 'g', -1, "9.999999999999997e+22"},

	{above1e23, 'e', 17, "1.00000000000000008e+23"},
	{above1e23, 'f', 17, "100000000000000008388608.00000000000000000"},
	{above1e23, 'g', 17, "1.0000000000000001e+23"},

	{above1e23, 'e', -1, "1.0000000000000001e+23"},
	{above1e23, 'f', -1, "100000000000000010000000"},
	{above1e23, 'g', -1, "1.0000000000000001e+23"},

	{fdiv(5e-304, 1e20), 'g', -1, "5e-324"},   // avoid constant arithmetic
	{fdiv(-5e-304, 1e20), 'g', -1, "-5e-324"}, // avoid constant arithmetic

	{32, 'g', -1, "32"},
	{32, 'g', 0, "3e+01"},

	{100, 'x', -1, "0x1.9p+06"},
	{100, 'y', -1, "%y"},

	{math.NaN(), 'g', -1, "NaN"},
	{-math.NaN(), 'g', -1, "NaN"},
	{math.Inf(0), 'g', -1, "+Inf"},
	{math.Inf(-1), 'g', -1, "-Inf"},
	{-math.Inf(0), 'g', -1, "-Inf"},

	{-1, 'b', -1, "-4503599627370496p-52"},

	// fixed bugs
	{0.9, 'f', 1, "0.9"},
	{0.09, 'f', 1, "0.1"},
	{0.0999, 'f', 1, "0.1"},
	{0.05, 'f', 1, "0.1"},
	{0.05, 'f', 0, "0"},
	{0.5, 'f', 1, "0.5"},
	{0.5, 'f', 0, "0"},
	{1.5, 'f', 0, "2"},

	// https://www.exploringbinary.com/java-hangs-when-converting-2-2250738585072012e-308/
	{2.2250738585072012e-308, 'g', -1, "2.2250738585072014e-308"},
	// https://www.exploringbinary.com/php-hangs-on-numeric-value-2-2250738585072011e-308/
	{2.2250738585072011e-308, 'g', -1, "2.225073858507201e-308"},

	// Issue 2625.
	{383260575764816448, 'f', 0, "383260575764816448"},
	{383260575764816448, 'g', -1, "3.8326057576481645e+17"},

	// Issue 29491.
	{498484681984085570, 'f', -1, "498484681984085570"},
	{-5.8339553793802237e+23, 'g', -1, "-5.8339553793802237e+23"},

	// Issue 52187
	{123.45, '?', 0, "%?"},
	{123.45, '?', 1, "%?"},
	{123.45, '?', -1, "%?"},

	// rounding
	{2.275555555555555, 'x', -1, "0x1.23456789abcdep+01"},
	{2.275555555555555, 'x', 0, "0x1p+01"},
	{2.275555555555555, 'x', 2, "0x1.23p+01"},
	{2.275555555555555, 'x', 16, "0x1.23456789abcde000p+01"},
	{2.275555555555555, 'x', 21, "0x1.23456789abcde00000000p+01"},
	{2.2755555510520935, 'x', -1, "0x1.2345678p+01"},
	{2.2755555510520935, 'x', 6, "0x1.234568p+01"},
	{2.275555431842804, 'x', -1, "0x1.2345668p+01"},
	{2.275555431842804, 'x', 6, "0x1.234566p+01"},
	{3.999969482421875, 'x', -1, "0x1.ffffp+01"},
	{3.999969482421875, 'x', 4, "0x1.ffffp+01"},
	{3.999969482421875, 'x', 3, "0x1.000p+02"},
	{3.999969482421875, 'x', 2, "0x1.00p+02"},
	{3.999969482421875, 'x', 1, "0x1.0p+02"},
	{3.999969482421875, 'x', 0, "0x1p+02"},
	{3.999969482421875, 'x', 0, "0x1p+02"},
}

func TestFtoa(t *testing.T) {
	for i := 0; i < len(ftoatests); i++ {
		test := &ftoatests[i]

		v, err := vars.NewValue(test.f)
		testutils.NoError(t, err)
		if test.fmt == 'g' && test.prec == -1 {
			testutils.Equal(t, test.s, v.String())
		}

		s := v.FormatFloat(test.fmt, test.prec, 64)
		if s != test.s {
			t.Error("testN=64", test.f, string(test.fmt), test.prec, "want", test.s, "got", s)
		}
		s2 := strconv.FormatFloat(test.f, test.fmt, test.prec, 64)
		testutils.Equal(t, s2, s, "float64 strconv.FormatFloat and v.FormatFloat mismatch")

		if float64(float32(test.f)) == test.f && test.fmt != 'b' {
			s32 := v.FormatFloat(test.fmt, test.prec, 32)
			if s != test.s {
				t.Error("testN=32", test.f, string(test.fmt), test.prec, "want", test.s, "got", s32)
			}
			s232 := strconv.FormatFloat(test.f, test.fmt, test.prec, 64)
			testutils.Equal(t, s232, s32, "float32 strconv.FormatFloat and v.FormatFloat mismatch")
		}
	}
}

func TestFtoaPowersOfTwo(t *testing.T) {
	for exp := -2048; exp <= 2048; exp++ {
		f := math.Ldexp(1, exp)

		if !math.IsInf(f, 0) {
			v, err := vars.NewValue(f)
			testutils.NoError(t, err)
			s := v.FormatFloat('e', -1, 64)
			if x, _ := vars.New("float64", s, true); x.Float64() != f {
				t.Errorf("failed roundtrip %v => %s => %v", f, s, x)
			}
		}
		f32 := float32(f)
		if !math.IsInf(float64(f32), 0) {
			v, err := vars.NewValue(f)
			testutils.NoError(t, err)
			s := v.FormatFloat('e', -1, 32)
			if x, _ := vars.New("float32", s, true); x.Float32() != f32 {
				t.Errorf("failed roundtrip %v => %s => %v", f32, s, x.Float32())
			}
		}
	}
}

func TestFtoaRandom(t *testing.T) {
	N := int(1e4)
	if testing.Short() {
		N = 100
	}
	t.Logf("testing %d random numbers with fast and slow FormatFloat", N)
	for i := 0; i < N; i++ {
		bits := uint64(rand.Uint32())<<32 | uint64(rand.Uint32())
		x := math.Float64frombits(bits)

		v, err := vars.NewValue(x)
		testutils.NoError(t, err)

		shortFast := v.FormatFloat('g', -1, 64)
		vars.SetOptimize(false)
		shortSlow := v.FormatFloat('g', -1, 64)
		vars.SetOptimize(true)
		if shortSlow != shortFast {
			t.Errorf("%b printed as %s, want %s", x, shortFast, shortSlow)
		}

		prec := rand.Intn(12) + 5
		shortFast = v.FormatFloat('e', prec, 64)
		vars.SetOptimize(false)
		shortSlow = v.FormatFloat('e', prec, 64)
		vars.SetOptimize(true)
		if shortSlow != shortFast {
			t.Errorf("%b printed as %s, want %s", x, shortFast, shortSlow)
		}
	}
}

func TestFormatFloatInvalidBitSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic due to invalid bitSize")
		}
	}()
	v, err := vars.NewValue(3.14)
	testutils.NoError(t, err)
	_ = v.FormatFloat('g', -1, 100)
}

func TestFormatFloatIntBase(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic due to invalid base")
		}
	}()
	v, err := vars.NewValue(3.14)
	testutils.NoError(t, err)
	_ = v.FormatInt(0)
}

func TestFormatFloatUintBase(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic due to invalid base")
		}
	}()
	v, err := vars.NewValue(3.14)
	testutils.NoError(t, err)
	_ = v.FormatUint(0)
}

func TestInf(t *testing.T) {
	if v, err := vars.NewValue([]byte("INFi")); testutils.NoError(t, err) {
		if i, err := v.Int(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Int8(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Int16(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Int32(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Int64(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Uint(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Uint8(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Uint16(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Uint32(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Uint64(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Float64(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
		if i, err := v.Float32(); testutils.Error(t, err) {
			testutils.Equal(t, 0, i)
		}
	}
}
