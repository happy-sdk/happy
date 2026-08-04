// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func key(s string) tea.KeyPressMsg {
	if len(s) == 1 {
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}
	panic("unhandled key " + s)
}

// send drives the model through a sequence of keys, as the runtime would.
func send(m WorkspaceWizard, keys ...string) WorkspaceWizard {
	var model tea.Model = m
	for _, k := range keys {
		model, _ = model.Update(key(k))
	}
	return model.(WorkspaceWizard)
}

// enterAll walks to the last field and submits, which is how the form is
// completed.
func submit(m WorkspaceWizard) WorkspaceWizard {
	keys := make([]string, 0, len(m.fields))
	for range m.fields {
		keys = append(keys, "enter")
	}
	return send(m, keys...)
}

// A user who passed --org must not be asked for it again with an empty box.
func TestWizardSeedsFromFlags(t *testing.T) {
	seed := WorkspaceAnswers{
		Root:     "/tmp/ws",
		Org:      "acme",
		Remote:   "git@github.com:acme/{repo}.git",
		ReposDir: "src",
		Scratch:  "workspace",
	}
	m := NewWorkspaceWizard(seed)

	if got := m.inputs[0].Value(); got != "/tmp/ws" {
		t.Errorf("root input = %q, want the seeded value", got)
	}
	if got := m.inputs[1].Value(); got != "acme" {
		t.Errorf("org input = %q, want the seeded value", got)
	}
}

func TestWizardSubmitCollectsAnswers(t *testing.T) {
	m := submit(NewWorkspaceWizard(WorkspaceAnswers{
		Root:     "/tmp/ws",
		Org:      "acme",
		Remote:   "git@github.com:acme/{repo}.git",
		Repos:    "alpha,beta",
		ReposDir: "src",
		Scratch:  "workspace",
		Clone:    true,
	}))

	if m.Cancelled {
		t.Fatal("submitting must not report cancellation")
	}
	if !m.done {
		t.Fatal("expected the wizard to finish")
	}
	if m.Answers.Org != "acme" || m.Answers.Repos != "alpha,beta" {
		t.Fatalf("answers not collected: %+v", m.Answers)
	}
	if !m.Answers.Clone {
		t.Error("clone answer lost; y must round-trip through the boolean field")
	}
}

// Abandoning the wizard must not be mistaken for accepting the defaults.
func TestWizardCancel(t *testing.T) {
	m := send(NewWorkspaceWizard(WorkspaceAnswers{Root: "/tmp/ws", ReposDir: "src"}), "esc")

	if !m.Cancelled {
		t.Fatal("esc must cancel")
	}
	if !m.done {
		t.Fatal("cancelling must end the program")
	}
}

// Validation runs on submit and keeps the user's other answers, so a mistake
// costs one field rather than the whole form.
func TestWizardValidation(t *testing.T) {
	for _, tt := range []struct {
		name    string
		answers WorkspaceAnswers
		want    string
	}{
		{
			"root required",
			WorkspaceAnswers{ReposDir: "src"},
			"root is required",
		},
		{
			"checkout directory required",
			WorkspaceAnswers{Root: "/tmp/ws"},
			"Checkout directory is required",
		},
		{
			"declaring repositories needs a remote",
			WorkspaceAnswers{Root: "/tmp/ws", ReposDir: "src", Repos: "alpha"},
			"needs a remote template",
		},
		{
			"remote must be a template",
			WorkspaceAnswers{Root: "/tmp/ws", ReposDir: "src", Remote: "git@github.com:acme/happy.git"},
			"must contain {repo}",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := submit(NewWorkspaceWizard(tt.answers))

			if m.done {
				t.Fatal("an invalid form must not complete")
			}
			if !strings.Contains(m.err, tt.want) {
				t.Fatalf("error %q does not mention %q", m.err, tt.want)
			}
		})
	}
}

func TestWizardFocusWraps(t *testing.T) {
	m := NewWorkspaceWizard(WorkspaceAnswers{Root: "/tmp/ws", ReposDir: "src"})
	last := len(m.fields) - 1

	// tab from the last field wraps to the first, so the form cannot dead-end.
	keys := make([]string, 0, last+1)
	for i := 0; i <= last; i++ {
		keys = append(keys, "tab")
	}
	if got := send(m, keys...).focus; got != 0 {
		t.Fatalf("focus = %d after wrapping, want 0", got)
	}
}

// The last field is a decision, and a typo in it should be corrected rather
// than silently read as "no".
func TestWizardRejectsNonBooleanAnswer(t *testing.T) {
	m := NewWorkspaceWizard(WorkspaceAnswers{Root: "/tmp/ws", ReposDir: "src"})
	m.inputs[len(m.fields)-1].SetValue("maybe")

	m = submit(m)
	if m.done {
		t.Fatal("an unparseable y/n answer must not complete the form")
	}
	if !strings.Contains(m.err, "answer y or n") {
		t.Fatalf("error %q does not explain the expected answer", m.err)
	}
}
