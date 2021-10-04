// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag_test

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mkungla/varflag/v5"
)

func ExampleNew() {
	os.Args = []string{
		"/bin/app",
		"-v",
		"--str",
		"some value",
		"arg1",
		"arg2",
		"--str2",
		"some other value",
	}

	str, err := varflag.New("str", ".", "some string")
	if err != nil {
		log.Println(err)
		return
	}
	strWasProvided, err := str.Parse(os.Args)
	if err != nil {
		log.Println(err)
		return
	}

	str2, err := varflag.New("str2", "", "some other string")
	if err != nil {
		log.Println(err)
		return
	}
	str2WasProvided, err := str2.Parse(os.Args)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Printf(
		"found %q with value %q, (%t) - it was global flag (%t) in position %d\n",
		str.Name(),
		str.Value(),
		strWasProvided,
		str.IsGlobal(),
		str.Pos(),
	)

	fmt.Printf(
		"found %q with value %q, (%t) - it was global flag (%t) in position %d\n",
		str2.Name(),
		str2.Value(),
		str2WasProvided,
		str2.IsGlobal(),
		str2.Pos(),
	)
	// Output:
	// found "str" with value "some value", (true) - it was global flag (true) in position 3
	// found "str2" with value "some other value", (true) - it was global flag (true) in position 7
}

func ExampleFlag() {
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
	fmt.Printf("%-12s%f\n", "float64", dur.Var().Float64())
	// Output:
	// duration    3630000000000
	// duration    1h0m30s
	// string      1h0m30s
	// int         3630000000000
	// int64       3630000000000
	// uint        3630000000000
	// uint64      3630000000000
	// float64     3630000000000.000000
}
