// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

// tuiTab identifies which of `l10n tui`'s three views is active.
type tuiTab int

const (
	// tabDashboard is the default/landing tab: translation coverage is the
	// first thing worth seeing before digging into individual keys (see
	// dashboardModel's doc comment).
	tabDashboard tuiTab = iota
	tabTranslate
	tabBrowse
	tuiTabCount
)

func (t tuiTab) String() string {
	switch t {
	case tabDashboard:
		return "Dashboard"
	case tabTranslate:
		return "Translate"
	case tabBrowse:
		return "Browse"
	default:
		return "?"
	}
}

// nextTuiTab advances t by dir (+1 or -1) tabs, wrapping around.
func nextTuiTab(t tuiTab, dir int) tuiTab {
	n := int(tuiTabCount)
	return tuiTab(((int(t)+dir)%n + n) % n)
}

// tuiRootModel is the top-level Bubble Tea model for `l10n tui`: a tab
// container that owns one sub-model per view and routes messages to
// whichever is active, plus a handful of global keys that make sense
// regardless of which tab is showing.
type tuiRootModel struct {
	active tuiTab

	dashboard dashboardModel
	translate translateModel
	browse    browseModel
}

func newTUIRootModel(sess *session.Context, lang language.Tag, withDeps bool) tuiRootModel {
	return tuiRootModel{
		active:    tabDashboard,
		dashboard: newDashboardModel(sess, withDeps),
		translate: newTranslateModel(sess, lang, withDeps),
		browse:    newBrowseModel(sess, withDeps),
	}
}

func (m tuiRootModel) Init() tea.Cmd {
	return m.translate.Init()
}

func (m tuiRootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Every tab gets sized regardless of which is active, so switching
		// to one never shows it laid out for a stale/zero size.
		var cmd tea.Cmd
		m.dashboard, _ = m.dashboard.Update(msg)
		m.translate, cmd = m.translate.Update(msg)
		m.browse, _ = m.browse.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.active = nextTuiTab(m.active, 1)
			return m, nil
		case "shift+tab":
			m.active = nextTuiTab(m.active, -1)
			return m, nil
		}

	case browseSelectKeyMsg:
		m.translate.jumpToKey(msg.Key)
		m.active = tabTranslate
		return m, nil
	}

	var cmd tea.Cmd
	switch m.active {
	case tabDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case tabTranslate:
		m.translate, cmd = m.translate.Update(msg)
	case tabBrowse:
		m.browse, cmd = m.browse.Update(msg)
	}
	return m, cmd
}

func (m tuiRootModel) View() tea.View {
	var tabs string
	for tab := tuiTab(0); tab < tuiTabCount; tab++ {
		label := tab.String()
		if tab == m.active {
			tabs += tabActiveStyle.Render(label) + "  "
		} else {
			tabs += tabInactiveStyle.Render(label) + "  "
		}
	}

	var body string
	switch m.active {
	case tabDashboard:
		body = m.dashboard.View().Content
	case tabTranslate:
		body = m.translate.View().Content
	case tabBrowse:
		body = m.browse.View().Content
	}

	content := tabs + "\n\n" + body + "\n" +
		helpStyle.Render("tab/shift+tab switch views | q/ctrl+c quit")

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
