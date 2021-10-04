# varflag

Package flag implements command-line flag parsing into vars.Variables for easy type handling with additional flag types.

[![PkgGoDev](https://pkg.go.dev/badge/github.com/mkungla/varflag/v5)](https://pkg.go.dev/github.com/mkungla/varflag/v5)
![license](https://img.shields.io/github/license/mkungla/varflag)
![GitHub last commit](https://img.shields.io/github/last-commit/mkungla/varflag)
![tests](https://github.com/mkungla/varflag/workflows/test/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/mkungla/varflag)](https://goreportcard.com/report/github.com/mkungla/varflag)
[![Coverage Status](https://coveralls.io/repos/github/mkungla/varflag/badge.svg?branch=main)](https://coveralls.io/github/mkungla/varflag?branch=main)


## Usage

> note that major version ensures compatibility with
> https://github.com/mkungla/vars package

`go get github.com/mkungla/varflag/v5`



### Simple string flag

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

### Duration flag

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

**test**

```
golangci-lint run --config=./.github/golangci.yaml --fix
go test -race -covermode atomic  .
```
