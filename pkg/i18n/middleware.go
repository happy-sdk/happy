// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"net/http"

	"golang.org/x/text/language"
)

// I18nMiddleware extracts language preference and adds it to request context
func I18nMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := extractLanguage(r)
		ctx := WithLanguage(r.Context(), lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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
