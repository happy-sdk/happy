// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package views

import (
	"fmt"
	"path"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/happy-sdk/happy/addons/devel/pkg/gomodule"
	"github.com/happy-sdk/happy/sdk/session"
)

var statusMessageStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#f40202")).
	Render

var configTableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#ffed56")).
	Render

func GetConfirmReleasablesView(sess *session.Context, pkgs []*gomodule.Package) (ConfirmReleasablesView, error) {
	var (
		longestPackage = 10
	)

	for _, pkg := range pkgs {
		if len(pkg.Import) > longestPackage {
			longestPackage = len(pkg.Import)
		}
	}
	columns := []table.Column{
		{Title: "Package", Width: longestPackage},
		{Title: "Action", Width: 10},
		{Title: "Current", Width: 10},
		{Title: "Next", Width: 10},
		{Title: "Update deps", Width: 20},
	}
	var rows []table.Row

	for _, pkg := range pkgs {
		action := "skip"
		if pkg.NeedsRelease {
			action = "release"
		}
		if pkg.FirstRelease {
			action = "initial"
		}
		rows = append(rows, table.Row{pkg.Import, action, path.Base(pkg.LastReleaseTag), path.Base(pkg.NextReleaseTag), fmt.Sprint(pkg.UpdateDeps)})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(30),
		table.WithFocused(true),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#ffed56")).
		Background(lipgloss.Color("0")).
		Bold(false)

	t.SetStyles(s)
	m := ConfirmReleasablesView{
		table: t,
	}
	return m, nil
}

type ConfirmReleasablesView struct {
	Yes      bool
	answered bool
	err      string
	table    table.Model
}

func (m ConfirmReleasablesView) Init() tea.Cmd {
	return nil
}

func (m ConfirmReleasablesView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "y", "Y":
			m.err = ""
			m.Yes = true
			m.answered = true
			return m, tea.Quit
		case "n", "N":
			m.err = ""
			m.Yes = false
			m.answered = true
			return m, tea.Quit
		case "up", "down":
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		default:
			m.err = fmt.Sprintf("invalid input %q", msg.String())
		}
	}

	return m, nil
}

func (m ConfirmReleasablesView) View() tea.View {
	if m.answered {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	view := "PACKAGES\n\n"
	view += configTableStyle(m.table.View()) + "\n\n"
	view += "Do you want to continue? [y/n]: \n"
	if m.err != "" {
		view += "\n" + statusMessageStyle(m.err)
	}
	v := tea.NewView(view)
	v.AltScreen = true
	return v
}
