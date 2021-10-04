// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"time"

	"github.com/mkungla/vars/v5"
)

// Duration returns new duration flag. Argument "a" can be any nr of aliases.
func Duration(name string, value time.Duration, usage string, aliases ...string) (*DurationFlag, error) {
	c, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f := &DurationFlag{val: value, Common: *c}
	f.usage = usage
	f.defval = vars.New(f.name, value)
	f.variable, _ = vars.NewTyped(name, "", vars.TypeString)
	return f, nil
}

// Parse duration flag.
func (f *DurationFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			f.variable = vv[0]
			val, err := time.ParseDuration(vv[0].String())
			if err != nil {
				return err
			}
			f.val = val
		}
		return err
	})
}

// Value returns duration flag value, it returns default value if not present
// or 0 if default is also not set.
func (f *DurationFlag) Value() time.Duration {
	return f.val
}

// Unset the bool flag value.
func (f *DurationFlag) Unset() {
	f.variable = f.defval
	f.isPresent = false
	val, _ := time.ParseDuration(f.defval.String())
	f.val = val
}
