// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func notEqualMsg(want, got interface{}) string {
	return fmt.Sprintf("not equal want(%v) got(%v)", want, got)
}

type testResource struct {
	Pattern string
	Res0    string
	Res1    string
	Last    string
	Len     int
}

// nolint: gochecknoglobals
var testResources = []testResource{
	// PATHS
	{"data/{P1/{10..19},P2/{20..29},P3/{30..39}}", "data/P1/10", "data/P1/11", "data/P3/39", 30},
	{"data/{-1..-3}", "data/-1", "data/-2", "data/-3", 3},
	{"data/{-3..-1}", "data/-3", "data/-2", "data/-1", 3},
	{"data/{-1..1}", "data/-1", "data/0", "data/1", 3},
	{"/usr/{ucb/{ex,edit},lib/{ex?.?*,how_ex}}", "/usr/ucb/ex", "/usr/ucb/edit", "/usr/lib/how_ex", 4},
	{"/usr/local/src/bash/{old,new,dist,bugs}", "/usr/local/src/bash/old", "/usr/local/src/bash/new", "/usr/local/src/bash/bugs", 4},
	// EXPRESSIONS
	{"a", "a", "", "", 1},
	{"{,}", "", "", "", 1},
	{"{,,}", "", "", "", 1},
	{"{,,,}", "", "", "", 1},
	{"\\\\", "\\", "", "", 1},
	{"-", "-", "", "", 1},
	{"{a,b}{1,2}", "a1", "a2", "b2", 4},
	{"a{d,c,b}e", "ade", "ace", "abe", 3},
	{"x{{a,b,c}}y", "x{a}y", "x{b}y", "x{c}y", 3},
	{"a,b", "a,b", "", "", 1},
	{"a,,b", "a,,b", "", "", 1},
	{"a,", "a,", "", "", 1},
	{",a", ",a", "", "", 1},
	{",", ",", "", "", 1},
	{",,", ",,", "", "", 1},
	{"abc,def", "abc,def", "", "", 1},
	{"{abc,def}", "abc", "def", "", 2},
	{"{}", "{}", "", "", 1},
	{"a{}", "a{}", "", "", 1},
	{"{a{}}", "{a{}}", "", "", 1},
	{"a{,}", "a", "a", "", 2},
	{"a{,}{,}", "a", "a", "a", 4},
	{"a{,,,}", "a", "a", "a", 4},
	{"a{b,c}", "ab", "ac", "", 2},
	{"a{,b,c}", "a", "ab", "ac", 3},
	{"a{b,c,}", "ab", "ac", "a", 3},
	{"a{b,d{e,f}}", "ab", "ade", "adf", 3},
	{"a{b,{c,d}}", "ab", "ac", "ad", 3},
	{"{a,b}{1,2}", "a1", "a2", "b2", 4},
	{"{a,b}x{1,2}", "ax1", "ax2", "bx2", 4},
	{"{abc}", "{abc}", "", "", 1},
	{"{abc}def", "{abc}def", "", "", 1},
	{"{a,b,c}1", "a1", "b1", "c1", 3},
	{"1{a,b,c}", "1a", "1b", "1c", 3},
	{"a{a,b,c}b", "aab", "abb", "acb", 3},
	{"{{1,2,3},1,2,3}", "1", "2", "3", 6},
	{"{{1,2,3}1,2,3}", "11", "21", "3", 5},
	{"{a,{{{b}}}}", "a", "{{{b}}}", "", 2},
	{"{a{1,2}b}", "{a1b}", "{a2b}", "", 2},
	{"{0,1,2}", "0", "1", "2", 3},
	{"{-0,-1,-2}", "-0", "-1", "-2", 3},
	{"{abc{def}}ghi", "{abc{def}}ghi", "", "", 1},
	{"{},a}b", "{},a}b", "", "", 1},
	{"{},abc", "{},abc", "", "", 1},
	{"a{},b}c", "a}c", "abc", "", 2},
	{"-", "-", "", "", 1},
	{"+", "+", "", "", 1},
	{"$", "$", "", "", 1},
	{"{2{,}}", "{2}", "{2}", "", 2},
	{"", "", "", "", 1},
	{"bb{2{,}}bb{{..,}}", "bb{2}bb{..}", "bb{2}bb{}", "bb{2}bb{}", 4},
	{"A{b,{d,e},{f,g}}Z", "AbZ", "AdZ", "AgZ", 5},
	{"PRE-{a,b}{{a,b},a,b}-POST", "PRE-aa-POST", "PRE-ab-POST", "PRE-bb-POST", 8},
	{"\\{a,b}{{a,b},a,b}", "{a,b}a", "{a,b}b", "{a,b}b", 4},
	{"{a,b}", "a", "b", "", 2},
	{"{,}b", "b", "b", "", 2},
	{"a{b}c", "a{b}c", "", "", 1},
	{"a{1..5}b", "a1b", "a2b", "a5b", 5},
	{"a{01..5}b", "a01b", "a02b", "a05b", 5},
	{"a{-01..5}b", "a-01b", "a000b", "a005b", 7},
	{"a{-01..5..3}b", "a-01b", "a002b", "a005b", 3},
	{"a{001..9}b", "a001b", "a002b", "a009b", 9},
	{"a{b,c{d,e},{f,g}h}x{y,z", "abx{y,z", "acdx{y,z", "aghx{y,z", 5},
	{"a{b,c{d,e},{f,g}h}x{y,z\\}", "abx{y,z}", "acdx{y,z}", "aghx{y,z}", 5},
	{"a{b,c{d,e},{f,g}h}x{y,z}", "abxy", "abxz", "aghxz", 10},
	{"a{b{c{d,e}f{x,y{{g}h", "a{b{cdf{x,y{{g}h", "a{b{cef{x,y{{g}h", "", 2},
	{"a{b{c{d,e}f{x,y{}g}h", "a{b{cdfxh", "a{b{cdfy{}gh", "a{b{cefy{}gh", 4},
	{"a{b{c{d,e}f{x,y}}g}h", "a{b{cdfx}g}h", "a{b{cdfy}g}h", "a{b{cefy}g}h", 4},
	{"a{b{c{d,e}f}g}h", "a{b{cdf}g}h", "a{b{cef}g}h", "", 2},
	{"a{{x,y},z}b", "axb", "ayb", "azb", 3},
	{"f{x,y{g,z}}h", "fxh", "fygh", "fyzh", 3},
	{"f{x,y{{g,z}}h", "f{x,y{g}h", "f{x,y{z}h", "", 2},
	{"f{x,y{{g,z}}h}", "fx", "fy{g}h", "fy{z}h", 3},
	{"f{x,y{{g}h", "f{x,y{{g}h", "", "", 1},
	{"f{x,y{{g}}h", "f{x,y{{g}}h", "", "", 1},
	{"f{x,y{}g}h", "fxh", "fy{}gh", "", 2},
	{"z{a,b{,c}d", "z{a,bd", "z{a,bcd", "", 2},
	{"z{a,b},c}d", "za,c}d", "zb,c}d", "", 2},
	{"{-01..5}", "-01", "000", "005", 7},
	{"{-05..100..5}", "-05", "000", "100", 22},
	{"{-05..100}", "-05", "-04", "100", 106},
	{"{0..5..2}", "0", "2", "4", 3},
	{"{0001..05..2}", "0001", "0003", "0005", 3},
	{"{0001..-5..2}", "0001", "-001", "-005", 4},
	{"{0001..-5..-2}", "0001", "-001", "-005", 4},
	{"{0001..5..-2}", "0001", "0003", "0005", 3},
	{"{01..5}", "01", "02", "05", 5},
	{"{1..05}", "01", "02", "05", 5},
	{"{1..05..3}", "01", "04", "", 2},
	{"{05..100}", "005", "006", "100", 96},
	{"{0a..0z}", "{0a..0z}", "", "", 1},
	{"{a,b\\}c,d}", "a", "b}c", "d", 3},
	{"{a,b{c,d}", "{a,bc", "{a,bd", "", 2},
	{"{a,b}c,d}", "ac,d}", "bc,d}", "", 2},
	{"{a..F}", "a", "`", "F", 28},
	{"{A..f}", "A", "B", "f", 38},
	{"{a..Z}", "a", "`", "Z", 8},
	{"{A..z}", "A", "B", "z", 58},
	{"{z..A}", "z", "y", "A", 58},
	{"{Z..a}", "Z", "[", "a", 8},
	{"{a..F..2}", "a", "_", "G", 14},
	{"{A..f..02}", "A", "C", "e", 19},
	{"{a..Z..5}", "a", "", "", 2},
	{"d{a..Z..5}b", "dab", "db", "", 2},
	{"{A..z..10}", "A", "K", "s", 6},
	{"{z..A..-2}", "z", "x", "B", 29},
	{"{Z..a..20}", "Z", "", "", 1},
	{"{a{,b}", "{a", "{ab", "", 2},
	{"{a},b}", "a}", "b", "", 2},
	{"{x,y{,}g}", "x", "yg", "yg", 3},
	{"{x,y{}g}", "x", "y{}g", "", 2},
	{"{a,b}", "a", "b", "", 2},
	{"{{a,b},c}", "a", "b", "c", 3},
	{"{{a,b}c}", "{ac}", "{bc}", "", 2},
	{"{{a,b},}", "a", "b", "", 2},
	{"X{{a,b},}X", "XaX", "XbX", "XX", 3},
	{"{{a,b},}c", "ac", "bc", "c", 3},
	{"{{a,b}.}", "{a.}", "{b.}", "", 2},
	{"{{a,b}}", "{a}", "{b}", "", 2},
	{"X{a..#}X", "X{a..#}X", "", "", 1},
	{"{-10..00}", "-10", "-09", "000", 11},
	{"{a,\\\\{a,b}c}", "a", "\\ac", "\\bc", 3},
	{"{a,\\\\{a,b}c}", "a", "\\ac", "\\bc", 3},
	{"{a,\\{a,b}c}", "ac}", "{ac}", "bc}", 3},
	{"a,\\{b,c}", "a,{b,c}", "", "", 1},
	{"{-10.\\.00}", "{-10..00}", "", "", 1},
	{"ff{c,b,a}", "ffc", "ffb", "ffa", 3},
	{"f{d,e,f}g", "fdg", "feg", "ffg", 3},
	{"{l,n,m}xyz", "lxyz", "nxyz", "mxyz", 3},
	{"{abc\\,def}", "{abc,def}", "", "", 1},
	{"{abc}", "{abc}", "", "", 1},
	{"{x\\,y,\\{abc\\},trie}", "x,y", "{abc}", "trie", 3},
	{"}", "}", "", "", 1},
	{"{", "{", "", "", 1},
	{"abcd{efgh", "abcd{efgh", "", "", 1},
	{"{1..10}", "1", "2", "10", 10},
	{"{0..10,braces}", "0..10", "braces", "", 2},
	{"{{0..10},braces}", "0", "1", "braces", 12},
	{"x{{0..10},braces}y", "x0y", "x1y", "xbracesy", 12},
	{"{3..3}", "3", "", "", 1},
	{"x{3..3}y", "x3y", "", "", 1},
	{"{10..1}", "10", "9", "1", 10},
	{"{10..1}y", "10y", "9y", "1y", 10},
	{"x{10..1}y", "x10y", "x9y", "x1y", 10},
	{"{a..f}", "a", "b", "f", 6},
	{"{f..a}", "f", "e", "a", 6},
	{"{a..A}", "a", "`", "A", 33},
	{"{A..a}", "A", "B", "a", 33},
	{"{f..f}", "f", "", "", 1},
	{"{1..f}", "{1..f}", "", "", 1},
	{"{f..1}", "{f..1}", "", "", 1},
	{"{-1..-10}", "-1", "-2", "-10", 10},
	{"{-20..0}", "-20", "-19", "0", 21},
	{"a-{b{d,e}}-c", "a-{bd}-c", "a-{be}-c", "", 2},
	{"a-{bdef-{g,i}-c", "a-{bdef-g-c", "a-{bdef-i-c", "", 2},
	{"{klklkl}{1,2,3}", "{klklkl}1", "{klklkl}2", "{klklkl}3", 3},
	{"{1..10..2}", "1", "3", "9", 5},
	{"{-1..-10..2}", "-1", "-3", "-9", 5},
	{"{-1..-10..-2}", "-1", "-3", "-9", 5},
	{"{10..1..-2}", "10", "8", "2", 5},
	{"{10..1..2}", "10", "8", "2", 5},
	{"{1..20..2}", "1", "3", "19", 10},
	{"{1..20..20}", "1", "", "", 1},
	{"{100..0..5}", "100", "95", "0", 21},
	{"{100..0..-5}", "100", "95", "0", 21},
	{"{a..z}", "a", "b", "z", 26},
	{"{a..z..2}", "a", "c", "y", 13},
	{"{z..a..-2}", "z", "x", "b", 13},
	{"{2147483645..2147483649}", "2147483645", "2147483646", "2147483649", 5},
	{"{10..0..2}", "10", "8", "0", 6},
	{"{10..0..-2}", "10", "8", "0", 6},
	{"{-50..-0..5}", "-50", "-45", "0", 11},
	{"{1..10.f}", "{1..10.f}", "", "", 1},
	{"{1..ff}", "{1..ff}", "", "", 1},
	{"{1..10..ff}", "{1..10..ff}", "", "", 1},
	{"{1.20..2}", "{1.20..2}", "", "", 1},
	{"{1..20..f2}", "{1..20..f2}", "", "", 1},
	{"{1..20..2f}", "{1..20..2f}", "", "", 1},
	{"{1..2f..2}", "{1..2f..2}", "", "", 1},
	{"{1..ff..2}", "{1..ff..2}", "", "", 1},
	{"{1..ff}", "{1..ff}", "", "", 1},
	{"{1..f}", "{1..f}", "", "", 1},
	{"{1..0f}", "{1..0f}", "", "", 1},
	{"{1..10f}", "{1..10f}", "", "", 1},
	{"{1..10.f}", "{1..10.f}", "", "", 1},

	// EmptyOption
	{"-v{,,,,}", "-v", "-v", "-v", 5},

	// Negative Increment
	{"{3..1}", "3", "2", "1", 3},
	{"{10..8}", "10", "9", "8", 3},
	{"{10..08}", "10", "09", "08", 3},
	{"{-10..-08}", "-10", "-09", "-08", 3},
	{"{c..a}", "c", "b", "a", 3},
	{"{4..0..2}", "4", "2", "0", 3},
	{"{4..0..-2}", "4", "2", "0", 3},
	{"{e..a..2}", "e", "c", "a", 3},

	// Nested
	{"{a,b{1..3},c}", "a", "b1", "c", 5},
	{"{{A..Z},{a..z}}", "A", "B", "z", 52},
	{"ppp{,config,oe{,conf}}", "ppp", "pppconfig", "pppoeconf", 4},

	// Order
	{"a{d,c,b}e", "ade", "ace", "abe", 3},

	// Pad
	{"{9..11}", "9", "10", "11", 3},
	{"{09..11}", "09", "10", "11", 3},

	// Same Type
	{"{a..9}", "{a..9}", "", "", 1},

	// Sequence Numeric
	{"a{1..2}b{2..3}c", "a1b2c", "a1b3c", "a2b3c", 4},
	{"{1..2}{2..3}", "12", "13", "23", 4},

	// Sequence Numeric with step count
	{"{0..8..2}", "0", "2", "8", 5},
	{"{1..8..2}", "1", "3", "7", 4},
	{"{1..10..-3}", "1", "4", "10", 4},
	{"{1..10..3}", "1", "4", "10", 4},
	{"{1..10..4}", "1", "5", "9", 3},
	{"{1..10..04}", "1", "5", "9", 3},
	{"{01..10..3}", "01", "04", "10", 4},
	{"{01..10..03}", "01", "04", "10", 4},

	// Sequence Numeric with negative
	{"{3..-2}", "3", "2", "-2", 6},
	{"{-3..2}", "-3", "-2", "2", 6},

	// Sequence Alphabetic
	{"{a..d}", "a", "b", "d", 4},
	{"{d..a}", "d", "c", "a", 4},
	{"{b..U}", "b", "a", "U", 14},
	{"{U..b}", "U", "V", "b", 14},

	// Sequence Alphabetic with step count
	{"{a..k..2}", "a", "c", "k", 6},
	{"{b..k..2}", "b", "d", "j", 5},
	{"{a..s..4}", "a", "e", "q", 5},
	{"{a..s..-4}", "a", "e", "q", 5},
	{"{s..a..4}", "s", "o", "c", 5},
	{"{s..a..-4}", "s", "o", "c", 5},

	{"{A..K..2}", "A", "C", "K", 6},
	{"{B..K..2}", "B", "D", "J", 5},
	{"{A..S..4}", "A", "E", "Q", 5},
	{"{A..S..-4}", "A", "E", "Q", 5},
	{"{S..A..4}", "S", "O", "C", 5},
	{"{S..A..-4}", "S", "O", "C", 5},

	{"{-e..-a..1}", "{-e..-a..1}", "", "", 1},
	{"{-a..c..1}", "{-a..c..1}", "", "", 1},
	{"{a..-c..1}", "{a..-c..1}", "", "", 1},
	{"{a..-c}", "{a..-c}", "", "", 1},
	{"{-c..a}", "{-c..a}", "", "", 1},

	// Space
	// {"{\"a \",\"b \",\"c \",\"d \"}1", "a 1", "b 1", "d 1", 4},
	// {"", "", "", "", 1},
	// FAIL
	// {"{a,b,c,d} 1", "{a,b,c,d} 1", "", "", 1},
	// {"{a, b, c, d }1", "{a, b, c, d }1", "", "", 1},

	// func TestSpace(t *testing.T) {

	// 	// assert.Equal(t, []string{"a 1", "b 1", "c 1", "d 1"}, Parse(""))
	// 	// assert.Equal(t, []string{"a 1", "b 1", "c 1", "d 1"}, Parse("{a,b,c,d}\" 1\""))

	// 	// s1 := fmt.Sprintf("{%q,%q,%q,%q}1", "a ", "b ", "c ", "d ")
	// 	// assert.Equal(t, []string{"a 1", "b 1", "c 1", "d 1"}, Parse(s1))

	//		// s2 := fmt.Sprintf("{a,b,c,d}%q", " 1")
	//		// assert.Equal(t, []string{"a 1", "b 1", "c 1", "d 1"}, Parse(s2))
	//	}
}

