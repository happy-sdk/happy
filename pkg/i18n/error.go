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

// LocalizedError is a translatable error: its message renders through T
// (see Error) using a key composed from the caller's own package (see
// composePackageKey), falling back to a caller-supplied plain string
// wherever no translation applies - either because i18n hasn't been
// Initialized yet, or because no translation was ever registered for this
// specific key. Construct one with NewError, NewErrorWithLocale, or
// NewErrorDepth; WithArgs, WithCode, and Extends each return a modified
// copy, so calling them on a shared package-level sentinel is safe to do
// from multiple call sites without racing or clobbering one another.
type LocalizedError struct {
	code     int
	key      string
	fallback string
	tag      language.Tag
	args     []any
	wraps    error
}

// NewError creates a LocalizedError keyed under the calling package's own
// reverse-DNS prefix (see GetPackagePrefix) plus key - e.g. key "disabled"
// called from package i18n itself becomes
// "com.github.happy-sdk.happy.pkg.i18n.disabled". If non-empty, fallback
// is queued (see QueueTranslation) as language.Und's translation for that
// key, so Error() has something sensible to render even before Initialize
// registers a real translation for it; it also becomes the text Error()
// falls back to before Initialize runs at all, or if no translation for
// this key ever gets registered. Use NewErrorWithLocale to key the error's
// own rendering to a specific language instead of the current one.
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

// NewErrorWithLocale is NewError for a LocalizedError that always renders
// in tag's language via TL, regardless of the process-wide current
// language (see SetLanguage) at the time Error() is called.
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

// NewErrorDepth is NewError for a caller that isn't itself the package the
// error should be keyed under - e.g. a helper called from several places
// that all want the error keyed under their own package, not the helper's.
// depth is a runtime.Caller depth counted from NewErrorDepth's own call
// frame: 1 names NewErrorDepth's direct caller (equivalent to NewError's
// fixed depth), 2 names that caller's own caller, and so on.
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

// WithCode returns a copy of e carrying code, so calling WithCode on a
// shared package-level sentinel (e.g. i18n.ErrLanguageNotSupported) never
// mutates that sentinel for every other caller.
func (e *LocalizedError) WithCode(code int) *LocalizedError {
	c := e.clone()
	c.code = code
	return c
}

// Translate registers msg as e's own translation for tag (see
// RegisterTranslation), then returns e itself for chaining - e.g.
// NewError("disabled", "disabled").Translate(language.French, "desactive").
// A registration failure is intentionally not surfaced here (there's no
// error return to give it): call RegisterTranslation directly instead if
// that failure needs to be checked. A blank msg is a no-op, so chaining
// Translate calls for every supported language doesn't require guarding
// each one against a locale nobody has a wording for yet.
func (e *LocalizedError) Translate(tag language.Tag, msg string) *LocalizedError {
	if msg != "" {
		_ = RegisterTranslation(tag, e.key, msg)
	}
	return e
}

// WithArgs returns a copy of e carrying args, for the same reason WithCode
// does - it's routine to call WithArgs on a shared sentinel (e.g.
// i18n.ErrLanguageNotSupported.WithArgs(lang.String())) and that must not
// race with or clobber every other caller doing the same.
func (e *LocalizedError) WithArgs(args ...any) *LocalizedError {
	c := e.clone()
	c.args = args
	return c
}

// Extends returns a copy of e that satisfies errors.Is(err, target), so a
// package can define its own sentinel LocalizedErrors that still identify
// as belonging to a broader category - e.g. every error this package
// returns extends its own package-level Error.
func (e *LocalizedError) Extends(target error) *LocalizedError {
	c := e.clone()
	c.wraps = target
	return c
}

// Unwrap lets errors.Is/errors.As reach whatever Extends was called with.
func (e *LocalizedError) Unwrap() error {
	return e.wraps
}

// Is reports whether target is a *LocalizedError sharing e's translation
// key, so a copy produced by WithArgs/WithCode - same underlying error,
// different call-specific details - still satisfies
// errors.Is(copy, originalSentinel).
func (e *LocalizedError) Is(target error) bool {
	t, ok := target.(*LocalizedError)
	if !ok || e.key == "" {
		return false
	}
	return e.key == t.key
}

