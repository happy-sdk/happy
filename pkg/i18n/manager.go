// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

var (
	initOnce      sync.Once
	globalCatalog *catalog.Builder
	mngr          *manager
)

func init() {
	mngr = newManager()
}

const Enabled = true

func initialize(fallback language.Tag) {
	initOnce.Do(func() {
		globalCatalog = catalog.NewBuilder(
			catalog.Fallback(fallback),
		)

		mngr.initialized = true
		mngr.fallbackLang = fallback
		mngr.currentLang = fallback

		if err := RegisterTranslationsFS(NewFS(translations)); err != nil {
			slog.Error(err.Error())
		}

		// Process any queued translations that were registered before initialization
		reload()
	})
}

func newManager() *manager {
	return &manager{
		langs: []language.Tag{
			language.English,
		},
		printerCache: make(map[language.Tag]*message.Printer),
		fallbackLang: language.English,
		currentLang:  language.English,
		dictionaries: make(map[language.Tag]map[string]any),
		rootKeys:     make(map[string]bool),
		keyToRoot:    make(map[string]string),
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
	initialized     bool
	// rootKeys tracks all root translation keys (package identifiers)
	// e.g., "com.github.happy-sdk.happy.sdk.cli"
	rootKeys map[string]bool
	// keyToRoot maps full translation keys to their root keys
	keyToRoot map[string]string
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
	for lang, dict := range queue {
		if lang == language.Und {
			lang = fallbackLang
		}
		for key, value := range dict {
			if err := m.registerTranslation(lang, "", key, value); err != nil {
				slog.Error(err.Error())
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
	switch lang {
	case m.currentLang:
		return m.currentPrinter, nil
	case m.fallbackLang:
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

	// Track root key for this translation key
	rootKey := m.extractRootKey(fullKey)
	if rootKey != "" {
		m.mu.Lock()
		m.rootKeys[rootKey] = true
		m.keyToRoot[fullKey] = rootKey
		m.mu.Unlock()
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

// mergeTranslationValues merges newValue into existingValue.
// If both are maps, they are merged recursively.
// Otherwise, newValue replaces existingValue.
func mergeTranslationValues(existingValue, newValue any) any {
	existingMap, existingIsMap := existingValue.(map[string]any)
	newMap, newIsMap := newValue.(map[string]any)

	// If both are maps, merge them recursively
	if existingIsMap && newIsMap {
		merged := make(map[string]any)
		// Copy existing values
		for k, v := range existingMap {
			merged[k] = v
		}
		// Merge new values (recursively for nested maps)
		for k, v := range newMap {
			if existingVal, exists := merged[k]; exists {
				merged[k] = mergeTranslationValues(existingVal, v)
			} else {
				merged[k] = v
			}
		}
		return merged
	}

	// If either is not a map, replace (overwrite)
	return newValue
}

func (m *manager) queueTranslation(lang language.Tag, key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.queue == nil {
		mngr.queue = make(map[language.Tag]map[string]any)
	}
	queuedict, ok := mngr.queue[lang]
	if !ok {
		queuedict = make(map[string]any)
		mngr.queue[lang] = queuedict
	}

	// If key already exists and both values are maps, merge them.
	// Otherwise, replace (overwrite) with new value.
	if existingValue, exists := queuedict[key]; exists {
		queuedict[key] = mergeTranslationValues(existingValue, value)
	} else {
		queuedict[key] = value
	}
	return nil
}

func (m *manager) addToDictionary(lang language.Tag, key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dictionary, ok := m.dictionaries[lang]
	if !ok {
		dictionary = make(map[string]any)
	}
	// Allow overwriting existing keys
	dictionary[key] = value
	m.dictionaries[lang] = dictionary
}

func registerTranslations(lang language.Tag, translations map[string]any) error {
	defer reload()

	// Check if translations are flat (no root key) or structured (with root key)
	// Root keys typically start with domain-like patterns (com., org., etc.)
	hasRootKey := false
	for key := range translations {
		if looksLikeRootKey(key) {
			hasRootKey = true
			break
		}
	}

	// Track root keys - top-level keys in the translations map are root keys
	// But only if they look like root keys (structured format)
	mngr.mu.Lock()
	if hasRootKey {
		for rootKey := range translations {
			if looksLikeRootKey(rootKey) {
				mngr.rootKeys[rootKey] = true
			}
		}
	}
	mngr.mu.Unlock()

	// If flat format (no root key), register as-is without prefix
	// The keys will be used directly (e.g., "app.description")
	// If structured format (has root key), register with empty prefix
	// The root key becomes part of the full key (e.g., "com.github.happy-sdk.banctl.app.description")
	for key, value := range translations {
		if err := mngr.registerTranslation(lang, "", key, value); err != nil {
			return err
		}
	}
	return nil
}

// looksLikeRootKey checks if a key looks like a package identifier root key
// Root keys typically start with domain-like patterns: com., org., net., etc.
func looksLikeRootKey(key string) bool {
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return false
	}
	// Check if first part is a common TLD or domain pattern
	firstPart := strings.ToLower(parts[0])
	commonTLDs := []string{"com", "org", "net", "io", "dev", "app", "github", "gitlab"}
	for _, tld := range commonTLDs {
		if firstPart == tld {
			return true
		}
	}
	// Also check if it has at least 3 parts (typical for package identifiers)
	// e.g., "com.github.happy-sdk"
	return len(parts) >= 3
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
	slog.Debug(T("com.github.happy-sdk.happy.pkg.i18n.set_default", lang.String()))
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

func getAllTranslations() []TranslationEntry {
	return mngr.getAllTranslations()
}

func getTranslationReport(lang language.Tag) TranslationReport {
	return mngr.getTranslationReport(lang)
}

func (m *manager) getAllTranslations() []TranslationEntry {
	m.mu.RLock()

	if !m.initialized {
		m.mu.RUnlock()
		return nil
	}

	// Collect all unique keys from all dictionaries and copy data we need
	allKeys := make(map[string]bool)
	fallbackLang := m.fallbackLang
	fallbackDict := m.dictionaries[fallbackLang]
	for key := range fallbackDict {
		allKeys[key] = true
	}
	for _, dict := range m.dictionaries {
		for key := range dict {
			allKeys[key] = true
		}
	}

	// Copy data we need while holding the lock
	langs := make([]language.Tag, len(m.langs))
	copy(langs, m.langs)
	fallbackPrinter := m.fallbackPrinter
	dictionariesCopy := make(map[language.Tag]map[string]any)
	for lang, dict := range m.dictionaries {
		dictCopy := make(map[string]any)
		for k, v := range dict {
			dictCopy[k] = v
		}
		dictionariesCopy[lang] = dictCopy
	}
	// Copy fallback dict
	var fallbackDictCopy map[string]any
	if fallbackDict != nil {
		fallbackDictCopy = make(map[string]any)
		for k, v := range fallbackDict {
			fallbackDictCopy[k] = v
		}
	}
	// Copy keyToRoot mapping
	keyToRootCopy := make(map[string]string)
	for k, v := range m.keyToRoot {
		keyToRootCopy[k] = v
	}

	m.mu.RUnlock()

	// Build translation entries (now we can safely call getPrinterFor)
	entries := make([]TranslationEntry, 0, len(allKeys))
	for key := range allKeys {
		rootKey := keyToRootCopy[key]
		if rootKey == "" {
			// Try to extract root key if not already tracked
			rootKey = m.extractRootKey(key)
		}
		entry := TranslationEntry{
			Key:          key,
			RootKey:      rootKey,
			Translations: make(map[language.Tag]string),
		}

		// Get fallback value
		if fallbackDictCopy != nil {
			if val, ok := fallbackDictCopy[key]; ok {
				entry.Fallback = fmt.Sprintf("%v", val)
			} else {
				// Try to get from printer (might be in catalog but not in dict)
				if fallbackPrinter != nil {
					result := fallbackPrinter.Sprintf(key)
					if result != key {
						entry.Fallback = result
					}
				}
			}
		}

		// Get translations for all supported languages
		for _, lang := range langs {
			if lang == fallbackLang {
				// Already handled as fallback
				continue
			}

			dict, ok := dictionariesCopy[lang]
			if ok {
				if val, exists := dict[key]; exists {
					entry.Translations[lang] = fmt.Sprintf("%v", val)
					continue
				}
			}

			// Try to get from printer (might be in catalog but not in dict)
			printer, err := m.getPrinterFor(lang)
			if err == nil && printer != nil {
				result := printer.Sprintf(key)
				if result != key {
					entry.Translations[lang] = result
				}
			}
		}

		entries = append(entries, entry)
	}

	return entries
}

// extractRootKey extracts the root key from a full translation key.
// For example: "com.github.happy-sdk.happy.sdk.cli.flags.version" -> "com.github.happy-sdk.happy.sdk.cli"
// For flat keys like "app.description" -> "app"
func (m *manager) extractRootKey(fullKey string) string {
	parts := strings.Split(fullKey, ".")
	if len(parts) == 0 {
		return ""
	}
	
	// For short keys (less than 5 parts), return the first segment as root key
	// e.g., "app.description" -> "app"
	if len(parts) < 5 {
		return parts[0]
	}
	
	// Common pattern: com.github.happy-sdk.happy.{pkg|sdk}.{name}
	// So root is typically first 5-6 parts
	if len(parts) >= 6 {
		// Check if it's a pkg or sdk pattern
		if parts[4] == "pkg" || parts[4] == "sdk" {
			return strings.Join(parts[:6], ".")
		}
	}
	// Default to first 5 parts
	return strings.Join(parts[:5], ".")
}

func (m *manager) getTranslationReport(lang language.Tag) TranslationReport {
	allEntries := m.getAllTranslations()

	if len(allEntries) == 0 {
		return TranslationReport{
			Language:       lang,
			Total:          0,
			Translated:     0,
			Missing:        0,
			Percentage:     0.0,
			MissingEntries: nil,
			RootKeys:       nil,
			PerRootKey:     make(map[string]RootKeyStats),
		}
	}

	var translatedCount int
	var missingEntries []TranslationEntry

	// Track root keys and per-root-key statistics
	rootKeysSet := make(map[string]bool)
	perRootKeyStats := make(map[string]struct {
		total      int
		translated int
		missing    int
	})

	for _, entry := range allEntries {
		rootKey := entry.RootKey
		if rootKey == "" {
			rootKey = "unknown"
		}
		rootKeysSet[rootKey] = true

		stats := perRootKeyStats[rootKey]
		stats.total++

		if _, hasTranslation := entry.Translations[lang]; hasTranslation {
			translatedCount++
			stats.translated++
		} else {
			missingEntries = append(missingEntries, entry)
			stats.missing++
		}
		perRootKeyStats[rootKey] = stats
	}

	total := len(allEntries)
	missing := len(missingEntries)
	percentage := 0.0
	if total > 0 {
		percentage = float64(translatedCount) / float64(total) * 100.0
	}

	// Build root keys list
	rootKeys := make([]string, 0, len(rootKeysSet))
	for rootKey := range rootKeysSet {
		rootKeys = append(rootKeys, rootKey)
	}
	slices.Sort(rootKeys)

	// Build per-root-key stats
	perRootKey := make(map[string]RootKeyStats)
	for rootKey, stats := range perRootKeyStats {
		rootPercentage := 0.0
		if stats.total > 0 {
			rootPercentage = float64(stats.translated) / float64(stats.total) * 100.0
		}
		perRootKey[rootKey] = RootKeyStats{
			RootKey:    rootKey,
			Total:      stats.total,
			Translated: stats.translated,
			Missing:    stats.missing,
			Percentage: rootPercentage,
		}
	}

	return TranslationReport{
		Language:       lang,
		Total:          total,
		Translated:     translatedCount,
		Missing:        missing,
		Percentage:     percentage,
		MissingEntries: missingEntries,
		RootKeys:       rootKeys,
		PerRootKey:     perRootKey,
	}
}
