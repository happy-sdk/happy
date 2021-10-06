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
func (f *Common) Default() vars.Variable {
	return f.defval
}

// Usage returns a usage description for that flag.
func (f *Common) Usage() string {
	if !f.defval.Empty() {
		return fmt.Sprintf("%s - default: %q", f.usage, f.defval.String())
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
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.hidden
}

// Hide flag from help menu.
func (f *Common) Hide() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hidden = true
}

// IsGlobal reports whether this flag is global.
// By default all flags are global flags. You can mark flag non-global
// by calling .BelongsTo(cmdname string).
func (f *Common) IsGlobal() bool {
	return f.global
}

// BelongsTo marks flag non global and belonging to provided named command.
// Parsing the flag will only succeed if naemd command was found before the flag.
// This is useful to have same flag both global and sub command level.
// Special case is .BelongsTo("*") which marks flag to be parsed
// if any subcommand is present.
// e.g. verbose flag:
// you can define 2 BoolFlag's with name "verbose" and alias "v"
// and mark one of these with BelongsTo("*").
// BelongsTo(os.Args[0] | "/") are same as global and will be.
func (f *Common) BelongsTo(cmdname string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.command) == 0 {
		f.command = cmdname
	}
}

// CommandName returns empty string if command is not set with .BelongsTo
// When BelongsTo is set to wildcard "*" then this function will return
// name of the command which triggered this flag to be parsed.
func (f *Common) CommandName() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.command
}

// Pos returns flags position after command and case of global since app name
// min value is 1 which means first global flag or first flag after command.
func (f *Common) Pos() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.pos
}

// Unset the value.
func (f *Common) Unset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.defval.Empty() {
		f.variable = f.defval
	} else {
		f.variable = vars.New(f.name, "")
	}
	f.isPresent = false
}

// Present reports whether flag was set in commandline.
func (f *Common) Present() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.isPresent
}

// Var returns vars.Variable for this flag.
// where key is flag and Value flags value.
func (f *Common) Var() vars.Variable {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.variable
}

// Value returns string value of flag.
func (f *Common) Value() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.variable.String()
}

// Required sets this flag as required.
func (f *Common) Required() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.required {
		f.required = true
	}
}

// IsRequired returns true if this flag is required.
func (f *Common) IsRequired() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
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
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.variable.String()
}

func (f *Common) input() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.in
}

func (f *Common) setCommandName(cmdname string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.command = cmdname
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

	f.mu.Lock()
	defer f.mu.Unlock()
	err := f.parseArgs(args, read)
	return f.isPresent, err
}

func (f *Common) parseArgs(args []string, read func([]vars.Variable) error) (err error) {
	var (
		values []vars.Variable
		poses  []int // slice of positions (useful for multiflag)
		pargs  []string
	)

	poses, pargs, err = f.normalize(args)
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

	values, err = f.parseValues(poses, pargs)
	if err != nil {
		return err
	}

	// what was before the flag including flag it self

	// default is global
	f.global = f.command == "" || f.command == "/"
	if len(pargs) < poses[0] {
		return nil
	}

	pre := pargs[:poses[0]]
	if f.command != "" {
		var cmd string

		opts := 0
		for _, arg := range pre {
			if len(arg) == 0 || arg[0] == '-' {
				opts = 0
				continue
			}
			opts++
			if opts > 1 {
				cmd = arg
			}
			// found portential command
			if len(cmd) > 0 && (f.command == "*" || cmd == f.command) {
				f.command = cmd
				break
			}
		}
	}

	return read(values)
}

func (f *Common) parseValues(poses []int, pargs []string) ([]vars.Variable, error) {
	values := []vars.Variable{}

	for _, pose := range poses {
		if f.pos == 0 {
			f.pos = pose
		}

		// handle bool flags
		if f.variable.Type() == vars.TypeBool {
			var value vars.Variable
			falsestr := "false"
			bval := "true" //nolint: goconst
			if len(pargs) > pose {
				val := pargs[pose]
				switch val {
				case falsestr, "0", "off":
					bval = falsestr
					f.in = append(f.in, val)
				case "true", "1", "on":
					f.in = append(f.in, val)
				}
			}
			// no need for err check since we only pass valid strings
			value, _ = vars.NewTyped(f.name, bval, vars.TypeBool)
			f.isPresent = true
			values = append(values, value)
			continue
		}
		if len(pargs) == pose {
			return values, fmt.Errorf("%w: %s", ErrMissingValue, f.name)
		}

		// update pose only once for first occourance

		f.isPresent = true
		value := pargs[pose]
		f.in = append(f.in, pargs[pose])
		// if we get other flags we can validate is value a flag or not
		values = append(values, vars.New(f.name, value))
	}

	return values, nil
}

// normalize reports flag positions if flag is present and returns normalized
// arg slice where key=val is already correctly splitted.
func (f *Common) normalize(args []string) (pos []int, pargs []string, err error) {
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
			f.in = append(f.in, arg)
		} else {
			// or is one of aliases
			for _, alias := range f.aliases {
				if currflag == alias {
					pos = append(pos, rpos)
					f.in = append(f.in, arg)
					break
				}
			}
		}

		// not this one
		if split {
			rpos++
			split = false
		}
	}
	return pos, pargs, err
}

func normalizeAliases(a []string) []string {
	aliases := []string{}
	for _, alias := range a {
		aliases = append(aliases, strings.TrimLeft(alias, "-"))
	}
	return aliases
}
