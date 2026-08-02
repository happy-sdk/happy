// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/text/language"
)

func newTUITestModel(t *testing.T) tuiRootModel {
	t.Helper()
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/tuiroottest")
	t.Cleanup(cleanup)
	return newTUIRootModel(sess, language.Und, false)
}

func TestTuiRootModelStartsOnDashboard(t *testing.T) {
	m := newTUITestModel(t)
	if m.active != tabDashboard {
		t.Errorf("active = %v, want tabDashboard", m.active)
	}
}

func TestTuiRootModelTabCyclesForwardAndBackward(t *testing.T) {
	m := newTUITestModel(t)

	newM, cmd := m.Update(tea.KeyPressMsg{Text: "tab", Code: tea.KeyTab})
	if cmd != nil {
		t.Error("expected no command from switching tabs")
	}
	rm := newM.(tuiRootModel)
	if rm.active != tabTranslate {
		t.Errorf("active = %v, want tabTranslate after one tab press", rm.active)
	}

	newM, _ = rm.Update(tea.KeyPressMsg{Text: "tab", Code: tea.KeyTab})
	rm = newM.(tuiRootModel)
	if rm.active != tabBrowse {
		t.Errorf("active = %v, want tabBrowse after two tab presses", rm.active)
	}

	newM, _ = rm.Update(tea.KeyPressMsg{Text: "shift+tab", Mod: tea.ModShift, Code: tea.KeyTab})
	rm = newM.(tuiRootModel)
	if rm.active != tabTranslate {
		t.Errorf("active = %v, want tabTranslate after shift+tab", rm.active)
	}
}

func TestTuiRootModelCtrlCAlwaysQuits(t *testing.T) {
	m := newTUITestModel(t)
	_, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+c", Mod: tea.ModCtrl, Code: 'c'})
	if cmd == nil {
		t.Fatal("expected ctrl+c to produce a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected ctrl+c to produce tea.Quit")
	}
}

// TestTuiRootModelQAlwaysQuits is a regression test for the "Translate" tab
// no longer being an editable text field: q used to need special-casing
// there (typing the letter q into a translation value must not quit the
// program), but since that tab is read-only now (see translateModel's doc
// comment), q quits from every tab exactly like ctrl+c.
func TestTuiRootModelQAlwaysQuits(t *testing.T) {
	m := newTUITestModel(t)

	for _, tab := range []tuiTab{tabDashboard, tabTranslate, tabBrowse} {
		m.active = tab
		_, cmd := m.Update(tea.KeyPressMsg{Text: "q", Code: 'q'})
		if cmd == nil {
			t.Fatalf("expected q to quit on the %s tab", tab)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("expected q to produce tea.Quit on the %s tab", tab)
		}
	}
}

func TestTuiRootModelBrowseSelectSwitchesToTranslate(t *testing.T) {
	m := newTUITestModel(t)
	m.active = tabBrowse

	newM, cmd := m.Update(browseSelectKeyMsg{Key: "com.github.happy-sdk.tuiroottest.some.key"})
	if cmd != nil {
		t.Error("expected no command from a browse selection")
	}
	rm := newM.(tuiRootModel)
	if rm.active != tabTranslate {
		t.Errorf("active = %v, want tabTranslate after a browse selection", rm.active)
	}
}

func TestTuiRootModelWindowSizeForwardsToAllTabs(t *testing.T) {
	m := newTUITestModel(t)
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	rm := newM.(tuiRootModel)
	if rm.browse.list.Width() != 100 {
		t.Errorf("expected browse tab to receive the resize, width = %d", rm.browse.list.Width())
	}
}

func TestTuiRootModelViewRendersTabsAndActiveBody(t *testing.T) {
	m := newTUITestModel(t)
	view := m.View()
	if !view.AltScreen {
		t.Error("expected the root view to request the alt screen")
	}
	plain := ansi.Strip(view.Content)
	for _, want := range []string{"Dashboard", "Translate", "Browse"} {
		if !strings.Contains(plain, want) {
			t.Errorf("expected view to mention tab %q, got:\n%s", want, plain)
		}
	}
}

func TestNextTuiTabWraps(t *testing.T) {
	if got := nextTuiTab(tabBrowse, 1); got != tabDashboard {
		t.Errorf("nextTuiTab(tabBrowse, 1) = %v, want tabDashboard", got)
	}
	if got := nextTuiTab(tabDashboard, -1); got != tabBrowse {
		t.Errorf("nextTuiTab(tabDashboard, -1) = %v, want tabBrowse", got)
	}
}
