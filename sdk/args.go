// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package sdk

import (
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

type Args struct {
	argv  []vars.Value
	argn  uint
	flags varflag.Flags
}

func NewArgs(argv []vars.Value, flags varflag.Flags) *Args {
	return &Args{
		argv:  argv,
		argn:  uint(len(argv)),
		flags: flags,
	}
}

func (a *Args) Arg(i uint) vars.Value {
	if a.argn <= i {
		return vars.EmptyValue
	}
	return a.argv[i]
}

func (a *Args) ArgDefault(i uint, value any) (vars.Value, error) {
	if a.argn <= i {
		return vars.NewValue(value)
	}
	return a.Arg(i), nil
}

func (a *Args) ArgVarDefault(i uint, key string, value any) (vars.Variable, error) {
	if a.argn <= i {
		return vars.New(key, value, true)
	}
	return vars.New(key, a.argv[i], true)
}

func (a *Args) Args() []vars.Value {
	return a.argv
}
func (a *Args) Argn() uint {
	return a.argn
}

func (a *Args) Flag(name string) varflag.Flag {
	f, err := a.flags.Get(name)
	if err != nil {
		ff, _ := varflag.Bool("unknown", false, "")
		return ff
	}
	return f
}
