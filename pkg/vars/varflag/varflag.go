// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

// Package varflag implements command-line flag parsing into compatible
// with package https://github.com/happy-sdk/happy/pkg/vars.
//
// Originally based of https://pkg.go.dev/github.com/mkungla/varflag/v5
package varflag

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/happy-sdk/happy/pkg/vars"
)

const (
	// FlagRe check flag name against following expression.
	FlagRe = "^[a-z][a-z0-9_-]*[a-z0-9]$"
)

var (
	// ErrFlag is returned when flag fails to initialize.
	ErrFlag = errors.New("flag error")
	// ErrFlag is returned when flag fails to initialize.
	ErrFlagExists = errors.New("flag already exists")
	// ErrParse is used to indicate parse errors.
	ErrParse = errors.New("flag parse error")
	// ErrMissingValue is used when flag value was not in parsed args.
	ErrMissingValue = errors.New("missing value for flag")
	// ErrInvalidValue is used when invalid value was provided.
	ErrInvalidValue = errors.New("invalid value for flag")
	// ErrFlagAlreadyParsed is returned when this flag was already parsed.
	ErrFlagAlreadyParsed = errors.New("flag is already parsed")
	// ErrMissingRequired indicates that required flag was not in argument set.
	ErrMissingRequired = errors.New("missing required flag")
	// ErrMissingOptions is returned when option flag parser does not find options.
	ErrMissingOptions = errors.New("missing options")
	// ErrNoNamedFlag is returned when flag lookup can not find named flag.
	ErrNoNamedFlag      = errors.New("no such flag")
	ErrInvalidArguments = errors.New("invalid arguments")
)

type (
	FlagCreateFunc func() (Flag, error)

	// Flag is howi/cli/flags.Flags interface.
	Flag interface {
		// Get primary name for the flag. Usually that is long option
		Name() string

		// Get flag default value
		Default() vars.Variable

		// Usage returns a usage description for that flag
		Usage() string

		// Flag returns flag with leading - or --
		// useful for help menus
		Flag() string

		// Return flag aliases
		Aliases() []string
		UsageAliases() string

		// IsHidden reports whether to show that flag in help menu or not.
		Hidden() bool

		// Hide flag from help menu.
		Hide()

		// IsGlobal reports whether this flag was global and was set before any command or arg
		Global() bool

		// BelongsTo marks flag non global and belonging to provided named command.
		AttachTo(cmdname string)
		// BelongsTo returns empty string if command is not set with .AttachTo
		// When AttachTo is set to wildcard "*" then this function will return
		// name of the command which triggered this flag to be parsed.
		BelongsTo() string

		// Pos returns flags position after command. In case of mulyflag first position is reported
		Pos() int
		// Unset unsets the value for the flag if it was parsed, handy for cases where
		// one flag cancels another like --debug cancels --verbose
		Unset()

		// Present reports whether flag was set in commandline
		Present() bool

		// Var returns vars.Variable for this flag.
		// where key is flag and Value flags value.
		Var() vars.Variable

		// Required sets this flag as required
		MarkAsRequired()

		// IsRequired returns true if this flag is required
		Required() bool

		// Parse value for the flag from given string. It returns true if flag
		// was found in provided args string and false if not.
		// error is returned when flag was set but had invalid value.
		Parse([]string) (bool, error)

		// String calls Value().String()
		String() string

		Input() []string
		// setCommandName(string)
	}

	// Flags provides interface for flag set.
	Flags interface {
		// Name of the flag set
		Name() string

		// Len returns number of flags in this set
		// not including subset flags.
		Len() int

		// Add flag to flag set
		Add(...Flag) error

		// Get named flag
		Get(name string) (Flag, error)

		// Add sub set of flags to flag set
		AddSet(...Flags) error

		// GetActiveSets.
		GetActiveSets() []Flags

		// Position of flag set
		Pos() int

		// Was flagset (sub command present)
		Present() bool

		Args() []vars.Value

		// AcceptsArgs returns true if set accepts any arguments.
		AcceptsArgs() bool

		// Flags returns slice of flags in this set
		Flags() []Flag

		// Sets retruns subsets of flags under this flagset.
		Sets() []Flags

		// Parse all flags and sub sets
		Parse(args []string) error
	}
)

// ValidFlagName returns true if s is string which is valid flag name.
func ValidFlagName(s string) bool {
	if len(s) == 1 {
		return unicode.IsLetter(rune(s[0]))
	}
	re := regexp.MustCompile(FlagRe)
	return re.MatchString(s)
}

// Parse parses flags against provided args,
// If one of the flags fails to parse, it will return error.
func Parse(flags []Flag, args []string) error {
	var errs error
	for _, flag := range flags {
		_, err := flag.Parse(args)
		if err != nil {
			if errs == nil {
				errs = err
			} else {
				errs = fmt.Errorf("%w: %s", errs, err)
			}
		}
	}
	return errs
}

