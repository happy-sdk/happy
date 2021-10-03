// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Int returns new int flag. Argument "a" can be any nr of aliases.
func Int(name string, aliases ...string) (*IntFlag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &IntFlag{val: 0, Common: *c}
	f.variable, _ = vars.NewTyped(name, "", vars.TypeInt)
	return f, nil
}

// Parse int flag.
func (f *IntFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			f.variable = vv[0]
			f.val = f.variable.Int()
		}
		return err
	})
}

// Get int flag value, it returns default value if not present
// or 0 if default is also not set.
func (f *IntFlag) Value() int {
	return f.val
}

// Set default value for int flag.
func (f *IntFlag) Default(def ...int) vars.Variable {
	if len(def) > 0 && f.defval.Empty() {
		f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(def[0]), vars.TypeInt)
		f.val = def[0]
	}
	return f.defval
}
