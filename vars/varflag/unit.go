// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Uint returns new uint flag. Argument "a" can be any nr of aliases.
func Uint(name string, value uint, usage string, aliases ...string) (*UintFlag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &UintFlag{val: value, Common: *c}
	f.usage = usage
	f.Default(value)
	f.variable, _ = vars.NewTyped(name, "", vars.TypeUint64)
	return f, nil
}

// Parse uint flag.
func (f *UintFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			val, err := vars.NewTyped(f.name, vv[0].String(), vars.TypeUint)
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
	return f.val
}

// Default sets default value for uint flag.
func (f *UintFlag) Default(def ...uint) vars.Variable {
	if len(def) > 0 && f.defval.Empty() {
		f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(def[0]), vars.TypeUint)
		f.val = def[0]
	}
	return f.defval
}

// Unset the int flag value.
func (f *UintFlag) Unset() {
	if !f.defval.Empty() {
		f.variable = f.defval
	} else {
		f.variable, _ = vars.NewTyped(f.name, "0", vars.TypeUint)
	}
	f.isPresent = false
	f.val = f.variable.Uint()
}
