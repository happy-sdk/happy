// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package varflag

import (
	"fmt"
	"strings"

	"github.com/happy-sdk/happy/pkg/strings/bexp"
	"github.com/happy-sdk/happy/pkg/vars"
)

// BexpFlag expands flag args with bash brace expansion.
type BexpFlag struct {
	Common
	val []string
}

// Bexp returns new Bash Brace Expansion flag.
func Bexp(name string, value string, usage string, aliases ...string) (flag *BexpFlag, err error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}

	flag = &BexpFlag{}
	flag.name = strings.TrimLeft(name, "-")
	flag.aliases = normalizeAliases(aliases)
	flag.defval, err = vars.NewAs(name, value, false, vars.KindString)
	if err != nil {
		return nil, err
	}
	flag.usage = usage
	flag.val = []string{}
	flag.variable, err = vars.NewAs(name, value, false, vars.KindString)
	return flag, err
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
	f.variable, err = vars.New(f.name, strings.Join(f.val, "|"), false)
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

	f.variable, _ = vars.New(f.name, strings.Join(defaults, "|"), false)
	f.isPresent = false
	f.val = defaults
}
