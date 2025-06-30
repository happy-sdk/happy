// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package cli

import (
	"time"

	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

type Flag = varflag.FlagCreateFunc

// Common CLI flags which are automatically attached to the CLI ubnless disabled ins settings.
// You still can manually add them to your CLI if you want to.
var (
	FlagVersion     = varflag.BoolFunc("version", false, "print application version")
	FlagHelp        = varflag.BoolFunc("help", false, "display help or help for the command. [...command --help]", "h")
	FlagX           = varflag.BoolFunc("x", false, "the -x flag prints all the cli commands as they are executed.")
	FlagSystemDebug = varflag.BoolFunc("system-debug", false, "enable system debug log level (very verbose)")
	FlagDebug       = varflag.BoolFunc("debug", false, "enable debug log level")
	FlagVerbose     = varflag.BoolFunc("verbose", false, "enable verbose log level", "v")
)

// FlagProd is a flag that forces the application into production mode.
var (
	FlagXProd = varflag.BoolFunc("x-prod", false, "DEV ONLY: force app into production mode setting app_is_devel false when running from source.")
)

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
