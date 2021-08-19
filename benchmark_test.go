// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp_test

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/mkungla/bexp/v2"
	"github.com/stretchr/testify/assert"
)

type benchTest struct {
	Pattern string
	Res0    string
	Res1    string
	Res2    string
}

type benchTestGroup struct {
	Name  string
	Tests []benchTest
}

//nolint: gochecknoglobals
var benchdata = []benchTestGroup{
	{
		"fs-path",
		[]benchTest{
			{
				"data/{P1/{10..19},P2/{20..29},P3/{30..39}}",
				"data/P1/10", "data/P1/11", "data/P1/12",
			},
			{
				"/usr/{ucb/{ex,edit},lib/{ex?.?*,how_ex}}",
				"/usr/ucb/ex", "/usr/ucb/edit", "/usr/lib/ex?.?*",
			},
			{
				"/usr/local/src/bash/{old,new,dist,bugs}",
				"/usr/local/src/bash/old",
				"/usr/local/src/bash/new",
				"/usr/local/src/bash/dist",
			},
		},
	},
	{
		"string",
		[]benchTest{
			{"{a,b}x{1,2}", "ax1", "ax2", "bx1"},
			{"{a,{{{b}}}}", "a", "{{{b}}}", ""},
			{"{a{1,2}b}", "{a1b}", "{a2b}", ""},
			{"a{b,c,}", "ab", "ac", "a"},
			{"{,,,}", "", "", ""},
			{"{}", "{}", "", ""},
			{"a,,b", "a,,b", "", ""},
			{",a", ",a", "", ""},
		},
	},
	{
		"url",
		[]benchTest{
			{
				"https://tile.openstreetmap.org/1/{10..30}/{10..30}.png",
				"https://tile.openstreetmap.org/1/10/10.png",
				"https://tile.openstreetmap.org/1/10/11.png",
				"https://tile.openstreetmap.org/1/10/12.png",
			},
			{
				"https://example.cdn/{image-series-{a,b}-{1,2}.png",
				"https://example.cdn/{image-series-a-1.png",
				"https://example.cdn/{image-series-a-2.png",
				"https://example.cdn/{image-series-b-1.png",
			},
		},
	},
	{
		"number",
		[]benchTest{
			{
				"{1..3}",
				"1",
				"2",
				"3",
			},
			{
				"{3..1}",
				"3",
				"2",
				"1",
			},
		},
	},
}

// TestBenchData tests bench data so that we dont need to validate results when benchmarking.
func TestBenchData(t *testing.T) {
	checkbash := true
	if _, err := exec.LookPath("bash"); err != nil {
		checkbash = false
	}
	for _, group := range benchdata {
		t.Run(group.Name, func(t *testing.T) {
			t.Parallel()
			for _, test := range group.Tests {
				v, _ := bexp.ParseValid(test.Pattern)
				if len(v) > 0 {
					assert.Equal(t, test.Res0, v[0])
				}
				if len(v) > 1 {
					assert.Equal(t, test.Res1, v[1])
				}
				if len(v) > 2 {
					assert.Equal(t, test.Res2, v[2])
				}
				if checkbash {
					out, err := exec.Command("bash", "-c", fmt.Sprintf("echo %s", test.Pattern)).Output()
					assert.NoError(t, err)
					res := strings.Fields(string(out))
					assert.False(t, len(test.Res0) > 0 && len(res) == 0, "bash -c returned empty result expecting: ", test.Res0)
					assert.False(t, len(test.Res1) > 0 && len(res) < 2, "bash -c less than 2 matches")
					assert.False(t, len(test.Res2) > 0 && len(res) < 3, "bash -c less than 3 matches")
				}
			}
		})
	}
}
func BenchmarkParseValid(b *testing.B) {
	for _, group := range benchdata {
		b.Run(group.Name, func(b *testing.B) {
			for _, test := range group.Tests {
				b.Run(test.Pattern, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						bexp.ParseValid(test.Pattern)
					}
				})
			}
		})
	}
}