// TestTestDataWithBash ensures that test data is valid
// and expecations match bash output.
func TestTestDataWithBash(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Log("bash not found")
		return
	}
	for _, test := range testResources {
		t.Run(test.Pattern, func(t *testing.T) {
			args := []string{
				"bash",
				"-c",
				fmt.Sprintf("echo %s", test.Pattern),
			}

			out, err := exec.Command(args[0], args[1:]...).Output()
			if err != nil {
				t.Errorf("failed to exec: %s", strings.Join(args, " "))
			}

			clean := strings.TrimRight(string(out), "\n")
			var res []string
			if clean != test.Pattern {
				res = strings.Split(clean, " ")
			} else {
				res = []string{clean}
			}

			if len(test.Res0) > 0 && len(res) == 0 {
				t.Errorf("bash -c returned empty result expecting: %s", test.Res0)
			}
			if len(test.Res1) > 0 && len(res) < 2 {
				t.Error("bash -c less than 2 matches")
			}
			if len(test.Last) > 0 && len(res) < 3 {
				t.Error("bash -c less than 3 matches")
			}

			if len(res) > 0 && test.Res0 != res[0] {
				t.Error(notEqualMsg(test.Res0, res[0]))
			}
			if len(res) > 1 && test.Res1 != res[1] {
				t.Error(notEqualMsg(test.Res1, res[1]))
			}
			if len(res) > 2 && test.Last != res[len(res)-1] {
				t.Error(notEqualMsg(test.Last, res[len(res)-1]))
			}
			if len(res) != test.Len {
				t.Errorf("%s: expected result len(%d) got len(%d) out(%s)", test.Pattern, test.Len, len(res), strings.TrimRight(string(out), "\n"))
			}
		})
	}
}

