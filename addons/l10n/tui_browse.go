// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

// browseSelectKeyMsg is sent when the user presses enter on a browsed key,
// asking tui_root.go to switch the active tab to Translate with that key
// pre-seeded (see translateModel.jumpToKey).
type browseSelectKeyMsg struct {
	Key string
}

// browseSearchMode selects what browseItem.FilterValue() (and so bubbles/v2/
// list's built-in fuzzy filter) matches against: the translation key itself,
// or the registered translation text in some locale (see browseModel.
// contentLang). See browseModel's doc comment for why both live in one tab.
type browseSearchMode int

const (
	// browseSearchByKey is the original Browse behavior: the filter matches
	// translation keys, for a user who already knows (part of) the key
	// they're after.
	browseSearchByKey browseSearchMode = iota
	// browseSearchByContent flips the filter to match the raw, authored
	// translation text in contentLang instead - for a user who spotted a
	// rendered string (e.g. a typo in some locale's UI copy) and needs to
	// find the key that produced it, without grepping the codebase.
	browseSearchByContent
)

// browseContentPreviewRunes caps how much of an entry's content-mode
// Description() preview is shown - long enough to recognize the match at a
// glance, short enough to keep the list's two-line delegate readable.
const browseContentPreviewRunes = 60

// browseItem adapts a key's translation status to bubbles/v2/list's Item
// (for filtering) and DefaultItem (for the default delegate's two-line
// rendering) interfaces. filterValue and description are precomputed by
// buildBrowseItems for the search mode/locale in effect at build time (see
// browseSearchMode) rather than derived here, since Title/Description/
// FilterValue take no arguments and so cannot themselves know which mode is
// active.
type browseItem struct {
	key         string
	translated  []string
	missing     []string
	filterValue string
	description string
}

func (i browseItem) FilterValue() string { return i.filterValue }
func (i browseItem) Title() string       { return i.key }
func (i browseItem) Description() string { return i.description }

// describeTranslationStatus renders browseSearchByKey's Description(): a
// one-line summary of which languages a key has (or is missing).
func describeTranslationStatus(translated, missing []string) string {
	if len(missing) == 0 {
		return "translated: " + strings.Join(translated, ", ")
	}
	return "missing: " + strings.Join(missing, ", ")
}

// describeContentPreview renders browseSearchByContent's Description(): a
// truncated preview of the raw text a key's FilterValue() was matched
// against, so the user can visually confirm the hit before pressing enter.
// content is empty for a key with nothing registered in lang - itself
// diagnostic (see buildBrowseItems), so it gets a distinct placeholder
// rather than reading as an empty/blank match.
func describeContentPreview(content string, lang language.Tag) string {
	if content == "" {
		return fmt.Sprintf("(no translation for %s)", lang.String())
	}
	runes := []rune(content)
	if len(runes) > browseContentPreviewRunes {
		return string(runes[:browseContentPreviewRunes]) + "…"
	}
	return content
}

// contentTextForLang returns entry's raw, authored translation text for
// lang - the literal template a translator typed (see
// i18n.TranslationEntry.Translations and Message.String()), never a T/TL-
// rendered value. A rendered value would run rich-message argument
// interpolation against no concrete arguments, injecting fmt's
// "%!name(MISSING)"-style noise into exactly the text search is meant to
// match. entry.Translations only holds non-fallback languages - the
// fallback's text lives in entry.Fallback instead (see
// (*manager).getAllTranslations) - so the fallback case is handled
// separately here rather than falling through to a map miss.
func contentTextForLang(entry i18n.TranslationEntry, lang, fallbackLang language.Tag) string {
	if lang == fallbackLang {
		return entry.Fallback
	}
	return entry.Translations[lang]
}

// browseContentLanguages returns every registered language (i18n.
// GetLanguages()), sorted alphabetically by tag string - the same
// deterministic ordering computeDashboardRows falls back to when the app
// declares no explicit supported-languages list - so browseModel's `[`/`]`
// cycling lands on a stable, predictable sequence rather than whatever
// iteration order the i18n manager's internal map happens to produce.
func browseContentLanguages() []language.Tag {
	langs := append([]language.Tag(nil), i18n.GetLanguages()...)
	sort.Slice(langs, func(i, j int) bool {
		return langs[i].String() < langs[j].String()
	})
	return langs
}

// browseListTitle renders list.Model's own header line with the current
// search mode/locale and the hotkeys that change them - the same
// discoverability convention dashboardModel.View() uses for its own
// "press d to toggle" line, adapted here to bubbles/v2/list's Title field
// since Browse's header is rendered by the list itself rather than a custom
// View().
func browseListTitle(mode browseSearchMode, contentLang language.Tag) string {
	if mode == browseSearchByContent {
		return fmt.Sprintf("Translation Keys - search: content [%s] (s: key, [ ]: prev/next locale)", contentLang.String())
	}
	return "Translation Keys - search: key (s: content, [ ]: prev/next locale)"
}

// browseModel is a read-only, filterable view over every translation key
// (i18n.GetAllTranslations()), built on the exact same
// processEntriesForDisplay helper `l10n list` itself uses so the two never
// disagree about what's "missing". Its d/m toggles mirror list's
// --with-deps/--missing flags; its fuzzy filter (bubbles/v2/list's built-in
// "/") replaces list's --lang grep, since browsing is meant for scanning
// interactively rather than scripting.
//
// searchMode/contentLang add a second way to drive that same filter: rather
// than a separate "Search" tab, a user who spotted a typo in some rendered
// locale's text flips search modes (s) to match on that locale's translated
// content instead of the key, optionally cycling which locale with [ and ] -
// deliberately not "/", which the list's own filter already owns - still
// landing on enter jumping to the same Translate-tab key as key-mode search
// does. contentLang defaults to i18n.GetFallbackLanguage(), the
// application's own default/source locale.
type browseModel struct {
	sess        *session.Context
	list        list.Model
	withDeps    bool
	missingOnly bool
	searchMode  browseSearchMode
	contentLang language.Tag
}

