// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2020 The Happy Authors

package bexp_test

import (
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/happy-sdk/happy-go/strings/bexp"
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
		if varName == "MY_ROOT_DIR" {
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

// ExampleExpandOsmTiles the example shows how to create Openstreetmap tiles
// around the desired latitude and longitude coordinates.
//
// https://{a-c}.tile.openstreetmap.org/{z}/{x}/{y}.png
// https://a.tile.openstreetmap.org/1/1/1.png
func ExampleParse_expandOsmTiles() {
	x, y, z := getCenterTile(51.03, 13.78, 5)
	pattern := fmt.Sprintf(
		"https://tile.openstreetmap.org/%d/{%d..%d}/{%d..%d}.png",
		z, x-2, x+2, y-2, y+2,
	)

	tiles := bexp.Parse(pattern)

	fmt.Println("pattern:", pattern)
	for _, tile := range tiles {
		fmt.Println(tile)
	}
	// Output:
	// pattern: https://tile.openstreetmap.org/5/{15..19}/{8..12}.png
	// https://tile.openstreetmap.org/5/15/8.png
	// https://tile.openstreetmap.org/5/15/9.png
	// https://tile.openstreetmap.org/5/15/10.png
	// https://tile.openstreetmap.org/5/15/11.png
	// https://tile.openstreetmap.org/5/15/12.png
	// https://tile.openstreetmap.org/5/16/8.png
	// https://tile.openstreetmap.org/5/16/9.png
	// https://tile.openstreetmap.org/5/16/10.png
	// https://tile.openstreetmap.org/5/16/11.png
	// https://tile.openstreetmap.org/5/16/12.png
	// https://tile.openstreetmap.org/5/17/8.png
	// https://tile.openstreetmap.org/5/17/9.png
	// https://tile.openstreetmap.org/5/17/10.png
	// https://tile.openstreetmap.org/5/17/11.png
	// https://tile.openstreetmap.org/5/17/12.png
	// https://tile.openstreetmap.org/5/18/8.png
	// https://tile.openstreetmap.org/5/18/9.png
	// https://tile.openstreetmap.org/5/18/10.png
	// https://tile.openstreetmap.org/5/18/11.png
	// https://tile.openstreetmap.org/5/18/12.png
	// https://tile.openstreetmap.org/5/19/8.png
	// https://tile.openstreetmap.org/5/19/9.png
	// https://tile.openstreetmap.org/5/19/10.png
	// https://tile.openstreetmap.org/5/19/11.png
	// https://tile.openstreetmap.org/5/19/12.png
}

func getCenterTile(lat, long float64, zoom int) (z, x, y int) {
	n := math.Exp2(float64(zoom))
	x = int(math.Floor((long + 180.0) / 360.0 * n))
	if float64(x) >= n {
		x = int(n - 1)
	}
	y = int(math.Floor((1.0 - math.Log(
		math.Tan(lat*math.Pi/180.0)+
			1.0/math.Cos(lat*math.Pi/180.0))/
		math.Pi) / 2.0 * n))
	return x, y, zoom
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
