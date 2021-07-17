// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp_test

import (
	"fmt"

	"github.com/mkungla/bexp/v2"
)

func ExampleParse() {
	var v []string
	v = bexp.Parse("file-{a,b,c}.jpg")
	fmt.Println(v)

	v = bexp.Parse("-v{,,}")
	fmt.Println(v)

	v = bexp.Parse("file{0..2}.jpg")
	fmt.Println(v)

	v = bexp.Parse("file{2..0}.jpg")
	fmt.Println(v)

	v = bexp.Parse("file{0..4..2}.jpg")
	fmt.Println(v)

	v = bexp.Parse("file-{a..e..2}.jpg")
	fmt.Println(v)

	v = bexp.Parse("file{00..10..5}.jpg")
	fmt.Println(v)

	v = bexp.Parse("{{A..C},{a..c}}")
	fmt.Println(v)

	v = bexp.Parse("ppp{,config,oe{,conf}}")
	fmt.Println(v)

	v = bexp.Parse("data/{P1/{10..19},P2/{20..29},P3/{30..39}}")
	fmt.Println(v)

	// Output:
	// [file-a.jpg file-b.jpg file-c.jpg]
	// [-v -v -v]
	// [file0.jpg file1.jpg file2.jpg]
	// [file2.jpg file1.jpg file0.jpg]
	// [file0.jpg file2.jpg file4.jpg]
	// [file-a.jpg file-c.jpg file-e.jpg]
	// [file00.jpg file05.jpg file10.jpg]
	// [A B C a b c]
	// [ppp pppconfig pppoe pppoeconf]
	// [data/P1/10 data/P1/11 data/P1/12 data/P1/13 data/P1/14 data/P1/15 data/P1/16 data/P1/17 data/P1/18 data/P1/19 data/P2/20 data/P2/21 data/P2/22 data/P2/23 data/P2/24 data/P2/25 data/P2/26 data/P2/27 data/P2/28 data/P2/29 data/P3/30 data/P3/31 data/P3/32 data/P3/33 data/P3/34 data/P3/35 data/P3/36 data/P3/37 data/P3/38 data/P3/39]

}
