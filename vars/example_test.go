// Copyright 2020 Marko Kungla.
// Source code is provider under MIT License.

package vars_test

import (
	"fmt"

	"github.com/howi-lib/vars/v2"
)

func ExampleValue() {
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
	fmt.Println(collection.Get("other4").Float(64))

	// Output:
	// _key3 true
	// 1.001 <nil>
}
