// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"fmt"
	"regexp"
	"strings"
)

// Arg is a single named translation argument, as returned by String, Int,
// Int64, Uint64, Float64, Bool, and Any, or written inline as a bare
// "key", value pair - mirroring slog's key-value/Attr convention. A
// translation template written as {name} or {name:verb} (e.g. {name:s},
// {notifications:d}, {amount:f.2}) is resolved against these by name
// rather than by position, so a translator can reorder or drop a
// placeholder without the call site needing to know or care.
type Arg struct {
	Key   string
	Value any
}

// String returns a named string argument.
func String(key, value string) Arg { return Arg{Key: key, Value: value} }

// Int returns a named int argument.
func Int(key string, value int) Arg { return Arg{Key: key, Value: value} }

// Int64 returns a named int64 argument.
func Int64(key string, value int64) Arg { return Arg{Key: key, Value: value} }

// Uint64 returns a named uint64 argument.
func Uint64(key string, value uint64) Arg { return Arg{Key: key, Value: value} }

// Float64 returns a named float64 argument.
func Float64(key string, value float64) Arg { return Arg{Key: key, Value: value} }

// Bool returns a named bool argument.
func Bool(key string, value bool) Arg { return Arg{Key: key, Value: value} }

// Any returns a named argument of any type - useful for %T/%v style
// placeholders where the point is the value's own type or default
// formatting rather than a specific verb.
func Any(key string, value any) Arg { return Arg{Key: key, Value: value} }

// namedPlaceholderRe matches {name} or {name:verb}, where verb is any
// fmt-style verb letter optionally followed by a precision, e.g. "s", "d",
// "q", "T", "f.2".
var namedPlaceholderRe = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(:[a-zA-Z][a-zA-Z0-9.]*)?\}`)

// hasNamedPlaceholders reports whether s uses this package's {name}/
// {name:verb} template syntax, as opposed to a plain %-verb fmt template
// (the latter is left entirely to the existing golang.org/x/text/message
// printer path, unchanged).
func hasNamedPlaceholders(s string) bool {
	return namedPlaceholderRe.MatchString(s)
}

// argsToMap normalizes a slog-style variadic argument list - bare
// "key", value pairs interleaved with typed Arg values from String, Int,
// Float64, etc. - into a lookup map for named template substitution.
// Malformed input (a trailing key with no value, or a value where a key
// was expected) never panics: it's recorded under a visible marker
// instead, the same spirit as fmt's own "print something, don't crash"
// contract for a bad verb/argument pairing.
func argsToMap(args []any) map[string]any {
	m := make(map[string]any, len(args))
	for i := 0; i < len(args); i++ {
		switch a := args[i].(type) {
		case Arg:
			m[a.Key] = a.Value
		case string:
			if i+1 < len(args) {
				m[a] = args[i+1]
				i++
			} else {
				m[a] = "%!(NOVALUE)"
			}
		default:
			m[fmt.Sprintf("!BADARG%d", i)] = a
		}
	}
	return m
}

// renderNamed substitutes every {name}/{name:verb} placeholder in template
// with its value from args, formatted per verb (default %v when no verb is
// given). A verb like "f.2" splits into fmt's own "%.2f". Formatting is
// delegated to fmt.Sprintf for the single matched value, so a verb/type
// mismatch (e.g. {count:d} given a string) produces fmt's own inline
// "%!d(string=...)" marker instead of panicking - exactly fmt's existing
// contract, just applied per named placeholder instead of positionally. A
// name with no matching argument produces a "%!name(MISSING)" marker.
func renderNamed(template string, args []any) string {
	return renderTemplate(template, argsToMap(args), nil)
}

// renderTemplate substitutes every {name}/{name:verb} placeholder in
// template against values, consulting resolveFragment first (if non-nil)
// for any name that might refer to something other than a literal arg -
// used by renderMessage/renderFragment to let a placeholder resolve to a
// Message's own pluralized fragment instead. Any name resolveFragment
// doesn't recognize (or when resolveFragment is nil, as for a plain
// {name}/{name:verb} template with no fragments at all) falls through to
// an ordinary values lookup, formatted per formatNamed.
func renderTemplate(template string, values map[string]any, resolveFragment func(name string) (string, bool)) string {
	return namedPlaceholderRe.ReplaceAllStringFunc(template, func(match string) string {
		sub := namedPlaceholderRe.FindStringSubmatch(match)
		name, verb := sub[1], strings.TrimPrefix(sub[2], ":")
		if resolveFragment != nil {
			if s, ok := resolveFragment(name); ok {
				return s
			}
		}
		value, ok := values[name]
		if !ok {
			return fmt.Sprintf("%%!%s(MISSING)", name)
		}
		return formatNamed(verb, value)
	})
}

// formatNamed formats value per verb (e.g. "s", "d", "T", "q", "f.2"),
// building the equivalent fmt verb string and delegating to fmt.Sprintf.
func formatNamed(verb string, value any) string {
	if verb == "" {
		return fmt.Sprintf("%v", value)
	}
	verbLetter := verb
	precision := ""
	if idx := strings.IndexByte(verb, '.'); idx != -1 {
		precision = verb[idx:]
		verbLetter = verb[:idx]
	}
	if verbLetter == "" {
		verbLetter = "v"
	}
	return fmt.Sprintf("%"+precision+verbLetter, value)
}
