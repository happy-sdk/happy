// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag_test

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/mkungla/varflag/v5"
)

func ExampleParse() {
	os.Args = []string{
		"/bin/app",
		"-v",
		"--str1",
		"some value",
		"arg1",
		"arg2",
		"--str2",
		"some other value",
	}

	str1, err := varflag.New("str1", ".", "some string")
	if err != nil {
		log.Println(err)
		return
	}

	str2, err := varflag.New("str2", "", "some other string")
	if err != nil {
		log.Println(err)
		return
	}

	varflag.Parse([]varflag.Flag{str1, str2}, os.Args)

	fmt.Printf(
		"found %q with value %q, (%t) - it was global flag (%t) in position %d\n",
		str1.Name(),
		str1.Value(),
		str1.Present(),
		str1.IsGlobal(),
		str1.Pos(),
	)

	fmt.Printf(
		"found %q with value %q, (%t) - it was global flag (%t) in position %d\n",
		str2.Name(),
		str2.Value(),
		str2.Present(),
		str2.IsGlobal(),
		str2.Pos(),
	)
	// Output:
	// found "str1" with value "some value", (true) - it was global flag (true) in position 3
	// found "str2" with value "some other value", (true) - it was global flag (true) in position 7
}

func ExampleNew() {
	os.Args = []string{
		"/bin/app",
		"--str",
		"some value",
	}
	// create flag named string with default value "default str"
	// and aditional aliases for this flag
	str, err := varflag.New("string", "default str", "some string", "s", "str")
	if err != nil {
		log.Println(err)
		return
	}
	// if you have single flag then parse it directly.
	// no need for pkg .Parse
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

func ExampleDuration() {
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
	fmt.Printf("%-12s%f\n", "float32", dur.Var().Float32())
	fmt.Printf("%-12s%f\n", "float64", dur.Var().Float64())
	// Output:
	// duration    3630000000000
	// duration    1h0m30s
	// string      1h0m30s
	// int         3630000000000
	// int64       3630000000000
	// uint        3630000000000
	// uint64      3630000000000
	// float32     3629999980544.000000
	// float64     3630000000000.000000
}

func ExampleFloat64() {
	os.Args = []string{"/bin/app", "--float", "1.001000023"}
	f, _ := varflag.Float64("float", 1.0, "")
	f.Parse(os.Args)

	fmt.Printf("%-12s%.10f\n", "float", f.Value())
	fmt.Printf("%-12s%s\n", "string", f.String())
	fmt.Printf("%-12s%.10f\n", "float32", f.Var().Float32())
	fmt.Printf("%-12s%.10f\n", "float64", f.Var().Float64())
	// Output:
	// float       1.0010000230
	// string      1.001000023
	// float32     1.0010000467
	// float64     1.0010000230
}

func ExampleInt() {
	os.Args = []string{"/bin/app", "--int", fmt.Sprint(math.MinInt64), "int64"}
	f, _ := varflag.Int("int", 1, "")
	f.Parse(os.Args)

	fmt.Printf("%-12s%s\n", "string", f.String())
	fmt.Printf("%-12s%d\n", "value", f.Value())
	fmt.Printf("%-12s%d\n", "int", f.Var().Int())
	fmt.Printf("%-12s%d\n", "int64", f.Var().Int64())
	fmt.Printf("%-12s%d\n", "uint", f.Var().Uint())
	fmt.Printf("%-12s%d\n", "uint64", f.Var().Uint64())
	fmt.Printf("%-12s%f\n", "float32", f.Var().Float32())
	fmt.Printf("%-12s%f\n", "float64", f.Var().Float64())
	// Output:
	// string      -9223372036854775808
	// value       -9223372036854775808
	// int         -9223372036854775808
	// int64       -9223372036854775808
	// uint        0
	// uint64      0
	// float32     -9223372036854775808.000000
	// float64     -9223372036854775808.000000
}

func ExampleUint() {
	os.Args = []string{"/bin/app", "--uint", strconv.FormatUint(math.MaxUint64, 10), "uint64"}
	f, _ := varflag.Uint("uint", 1, "")
	f.Parse(os.Args)

	fmt.Printf("%-12s%s\n", "string", f.String())
	fmt.Printf("%-12s%d\n", "value", f.Value())
	fmt.Printf("%-12s%d\n", "int", f.Var().Int())
	fmt.Printf("%-12s%d\n", "int64", f.Var().Int64())
	fmt.Printf("%-12s%d\n", "uint", f.Var().Uint())
	fmt.Printf("%-12s%d\n", "uint64", f.Var().Uint64())
	fmt.Printf("%-12s%f\n", "float32", f.Var().Float32())
	fmt.Printf("%-12s%f\n", "float64", f.Var().Float64())
	// Output:
	// string      18446744073709551615
	// value       18446744073709551615
	// int         9223372036854775807
	// int64       9223372036854775807
	// uint        18446744073709551615
	// uint64      18446744073709551615
	// float32     18446744073709551616.000000
	// float64     18446744073709551616.000000
}

func ExampleBool() {
	os.Args = []string{"/bin/app", "--bool"}
	f, _ := varflag.Bool("bool", false, "")
	f.Parse(os.Args)

	fmt.Printf("%-12s%s\n", "string", f.String())
	fmt.Printf("%-12s%t\n", "value", f.Value())
	fmt.Printf("%-12s%t\n", "bool", f.Var().Bool())
	// Output:
	// string      true
	// value       true
	// bool        true
}

func ExampleOption() {
	os.Args = []string{"/bin/app", "--option", "opt1", "--option", "opt2"}
	f, _ := varflag.Option("option", []string{"defaultOpt"}, "", []string{"opt1", "opt2", "opt3", "defaultOpt"})
	f.Parse(os.Args)

	fmt.Printf("%-12s%s\n", "string", f.String())
	fmt.Printf("%-12s%#v\n", "value", f.Value())

	// Output:
	// string      opt1|opt2
	// value       []string{"opt1", "opt2"}
}
