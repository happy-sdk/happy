// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

//go:build i18n

package i18n

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sync"

	"github.com/happy-sdk/happy/sdk/logging"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

const (
	loglevel = math.MinInt + 1
)

//go:embed lang/*
var translations embed.FS

var (
	initOnce      sync.Once
	globalCatalog *catalog.Builder
	mngr          *manager
)

func init() {
	mngr = newManager()
}

const Enabled = true

func initialize(fallback language.Tag, logger logging.Logger) {
	initOnce.Do(func() {
		if logger == nil {
			logopts := logging.ConsoleDefaultOptions()
			logopts.Level = logging.Level(loglevel)
			logger = logging.Console(logopts)
		}
		logger.LogDepth(0, logging.Level(loglevel), "loading i18n")
		globalCatalog = catalog.NewBuilder(
			catalog.Fallback(fallback),
		)

		mngr.logger = logger
		mngr.loggerDisposed = false
		mngr.initialized = true
		mngr.fallbackLang = fallback
		mngr.currentLang = fallback

		if err := RegisterTranslationsFS(NewFS(translations)); err != nil {
			logger.Error(err.Error())
		}
	})
}

func newManager() *manager {
	return &manager{
		langs: []language.Tag{
			language.English,
		},
		printerCache:   make(map[language.Tag]*message.Printer),
		fallbackLang:   language.English,
		currentLang:    language.English,
		loggerDisposed: true,
		dictionaries:   make(map[language.Tag]map[string]any),
	}
}

type manager struct {
	mu              sync.RWMutex
	langs           []language.Tag
	fallbackLang    language.Tag
	currentLang     language.Tag
	currentPrinter  *message.Printer
	fallbackPrinter *message.Printer
	printerCache    map[language.Tag]*message.Printer
	dictionaries    map[language.Tag]map[string]any
	queue           map[language.Tag]map[string]any
	logger          logging.Logger
	loggerDisposed  bool
	initialized     bool
}

func (m *manager) isInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

func (m *manager) supports(lang language.Tag) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Contains(m.langs, lang)
}

func (m *manager) support(lang language.Tag) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.langs = append(m.langs, lang)
}

func (m *manager) setCurrentLanguage(lang language.Tag) error {
	if !m.supports(lang) {
		return fmt.Errorf("i18n(%s): language not supported", lang.String())
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentLang = lang
	m.currentPrinter = message.NewPrinter(m.currentLang, message.Catalog(globalCatalog))
	return nil
}

func (m *manager) getCurrentLanguage() language.Tag {
	m.mu.Lock()
	defer m.mu.Unlock()
	lang := m.currentLang
	return lang
}

func (m *manager) getFallbackLanguage() language.Tag {
	m.mu.RLock()
	defer m.mu.RUnlock()
	lang := m.fallbackLang
	return lang
}

func (m *manager) usingFallbackLanguage() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentLang == m.fallbackLang
}

func (m *manager) log(level logging.Level, msg string) {
	if m.loggerDisposed {
		slog.Log(context.Background(), slog.Level(level), msg)
		return
	}

	m.logger.LogDepth(0, level, msg)

	ql, ok := m.logger.(*logging.QueueLogger)
	if ok && ql.Consumed() {
		m.logger = nil
		m.loggerDisposed = true
	}
}

func (m *manager) getSupported() []language.Tag {
	m.mu.RLock()
	defer m.mu.RUnlock()
	langs := m.langs
	return langs
}

func (m *manager) reload() {
	m.mu.RLock()
	if !m.initialized {
		m.mu.RUnlock()
		return
	}

	queue := m.queue
	fallbackLang := m.fallbackLang
	m.mu.RUnlock()
	if queue != nil {
		for lang, dict := range queue {
			if lang == language.Und {
				lang = fallbackLang
			}
			for key, value := range dict {
				if err := m.registerTranslation(lang, "", key, value); err != nil {
					m.mu.RLock()
					m.log(logging.LevelError, err.Error())
					m.mu.RUnlock()
				}
			}
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.queue = nil

	printerCache := make(map[language.Tag]*message.Printer)
	m.fallbackPrinter = message.NewPrinter(m.fallbackLang, message.Catalog(globalCatalog))
	printerCache[m.fallbackLang] = m.fallbackPrinter
	if m.fallbackLang != m.currentLang {
		m.currentPrinter = message.NewPrinter(m.currentLang, message.Catalog(globalCatalog))
		printerCache[m.currentLang] = m.currentPrinter
	} else {
		m.currentPrinter = m.fallbackPrinter
	}

	for _, lang := range m.langs {
		if lang == m.currentLang || lang == m.fallbackLang {
			continue
		}
		printerCache[lang] = message.NewPrinter(lang, message.Catalog(globalCatalog))
	}
}
func (m *manager) getPrinter() *message.Printer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	printer := m.currentPrinter
	return printer
}

func (m *manager) getFallbackPrinter() *message.Printer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	printer := m.fallbackPrinter
	return printer
}

