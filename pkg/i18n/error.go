// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"bytes"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/language"
)

type LocalizedError struct {
	code     int
	key      string
	fallback string
	tag      language.Tag
	args     []any
}

// NewError creates a new LocalizedError with the default language.English locale.
// It uses the provided key to form a unique translation key by prefixing it with the
// reverse module DNS name (e.g., "Error" becomes "com.github.happy-sdk.happy.pkg.i18n.Error").
// If translations are not loaded via Initialize, the defaultMsg is returned as the error message.
// Otherwise, defaultMsg is applied as the fallback for the default locale.
// For specific locales, use NewErrorWithLocale instead.
//
// Parameters:
//   - key: The local error key (e.g., "Error") to form a unique translation key.
//   - fallback: The fallback error message if no translation is found.
func NewError(key, fallback string) *LocalizedError {
	fullKey := composePackageKey(key, 2)
	if fallback != "" {
		_ = QueueTranslation(language.Und, fullKey, fallback)
	}
	return &LocalizedError{
		key:      fullKey,
		fallback: fallback,
		tag:      language.Und,
	}
}

func NewErrorWithLocale(tag language.Tag, key, fallback string) *LocalizedError {
	fullKey := composePackageKey(key, 2)
	if fallback != "" {
		_ = QueueTranslation(language.Und, fullKey, fallback)
	}
	return &LocalizedError{
		key:      fullKey,
		fallback: fallback,
		tag:      tag,
	}
}

// NewErrorDepth creates a new LocalizedError with the default
// locale and a specified depth to compose full key from package import path.
func NewErrorDepth(depth int, key, fallback string) *LocalizedError {
	fullKey := composePackageKey(key, depth)
	if fallback != "" {
		_ = QueueTranslation(language.Und, fullKey, fallback)
	}
	return &LocalizedError{
		key:      fullKey,
		fallback: fallback,
		tag:      language.Und,
	}
}

func (e *LocalizedError) WithCode(code int) *LocalizedError {
	e.code = code
	return e
}

func (e *LocalizedError) Translate(tag language.Tag, msg string) *LocalizedError {
	if msg != "" {
		_ = RegisterTranslation(tag, e.key, msg)
	}
	return e
}

func (e *LocalizedError) WithArgs(args ...any) *LocalizedError {
	e.args = args
	return e
}

func (e *LocalizedError) Error() string {
	if !isInitialized() {
		msg := e.key
		if e.fallback != "" {
			msg = e.fallback
		}
		if len(e.args) > 0 {
			return fmt.Sprintf(msg, e.args...)
		}
		return msg
	}
	result := T(e.key, e.args...)
	if result == e.key {
		if e.fallback != "" {
			return e.fallback
		}
		return e.key
	}
	if e.code == 0 {
		return result
	}
	return fmt.Sprintf("%d: %s", e.code, result)
}

// processFunctionName processes a function name to remove init function indices.
// This is extracted for testability.
func processFunctionName(fnName string) string {
	lastDotIndex := strings.LastIndex(fnName, ".")
	if lastDotIndex != -1 {
		fnNameNew := fnName[:lastDotIndex]
		removed := fnName[lastDotIndex+1:]
		fnName = fnNameNew
		// check if we removed pkg init function index
		if strings.IndexFunc(removed, func(c rune) bool { return c < '0' || c > '9' }) == -1 {
			lastDotIndex := strings.LastIndex(fnName, ".")
			if lastDotIndex != -1 {
				fnName = fnName[:lastDotIndex]
			}
		}
	}
	return fnName
}

func composePackageKey(key string, depth int) string {
	pc, _, _, ok := runtime.Caller(depth)
	if !ok {
		return key
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return key
	}
	fnName := fn.Name()
	fnName = processFunctionName(fnName)
	fnName = reverseDns(fnName)
	if key == "" {
		return fnName
	}
	return fmt.Sprintf("%s.%s", fnName, key)
}

func reverseDns(u string) string {
	var rev []string
	var rmdomain bool
	sl := strings.Split(u, "/")

	if strings.Contains(sl[0], ".") {
		rmdomain = true
		domainparts := strings.Split(sl[0], ".")
		slices.Reverse(domainparts)
		rev = append(rev, ensure(strings.Join(domainparts, ".")))
	}

	p := len(sl)
	for i := range p {
		if rmdomain && i == 0 {
			continue
		}
		rev = append(rev, (sl[i]))
	}
	// slices.Reverse(rev)
	rdns := strings.Join(rev, ".")
	return rdns
}

var alnum = &unicode.RangeTable{ //nolint:gochecknoglobals
	R16: []unicode.Range16{
		{'0', '9', 1},
		{'A', 'Z', 1},
		{'a', 'z', 1},
	},
}

const dot = '.'

func ensure(in string) string {
	if in == "-" {
		return in
	}

	var b bytes.Buffer
	for _, c := range in {
		isAlnum := unicode.Is(alnum, c)
		isSpace := unicode.IsSpace(c)
		isLower := unicode.IsLower(c)
		if isSpace || (!isAlnum && c != dot) {
			continue
		}
		if !isLower {
			c = unicode.ToLower(c)
		}
		b.WriteRune(c)
	}
	return b.String()
}
