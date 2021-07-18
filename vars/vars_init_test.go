// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package vars_test

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
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

type newTest struct {
	key string
	val interface{}
	err error
}

type boolTest struct {
	key  string
	in   string
	want bool
	err  error
}

type testInt struct {
	key   string
	val   string
	int   int
	int8  int8
	int16 int16
	int32 int32
	int64 int64
	errs  uint
}

type testUint struct {
	key    string
	val    string
	uint   uint
	uint8  uint8
	uint16 uint16
	uint32 uint32
	uint64 uint64
	errs   uint
}

type float32Test struct {
	key         string
	in          string
	wantStr     string
	wantFloat32 float32
	err         error
}

type float64Test struct {
	key         string
	in          string
	wantStr     string
	wantFloat64 float64
	err         error
}

type complex64Test struct {
	key           string
	in            string
	wantStr       string
	wantComplex64 complex64
	err           error
}

type complex128Test struct {
	key            string
	in             string
	wantStr        string
	wantComplex128 complex128
	err            error
}

type stringTest struct {
	key string
	val string
}

type typeTest struct {
	key string
	in  string
	// expected
	bool       bool
	float32    float32
	float64    float64
	complex64  complex64
	complex128 complex128
	int        int
	int8       int8
	int16      int16
	int32      int32
	int64      int64
	uint       uint
	uint8      uint8
	uint16     uint16
	uint32     uint32
	uint64     uint64
	uintptr    uintptr
	string     string
	bytes      []byte
	runes      []rune
}

func checkUintString(t *testing.T, expected uint64, str string) {
	if str != fmt.Sprintf("%d", expected) {
		t.Errorf("expected %d got %q", expected, str)
	}
}

func checkErrors(t *testing.T, val string, flags uint, flag uint, err error) {
	if (flags&errNone != 0 || flags&flag == 0) && err != nil {
		t.Errorf("did not expect error got %#v", err)
	}

	if flags&flag != 0 && !errors.Is(err, strconv.ErrSyntax) && !errors.Is(err, strconv.ErrRange) {
		t.Errorf("expected strconv.ErrRange got %#v", err)
	}
}

func checkIntString(t *testing.T, expected int64, str string) {
	if str != fmt.Sprintf("%d", expected) {
		t.Errorf("expected %d got %q", expected, str)
	}
}

var newTests = []newTest{
	{"key", "<nil>", nil},
	{"key", "val", nil},
	{"bool", true, nil},
	{"float32", float32(32), nil},
	{"float64", float64(64), nil},
	{"complex64", complex64(64), nil},
	{"complex128", complex128(128), nil},
	{"int", int(1), nil},
	{"int8", int8(8), nil},
	{"int16", int16(16), nil},
	{"int32", int32(32), nil},
	{"int64", int64(64), nil},
	{"uint", uint(1), nil},
	{"uint8", uint8(8), nil},
	{"uint16", uint16(16), nil},
	{"uint32", uint32(32), nil},
	{"uint64", uint64(64), nil},
	{"uintptr", uintptr(10), nil},
	{"string", "string", nil},
	{"byte_arr", []byte{1, 2, 3}, nil},
}

var boolTests = []boolTest{
	{"ATOB_1", "", false, strconv.ErrSyntax},
	{"ATOB_2", "asdf", false, strconv.ErrSyntax},
	{"ATOB_3", "false1", false, strconv.ErrSyntax},
	{"ATOB_4", "0", false, nil},
	{"ATOB_5", "f", false, nil},
	{"ATOB_6", "F", false, nil},
	{"ATOB_7", "FALSE", false, nil},
	{"ATOB_8", "false", false, nil},
	{"ATOB_9", "False", false, nil},
	{"ATOB_10", "true1", false, strconv.ErrSyntax},
	{"ATOB_11", "1", true, nil},
	{"ATOB_12", "t", true, nil},
	{"ATOB_13", "T", true, nil},
	{"ATOB_14", "TRUE", true, nil},
	{"ATOB_15", "true", true, nil},
	{"ATOB_16", "True", true, nil},
}

var intTests = []testInt{
	{"INT_1", "", 0, 0, 0, 0, 0, errInt | errInt8 | errInt16 | errInt32 | errInt64},
	{"INT_2", "0", 0, 0, 0, 0, 0, errNone},
	{"INT_3", "-0", 0, 0, 0, 0, 0, errNone},
	{"INT_4", "1", 1, 1, 1, 1, 1, errNone},
	{"INT_5", "-1", -1, -1, -1, -1, -1, errNone},
	{"INT_6", "12345", 12345, 127, 12345, 12345, 12345, errInt8},
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
}

var uintTests = []testUint{
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
}

