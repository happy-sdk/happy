// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package cli

import (
	"time"

	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

type Flag = varflag.FlagCreateFunc

func NewStringFlag(name string, value string, usage string, aliases ...string) Flag {
	return varflag.StringFunc(name, value, usage, aliases...)
}

func NewBoolFlag(name string, value bool, usage string, aliases ...string) Flag {
	return varflag.BoolFunc(name, value, usage, aliases...)
}

func NewUintFlag(name string, value uint, usage string, aliases ...string) Flag {
	return varflag.UintFunc(name, value, usage, aliases...)
}

func NewIntFlag(name string, value int, usage string, aliases ...string) Flag {
	return varflag.IntFunc(name, value, usage, aliases...)
}

func NewFloat64Flag(name string, value float64, usage string, aliases ...string) Flag {
	return varflag.Float64Func(name, value, usage, aliases...)
}

func NewDurationFlag(name string, value time.Duration, usage string, aliases ...string) Flag {
	return varflag.DurationFunc(name, value, usage, aliases...)
}

func NewOptionFlag(name string, value []string, opts []string, usage string, aliases ...string) Flag {
	return varflag.OptionFunc(name, value, opts, usage, aliases...)
}
