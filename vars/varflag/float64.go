// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package varflag

import (
	"fmt"
	"strings"

	"github.com/happy-sdk/vars"
)

// Float64Flag defines a float64 flag with specified name.
type Float64Flag struct {
	Common
	val float64
}

// Float64 returns new float flag. Argument "a" can be any nr of aliases.
func Float64(name string, value float64, usage string, aliases ...string) (flag *Float64Flag, err error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}

	flag = &Float64Flag{}
	flag.name = strings.TrimLeft(name, "-")
	flag.val = value
	flag.aliases = normalizeAliases(aliases)
	flag.usage = usage
	flag.defval, err = vars.NewAs(name, value, true, vars.KindFloat64)
	if err != nil {
		return nil, err
	}
	flag.variable, err = vars.NewAs(name, value, false, vars.KindFloat64)
	return flag, err
}

func Float64Func(name string, value float64, usage string, aliases ...string) FlagCreateFunc {
	return func() (Flag, error) {
		return Float64(name, value, usage, aliases...)
	}
}

// Parse float flag.
func (f *Float64Flag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			val, err := vars.ParseVariableAs(f.name, vv[0].String(), false, vars.KindFloat64)
			if err != nil {
				return fmt.Errorf("%w: %q", ErrInvalidValue, err)
			}
			f.variable = val
			f.val = f.variable.Float64()
		}
		return err
	})
}

// Value return float64 flag value, it returns default value if not present
// or 0 if default is also not set.
func (f *Float64Flag) Value() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

// Unset the bool flag value.
func (f *Float64Flag) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Float64()
}
