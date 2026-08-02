// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

// translateModel is a read-only inspector for one translation key at a
// time: its current text in every locale, and - the whole reason this tab
// exists rather than just showing the same summary Browse already does -
// every piece of translator-facing detail the schema carries but T/TL
// itself never needs: a *Message's fragments and every one of their CLDR
// forms (not just the one that happens to render for a given argument),
// each documented argument's type and description, and the owning
// bundle's translator note for the locale in view. Editing/saving is
// intentionally not implemented here for now - see saveTranslation in
// translate.go, still used by the CLI's own `l10n translate` prompt,
// which this tab deliberately never calls.
//
// It walks an ordered, missing-first worklist for the current language
// (the exact same traversal interactiveMode in translate.go already
// performs, reused via translateWorklist rather than reimplemented), or -
// after a jump from the Browse tab (see jumpToKey) - inspects one specific
// key regardless of its translation status.
type translateModel struct {
	sess     *session.Context
	withDeps bool

	lang  language.Tag
	langs []language.Tag

	worklist []i18n.TranslationEntry
	cursor   int

	// manualEntry is non-nil while inspecting a key chosen via the Browse
	// tab's enter key, overriding the missing-worklist traversal until esc
	// clears it.
	manualEntry *i18n.TranslationEntry

	viewport viewport.Model
	errMsg   string
}

func newTranslateModel(sess *session.Context, lang language.Tag, withDeps bool) translateModel {
	langs := tuiTranslatableLanguages(sess)
	if lang == language.Und && len(langs) > 0 {
		lang = langs[0]
	}

	m := translateModel{
		sess:     sess,
		withDeps: withDeps,
		lang:     lang,
		langs:    langs,
		viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(15)),
	}
	m.refreshWorklist()
	m.refreshViewportContent()
	return m
}

// tuiTranslatableLanguages resolves the languages the translate/dashboard/
// browse views cycle through: the app's own configured supported
// languages if set, otherwise every registered i18n language except the
// fallback (which is definitionally always "translated"). This mirrors the
// identical language-resolution block repeated in translate.go's
// interactiveMode/translateSpecificKeyInteractive and report.go/list.go -
// duplicated here rather than factored out from those, since the CLI
// commands' own logic is deliberately left untouched by this change (see
// this session's translate.go refactor, which is a pure behavior-preserving
// split, not a wider dedup pass).
func tuiTranslatableLanguages(sess *session.Context) []language.Tag {
	if appLangs := getAppSupportedLanguages(sess); len(appLangs) > 0 {
		return appLangs
	}
	fallbackLang := i18n.GetFallbackLanguage()
	var langs []language.Tag
	for _, lang := range i18n.GetLanguages() {
		if lang != fallbackLang {
			langs = append(langs, lang)
		}
	}
	return langs
}

// translateWorklist returns lang's missing application (or, with withDeps,
// application-and-dependency) entries, in the same order and filtered the
// same way interactiveMode's loop body already does - see that function in
// translate.go. Reused rather than reimplemented so the TUI and the CLI's
// interactive prompt can never disagree about what counts as "missing".
func translateWorklist(sess *session.Context, lang language.Tag, withDeps bool) ([]i18n.TranslationEntry, error) {
	report := i18n.GetTranslationReport(lang)
	deps, err := getDependencyIdentifiers(sess)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency identifiers: %w", err)
	}

	items := make([]i18n.TranslationEntry, 0, len(report.MissingEntries))
	for _, entry := range report.MissingEntries {
		isDep, err := isDependencyKeyForEntry(entry.Key, deps)
		if err != nil {
			return nil, err
		}
		if !withDeps && isDep {
			continue
		}
		items = append(items, entry)
	}
	return items, nil
}

// refreshWorklist recomputes the worklist for the current language/deps
// filter. Called on language/deps changes - never cached across those
// events, since a filter/language change can move entries in or out of the
// "missing" set.
func (m *translateModel) refreshWorklist() {
	items, err := translateWorklist(m.sess, m.lang, m.withDeps)
	if err != nil {
		m.errMsg = err.Error()
		m.worklist = nil
		return
	}
	m.worklist = items
	if m.cursor >= len(m.worklist) {
		m.cursor = 0
	}
}

// currentEntry returns the entry currently in view: the browse-tab jump
// target if one is active, otherwise the worklist entry at cursor.
func (m translateModel) currentEntry() (i18n.TranslationEntry, bool) {
	if m.manualEntry != nil {
		return *m.manualEntry, true
	}
	if len(m.worklist) == 0 || m.cursor >= len(m.worklist) {
		return i18n.TranslationEntry{}, false
	}
	return m.worklist[m.cursor], true
}

// refreshViewportContent rebuilds the inspector pane for currentEntry/
// m.lang. Called after every event that changes which entry or language is
// in view (cursor/lang/manualEntry/worklist changes).
func (m *translateModel) refreshViewportContent() {
	m.viewport.SetContent(buildInspectContent(m.lang, m.currentEntry))
	m.viewport.SetYOffset(0)
}

