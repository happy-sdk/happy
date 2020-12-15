# Bash Brace Expansion in go

Implementing: [3.5.1 Brace Expansion][bash-be]

![license](https://img.shields.io/github/license/mkungla/bexp)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/mkungla/bexp/v1)](https://pkg.go.dev/github.com/mkungla/bexp/v1)
![tests](https://github.com/mkungla/bexp/workflows/tests/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/mkungla/bexp)](https://goreportcard.com/report/github.com/mkungla/bexp)
[![Coverage Status](https://coveralls.io/repos/github/mkungla/bexp/badge.svg?branch=main)](https://coveralls.io/github/mkungla/bexp?branch=main)
![benchmarks](https://github.com/mkungla/bexp/workflows/benchmarks/badge.svg) 
![GitHub last commit](https://img.shields.io/github/last-commit/mkungla/bexp)
## Usage

```go
package bexp_test

import (
	"bexp/v1"
	"fmt"
)

func ExampleExpand() {
	var v []string
	v = bexp.Expand("file-{a,b,c}.jpg")
	fmt.Println(v)
	// [file-a.jpg file-b.jpg file-c.jpg]

	v = bexp.Expand("-v{,,}")
	fmt.Println(v)
	// [-v -v -v]

	v = bexp.Expand("file{0..2}.jpg")
	fmt.Println(v)
	// [file0.jpg file1.jpg file2.jpg]

	v = bexp.Expand("file{2..0}.jpg")
	fmt.Println(v)
	// [file2.jpg file1.jpg file0.jpg]

	v = bexp.Expand("file{0..4..2}.jpg")
	fmt.Println(v)
	// [file0.jpg file2.jpg file4.jpg]

	v = bexp.Expand("file-{a..e..2}.jpg")
	fmt.Println(v)
	// [file-a.jpg file-c.jpg file-e.jpg]

	v = bexp.Expand("file{00..10..5}.jpg")
	fmt.Println(v)
	// [file00.jpg file05.jpg file10.jpg]

	v = bexp.Expand("{{A..C},{a..c}}")
	fmt.Println(v)
	// [A B C a b c]

	v = bexp.Expand("ppp{,config,oe{,conf}}")
	fmt.Println(v)
	// [ppp pppconfig pppoe pppoeconf]

	v = bexp.Expand("data/{P1/{10..19},P2/{20..29},P3/{30..39}}")
	fmt.Println(v)
	// [data/P1/10 data/P1/11 data/P1/12 data/P1/13 data/P1/14 data/P1/15 data/P1/16 data/P1/17 data/P1/18 data/P1/19 data/P2/20 data/P2/21 data/P2/22 data/P2/23 data/P2/24 data/P2/25 data/P2/26 data/P2/27 data/P2/28 data/P2/29 data/P3/30 data/P3/31 data/P3/32 data/P3/33 data/P3/34 data/P3/35 data/P3/36 data/P3/37 data/P3/38 data/P3/39]
}
```

## This package is inspired by
- [go-brace-expansion]
- [thomasheller/braceexpansion]
- [kujtimiihoxha/go-brace-expansion]
- [juliangruber/brace-expansion]

<!-- LINKS -->
[bash-be]: https://www.gnu.org/software/bash/manual/html_node/Brace-Expansion.html
[kujtimiihoxha/go-brace-expansion]: https://github.com/kujtimiihoxha/go-brace-expansion
[braceexpansion]: https://github.com/thomasheller/braceexpansion
[juliangruber/brace-expansion]: https://github.com/kujtimiihoxha/go-balanced-match
