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

// UintFlag defines a uint flag with specified name.
type UintFlag struct {
	Common
	val uint
}

// Uint returns new uint flag. Argument "a" can be any nr of aliases.
func Uint(name string, value uint, usage string, aliases ...string) (flag *UintFlag, err error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}
	flag = &UintFlag{}
	flag.usage = usage
	flag.name = strings.TrimLeft(name, "-")
	flag.val = value
	flag.aliases = normalizeAliases(aliases)
	flag.val = value
	flag.defval, err = vars.Uint(flag.name, value, true)
	if err != nil {
		return nil, err
	}
	flag.variable, err = vars.Uint(flag.name, value, false)
	return flag, err
}

func UintFunc(name string, value uint, usage string, aliases ...string) CreateFlagFunc {
	return func() (Flag, error) {
		return Uint(name, value, usage, aliases...)
	}
}

// Parse uint flag.
func (f *UintFlag) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			val, err := vars.NewTyped(f.name, vv[0].String(), vars.TypeUint)
			if err != nil {
				return fmt.Errorf("%w: %q", ErrInvalidValue, err)
			}
			f.variable = val
			f.val = f.variable.Uint()
		}
		return err
	})
}

// Value returns uint flag value, it returns default value if not present
// or 0 if default is also not set.
func (f *UintFlag) Value() uint {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

// Unset the int flag value.
func (f *UintFlag) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.variable = f.defval
	f.isPresent = false
	f.val = f.variable.Uint()
}
