// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"
	"strings"

	"github.com/mkungla/vars/v6"
)

// Float64 returns new float flag. Argument "a" can be any nr of aliases.
func Float64(name string, value float64, usage string, aliases ...string) (*Float64Flag, error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}

	f := &Float64Flag{}
	f.name = strings.TrimLeft(name, "-")
	f.val = value
	f.aliases = normalizeAliases(aliases)
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
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

// Unset the bool flag value.
func (f *Float64Flag) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Float64()
}
