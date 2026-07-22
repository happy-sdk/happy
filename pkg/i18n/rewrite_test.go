// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/currency"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
)

// Plural-operand fidelity (section 6): a fractional count must select the
// correct CLDR category. Before the rewrite, toPluralOperand truncated to the
// integer part and passed v=w=f=t=0, so "1.5" wrongly selected "one" in
// English. It must now select "other".

func TestCldrCategory_FractionalEnglishCardinal(t *testing.T) {
	// Integer 1 -> "one"; fractional 1.5 -> "other" (visible fraction digits).
	testutils.Equal(t, "one", cldrCategory(plural.Cardinal, language.English, 1))
	testutils.Equal(t, "other", cldrCategory(plural.Cardinal, language.English, 1.5))
	testutils.Equal(t, "other", cldrCategory(plural.Cardinal, language.English, 2.5))
	// 1.0 is indistinguishable from int 1 at runtime, so it stays "one".
	testutils.Equal(t, "one", cldrCategory(plural.Cardinal, language.English, 1.0))
}

func TestPluralOperands_Fractions(t *testing.T) {
	i, v, w, f, tt, ok := pluralOperands(1.5)
	testutils.Assert(t, ok, "expected ok for a float")
	testutils.Equal(t, 1, i)
	testutils.Equal(t, 1, v)
	testutils.Equal(t, 1, w)
	testutils.Equal(t, 5, f)
	testutils.Equal(t, 5, tt)

	i, v, w, f, tt, ok = pluralOperands(2)
	testutils.Assert(t, ok, "expected ok for an int")
	testutils.Equal(t, 2, i)
	testutils.Equal(t, 0, v)
	testutils.Equal(t, 0, w)
	testutils.Equal(t, 0, f)
	testutils.Equal(t, 0, tt)
}

func TestRenderPlural_FractionalEndToEnd(t *testing.T) {
	Initialize(language.English)
	key := "test.rewrite.frac_items"
	_ = RegisterTranslation(language.English, key, map[string]any{
		messageKey: map[string]any{
			"msg": "{items_p}",
			"fragments": map[string]any{
				"items_p": map[string]any{
					"arg": "n", "one": "1 item", "other": "{n:number} items",
				},
			},
		},
	})
	testutils.Equal(t, "1 item", T(key, "n", 1))
	testutils.Equal(t, "1.5 items", T(key, "n", 1.5))
}

// Fallback chain (section 4): a real BCP 47 parent (en-GB -> en) must resolve,
// and a bundle-source fallback must resolve when the requested locale has no
// entry at all.

func TestFallback_ParentChain(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "test.rewrite.parent", "parent value")
	// en-GB is not itself registered; its BCP 47 parent en is.
	enGB := language.MustParse("en-GB")
	testutils.Equal(t, "parent value", TL(enGB, "test.rewrite.parent"))
}

func TestFallback_BundleSource(t *testing.T) {
	Initialize(language.English)
	// A v2 bundle whose source is German, with only a German entry. Asking for
	// French (supported via another registration) must fall back to the
	// bundle's declared source locale, not just the global fallback.
	_ = RegisterTranslation(language.French, "other.rewrite.anchor", "ancre")
	err := RegisterTranslations(language.Und, map[string]any{
		"version": float64(2),
		"bundle":  "test.rewrite.srcbundle",
		"source":  "de",
		"locales": map[string]any{
			"de": map[string]any{"keys": map[string]any{"only": "nur Deutsch"}},
		},
	})
	testutils.NoError(t, err)
	// French has no entry for this key; global fallback (English) has none
	// either; the bundle source (German) does.
	testutils.Equal(t, "nur Deutsch", TL(language.French, "test.rewrite.srcbundle.only"))
}

// Locale-aware number verb (section 7).

func TestNumberVerb_LocaleGrouping(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "test.rewrite.num_en", "total: {n:number}")
	_ = RegisterTranslation(language.German, "test.rewrite.num_de", "total: {n:number}")
	testutils.Equal(t, "total: 1,234,567", T("test.rewrite.num_en", "n", 1234567))
	// German groups with '.' and uses ',' as decimal separator.
	testutils.Equal(t, "total: 1.234.567", TL(language.German, "test.rewrite.num_de", "n", 1234567))
	testutils.Equal(t, "total: 1,234.5", T("test.rewrite.num_en", "n", 1234.5))
}

