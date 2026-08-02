// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
)

func TestL10nTUIConstructor(t *testing.T) {
	cmd := l10nTUI()
	if err := cmd.Err(); err != nil {
		t.Fatalf("l10nTUI() returned error: %v", err)
	}
	if cmd.Name() != "tui" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "tui")
	}
}

// runTUIHeadless runs a real (but headless - no real terminal/input
// needed) Bubble Tea program against m, exactly the pattern
// lib/taskrunner/executor_test.go's runHeadless uses, via this package's
// own newProgram indirection (see tui.go) rather than tea.NewProgram
// directly - proving that indirection point actually composes with a
// substituted headless program the way l10nTUI's Do action relies on it
// to for real use.
func runTUIHeadless(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	restore := newProgram
	defer func() { newProgram = restore }()

	var p *tea.Program
	newProgram = func(m tea.Model, opts ...tea.ProgramOption) *tea.Program {
		opts = append(opts, tea.WithInput(nil), tea.WithoutRenderer())
		p = tea.NewProgram(m, opts...)
		return p
	}

	prog := newProgram(m)
	done := make(chan tea.Model, 1)
	go func() {
		fm, err := prog.Run()
		testutils.NoError(t, err)
		done <- fm
	}()
	prog.Send(tea.KeyPressMsg{Text: "ctrl+c", Mod: tea.ModCtrl, Code: 'c'})

	select {
	case fm := <-done:
		return fm
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the headless TUI program to exit")
		return nil
	}
}

func TestTUIRootModelRunsHeadlessAndQuitsCleanly(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/tuiheadlesstest")
	defer cleanup()

	m := newTUIRootModel(sess, language.Und, false)
	fm := runTUIHeadless(t, m)
	if _, ok := fm.(tuiRootModel); !ok {
		t.Fatalf("expected the final model to be a tuiRootModel, got %T", fm)
	}
}