// entrySourceLang reports the authoritative source locale for entry: its
// owning bundle's own declared source (see i18n.GetKeyBundle,
// GetBundleSourceLanguage) if it has one, or the application's global
// fallback language otherwise (a legacy/unbundled key, or one whose bundle
// predates schema version 2, has no bundle-declared source of its own).
// This is deliberately not always i18n.GetFallbackLanguage(): an imported
// happy-compatible package ships its own bundle, which may declare a
// source locale different from the embedding application's own fallback -
// entrySourceLang is what makes the inspector show that package's actual
// source text and notes instead of the application's, whichever locale
// that happens to be.
func entrySourceLang(entry i18n.TranslationEntry) language.Tag {
	if bundle, ok := i18n.GetKeyBundle(entry.Key); ok {
		if src, ok := i18n.GetBundleSourceLanguage(bundle); ok {
			return src
		}
	}
	return i18n.GetFallbackLanguage()
}

// buildInspectContent renders every schema-level detail available for
// entry (as returned by currentEntryFn) in lang: its source text (see
// entrySourceLang) and, if different, its text in lang; its owning
// bundle's declared source locale and any translator note left for lang
// (see i18n.GetKeyBundle, GetBundleSourceLanguage, GetBundleNote); and -
// when the registered value is a rich *Message (see i18n.GetMessage) - its
// translator-facing description, every fragment's every CLDR/select form
// (not just whichever one happens to render for a given argument value),
// and each documented argument's derived type, example value, and
// description. This is the entire point of the schema's optional
// description/fragment/notes fields existing at all: surfacing them to
// whoever is about to translate a key, not just to T/TL's own rendering.
func buildInspectContent(lang language.Tag, currentEntryFn func() (i18n.TranslationEntry, bool)) string {
	entry, ok := currentEntryFn()
	if !ok {
		return "Nothing to inspect for this language/filter - press ctrl+left/right to switch language or ctrl+d to include dependencies."
	}

	fallbackLang := i18n.GetFallbackLanguage()
	sourceLang := entrySourceLang(entry)
	bundle, hasBundle := i18n.GetKeyBundle(entry.Key)
	var b strings.Builder

	fmt.Fprintf(&b, "Key: %s\n", entry.Key)
	if hasBundle {
		fmt.Fprintf(&b, "Bundle: %s\n", bundle)
	}
	fmt.Fprintf(&b, "Source locale: %s\n", sourceLang.String())
	if note, ok := i18n.GetBundleNote(bundle, lang); ok {
		fmt.Fprintf(&b, "\nTranslator note (%s): %s\n", lang.String(), note)
	}

	b.WriteString("\n")
	if sourceText := contentTextForLang(entry, sourceLang, fallbackLang); sourceText != "" {
		fmt.Fprintf(&b, "Source (%s): %s\n", sourceLang.String(), sourceText)
	} else {
		fmt.Fprintf(&b, "Source (%s): (missing)\n", sourceLang.String())
	}
	if lang != sourceLang {
		if content := contentTextForLang(entry, lang, fallbackLang); content != "" {
			fmt.Fprintf(&b, "%s: %s\n", lang.String(), content)
		} else {
			fmt.Fprintf(&b, "%s: (missing)\n", lang.String())
		}
	}

	msg, hasMessage := i18n.GetMessage(lang, entry.Key)
	if hasMessage {
		writeMessageSchema(&b, msg)
	}

	if types, ok := i18n.GetMessageArgTypes(lang, entry.Key); ok && len(types) > 0 {
		writeArgumentDetails(&b, types, msg)
	}

	return b.String()
}

// writeMessageSchema appends msg's translator-facing description and every
// fragment's forms to b - the detail a plain rendered string can never
// show, since rendering only ever picks one form per fragment for one
// concrete argument value.
func writeMessageSchema(b *strings.Builder, msg *i18n.Message) {
	b.WriteString("\n--- Message schema ---\n")
	if msg.Description != nil && msg.Description.Msg != "" {
		fmt.Fprintf(b, "Description: %s\n", msg.Description.Msg)
	}
	if len(msg.Fragments) == 0 {
		return
	}

	names := make([]string, 0, len(msg.Fragments))
	for name := range msg.Fragments {
		names = append(names, name)
	}
	sort.Strings(names)

	b.WriteString("\nFragments:\n")
	for _, name := range names {
		frag := msg.Fragments[name]
		fmt.Fprintf(b, "  %s (%s, arg: %s)\n", name, frag.Kind, frag.Arg)
		forms := make([]string, 0, len(frag.Forms))
		for form := range frag.Forms {
			forms = append(forms, form)
		}
		sort.Strings(forms)
		for _, form := range forms {
			fmt.Fprintf(b, "    %-6s %s\n", form+":", frag.Forms[form])
		}
	}
}