var float32Tests = []float32Test{
	{"FLOAT_1", "1.000000059604644775390625", "1", 1, nil},
	{"FLOAT_2", "1.000000059604644775390624", "1", 1, nil},
	{"FLOAT_3", "1.000000059604644775390626", "1.0000001", 1.0000001, nil},
	{"FLOAT_3", "+1.000000059604644775390626", "1.0000001", 1.0000001, nil},
	{"FLOAT_4", "1.000000059604644775390625" + strings.Repeat("0", 10000) + "1", "1.0000001", 1.0000001, nil},
	{"FLOAT_5", "340282346638528859811704183484516925440", "3.4028235e+38", 3.4028235e+38, nil},
	{"FLOAT_6", "-340282346638528859811704183484516925440", "-3.4028235e+38", -3.4028235e+38, nil},
	{"FLOAT_7", "3.4028236e38", "+Inf", float32(math.Inf(1)), strconv.ErrRange},
	{"FLOAT_8", "-3.4028236e38", "-Inf", float32(math.Inf(-1)), strconv.ErrRange},
	{"FLOAT_9", "3.402823567e38", "3.4028235e+38", 3.4028235e+38, nil},
	{"FLOAT_10", "-3.402823567e38", "-3.4028235e+38", -3.4028235e+38, nil},
	{"FLOAT_11", "3.4028235678e38", "+Inf", float32(math.Inf(1)), strconv.ErrRange},
	{"FLOAT_12", "-3.4028235678e38", "-Inf", float32(math.Inf(-1)), strconv.ErrRange},
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

var float64Tests = []float64Test{
	{"FLOAT_1", "", "0", 0, strconv.ErrSyntax},
	{"FLOAT_2", "1", "1", 1, nil},
	{"FLOAT_3", "+1", "1", 1, nil},
	{"FLOAT_4", "1x", "0", 0, strconv.ErrSyntax},
	{"FLOAT_5", "1.1.", "0", 0, strconv.ErrSyntax},
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
	{"FLOAT_55", "1.7976931348623159e308", "+Inf", math.Inf(1), strconv.ErrRange},
	{"FLOAT_56", "-1.7976931348623159e308", "-Inf", math.Inf(-1), strconv.ErrRange},
	{"FLOAT_57", "1.7976931348623158e308", "1.7976931348623157e+308", 1.7976931348623157e+308, nil},
	{"FLOAT_58", "-1.7976931348623158e308", "-1.7976931348623157e+308", -1.7976931348623157e+308, nil},
	{"FLOAT_59", "1.797693134862315808e308", "+Inf", math.Inf(1), strconv.ErrRange},
	{"FLOAT_60", "-1.797693134862315808e308", "-Inf", math.Inf(-1), strconv.ErrRange},
	{"FLOAT_61", "1e308", "1e+308", 1e+308, nil},
	{"FLOAT_62", "2e308", "+Inf", math.Inf(1), strconv.ErrRange},
	{"FLOAT_63", "1e309", "+Inf", math.Inf(1), strconv.ErrRange},
	{"FLOAT_64", "1e310", "+Inf", math.Inf(1), strconv.ErrRange},
	{"FLOAT_65", "-1e310", "-Inf", math.Inf(-1), strconv.ErrRange},
	{"FLOAT_66", "1e400", "+Inf", math.Inf(1), strconv.ErrRange},
	{"FLOAT_67", "-1e400", "-Inf", math.Inf(-1), strconv.ErrRange},
	{"FLOAT_68", "1e400000", "+Inf", math.Inf(1), strconv.ErrRange},
	{"FLOAT_69", "-1e400000", "-Inf", math.Inf(-1), strconv.ErrRange},
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
	{"FLOAT_84", "1e+4294967296", "+Inf", math.Inf(1), strconv.ErrRange},
	{"FLOAT_85", "1e-18446744073709551616", "0", 0, nil},
	{"FLOAT_86", "1e+18446744073709551616", "+Inf", math.Inf(1), strconv.ErrRange},
	{"FLOAT_87", "1e", "0", 0, strconv.ErrSyntax},
	{"FLOAT_88", "1e-", "0", 0, strconv.ErrSyntax},
	{"FLOAT_89", ".e-1", "0", 0, strconv.ErrSyntax},
	{"FLOAT_90", "1\x00.2", "0", 0, strconv.ErrSyntax},
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

var complex64Tests = []complex64Test{
	{"COMPLEX64_1", "1.000000059604644775390625 1.000000059604644775390624", "1 1", complex64(complex(1.000000059604644775390625, 1.000000059604644775390624)), nil},
	{"COMPLEX64_2", "1", "", complex64(0), strconv.ErrSyntax},
	{"COMPLEX64_3", "1.000000059604644775390626 2", "1.0000001 2", complex(1.0000001, 2), nil},
	{"COMPLEX64_4", "1x -0", "", complex64(0), strconv.ErrSyntax},
	{"COMPLEX64_5", "-0 1x", "", complex64(0), strconv.ErrSyntax},
}

var complex128Tests = []complex128Test{
	{"COMPLEX128_1", " 1", "", complex128(0), strconv.ErrSyntax},
	{"COMPLEX128_2", "+1 -1", "1 -1", complex128(complex(1, -1)), nil},
	{"COMPLEX128_3", "1x -0", "", complex128(0), strconv.ErrSyntax},
	{"COMPLEX128_3", "-0 1x", "", complex128(0), strconv.ErrSyntax},
	{"COMPLEX128_4", "1.1. 0", "", complex128(0), strconv.ErrSyntax},
	{"COMPLEX128_5", "1e23 1E23", "1e+23 1e+23", complex128(complex(1e+23, 1e+23)), nil},
	{"COMPLEX128_6", "100000000000000000000000 1e-100", "1e+23 1e-100", complex128(complex(1e+23, 1e-100)), nil},
	{"COMPLEX128_7", "123456700 1e-100", "1.234567e+08 1e-100", complex128(complex(1.234567e+08, 1e-100)), nil},
	{"COMPLEX128_8", "99999999999999974834176 100000000000000000000001", "9.999999999999997e+22 1.0000000000000001e+23", complex128(complex(9.999999999999997e+22, 1.0000000000000001e+23)), nil},
	{"COMPLEX128_9", "100000000000000008388608 100000000000000016777215", "1.0000000000000001e+23 1.0000000000000001e+23", complex128(complex(1.0000000000000001e+23, 1.0000000000000001e+23)), nil},
	{"COMPLEX128_10", "1e-20 625e-3", "1e-20 0.625", complex128(complex(1e-20, 0.625)), nil},
}

var stringTests = []stringTest{
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

var typeTests = []typeTest{
	{"INT_1", "1", true, 1, 1, complex(0, 0i), complex(0, 0i), 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, "1", []byte{49}, []rune{49}},
	{"INT_2", "2147483647", false, 2.1474836e+09, 2.147483647e+09, complex(0, 0i), complex(0, 0i), 2147483647, 127, 32767, 2147483647, 2147483647, 2147483647, 255, 65535, 2147483647, 2147483647, 2147483647, "2147483647", []byte{50, 49, 52, 55, 52, 56, 51, 54, 52, 55}, []rune{'2', '1', '4', '7', '4', '8', '3', '6', '4', '7'}},
	{"STRING_1", "asdf", false, 0, 0, complex(0, 0i), complex(0, 0i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "asdf", []byte{97, 115, 100, 102}, []rune{'a', 's', 'd', 'f'}},
	{"FLOAT_1", "2." + strings.Repeat("2", 40) + "e+1", false, 22.222221, 22.22222222222222, complex(0, 0i), complex(0, 0i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, "2.2222222222222222222222222222222222222222e+1", []byte("2.2222222222222222222222222222222222222222e+1"), []rune("2.2222222222222222222222222222222222222222e+1")},
	{"COMPLEX128_1", "123456700 1e-100", false, 0, 0, complex(1.234567e+08, 0i), complex(1.234567e+08, 1e-100), 0, 127, 32767, 0, 0, 0, 255, 65535, 0, 0, 0, "123456700 1e-100", []byte("123456700 1e-100"), []rune("123456700 1e-100")},
}

func genAtoi64TestBytes() []byte {
	var out []byte
	for _, data := range intTests {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.key, data.val)
		out = append(out, []byte(line)...)
	}
	return out
}

func genAtoui64TestBytes() []byte {
	var out []byte
	for _, data := range uintTests {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.key, data.val)
		out = append(out, []byte(line)...)
	}
	return out
}

func genAtof32TestBytes() []byte {
	var out []byte
	for _, data := range float32Tests {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.key, data.in)
		out = append(out, []byte(line)...)
	}
	return out
}

func genAtofTestBytes() []byte {
	var out []byte
	for _, data := range float64Tests {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.key, data.in)
		out = append(out, []byte(line)...)
	}
	return out
}

func genAtobTestBytes() []byte {
	var out []byte
	for _, data := range boolTests {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.key, data.in)
		out = append(out, []byte(line)...)
	}
	return out
}

func genStringTestBytes() []byte {
	var out []byte
	for _, data := range stringTests {
		line := fmt.Sprintf(`%s="%s"`+"\n", data.key, data.val)
		out = append(out, []byte(line)...)
	}
	return out
}
