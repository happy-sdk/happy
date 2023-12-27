# Bash Brace Expansion in Go

last version

Go implementation of Brace Expansion mechanism to generate arbitrary strings.

Implementing: [3.5.1 Brace Expansion][bash-be]

[![PkgGoDev](https://pkg.go.dev/badge/github.com/mkungla/bexp/v3)](https://pkg.go.dev/github.com/mkungla/bexp/v3)
![license](https://img.shields.io/github/license/mkungla/bexp)
![GitHub last commit](https://img.shields.io/github/last-commit/mkungla/bexp)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

![tests](https://github.com/mkungla/bexp/workflows/tests/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/mkungla/bexp/v3)](https://goreportcard.com/report/github.com/mkungla/bexp/v3)
[![Coverage Status](https://coveralls.io/repos/github/mkungla/bexp/badge.svg?branch=main)](https://coveralls.io/github/mkungla/bexp?branch=main)

[![benchmarks](https://github.com/mkungla/bexp/workflows/benchmarks/badge.svg)](https://dashboard.bencher.orijtech.com/graphs?tab=1&branch=&repo=https%3A%2F%2Fgithub.com%2Fmkungla%2Fbexp.git&start=1670554071&end=1671158871&searchTerm=&yScale=linear&highlight=)
<!-- ![GitHub all releases](https://img.shields.io/github/downloads/mkungla/bexp/total) -->



- [Bash Brace Expansion in Go](#bash-brace-expansion-in-go)
  - [Usage](#usage)
  - [Get string slice](#get-string-slice)
  - [Generating directory tree](#generating-directory-tree)
  - [Expand URLS](#expand-urls)
  - [Need error checking?](#need-error-checking)
  - [With **os.Expand**](#with-osexpand)
  - [With **os.ExpandEnv**](#with-osexpandenv)
  - [Inspired by and other similar libraries](#inspired-by-and-other-similar-libraries)


## Usage

`go get github.com/mkungla/bexp/v3`

## Get string slice

```go
package main

import (
  "fmt"

  "github.com/mkungla/bexp/v3"
)

func main() {
  var v []string
  v = bexp.Parse("file-{a,b,c}.jpg")
  fmt.Println(v)
  // [file-a.jpg file-b.jpg file-c.jpg]

  v = bexp.Parse("-v{,,}")
  fmt.Println(v)
  // [-v -v -v]

  v = bexp.Parse("file{0..2}.jpg")
  fmt.Println(v)
  // [file0.jpg file1.jpg file2.jpg]

  v = bexp.Parse("file{2..0}.jpg")
  fmt.Println(v)
  // [file2.jpg file1.jpg file0.jpg]

  v = bexp.Parse("file{0..4..2}.jpg")
  fmt.Println(v)
  // [file0.jpg file2.jpg file4.jpg]

  v = bexp.Parse("file-{a..e..2}.jpg")
  fmt.Println(v)
  // [file-a.jpg file-c.jpg file-e.jpg]

  v = bexp.Parse("file{00..10..5}.jpg")
  fmt.Println(v)
  // [file00.jpg file05.jpg file10.jpg]

  v = bexp.Parse("{{A..C},{a..c}}")
  fmt.Println(v)
  // [A B C a b c]

  v = bexp.Parse("ppp{,config,oe{,conf}}")
  fmt.Println(v)
  // [ppp pppconfig pppoe pppoeconf]

  v = bexp.Parse("data/{P1/{10..19},P2/{20..29},P3/{30..39}}")
  fmt.Println(v)
  // [data/P1/10 data/P1/11 data/P1/12 data/P1/13 data/P1/14 data/P1/15 data/P1/16 data/P1/17 data/P1/18 data/P1/19 data/P2/20 data/P2/21 data/P2/22 data/P2/23 data/P2/24 data/P2/25 data/P2/26 data/P2/27 data/P2/28 data/P2/29 data/P3/30 data/P3/31 data/P3/32 data/P3/33 data/P3/34 data/P3/35 data/P3/36 data/P3/37 data/P3/38 data/P3/39]
}
```

## Generating directory tree

```go
package main

import (
  "log"

  "github.com/mkungla/bexp/v3"
)

func main() {
  const (
    rootdir = "/tmp/bexp"
    treeexp = rootdir + "/{dir1,dir2,dir3/{subdir1,subdir2}}"
  )
  if err := bexp.MkdirAll(treeexp, 0750); err != nil {
    log.Fatal(err)
  }

  // Will produce directory tree
  // /tmp/bexp
  // /tmp/bexp/dir1
  // /tmp/bexp/dir2
  // /tmp/bexp/dir3
  // /tmp/bexp/dir3/subdir1
  // /tmp/bexp/dir3/subdir2
}
``` 

## Expand URLS

The example shows how to create Openstreetmap tiles  
around the desired latitude and longitude coordinates.

```go
package main

import (
  "fmt"
  "log"
  "math"

  "github.com/mkungla/bexp/v3"
)

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

func main() {
  x, y, z := getCenterTile(51.03, 13.78, 5)
  pattern := fmt.Sprintf(
    "https://tile.openstreetmap.org/%d/{%d..%d}/{%d..%d}.png",
    z, x-2, x+2, y-2, y+2,
  )
  tiles := bexp.Parse(pattern)
  fmt.Println("pattern: ", pattern)
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
```

## Need error checking?

```go
package main

import (
  "errors"
  "fmt"

  "github.com/mkungla/bexp/v3"
)

func main() {
  empty, err := bexp.ParseValid("")
  fmt.Printf("%q - %t\n", empty[0], errors.Is(err, bexp.ErrEmptyResult))

  abc, err := bexp.ParseValid("abc")
  fmt.Printf("%q - %t\n", abc[0], errors.Is(err, bexp.ErrUnchangedBraceExpansion))

  // Output:
  // "" - true
  // "abc" - true
}
```

## With **os.Expand**

[os.Expand](https://pkg.go.dev/os#Expand)

```go
package main

import (
  "fmt"
  "os"

  "github.com/mkungla/bexp/v3"
)

func main() {
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
```

## With **os.ExpandEnv**

[os.ExpandEnv](https://pkg.go.dev/os#ExpandEnv)

```go
package main

import (
  "fmt"
  "os"

  "github.com/mkungla/bexp/v3"
)

func main() {
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
```

## Inspired by and other similar libraries

> following package were inspiration to create this package, The motivation 
> of this package is to improve performance and reduce memory allocations 
> compared to other solutions. Also to add some commonly used API's 
> when working with brace expansion strings

- @kujtimiihoxha [go-brace-expansion] Go bash style brace expansion
- @thomasheller [braceexpansion] Shell brace expansion implemented in Go (golang).
- @pittfit [ortho] Go brace expansion library
- @kujtimiihoxha [go-balanced-match] Go balanced match

<!-- LINKS -->
[bash-be]: https://www.gnu.org/software/bash/manual/html_node/Brace-Expansion.html
[go-brace-expansion]: https://github.com/kujtimiihoxha/go-brace-expansion
[braceexpansion]: https://github.com/thomasheller/braceexpansion
[ortho]: https://github.com/pittfit/ortho
[go-balanced-match]: https://github.com/kujtimiihoxha/go-balanced-match
