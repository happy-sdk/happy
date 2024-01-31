// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package releaser

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/cli"
)

type configuration struct {
	WD string
}

func newConfiguration(sess *happy.Session, path string) (*configuration, error) {
	c := &configuration{}
	if path == "" {
		path = "."
	}
	if err := c.resolveProjectWD(sess, path); err != nil {
		return nil, err
	}

	gitinfo, err := c.getGitInfo(sess)
	if err != nil {
		return nil, err
	}
	if gitinfo.dirty == "true" {
		return nil, fmt.Errorf("git repository is dirty - commit or stash changes before releasing")
	}

	totalmodules := 0
	if err := filepath.Walk(c.WD, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		goModPath := filepath.Join(path, "go.mod")
		if _, err := os.Stat(goModPath); err != nil {
			return nil
		}
		totalmodules++
		return nil
	}); err != nil {
		return nil, err
	}

	var opts map[string]string = map[string]string{
		"releaser.working.directory": c.WD,
		"releaser.git.branch":        gitinfo.branch,
		"releaser.git.remote.url":    gitinfo.remoteURL,
		"releaser.git.remote.name":   gitinfo.remoteName,
		"releaser.git.dirty":         gitinfo.dirty,
		"releaser.git.committer":     gitinfo.committer,
		"releaser.git.email":         gitinfo.email,
		"releaser.go.modules.count":  fmt.Sprint(totalmodules),
		"releaser.go.monorepo":       fmt.Sprintf("%t", totalmodules > 1),
	}

	for key, value := range opts {
		if err := sess.Set(key, value); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// resolveProjectWD resolves the working directory of the project.
func (c *configuration) resolveProjectWD(sess *happy.Session, path string) error {
	currentPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Loop until a Git repository is found or the root directory is reached
	for {
		// Check if the current path is a Git repository
		if isGitRepo(currentPath) {
			c.WD = currentPath
			return nil
		}

		parent := filepath.Dir(currentPath)
		// Check if we've reached the root directory
		if parent == currentPath {
			break
		}
		currentPath = parent
	}

	return errors.New("Git repository not found in any parent directory")
}

type gitinfo struct {
	branch     string // current branch
	remoteURL  string // URL of the remote repository
	remoteName string // Name of the remote repository
	dirty      string // true if there are uncommitted changes
	committer  string // name of the committer
	email      string // email of the committer
}

func (c *configuration) getGitInfo(sess *happy.Session) (*gitinfo, error) {
	info := &gitinfo{}

	// Get current branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = c.WD
	branch, err := cli.ExecCommandRaw(sess, branchCmd)
	if err != nil {
		return nil, err
	}
	info.branch = strings.TrimSpace(string(branch))

	// Get remote name
	remoteCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "@{u}")
	remoteCmd.Dir = c.WD
	remote, err := cli.ExecCommandRaw(sess, remoteCmd)
	if err != nil {
		return nil, err
	}
	remoteParts := strings.SplitN(strings.TrimSpace(string(remote)), "/", 2)
	if len(remoteParts) > 0 {
		info.remoteName = remoteParts[0]
	}

	// Get origin URL
	remoteURLCmd := exec.Command("git", "config", "--get", "remote."+info.remoteName+".url")
	remoteURLCmd.Dir = c.WD
	remoteURL, err := cli.ExecCommandRaw(sess, remoteURLCmd)
	if err != nil {
		return nil, err
	}
	info.remoteURL = strings.TrimSpace(string(remoteURL))

	// Check for uncommitted changes
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = c.WD
	status, err := cli.ExecCommandRaw(sess, statusCmd)
	if err != nil {
		return nil, err
	}
	if bytes.TrimSpace(status) != nil {
		info.dirty = "true"
	} else {
		info.dirty = "false"
	}

	// Get committer name and email
	committerCmd := exec.Command("git", "config", "user.name")
	committerCmd.Dir = c.WD
	committer, err := cli.ExecCommandRaw(sess, committerCmd)
	if err != nil {
		return nil, err
	}
	info.committer = strings.TrimSpace(string(committer))

	emailCmd := exec.Command("git", "config", "user.email")
	emailCmd.Dir = c.WD
	email, err := cli.ExecCommandRaw(sess, emailCmd)
	if err != nil {
		return nil, err
	}
	info.email = strings.TrimSpace(string(email))

	return info, nil
}

type DescribedOption struct {
	Name        string
	Description string
	Value       string
}

// getConfirmConfigModel returns the model for the confirmation table.
func (c *configuration) getConfirmConfigModel(sess *happy.Session) (configTable, error) {
	releaserOptions := sess.Opts().ExtractWithPrefix("releaser.")
	// sort keys
	var (
		longestKey         int = 10
		longestValue       int = 10
		longestDescription int = 10
	)

	var options []DescribedOption

	releaserOptions.Range(func(v vars.Variable) bool {
		if len("releaser."+v.Name()) > longestKey {
			longestKey = len("releaser." + v.Name())
		}
		if v.Len() > longestValue {
			longestValue = v.Len()
		}
		desc := sess.Describe("releaser." + v.Name())
		if len(desc) > longestDescription {
			longestDescription = len(desc)
		}
		options = append(options, DescribedOption{
			Name:        "releaser." + v.Name(),
			Description: desc,
			Value:       v.String(),
		})
		return true
	})

	sort.Slice(options, func(i, j int) bool {
		return options[i].Name < options[j].Name
	})

	// build table

	columns := []table.Column{
		{Title: "Key", Width: longestKey + 2},
		{Title: "Value", Width: longestValue + 2},
		{Title: "Description", Width: longestDescription + 2},
	}
	var rows []table.Row

	for _, option := range options {
		rows = append(rows, table.Row{option.Name, option.Value, option.Description})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#ffed56")).
		Background(lipgloss.Color("0")).
		Bold(false)
	t.SetStyles(s)
	m := configTable{
		table: t,
	}
	return m, nil
}

var statusMessageStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#f40202")).
	Render

var configTableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#ffed56")).
	Render

type configTable struct {
	answered bool
	yes      bool
	err      string
	table    table.Model
}

func (m configTable) Init() tea.Cmd {
	return nil
}

func (m configTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.err = ""
			m.yes = true
			m.answered = true
			return m, tea.Quit
		case "n", "N":
			m.err = ""
			m.yes = false
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

func (m configTable) View() string {
	if m.answered {
		return ""
	}
	view := "RELEASE SETTINGS\n\n"
	view += "The following settings will be used to create the release.\n\n"
	view += configTableStyle(m.table.View()) + "\n\n"
	view += "Do you want to continue? [y/n]: \n"
	if m.err != "" {
		return view + "\n" + statusMessageStyle(m.err)
	}
	return view
}

// isGitRepo checks if the given directory is a Git repository.
func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	return err == nil || !os.IsNotExist(err)
}
