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

	"github.com/happy-sdk/happy/pkg/i18n/schema"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

var (
	initOnce      sync.Once
	globalCatalog *catalog.Builder
	// mngr is a plain var initializer, not a func init(), so that Go's
	// package-init dependency analysis (which traces identifier use
	// through function/method bodies within the package, not just direct
	// initializer expressions) orders it before this package's own
	// sentinel LocalizedErrors (see i18n.go), whose initializers queue a
	// translation - reaching mngr - before Initialize is ever called.
	mngr = newManager()
)

func initialize(fallback language.Tag) []InitIssue {
	var issues []InitIssue
	initOnce.Do(func() {
		globalCatalog = catalog.NewBuilder(
			catalog.Fallback(fallback),
		)

		mngr.mu.Lock()
		mngr.initialized = true
		mngr.fallbackLang = fallback
		mngr.currentLang = fallback
		mngr.mu.Unlock()
		storeCurrentLang(fallback)

		// This package's own bundle (its sentinel error translations) must
		// be valid - a monorepo module isn't released unless every bundle
		// it ships is fully loadable, and this one is more foundational
		// than any other: if it can't load, nothing built on top of it can
		// be trusted either. Reported as Fatal here rather than panicking
		// (as MustEmbed would) so a caller building its own boot sequence
		// (see happy-sdk's own initializer) can fail the application
		// cleanly - the same way any other boot/init error does - instead
		// of a raw panic surfacing from deep inside this package.
		if err := RegisterTranslationsFS(NewFS(locales)); err != nil {
			issues = append(issues, InitIssue{Err: err, Fatal: true})
		}

		// Process any queued translations that were registered before
		// initialization - never Fatal, see InitIssue.
		issues = append(issues, reload()...)
	})
	return issues
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
		bundleSource: make(map[string]language.Tag),
		bundleNotes:  make(map[string]map[language.Tag]string),
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
	// bundleSource maps a schema version 2 bundle (root key) to its
	// declared source locale (see schema/v2.KeySource) - the language its
	// translations are authored/maintained in. A v1 bundle never appears
	// here at all; callers should treat that as "unknown", not assume
	// schema/v2.DefaultSource.
	bundleSource map[string]language.Tag
	// bundleNotes maps a schema version 2 bundle to its per-locale
	// translator notes (see schema/v2.KeyLocaleNotes), folded into the
	// immutable snapshot's bundleMeta so tooling reads them without a
	// separate side map.
	bundleNotes map[string]map[language.Tag]string
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
		return ErrLanguageNotSupported.WithArgs("lang", lang.String())
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentLang = lang
	m.currentPrinter = message.NewPrinter(m.currentLang, message.Catalog(globalCatalog))
	storeCurrentLang(lang)
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

// getBundleSource reports bundle's declared source locale (see schema/v2.
// KeySource) and whether bundle is even known (a schema version 2 bundle
// this manager has registered at least one locale for). ok is false for a
// v1 (legacy, unversioned) bundle or one that's never been registered -
// callers should not assume schema/v2.DefaultSource in that case, since a
// v1 bundle predates the concept entirely.
func (m *manager) getBundleSource(bundle string) (lang language.Tag, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	lang, ok = m.bundleSource[bundle]
	return lang, ok
}

func (m *manager) getSupported() []language.Tag {
	m.mu.RLock()
	defer m.mu.RUnlock()
	langs := m.langs
	return langs
}

// messageArgTypes reports the derived ArgType of every literal argument the
// translation registered for (lang, key) interpolates - the same derivation
// Message.ArgTypes() performs, whether the registered value is a rich *Message
// or a plain {name:verb} string. ok is false if nothing is registered for
// (lang, key). Intended for code-generation/tooling (see addons/l10n), never
// the render path.
func (m *manager) messageArgTypes(lang language.Tag, key string) (map[string]ArgType, bool) {
	m.mu.RLock()
	dict := m.dictionaries[lang]
	var v any
	var ok bool
	if dict != nil {
		v, ok = dict[key]
	}
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	switch t := v.(type) {
	case *Message:
		return t.ArgTypes(), true
	case string:
		// A plain named template ({name:verb}) has the same arg-type
		// derivation as a Message whose Msg is that string and which has no
		// fragments.
		return NewMessage(t).ArgTypes(), true
	default:
		return map[string]ArgType{}, true
	}
}

// message returns the rich *Message registered for (lang, key), if that's
// what's actually registered there - unlike messageArgTypes, which derives
// the same arg-type info uniformly whether a key is a *Message or a plain
// string, this reports false for anything other than a genuine *Message,
// since a plain string or legacy catalog.Message has no fragments/
// description for a caller to inspect. Intended for translation tooling
// (see addons/l10n) wanting schema-level detail, never the render path.
func (m *manager) message(lang language.Tag, key string) (*Message, bool) {
	m.mu.RLock()
	dict := m.dictionaries[lang]
	var v any
	var ok bool
	if dict != nil {
		v, ok = dict[key]
	}
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	msg, ok := v.(*Message)
	return msg, ok
}

// keyBundle reports the schema version 2 bundle key belongs to (see
// owningBundle) and whether it belongs to one at all - false for a legacy
// (schema version 1) or unregistered key.
func (m *manager) keyBundle(key string) (bundle string, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bundle = m.owningBundle(key)
	return bundle, bundle != ""
}

// bundleNote returns bundle's translator note for lang (see
// schema/v2.KeyLocaleNotes) and whether it has one at all - most
// bundle/locale pairs won't, since notes are optional.
func (m *manager) bundleNote(bundle string, lang language.Tag) (note string, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	note, ok = m.bundleNotes[bundle][lang]
	return note, ok
}

func (m *manager) reload() []InitIssue {
	m.mu.RLock()
	if !m.initialized {
		m.mu.RUnlock()
		return nil
	}

	queue := m.queue
	fallbackLang := m.fallbackLang
	m.mu.RUnlock()
	var issues []InitIssue
	for lang, dict := range queue {
		if lang == language.Und {
			lang = fallbackLang
		}
		for key, value := range dict {
			if err := m.registerTranslation(lang, "", key, value); err != nil {
				issues = append(issues, InitIssue{Err: err})
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
	m.printerCache = printerCache

	// Prune any unknown locale key before the snapshot is built and published,
	// so a typo'd key never becomes visible to T/TL, GetAllTranslations, or
	// any other reader of m.dictionaries - only the InitIssue warning survives
	// it, never the key itself.
	m.pruneUnknownLocaleKeysLocked()

	// Publish a fresh immutable snapshot for the lock-free read path. This is
	// the only place a snapshot is built; T/TL never take m.mu.
	storeSnapshot(m.buildSnapshotLocked(printerCache))
	return issues
}

// pruneUnknownLocaleKeysLocked checks every known bundle's non-source locales
// (see bundleSource) for a key that doesn't exist anywhere in that bundle's
// declared source locale - almost always a typo in the key name itself (as
// opposed to its translated value), which schema validation can't catch (a
// typo'd key is still a perfectly well-shaped key - it's just not the one the
// source actually has). Any such key is deleted from m.dictionaries right
// here, before the caller builds and publishes the next snapshot from it: the
// library must never register a key it can't vouch for, so the fix for a
// found typo is to warn about it while excluding it from rendering and
// reporting, not merely warn.
//
// Found issues go through recordPendingIssue rather than this function's own
// return value: reload runs many times per registration batch (once per
// queued key, via each registerTranslation's own deferred reload), and only
// the first call with enough state loaded to compare against the source
// locale will ever see the bad key - by design, since it's pruned the moment
// it's found, every later call in the same batch recomputes over an already-
// clean dictionary and finds nothing. That first call is essentially never
// the batch's final, externally-visible Initialize/Reload, whose return
// value is what callers actually check - it would go unseen if reported only
// there. Recording it instead means whichever Initialize/Reload call comes
// next drains and surfaces it, exactly like Embed/EmbedIssues issues do. Must
// be called with m.mu held; mutates m.dictionaries.
func (m *manager) pruneUnknownLocaleKeysLocked() {
	for bundle, source := range m.bundleSource {
		sourceDict, ok := m.dictionaries[source]
		if !ok {
			continue // source locale itself not loaded yet
		}
		sourceKeys := bundleRelativeKeys(sourceDict, bundle)
		if len(sourceKeys) == 0 {
			continue
		}
		for lang, dict := range m.dictionaries {
			if lang == source {
				continue
			}
			for key := range bundleRelativeKeys(dict, bundle) {
				if sourceKeys[key] {
					continue
				}
				suggestion, hasSuggestion := closestKey(key, sourceKeys)
				hasSuggestionStr := "no"
				if hasSuggestion {
					hasSuggestionStr = "yes"
				}
				recordPendingIssue(InitIssue{Err: ErrUnknownLocaleKey.WithArgs(
					"bundle", bundle,
					"lang", lang.String(),
					"key", key,
					"source", source.String(),
					"has_suggestion", hasSuggestionStr,
					"suggestion", suggestion,
				)})
				delete(dict, bundle+"."+key)
			}
		}
	}
}

// bundleRelativeKeys returns the set of dict's keys that belong to bundle
// (i.e. start with "bundle."), with that prefix stripped - so a key reads
// the same way it does in the source *.json file's own "keys" tree,
// rather than as the full bundle-qualified key T/TL actually look up.
func bundleRelativeKeys(dict map[string]any, bundle string) map[string]bool {
	prefix := bundle + "."
	keys := make(map[string]bool, len(dict))
	for k := range dict {
		if rel, ok := strings.CutPrefix(k, prefix); ok {
			keys[rel] = true
		}
	}
	return keys
}

// buildSnapshotLocked compiles every registered translation into an immutable
// snapshot ready to render with no locking. It must be called with m.mu held.
// printers is the per-language printer set just rebuilt by reload, reused here
// so number/currency/legacy/catalog rendering can format locale-aware output
// off the immutable snapshot without any further locking.
func (m *manager) buildSnapshotLocked(printers map[language.Tag]*message.Printer) *snapshot {
	s := &snapshot{
		byLang:    make(map[language.Tag]map[string]*program, len(m.dictionaries)),
		bundles:   make(map[string]bundleMeta, len(m.bundleSource)),
		keyBundle: make(map[string]string),
		langs:     append([]language.Tag(nil), m.langs...),
		fallback:  m.fallbackLang,
		printers:  printers,
	}
	for lang, dict := range m.dictionaries {
		pm := make(map[string]*program, len(dict))
		for key, val := range dict {
			pm[key] = compileValue(key, val)
			if b := m.owningBundle(key); b != "" {
				s.keyBundle[key] = b
			}
		}
		s.byLang[lang] = pm
	}
	for bundle, src := range m.bundleSource {
		s.bundles[bundle] = bundleMeta{source: src, notes: m.bundleNotes[bundle]}
	}
	s.matcher = language.NewMatcher(s.langs)
	return s
}

// owningBundle returns the schema version 2 bundle key owns, or "" for a
// v1/unbundled key. A key belongs to bundle B when it is B itself or is nested
// directly under it (B+"."); this is authoritative, not a heuristic - a v2
// document's keys are literally registered under their declared bundle. The
// longest matching bundle wins, so a nested bundle is preferred over an
// ancestor that happens to share its prefix.
func (m *manager) owningBundle(key string) string {
	best := ""
	for bundle := range m.bundleSource {
		if key == bundle || strings.HasPrefix(key, bundle+".") {
			if len(bundle) > len(best) {
				best = bundle
			}
		}
	}
	return best
}

// compileValue turns one dictionary value (a string, a rich *Message, or a
// legacy x/text catalog.Message) into its compiled render program.
func compileValue(key string, val any) *program {
	switch v := val.(type) {
	case string:
		return compileString(v)
	case *Message:
		return compileMessage(v)
	default:
		// catalog.Message / []catalog.Message and anything else registered
		// through the legacy x/text path render through the language printer
		// by key.
		return &program{catalogKey: key}
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
		return nil, ErrLanguageNotSupported.WithArgs("lang", lang.String())
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

	fullKey := key
	if prefix != "" {
		fullKey = prefix + "." + key
	}

	if !m.isInitialized() {
		// Queue under fullKey, not key alone: prefix (e.g. a schema
		// version 2 document's own bundle) must not be lost here just
		// because registration happened before Initialize - reload()'s
		// later flush calls back in with prefix == "", trusting the
		// queued key to already be complete.
		return m.queueTranslation(lang, fullKey, value)
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
		// catalog.Builder isn't safe for concurrent writers - unlike the
		// snapshot rebuild in reload() (which already serializes under
		// m.mu), this write path is reachable directly from any concurrent
		// RegisterTranslation(s) caller, so it needs its own lock. Held
		// only around the mutation itself, never across the recursive
		// call in the map[string]any case below, which would deadlock.
		m.mu.Lock()
		err := globalCatalog.SetString(lang, fullKey, v)
		m.mu.Unlock()
		return err
	case catalog.Message:
		shouldSupport = true
		m.mu.Lock()
		err := globalCatalog.Set(lang, fullKey, v)
		m.mu.Unlock()
		return err
	case []catalog.Message:
		shouldSupport = true
		m.mu.Lock()
		err := globalCatalog.Set(lang, fullKey, v...)
		m.mu.Unlock()
		return err
	case *Message:
		if err := v.validate(); err != nil {
			return err
		}
		shouldSupport = true
		return nil
	case map[string]any:
		if msg, isMessage, err := parseMessageObject(v); isMessage {
			if err != nil {
				return err
			}
			shouldSupport = true
			value = msg
			return nil
		}
		for key, value := range v {
			if err := m.registerTranslation(lang, fullKey, key, value); err != nil {
				return err
			}
		}
	default:
		return ErrUnsupportedType.WithArgs("type", value, "key", fullKey, "lang", lang.String())
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

// registerTranslations registers translations - a translation file's (or a
// direct RegisterTranslations caller's) top-level parsed JSON object -
// dispatching it through package schema to figure out which language(s) it
// actually covers: an unversioned (schema version 1) document carries
// exactly one, supplied by lang since the document itself doesn't know its
// own locale; a version 2 document is self-describing and may carry
// several, in which case lang is ignored entirely in favor of whatever the
// document itself declares.
func registerTranslations(lang language.Tag, translations map[string]any) error {
	defer reload()

	doc, err := schema.Parse(lang, translations)
	if err != nil {
		return ErrInvalidSchemaVersion.WithArgs("lang", lang.String(), "cause", err.Error())
	}
	if doc.Bundle != "" {
		mngr.mu.Lock()
		mngr.bundleSource[doc.Bundle] = doc.Source
		if len(doc.Notes) > 0 {
			notes := mngr.bundleNotes[doc.Bundle]
			if notes == nil {
				notes = make(map[language.Tag]string, len(doc.Notes))
				mngr.bundleNotes[doc.Bundle] = notes
			}
			for l, n := range doc.Notes {
				notes[l] = n
			}
		}
		mngr.mu.Unlock()
	}
	for docLang, body := range doc.Locales {
		if err := registerTranslationsBody(docLang, doc.Bundle, body); err != nil {
			return err
		}
	}
	return nil
}

// registerTranslationsBody registers body as one language's own resolved
// translation tree. bundle, if non-empty (a schema version 2 document
// always supplies one - see schema.Document.Bundle), is body's already-
// known root key: every entry in body nests directly under it, with no
// need to guess via looksLikeRootKey at all. bundle == "" is schema
// version 1's legacy case, where body's own top-level keys are root keys
// (or the whole tree is flat/unprefixed) - preserved exactly as it always
// worked, via looksLikeRootKey.
func registerTranslationsBody(lang language.Tag, bundle string, body map[string]any) error {
	if bundle != "" {
		mngr.mu.Lock()
		mngr.rootKeys[bundle] = true
		mngr.mu.Unlock()

		for key, value := range body {
			if err := mngr.registerTranslation(lang, bundle, key, value); err != nil {
				return err
			}
		}
		return nil
	}

	// Check if translations are flat (no root key) or structured (with root key)
	// Root keys typically start with domain-like patterns (com., org., etc.)
	hasRootKey := false
	for key := range body {
		if looksLikeRootKey(key) {
			hasRootKey = true
			break
		}
	}

	// Track root keys - top-level keys in the translations map are root keys
	// But only if they look like root keys (structured format)
	mngr.mu.Lock()
	if hasRootKey {
		for rootKey := range body {
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
	for key, value := range body {
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
		return ErrLanguageNotSupported.WithArgs("lang", lang.String())
	}
	if err := mngr.setCurrentLanguage(lang); err != nil {
		return err
	}
	slog.Debug(T("com.github.happy-sdk.happy.pkg.i18n.set_default", "lang", lang.String()))
	return nil
}

func getLanguage() language.Tag {
	// Lock-free read of the current language: T/TL call this on every render,
	// so it must never take the manager mutex. setCurrentLanguage/initialize
	// keep this atomic in sync with mngr.currentLang.
	return loadCurrentLang()
}

func getFallbackLanguage() language.Tag {
	lang := mngr.getFallbackLanguage()
	return lang
}

// getLanguages reports every language with at least one registered
// translation. It reads mngr's own tracking (updated for every
// translation type via registerTranslation's shared m.support(lang) call)
// rather than globalCatalog.Languages(): a *Message value never touches
// globalCatalog at all (it's rendered by this package's own engine, not
// x/text's), so a language whose bundle is entirely Message-based would
// otherwise never show up as supported.
func getLanguages() []language.Tag {
	return mngr.getSupported()
}

func parseLanguage(langStr string) language.Tag {
	if langStr == "" {
		return mngr.getFallbackLanguage()
	}

	tag, err := language.Parse(langStr)
	if err != nil {
		return mngr.getFallbackLanguage()
	}

	// Prefer the immutable snapshot's matcher, built once per rebuild rather
	// than allocated per call.
	if s := loadSnapshot(); s != nil && s.matcher != nil {
		if _, index, _ := s.matcher.Match(tag); index >= 0 && index < len(s.langs) {
			return s.langs[index]
		}
	}

	supportedLangs := mngr.getSupported()
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

func reload() []InitIssue {
	return mngr.reload()
}

// t translates key using the current language, cascading through the snapshot's
// fallback chain (see snapshot.resolve) and returning the raw key as a last
// resort if nothing is registered. It reads the immutable snapshot with no
// locking.
func t(key string, args ...any) string {
	return translate(getLanguage(), key, args)
}

// tForLanguage translates key using an explicit language tag, otherwise
// identical to t. An unsupported language simply resolves nothing for that tag
// and cascades to the fallback, still returning the raw key only as a last
// resort.
func tForLanguage(lang language.Tag, key string, args ...any) string {
	return translate(lang, key, args)
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
	fallbackLang := m.getFallbackLanguage()

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

		// For the fallback language, the value lives in entry.Fallback rather
		// than entry.Translations (see getAllTranslations), so check that
		// field instead when the requested language is the fallback language.
		var hasTranslation bool
		if lang == fallbackLang {
			hasTranslation = entry.Fallback != ""
		} else {
			_, hasTranslation = entry.Translations[lang]
		}

		if hasTranslation {
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
