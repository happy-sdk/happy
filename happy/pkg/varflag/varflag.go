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

// Package varflag implements command-line flag parsing into compatible
// with var library.
//
// Originally based of https://pkg.go.dev/github.com/mkungla/varflag/v5
package varflag

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/mkungla/happy/pkg/vars"
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
	ErrNoNamedFlag = errors.New("no such flag")
)

type (
	CreateFlagFunc func() (Flag, error)

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
		Add(...Flag) error

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
