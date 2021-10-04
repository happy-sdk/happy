// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Float64 returns new float flag. Argument "a" can be any nr of aliases.
func Float64(name string, value float64, usage string, aliases ...string) (*Float64Flag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &Float64Flag{val: value, Common: *c}
	f.usage = usage
	f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(value), vars.TypeFloat64)
	f.variable = f.defval
	return f, nil
}

// Parse float flag.
func (f *Float64Flag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			val, err := vars.NewTyped(f.name, vv[0].String(), vars.TypeFloat64)
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
	return f.val
}

// Unset the bool flag value.
func (f *Float64Flag) Unset() {
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Float64()
}
