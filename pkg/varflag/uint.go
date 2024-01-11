// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package varflag

import (
	"fmt"
	"strings"

	"github.com/happy-sdk/happy-go/vars"
)

// UintFlag defines a uint flag with specified name.
type UintFlag struct {
	Common
	val uint
}

// Uint returns new uint flag. Argument "a" can be any nr of aliases.
func Uint(name string, value uint, usage string, aliases ...string) (flag *UintFlag, err error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}
	flag = &UintFlag{}
	flag.usage = usage
	flag.name = strings.TrimLeft(name, "-")
	flag.val = value
	flag.aliases = normalizeAliases(aliases)
	flag.val = value
	flag.defval, err = vars.NewAs(flag.name, value, true, vars.KindUint)
	if err != nil {
		return nil, err
	}
	flag.variable, err = vars.NewAs(flag.name, value, false, vars.KindUint)
	return flag, err
}

func UintFunc(name string, value uint, usage string, aliases ...string) FlagCreateFunc {
	return func() (Flag, error) {
		return Uint(name, value, usage, aliases...)
	}
}

// Parse uint flag.
func (f *UintFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			val, err := vars.ParseVariableAs(f.name, vv[0].String(), true, vars.KindUint)
			if err != nil {
				return fmt.Errorf("%w: %q", ErrInvalidValue, err)
			}
			f.variable = val
			f.val = f.variable.Uint()
		}
		return err
	})
}

// Value returns uint flag value, it returns default value if not present
// or 0 if default is also not set.
func (f *UintFlag) Value() uint {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

// Unset the int flag value.
func (f *UintFlag) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Uint()
}
