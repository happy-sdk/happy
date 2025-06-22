// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

//go:build !i18n

package i18n

import (
	"fmt"

	"github.com/happy-sdk/happy/sdk/logging"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func initialize(fallback language.Tag, logger logging.Logger) {}

func registerTranslations(lang language.Tag, translations map[string]any) error {
	return ErrDisabled
}

func registerTranslation(lang language.Tag, prefix string, key string, value any) error {
	return ErrDisabled
}

func supportsLanguage(lang language.Tag) bool {
	return false
}

func queueTranslation(lang language.Tag, key string, value any) error {
	return ErrDisabled
}

func queueTranslations(lang language.Tag, translations map[string]any) error {
	return ErrDisabled
}

func setLanguage(lang language.Tag) error {
	return ErrDisabled
}

func getLanguage() language.Tag {
	return language.Und
}

func getFallbackLanguage() language.Tag {
	return language.Und
}

func getLanguages() []language.Tag {
	return []language.Tag{language.Und}
}

func parseLanguage(langStr string) language.Tag {
	tag, err := language.Parse(langStr)
	if err != nil {
		return language.Und
	}
	return tag
}

func getPrinter() *message.Printer {
	return nil
}

func getPrinterFor(lang language.Tag) (p *message.Printer, err error) {
	return nil, ErrDisabled
}

func getFallbackPrinter() (p *message.Printer) {
	return nil
}

func registerTranslationsFS(fs *FS) error {
	return ErrDisabled
}

func reload() {}

func t(key string, args ...any) string {
	return fmt.Sprintf(key, args...)
}

func isInitialized() bool {
	return false
}
