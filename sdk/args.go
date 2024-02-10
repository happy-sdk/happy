// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package sdk

import (
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

type Args struct {
	args  []vars.Value
	argn  uint
	flags varflag.Flags
}

func NewArgs(flags varflag.Flags) *Args {
	args := flags.Args()
	return &Args{
		args:  args,
		argn:  uint(len(args)),
		flags: flags,
	}
}

func (a *Args) Arg(i uint) vars.Value {
	if a.argn <= i {
		return vars.EmptyValue
	}
	return a.args[i]
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
	return vars.New(key, a.args[i], true)
}

func (a *Args) Args() []vars.Value {
	return a.args
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
