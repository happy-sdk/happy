// Copyright 2021 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"
	"strings"

	"github.com/mkungla/bexp/v3"

	"github.com/mkungla/vars/v6"
)

// Bexp returns new Bash Brace Expansion flag.
func Bexp(name string, value string, usage string, aliases ...string) (*BexpFlag, error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}

	f := &BexpFlag{}
	f.name = strings.TrimLeft(name, "-")
	f.aliases = normalizeAliases(aliases)
	f.defval = vars.New(f.name, value)
	f.usage = usage
	f.val = []string{}
	f.variable = vars.New(name, "")
	return f, nil
}

// Parse BexpFlag.
func (f *BexpFlag) Parse(args []string) (ok bool, err error) {
	defaults := bexp.Parse(f.defval.String())
	ok, err = f.parse(args, func(v []vars.Variable) (err error) {
		for _, vv := range v {
			exp := bexp.Parse(vv.String())
			f.val = append(f.val, exp...)
		}
		return nil
	})

	f.mu.Lock()
	defer f.mu.Unlock()
	if !ok {
		f.val = defaults
	}
	f.variable = vars.New(f.name, strings.Join(f.val, "|"))
	return ok, err
}

// Value returns parsed options.
func (f *BexpFlag) Value() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.isPresent {
		return f.val
	}
	return f.val
}

// Unset the int flag value.
func (f *BexpFlag) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	defaults := bexp.Parse(f.defval.String())

	f.variable = vars.New(f.name, strings.Join(defaults, "|"))
	f.isPresent = false
	f.val = defaults
}
