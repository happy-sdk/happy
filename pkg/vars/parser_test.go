// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package vars_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/vars"
)

func TestErrors(t *testing.T) {
	_, err := vars.ParseVariableAs("", "", false, vars.KindString)
	testutils.ErrorIs(t, err, vars.ErrKey)
}

type keyValueParseTest struct {
	Key      string
	WantKey  string
	Val      string
	WantVal  string
	WantValq string
	Err      error
	Fuzz     bool
}

func getKeyValueParseTests() []keyValueParseTest {
	return []keyValueParseTest{
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
		{"key", "key", "\x81", "\x81", "\\x81", nil, false},
		{"key", "key", "\xC3\xA9", "\xC3\xA9", "é", nil, false},
		{"key", "key", "\xC3", "\xC3", "\\xc3", nil, false},
		{"key", "key", "\xE2\x82\xAC", "\xE2\x82\xAC", "€", nil, false},
		{"key", "key", "\xE2\x82", "\xE2\x82", "\\xe2\\x82", nil, false},
		{"key", "key", "\xF0\x9F\x92\xA9", "\xF0\x9F\x92\xA9", "💩", nil, false},
		{"key", "key", "\xF0\x9F\x92", "\xF0\x9F\x92", "\\xf0\\x9f\\x92", nil, false},
		{"key", "key", "\xF8\x80\x80\x80", "\xF8\x80\x80\x80", "\\xf8\\x80\\x80\\x80", nil, false},
		// special cases
		{"key", "key", "0-", "0-", "0-", nil, false},
		{"key", "key", "+0_0", "+0_0", "+0_0", nil, false},
		{"key", "key", "1", "1", "1", nil, false},
		{"key", "key", "Ǫ̃", "Ǫ̃", "Ǫ̃", nil, false},
		{"key", "key", "Ȁ0", "Ȁ0", "Ȁ0", nil, false},
		{"key", "key", "\x00", "\x00", "\\x00", nil, false},
		{"key", "key", "\xf3̏0", "\xf3̏0", "\\xf3̏0", nil, false},
		{"key", "key", " va\x80\xffe", "va\x80\xffe", " va\\x80\\xffe", nil, false},
		{"key", "key", "۞", "۞", "۞", nil, false},
		{"key", "key", "infi0", "infi0", "infi0", nil, false},
		{"key", "key", "inf+", "inf+", "inf+", nil, false},
		{"key", "key", "inf +", "inf +", "inf +", nil, false},
		{"key", "key", "inf-", "inf-", "inf-", nil, false},
		{"key", "key", "inf -", "inf -", "inf -", nil, false},
		{"key", "key", "\u2000", "", "\\u2000", nil, false},
		{"key", "key", "0X1FFFFFFp0", "0X1FFFFFFp0", "0X1FFFFFFp0", nil, false},
		{"key", "key", "\xf0\x8500", "\xf0\x8500", "\\xf0\\x8500", nil, false},
		{"key", "key", "⤌", "⤌", "⤌", nil, false},
		{"key", "key", "01_0E100", "01_0E100", "01_0E100", nil, false},
		{"key", "key", "\U0004faef", "\U0004faef", "\\U0004faef", nil, false},
		{"key", "key", "\xed\x81̄", "\xed\x81̄", "\\xed\\x81̄", nil, false},
		{"key", "key", "\xf0\x85\x850", "\xf0\x85\x850", "\\xf0\\x85\\x850", nil, false},
		{"key", "key", "0E0_", "0E0_", "0E0_", nil, false},
		{"key", "key", "🛡", "🛡", "🛡", nil, false},
		{"key", "key", "10000000000000000000000_0", "10000000000000000000000_0", "10000000000000000000000_0", nil, false},
		{"key", "key", "-0X0p0", "-0X0p0", "-0X0p0", nil, false},
		{"key", "key", "\xe4\xb6\xe2", "\xe4\xb6\xe2", "\\xe4\\xb6\\xe2", nil, false},
		{"key", "key", "1E0_1000", "1E0_1000", "1E0_1000", nil, false},
		{"key", "key", "Õ", "Õ", "Õ", nil, false},
		{"key", "key", "0X1000000000000000A", "0X1000000000000000A", "0X1000000000000000A", nil, false},
		{"key", "key", "\xec", "\xec", "\\xec", nil, false},
		{"key", "key", "넳̄", "넳̄", "넳̄", nil, false},
		{"key", "key", "\xd2", "\xd2", "\\xd2", nil, false},
		{"key", "key", "0_.", "0_.", "0_.", nil, false},
		{"key", "key", "\xf3\x85\x99\xee", "\xf3\x85\x99\xee", "\\xf3\\x85\\x99\\xee", nil, false},
		{"key", "key", "ㄵ", "ㄵ", "ㄵ", nil, false},
		{"key", "key", "\xae.\x83\x1f0Vh\x9eV\xd6@\xe05\xb0\xe2i\xf9", "\xae.\x83\x1f0Vh\x9eV\xd6@\xe05\xb0\xe2i\xf9", "\\xae.\\x83\\x1f0Vh\\x9eV\\xd6@\\xe05\\xb0\\xe2i\\xf9", nil, false},
		{"key", "key", "\xd2 ", "\xd2", "\\xd2 ", nil, false},
		{"key", "key", "0X1000000000000000Ap0", "0X1000000000000000Ap0", "0X1000000000000000Ap0", nil, false},
		{"key", "key", ".1", ".1", ".1", nil, false},
		{"key", "key", "0X0p0", "0X0p0", "0X0p0", nil, false},
		{"key", "key", "\xf0", "\xf0", "\\xf0", nil, false},
		{"key", "key", "0X1000007p0", "0X1000007p0", "0X1000007p0", nil, false},
		{"key", "key", "\xae", "\xae", "\\xae", nil, false},
		{"key", "key", "_0", "_0", "_0", nil, false},
		{"key", "key", "0X1p-200", "0X1p-200", "0X1p-200", nil, false},
		{"key", "key", "0+0", "0+0", "0+0", nil, false},
		{"key", "key", "0XA", "0XA", "0XA", nil, false},
		{"key", "key", "1E-310", "1E-310", "1E-310", nil, false},
		{"key", "key", "\xe2", "\xe2", "\\xe2", nil, false},
		{"key", "key", "0X0", "0X0", "0X0", nil, false},
		{"key", "key", "0_0.0", "0_0.0", "0_0.0", nil, false},
		{"key", "key", "0ᆰ", "0ᆰ", "0ᆰ", nil, false},
		{"key", "key", "넳̄̄", "넳̄̄", "넳̄̄", nil, false},
		{"key", "key", "0E0_0", "0E0_0", "0E0_0", nil, false},
		{"key", "key", "1E010_0", "1E010_0", "1E010_0", nil, false},
		{"key", "key", "0X_0p0", "0X_0p0", "0X_0p0", nil, false},
		{"key", "key", "0X10000000p0", "0X10000000p0", "0X10000000p0", nil, false},
		{"key", "key", "0X1p1000", "0X1p1000", "0X1p1000", nil, false},
		{"key", "key", "\xe2\x800", "\xe2\x800", "\\xe2\\x800", nil, false},
		{"key", "key", "1234600x", "1234600x", "1234600x", nil, true},
		// {"key", "key", "", "", "", nil, false},
	}
}