func newBrowseModel(sess *session.Context, withDeps bool) browseModel {
	m := browseModel{
		sess:        sess,
		withDeps:    withDeps,
		contentLang: i18n.GetFallbackLanguage(),
		list:        list.New(nil, list.NewDefaultDelegate(), 0, 0),
	}
	m.list.SetShowStatusBar(true)
	m.list.SetStatusBarItemName("key", "keys")
	m.Invalidate()
	return m
}

// Invalidate rebuilds the list's items and header from the current i18n
// state. Called on this view's own d/m/s/[/] toggles (d, m, s, [, and ]) and
// whenever a translation is saved elsewhere in the program (see
// tui_root.go), for the same lazy-recompute reason documented on
// dashboardModel.Invalidate.
func (m *browseModel) Invalidate() {
	m.list.SetItems(buildBrowseItems(m.sess, m.withDeps, m.missingOnly, m.searchMode, m.contentLang))
	m.list.Title = browseListTitle(m.searchMode, m.contentLang)
}

// cycleContentLang moves contentLang forward (delta > 0) or backward
// (delta < 0) through browseContentLanguages(), wrapping at either end. A
// contentLang no longer present in that list (e.g. nothing has been
// registered for it) is treated as if it were at index 0, so cycling still
// produces a sensible next/previous locale rather than getting stuck.
func (m *browseModel) cycleContentLang(delta int) {
	langs := browseContentLanguages()
	if len(langs) == 0 {
		return
	}
	idx := 0
	for i, lang := range langs {
		if lang == m.contentLang {
			idx = i
			break
		}
	}
	idx = ((idx+delta)%len(langs) + len(langs)) % len(langs)
	m.contentLang = langs[idx]
}

func buildBrowseItems(sess *session.Context, withDeps, missingOnly bool, mode browseSearchMode, contentLang language.Tag) []list.Item {
	allEntries := i18n.GetAllTranslations()
	deps, err := getDependencyIdentifiers(sess)
	if err != nil {
		deps = map[string]bool{}
	}

	entries := make([]i18n.TranslationEntry, 0, len(allEntries))
	for _, entry := range allEntries {
		isDep, ierr := isDependencyKeyForEntry(entry.Key, deps)
		if ierr != nil {
			continue
		}
		if !withDeps && isDep {
			continue
		}
		entries = append(entries, entry)
	}

	appSupportedLangs := getAppSupportedLanguages(sess)
	fallbackLang := i18n.GetFallbackLanguage()
	var translatableLangs []language.Tag
	if len(appSupportedLangs) > 0 {
		translatableLangs = appSupportedLangs
	} else {
		for _, lang := range i18n.GetLanguages() {
			if lang != fallbackLang {
				translatableLangs = append(translatableLangs, lang)
			}
		}
	}

	statuses := processEntriesForDisplay(entries, translatableLangs, language.Und, fallbackLang, missingOnly)

	var byKey map[string]i18n.TranslationEntry
	if mode == browseSearchByContent {
		byKey = make(map[string]i18n.TranslationEntry, len(entries))
		for _, entry := range entries {
			byKey[entry.Key] = entry
		}
	}

	items := make([]list.Item, len(statuses))
	for i, s := range statuses {
		item := browseItem{key: s.Key, translated: s.TranslatedLangs, missing: s.MissingLangs}
		if mode == browseSearchByContent {
			content := contentTextForLang(byKey[s.Key], contentLang, fallbackLang)
			item.filterValue = content
			item.description = describeContentPreview(content, contentLang)
		} else {
			item.filterValue = s.Key
			item.description = describeTranslationStatus(s.TranslatedLangs, s.MissingLangs)
		}
		items[i] = item
	}
	return items
}

func (m browseModel) Init() tea.Cmd {
	return nil
}

func (m browseModel) Update(msg tea.Msg) (browseModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, max(msg.Height-2, 1))
		return m, nil
	case tea.KeyPressMsg:
		// Only treat these as browse hotkeys outside of filter text entry -
		// the list's own "/" filter needs to accept any character,
		// including these, as part of the query being typed.
		if !m.list.SettingFilter() {
			switch msg.String() {
			case "d":
				m.withDeps = !m.withDeps
				m.Invalidate()
				return m, nil
			case "m":
				m.missingOnly = !m.missingOnly
				m.Invalidate()
				return m, nil
			case "s":
				if m.searchMode == browseSearchByKey {
					m.searchMode = browseSearchByContent
				} else {
					m.searchMode = browseSearchByKey
				}
				m.Invalidate()
				return m, nil
			case "[":
				m.cycleContentLang(-1)
				m.Invalidate()
				return m, nil
			case "]":
				m.cycleContentLang(1)
				m.Invalidate()
				return m, nil
			case "enter":
				if item, ok := m.list.SelectedItem().(browseItem); ok {
					key := item.key
					return m, func() tea.Msg { return browseSelectKeyMsg{Key: key} }
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m browseModel) View() tea.View {
	return tea.NewView(m.list.View())
}
