// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package varflag

import (
	"fmt"
	"strings"

	"github.com/happy-sdk/happy/pkg/vars"
)

// IntFlag defines an int flag with specified name,.
type IntFlag struct {
	Common
	val int
}

// Int returns new int flag. Argument "a" can be any nr of aliases.
func Int(name string, value int, usage string, aliases ...string) (flag *IntFlag, err error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}
	flag = &IntFlag{}
	flag.usage = usage
	flag.name = strings.TrimLeft(name, "-")
	flag.val = value
	flag.aliases = normalizeAliases(aliases)
	flag.defval, err = vars.NewAs(name, value, true, vars.KindInt)
	if err != nil {
		return nil, err
	}
	flag.variable, err = vars.NewAs(name, value, false, vars.KindInt)
	return flag, err
}

func IntFunc(name string, value int, usage string, aliases ...string) FlagCreateFunc {
	return func() (Flag, error) {
		return Int(name, value, usage, aliases...)
	}
}

// Parse int flag.
func (f *IntFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			val, err := vars.ParseVariableAs(f.name, vv[0].String(), false, vars.KindInt)
			if err != nil {
				return fmt.Errorf("%w: %q", ErrInvalidValue, err)
			}
			f.variable = val
			f.val = f.variable.Int()
		}
		return err
	})
}

// Value returns int flag value, it returns default value if not present
// or 0 if default is also not set.
func (f *IntFlag) Value() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

// Unset the int flag value.
func (f *IntFlag) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Int()
}
