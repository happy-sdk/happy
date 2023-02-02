// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

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

	flag.defval, err = vars.NewAs(name, strings.Join(value, "|"), true, vars.KindString)
	if err != nil {
		return nil, err
	}
	for _, o := range opts {
		flag.opts[o] = false
	}

	flag.variable, err = vars.NewAs(name, strings.Join(value, "|"), true, vars.KindString)
	return flag, err
}

func OptionFunc(name string, value []string, opts []string, usage string, aliases ...string) FlagCreateFunc {
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
			v, err := vars.NewAs(f.name, dd, false, vars.KindString)
			if err != nil {
				return false, err
			}
			opts = append(opts, v)
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
		v, err := vars.NewAs(f.name, strings.Join(str, "|"), false, vars.KindString)
		if err != nil {
			return false, err
		}
		f.variable = v
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
