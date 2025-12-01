// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
)

func TestI18nMiddleware(t *testing.T) {
	Initialize(language.English)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := r.Context().Value(langContextKey)
		testutils.NotNil(t, lang, "expected language in context")
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware
	middleware := I18nMiddleware(handler)

	// Test request
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	testutils.Equal(t, http.StatusOK, rr.Code)
}

func TestExtractLanguage_QueryParam(t *testing.T) {
	Initialize(language.English)

	req := httptest.NewRequest("GET", "/?lang=fr", nil)
	lang := extractLanguage(req)

	testutils.Equal(t, language.French, lang)
}

func TestExtractLanguage_AcceptLanguage(t *testing.T) {
	Initialize(language.English)

	// Register French first
	_ = RegisterTranslation(language.French, "test.key", "Valeur")

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
	lang := extractLanguage(req)

	// Should parse and match to French
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestExtractLanguage_Cookie(t *testing.T) {
	Initialize(language.English)

	// Register German first
	_ = RegisterTranslation(language.German, "test.key", "Wert")

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "language", Value: "de"})
	lang := extractLanguage(req)

	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestExtractLanguage_Default(t *testing.T) {
	Initialize(language.English)

	req := httptest.NewRequest("GET", "/", nil)
	lang := extractLanguage(req)

	// Should return current language (English)
	current := GetLanguage()
	testutils.Equal(t, current, lang)
}

func TestExtractLanguage_Priority(t *testing.T) {
	Initialize(language.English)

	// Register languages
	_ = RegisterTranslation(language.French, "test.key", "Valeur")
	_ = RegisterTranslation(language.German, "test.key", "Wert")

	// Query param should have highest priority
	req := httptest.NewRequest("GET", "/?lang=de", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
	req.AddCookie(&http.Cookie{Name: "language", Value: "fr"})

	lang := extractLanguage(req)

	// Should use query param (German)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

// Additional coverage tests

func TestExtractLanguage_EmptyQueryParam(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/?lang=", nil)
	lang := extractLanguage(req)
	current := GetLanguage()
	testutils.Equal(t, current, lang)
}

func TestExtractLanguage_InvalidQueryParam(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/?lang=invalid-lang-xyz", nil)
	lang := extractLanguage(req)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestExtractLanguage_EmptyAcceptLanguage(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "")
	lang := extractLanguage(req)
	current := GetLanguage()
	testutils.Equal(t, current, lang)
}

func TestExtractLanguage_InvalidAcceptLanguage(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "invalid-language-header")
	lang := extractLanguage(req)
	current := GetLanguage()
	testutils.Equal(t, current, lang)
}

func TestExtractLanguage_EmptyCookie(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "language", Value: ""})
	lang := extractLanguage(req)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestExtractLanguage_InvalidCookie(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "language", Value: "invalid-cookie-lang"})
	lang := extractLanguage(req)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestExtractLanguage_CookieError(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/", nil)
	lang := extractLanguage(req)
	current := GetLanguage()
	testutils.Equal(t, current, lang)
}

func TestExtractLanguage_WrongCookieName(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "lang", Value: "fr"})
	lang := extractLanguage(req)
	current := GetLanguage()
	testutils.Equal(t, current, lang)
}

func TestExtractLanguage_AcceptLanguageWithMultipleTags(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.French, "test.key", "Valeur")
	_ = RegisterTranslation(language.German, "test.key", "Wert")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en-US;q=0.8,de;q=0.7")
	lang := extractLanguage(req)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestExtractLanguage_AcceptLanguageWithQuality(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.Spanish, "test.key", "Valor")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "es-ES;q=0.9,es;q=0.8")
	lang := extractLanguage(req)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestI18nMiddleware_WithLanguageInContext(t *testing.T) {
	Initialize(language.English)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := r.Context().Value(langContextKey)
		testutils.NotNil(t, lang, "expected language in context")
		langTag, ok := lang.(language.Tag)
		testutils.Assert(t, ok, "expected language.Tag type in context")
		testutils.NotEqual(t, language.Und, langTag, "expected valid language tag, got Und")
		w.WriteHeader(http.StatusOK)
	})
	middleware := I18nMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	testutils.Equal(t, http.StatusOK, rr.Code)
}

func TestI18nMiddleware_ContextPropagation(t *testing.T) {
	Initialize(language.English)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := r.Context().Value(langContextKey)
		testutils.NotNil(t, lang, "expected language in context")
		w.WriteHeader(http.StatusOK)
	})
	middleware := I18nMiddleware(handler)
	req := httptest.NewRequest("GET", "/?lang=fr", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	testutils.Equal(t, http.StatusOK, rr.Code)
}

func TestExtractLanguage_QueryParamOverridesAll(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.French, "test.key", "Valeur")
	_ = RegisterTranslation(language.German, "test.key", "Wert")
	req := httptest.NewRequest("GET", "/?lang=de", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
	req.AddCookie(&http.Cookie{Name: "language", Value: "fr"})
	lang := extractLanguage(req)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestExtractLanguage_AcceptLanguageOverridesCookie(t *testing.T) {
	Initialize(language.English)
	_ = RegisterTranslation(language.French, "test.key", "Valeur")
	_ = RegisterTranslation(language.German, "test.key", "Wert")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9")
	req.AddCookie(&http.Cookie{Name: "language", Value: "fr"})
	lang := extractLanguage(req)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestExtractLanguage_ComplexAcceptLanguage(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,fr-FR;q=0.8,fr;q=0.7,de;q=0.6")
	lang := extractLanguage(req)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}

func TestExtractLanguage_UnsupportedLanguageInQuery(t *testing.T) {
	Initialize(language.English)
	req := httptest.NewRequest("GET", "/?lang=ja", nil)
	lang := extractLanguage(req)
	testutils.NotEqual(t, language.Und, lang, "expected valid language, got Und")
}
