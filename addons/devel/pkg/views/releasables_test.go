// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/addons/devel/pkg/gomodule"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func testPackages() []*gomodule.Package {
	return []*gomodule.Package{
		{Import: "github.com/happy-sdk/happy", NeedsRelease: true, LastReleaseTag: "happy/v1.0.0", NextReleaseTag: "happy/v1.1.0"},
		{Import: "github.com/happy-sdk/happy/pkg/foo", NeedsRelease: false, LastReleaseTag: "pkg/foo/v0.1.0", NextReleaseTag: "pkg/foo/v0.1.0"},
		{Import: "github.com/happy-sdk/happy/pkg/new", FirstRelease: true, NeedsRelease: true, NextReleaseTag: "pkg/new/v0.1.0"},
	}
}

// TestGetConfirmReleasablesViewRendersRows is a regression test: the table's
// viewport requires an explicit width (table.WithWidth) in bubbles v2 -
// without it, viewport.visibleLines() returns nil unconditionally whenever
// maxWidth is 0, silently dropping every row while the header/border still
// render fine. That made the bug invisible without actually inspecting
// rendered content.
func TestGetConfirmReleasablesViewRendersRows(t *testing.T) {
	m, err := GetConfirmReleasablesView(nil, testPackages())
	testutils.NoError(t, err)

	content := m.View().Content

	for _, want := range []string{
		"github.com/happy-sdk/happy",
		"github.com/happy-sdk/happy/pkg/foo",
		"github.com/happy-sdk/happy/pkg/new",
		"release",
		"skip",
		"initial",
	} {
		testutils.Assert(t, strings.Contains(content, want), "expected rendered view to contain %q, got:\n%s", want, content)
	}
}

func TestGetConfirmReleasablesViewNoPackages(t *testing.T) {
	m, err := GetConfirmReleasablesView(nil, nil)
	testutils.NoError(t, err)
	testutils.Assert(t, m.View().Content != "", "expected a non-empty view even with no packages (headers/prompt still render)")
}

func TestConfirmReleasablesViewInit(t *testing.T) {
	m, _ := GetConfirmReleasablesView(nil, testPackages())
	testutils.Assert(t, m.Init() == nil, "expected Init to return no initial command")
}

func TestConfirmReleasablesViewUpdateConfirm(t *testing.T) {
	tests := []struct {
		key     string
		wantYes bool
	}{
		{"y", true},
		{"Y", true},
		{"n", false},
		{"N", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			m, _ := GetConfirmReleasablesView(nil, testPackages())
			newM, cmd := m.Update(tea.KeyPressMsg{Text: tt.key, Code: rune(tt.key[0])})
			nm := newM.(ConfirmReleasablesView)

			testutils.Assert(t, nm.answered, "expected answered to be true")
			testutils.Equal(t, tt.wantYes, nm.Yes, "unexpected Yes value")
			testutils.Assert(t, cmd != nil, "expected a quit command")
			_, isQuit := cmd().(tea.QuitMsg)
			testutils.Assert(t, isQuit, "expected tea.QuitMsg")
		})
	}
}

func TestConfirmReleasablesViewUpdateInvalidKeySetsErr(t *testing.T) {
	m, _ := GetConfirmReleasablesView(nil, testPackages())
	newM, cmd := m.Update(tea.KeyPressMsg{Text: "z", Code: 'z'})
	nm := newM.(ConfirmReleasablesView)

	testutils.Assert(t, !nm.answered, "expected an unrecognized key not to answer the prompt")
	testutils.Assert(t, cmd == nil, "expected no command for an unrecognized key")
	testutils.Assert(t, strings.Contains(nm.err, "z"), "expected err to mention the invalid key, got %q", nm.err)
	testutils.Assert(t, strings.Contains(nm.View().Content, nm.err), "expected the error to show up in the rendered view")
}

func TestConfirmReleasablesViewViewAfterAnswered(t *testing.T) {
	m, _ := GetConfirmReleasablesView(nil, testPackages())
	newM, _ := m.Update(tea.KeyPressMsg{Text: "y", Code: 'y'})
	nm := newM.(ConfirmReleasablesView)

	v := nm.View()
	testutils.Equal(t, "", v.Content, "expected an empty view once answered")
	testutils.Assert(t, v.AltScreen, "expected AltScreen to still be requested")
}