// returns elements in a which are not in b.
func slicediff(a, b []string) []string {
	var noop = struct{}{}
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = noop
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func normalizeAliases(a []string) []string {
	aliases := []string{}
	for _, alias := range a {
		aliases = append(aliases, strings.TrimLeft(alias, "-"))
	}
	return aliases
}

// NewFlagSet is wrapper to parse flags together.
// e.g. under specific command. Where "name" is command name
// to search before parsing the flags under this set.
// argsn is number of command line arguments allowed within this set.
// If argsn is -gt 0 then parser will stop after finding argsn+1 argument
// which is not a flag.
func NewFlagSetAs[
	FLAGS FlagsIface[FLAGSET, FLAG, VAR, VAL],
	FLAGSET FlagsSetIface[FLAG, VAR, VAL],
	FLAG FlagIface[VAR, VAL],
	VAR vars.VariableIface[VAL],
	VAL vars.ValueIface,
](name string, argsn int) (FLAGS, error) {
	var flags FlagsIface[FLAGSET, FLAG, VAR, VAL]

	n, err := vars.ParseKey(name)
	if err != nil {
		return flags.(FLAGS), fmt.Errorf("name %q is not valid for flag set", name)
	}
	if len(os.Args) > 0 && n == filepath.Base(os.Args[0]) {
		n = "/"
	}
	flags = &GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]{name: n, argsn: argsn}
	return flags.(FLAGS), nil
}

type FlagsSetIface[
	FLAG FlagIface[VAR, VAL],
	VAR vars.VariableIface[VAL],
	VAL vars.ValueIface,
] interface {
	Name() string
	Add(flag ...FLAG) error
	Get(name string) (FLAG, error)
	Args() []VAL
	Flags() []FLAG
	Present() bool
	Parse(args []string) error
}

type FlagsIface[
	FLAGS FlagsSetIface[FLAG, VAR, VAL],
	FLAG FlagIface[VAR, VAL],
	VAR vars.VariableIface[VAL],
	VAL vars.ValueIface,
] interface {
	FlagsSetIface[FLAG, VAR, VAL]
	GetActiveSets() []FLAGS
	Sets() []FLAGS
}

