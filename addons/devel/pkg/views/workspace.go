// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	wizardTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffed56")).
				Bold(true).
				Render

	wizardLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7dcfff")).
				Render

	wizardHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7a7a7a")).
			Render

	wizardErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f40202")).
			Render
)

// WorkspaceAnswers is what the wizard collected. Every field mirrors a flag on
// `workspace init`, so the wizard only ever fills in the same contract the
// command already has - it never becomes the only way to do something.
type WorkspaceAnswers struct {
	Root         string
	Org          string
	Remote       string
	Repos        string
	ReposDir     string
	Scratch      string
	Instructions string
	Clone        bool
}

// Cancelled reports whether the user abandoned the wizard, which must not be
// treated as "accept the defaults".
type WorkspaceWizard struct {
	Answers   WorkspaceAnswers
	Cancelled bool

	inputs []textinput.Model
	fields []wizardField
	focus  int
	err    string
	done   bool
}

type wizardField struct {
	label string
	hint  string
	// boolean fields accept only y/n, so the summary can render them as a
	// decision rather than as text the user has to get exactly right.
	boolean bool
	get     func(*WorkspaceAnswers) string
	set     func(*WorkspaceAnswers, string)
}

// NewWorkspaceWizard builds the interactive form, seeded with whatever the
// flags already provided. Seeding matters: a user who passed --org should not
// be asked for it again with an empty box.
func NewWorkspaceWizard(seed WorkspaceAnswers) WorkspaceWizard {
	fields := []wizardField{
		{
			label: "Workspace root",
			hint:  "Directory to create the workspace in.",
			get:   func(a *WorkspaceAnswers) string { return a.Root },
			set:   func(a *WorkspaceAnswers, v string) { a.Root = v },
		},
		{
			label: "Organization",
			hint:  "Optional name, used for display only.",
			get:   func(a *WorkspaceAnswers) string { return a.Org },
			set:   func(a *WorkspaceAnswers, v string) { a.Org = v },
		},
		{
			label: "Remote template",
			hint:  "Clone URL with {repo}, e.g. git@github.com:acme/{repo}.git",
			get:   func(a *WorkspaceAnswers) string { return a.Remote },
			set:   func(a *WorkspaceAnswers, v string) { a.Remote = v },
		},
		{
			label: "Repositories",
			hint:  "Comma separated, to declare now. Others can be added later with clone.",
			get:   func(a *WorkspaceAnswers) string { return a.Repos },
			set:   func(a *WorkspaceAnswers, v string) { a.Repos = v },
		},
		{
			label: "Checkout directory",
			hint:  "Where repositories are cloned.",
			get:   func(a *WorkspaceAnswers) string { return a.ReposDir },
			set:   func(a *WorkspaceAnswers, v string) { a.ReposDir = v },
		},
		{
			label: "Scratch directory",
			hint:  "Your own notes and files. Leave empty for none.",
			get:   func(a *WorkspaceAnswers) string { return a.Scratch },
			set:   func(a *WorkspaceAnswers, v string) { a.Scratch = v },
		},
		{
			label: "Agent instructions",
			hint:  "Path to workspace instructions, often inside a cloned repository.",
			get:   func(a *WorkspaceAnswers) string { return a.Instructions },
			set:   func(a *WorkspaceAnswers, v string) { a.Instructions = v },
		},
		{
			label:   "Clone now",
			hint:    "y/n - cloning reaches the network and writes to disk.",
			boolean: true,
			get: func(a *WorkspaceAnswers) string {
				if a.Clone {
					return "y"
				}
				return "n"
			},
			set: func(a *WorkspaceAnswers, v string) {
				a.Clone = strings.EqualFold(strings.TrimSpace(v), "y")
			},
		},
	}

	w := WorkspaceWizard{Answers: seed, fields: fields}
	for i, f := range fields {
		in := textinput.New()
		in.Prompt = ""
		in.SetValue(f.get(&w.Answers))
		in.Placeholder = f.hint
		in.CharLimit = 256
		if i == 0 {
			in.Focus()
		}
		w.inputs = append(w.inputs, in)
	}
	return w
}

func (m WorkspaceWizard) Init() tea.Cmd { return textinput.Blink }

func (m WorkspaceWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Abandoning must not be mistaken for accepting the defaults.
			m.Cancelled = true
			m.done = true
			return m, tea.Quit
		case "enter":
			if m.focus == len(m.inputs)-1 {
				if err := m.collect(); err != "" {
					m.err = err
					return m, nil
				}
				m.done = true
				return m, tea.Quit
			}
			return m.moveFocus(1), nil
		case "tab", "down":
			return m.moveFocus(1), nil
		case "shift+tab", "up":
			return m.moveFocus(-1), nil
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	m.err = ""
	return m, cmd
}

func (m WorkspaceWizard) moveFocus(delta int) WorkspaceWizard {
	m.inputs[m.focus].Blur()
	m.focus = (m.focus + delta + len(m.inputs)) % len(m.inputs)
	m.inputs[m.focus].Focus()
	m.err = ""
	return m
}

// collect writes the inputs into Answers, returning a message when something
// is unusable. Validation lives here rather than in the command so the user
// can fix it without losing everything else they typed.
func (m *WorkspaceWizard) collect() string {
	for i, f := range m.fields {
		v := strings.TrimSpace(m.inputs[i].Value())
		if f.boolean && v != "" && !strings.EqualFold(v, "y") && !strings.EqualFold(v, "n") {
			return fmt.Sprintf("%s: answer y or n", f.label)
		}
		f.set(&m.Answers, v)
	}
	if m.Answers.Root == "" {
		return "Workspace root is required"
	}
	if m.Answers.ReposDir == "" {
		return "Checkout directory is required"
	}
	if m.Answers.Repos != "" && m.Answers.Remote == "" {
		return "Declaring repositories needs a remote template containing {repo}"
	}
	if m.Answers.Remote != "" && !strings.Contains(m.Answers.Remote, "{repo}") {
		return "Remote template must contain {repo}"
	}
	return ""
}

func (m WorkspaceWizard) View() tea.View {
	if m.done {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", wizardTitleStyle("Create a workspace"))
	fmt.Fprintf(&b, "%s\n", wizardHintStyle("A workspace holds several repository checkouts side by side."))
	fmt.Fprintf(&b, "%s\n\n", wizardHintStyle("It is local state, not a repository, and is never committed."))

	for i, f := range m.fields {
		marker := "  "
		if i == m.focus {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%s\n  %s\n", marker, wizardLabelStyle(f.label), m.inputs[i].View())
		if i == m.focus {
			fmt.Fprintf(&b, "  %s\n", wizardHintStyle(f.hint))
		}
		b.WriteString("\n")
	}

	if m.err != "" {
		fmt.Fprintf(&b, "%s\n\n", wizardErrStyle(m.err))
	}
	fmt.Fprintf(&b, "%s\n", wizardHintStyle("tab/↑↓ move · enter on the last field creates · esc cancels"))

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}
