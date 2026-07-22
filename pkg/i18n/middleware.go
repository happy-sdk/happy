// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"net/http"

	"golang.org/x/text/language"
)

// I18nMiddleware determines the request's language and adds it to the
// request context (see WithLanguage), for a downstream handler to read
// back with LanguageFromContext and render with TL:
//
//	lang, _ := i18n.LanguageFromContext(r.Context())
//	fmt.Fprint(w, i18n.TL(lang, "com.example.app.greeting"))
//
// The language is the first of, in order: the "lang" query parameter, the
// "Accept-Language" header's most preferred tag, the "language" cookie, or
// (if none of those is present) GetLanguage's process-wide current
// language. Whichever source is used, the value is resolved through
// ParseLanguage - so an unparseable or unsupported request never fails
// the request, it just falls through to the next source.
func I18nMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := extractLanguage(r)
		ctx := WithLanguage(r.Context(), lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractLanguage implements I18nMiddleware's source precedence - see its
// doc comment.
func extractLanguage(r *http.Request) language.Tag {
	if langParam := r.URL.Query().Get("lang"); langParam != "" {
		return ParseLanguage(langParam)
	}

	if acceptLang := r.Header.Get("Accept-Language"); acceptLang != "" {
		tags, _, _ := language.ParseAcceptLanguage(acceptLang)
		if len(tags) > 0 {
			return ParseLanguage(tags[0].String())
		}
	}

	if cookie, err := r.Cookie("language"); err == nil {
		return ParseLanguage(cookie.Value)
	}

	return getLanguage()
}
