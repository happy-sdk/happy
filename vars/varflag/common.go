// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

package varflag

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mkungla/vars/v5"
)

// Name returns primary name for the flag usually that is long option.
func (f *Common) Name() string {
	return f.name
}

// Default sets flag default.
func (f *Common) Default(def ...interface{}) vars.Variable {
	if len(def) > 0 && f.defval.Empty() {
		f.defval = vars.New(f.name, def[0])
	}
	return f.defval
}

// Usage returns a usage description for that flag.
func (f *Common) Usage(usage ...string) string {
	if len(usage) > 0 {
		f.usage = strings.TrimSpace(usage[0])
	}
	if !f.defval.Empty() {
		return fmt.Sprintf("%s default: %q", f.usage, f.defval.String())
	}
	return f.usage
}

// Flag returns flag with leading - or --.
func (f *Common) Flag() string {
	if len(f.name) == 1 {
		return "-" + f.name
	}
	return "--" + f.name
}

// Aliases returns all aliases for the flag together with primary "name".
func (f *Common) Aliases() []string {
	return f.aliases
}

// AliasesString returns string representing flag aliases.
// e.g. used in help menu.
func (f *Common) AliasesString() string {
	if len(f.aliases) <= 1 {
		return ""
	}
	aliases := []string{}

	for _, a := range f.aliases {
		if len(a) == 1 {
			aliases = append(aliases, "-"+a)
			continue
		}
		aliases = append(aliases, "--"+a)
	}
	return strings.Join(aliases, ",")
}

// IsHidden reports whether flag should be visible in help menu.
func (f *Common) IsHidden() bool {
	return f.hidden
}

// Hide flag from help menu.
func (f *Common) Hide() {
	f.hidden = true
}

// IsGlobal reports whether this flag is global.
func (f *Common) IsGlobal() bool {
	return f.global
}

// Pos returns flags position after command and case of global since app name
// min value is 1 which means first global flag or first flag after command.
func (f *Common) Pos() int {
	return f.pos
}

// Unset the value.
func (f *Common) Unset() {
	if !f.defval.Empty() {
		f.variable = vars.New(f.name, f.defval)
	} else {
		f.variable = vars.New(f.name, "")
	}
	f.isPresent = false
}

// Present reports whether flag was set in commandline.
func (f *Common) Present() bool {
	return f.isPresent
}

// Var returns vars.Variable for this flag.
// where key is flag and Value flags value.
func (f *Common) Var() vars.Variable {
	return f.variable
}

// Value returns string value of flag.
func (f *Common) Value() string {
	return f.variable.String()
}

// Required sets this flag as required.
func (f *Common) Required() {
	f.required = true
}

// IsRequired returns true if this flag is required.
func (f *Common) IsRequired() bool {
	return f.required
}

// Parse the StringFlag.
func (f *Common) Parse(args []string) (bool, error) {
	return f.parse(args, func(vv []vars.Variable) (err error) {
		if len(vv) > 0 {
			f.variable = vv[0]
		}
		return err
	})
}

// String calls Value().String().
func (f *Common) String() string {
	return f.Var().String()
}

// Parse value for the flag from given string.
// It returns true if flag has been parsed
// and error if flag has been already parsed.
func (f *Common) parse(args []string, read func([]vars.Variable) error) (bool, error) {
	if f.parsed || f.isPresent {
		return f.isPresent, fmt.Errorf("%w: %s", ErrFlagAlreadyParsed, f.name)
	}

	if len(args) == 0 {
		return false, fmt.Errorf("%w: no arguments", ErrParse)
	}

	var err error
	f.isPresent, err = f.parseArgs(args, read)
	return f.isPresent, err
}

func (f *Common) parseArgs(args []string, read func([]vars.Variable) error) (pr bool, err error) {
	var (
		values []vars.Variable
		poses  []int // slice of positions (useful for multiflag)
		pargs  []string
	)

	poses, pargs, err = f.findFlag(args)
	if err != nil {
		return
	}

	// locate flag positions
	if len(poses) == 0 {
		if f.required {
			err = fmt.Errorf("%w: %s", ErrMissingRequired, f.name)
		}
		return
	}

	sort.Ints(poses)

	values, pr, err = f.parseValues(poses, pargs)
	if err != nil {
		return pr, err
	}

	// what was before the flag including flag it self
	pre := pargs[:poses[0]]
	// check global since first arg was a command
	if !strings.HasPrefix(pre[0], "-") {
		cmd := pre[0]
		opts := 0
		for _, arg := range pargs {
			if arg[0] == '-' {
				opts = 0
				continue
			}
			opts++
			if opts > 1 {
				cmd = arg
			}
		}
		if f.command == "" || cmd == f.command {
			f.global = true
		}
	}
	err = read(values)
	return pr, err
}

func (f *Common) parseValues(poses []int, pargs []string) ([]vars.Variable, bool, error) {
	var (
		pr     bool
		values = []vars.Variable{}
	)

	for _, pose := range poses {
		// handle bool flags
		if f.variable.Type() == vars.TypeBool {
			var value vars.Variable
			falsestr := "false"
			bval := "true"
			if len(pargs) > pose {
				switch pargs[pose] {
				case falsestr, "0", "off":
					bval = falsestr
					// case "true", "1", "on":
				}
			}
			// no need for err check since we only pass valid strings
			value, _ = vars.NewTyped(f.name, bval, vars.TypeBool)
			pr = true
			values = append(values, value)
			continue
		}

		if len(pargs) == pose {
			return values, pr, fmt.Errorf("%w: %s", ErrMissingValue, f.name)
		}
		// update pose only once for first occourance
		if f.pos == 0 {
			f.pos = pose
		}
		pr = true
		value := pargs[pose]
		// if we get other flags we can validate is value a flag or not
		values = append(values, vars.New(f.name, value))
	}
	return values, pr, nil
}

// findFlag reports flag positions if flag is present and returns normalized
// arg slice where key=val is already correctly splitted.
func (f *Common) findFlag(args []string) (pos []int, pargs []string, err error) {
	var (
		currflag string
		split    bool
		rpos     int // real pose differs if flag=value is provided
	)

	for _, arg := range args {
		rpos++

		if len(arg) == 0 || arg[0] != '-' {
			pargs = append(pargs, arg)
			continue
		}

		// found flag
		currflag = strings.TrimLeft(arg, "-")
		if strings.Contains(arg, "=") {
			var curr vars.Variable
			// no need for err check we alwways have key=val
			curr, _ = vars.NewFromKeyVal(arg)
			currflag = strings.TrimLeft(curr.Key(), "-")
			// handle only possible errors
			if len(currflag) == 0 {
				err = fmt.Errorf("%w: invalid argument -=", ErrParse)
				return
			}
			split = true
			pargs = append(pargs, curr.Key(), curr.String())
		} else {
			pargs = append(pargs, arg)
		}

		// is our flag?
		if currflag == f.name {
			pos = append(pos, rpos)
		} else {
			// or is one of aliases
			for _, alias := range f.aliases {
				if currflag == alias {
					pos = append(pos, rpos)
					break
				}
			}
		}

		// not this one
		if split {
			rpos++
		}
	}
	return pos, pargs, err
}
