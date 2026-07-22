// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
)

func TestHasNamedPlaceholders(t *testing.T) {
	testutils.Assert(t, hasNamedPlaceholders("hello {name}"), "expected {name} to be detected")
	testutils.Assert(t, hasNamedPlaceholders("count: {count:d}"), "expected {count:d} to be detected")
	testutils.Assert(t, hasNamedPlaceholders("amount: {amount:f.2}"), "expected {amount:f.2} to be detected")
	testutils.Assert(t, !hasNamedPlaceholders("hello %s"), "expected plain %%s template not to be detected as named")
	testutils.Assert(t, !hasNamedPlaceholders("just a plain string"), "expected plain string not to be detected as named")
	testutils.Assert(t, !hasNamedPlaceholders("literal braces {} without a name"), "expected an empty {} not to be detected as named")
}

func TestRenderNamed_DefaultVerb(t *testing.T) {
	out := renderNamed("hello {name}", []any{"name", "World"})
	testutils.Equal(t, "hello World", out)
}

func TestRenderNamed_ExplicitVerbs(t *testing.T) {
	out := renderNamed("{s:s} {d:d} {q:q} {t:T}", []any{
		"s", "str",
		"d", 42,
		"q", "quoted",
		"t", 3.14,
	})
	testutils.Equal(t, `str 42 "quoted" float64`, out)
}

func TestRenderNamed_PrecisionVerb(t *testing.T) {
	out := renderNamed("total: {amount:f.2}", []any{"amount", 3.14159})
	testutils.Equal(t, "total: 3.14", out)
}

func TestRenderNamed_TypedArgConstructors(t *testing.T) {
	out := renderNamed("{name:s} has {count:d} messages", []any{
		String("name", "Alice"),
		Int("count", 3),
	})
	testutils.Equal(t, "Alice has 3 messages", out)
}

func TestRenderNamed_MixedBareAndTypedArgs(t *testing.T) {
	out := renderNamed("{name:s} has {count:d} messages", []any{
		"name", "Bob",
		Int("count", 5),
	})
	testutils.Equal(t, "Bob has 5 messages", out)
}

func TestRenderNamed_MissingArg(t *testing.T) {
	out := renderNamed("hello {name}", nil)
	testutils.Equal(t, "hello %!name(MISSING)", out)
}

func TestRenderNamed_TypeMismatchDoesNotPanic(t *testing.T) {
	// {count:d} but the value is a string - fmt's own graceful mismatch
	// marker should appear, not a panic.
	out := renderNamed("count: {count:d}", []any{"count", "not-a-number"})
	testutils.Assert(t, strings.Contains(out, "%!d(string="), "expected fmt-style mismatch marker, got %q", out)
}

func TestArgsToMap_TrailingKeyNoValue(t *testing.T) {
	// A caller bug - "key" with nothing after it - must not panic.
	m := argsToMap([]any{"onlykey"})
	testutils.Equal(t, 1, len(m))
}

func TestArgsToMap_ValueWhereKeyExpected(t *testing.T) {
	// A caller bug - a bare non-string value where a key was expected -
	// must not panic, and must still be visible in the map somehow.
	m := argsToMap([]any{42})
	testutils.Equal(t, 1, len(m))
}

func TestArgsToMap_EmptyArgs(t *testing.T) {
	m := argsToMap(nil)
	testutils.Equal(t, 0, len(m))
}

func TestFormatNamed_NoVerbUsesV(t *testing.T) {
	testutils.Equal(t, "42", formatNamed("", 42))
}

func TestFormatNamed_PrecisionOnlyVerbDefaultsToV(t *testing.T) {
	// A verb spec that's only a precision (no letter before the dot) can't
	// come from namedPlaceholderRe itself (it requires a leading letter),
	// but formatNamed is defensive about it anyway: falls back to %v
	// rather than building an invalid "%.2" format string.
	testutils.Equal(t, "3.1", formatNamed(".2", 3.14159))
}

func TestArgConstructors(t *testing.T) {
	testutils.Equal(t, Arg{Key: "k", Value: "v"}, String("k", "v"))
	testutils.Equal(t, Arg{Key: "k", Value: 1}, Int("k", 1))
	testutils.Equal(t, Arg{Key: "k", Value: int64(1)}, Int64("k", 1))
	testutils.Equal(t, Arg{Key: "k", Value: uint64(1)}, Uint64("k", 1))
	testutils.Equal(t, Arg{Key: "k", Value: 1.5}, Float64("k", 1.5))
	testutils.Equal(t, Arg{Key: "k", Value: true}, Bool("k", true))
	testutils.Equal(t, Arg{Key: "k", Value: "anything"}, Any("k", "anything"))
}

// End-to-end: a real registered {name:verb} translation, resolved through
// T/TL - not just the isolated renderNamed helper - proving the manager
// actually detects and routes named templates, while leaving legacy
// %-verb templates on the existing positional path untouched.

func TestT_NamedTemplate_EndToEnd(t *testing.T) {
	Initialize(language.English)
	key := "test.named.greeting"
	_ = RegisterTranslation(language.English, key, "hello {name:s}, you have {count:d} new messages")

	out := T(key, "name", "Ada", "count", 7)
	testutils.Equal(t, "hello Ada, you have 7 new messages", out)
}

func TestTL_NamedTemplate_EndToEnd(t *testing.T) {
	Initialize(language.English)
	key := "test.named.tl_greeting"
	_ = RegisterTranslation(language.English, key, "hi {name:s}")

	out := TL(language.English, key, String("name", "Grace"))
	testutils.Equal(t, "hi Grace", out)
}

func TestT_LegacyPositionalTemplate_StillWorks(t *testing.T) {
	// Regression guard: a plain %-verb template must keep using the
	// existing positional golang.org/x/text/message path, unaffected by
	// the {name:verb} engine coexisting alongside it.
	Initialize(language.English)
	key := "test.named.legacy"
	_ = RegisterTranslation(language.English, key, "legacy %s and %d")

	out := T(key, "value", 5)
	testutils.Equal(t, "legacy value and 5", out)
}
