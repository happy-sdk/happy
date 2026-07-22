// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/happy-sdk/happy/pkg/i18n"
	"golang.org/x/text/language"
)

// Example demonstrates the basic Initialize -> register -> render flow
// every application built on this package follows. A real package embeds
// its translations instead of registering them one at a time - see Embed.
func Example() {
	i18n.Initialize(language.English)

	_ = i18n.RegisterTranslation(language.English, "example.greeting", "Hello, {name}!")
	_ = i18n.RegisterTranslation(language.French, "example.greeting", "Bonjour, {name} !")

	_ = i18n.SetLanguage(language.English)
	fmt.Println(i18n.T("example.greeting", "name", "World"))
	fmt.Println(i18n.TL(language.French, "example.greeting", "name", "World"))

	// Output:
	// Hello, World!
	// Bonjour, World !
}

// ExampleTL demonstrates rendering in an explicit language rather than the
// process-wide current one, and the fallback cascade a language with no
// wording of its own falls through to: German has no translation
// registered for this key, so it renders in the global fallback language
// (English, set by Initialize) instead.
func ExampleTL() {
	i18n.Initialize(language.English)
	_ = i18n.RegisterTranslation(language.English, "example.farewell", "Goodbye, {name}!")
	_ = i18n.RegisterTranslation(language.French, "example.farewell", "Au revoir, {name} !")

	fmt.Println(i18n.TL(language.French, "example.farewell", "name", "World"))
	fmt.Println(i18n.TL(language.German, "example.farewell", "name", "World"))

	// Output:
	// Au revoir, World !
	// Goodbye, World!
}

// ExampleString demonstrates named arguments given as typed Arg values
// (String, Int, ...) instead of bare "key", value pairs - both forms are
// equivalent, and may be mixed freely in the same call.
func ExampleString() {
	i18n.Initialize(language.English)
	_ = i18n.RegisterTranslation(language.English, "example.profile", "{name} is {age:d} years old")

	fmt.Println(i18n.T("example.profile", i18n.String("name", "Ada"), i18n.Int("age", 36)))

	// Output:
	// Ada is 36 years old
}

// ExampleMessage_WithFragment demonstrates a *Message whose wording varies
// by number: WithFragment selects one of forms's CLDR cardinal plural
// categories (here just "one"/"other") by count's value.
func ExampleMessage_WithFragment() {
	i18n.Initialize(language.English)
	msg := i18n.NewMessage("{count_p}").WithFragment("count_p", "count", map[string]string{
		"one":   "{count:d} item",
		"other": "{count:d} items",
	})
	_ = i18n.RegisterTranslation(language.English, "example.items", msg)

	fmt.Println(i18n.T("example.items", "count", 1))
	fmt.Println(i18n.T("example.items", "count", 5))

	// Output:
	// 1 item
	// 5 items
}

// ExampleMessage_WithSelectFragment demonstrates a *Message whose wording
// varies by an arbitrary category rather than a plural rule - here,
// gender - matched exactly against the arg's value, with "other" as the
// required fallback for any value with no dedicated wording.
func ExampleMessage_WithSelectFragment() {
	i18n.Initialize(language.English)
	msg := i18n.NewMessage("{greeting}").WithSelectFragment("greeting", "gender", map[string]string{
		"male":   "He",
		"female": "She",
		"other":  "They",
	})
	_ = i18n.RegisterTranslation(language.English, "example.pronoun", msg)

	fmt.Println(i18n.T("example.pronoun", "gender", "male"))
	fmt.Println(i18n.T("example.pronoun", "gender", "female"))
	fmt.Println(i18n.T("example.pronoun", "gender", "unspecified"))

	// Output:
	// He
	// She
	// They
}

// ExampleQueueTranslation demonstrates deferred registration: a queued
// translation has no effect until the next Reload, unlike
// RegisterTranslation which applies immediately.
func ExampleQueueTranslation() {
	i18n.Initialize(language.English)
	_ = i18n.QueueTranslation(language.English, "example.queued", "queued value")
	fmt.Println(i18n.T("example.queued") == "example.queued") // not applied yet

	i18n.Reload()
	fmt.Println(i18n.T("example.queued"))

	// Output:
	// true
	// queued value
}

// ExampleNewError demonstrates a translatable error extended (see
// LocalizedError.Extends) to satisfy errors.Is against a package's own
// sentinel, so callers can detect the failure without depending on its
// exact rendered text.
func ExampleNewError() {
	i18n.Initialize(language.English)
	errNotFound := i18n.NewError("example_not_found", "example not found").Extends(i18n.Error)

	fmt.Println(errNotFound.Error())
	fmt.Println(errors.Is(errNotFound, i18n.Error))

	// Output:
	// example not found
	// true
}

// ExampleI18nMiddleware demonstrates a request-scoped language distinct
// from the process-wide current one: I18nMiddleware resolves it (here,
// from the "lang" query parameter) and stores it in the request context,
// for a handler to read back with LanguageFromContext and render with TL.
func ExampleI18nMiddleware() {
	i18n.Initialize(language.English)
	_ = i18n.RegisterTranslation(language.English, "example.middleware_greeting", "Hello!")
	_ = i18n.RegisterTranslation(language.French, "example.middleware_greeting", "Bonjour !")

	handler := i18n.I18nMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang, _ := i18n.LanguageFromContext(r.Context())
		_, _ = fmt.Fprint(w, i18n.TL(lang, "example.middleware_greeting"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/?lang=fr", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	fmt.Println(rec.Body.String())

	// Output:
	// Bonjour !
}
