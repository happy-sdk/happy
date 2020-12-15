# VARS

![license](https://img.shields.io/github/license/mkungla/vars) [![PkgGoDev](https://pkg.go.dev/badge/github.com/mkungla/vars/v3)](https://pkg.go.dev/github.com/mkungla/vars/v3) ![tests](https://github.com/mkungla/vars/workflows/tests/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/mkungla/vars)](https://goreportcard.com/report/github.com/mkungla/vars) [![Coverage Status](https://coveralls.io/repos/github/mkungla/vars/badge.svg?branch=main)](https://coveralls.io/github/mkungla/vars?branch=main)![GitHub last commit](https://img.shields.io/github/last-commit/mkungla/vars)

## About
Package vars provides the API to parse variables from various input formats/types to common key value pair vars.Value or variable sets to vars.Collection




## Install

```
go get github.com/mkungla/vars/v3
```

## Usage

**working with [vars.Value](https://pkg.go.dev/github.com/mkungla/vars/v3#Value)**
```go
package main

import (
  "fmt"
  "github.com/mkungla/vars/v3"
)

func main() {
  empty1, _ := vars.ParseValue(nil)
  empty2 := vars.NewValue("")
  if empty1.String() == empty2.String() {
    // both produce empty var
  }
  v, _ := vars.ParseValue(123456)
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

**working with [vars.Collection](https://pkg.go.dev/github.com/mkungla/vars/v3#Value)**
```go
package main

import (
  "fmt"
  "github.com/mkungla/vars/v3"
)

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
  fmt.Println(collection.Get("other4").Float64())

  // Output:
  // _key3 true
  // 1.001
}
```

**read collection from file**
```go
package main

import (
  "fmt"
  "io/ioutil"
  "github.com/mkungla/vars/v3"
)

func main() {
  content, err := ioutil.ReadFile("testdata/dot_env")
  if err != nil {
    fmt.Println(err)
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
