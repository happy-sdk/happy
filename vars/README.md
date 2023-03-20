# VARS

![license](https://img.shields.io/github/license/happy-sdk/vars)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/vars)](https://pkg.go.dev/github.com/happy-sdk/vars)
![tests](https://github.com/happy-sdk/vars/workflows/tests/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/happy-sdk/vars)](https://goreportcard.com/report/github.com/happy-sdk/vars)
[![Coverage Status](https://coveralls.io/repos/github/happy-sdk/vars/badge.svg?branch=main)](https://coveralls.io/github/happy-sdk/vars?branch=main)
<!-- [![benchmarks](https://github.com/mkungla/vars/workflows/benchmarks/badge.svg)](https://dashboard.github.orijtech.com/graphs?repo=https%3A%2F%2Fgithub.com%2Fmkungla%2Fvars.git) -->
![GitHub last commit](https://img.shields.io/github/last-commit/happy-sdk/vars)

## About
Package vars provides the API to parse variables from various input formats/types to common key value pair vars.Value or variable sets to vars.Collection


## Install

```
go get github.com/happy-sdk/vars
```

## Usage

**working with [vars.Value](https://pkg.go.dev/github.com/happy-sdk/vars#Value)**

```go
package main

import (
  "fmt"
  "github.com/happy-sdk/vars"
)

func main() {
  vnil := vars.NewValue(nil)
  fmt.Printf("%t\n", vnil.Type() == vars.TypeUnknown)
  fmt.Println(vnil.String())

  v := vars.NewValue(123456)
  fmt.Printf("%t\n", v.Type() == vars.TypeInt)
  fmt.Println(v.String())

  fmt.Println(v.Int())
  fmt.Println(v.Empty())
  fmt.Println(v.Int64())
  fmt.Println(v.Float32())
  fmt.Println(v.Float64())
  fmt.Println(v.Len())
  fmt.Println(v.Runes())
  fmt.Println(v.Uint64())
  fmt.Println(v.Uintptr())

  // Output:
  // true
  // <nil>
  // true
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

**working with [vars.Collection](https://pkg.go.dev/github.com/happy-sdk/vars#Collection)**

> Because of underlying `sync.Map` it is meant to be populated once and read many times
> read thoroughly sync.Map docs to understand where .Collection may not me right for you!

```go
package main

import (
  "fmt"
  "github.com/happy-sdk/vars"
)

func main() {
  collection := vars.ParseKeyValSlice([]string{
    "key1=val1",
    "key2=2",
    "_key31=true",
    "_key32=true",
    "_key33=true",
    "_key34=true",
  })
  collection.Set("other4", "1.001")

  set := collection.GetWithPrefix("_key3")

  var keys []string

  set.Range(func(key string, val vars.Value) bool {
    keys = append(keys, key)
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
  "github.com/happy-sdk/vars"
)

func main() {
  content, err := ioutil.ReadFile("testdata/dot_env")
  if err != nil {
    fmt.Println(err)
    return
  }

  collection := vars.ParseFromBytes(content)
  goarch := collection.Get("GOARCH")
  fmt.Printf("GOARCH = %s\n", goarch)

  // Output:
  // GOARCH = amd64
}
```

## License

Package vars is licensed under the [MIT LICENSE](./LICENSE).
