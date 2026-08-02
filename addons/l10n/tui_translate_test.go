// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/pkg/i18n"
	"golang.org/x/text/language"
)

func TestTranslateWorklistMatchesInteractiveModeFiltering(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/translateworklisttest",
		"github.com/some/dependency v1.0.0",
	)
	defer cleanup()
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	appKey := fmt.Sprintf("com.github.happy-sdk.translateworklisttest.unique.app%d", nonce)
	depKey := fmt.Sprintf("com.github.some.dependency.unique.dep%d", nonce)
	if err := i18n.RegisterTranslation(language.English, appKey, "App"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	if err := i18n.RegisterTranslation(language.English, depKey, "Dep"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	appOnly, err := translateWorklist(sess, language.French, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if containsEntryKey(appOnly, depKey) {
		t.Error("expected dependency key to be excluded when withDeps is false")
	}
	if !containsEntryKey(appOnly, appKey) {
		t.Error("expected app key (missing in French) to be present")
	}

	withDeps, err := translateWorklist(sess, language.French, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsEntryKey(withDeps, depKey) {
		t.Error("expected dependency key to be present when withDeps is true")
	}
}

func containsEntryKey(entries []i18n.TranslationEntry, key string) bool {
	for _, e := range entries {
		if e.Key == key {
			return true
		}
	}
	return false
}

func TestTranslateModelCycleLang(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/translatecycletest")
	defer cleanup()

	m := newTranslateModel(sess, language.French, false)
	m.langs = []language.Tag{language.French, language.German, language.Estonian}

	m.cycleLang(1)
	if m.lang != language.German {
		t.Errorf("expected German after cycling forward from French, got %s", m.lang)
	}
	m.cycleLang(1)
	if m.lang != language.Estonian {
		t.Errorf("expected Estonian, got %s", m.lang)
	}
	m.cycleLang(1)
	if m.lang != language.French {
		t.Errorf("expected wraparound back to French, got %s", m.lang)
	}
	m.cycleLang(-1)
	if m.lang != language.Estonian {
		t.Errorf("expected wraparound backward to Estonian, got %s", m.lang)
	}
}

func TestTranslateModelJumpToKeyAndEsc(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/translatejumptest")
	defer cleanup()
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	key := fmt.Sprintf("com.github.happy-sdk.translatejumptest.unique.k%d", nonce)
	if err := i18n.RegisterTranslation(language.English, key, "Hello"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	if err := i18n.RegisterTranslation(language.French, key, "Bonjour"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	m := newTranslateModel(sess, language.French, false)
	m.jumpToKey(key)

	if m.manualEntry == nil {
		t.Fatal("expected manualEntry to be set after jumpToKey")
	}
	if m.manualEntry.Key != key {
		t.Errorf("manualEntry.Key = %q, want %q", m.manualEntry.Key, key)
	}
	// Regression: jumpToKey must default to the key's own source locale
	// (English here, since this is an unbundled key with no declared
	// source - see entrySourceLang), not whatever language was selected
	// before the jump (French) - see this function's own doc comment.
	if m.lang != language.English {
		t.Errorf("expected jumpToKey to default to the source locale (English), got %s", m.lang)
	}
	if !strings.Contains(m.viewport.View(), "Hello") {
		t.Errorf("expected inspector content to show the source (English) text by default, got:\n%s", m.viewport.View())
	}

	// The French translation is still reachable by cycling - source locale
	// is added to the cycle, it doesn't replace the target languages. Other
	// tests in this package register other languages globally too, so the
	// exact number of cycleLang(1) steps to reach French isn't fixed -
	// cycle until it's found (bounded by the cycle's own length, so this
	// can't loop forever if French were somehow unreachable).
	found := false
	for range m.languagesToCycle() {
		m.cycleLang(1)
		if m.lang == language.French {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected French to be reachable by cycling")
	}
	if !strings.Contains(m.viewport.View(), "Bonjour") {
		t.Errorf("expected inspector content to show the French translation, got:\n%s", m.viewport.View())
	}

	newM, _ := m.Update(tea.KeyPressMsg{Text: "esc", Code: tea.KeyEscape})
	if newM.manualEntry != nil {
		t.Error("expected esc to clear manualEntry")
	}
}

func TestTranslateModelNoWorklistShowsPlaceholder(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/translateemptyworklisttest")
	defer cleanup()

	m := newTranslateModel(sess, language.French, false)
	content := m.View().Content
	if !strings.Contains(strings.ToLower(content), "language") {
		t.Errorf("expected view to render, got:\n%s", content)
	}
}

// TestTranslateModelHasNoSaveKeys is a regression test: this tab used to
// save an edited value on enter/ctrl+s. It's read-only for now (see
// translateModel's doc comment) - neither key should do anything besides
// whatever the viewport itself does with them (nothing, since viewport's
// own keymap doesn't bind either), and in particular must never dispatch a
// tea.Cmd (a save would have).
func TestTranslateModelHasNoSaveKeys(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/translatenosavetest")
	defer cleanup()
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	key := fmt.Sprintf("com.github.happy-sdk.translatenosavetest.unique.k%d", nonce)
	if err := i18n.RegisterTranslation(language.English, key, "Hello"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	m := newTranslateModel(sess, language.French, false)
	for _, key := range []string{"enter", "ctrl+s"} {
		_, cmd := m.Update(tea.KeyPressMsg{Text: key})
		if cmd != nil {
			t.Errorf("expected no command for key %q on a read-only view", key)
		}
	}
}

func TestBuildInspectContentPlainString(t *testing.T) {
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	key := fmt.Sprintf("com.github.happy-sdk.inspectplaintest.unique.k%d", nonce)
	if err := i18n.RegisterTranslation(language.English, key, "Hello {name}"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	if err := i18n.RegisterTranslation(language.French, key, "Bonjour {name}"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	var entries []i18n.TranslationEntry
	for _, e := range i18n.GetAllTranslations() {
		if e.Key == key {
			entries = append(entries, e)
		}
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one entry for %q, got %d", key, len(entries))
	}
	entry := entries[0]

	content := buildInspectContent(language.French, func() (i18n.TranslationEntry, bool) { return entry, true })
	if !strings.Contains(content, key) {
		t.Errorf("expected content to mention the key, got:\n%s", content)
	}
	if !strings.Contains(content, "Bonjour {name}") {
		t.Errorf("expected content to show the French text, got:\n%s", content)
	}
	if !strings.Contains(content, "Hello {name}") {
		t.Errorf("expected content to show the fallback text, got:\n%s", content)
	}
	if !strings.Contains(content, "Arguments:") || !strings.Contains(content, "name") {
		t.Errorf("expected content to list the \"name\" argument, got:\n%s", content)
	}
	if strings.Contains(content, "Message schema") {
		t.Errorf("did not expect a Message schema section for a plain string, got:\n%s", content)
	}
}

func TestBuildInspectContentRichMessageShowsFragmentsAndDescriptions(t *testing.T) {
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	bundle := fmt.Sprintf("com.github.happy-sdk.inspectmessagetest.bundle%d", nonce)
	err := i18n.RegisterTranslations(language.Und, map[string]any{
		"version": float64(2),
		"bundle":  bundle,
		"locales": map[string]any{
			"en": map[string]any{
				"keys": map[string]any{
					"items": map[string]any{
						"$message": map[string]any{
							"msg": "{count_p}",
							"fragments": map[string]any{
								"count_p": map[string]any{
									"arg":   "count",
									"one":   "{count:d} item",
									"other": "{count:d} items",
								},
							},
							"description": map[string]any{
								"msg": "shown in the cart summary",
								"args": map[string]any{
									"count": map[string]any{"description": "number of items in the cart"},
								},
							},
						},
					},
				},
				"notes": "keep this short - shown in a narrow badge",
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterTranslations failed: %v", err)
	}

	key := bundle + ".items"
	var entry i18n.TranslationEntry
	found := false
	for _, e := range i18n.GetAllTranslations() {
		if e.Key == key {
			entry, found = e, true
		}
	}
	if !found {
		t.Fatalf("expected %q to be registered", key)
	}

	content := buildInspectContent(language.English, func() (i18n.TranslationEntry, bool) { return entry, true })

	for _, want := range []string{
		"Message schema",
		"shown in the cart summary",
		"count_p",
		"one:",
		"{count:d} item",
		"other:",
		"{count:d} items",
		"Arguments:",
		"count",
		"number of items in the cart",
		"Translator note",
		"keep this short",
		"Bundle: " + bundle,
		"Source locale: en",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("expected content to contain %q, got:\n%s", want, content)
		}
	}
}

func TestBuildInspectContentMissingTranslationShowsPlaceholder(t *testing.T) {
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	key := fmt.Sprintf("com.github.happy-sdk.inspectmissingtest.unique.k%d", nonce)
	if err := i18n.RegisterTranslation(language.English, key, "Hello"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	var entry i18n.TranslationEntry
	for _, e := range i18n.GetAllTranslations() {
		if e.Key == key {
			entry = e
		}
	}

	content := buildInspectContent(language.German, func() (i18n.TranslationEntry, bool) { return entry, true })
	if !strings.Contains(content, "de: (missing)") {
		t.Errorf("expected a (missing) placeholder for a locale with no translation, got:\n%s", content)
	}
}

func TestBuildInspectContentNoEntry(t *testing.T) {
	content := buildInspectContent(language.French, func() (i18n.TranslationEntry, bool) {
		return i18n.TranslationEntry{}, false
	})
	if !strings.Contains(strings.ToLower(content), "nothing to inspect") {
		t.Errorf("expected a placeholder message when there's nothing to inspect, got:\n%s", content)
	}
}