func (m *manager) getPrinterFor(lang language.Tag) (*message.Printer, error) {
	if !m.supports(lang) {
		return nil, fmt.Errorf("i18n(%s): language not supported", lang.String())
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	if lang == m.currentLang {
		return m.currentPrinter, nil
	} else if lang == m.fallbackLang {
		return m.fallbackPrinter, nil
	}

	if printer, exists := m.printerCache[lang]; exists {
		return printer, nil
	}

	p := m.printerCache[lang]
	if p == nil {
		p = message.NewPrinter(lang, message.Catalog(globalCatalog))
		m.printerCache[lang] = p
	}
	return p, nil
}

func (m *manager) registerTranslation(lang language.Tag, prefix string, key string, value any) error {
	var shouldSupport bool

	if !m.isInitialized() {
		return m.queueTranslation(lang, key, value)
	}

	fullKey := key
	if prefix != "" {
		fullKey = prefix + "." + key
	}

	defer func() {
		if shouldSupport && !SupportsLanguage(lang) {
			m.support(lang)
		}
		if shouldSupport {
			m.addToDictionary(lang, fullKey, value)
		}
	}()
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
			if err := m.registerTranslation(lang, fullKey, key, value); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("i18n(%s): unsupported translation %s type %T", lang.String(), fullKey, value)
	}
	return nil
}

func (m *manager) queueTranslation(lang language.Tag, key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dictionary, ok := m.dictionaries[lang]
	if !ok {
		dictionary = make(map[string]any)
	}
	if _, ok := dictionary[key]; ok {
		return fmt.Errorf("%s: translation key %q already exists in dictionary", lang.String(), key)
	}
	if m.queue == nil {
		mngr.queue = make(map[language.Tag]map[string]any)
	}
	queuedict, ok := mngr.queue[lang]
	if !ok {
		queuedict = make(map[string]any)
		mngr.queue[lang] = queuedict
	}
	if _, ok := queuedict[key]; ok {
		return fmt.Errorf("%s: translation key %q already exists in queue", lang.String(), key)
	}
	queuedict[key] = value
	return nil
}

func (m *manager) addToDictionary(lang language.Tag, key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dictionary, ok := m.dictionaries[lang]
	if !ok {
		dictionary = make(map[string]any)
	}
	if _, ok := dictionary[key]; ok {
		return
	}
	dictionary[key] = value
	m.dictionaries[lang] = dictionary
}

func registerTranslations(lang language.Tag, translations map[string]any) error {
	defer reload()
	for key, value := range translations {
		if err := mngr.registerTranslation(lang, "", key, value); err != nil {
			return err
		}
	}
	return nil
}

func registerTranslation(lang language.Tag, prefix string, key string, value any) error {
	defer reload()
	return mngr.registerTranslation(lang, prefix, key, value)
}

func supportsLanguage(lang language.Tag) bool {
	return mngr.supports(lang)
}

func queueTranslation(lang language.Tag, key string, value any) error {
	return mngr.queueTranslation(lang, key, value)
}

func queueTranslations(lang language.Tag, translations map[string]any) error {
	for key, value := range translations {
		if err := mngr.queueTranslation(lang, key, value); err != nil {
			return err
		}
	}
	return nil
}

func setLanguage(lang language.Tag) error {
	if !SupportsLanguage(lang) {
		return fmt.Errorf("i18n(%s): language not supported", lang.String())
	}
	if err := mngr.setCurrentLanguage(lang); err != nil {
		return err
	}
	mngr.log(logging.LevelDebug, T("com.github.happy-sdk.happy.pkg.i18n.set_default", lang.String()))
	return nil
}

func getLanguage() language.Tag {
	lang := mngr.getCurrentLanguage()
	return lang
}

func getFallbackLanguage() language.Tag {
	lang := mngr.getFallbackLanguage()
	return lang
}

func getLanguages() []language.Tag {
	return globalCatalog.Languages()
}

func parseLanguage(langStr string) language.Tag {
	if langStr == "" {
		return mngr.getFallbackLanguage()
	}

	tag, err := language.Parse(langStr)
	if err != nil {
		return mngr.getFallbackLanguage()
	}

	supportedLangs := mngr.getSupported()

	// Find best match from supported languages
	matcher := language.NewMatcher(supportedLangs)
	_, index, _ := matcher.Match(tag)

	return supportedLangs[index]
}

func getPrinter() *message.Printer {
	return mngr.getPrinter()
}

func getPrinterFor(lang language.Tag) (p *message.Printer, err error) {
	return mngr.getPrinterFor(lang)
}

func getFallbackPrinter() (p *message.Printer) {
	return mngr.getFallbackPrinter()
}

func reload() {
	mngr.reload()
}

func t(key string, args ...any) string {
	printer := getPrinter()
	result := printer.Sprintf(key, args...)
	if result == key {
		if !mngr.usingFallbackLanguage() {
			printer = getFallbackPrinter()
			result = printer.Sprintf(key, args...)
		}
	}
	return result
}

func isInitialized() bool {
	return mngr.isInitialized()
}
