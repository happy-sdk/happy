// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package varflag_test

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/happy-sdk/varflag"
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

	_ = varflag.Parse([]varflag.Flag{str1, str2}, os.Args)

	fmt.Printf(
		"found %q with value %q, (%t) - it was global flag (%t) in position %d\n",
		str1.Name(),
		str1.Value(),
		str1.Present(),
		str1.Global(),
		str1.Pos(),
	)

	fmt.Printf(
		"found %q with value %q, (%t) - it was global flag (%t) in position %d\n",
		str2.Name(),
		str2.Value(),
		str2.Present(),
		str2.Global(),
		str2.Pos(),
	)
	// Output:
	// found "str1" with value "some value", (true) - it was global flag (true) in position 3
	// found "str2" with value "some other value", (true) - it was global flag (true) in position 7
}

func ExampleFlagSet() {
	os.Args = []string{
		"/bin/app", "cmd1", "--flag1", "val1", "--flag2", "flag2-value", "arg1", "--flag3=on",
		"-v",                                                          // global flag can be any place
		"subcmd", "--flag4", "val 4 flag", "arg2", "arg3", "-x", "on", // global flag can be any place
	}

	// Global app flags
	global, _ := varflag.NewFlagSet(os.Args[0], 0)

	v, _ := varflag.Bool("verbose", false, "increase verbosity", "v")
	x, _ := varflag.Bool("x", false, "print commands")
	r, _ := varflag.Bool("random", false, "random flag")
	_ = global.Add(v, x, r)

	flag1, _ := varflag.New("flag1", "", "first flag for first cmd")
	flag2, _ := varflag.New("flag2", "", "another flag for first cmd")
	flag3, _ := varflag.Bool("flag3", false, "bool flag for first command")
	cmd1, _ := varflag.NewFlagSet("cmd1", 1)
	_ = cmd1.Add(flag1, flag2, flag3)

	cmd2, _ := varflag.NewFlagSet("cmd2", 0)
	flag5, _ := varflag.New("flag5", "", "flag5 for second cmd")
	_ = cmd2.Add(flag5)

	subcmd, _ := varflag.NewFlagSet("subcmd", 1)
	flag4, _ := varflag.New("flag4", "", "flag4 for sub command")
	_ = subcmd.Add(flag4)
	_ = cmd1.AddSet(subcmd)

	_ = global.AddSet(cmd1, cmd2)
	_ = global.Parse(os.Args)

	// result
	fmt.Printf("%-12s%t (%t) - %s (%s)\n", "verbose", v.Present(), v.Global(), v.String(), v.BelongsTo())
	fmt.Printf("%-12s%t (%t) - %s (%s)\n", "x", x.Present(), x.Global(), x.String(), x.BelongsTo())
	fmt.Printf("%-12s%t (%t) - %s (%s)\n", "random", r.Present(), r.Global(), r.String(), r.BelongsTo())
	fmt.Printf("%-12s %v\n", "gloabal args", global.Args())

	fmt.Printf("\n%-12s%t\n", "cmd1", cmd1.Present())
	fmt.Printf("%-12s%t (%t) - %s\n", "flag1", flag1.Present(), flag1.Global(), flag1.String())
	fmt.Printf("%-12s%t (%t) - %s\n", "flag2", flag2.Present(), flag2.Global(), flag2.String())
	fmt.Printf("%-12s%t (%t) - %s\n", "flag3", flag3.Present(), flag3.Global(), flag3.String())
	fmt.Printf("%-12s %v\n", "cmd1 args", cmd1.Args())

	fmt.Printf("\n%-12s%t\n", "subcmd", subcmd.Present())
	fmt.Printf("%-12s%t (%t) - %s\n", "flag4", flag4.Present(), flag4.Global(), flag4.String())
	fmt.Printf("%-12s %v\n", "subcmd args", subcmd.Args())

	fmt.Printf("\n%-12s%t\n", "cmd2", cmd2.Present())

	// flag3 will not be present since it belongs to cmd 2
	fmt.Printf("%-12s%t (%t)\n", "flag5", flag5.Present(), flag5.Global())

	// Output:
	// verbose     true (true) - true (/)
	// x           true (true) - true (/)
	// random      false (false) - false (/)
	// gloabal args []
	//
	// cmd1        true
	// flag1       true (false) - val1
	// flag2       true (false) - flag2-value
	// flag3       true (false) - true
	// cmd1 args    [arg1]
	//
	// subcmd      true
	// flag4       true (false) - val 4 flag
	// subcmd args  [arg2 arg3]
	//
	// cmd2        false
	// flag5       false (false)
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
	fmt.Printf("%-12s%s\n", "aliases-str", str.UsageAliases())
	fmt.Printf("%-12s%#v\n", "aliases", str.Aliases())
	// You can mark flag as hidden by calling .Hide()
	// Helpful when you are composing help menu.
	fmt.Printf("%-12s%t\n", "is:hidden", str.Hidden())
	// by default flag is global regardless which position it was found.
	// You can mark flag non global by calling .BelongsTo(cmdname string).
	fmt.Printf("%-12s%t\n", "is:global", str.Global())
	fmt.Printf("%-12s%q\n", "command", str.BelongsTo())
	fmt.Printf("%-12s%d\n", "position", str.Pos())
	fmt.Printf("%-12s%t\n", "present", str.Present())
	// Var is underlying vars.Variable for convinient type conversion
	fmt.Printf("%-12s%s\n", "var", str.Var())
	// you can set flag as required by calling Required before you parse flags.
	fmt.Printf("%-12s%t\n", "required", str.Required())
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

// func ExampleDuration() {
// 	os.Args = []string{"/bin/app", "--duration", "1h30s"}
// 	dur, _ := varflag.Duration("duration", 1*time.Second, "")
// 	_, _ = dur.Parse(os.Args)

// 	fmt.Printf("%-12s%d\n", "duration", dur.Value())
// 	fmt.Printf("%-12s%s\n", "duration", dur.Value())
// 	fmt.Printf("%-12s%s\n", "string", dur.String())
// 	fmt.Printf("%-12s%d\n", "int", dur.Var().Int())
// 	fmt.Printf("%-12s%d\n", "int64", dur.Var().Int64())
// 	fmt.Printf("%-12s%d\n", "uint", dur.Var().Uint())
// 	fmt.Printf("%-12s%d\n", "uint64", dur.Var().Uint64())
// 	fmt.Printf("%-12s%f\n", "float32", dur.Var().Float32())
// 	fmt.Printf("%-12s%f\n", "float64", dur.Var().Float64())
// 	// Output:
// 	// duration    3630000000000
// 	// duration    1h0m30s
// 	// string      1h0m30s
// 	// int         3630000000000
// 	// int64       3630000000000
// 	// uint        3630000000000
// 	// uint64      3630000000000
// 	// float32     3629999980544.000000
// 	// float64     3630000000000.000000
// }

func ExampleFloat64() {
	os.Args = []string{"/bin/app", "--float", "1.001000023"}
	f, _ := varflag.Float64("float", 1.0, "")
	_, _ = f.Parse(os.Args)

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
	_, _ = f.Parse(os.Args)

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
	_, _ = f.Parse(os.Args)

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
	_, _ = f.Parse(os.Args)

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
	f, _ := varflag.Option("option", []string{"defaultOpt"}, []string{"opt1", "opt2", "opt3", "defaultOpt"}, "")
	_, _ = f.Parse(os.Args)

	fmt.Printf("%-12s%s\n", "string", f.String())
	fmt.Printf("%-12s%#v\n", "value", f.Value())

	// Output:
	// string      opt1|opt2
	// value       []string{"opt1", "opt2"}
}

func ExampleBexp() {
	os.Args = []string{"/bin/app", "--images", "image-{0..2}.jpg"}
	f, _ := varflag.Bexp("images", "image-{a,b,c}.jpg", "expand path", "")
	_, _ = f.Parse(os.Args)

	fmt.Printf("%-12s%s\n", "string", f.String())
	fmt.Printf("%-12s%#v\n", "value", f.Value())

	// Output:
	// string      image-0.jpg|image-1.jpg|image-2.jpg
	// value       []string{"image-0.jpg", "image-1.jpg", "image-2.jpg"}
}