func TestParse(t *testing.T) {
	for _, test := range testResources {
		t.Run(test.Pattern, func(t *testing.T) {
			res := Parse(test.Pattern)
			if len(res) > 0 && test.Res0 != res[0] {
				t.Error(notEqualMsg(test.Res0, res[0]))
			}
			if len(res) > 1 && test.Res1 != res[1] {
				t.Error(notEqualMsg(test.Res1, res[1]))
			}
			if len(res) > 2 && test.Last != res[len(res)-1] {
				t.Error(notEqualMsg(test.Last, res[len(res)-1]))
			}
			if len(res) != test.Len {
				t.Errorf("%q: expected result len(%d) got len(%d)", test.Pattern, test.Len, len(res))
			}
		})
	}
}

func TestParseValid(t *testing.T) {
	empty, err := ParseValid("")
	if !errors.Is(err, ErrEmptyResult) {
		t.Error("expected error to be ErrEmptyResult")
	}
	if empty[0] != "" {
		t.Error(notEqualMsg("", empty[0]))
	}

	single, err := ParseValid("a")
	if !errors.Is(err, ErrUnchangedBraceExpansion) {
		t.Error("expected error to be ErrUnchangedBraceExpansion")
	}
	if single[0] != "a" {
		t.Error(notEqualMsg("a", single[0]))
	}

	for _, test := range testResources {
		t.Run(test.Pattern, func(t *testing.T) {
			res, _ := ParseValid(test.Pattern)
			if len(res) > 0 && test.Res0 != res[0] {
				t.Error(notEqualMsg(test.Res0, res[0]))
			}
			if len(res) > 1 && test.Res1 != res[1] {
				t.Error(notEqualMsg(test.Res1, res[1]))
			}
			if len(res) > 2 && test.Last != res[len(res)-1] {
				t.Error(notEqualMsg(test.Last, res[len(res)-1]))
			}
			if len(res) != test.Len {
				t.Errorf("%q: expected result len(%d) got len(%d)", test.Pattern, test.Len, len(res))
			}
		})
	}
}