// writeArgumentDetails appends every interpolated argument's derived type
// (see i18n.ArgType), a representative example value, and - when msg
// documents it - its translator-facing description. types comes from
// i18n.GetMessageArgTypes, which works whether entry's value is a rich
// *Message or a plain {name:verb} string; msg is nil in the latter case,
// so an argument's description is simply omitted rather than looked up.
func writeArgumentDetails(b *strings.Builder, types map[string]i18n.ArgType, msg *i18n.Message) {
	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)

	b.WriteString("\nArguments:\n")
	for _, name := range names {
		argType := types[name]
		fmt.Fprintf(b, "  %s (%s, e.g. %s)", name, argType.String(), argType.ExampleValue())
		if msg != nil && msg.Description != nil {
			if ad, ok := msg.Description.Args[name]; ok && ad.Description != "" {
				fmt.Fprintf(b, " - %s", ad.Description)
			}
		}
		b.WriteString("\n")
	}
}

// jumpToKey switches to inspecting key directly - regardless of its
// translation status - in response to a browseSelectKeyMsg from the
// Browse tab (see tui_root.go). It looks the key up in the live i18n
// registry rather than the current worklist, since the whole point is to
// let the user land on a key the worklist wouldn't otherwise surface
// (e.g. one that's already translated, to review a typo found via
// Browse's content search).
//
// It defaults the displayed language to the key's own source locale (see
// entrySourceLang), not whatever m.lang happened to be left at - landing
// on a key you specifically asked to inspect should show its authoritative
// original text and notes first, not an arbitrary target language that may
// say nothing useful about it yet (e.g. because it's still untranslated
// there).
func (m *translateModel) jumpToKey(key string) {
	for _, entry := range i18n.GetAllTranslations() {
		if entry.Key == key {
			e := entry
			m.manualEntry = &e
			m.lang = entrySourceLang(e)
			m.errMsg = ""
			m.refreshViewportContent()
			return
		}
	}
}

// languagesToCycle returns the language cycle ctrl+left/right moves
// through: m.langs (the app's configured/registered target languages)
// while paging the missing-first worklist, since that mode's whole point
// is choosing which target language's worklist to browse - or every
// registered language, source included, while inspecting one specific key
// jumped to from Browse (see jumpToKey and manualEntry). The latter
// matters because a jumped-to key's own source locale may not be one of
// the app's configured target languages at all - e.g. a dependency bundle
// authored in a locale the application itself never translates into -
// yet it must still be reachable so its source text/notes stay visible.
func (m translateModel) languagesToCycle() []language.Tag {
	if m.manualEntry != nil {
		return browseContentLanguages()
	}
	return m.langs
}

func (m *translateModel) cycleLang(dir int) {
	langs := m.languagesToCycle()
	if len(langs) == 0 {
		return
	}
	idx := 0
	for i, l := range langs {
		if l == m.lang {
			idx = i
			break
		}
	}
	idx = ((idx+dir)%len(langs) + len(langs)) % len(langs)
	m.lang = langs[idx]
	m.refreshWorklist()
	m.refreshViewportContent()
}

func (m translateModel) Init() tea.Cmd {
	return nil
}

func (m translateModel) Update(msg tea.Msg) (translateModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if w := msg.Width - 4; w > 10 {
			m.viewport.SetWidth(min(w, 120))
		}
		// Leave headroom for tui_root's own tab/help chrome plus this tab's
		// own header/footer lines (see View).
		if h := msg.Height - 8; h > 3 {
			m.viewport.SetHeight(h)
		}
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.manualEntry != nil {
				m.manualEntry = nil
				m.errMsg = ""
				m.refreshWorklist()
				m.refreshViewportContent()
			}
			return m, nil
		case "ctrl+n":
			if len(m.worklist) > 0 {
				m.cursor = (m.cursor + 1) % len(m.worklist)
				m.refreshViewportContent()
			}
			return m, nil
		case "ctrl+p":
			if len(m.worklist) > 0 {
				m.cursor = (m.cursor - 1 + len(m.worklist)) % len(m.worklist)
				m.refreshViewportContent()
			}
			return m, nil
		case "ctrl+right":
			m.cycleLang(1)
			return m, nil
		case "ctrl+left":
			m.cycleLang(-1)
			return m, nil
		case "ctrl+d":
			m.withDeps = !m.withDeps
			m.refreshWorklist()
			m.refreshViewportContent()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m translateModel) View() tea.View {
	var b strings.Builder

	depsLabel := ""
	if m.withDeps {
		depsLabel = "  [+deps]"
	}
	jumpLabel := ""
	if m.manualEntry != nil {
		jumpLabel = "  [inspecting selected key - esc to return to worklist]"
	}
	fmt.Fprintf(&b, "Language: %s%s%s  (read-only preview)\n\n", m.lang.String(), depsLabel, jumpLabel)

	b.WriteString(m.viewport.View())

	if len(m.worklist) > 0 && m.manualEntry == nil {
		fmt.Fprintf(&b, "\n\n%d/%d missing\n", m.cursor+1, len(m.worklist))
	}

	if m.errMsg != "" {
		b.WriteString("\n" + errorStyle.Render("error: "+m.errMsg) + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("↑/↓ scroll | ctrl+n/ctrl+p next/prev | ctrl+left/right language | ctrl+d toggle deps | esc cancel jump"))

	return tea.NewView(b.String())
}
