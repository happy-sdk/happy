// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// Package i18n provides managment api for application translations
// and support for looking up messages based on locale preferences.
package i18n

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed i18n/*
var translations embed.FS

type contextKey string

const (
	langContextKey contextKey = "language"
)

var ErrDisabled = errors.New("i18n: disabled")
var ErrLanguageNotSupported = errors.New("i18n: language not supported")

// TranslationEntry represents a single translation entry with its key,
// fallback value, and translations for all supported languages.
type TranslationEntry struct {
	// Key is the translation key
	Key string
	// RootKey is the root translation key (package identifier) this entry belongs to.
	// For example, "com.github.happy-sdk.happy.sdk.cli" or "com.github.happy-sdk.happy.pkg.vars.varflag"
	RootKey string
	// Fallback is the translation value in the fallback language
	Fallback string
	// Translations is a map of language tags to their translation values.
	// If a language doesn't have a translation, it won't be present in the map.
	Translations map[language.Tag]string
}

// TranslationReport provides a report for a specific language's translation status.
type TranslationReport struct {
	// Language is the language tag this report is for
	Language language.Tag
	// Total is the total number of translation keys
	Total int
	// Translated is the number of keys that have translations for this language
	Translated int
	// Missing is the number of keys that are missing translations for this language
	Missing int
	// Percentage is the percentage of keys that are translated (0.0 to 100.0)
	Percentage float64
	// MissingEntries contains all translation entries that are missing translations for this language
	MissingEntries []TranslationEntry
	// RootKeys is the list of all root translation keys (package identifiers) found in translations
	RootKeys []string
	// PerRootKey is a map of root key to its translation statistics for this language
	PerRootKey map[string]RootKeyStats
}

// RootKeyStats provides translation statistics for a specific root key.
type RootKeyStats struct {
	// RootKey is the root translation key
	RootKey string
	// Total is the total number of translation keys for this root
	Total int
	// Translated is the number of keys that have translations for this language
	Translated int
	// Missing is the number of keys that are missing translations for this language
	Missing int
	// Percentage is the percentage of keys that are translated (0.0 to 100.0)
	Percentage float64
}

func Initialize(fallback language.Tag) {
	initialize(fallback)
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

func TD(key string, fallback string, args ...any) string {
	result := t(key, args...)
	// If translation not found (result equals key), use fallback
	if result == key {
		return fallback
	}
	return result
}

func PT(prefix, localKey string, args ...any) string {
	return t(fmt.Sprintf("%s.%s", prefix, localKey), args...)
}

func PTD(prefix, localKey, fallback string, args ...any) string {
	return t(fmt.Sprintf("%s.%s", prefix, localKey), args...)
}

func GetPackagePrefix() string {
	return composePackageKey("", 2)
}

// GetAllTranslations returns all registered translation entries.
// Each entry contains the translation key, the fallback value, and
// translations for all supported languages.
//
// This can be used to:
//   - List all available translation keys
//   - Compute translation status (percentage of how much each language is translated)
//   - Export translations for external tools
func GetAllTranslations() []TranslationEntry {
	return getAllTranslations()
}

// GetTranslationReport returns a translation report for the specified language.
// The report includes the translation percentage and a list of entries
// that are missing translations for that language.
func GetTranslationReport(lang language.Tag) TranslationReport {
	return getTranslationReport(lang)
}
