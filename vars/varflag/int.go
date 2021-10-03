// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Int returns new int flag. Argument "a" can be any nr of aliases.
func Int(name string, value int, usage string, aliases ...string) (*IntFlag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &IntFlag{val: value, Common: *c}
	f.usage = usage
	f.Default(value)
	f.variable, _ = vars.NewTyped(name, "", vars.TypeInt64)
	return f, nil
}

// Parse int flag.
func (f *IntFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			val, err := vars.NewTyped(f.name, vv[0].String(), vars.TypeInt64)
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
	return f.val
}

// Default sets default value for int flag.
func (f *IntFlag) Default(def ...int) vars.Variable {
	if len(def) > 0 && f.defval.Empty() {
		f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(def[0]), vars.TypeInt64)
		f.val = def[0]
	}
	return f.defval
}

// Unset the int flag value.
func (f *IntFlag) Unset() {
	if !f.defval.Empty() {
		f.variable = f.defval
	} else {
		f.variable, _ = vars.NewTyped(f.name, "0", vars.TypeInt64)
	}
	f.isPresent = false
	f.val = f.variable.Int()
}
