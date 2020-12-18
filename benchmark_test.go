// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp

import "testing"

type benchResource struct {
	Pattern string
	Res0    string
	Res1    string
	Res2    string
}

var benchdata = []benchResource{
	{"data/{P1/{10..19},P2/{20..29},P3/{30..39}}", "data/P1/10", "data/P1/11", "data/P1/12"},
	{"{a{1,2}b}", "{a1b}", "{a2b}", ""},
	{"/usr/{ucb/{ex,edit},lib/{ex?.?*,how_ex}}", "/usr/ucb/ex", "/usr/ucb/edit", "/usr/lib/ex?.?*"},
	{"{a,b}x{1,2}", "ax1", "ax2", "bx1"},
	{"{a,{{{b}}}}", "a", "{{{b}}}", ""},
	{"/usr/local/src/bash/{old,new,dist,bugs}", "/usr/local/src/bash/old", "/usr/local/src/bash/new", "/usr/local/src/bash/dist"},
	{"a{b,c,}", "ab", "ac", "a"},
	{"{,,,}", "", "", ""},
	{"{}", "{}", "", ""},
	{"a,,b", "a,,b", "", ""},
	{",a", ",a", "", ""},
	{braceexpansion, braceexpansion, "", ""},
}

func BenchmarkParse(b *testing.B) {
	for _, test := range benchdata {
		b.Run(test.Pattern, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				v := Parse(test.Pattern)
				l := len(v)
				if l > 0 && v[0] != test.Res0 {
					b.Errorf("Unexpected result: expected: %q  got: %q from: %v", test.Res0, v[0], v)
				}
				if l > 1 && v[1] != test.Res1 {
					b.Error("Unexpected result: " + v[1])
				}
				if l > 2 && v[2] != test.Res2 {
					b.Error("Unexpected result: " + v[2])
				}
			}
		})
	}
}
