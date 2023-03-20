// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package vars_test

import (
	"fmt"
	"os"
	"sort"

	"github.com/happy-sdk/vars"
)

func ExampleValue() {
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

func ExampleMap() {
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
	collection.Store("other4", "1.001")

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

func ExampleMap_envfile() {
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
