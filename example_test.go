// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

package bexp_test

import (
	"errors"
	"fmt"
	"os"

	"github.com/mkungla/bexp/v2"
)

func ExampleParse() {
	var v []string

	v = bexp.Parse("/path/unmodified")
	fmt.Println(v)

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
	// [/path/unmodified]
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

func ExampleParse_osExpand() {
	const treeExp = "$MY_ROOT_DIR/dir{1..3}/{subdir1,subdir2}"
	mapper := func(varName string) string {
		switch varName {
		case "MY_ROOT_DIR":
			return "/my_root"
		}
		return ""
	}
	str := os.Expand(treeExp, mapper)
	fmt.Println("str := os.Expand(treeExp, mapper)")
	fmt.Println(str)

	fmt.Println("v := bexp.Parse(str)")
	v := bexp.Parse(str)
	for _, p := range v {
		fmt.Println(p)
	}

	// Output:
	// str := os.Expand(treeExp, mapper)
	// /my_root/dir{1..3}/{subdir1,subdir2}
	// v := bexp.Parse(str)
	// /my_root/dir1/subdir1
	// /my_root/dir1/subdir2
	// /my_root/dir2/subdir1
	// /my_root/dir2/subdir2
	// /my_root/dir3/subdir1
	// /my_root/dir3/subdir2
}

func ExampleParse_osExpandEnv() {
	const treeExp = "$MY_ROOT_DIR/dir{1..3}/{subdir1,subdir2}"
	os.Setenv("MY_ROOT_DIR", "/my_root")

	str := os.ExpandEnv(treeExp)
	fmt.Println("str := os.ExpandEnv(treeExp)")
	fmt.Println(str)

	fmt.Println("v := bexp.Parse(str)")
	v := bexp.Parse(str)
	for _, p := range v {
		fmt.Println(p)
	}

	// Output:
	// str := os.ExpandEnv(treeExp)
	// /my_root/dir{1..3}/{subdir1,subdir2}
	// v := bexp.Parse(str)
	// /my_root/dir1/subdir1
	// /my_root/dir1/subdir2
	// /my_root/dir2/subdir1
	// /my_root/dir2/subdir2
	// /my_root/dir3/subdir1
	// /my_root/dir3/subdir2
}

func ExampleParseValid() {
	empty, err := bexp.ParseValid("")
	fmt.Printf("%q - %t\n", empty[0], errors.Is(err, bexp.ErrEmptyResult))

	abc, err := bexp.ParseValid("abc")
	fmt.Printf("%q - %t\n", abc[0], errors.Is(err, bexp.ErrUnchangedBraceExpansion))

	// Output:
	// "" - true
	// "abc" - true
}