func TestIgnoreDollar(t *testing.T) {
	dollars := []string{
		"${1..3}",
		"${a,b}${c,d}",
		"x${a,b}x${c,d}x",
	}
	for _, dollar := range dollars {
		res := Parse(dollar)
		if len(res) != 1 || res[0] != dollar {
			t.Errorf("want([%s]) got(%v)", dollar, res)
		}
	}
}

func TestString(t *testing.T) {
	r := Parse("/{dir1,dir2}")
	want := "/dir1 /dir2"
	if want != BraceExpansion(r).String() {
		t.Errorf("expected(/dir1 /dir2) got(%s)", BraceExpansion(r).String())
	}
}

func TestResult(t *testing.T) {
	r := Parse("/{dir1,dir2}")
	want := "/dir1"
	res := BraceExpansion(r).Result()
	if want != res[0] {
		t.Errorf("expected(/dir1 /dir2) got(%s)", BraceExpansion(r).String())
	}
}

func TestIsPadded(t *testing.T) {
	tests := []struct {
		in       string
		isPadded bool
	}{
		{"01", true},
		{"00001", true},
		{"-01", true},
		{"-00001", true},
		{"-00", true},
		{"00", true},
		{"00string", true},
		{"-00string", true},
		{"0string", false},
		{"-0string", false},
		{"string", false},
		{"-string", false},
		{"-", false},
		{"1", false},
		{"-1", false},
		{"0", false},
		{"-0", false},
	}
	for _, test := range tests {
		if isPadded(test.in) != test.isPadded {
			t.Errorf("%s: expected to return %t", test.in, test.isPadded)
		}
	}
}

func TestMkdirAllError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip()
		return
	}
	const (
		rootdir = ""
		treeexp = rootdir + "/{dir1,dir2,dir3/{subdir1,subdir2}}"
	)
	if err := MkdirAll(treeexp, 0750); err == nil {
		t.Error(err)
	}
}
