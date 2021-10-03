// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Int64 returns new int64 flag. Argument "a" can be any nr of aliases.
func Int64(name string, aliases ...string) (*Int64Flag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &Int64Flag{val: 0, Common: *c}
	f.variable, _ = vars.NewTyped(name, "", vars.TypeInt64)
	return f, nil
}

// Parse int64 flag.
func (f *Int64Flag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			f.variable = vv[0]
			f.val = f.variable.Int64()
		}
		return err
	})
}

// Value returns int64 flag value, it returns default value if not present
// or 0 if default is also not set.
func (f *Int64Flag) Value() int64 {
	return f.val
}

// Default sets default value for int64 flag.
func (f *Int64Flag) Default(def ...int64) vars.Variable {
	if len(def) > 0 && f.defval.Empty() {
		f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(def[0]), vars.TypeInt64)
		f.val = def[0]
	}
	return f.defval
}
