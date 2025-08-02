// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package action

import (
	"time"

	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/session"
)

type Action func(sess *session.Context) error

type Register func(sess session.Register) error

// type ActionWithFlags func(sess *Session, flags Flags) error
type WithArgs func(sess *session.Context, args Args) error
type Tick func(sess *session.Context, ts time.Time, delta time.Duration) error
type Tock func(sess *session.Context, delta time.Duration, tps int) error
type WithPrevErr func(sess *session.Context, err error) error
type WithOptions func(sess *session.Context, opts *options.Options) error

type Args interface {
	Arg(i uint) vars.Value
	ArgDefault(i uint, value any) (vars.Value, error)
	Args() []vars.Value
	Argn() uint
	Flag(name string) varflag.Flag
}

type args struct {
	args  []vars.Value
	argn  uint
	flags varflag.Flags
}

func NewArgs(flags varflag.Flags) Args {
	fargs := flags.Args()
	return &args{
		args:  fargs,
		argn:  uint(len(fargs)),
		flags: flags,
	}
}

func (a *args) Arg(i uint) vars.Value {
	if a.argn <= i {
		return vars.EmptyValue
	}
	return a.args[i]
}

func (a *args) ArgDefault(i uint, value any) (vars.Value, error) {
	if a.argn <= i {
		return vars.NewValue(value)
	}
	return a.Arg(i), nil
}

func (a *args) ArgVarDefault(i uint, key string, value any) (vars.Variable, error) {
	if a.argn <= i {
		return vars.New(key, value, true)
	}
	return vars.New(key, a.args[i], true)
}

func (a *args) Args() []vars.Value {
	return a.args
}
func (a *args) Argn() uint {
	return a.argn
}

func (a *args) Flag(name string) varflag.Flag {
	f, err := a.flags.Get(name)
	if err != nil {
		ff, _ := varflag.Bool("unknown", false, "")
		return ff
	}
	return f
}

var ActionNoop func(*session.Context) error = func(*session.Context) error { return nil }
