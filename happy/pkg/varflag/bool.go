// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package varflag

import (
	"fmt"
	"strings"

	"github.com/mkungla/happy/pkg/vars"
)

// BoolFlag is boolean flag type with default value "false".
type BoolFlag struct {
	Common
	val bool
}

// Bool returns new bool flag. Argument "a" can be any nr of aliases.
func Bool(name string, value bool, usage string, aliases ...string) (flag *BoolFlag, err error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}

	flag = &BoolFlag{}
	flag.name = strings.TrimLeft(name, "-")
	flag.val = value
	flag.aliases = normalizeAliases(aliases)
	flag.usage = usage
	flag.defval, err = vars.NewAs(name, value, true, vars.KindBool)
	if err != nil {
		return nil, err
	}
	flag.variable, err = vars.NewAs(name, value, true, vars.KindBool)
	return flag, err
}

func BoolFunc(name string, value bool, usage string, aliases ...string) FlagCreateFunc {
	return func() (Flag, error) {
		return Bool(name, value, usage, aliases...)
	}
}

// Parse bool flag.
func (f *BoolFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			f.variable = vv[0]
			f.val = f.variable.Bool()
		}
		return err
	})
}

// Value returns bool flag value, it returns default value if not present
// or false if default is also not set.
func (f *BoolFlag) Value() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

// Unset the bool flag value.
func (f *BoolFlag) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Bool()
}
