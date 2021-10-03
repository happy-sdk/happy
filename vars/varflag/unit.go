// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Uint returns new uint flag. Argument "a" can be any nr of aliases.
func Uint(name string, aliases ...string) (*UintFlag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &UintFlag{val: 0, Common: *c}
	f.variable, _ = vars.NewTyped(name, "", vars.TypeUint)
	return f, nil
}

// Parse uint flag.
func (f *UintFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			f.variable = vv[0]
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