func TestParseVariableFromString(t *testing.T) {
	tests := getKeyValueParseTests()
	for _, test := range tests {
		kv := fmt.Sprintf("%s=%s", test.Key, test.Val)
		t.Run(kv, func(t *testing.T) {
			v, err := vars.ParseVariableFromString(kv)

			testutils.ErrorIs(t, err, test.Err, kv)
			if err == nil {
				testutils.Equal(t, vars.KindString, v.Kind())
				testutils.EqualAnyf(t, test.WantVal, v.Any(), "val1.Underlying = in(%s)", test.Val)
			} else {
				testutils.Equal(t, vars.KindInvalid, v.Kind())
				testutils.EqualAnyf(t, nil, v.Any(), "val1.Underlying = in(%s)", test.Val)
			}
			testutils.Equal(t, test.WantKey, v.Name(), "key1 = in(%s)", test.Key)
			testutils.Equal(t, test.WantVal, v.String(), "val1.String = in(%s)", test.Val)

			if strings.Contains(test.Key, "=") {
				return
			}
			kvq := fmt.Sprintf("%q=%q", test.Key, test.Val)
			vq, err := vars.ParseVariableFromString(kvq)
			testutils.ErrorIs(t, err, test.Err, kv)
			if err == nil {
				testutils.Equal(t, vars.KindString, vq.Kind())
				testutils.EqualAnyf(t, test.WantValq, vq.Any(), "val2.Underlying = in(%q)", test.Val)
			} else {
				testutils.Equal(t, vars.KindInvalid, vq.Kind())
				testutils.EqualAnyf(t, nil, vq.Any(), "val2.Underlying = in(%q)", test.Val)
			}
			testutils.Equal(t, test.WantKey, vq.Name(), "key2  in(%q)", test.Key)
			testutils.Equal(t, test.WantValq, vq.String(), "val2.String = in(%q)", test.Val)
		})

	}
	v, err := vars.ParseVariableFromString("X=1")
	testutils.Equal(t, "X", v.Name())
	testutils.Assert(t, !v.Empty())
	testutils.Equal(t, 1, v.Int())
	testutils.EqualAny(t, err, nil)
}

func TestParseVariableFromStringEmpty(t *testing.T) {
	v, err := vars.ParseVariableFromString("")
	testutils.Assert(t, v.Empty())
	testutils.Error(t, err)
	testutils.ErrorIs(t, err, vars.ErrKey)
}

func TestParseVariableFromStringEmptyKey(t *testing.T) {
	_, err := vars.ParseVariableFromString("=val")
	testutils.Error(t, err)
	testutils.ErrorIs(t, err, vars.ErrKey)
}
