// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/pkg/i18n"
	"golang.org/x/text/language"
)

func TestBuildBrowseItemsExcludesDepsByDefault(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browseitemstest",
		"github.com/some/dependency v1.0.0",
	)
	defer cleanup()
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	appKey := fmt.Sprintf("com.github.happy-sdk.browseitemstest.unique.app%d", nonce)
	depKey := fmt.Sprintf("com.github.some.dependency.unique.dep%d", nonce)
	if err := i18n.RegisterTranslation(language.English, appKey, "App"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	if err := i18n.RegisterTranslation(language.English, depKey, "Dep"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	appOnly := buildBrowseItems(sess, false, false, browseSearchByKey, i18n.GetFallbackLanguage())
	if containsBrowseKey(appOnly, depKey) {
		t.Error("expected dependency key to be excluded when withDeps is false")
	}
	if !containsBrowseKey(appOnly, appKey) {
		t.Error("expected app key to be present when withDeps is false")
	}

	withDeps := buildBrowseItems(sess, true, false, browseSearchByKey, i18n.GetFallbackLanguage())
	if !containsBrowseKey(withDeps, depKey) {
		t.Error("expected dependency key to be present when withDeps is true")
	}
}

func containsBrowseKey(items []list.Item, key string) bool {
	for _, item := range items {
		if item.FilterValue() == key {
			return true
		}
	}
	return false
}

func TestBrowseModelToggleKeysIgnoredWhileFiltering(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browsefiltertest")
	defer cleanup()
	ensureI18nInitialized(t)

	// The list's "/" filter keybinding is only enabled when it has items
	// (see bubbles/v2/list's updateKeybindings), so this test needs at
	// least one registered translation to actually exercise filter mode.
	key := fmt.Sprintf("com.github.happy-sdk.browsefiltertest.unique.k%d", time.Now().UnixNano())
	if err := i18n.RegisterTranslation(language.English, key, "Value"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	m := newBrowseModel(sess, false)
	m.list.SetSize(80, 20)
	m.Invalidate()

	// Enter filtering mode.
	newM, _ := m.Update(tea.KeyPressMsg{Text: "/", Code: '/'})
	if !newM.list.SettingFilter() {
		t.Fatal("expected '/' to enter filter mode")
	}

	before := newM.withDeps
	newM, _ = newM.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	if newM.withDeps != before {
		t.Error("expected 'd' typed while filtering to be treated as filter text, not a toggle")
	}
}

func TestBrowseModelDToggleOutsideFilter(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browsetoggletest")
	defer cleanup()

	m := newBrowseModel(sess, false)
	m.list.SetSize(80, 20)

	newM, cmd := m.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	if cmd != nil {
		t.Error("expected no command from toggling deps")
	}
	if !newM.withDeps {
		t.Error("expected 'd' to toggle withDeps on")
	}
}

func TestBrowseModelEnterSendsSelectKeyMsg(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browseentertest")
	defer cleanup()
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	key := fmt.Sprintf("com.github.happy-sdk.browseentertest.unique.k%d", nonce)
	if err := i18n.RegisterTranslation(language.English, key, "Value"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	m := newBrowseModel(sess, false)
	m.list.SetSize(80, 20)
	m.Invalidate()

	// Select the item we just registered so enter's message is deterministic.
	for i, item := range m.list.Items() {
		if bi, ok := item.(browseItem); ok && bi.key == key {
			m.list.Select(i)
		}
	}

	_, cmd := m.Update(tea.KeyPressMsg{Text: "enter", Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected enter to produce a command")
	}
	msg, ok := cmd().(browseSelectKeyMsg)
	if !ok {
		t.Fatalf("expected browseSelectKeyMsg, got %T", msg)
	}
	if msg.Key != key {
		t.Errorf("Key = %q, want %q", msg.Key, key)
	}
}

func TestBrowseModelSToggleSearchMode(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browsesearchmodetest")
	defer cleanup()

	m := newBrowseModel(sess, false)
	m.list.SetSize(80, 20)

	if m.searchMode != browseSearchByKey {
		t.Fatalf("initial searchMode = %v, want browseSearchByKey", m.searchMode)
	}
	keyTitle := m.list.Title

	newM, cmd := m.Update(tea.KeyPressMsg{Text: "s", Code: 's'})
	if cmd != nil {
		t.Error("expected no command from toggling search mode")
	}
	if newM.searchMode != browseSearchByContent {
		t.Error("expected 's' to switch searchMode to browseSearchByContent")
	}
	if newM.list.Title == keyTitle {
		t.Error("expected list.Title to change after switching search mode")
	}

	backM, _ := newM.Update(tea.KeyPressMsg{Text: "s", Code: 's'})
	if backM.searchMode != browseSearchByKey {
		t.Error("expected a second 's' to switch searchMode back to browseSearchByKey")
	}
	if backM.list.Title != keyTitle {
		t.Error("expected list.Title to be restored after switching back to key mode")
	}
}

func TestBrowseModelSToggleIgnoredWhileFiltering(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browsesearchmodefiltertest")
	defer cleanup()
	ensureI18nInitialized(t)

	key := fmt.Sprintf("com.github.happy-sdk.browsesearchmodefiltertest.unique.k%d", time.Now().UnixNano())
	if err := i18n.RegisterTranslation(language.English, key, "Value"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	m := newBrowseModel(sess, false)
	m.list.SetSize(80, 20)
	m.Invalidate()

	newM, _ := m.Update(tea.KeyPressMsg{Text: "/", Code: '/'})
	if !newM.list.SettingFilter() {
		t.Fatal("expected '/' to enter filter mode")
	}

	before := newM.searchMode
	newM, _ = newM.Update(tea.KeyPressMsg{Text: "s", Code: 's'})
	if newM.searchMode != before {
		t.Error("expected 's' typed while filtering to be treated as filter text, not a mode toggle")
	}
}

func TestBrowseModelCycleContentLangWrapsBothDirections(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browsecyclelangtest")
	defer cleanup()
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	key := fmt.Sprintf("com.github.happy-sdk.browsecyclelangtest.unique.k%d", nonce)
	if err := i18n.RegisterTranslation(language.English, key, "Value"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	if err := i18n.RegisterTranslation(language.Estonian, key, "Väärtus"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	langs := browseContentLanguages()
	if len(langs) < 2 {
		t.Fatalf("expected at least 2 registered languages for a meaningful wrap test, got %d (%v)", len(langs), langs)
	}

	m := newBrowseModel(sess, false)
	m.list.SetSize(80, 20)

	// Cycling backward from the first language should wrap to the last.
	m.contentLang = langs[0]
	m.cycleContentLang(-1)
	if m.contentLang != langs[len(langs)-1] {
		t.Errorf("cycleContentLang(-1) from first = %v, want wrap to last %v", m.contentLang, langs[len(langs)-1])
	}

	// Cycling forward from the last language should wrap to the first.
	m.contentLang = langs[len(langs)-1]
	m.cycleContentLang(1)
	if m.contentLang != langs[0] {
		t.Errorf("cycleContentLang(1) from last = %v, want wrap to first %v", m.contentLang, langs[0])
	}

	// A middle step (not wrapping) should just move by one.
	if len(langs) >= 3 {
		m.contentLang = langs[1]
		m.cycleContentLang(1)
		if m.contentLang != langs[2] {
			t.Errorf("cycleContentLang(1) from middle = %v, want %v", m.contentLang, langs[2])
		}
	}
}

func TestBrowseModelCycleContentLangKeysOutsideFilter(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browsecyclelangkeystest")
	defer cleanup()
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	key := fmt.Sprintf("com.github.happy-sdk.browsecyclelangkeystest.unique.k%d", nonce)
	if err := i18n.RegisterTranslation(language.English, key, "Value"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	if err := i18n.RegisterTranslation(language.Estonian, key, "Väärtus"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	m := newBrowseModel(sess, false)
	m.list.SetSize(80, 20)
	before := m.contentLang

	newM, cmd := m.Update(tea.KeyPressMsg{Text: "]", Code: ']'})
	if cmd != nil {
		t.Error("expected no command from cycling contentLang")
	}
	if newM.contentLang == before {
		t.Error("expected ']' to change contentLang when at least 2 languages are registered")
	}

	backM, _ := newM.Update(tea.KeyPressMsg{Text: "[", Code: '['})
	if backM.contentLang != before {
		t.Errorf("expected '[' to undo the ']' cycle: got %v, want %v", backM.contentLang, before)
	}
}

func TestBuildBrowseItemsContentModeUsesRawTranslationText(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browsecontentmodetest")
	defer cleanup()
	ensureI18nInitialized(t)

	nonce := time.Now().UnixNano()
	withEstonian := fmt.Sprintf("com.github.happy-sdk.browsecontentmodetest.unique.with%d", nonce)
	withoutEstonian := fmt.Sprintf("com.github.happy-sdk.browsecontentmodetest.unique.without%d", nonce)

	if err := i18n.RegisterTranslation(language.English, withEstonian, "Hello"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	if err := i18n.RegisterTranslation(language.Estonian, withEstonian, "Tere"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	if err := i18n.RegisterTranslation(language.English, withoutEstonian, "Goodbye"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}

	items := buildBrowseItems(sess, false, false, browseSearchByContent, language.Estonian)

	var found, foundWithout bool
	for _, it := range items {
		bi, ok := it.(browseItem)
		if !ok {
			continue
		}
		switch bi.key {
		case withEstonian:
			found = true
			if bi.FilterValue() != "Tere" {
				t.Errorf("FilterValue() = %q, want the raw registered text %q", bi.FilterValue(), "Tere")
			}
			if strings.Contains(bi.FilterValue(), "MISSING") {
				t.Errorf("FilterValue() contains rendered-missing-arg noise: %q", bi.FilterValue())
			}
		case withoutEstonian:
			foundWithout = true
			if bi.FilterValue() != "" {
				t.Errorf("FilterValue() for a key with no Estonian translation = %q, want empty", bi.FilterValue())
			}
			if !strings.Contains(bi.Description(), "no translation for") {
				t.Errorf("Description() = %q, want a no-translation placeholder", bi.Description())
			}
		}
	}
	if !found {
		t.Fatal("expected the Estonian-translated key to be present in content-mode items")
	}
	if !foundWithout {
		t.Fatal("expected the key with no Estonian translation to still be LISTED in content-mode items, not hidden")
	}
}

func TestDescribeContentPreviewTruncatesLongContent(t *testing.T) {
	long := strings.Repeat("a", browseContentPreviewRunes+10)
	preview := describeContentPreview(long, language.English)
	runes := []rune(preview)
	if len(runes) != browseContentPreviewRunes+1 {
		t.Fatalf("preview length = %d, want %d (truncated text + ellipsis)", len(runes), browseContentPreviewRunes+1)
	}
	if runes[len(runes)-1] != '…' {
		t.Errorf("expected preview to end with an ellipsis, got %q", preview)
	}

	short := "short text"
	if describeContentPreview(short, language.English) != short {
		t.Error("expected short content to be returned unchanged, without truncation")
	}

	if got := describeContentPreview("", language.Estonian); got != "(no translation for et)" {
		t.Errorf("describeContentPreview(\"\", et) = %q, want %q", got, "(no translation for et)")
	}
}

func TestBrowseModelWindowSizeMsg(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/browsesizetest")
	defer cleanup()

	m := newBrowseModel(sess, false)
	newM, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd != nil {
		t.Error("expected no command from a resize")
	}
	if newM.list.Width() != 100 {
		t.Errorf("list width = %d, want 100", newM.list.Width())
	}
}
