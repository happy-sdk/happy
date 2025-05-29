// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package releaser

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/cli"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newConfiguration(sess *session.Context, path string, allowDirty bool) error {
	if path == "" {
		path = "."
	}

	if err := resolveProjectWD(sess, path); err != nil {
		return err
	}

	gitinfo, err := getGitInfo(sess)
	if err != nil {
		return err
	}

	if gitinfo.dirty == "true" {
		if !allowDirty {
			return fmt.Errorf("git repository is dirty - commit or stash changes before releasing")
		}

		addCmd := exec.Command("git", "add", "-A")
		addCmd.Dir = sess.Get("releaser.wd").String()
		if err := cli.Run(sess, addCmd); err != nil {
			return err
		}
		// commitCmd := exec.Command("git", "commit", "--amend", "--no-edit")
		// commitCmd.Dir = sess.Get("releaser.wd").String()
		// if err := cli.Run(sess, commitCmd); err != nil {
		// 	return err
		// }
		commitCmd := exec.Command("git", "commit", "-sm", "wip: prepare release")
		commitCmd.Dir = sess.Get("releaser.wd").String()
		if err := cli.Run(sess, commitCmd); err != nil {
			return err
		}
	}

	totalmodules := 0
	if err := filepath.Walk(sess.Get("releaser.wd").String(), func(path string, info os.FileInfo, err error) error {
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
		return err
	}

	dotenvp := filepath.Join(sess.Get("releaser.wd").String(), ".env")
	dotenvb, err := os.ReadFile(dotenvp)
	if err == nil {
		sess.Log().Debug("loading .env file", slog.String("path", dotenvp))
		env, err := vars.ParseMapFromBytes(dotenvb)
		if err != nil {
			return err
		}
		env.Range(func(v vars.Variable) bool {
			sess.Log().Debug("setting env var", slog.String("env", v.Name()))
			if err = os.Setenv(v.Name(), v.String()); err != nil {
				sess.Log().Error("error setting env var", slog.String("env", v.Name()), slog.String("value", v.String()), slog.String("err", err.Error()))
			}
			return true
		})
	}

	var opts = map[string]string{
		"releaser.git.branch":       gitinfo.branch,
		"releaser.git.remote.url":   gitinfo.remoteURL,
		"releaser.git.remote.name":  gitinfo.remoteName,
		"releaser.git.dirty":        gitinfo.dirty,
		"releaser.git.committer":    gitinfo.committer,
		"releaser.git.email":        gitinfo.email,
		"releaser.go.modules.count": fmt.Sprint(totalmodules),
		"releaser.go.monorepo":      fmt.Sprintf("%t", totalmodules > 1),
		"releaser.github.token":     os.Getenv("GITHUB_TOKEN"),
		"releaser.git.allow.dirty":  fmt.Sprintf("%t", allowDirty),
	}
	for key, value := range opts {
		if err := sess.Opts().Set(key, value); err != nil {
			return err
		}
	}

	return nil
}

// resolveProjectWD resolves the working directory of the project.
func resolveProjectWD(sess *session.Context, path string) error {
	currentPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Loop until a Git repository is found or the root directory is reached
	for {
		// Check if the current path is a Git repository
		if isGitRepo(currentPath) {
			return sess.Opts().Set("releaser.wd", currentPath)
		}

		parent := filepath.Dir(currentPath)
		// Check if we've reached the root directory
		if parent == currentPath {
			break
		}
		currentPath = parent
	}

	return errors.New("git repository not found in any parent directory")
}

// isGitRepo checks if the given directory is a Git repository.
func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	return err == nil || !os.IsNotExist(err)
}

type gitinfo struct {
	branch     string // current branch
	remoteURL  string // URL of the remote repository
	remoteName string // Name of the remote repository
	dirty      string // true if there are uncommitted changes
	committer  string // name of the committer
	email      string // email of the committer
}

func getGitInfo(sess *session.Context) (*gitinfo, error) {
	info := &gitinfo{}

	// Get current branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = sess.Get("releaser.wd").String()
	branch, err := cli.ExecRaw(sess, branchCmd)
	if err != nil {
		return nil, err
	}
	info.branch = strings.TrimSpace(string(branch))

	// Get remote name
	remoteCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "@{u}")
	remoteCmd.Dir = sess.Get("releaser.wd").String()
	remote, err := cli.ExecRaw(sess, remoteCmd)
	if err != nil {
		return nil, err
	}
	remoteParts := strings.SplitN(strings.TrimSpace(string(remote)), "/", 2)
	if len(remoteParts) > 0 {
		info.remoteName = remoteParts[0]
	}

	// Get origin URL
	remoteURLCmd := exec.Command("git", "config", "--get", "remote."+info.remoteName+".url")
	remoteURLCmd.Dir = sess.Get("releaser.wd").String()
	remoteURL, err := cli.ExecRaw(sess, remoteURLCmd)
	if err != nil {
		return nil, err
	}
	info.remoteURL = strings.TrimSpace(string(remoteURL))

	// Check for uncommitted changes
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = sess.Get("releaser.wd").String()
	status, err := cli.ExecRaw(sess, statusCmd)
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
	committerCmd.Dir = sess.Get("releaser.wd").String()
	committer, err := cli.ExecRaw(sess, committerCmd)
	if err != nil {
		return nil, err
	}
	info.committer = strings.TrimSpace(string(committer))

	emailCmd := exec.Command("git", "config", "user.email")
	emailCmd.Dir = sess.Get("releaser.wd").String()
	email, err := cli.ExecRaw(sess, emailCmd)
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

func getConfirmConfigModel(sess *session.Context) (configTable, error) {
	releaserOptions := sess.Opts().WithPrefix("releaser.")
	// sort keys
	var (
		longestKey         = 10
		longestValue       = 10
		longestDescription = 10
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
		var value string
		if option.Name == "releaser.github.token" {
			value = "********"
		} else {
			value = option.Value
		}
		rows = append(rows, table.Row{option.Name, value, option.Description})
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
