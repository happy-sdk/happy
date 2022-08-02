// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

// Package varflag implements command-line flag parsing into vars.Variables
// for easy type handling with additional flag types.
package varflag

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/mkungla/vars/v6"
)

const (
	// FlagRe check flag name against following expression.
	FlagRe = "^[a-z][a-z0-9_-]*[a-z0-9]$"
)

var (
	// ErrFlag is returned when flag fails to initialize.
	ErrFlag = errors.New("flag error")
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
	ErrNoNamedFlag = errors.New("no such flag")
)

type (
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

		// AliasesString returns string representing flag aliases.
		// e.g. used in help menu
		AliasesString() string

		// IsHidden reports whether to show that flag in help menu or not.
		IsHidden() bool

		// Hide flag from help menu.
		Hide()

		// IsGlobal reports whether this flag was global and was set before any command or arg
		IsGlobal() bool

		// BelongsTo marks flag non global and belonging to provided named command.
		BelongsTo(cmdname string)

		// CommandName returns empty string if command is not set with .BelongsTo
		// When BelongsTo is set to wildcard "*" then this function will return
		// name of the command which triggered this flag to be parsed.
		CommandName() string

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
		Required()

		// IsRequired returns true if this flag is required
		IsRequired() bool

		// Parse value for the flag from given string. It returns true if flag
		// was found in provided args string and false if not.
		// error is returned when flag was set but had invalid value.
		Parse([]string) (bool, error)

		// String calls Value().String()
		String() string

		input() []string
		setCommandName(string)
	}

	// Flags provides interface for flag set.
	Flags interface {
		// Add flag to flag set
		Add(...Flag)

		// Add sub set of flags to flag set
		AddSet(...Flags)

		// Parse all flags and sub sets
		Parse(args []string) error

		// Was flagset (sub command present)
		Present() bool

		// Name of the flag set
		Name() string

		// Position of flag set
		Pos() int

		// GetActiveSetTree.
		GetActiveSetTree() []Flags

		// Get named flag
		Get(name string) (Flag, error)

		// Len returns number of flags in this set
		// not including subset flags.
		Len() int

		// AcceptsArgs returns true if set accepts any arguments.
		AcceptsArgs() bool

		// Flags returns slice of flags in this set
		Flags() []Flag

		// Sets retruns subsets of flags under this flagset.
		Sets() []Flags
	}

	// Common is default string flag. Common flag ccan be used to
	// as base for custom flags by owerriding .Parse func.
	Common struct {
		mu sync.RWMutex
		// name of this flag
		name string
		// aliases for this flag
		aliases []string
		// hide from help menu
		hidden bool
		// global is set to true if value was parsed before any command or arg occurred
		global bool
		// position in os args how many commands where before that flag
		pos int
		// usage string
		usage string
		// isPresent enables to mock removal and .Unset the flag it reports whether flag was "present"
		isPresent bool
		// value for this flag
		variable vars.Variable
		// is this flag required
		required bool
		// default value
		defval vars.Variable
		// flag already parsed
		parsed bool
		// potential command after which this flag was found
		command string
		// arg or args based on which this flag was parsed
		in []string
	}

	// FlagSet holds collection of flags for parsing
	// e.g. global, sub command or custom flag collection.
	FlagSet struct {
		mu      sync.RWMutex
		name    string
		argsn   uint
		present bool
		flags   []Flag
		sets    []Flags
		args    []vars.Value
		pos     int
	}

	// OptionFlag is string flag type which can have value of one of the options.
	OptionFlag struct {
		Common
		opts map[string]bool
		val  []string
	}

	// BoolFlag is boolean flag type with default value "false".
	BoolFlag struct {
		Common
		val bool
	}

	// DurationFlag defines a time.Duration flag with specified name.
	DurationFlag struct {
		Common
		val time.Duration
	}

	// Float64Flag defines a float64 flag with specified name.
	Float64Flag struct {
		Common
		val float64
	}

	// IntFlag defines an int flag with specified name,.
	IntFlag struct {
		Common
		val int
	}

	// UintFlag defines a uint flag with specified name.
	UintFlag struct {
		Common
		val uint
	}

	// BexpFlag expands flag args with bash brace expansion.
	BexpFlag struct {
		Common
		val []string
	}
)

// New returns new common string flag. Argument "a" can be any nr of aliases.
func New(name string, value string, usage string, aliases ...string) (*Common, error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}

	f := &Common{}
	f.name = strings.TrimLeft(name, "-")
	f.aliases = normalizeAliases(aliases)
	f.defval = vars.New(f.name, value)
	f.usage = usage
	f.variable = vars.New(name, "")
	return f, nil
}

// Parse parses flags against provided args,
// If one of the flags fails to parse, it will return
// wrapped error these errors.
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

// NewFlagSet is wrapper to parse flags together.
// e.g. under specific command. Where "name" is command name
// to search before parsing the flags under this set.
// argsn is number of command line arguments allowed within this set.
// If argsn is -gt 0 then parser will stop after finding argsn+1 argument
// which is not a flag.
func NewFlagSet(name string, argsn uint) (*FlagSet, error) {
	if name == "/" || (len(os.Args) > 0 && name == os.Args[0]) {
		name = "/"
	} else if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: name %q is not valid for flag set", ErrFlag, name)
	}
	return &FlagSet{name: name, argsn: argsn}, nil
}

// ValidFlagName returns true if s is string which is valid flag name.
func ValidFlagName(s string) bool {
	if len(s) == 1 {
		return unicode.IsLetter(rune(s[0]))
	}
	re := regexp.MustCompile(FlagRe)
	return re.MatchString(s)
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
