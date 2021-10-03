// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Bool returns new bool flag. Argument "a" can be any nr of aliases.
func Bool(name string, aliases ...string) (*BoolFlag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &BoolFlag{val: false, Common: *c}
	f.variable, _ = vars.NewTyped(name, "false", vars.TypeBool)
	return f, nil
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
	return f.val
}

// Default sets default value for boool flag.
func (f *BoolFlag) Default(def ...bool) vars.Variable {
	if len(def) > 0 && f.defval.Empty() {
		f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(def[0]), vars.TypeBool)
		f.val = def[0]
	}
	return f.defval
}

// Unset the bool flag value.
func (f *BoolFlag) Unset() {
	if !f.defval.Empty() {
		f.variable = f.defval
	} else {
		f.variable, _ = vars.NewTyped(f.name, "false", vars.TypeBool)
	}
	f.isPresent = false
	f.val = f.variable.Bool()
}
