// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Bool returns new bool flag. Argument "a" can be any nr of aliases.
func Bool(name string, value bool, usage string, aliases ...string) (*BoolFlag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &BoolFlag{
		val:    value,
		Common: *c,
	}
	f.usage = usage
	f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(value), vars.TypeBool)
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

// Unset the bool flag value.
func (f *BoolFlag) Unset() {
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Bool()
}
