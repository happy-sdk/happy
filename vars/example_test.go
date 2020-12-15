// Copyright 2020 Marko Kungla.
// Source code is provider under MIT License.

package vars_test

import (
	"fmt"

	"github.com/mkungla/vars/v3"
)

func ExampleValue() {
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

func ExampleCollection() {
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