func (e *LocalizedError) clone() *LocalizedError {
	c := *e
	return &c
}

// Error renders e's message: via TL if built with NewErrorWithLocale (tag
// pinned to a specific language), otherwise via T (the process-wide
// current language, see SetLanguage), with e.args interpolated either way.
// Before Initialize has run, or if this key has no registered translation
// anywhere, it renders e.fallback instead (formatted the same way - see
// formatFallback), or e.key itself if fallback is also empty. A non-zero
// code (see WithCode) is prefixed as "code: message".
func (e *LocalizedError) Error() string {
	if !isInitialized() {
		msg := e.key
		if e.fallback != "" {
			msg = e.fallback
		}
		return formatFallback(msg, e.args)
	}
	var result string
	if e.tag != language.Und {
		result = TL(e.tag, e.key, e.args...)
	} else {
		result = T(e.key, e.args...)
	}
	if result == e.key {
		if e.fallback != "" {
			return formatFallback(e.fallback, e.args)
		}
		return e.key
	}
	if e.code == 0 {
		return result
	}
	return fmt.Sprintf("%d: %s", e.code, result)
}

// formatFallback renders msg with args when no translation applies (either
// i18n isn't initialized yet, or this specific key has no registered
// translation) - using the {name}/{name:verb} engine if msg uses that
// syntax, or plain fmt.Sprintf otherwise, matching whichever style the
// caller's fallback text was written in.
func formatFallback(msg string, args []any) string {
	if hasNamedPlaceholders(msg) {
		return renderNamed(msg, args)
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

// processFunctionName strips fnName's trailing function/method name, and -
// if what's left inside "init" is itself only digits (as runtime names the
// N'th init function in a file, e.g. "pkg.init.0") - strips that generated
// index too, leaving just the package's own import path. Called from
// composePackageKey, which needs a caller's package path, not the name of
// whatever function or init() block happened to be on the stack at depth.
func processFunctionName(fnName string) string {
	lastDotIndex := strings.LastIndex(fnName, ".")
	if lastDotIndex != -1 {
		fnNameNew := fnName[:lastDotIndex]
		removed := fnName[lastDotIndex+1:]
		fnName = fnNameNew
		if strings.IndexFunc(removed, func(c rune) bool { return c < '0' || c > '9' }) == -1 {
			lastDotIndex := strings.LastIndex(fnName, ".")
			if lastDotIndex != -1 {
				fnName = fnName[:lastDotIndex]
			}
		}
	}
	return fnName
}

// composePackageKey builds a full translation key by prefixing key with
// the reverse-DNS form (see reverseDns) of the package found at depth on
// the call stack (a runtime.Caller depth, counted from composePackageKey's
// own frame) - e.g. NewError, GetPackagePrefix. An empty key returns just
// the reverse-DNS prefix on its own.
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

// reverseDns turns a Go import path (e.g.
// "github.com/happy-sdk/happy/pkg/i18n") into its reverse-DNS form (e.g.
// "com.github.happy-sdk.happy.pkg.i18n") - the convention every key this
// package's own sentinels, and NewError/NewErrorDepth/GetPackagePrefix
// callers, are rooted under.
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
	rdns := strings.Join(rev, ".")
	return rdns
}

// alnum is the ASCII alphanumeric range ensure keeps; anything else
// (except dot) is dropped.
var alnum = &unicode.RangeTable{ //nolint:gochecknoglobals
	R16: []unicode.Range16{
		{'0', '9', 1},
		{'A', 'Z', 1},
		{'a', 'z', 1},
	},
}

const dot = '.'

// ensure lowercases in and strips everything but ASCII alphanumerics and
// "." - used by reverseDns to clean up a domain apex (e.g. "GitHub.COM")
// before reversing it into a translation key segment. A bare "-" is passed
// through unchanged rather than reduced to "": every other character
// ensure strips would otherwise collapse it to an empty segment, which
// would join two dots together in the caller's result.
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
