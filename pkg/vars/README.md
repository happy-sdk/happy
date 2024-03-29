# VARS

![license](https://img.shields.io/github/license/happy-sdk/vars)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy/pkg/vars)](https://pkg.go.dev/github.com/happy-sdk/happy/pkg/vars)
![tests](https://github.com/happy-sdk/happy/pkg/vars/workflows/tests/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/happy-sdk/happy/pkg/vars)](https://goreportcard.com/report/github.com/happy-sdk/happy/pkg/vars)
[![Coverage Status](https://coveralls.io/repos/github/happy-sdk/vars/badge.svg?branch=main)](https://coveralls.io/github/happy-sdk/vars?branch=main)
<!-- [![benchmarks](https://github.com/mkungla/vars/workflows/benchmarks/badge.svg)](https://dashboard.github.orijtech.com/graphs?repo=https%3A%2F%2Fgithub.com%2Fmkungla%2Fvars.git) -->
![GitHub last commit](https://img.shields.io/github/last-commit/happy-sdk/happy/vars)

## About
Package vars provides the API to parse variables from various input formats/types to common key value pair vars.Value or variable sets to vars.Collection


## Install

```
go get github.com/happy-sdk/happy/pkg/vars
```

## Usage

**working with [vars.Value](https://pkg.go.dev/github.com/happy-sdk/happy/pkg/vars#Value)**

```go
package main

import (
  "fmt"
  "github.com/happy-sdk/happy/pkg/vars"
)

func main() {
  vnil, _ := vars.NewValue(nil)
  fmt.Printf("%t\n", vnil.Kind() == vars.KindInvalid)
  fmt.Println(vnil.String())
  fmt.Println("")
  
  v, _ := vars.New("eg", 123456, false)
  fmt.Printf("%T %t\n", v.Kind(), v.Kind() == vars.KindInt)
  fmt.Println(v.String())
  fmt.Println(v.Any())
  fmt.Println(v.Empty())
  fmt.Println(v.Len())
  
  fmt.Println(v.Bool())
  fmt.Println(v.Int())
  fmt.Println(v.Int8())
  fmt.Println(v.Int16())
  fmt.Println(v.Int32())
  fmt.Println(v.Int64())
  fmt.Println(v.Uint())
  fmt.Println(v.Uint8())
  fmt.Println(v.Uint16())
  fmt.Println(v.Uint32())
  fmt.Println(v.Uint64())
  fmt.Println(v.Float32())
  fmt.Println(v.Float64())
  fmt.Println(v.Complex64())
  fmt.Println(v.Complex128())
  fmt.Println(v.Uintptr())
  fmt.Println(v.Fields())
  // Output:
  // true
  // <nil>
  //
  // vars.Kind true
  // 123456
  // 123456
  // false
  // 6
  // false
  // 123456
  // 127
  // 32767
  // 123456
  // 123456
  // 123456
  // 255
  // 65535
  // 123456
  // 123456
  // 123456
  // 123456
  // (123456+0i)
  // (123456+0i)
  // 123456
  // [123456]
}
```

**working with [vars.Collection](https://pkg.go.dev/github.com/happy-sdk/happy/pkg/vars#Collection)**

> Because of underlying `sync.Map` it is meant to be populated once and read many times
> read thoroughly sync.Map docs to understand where .Collection may not me right for you!

```go
package main

import (
  "fmt"
  "github.com/happy-sdk/happy/pkg/vars"
)

func main() {
  collection, err := vars.ParseMapFromSlice([]string{
    "key1=val1",
    "key2=2",
    "_key31=true",
    "_key32=true",
    "_key33=true",
    "_key34=true",
  })
  if err != nil {
    panic("did not expect error: " + err.Error())
  }
  if err := collection.Store("other4", "1.001"); err != nil {
    panic("did not expect error: " + err.Error())
  }

  set, _ := collection.LoadWithPrefix("_key3")

  var keys []string

  set.Range(func(v vars.Variable) bool {
    keys = append(keys, v.Name())
    return true
  })
  sort.Strings(keys)
  for _, k := range keys {
    fmt.Println(k)
  }
  fmt.Println(collection.Get("other4").Float64())

  // Output:
  // _key31
  // _key32
  // _key33
  // _key34
  // 1.001
}
```

**read collection from file**

```go
package main

import (
  "fmt"
  "io/ioutil"
  "github.com/happy-sdk/happy/pkg/vars"
)

func main() {
  content, err := os.ReadFile("testdata/dot_env")
  if err != nil {
    fmt.Println(err)
    return
  }

  collection, err := vars.ParseMapFromBytes(content)
  if err != nil {
    panic("did not expect error: " + err.Error())
  }
  goarch := collection.Get("GOARCH")
  fmt.Printf("GOARCH = %s\n", goarch)

  // Output:
  // GOARCH = amd64
}
```

