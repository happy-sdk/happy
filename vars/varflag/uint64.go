// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"

	"github.com/mkungla/vars/v5"
)

// Uint64 returns new uint64 flag. Argument "a" can be any nr of aliases.
func Uint64(name string, aliases ...string) (*Uint64Flag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &Uint64Flag{val: 0, Common: *c}
	f.variable, _ = vars.NewTyped(name, "", vars.TypeUint64)
	return f, nil
}

// Parse uint flag.
func (f *Uint64Flag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			f.variable = vv[0]
			f.val = f.variable.Uint64()
		}
		return err
	})
}

// Get uint64 flag value, it returns default value if not present
// or 0 if default is also not set.
func (f *Uint64Flag) Value() uint64 {
	return f.val
}

// Set default value for uint flag.
func (f *Uint64Flag) Default(def ...uint64) vars.Variable {
	if len(def) > 0 && f.defval.Empty() {
		f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(def[0]), vars.TypeUint64)
		f.val = def[0]
	}
	return f.defval
}
