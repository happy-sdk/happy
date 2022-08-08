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
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mkungla/happy/pkg/vars"
)

// OptionFlag is string flag type which can have value of one of the options.
type OptionFlag struct {
	Common
	opts map[string]bool
	val  []string
}

// Option returns new string flag. Argument "opts" is string slice
// of options this flag accepts.
func Option(name string, value []string, opts []string, usage string, aliases ...string) (flag *OptionFlag, err error) {
	if len(opts) == 0 {
		return nil, ErrMissingOptions
	}
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}
	flag = &OptionFlag{}
	flag.usage = usage
	flag.opts = make(map[string]bool, len(opts))
	flag.name = strings.TrimLeft(name, "-")
	flag.aliases = normalizeAliases(aliases)

	flag.defval, err = vars.String(name, strings.Join(value, "|"), true)
	if err != nil {
		return nil, err
	}
	for _, o := range opts {
		flag.opts[o] = false
	}

	flag.variable, err = vars.String(name, strings.Join(value, "|"), true)
	return flag, err
}

func OptionFunc(name string, value []string, opts []string, usage string, aliases ...string) CreateFlagFunc {
	return func() (Flag, error) {
		return Option(name, value, opts, usage, aliases...)
	}
}

// Parse the OptionFlag.
func (f *OptionFlag) Parse(args []string) (ok bool, err error) {
	if f.parsed {
		return false, ErrFlagAlreadyParsed
	}
	var opts []vars.Variable

	if !f.defval.Empty() {
		defval := strings.Split(f.defval.String(), "|")
		for _, dd := range defval {
			opts = append(opts, vars.New(f.name, dd))
			// opts = append(opts, vars.New(f.name+":default", dd))
		}
	}

	_, err = f.parse(args, func(v []vars.Variable) (err error) {
		opts = v
		return err
	})

	f.mu.Lock()
	defer f.mu.Unlock()

	if err != nil && f.defval.Empty() {
		return f.isPresent, err
	}

	if len(opts) > 0 {
		var str []string
		for _, o := range opts {
			if _, isSet := f.opts[o.String()]; !isSet {
				return f.isPresent, fmt.Errorf("%w: (%s=%q)", ErrInvalidValue, f.name, o)
			}
			f.opts[o.String()] = true
			str = append(str, o.String())
		}
		sort.Strings(str)
		f.val = str
		f.parsed = true
		f.variable = vars.New(f.name, strings.Join(str, "|"))
	}

	if !f.defval.Empty() && errors.Is(err, ErrMissingValue) {
		f.isPresent = true
		return f.isPresent, nil
	}

	return f.isPresent, err
}

// Value returns parsed options.
func (f *OptionFlag) Value() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}
