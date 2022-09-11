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

package testutils

import (
	"errors"
	"fmt"
	"github.com/mkungla/happy/x/pkg/vars"
	"math"
	"strconv"
	"strings"
)

type KindTest struct {
	Key string
	In  string
	// expected
	Bool       bool
	Float32    float32
	Float64    float64
	Complex64  complex64
	Complex128 complex128
	Int        int
	Int8       int8
	Int16      int16
	Int32      int32
	Int64      int64
	Uint       uint
	Uint8      uint8
	Uint16     uint16
	Uint32     uint32
	Uint64     uint64
	Uintptr    uintptr
	String     string
	Bytes      []byte
	Runes      []rune
}

func GetKindTests() []KindTest {
	return []KindTest{
		{"INT_1", "1", true, 1, 1, complex(1, 0i), complex(1, 0i), 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, "1", []byte{49}, []rune{49}},
		{"INT_2", "2147483647", false, 2.1474836e+09, 2.147483647e+09, complex(2.147483647e+09, 0i), complex(2.147483647e+09, 0i), 2147483647, 127, 32767, 2147483647, 2147483647, 2147483647, 255, 65535, 2147483647, 2147483647, 2147483647, "2147483647", []byte{50, 49, 52, 55, 52, 56, 51, 54, 52, 55}, []rune{'2', '1', '4', '7', '4', '8', '3', '6', '4', '7'}},
		{"STRING_1", "asdf", false, 0, 0, complex(0, 0i), complex(0, 0i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "asdf", []byte{97, 115, 100, 102}, []rune{'a', 's', 'd', 'f'}},
		{"FLOAT_1", "2." + strings.Repeat("2", 40) + "e+1", false, 22.222221, 22.22222222222222, complex(22.22222222222222, 0i), complex(22.22222222222222, 0i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "2.2222222222222222222222222222222222222222e+1", []byte("2.2222222222222222222222222222222222222222e+1"), []rune("2.2222222222222222222222222222222222222222e+1")},
		{"COMPLEX128_1", "123456700 1e-100", false, 0, 0, complex(1.234567e+08, 0i), complex(1.234567e+08, 1e-100), 0, 127, 32767, 0, 0, 0, 255, 65535, 0, 0, 0, "123456700 1e-100", []byte("123456700 1e-100"), []rune("123456700 1e-100")},
	}
}

type NewTest struct {
	Key string
	Val any
}

func GetNewTests() []NewTest {
	return []NewTest{
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

type KeyValueParseTest struct {
	Key      string
	WantKey  string
	Val      string
	WantVal  string
	WantValq string
	Err      error
	Fuzz     bool
}

func GetKeyValueParseTests() []KeyValueParseTest {
	return []KeyValueParseTest{
		// keys
		{"key", "key", "value", "value", "value", nil, true},
		{"\"key\"", "key", "value", "value", "value", nil, true},
		{" key", "key", "value", "value", "value", nil, true},
		{"key ", "key", "value", "value", "value", nil, true},
		{" key ", "key", "value", "value", "value", nil, true},
		{" k e y ", "k e y", "", "", "", nil, true},
		// values
		{"key", "key", " value ", "value", " value ", nil, true},
		{"key", "key", " value", "value", " value", nil, true},
		{"key", "key", "value ", "value", "value ", nil, true},
		{"key", "key", `expected" value`, `expected" value`, `expected\" value`, nil, true},
		{`"`, "", "", "", "", vars.ErrKey, false},
		{" ", "", "", "", "", vars.ErrKey, false},
		// {"key", "key", "\x93", "\\x93", "\\x93", nil, true},
		// {"key", "key", "\x00", "", "", nil, true},
		// {"key", "key", "\xff", "\xff", "\xff", nil, false},
		{"key=", "key", "", "=", "=", nil, false},
		{"=key", "", "", "", "", vars.ErrKey, false},
		{"key", "key", "=", "=", "=", nil, false},
		{"###", "###", "value", "value", "value", nil, true},
		{"key=", "key", "value", "=value", "=value", nil, false},
	}
}

type BoolTest struct {
	Key  string
	In   string
	Want bool
	Err  error
}

func GetBoolTests() []BoolTest {
	return []BoolTest{
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

type Float32Test struct {
	Key         string
	In          string
	WantStr     string
	WantFloat32 float32
	Err         error
}

func GetFloat32Tests() []Float32Test {
	return []Float32Test{
		{"FLOAT_1", "1.000000059604644775390625", "1", 1, nil},
		{"FLOAT_2", "1.000000059604644775390624", "1", 1, nil},
		{"FLOAT_3", "1.000000059604644775390626", "1.0000001", 1.0000001, nil},
		{"FLOAT_3", "+1.000000059604644775390626", "1.0000001", 1.0000001, nil},
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
	}
}

type Float64Test struct {
	Key         string
	In          string
	WantStr     string
	WantFloat64 float64
	Err         error
}

func GetFloat64Tests() []Float64Test {
	return []Float64Test{
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
		{"FLOAT_11", "99999999999999974834176", "9.999999999999997e+22", 9.999999999999997e+22, nil},
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

type Complex64Test struct {
	Key           string
	In            string
	WantStr       string
	WantComplex64 complex64
	Err           error
}

func GetComplex64Tests() []Complex64Test {
	return []Complex64Test{
		{"COMPLEX64_1", "1.000000059604644775390625 1.000000059604644775390624", "1 1", complex64(complex(1.000000059604644775390625, 1.000000059604644775390624)), nil},
		{"COMPLEX64_2", "1", "(1+0i)", complex64(1), nil},
		{"COMPLEX64_3", "1.000000059604644775390626 2", "1.0000001 2", complex(1.0000001, 2), nil},
		{"COMPLEX64_4", "1x -0", "(0+0i)", complex64(0), vars.ErrValueConv},
		{"COMPLEX64_5", "-0 1x", "(0+0i)", complex64(0), vars.ErrValueConv},
	}
}

type Complex128Test struct {
	Key            string
	In             string
	WantStr        string
	WantComplex128 complex128
	Err            error
}

func GetComplex128Tests() []Complex128Test {
	return []Complex128Test{
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

const (
	ErrNone = 1 << iota
	ErrInt
	ErrInt8
	ErrInt16
	ErrInt32
	ErrInt64
	ErrUint
	ErrUint8
	ErrUint16
	ErrUint32
	ErrUint64
	ErrUintptr
)

type IntTest struct {
	Key   string
	Val   string
	Int   int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64
	Errs  uint
}

func CheckIntErrors(val string, flags uint, flag uint, err error) error {
	if (flags&ErrNone != 0 || flags&flag == 0) && err != nil {
		return fmt.Errorf("%s: did not expect error got %#v", val, err)
	}

	if flags&flag != 0 && !errors.Is(err, vars.ErrValueConv) {
		return fmt.Errorf("%s: expected vars.ErrValueConv got %#v", val, err)
	}
	return nil
}

func GetIntTests() []IntTest {
	return []IntTest{
		{"INT_1", "", 0, 0, 0, 0, 0, ErrInt | ErrInt8 | ErrInt16 | ErrInt32 | ErrInt64},
		{"INT_2", "0", 0, 0, 0, 0, 0, ErrNone},
		{"INT_3", "-0", 0, 0, 0, 0, 0, ErrNone},
		{"INT_4", "1", 1, 1, 1, 1, 1, ErrNone},
		{"INT_5", "-1", -1, -1, -1, -1, -1, ErrNone},
		{"INT_6", "12345", 12345, 127, 12345, 12345, 12345, ErrInt8},
		{"INT_6", "12345", 12_345, 12_7, 12_345, 12_345, 12_345, ErrInt8},
		{"INT_6", "+12345", 12345, 127, 12345, 12345, 12345, ErrInt8},
		{"INT_7", "-12345", -12345, -128, -12345, -12345, -12345, ErrInt8},
		{"INT_8", "012345", 12345, 127, 12345, 12345, 12345, ErrInt8},
		{"INT_9", "-012345", -12345, -128, -12345, -12345, -12345, ErrInt8},
		{"INT_10", "32767", 32767, 127, 32767, 32767, 32767, ErrInt8},
		{"INT_11", "-32768", -32768, -128, -32768, -32768, -32768, ErrInt8},
		{"INT_12", "32768", 32768, 127, 32767, 32768, 32768, ErrInt8 | ErrInt16},
		{"INT_13", "-32769", -32769, -128, -32768, -32769, -32769, ErrInt8 | ErrInt16},
		{"INT_14", "2147483647", 2147483647, 127, 32767, 2147483647, 2147483647, ErrInt8 | ErrInt16},
		{"INT_15", "2147483648", 2147483648, 127, 32767, 2147483647, 2147483648, ErrInt8 | ErrInt16 | ErrInt32},
		{"INT_16", "2147483647", 1<<31 - 1, 127, 32767, 1<<31 - 1, 1<<31 - 1, ErrInt8 | ErrInt16},
		{"INT_17", "98765432100", 98765432100, 127, 32767, 2147483647, 98765432100, ErrInt8 | ErrInt16 | ErrInt32},
		{"INT_18", "-98765432100", -98765432100, -128, -32768, -2147483648, -98765432100, ErrInt8 | ErrInt16 | ErrInt32},
		{"INT_19", "127x", 0, 0, 0, 0, 0, ErrInt | ErrInt8 | ErrInt16 | ErrInt32 | ErrInt64},
		{"INT_20", "1271", 1271, 127, 1271, 1271, 1271, ErrInt8},
		{"INT_21", "32768x", 0, 127, 0, 0, 0, ErrInt | ErrInt8 | ErrInt16 | ErrInt32 | ErrInt64},
		{"INT_22", "327681x", 0, 127, 32767, 0, 0, ErrInt | ErrInt8 | ErrInt16 | ErrInt32 | ErrInt64},
		{"INT_23", "2147483647x", 0, 127, 32767, 0, 0, ErrInt | ErrInt8 | ErrInt16 | ErrInt32 | ErrInt64},
		{"INT_24", "21474836471x", 0, 127, 32767, 2147483647, 0, ErrInt | ErrInt8 | ErrInt16 | ErrInt32 | ErrInt64},
		{"INT_25", "2147483647", 1<<31 - 1, 127, 32767, 1<<31 - 1, 1<<31 - 1, ErrInt8 | ErrInt16},
		{"INT_26", "-2147483647", -(1<<31 - 1), -128, -32768, -(1<<31 - 1), -(1<<31 - 1), ErrInt8 | ErrInt16},
		{"INT_27", strconv.FormatInt(math.MaxInt64, 10), int(math.MaxInt64), math.MaxInt8, math.MaxInt16, math.MaxInt32, math.MaxInt64, ErrInt8 | ErrInt16 | ErrInt32},
	}
}

type UintTest struct {
	Key    string
	Val    string
	Uint   uint
	Uint8  uint8
	Uint16 uint16
	Uint32 uint32
	Uint64 uint64
	Errs   uint
}

func GetUintTests() []UintTest {
	return []UintTest{
		{"UINT_1", "", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_2", "0", 0, 0, 0, 0, 0, ErrNone},
		{"UINT_3", "-0", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_4", "1", 1, 1, 1, 1, 1, ErrNone},
		{"UINT_5", "-1", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_6", "12345", 12345, 255, 12345, 12345, 12345, ErrUint8},
		{"UINT_6", "12345", 12345, 255, 12345, 12345, 12345, ErrUint8},
		{"UINT_7", "-12345", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_8", "012345", 12345, 255, 12345, 12345, 12345, ErrUint8},
		{"UINT_9", "-012345", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_10", "32767", 32767, 255, 32767, 32767, 32767, ErrUint8},
		{"UINT_11", "-32768", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_12", "32768", 32768, 255, 32768, 32768, 32768, ErrUint8},
		{"UINT_13", "-32769", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_14", "2147483647", 2147483647, 255, 65535, 2147483647, 2147483647, ErrUint8 | ErrUint16},
		{"UINT_15", "2147483648", 2147483648, 255, 65535, 2147483648, 2147483648, ErrUint8 | ErrUint16},
		{"UINT_16", "2147483647", 1<<31 - 1, 255, 65535, 1<<31 - 1, 1<<31 - 1, ErrUint8 | ErrUint16},
		{"UINT_17", "98765432100", 98765432100, 255, 65535, 4294967295, 98765432100, ErrUint8 | ErrUint16 | ErrUint32},
		{"UINT_18", "-98765432100", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_19", "127x", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_20", "1271", 1271, 255, 1271, 1271, 1271, ErrUint8},
		{"UINT_21", "32768x", 0, 255, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_22", "327681x", 0, 255, 65535, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_23", "2147483647x", 0, 255, 65535, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_24", "21474836471x", 0, 255, 65535, 4294967295, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_25", "2147483647", 1<<31 - 1, 255, 65535, 1<<31 - 1, 1<<31 - 1, ErrUint8 | ErrUint16},
		{"UINT_26", "-2147483647", 0, 0, 0, 0, 0, ErrUint | ErrUint8 | ErrUint16 | ErrUint32 | ErrUint64},
		{"UINT_27", strconv.FormatUint(math.MaxUint64, 10), uint(math.MaxUint64), math.MaxUint8, math.MaxUint16, math.MaxUint32, math.MaxUint64, ErrUint8 | ErrUint16 | ErrUint32},
	}
}

type StringTest struct {
	Key string
	Val string
}

func GetStringTests() []StringTest {
	return []StringTest{
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

func OnErrorMsg(key string, val any) string {
	return fmt.Sprintf("key(%v) = val(%v)", key, val)
}

func GenAtoi64TestBytes() []byte {
	var out []byte
	for _, data := range GetIntTests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.Val)
		out = append(out, []byte(line)...)
	}
	return out
}

func GenAtoui64TestBytes() []byte {
	var out []byte
	for _, data := range GetUintTests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.Val)
		out = append(out, []byte(line)...)
	}
	return out
}

func GenAtof32TestBytes() []byte {
	var out []byte
	for _, data := range GetFloat32Tests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.In)
		out = append(out, []byte(line)...)
	}
	return out
}

func GenAtofTestBytes() []byte {
	var out []byte
	for _, data := range GetFloat64Tests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.In)
		out = append(out, []byte(line)...)
	}
	return out
}

func GenAtobTestBytes() []byte {
	var out []byte
	for _, data := range GetBoolTests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.In)
		out = append(out, []byte(line)...)
	}
	return out
}

func GenStringTestBytes() []byte {
	var out []byte
	for _, data := range GetStringTests() {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.Key, data.Val)
		out = append(out, []byte(line)...)
	}
	// add empty line
	out = append(out, []byte("")...)
	return out
}

func NewUnsafeValue(val any) vars.Value {
	v, _ := vars.NewValue(val)
	return v
}
