// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// Package i18n provides managment api for application translations
// and support for looking up messages based on locale preferences.
package i18n

import (
	"context"
	"errors"

	"github.com/happy-sdk/happy/sdk/logging"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type contextKey string

const (
	langContextKey contextKey = "language"
)

var ErrDisabled = errors.New("i18n: disabled")

func Initialize(fallback language.Tag, logger logging.Logger) {
	initialize(fallback, logger)
}

func RegisterTranslationsFS(fs *FS) error {
	return registerTranslationsFS(fs)
}

func RegisterTranslations(lang language.Tag, translations map[string]any) error {
	return registerTranslations(lang, translations)
}

func RegisterTranslation(lang language.Tag, key string, value any) error {
	return registerTranslation(lang, "", key, value)
}

// SupportsLanguage checks if the given language is supported.
func SupportsLanguage(lang language.Tag) bool {
	return supportsLanguage(lang)
}

// QueueTranslation adds a single translation for a specific language to the language manager's queue.
// The translation is applied to the global catalog only after calling Reload.
// It returns an error if the registration fails.
//
// Parameters:
//   - lang: The language tag for the translation.
//   - key: The unique translation key.
//   - value: The translation value.
func QueueTranslation(lang language.Tag, key string, value any) error {
	return queueTranslation(lang, key, value)
}

// QueueTranslations adds multiple translations for a specific language to the language manager's queue.
// The translations are applied to the global catalog only after calling Reload.
// It returns an error if any registration fails.
//
// Parameters:
//   - lang: The language tag for the translations.
//   - translations: A map of translation keys to their values.
func QueueTranslations(lang language.Tag, translations map[string]any) error {
	return queueTranslations(lang, translations)
}

// SetLanguage sets the current language.
func SetLanguage(lang language.Tag) error {
	return setLanguage(lang)
}

// GetLanguage returns the current language.
func GetLanguage() language.Tag {
	return getLanguage()
}

// GetFallbackLanguage returns the fallback language.
func GetFallbackLanguage() language.Tag {
	return getFallbackLanguage()
}

// GetLanguages returns the supported languages.
func GetLanguages() []language.Tag {
	return getLanguages()
}

// ParseLanguage safely parses language string
func ParseLanguage(langStr string) language.Tag {
	return parseLanguage(langStr)
}

// WithLanguage adds language to context
func WithLanguage(ctx context.Context, lang language.Tag) context.Context {
	return context.WithValue(ctx, langContextKey, lang)
}

// GetPrinter returns a message printer for the given language
// or error if language is not supported with printer with default language.
func GetPrinter() (p *message.Printer) {
	return getPrinter()
}

// GetPrinterFor returns a message printer for the given language
// or error if language is not supported with printer with default language.
func GetPrinterFor(lang language.Tag) (p *message.Printer, err error) {
	return getPrinterFor(lang)
}

func GetFallbackPrinter() (p *message.Printer) {
	return getFallbackPrinter()
}

// Reload reloads the language manager to apply pending translations to the global catalog.
// It is only necessary when using QueueTranslation or QueueTranslations, as
// RegisterTranslation and RegisterTranslations automatically reload the manager.
func Reload() {
	reload()
}

// T is a convenience function for translation with default language,
// Fallback to English if key is not found on current language.
func T(key string, args ...any) string {
	return t(key, args...)
}

func GetPackagePrefix() string {
	return composePackageKey("", 2)
}
