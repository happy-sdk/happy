// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// Package i18n provides managment api for application translations
// and support for looking up messages based on locale preferences.
package i18n

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/happy-sdk/happy/sdk/logging"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

var (
	initOnce sync.Once

	globalCatalog *catalog.Builder
	mngr          *manager
)

//go:embed lang/*
var translations embed.FS

func Initialize(logger logging.Logger) {
	initOnce.Do(func() {
		if logger == nil {
			logopts := logging.ConsoleDefaultOptions()
			logopts.Level = logging.Level(loglevel)
			logger = logging.Console(logopts)
		}
		logger.LogDepth(0, logging.Level(loglevel), "loading i18n")
		globalCatalog = catalog.NewBuilder(
			catalog.Fallback(language.English),
		)

		mngr = &manager{
			langs: []language.Tag{
				language.English,
			},
			printerCache: make(map[language.Tag]*message.Printer),
			defaultLang:  language.English,
			logger:       logger,
		}

		if err := RegisterTranslationsFS(NewFS(translations)); err != nil {
			logger.Error(err.Error())
		}
	})
}

func RegisterTranslationsFS(fs *FS) error {
	langDirs, err := fs.readRoot()
	if err != nil {
		return fmt.Errorf("i18n loading translations from fs failed: %s", err.Error())
	}
	for _, dir := range langDirs {
		if dir.IsDir() {
			lang, err := language.Parse(dir.Name())
			if err != nil {
				return fmt.Errorf("i18n parsing language tag from dir %s failed: %s", dir.Name(), err.Error())
			}
			if err := fs.load(lang, filepath.Join(fs.prefix, dir.Name())); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("i18n expected language directory at root of fs got: %s", dir.Name())
		}
	}

	return nil
}

func RegisterTranslations(lang language.Tag, translations map[string]any) error {
	defer mngr.reload()
	for key, value := range translations {
		if err := registerTranslation(lang, "", key, value); err != nil {
			return err
		}
	}
	return nil
}

func RegisterTranslation(lang language.Tag, key string, value any) error {
	defer mngr.reload()
	return registerTranslation(lang, "", key, value)
}

// SupportsLanguage checks if the given language is supported.
func SupportsLanguage(lang language.Tag) bool {
	return mngr.supports(lang)
}

// SetLanguage sets the current language.
func SetLanguage(lang language.Tag) error {
	if !SupportsLanguage(lang) {
		return fmt.Errorf("i18n(%s): language not supported", lang.String())
	}
	if err := mngr.set(lang); err != nil {
		return err
	}
	mngr.log(logging.LevelDebug, T("language.set_default", lang.String()))
	return nil
}

func GetLanguage() language.Tag {
	lang := mngr.get()
	return lang
}

func GetLanguages() []language.Tag {
	return globalCatalog.Languages()
}

// ParseLanguage safely parses language string
func ParseLanguage(langStr string) language.Tag {
	if langStr == "" {
		return mngr.getDefault()
	}

	tag, err := language.Parse(langStr)
	if err != nil {
		return mngr.getDefault()
	}

	supportedLangs := mngr.getSupported()

	// Find best match from supported languages
	matcher := language.NewMatcher(supportedLangs)
	_, index, _ := matcher.Match(tag)

	return supportedLangs[index]
}

// WithLanguage adds language to context
func WithLanguage(ctx context.Context, lang language.Tag) context.Context {
	return context.WithValue(ctx, langContextKey, lang)
}

// GetPrinter returns a message printer for the given language
// or error if language is not supported with printer with default language.
func GetPrinter(lang language.Tag) (p *message.Printer, err error) {
	return mngr.getPrinter(lang)
}

func GetDefaultPrinter() (p *message.Printer) {
	Initialize(nil)
	return mngr.getDefaultPrinter()
}

// T is a convenience function for translation with default language,
// Fallback to English if key is not found on current language.
func T(key string, args ...any) string {
	printer := GetDefaultPrinter()
	result := printer.Sprintf(key, args...)
	if result == key {
		if GetLanguage() != language.English {
			printer, _ = GetPrinter(language.English)
			result = printer.Sprintf(key, args...)
		}
	}
	return result
}

func registerTranslation(lang language.Tag, prefix string, key string, value any) error {
	var shouldSupport bool

	defer func() {
		if shouldSupport && !SupportsLanguage(lang) {
			mngr.support(lang)
		}
	}()

	fullKey := key
	if prefix != "" {
		fullKey = prefix + "." + key
	}
	switch v := value.(type) {
	case string:
		shouldSupport = true
		return globalCatalog.SetString(lang, fullKey, v)
	case catalog.Message:
		shouldSupport = true
		return globalCatalog.Set(lang, fullKey, v)
	case []catalog.Message:
		shouldSupport = true
		return globalCatalog.Set(lang, fullKey, v...)
	case map[string]any:
		for key, value := range v {
			if err := registerTranslation(lang, fullKey, key, value); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("i18n(%s): unsupported translation %s type %T", lang.String(), fullKey, value)
	}
	return nil
}
