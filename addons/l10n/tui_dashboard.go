// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

const dashboardBarWidth = 24

// dashboardRow is one language's translation coverage. It's recomputed
// wholesale by refresh rather than patched incrementally, since a single
// RegisterTranslation call (see saveTranslation) can shift both the
// language's own numbers and, when --with-deps toggles, which keys are
// even in scope.
type dashboardRow struct {
	Language   language.Tag
	Total      int
	Translated int
	Missing    int
	Percentage float64
}

// dashboardModel is the `l10n tui`'s landing tab: a read-only, more
// visual analog of `l10n report`, built from the exact same report data
// (i18n.GetTranslationReport, filterReportByDependencies) so its numbers
// never drift from the CLI command's. It renders first/by default because
// translation coverage - "how much is left to do" - is the first thing
// worth seeing before jumping into the editor.
type dashboardModel struct {
	sess     *session.Context
	withDeps bool
	rows     []dashboardRow
	bar      progress.Model
	width    int
}

func newDashboardModel(sess *session.Context, withDeps bool) dashboardModel {
	m := dashboardModel{
		sess:     sess,
		withDeps: withDeps,
		bar: progress.New(
			progress.WithoutPercentage(),
			progress.WithWidth(dashboardBarWidth),
		),
		width: 80,
	}
	m.Invalidate()
	return m
}

// Invalidate recomputes every row from the current i18n/report state. It's
// called on the dashboard's own toggle keys and whenever a translation is
// saved elsewhere in the program (see tui_root.go) - the dashboard never
// caches a row across those events, since pkg/i18n.RegisterTranslation
// applies immediately and a stale dashboard would misreport coverage right
// after the save that was meant to improve it.
func (m *dashboardModel) Invalidate() {
	m.rows = computeDashboardRows(m.sess, m.withDeps)
}

func computeDashboardRows(sess *session.Context, withDeps bool) []dashboardRow {
	appSupportedLangs := getAppSupportedLanguages(sess)
	fallbackLang := i18n.GetFallbackLanguage()

	var languages []language.Tag
	if len(appSupportedLangs) > 0 {
		languages = append(languages, appSupportedLangs...)
	} else {
		languages = append(languages, i18n.GetLanguages()...)
		sort.Slice(languages, func(i, j int) bool {
			return languages[i].String() < languages[j].String()
		})
	}

	deps, err := getDependencyIdentifiers(sess)
	if err != nil {
		deps = map[string]bool{}
	}

	rows := make([]dashboardRow, 0, len(languages))
	for _, lang := range languages {
		report := i18n.GetTranslationReport(lang)
		if !withDeps {
			report = filterReportByDependencies(sess, report, deps, false)
		}
		if lang == fallbackLang {
			// The fallback language is the source of truth for every key by
			// definition - see report.go's identical treatment.
			report.Percentage = 100.0
			report.Translated = report.Total
			report.Missing = 0
		}
		rows = append(rows, dashboardRow{
			Language:   lang,
			Total:      report.Total,
			Translated: report.Translated,
			Missing:    report.Missing,
			Percentage: report.Percentage,
		})
	}
	return rows
}

func (m dashboardModel) Init() tea.Cmd {
	return nil
}

func (m dashboardModel) Update(msg tea.Msg) (dashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "d" {
			m.withDeps = !m.withDeps
			m.Invalidate()
		}
	}
	return m, nil
}

func (m dashboardModel) View() tea.View {
	var b strings.Builder

	depsLabel := "app only"
	if m.withDeps {
		depsLabel = "app + dependencies"
	}
	fmt.Fprintf(&b, "Translation coverage (%s) - press d to toggle\n\n", depsLabel)

	if len(m.rows) == 0 {
		b.WriteString("No languages registered.\n")
		return tea.NewView(b.String())
	}

	longestLang := 4
	for _, row := range m.rows {
		if l := len(row.Language.String()); l > longestLang {
			longestLang = l
		}
	}

	for _, row := range m.rows {
		bar := m.bar.ViewAs(row.Percentage / 100)
		style := coverageStyle(row.Percentage)
		fmt.Fprintf(&b, "%-*s  %s  %s  %d/%d translated (%d missing)\n",
			longestLang, row.Language.String(),
			bar,
			style.Render(fmt.Sprintf("%5.1f%%", row.Percentage)),
			row.Translated, row.Total, row.Missing,
		)
	}

	return tea.NewView(b.String())
}
