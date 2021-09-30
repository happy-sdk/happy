// Copyright 2020 Marko Kungla.
// Source code is provider under MIT License.

package vars_test

import (
	"fmt"
	"io/ioutil"
	"sort"

	"github.com/mkungla/vars/v5"
)

func ExampleValue() {
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

func ExampleCollection() {
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

func ExampleCollection_envfile() {
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