func TestNumberVerb_ArgType(t *testing.T) {
	msg := NewMessage("{n:number} things")
	testutils.Equal(t, ArgTypeNumber, msg.ArgTypes()["n"])
	testutils.Equal(t, ArgTypeNumber, argTypeForVerb("number"))
}

// Locale-aware currency verb (section 7). The calling convention is a
// currency.Amount value (constructed by the caller, e.g. via i18n.Any), since a
// single placeholder can't carry both an amount and a currency code.

func TestCurrencyVerb_LocaleFormatting(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "test.rewrite.price", "price: {p:currency}")
	amt := currency.USD.Amount(1234.5)
	out := T("test.rewrite.price", Any("p", amt))
	// x/text renders the ISO code + grouped amount, e.g. "USD 1,234.50".
	testutils.Assert(t, out != "price: {p:currency}" && out != "price: %!currency", "expected a formatted currency, got %q", out)
	testutils.Assert(t, len(out) > len("price: "), "expected non-empty currency rendering, got %q", out)
}

func TestCurrencyVerb_ArgType(t *testing.T) {
	testutils.Equal(t, ArgTypeCurrency, argTypeForVerb("currency"))
}

// TestCurrencyVerb_TypeMismatchIsVisible is a regression test for a bug where
// a {p:currency} placeholder given anything other than a currency.Amount (e.g.
// a plain float64 or string - an easy mistake, since the dynamic i18n.T API
// gives no compile-time check) silently rendered via the final fallback
// fmt.Appendf(dst, "%v", val) - a bare number with no currency symbol, no
// grouping, and no "%!" marker at all, indistinguishable from a real (if
// oddly-formatted) price. Every other verb mismatch in this package surfaces
// visibly (e.g. a missing arg renders "%!name(MISSING)"); currency was the
// one exception. Now it must render fmt's own bad-argument convention
// instead (%!currency(type=value), matching e.g. %!d(string=x)).
func TestCurrencyVerb_TypeMismatchIsVisible(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "test.rewrite.price_mismatch", "price: {p:currency}")
	out := T("test.rewrite.price_mismatch", Any("p", 1234.5))
	testutils.Assert(t, strings.Contains(out, "%!currency"), "expected a visible %%!currency marker for a non-currency.Amount value, got %q", out)
}

// Bidi isolation opt-in (section 7). Off by default; when enabled for an RTL
// locale it wraps interpolated segments in FSI/PDI marks.

func TestBidiIsolation_OffByDefault(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.Arabic, "test.rewrite.bidi", "مرحبا {name:s}")
	// Without the option, no isolation marks are inserted.
	out := TL(language.Arabic, "test.rewrite.bidi", "name", "Ada")
	testutils.Assert(t, out == "مرحبا Ada", "expected no isolation marks by default, got %q", out)
}

func TestBidiIsolation_WrapsInterpolatedForRTL(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.Arabic, "test.rewrite.bidi2", "مرحبا {name:s}")
	out := TL(language.Arabic, "test.rewrite.bidi2", "name", "Ada", WithBidiIsolation())
	testutils.Assert(t, out == "مرحبا "+fsi+"Ada"+pdi, "expected FSI/PDI around the interpolated arg, got %q", out)
}

func TestBidiIsolation_NoOpForLTR(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "test.rewrite.bidi_ltr", "hello {name:s}")
	// The option is a no-op for a left-to-right locale.
	out := T("test.rewrite.bidi_ltr", "name", "Ada", WithBidiIsolation())
	testutils.Equal(t, "hello Ada", out)
}

func TestIsRTL(t *testing.T) {
	testutils.Assert(t, isRTL(language.Arabic), "expected Arabic to be RTL")
	testutils.Assert(t, isRTL(language.Hebrew), "expected Hebrew to be RTL")
	testutils.Assert(t, !isRTL(language.English), "expected English not to be RTL")
	testutils.Assert(t, !isRTL(language.German), "expected German not to be RTL")
}

// Legacy positional %-verb templates must still render (now via the compiled
// legacy program), and plain strings must still round-trip byte-for-byte.

func TestLegacyPositional_StillWorksAfterRewrite(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.English, "test.rewrite.legacy", "legacy %s and %d")
	testutils.Equal(t, "legacy value and 5", T("test.rewrite.legacy", "value", 5))
}
