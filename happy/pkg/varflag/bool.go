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

	"github.com/mkungla/happy/pkg/vars"
)

// BoolFlag is boolean flag type with default value "false".
type BoolFlag struct {
	Common
	val bool
}

// Bool returns new bool flag. Argument "a" can be any nr of aliases.
func Bool(name string, value bool, usage string, aliases ...string) (flag *BoolFlag, err error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}

	flag = &BoolFlag{}
	flag.name = strings.TrimLeft(name, "-")
	flag.val = value
	flag.aliases = normalizeAliases(aliases)
	flag.usage = usage
	flag.defval, err = vars.Bool(name, value, true)
	if err != nil {
		return nil, err
	}
	flag.variable, err = vars.Bool(name, value, false)
	return flag, err
}

func BoolFunc(name string, value bool, usage string, aliases ...string) CreateFlagFunc {
	return func() (Flag, error) {
		return Bool(name, value, usage, aliases...)
	}
}

// Parse bool flag.
func (f *BoolFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			f.variable = vv[0]
			f.val = f.variable.Bool()
		}
		return err
	})
}

// Value returns bool flag value, it returns default value if not present
// or false if default is also not set.
func (f *BoolFlag) Value() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

// Unset the bool flag value.
func (f *BoolFlag) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Bool()
}
