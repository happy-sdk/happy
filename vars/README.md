# VARS

![license](https://img.shields.io/github/license/howi-lib/vars) [![PkgGoDev](https://pkg.go.dev/badge/github.com/howi-lib/vars/v2)](https://pkg.go.dev/github.com/howi-lib/vars/v2) ![tests](https://github.com/howi-lib/vars/workflows/tests/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/howi-lib/vars)](https://goreportcard.com/report/github.com/howi-lib/vars) [![Coverage Status](https://coveralls.io/repos/github/howi-lib/vars/badge.svg)](https://coveralls.io/github/howi-lib/vars)![GitHub last commit](https://img.shields.io/github/last-commit/howi-lib/vars)

## About
Package vars provides the API to parse variables from various input formats/types to common key value pair vars.Value or variable sets to vars.Collection




## Install

```
go get github.com/howi-lib/vars/v2
```

## Usage

**working with [vars.Value](https://pkg.go.dev/github.com/howi-lib/vars/v2#Value)**
```go
package main

import "github.com/howi-lib/vars/v2"

func main() {
  empty1 := vars.NewValue(nil)
  empty2 := vars.NewValue("")
  if empty1.String() == empty2.String() {
  	// both produce empty var
  }
  v := vars.NewValue(123456)
  fmt.Println(v.String())

  if out, err := v.AsInt(); err == nil {
  	fmt.Println(out)
  }
  if out := v.Empty(); !out {
  	fmt.Println(out)
  }
  if out, err := v.Int(10, 64); err == nil {
  	fmt.Println(out)
  }
  if out, err := v.Float(32); err == nil {
  	fmt.Println(out)
  }
  if out, err := v.Float(64); err == nil {
  	fmt.Println(out)
  }
  if out := v.Len(); out > 0 {
  	fmt.Println(out)
  }
  if out := v.Rune(); out != nil {
  	fmt.Println(out)
  }
  if out, err := v.Uint(10, 64); err == nil {
  	fmt.Println(out)
  }
  if out, err := v.Uintptr(); err == nil {
  	fmt.Println(out)
  }
  // Output:
  // 123456
  // 123456
  // false
  // 123456
  // 123456
  // 123456
  // 6
  // [49 50 51 52 53 54]
  // 123456
  // 123456
}
```

**working with [vars.Collection](https://pkg.go.dev/github.com/howi-lib/vars/v2#Value)**
```go
package main

import "github.com/howi-lib/vars/v2"

func main() {
  collection := vars.ParseKeyValSlice([]string{
    "key1=val1",
    "key2=2",
    "_key3=true",
  })
  collection.Set("other4", "1.001")

  set := collection.GetWithPrefix("_key3")
  for key, val := range set {
    fmt.Println(key, val)
  }
  fmt.Println(collection.Get("other4").Float(64))
  // Output:
  // _key3 true
  // 1.001 <nil>
}
```

**read collection from file**
```go
package main

import "github.com/howi-lib/vars/v2"

func main() {
  content, err := ioutil.ReadFile("testdata/dot_env")
  if err != nil {
    t.Error(err)
    return
  }
  collection := vars.ParseFromBytes(content)
  if val := collection.Get("GOARCH"); val != "amd64" {
    fmt.Println(fmt.Sprintf("expected GOARCH to equal amd64 got %s", val))
  }
}
```

## License

Package vars is licensed under the [MIT LICENSE](./LICENSE).
