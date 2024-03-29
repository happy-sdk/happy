// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package varflag

import (
	"fmt"
	"strings"
	"time"

	"github.com/happy-sdk/happy/pkg/vars"
)

type DurationFlag struct {
	Common
	val time.Duration
}

func Duration(name string, value time.Duration, usage string, aliases ...string) (flag *DurationFlag, err error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}
	flag = &DurationFlag{}
	flag.usage = usage
	flag.name = strings.TrimLeft(name, "-")
	flag.val = value
	flag.aliases = normalizeAliases(aliases)
	flag.defval, err = vars.NewAs(name, value, true, vars.KindDuration)
	if err != nil {
		return nil, err
	}
	flag.variable, err = vars.NewAs(name, value, false, vars.KindDuration)
	return flag, err
}

func DurationFunc(name string, value time.Duration, usage string, aliases ...string) FlagCreateFunc {
	return func() (Flag, error) {
		return Duration(name, value, usage, aliases...)
	}
}

func (f *DurationFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			val, err := vars.ParseVariableAs(f.name, vv[0].String(), false, vars.KindDuration)
			if err != nil {
				return fmt.Errorf("%w: %q", ErrInvalidValue, err)
			}
			f.variable = val
			f.val = f.variable.Duration()
		}
		return err
	})
}
