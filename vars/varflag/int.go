// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"
	"strings"

	"github.com/mkungla/vars/v6"
)

// Int returns new int flag. Argument "a" can be any nr of aliases.
func Int(name string, value int, usage string, aliases ...string) (*IntFlag, error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}
	f := &IntFlag{}
	f.usage = usage
	f.name = strings.TrimLeft(name, "-")
	f.val = value
	f.aliases = normalizeAliases(aliases)
	f.defval, _ = vars.NewTyped(f.name, fmt.Sprint(value), vars.TypeInt)
	f.variable = f.defval
	return f, nil
}

// Parse int flag.
func (f *IntFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			val, err := vars.NewTyped(f.name, vv[0].String(), vars.TypeInt)
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
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

// Unset the int flag value.
func (f *IntFlag) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Int()
}
