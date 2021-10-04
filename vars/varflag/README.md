# varflag

Package flag implements command-line flag parsing into vars.Variables for easy type handling with additional flag types.

[![PkgGoDev](https://pkg.go.dev/badge/github.com/mkungla/varflag/v5)](https://pkg.go.dev/github.com/mkungla/varflag/v5)
![license](https://img.shields.io/github/license/mkungla/varflag)
![GitHub last commit](https://img.shields.io/github/last-commit/mkungla/varflag)
![tests](https://github.com/mkungla/varflag/workflows/test/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/mkungla/varflag)](https://goreportcard.com/report/github.com/mkungla/varflag)
[![Coverage Status](https://coveralls.io/repos/github/mkungla/varflag/badge.svg?branch=main)](https://coveralls.io/github/mkungla/varflag?branch=main)
[![benchmarks](https://img.shields.io/badge/benchmark-result-green)](https://dashboard.github.orijtech.com/graphs?repo=https%3A%2F%2Fgithub.com%2Fmkungla%varflag.git)

- [varflag](#varflag)
- [Usage](#usage)
  - [String flag](#string-flag)
  - [Duration flag](#duration-flag)
  - [Float flag](#float-flag)
  - [Int flag](#int-flag)
  - [Uint Flag](#uint-flag)
  - [Bool Flag](#bool-flag)
  - [Option Flag](#option-flag)

# Usage

> note that major version ensures compatibility with
> https://github.com/mkungla/vars package

`go get github.com/mkungla/varflag/v5`

## String flag

```go
package main

import (
  "fmt"
  "log"
  "os"
  "github.com/mkungla/varflag/v5"
)

func main() {
  os.Args = []string{"/bin/app", "--str", "some value"}
  // create flag named string with default value "default str"
  // and aditional aliases for this flag
  str, err := varflag.New("string", "default str", "some string", "s", "str")
  if err != nil {
    log.Println(err)
    return
  }
  found, err := str.Parse(os.Args)
  if err != nil {
    log.Println(err)
    return
  }
  fmt.Printf("%-12s%s\n", "name", str.Name())
  fmt.Printf("%-12s%s\n", "flag", str.Flag())
  fmt.Printf("%-12s%t\n", "found", found)
  fmt.Printf("%-12s%s\n", "value", str.Value())
  // all flags satisfy Stringer interface
  fmt.Printf("%-12s%s\n", "string", str.String())
  fmt.Printf("%-12s%s\n", "default", str.Default())
  fmt.Printf("%-12s%s\n", "usage", str.Usage())
  fmt.Printf("%-12s%s\n", "aliases-str", str.AliasesString())
  fmt.Printf("%-12s%#v\n", "aliases", str.Aliases())
  // You can mark flag as hidden by calling .Hide()
  // Helpful when you are composing help menu.
  fmt.Printf("%-12s%t\n", "is:hidden", str.IsHidden())
  // by default flag is global regardless which position it was found.
  // You can mark flag non global by calling .BelongsTo(cmdname string).
  fmt.Printf("%-12s%t\n", "is:global", str.IsGlobal())
  fmt.Printf("%-12s%q\n", "command", str.CommandName())
  fmt.Printf("%-12s%d\n", "position", str.Pos())
  fmt.Printf("%-12s%t\n", "present", str.Present())
  // Var is underlying vars.Variable for convinient type conversion
  fmt.Printf("%-12s%s\n", "var", str.Var())
  // you can set flag as required by calling Required before you parse flags.
  fmt.Printf("%-12s%t\n", "required", str.IsRequired())
  // Output:
  // name        string
  // flag        --string
  // found       true
  // value       some value
  // string      some value
  // default     default str
  // usage       some string - default: "default str"
  // aliases-str -s,--str
  // aliases     []string{"s", "str"}
  // is:hidden   false
  // is:global   true
  // command     ""
  // position    2
  // present     true
  // var         some value
  // required    false
}
```

## Duration flag

```go
os.Args = []string{"/bin/app", "--duration", "1h30s"}
dur, _ := varflag.Duration("duration", 1*time.Second, "")
dur.Parse(os.Args)

fmt.Printf("%-12s%d\n", "duration", dur.Value())
fmt.Printf("%-12s%s\n", "duration", dur.Value())
fmt.Printf("%-12s%s\n", "string", dur.String())
fmt.Printf("%-12s%d\n", "int", dur.Var().Int())
fmt.Printf("%-12s%d\n", "int64", dur.Var().Int64())
fmt.Printf("%-12s%d\n", "uint", dur.Var().Uint())
fmt.Printf("%-12s%d\n", "uint64", dur.Var().Uint64())
fmt.Printf("%-12s%f\n", "float64", dur.Var().Float64())
// Output:
// duration    3630000000000
// duration    1h0m30s
// string      1h0m30s
// int         3630000000000
// int64       3630000000000
// uint        3630000000000
// uint64      3630000000000
// float64     3630000000000.000000
```

## Float flag

```go
os.Args = []string{"/bin/app", "--float", "1.001000023"}
f, _ := varflag.Float64("float", 1.0, "")
f.Parse(os.Args)

fmt.Printf("%-12s%.10f\n", "float", f.Value())
fmt.Printf("%-12s%s\n", "string", f.String())
fmt.Printf("%-12s%.10f\n", "float32", f.Var().Float32())
fmt.Printf("%-12s%.10f\n", "float64", f.Var().Float64())
// Output:
// float       1.0010000230
// string      1.001000023
// float32     1.0010000467
// float64     1.0010000230
```

## Int flag

```go
os.Args = []string{"/bin/app", "--int", fmt.Sprint(math.MinInt64), "int64"}
f, _ := varflag.Int("int", 1, "")
f.Parse(os.Args)

fmt.Printf("%-12s%s\n", "string", f.String())
fmt.Printf("%-12s%d\n", "value", f.Value())
fmt.Printf("%-12s%d\n", "int", f.Var().Int())
fmt.Printf("%-12s%d\n", "int64", f.Var().Int64())
fmt.Printf("%-12s%d\n", "uint", f.Var().Uint())
fmt.Printf("%-12s%d\n", "uint64", f.Var().Uint64())
fmt.Printf("%-12s%f\n", "float32", f.Var().Float32())
fmt.Printf("%-12s%f\n", "float64", f.Var().Float64())
// Output:
// string      -9223372036854775808
// value       -9223372036854775808
// int         -9223372036854775808
// int64       -9223372036854775808
// uint        0
// uint64      0
// float32     -9223372036854775808.000000
// float64     -9223372036854775808.000000
```

## Uint Flag

```go
os.Args = []string{"/bin/app", "--uint", strconv.FormatUint(math.MaxUint64, 10), "uint64"}
f, _ := varflag.Uint("uint", 1, "")
f.Parse(os.Args)

fmt.Printf("%-12s%s\n", "string", f.String())
fmt.Printf("%-12s%d\n", "value", f.Value())
fmt.Printf("%-12s%d\n", "int", f.Var().Int())
fmt.Printf("%-12s%d\n", "int64", f.Var().Int64())
fmt.Printf("%-12s%d\n", "uint", f.Var().Uint())
fmt.Printf("%-12s%d\n", "uint64", f.Var().Uint64())
fmt.Printf("%-12s%f\n", "float32", f.Var().Float32())
fmt.Printf("%-12s%f\n", "float64", f.Var().Float64())
// Output:
// string      18446744073709551615
// value       18446744073709551615
// int         9223372036854775807
// int64       9223372036854775807
// uint        18446744073709551615
// uint64      18446744073709551615
// float32     18446744073709551616.000000
// float64     18446744073709551616.000000
}
```

## Bool Flag

Valid bool flag args are

**true**

`--bool, --bool=true, --bool=1 --bool=on, --bool true, --bool 1, --bool on`

**false**

`--bool, --bool=false, --bool=0 --bool=off, --bool false, --bool 0, --bool off`

```go
os.Args = []string{"/bin/app", "--bool"}
f, _ := varflag.Bool("bool", false, "")
f.Parse(os.Args)

fmt.Printf("%-12s%s\n", "string", f.String())
fmt.Printf("%-12s%t\n", "value", f.Value())
fmt.Printf("%-12s%t\n", "bool", f.Var().Bool())
// Output:
// string      true
// value       true
// bool        true
```

## Option Flag

```go
os.Args = []string{"/bin/app", "--option", "opt1", "--option", "opt2"}
f, _ := varflag.Option("option", []string{"defaultOpt"}, "", []string{"opt1", "opt2", "opt3", "defaultOpt"})
f.Parse(os.Args)

fmt.Printf("%-12s%s\n", "string", f.String())
fmt.Printf("%-12s%#v\n", "value", f.Value())

// Output:
// string      opt1|opt2
// value       []string{"opt1", "opt2"}
```

**test**

```
golangci-lint run --config=./.github/golangci.yaml --fix
go test -race -covermode atomic  .
```