type GenericFlagSet[
	FLAGS FlagsIface[FLAGSET, FLAG, VAR, VAL],
	FLAGSET FlagsSetIface[FLAG, VAR, VAL],
	FLAG FlagIface[VAR, VAL],
	VAR vars.VariableIface[VAL],
	VAL vars.ValueIface,
] struct {
	mu      sync.RWMutex
	name    string
	argsn   int
	present bool
	flags   []FLAG
	sets    []FLAGS
	args    []VAL
	pos     int
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.name
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.flags)
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Add(flag ...FLAG) error {
	for _, f := range flag {
		_, err := s.Get(f.Name())
		if !errors.Is(err, ErrNoNamedFlag) {
			return fmt.Errorf("%w: %s", ErrFlagExists, f.Name())
		}
		f.AttachTo(s.name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flags = append(s.flags, flag...)
	return nil
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Get(name string) (f FLAG, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, flag := range s.flags {
		if flag.Name() == name {
			return flag, nil
		}
	}
	return f, ErrNoNamedFlag
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) AddSet(set ...FLAGS) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sets = append(s.sets, set...)
	return nil
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) GetActiveSets() []FLAGSET {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var tree []FLAGSET
	var root FlagsIface[FLAGSET, FLAG, VAR, VAL] = s

	tree = append(tree, root.(FLAGSET))
	if s.Present() {
		for _, set := range s.sets {
			if set.Present() {
				tree = append(tree, set.GetActiveSets()...)
			}
		}
	}
	return tree
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) extractArgs(args []string) error {
	if len(args) == 0 || s.argsn == 0 {
		return nil
	}
	// rm subcmds
includessubset:
	for _, set := range s.sets {
		if set.Present() {
			for i, arg := range args {
				if arg == set.Name() {
					args = args[:i]
					if i == 0 {
						return nil
					}
					// there is no args for cuurent set since
					// there is sub set which was first arg
					break includessubset
				}
			}
		}
	}

	// filter flags
	used := []string{}
	if args[0] == s.name || args[0] == os.Args[0] {
		used = append(used, args[0])
	}

	sargs := slicediff(args, used)
	for _, arg := range sargs {
		a, err := vars.New(s.name, arg, true)
		a1 := vars.AsVariable[VAR, VAL](a)
		if err != nil {
			return err
		}

		if s.argsn == -1 || len(s.args) <= s.argsn {
			s.args = append(s.args, a1.Value())
		} else {
			break
		}
	}
	return nil
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Pos() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pos
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Present() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.present
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Args() []VAL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.args
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) AcceptsArgs() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.argsn > 0
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Flags() []FLAG {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.flags
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Sets() []FLAGSET {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sets []FLAGSET

	for _, s := range s.sets {
		var set FlagsSetIface[FLAG, VAR, VAL] = s
		sets = append(sets, set.(FLAGSET))
	}

	return sets
}

func (s *GenericFlagSet[FLAGS, FLAGSET, FLAG, VAR, VAL]) Parse(args []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var currargs []string
	if s.name != "/" && s.name != "*" {
		for i, arg := range args {
			if arg == s.name {
				s.pos = i
				currargs = args[i:]
				s.present = true
			}
		}
	} else {
		currargs = args
		// root cmd is always considered as present
		s.present = true
	}

	// if set is not present it is not an error
	if !s.present {
		return nil
	}

	// first parse flags for current set
	for _, gflag := range s.flags {
		_, err := gflag.Parse(currargs)
		if err != nil && !errors.Is(err, ErrFlagAlreadyParsed) {
			return fmt.Errorf("%s %w", s.name, err)
		}
		// this flag need to be removed from sub command args
		if gflag.Present() {
			currargs = slicediff(currargs, gflag.Input())
		}
	}

	// parse flags for sets
	for _, set := range s.sets {
		err := set.Parse(currargs)
		if err != nil {
			return err
		}
		if set.Present() {
			if s.name == "/" {
				// update global flag command names
				for _, flag := range s.flags {
					if !flag.Present() {
						continue
					}
				}
			}
			break
		}
	}

	// since we did not have errors we can look up args
	return s.extractArgs(currargs)
}

type FlagIface[VAR vars.VariableIface[VAL], VAL vars.ValueIface] interface {
	Name() string
	Default() VAR
	Usage() string
	Flag() string
	Aliases() []string
	UsageAliases() string
	Hidden() bool
	Hide()
	Global() bool
	AttachTo(cmdname string)
	BelongsTo() string
	Pos() int
	Unset()
	Present() bool
	Var() VAR
	MarkAsRequired()
	Required() bool
	Parse([]string) (bool, error)
	Input() []string
}

func AsFlag[
	FLAG FlagIface[VAR, VAL],
	VAR vars.VariableIface[VAL],
	VAL vars.ValueIface,
](in Flag) FLAG {
	var f FlagIface[VAR, VAL] = &GenericFlag[VAR, VAL]{in}
	return f.(FLAG)
}

type GenericFlag[VAR vars.VariableIface[VAL], VAL vars.ValueIface] struct {
	f Flag
}

func (f *GenericFlag[VAR, VAL]) Var() (v VAR) {
	return vars.AsVariable[VAR, VAL](f.f.Var())
}

func (f *GenericFlag[VAR, VAL]) Default() VAR {
	return vars.AsVariable[VAR, VAL](f.f.Default())
}

// Get primary name for the flag. Usually that is long option
func (f *GenericFlag[VAR, VAL]) Name() string {
	return f.f.Name()
}

// Usage returns a usage description for that flag
func (f *GenericFlag[VAR, VAL]) Usage() string {
	return f.f.Usage()
}

// Flag returns flag with leading - or --
// useful for help menus
func (f *GenericFlag[VAR, VAL]) Flag() string {
	return f.f.Flag()
}

// Return flag aliases
func (f *GenericFlag[VAR, VAL]) Aliases() []string {
	return f.f.Aliases()
}
func (f *GenericFlag[VAR, VAL]) UsageAliases() string {
	return f.f.UsageAliases()
}

// IsHidden reports whether to show that flag in help menu or not.
func (f *GenericFlag[VAR, VAL]) Hidden() bool {
	return f.f.Hidden()
}

// Hide flag from help menu.
func (f *GenericFlag[VAR, VAL]) Hide() {
	f.f.Hide()
}

// IsGlobal reports whether this flag was global and was set before any command or arg
func (f *GenericFlag[VAR, VAL]) Global() bool {
	return f.f.Global()
}

// BelongsTo marks flag non global and belonging to provided named command.
func (f *GenericFlag[VAR, VAL]) AttachTo(cmdname string) {
	f.f.AttachTo(cmdname)
}

// BelongsTo returns empty string if command is not set with .BelongsTo
// When BelongsTo is set to wildcard "*" then this function will return
// name of the command which triggered this flag to be parsed.
func (f *GenericFlag[VAR, VAL]) BelongsTo() string {
	return f.f.BelongsTo()
}

// Pos returns flags position after command. In case of mulyflag first position is reported
func (f *GenericFlag[VAR, VAL]) Pos() int {
	return f.f.Pos()
}

// Unset unsets the value for the flag if it was parsed, handy for cases where
// one flag cancels another like --debug cancels --verbose
func (f *GenericFlag[VAR, VAL]) Unset() {
	f.f.Unset()
}

// Present reports whether flag was set in commandline
func (f *GenericFlag[VAR, VAL]) Present() bool {
	return f.f.Present()
}

// Required sets this flag as required
func (f *GenericFlag[VAR, VAL]) MarkAsRequired() {
	f.f.MarkAsRequired()
}

// IsRequired returns true if this flag is required
func (f *GenericFlag[VAR, VAL]) Required() bool {
	return f.f.Required()
}

// Parse value for the flag from given string. It returns true if flag
// was found in provided args string and false if not.
// error is returned when flag was set but had invalid value.
func (f *GenericFlag[VAR, VAL]) Parse(args []string) (bool, error) {
	return f.f.Parse(args)
}

// String calls Value().String()
func (f *GenericFlag[VAR, VAL]) String() string {
	return f.f.String()
}

func (f *GenericFlag[VAR, VAL]) Input() []string {
	return f.f.Input()
}
