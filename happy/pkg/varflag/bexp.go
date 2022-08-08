// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package varflag

import (
	"fmt"
	"strings"

	"github.com/mkungla/bexp/v3"
	"github.com/mkungla/happy/pkg/vars"
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
	flag.defval, err = vars.String(name, value, true)
	if err != nil {
		return nil, err
	}
	flag.usage = usage
	flag.val = []string{}
	flag.variable, err = vars.String(name, value, false)
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
