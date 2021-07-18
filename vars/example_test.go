// Copyright 2020 Marko Kungla.
// Source code is provider under MIT License.

package vars_test

import (
	"fmt"

	"github.com/mkungla/vars/v4"
)

func ExampleValue() {
	vnil, _ := vars.NewValue(nil)
	fmt.Println(vnil.String())

	v, _ := vars.NewValue(123456)
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
	// <nil>
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
