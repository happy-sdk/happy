// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Float64 returns new float64 flag. Argument "a" can be any nr of aliases.
func Float64(name string, aliases ...string) (*Float64Flag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &Float64Flag{val: 0, Common: *c}
	f.variable, _ = vars.NewTyped(name, "", vars.TypeFloat64)
	return f, nil
}

// Parse float64 flag.
func (f *Float64Flag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			f.variable = vv[0]
			f.val = f.variable.Float64()
		}
		return err
	})
}

// Get float64 flag value, it returns default value if not present
// or 0 if default is also not set.
func (f *Float64Flag) Value() float64 {
	return f.val
}

// Set default value for float64 flag.
func (f *Float64Flag) Default(def ...float64) vars.Variable {
	if len(def) > 0 && f.defval.Empty() {
		f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(def[0]), vars.TypeFloat64)
		f.val = def[0]
	}
	return f.defval
}

// Unset the bool flag value.
func (f *Float64Flag) Unset() {
	if !f.defval.Empty() {
		f.variable = f.defval
	} else {
		f.variable, _ = vars.NewTyped(f.name, "0", vars.TypeFloat64)
	}
	f.isPresent = false
	f.val = f.variable.Float64()
}
