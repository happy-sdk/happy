// Copyright 2016 Marko Kungla. All rights reserved.
// Use of this source code is governed by a The Apache-style
// license that can be found in the LICENSE file.

// Package varflag implements command-line flag parsing into vars.Variables
// for easy type handling with additional flag types.
package varflag

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/mkungla/vars/v5"
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
)

type (
	// Flag is howi/cli/flags.Flags interface.
	Flag interface {
		// Parse value for the flag from given string. It returns true if flag
		// was found in provided args string and false if not.
		// error is returned when flag was set but had invalid value.
		Parse([]string) (bool, error)
		// Get primary name for the flag. Usually that is long option
		Name() string
		// Usage returns a usage description for that flag
		Usage(...string) string
		// Flag returns flag with leading - or --
		// useful for help menus
		Flag() string
		// Hide flag from help menu.
		Hide()
		// Return flag aliases
		Aliases() []string
		// AliasesString returns string representing flag aliases.
		// e.g. used in help menu
		AliasesString() string
		// IsHidden reports whether to show that flag in help menu or not.
		IsHidden() bool
		// IsGlobal reports whether this flag was global and was set before any command or arg
		IsGlobal() bool
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
		// Set flag default value
		Default(def ...interface{}) vars.Variable
		// String calls Value().String()
		String() string
	}

	// Common is default string flag. Common flag ccan be used to
	// as base for custom flags by owerriding .Parse func.
	Common struct {
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
	}

	// OptionFlag is string flag type which can have value of one of the options.
	OptionFlag struct {
		Common
		opts map[string]bool
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
	FloatFlag struct {
		Common
		val float64
	}

	// IntFlag defines an int flag with specified name,.
	IntFlag struct {
		Common
		val int64
	}

	// UintFlag defines a uint flag with specified name.
	UintFlag struct {
		Common
		val uint64
	}
)

// New returns new common string flag. Argument "a" can be any nr of aliases.
func New(name string, aliases ...string) (*Common, error) {
	f, err := newCommon(name, aliases...)
	if err != nil {
		return nil, err
	}
	f.variable = vars.New(name, "")
	return f, err
}

// ValidFlagName returns true if s is string which is valid flag name.
func ValidFlagName(s string) bool {
	if len(s) == 1 {
		return unicode.IsLetter(rune(s[0]))
	}
	re := regexp.MustCompile(FlagRe)
	return re.MatchString(s)
}

func newCommon(name string, aliases ...string) (*Common, error) {
	if !ValidFlagName(name) {
		return nil, fmt.Errorf("%w: flag name %q is not valid", ErrFlag, name)
	}

	f := &Common{}
	f.name = strings.TrimLeft(name, "-")
	for _, alias := range aliases {
		f.aliases = append(f.aliases, strings.TrimLeft(alias, "-"))
	}
	return f, nil
}
