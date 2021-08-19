// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp_test

import (
	"testing"

	"github.com/mkungla/bexp/v2"
)

type benchResource struct {
	Name    string
	Pattern string
	Res0    string
	Res1    string
	Res2    string
}

func BenchmarkParse(b *testing.B) {
	var benchdata = []benchResource{
		{"fs-path", "data/{P1/{10..19},P2/{20..29},P3/{30..39}}", "data/P1/10", "data/P1/11", "data/P1/12"},
		{"fs-path", "/usr/{ucb/{ex,edit},lib/{ex?.?*,how_ex}}", "/usr/ucb/ex", "/usr/ucb/edit", "/usr/lib/ex?.?*"},
		{
			"fs-path",
			"/usr/local/src/bash/{old,new,dist,bugs}",
			"/usr/local/src/bash/old",
			"/usr/local/src/bash/new",
			"/usr/local/src/bash/dist",
		},
		{"string", "{a,b}x{1,2}", "ax1", "ax2", "bx1"},
		{"string", "{a,{{{b}}}}", "a", "{{{b}}}", ""},
		{"string", "{a{1,2}b}", "{a1b}", "{a2b}", ""},
		{"string", "a{b,c,}", "ab", "ac", "a"},
		{"string", "{,,,}", "", "", ""},
		{"string", "{}", "{}", "", ""},
		{"string", "a,,b", "a,,b", "", ""},
		{"string", ",a", ",a", "", ""},
	}

	for _, test := range benchdata {

		name := test.Name + "("
		if len(test.Pattern) > 10 {
			name += test.Pattern[0:10] + ")"
		} else {
			name += test.Pattern + ")"
		}
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				v := bexp.Parse(test.Pattern)
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
