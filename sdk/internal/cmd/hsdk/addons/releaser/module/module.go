// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package module

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/internal/cmd/hsdk/addons/releaser/changelog"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

type Package struct {
	ModFilePath  string
	Dir          string
	TagPrefix    string
	Import       string
	Modfile      *modfile.File
	FirstRelease bool
	NeedsRelease bool
	UpdateDeps   bool
	NextRelease  string
	LastRelease  string
	Changelog    *changelog.Changelog
}

func Load(path string) (pkg *Package, err error) {
	if path == "" {
		return nil, errors.New("can not load module, path is empty")
	}

	pkg = &Package{}

	if filepath.Base(path) == "go.mod" {
		pkg.ModFilePath = path
		pkg.Dir = filepath.Dir(path)
	} else {
		pkg.Dir, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		pkg.ModFilePath = filepath.Join(path, "go.mod")
	}

	if len(pkg.Dir) < 5 {
		return nil, fmt.Errorf("invalid module directory %s", pkg.Dir)
	}

	dirstat, err := os.Stat(pkg.Dir)
	if err != nil {
		return nil, err
	}
	if !dirstat.IsDir() {
		return nil, fmt.Errorf("invalid module directory %s", pkg.Dir)
	}

	modstat, err := os.Stat(pkg.ModFilePath)
	if err != nil {
		return nil, err
	}
	if modstat.IsDir() {
		return nil, fmt.Errorf("invalid module go.mod path %s", pkg.ModFilePath)
	}

	data, err := os.ReadFile(pkg.ModFilePath)
	if err != nil {
		return nil, err
	}

	pkg.Modfile, err = modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}
	pkg.Import = pkg.Modfile.Module.Mod.Path

	return pkg, nil
}

func (p *Package) LoadReleaseInfo(sess *session.Context) error {
	sess.Log().Debug("getting latest release", slog.String("package", p.Modfile.Module.Mod.Path))
	tagscmd := exec.Command("git", "tag", "--list", p.TagPrefix+"*")
	tagscmd.Dir = p.Dir
	tagsout, err := cli.Exec(sess, tagscmd)
	if err != nil {
		return err
	}
	if tagsout == "" {
		// First release
		p.FirstRelease = true
		p.NeedsRelease = true
		p.NextRelease = fmt.Sprintf("%s%s", p.TagPrefix, "v0.1.0")
		p.LastRelease = fmt.Sprintf("%s%s", p.TagPrefix, "v0.0.0")
		if strings.Contains(p.Import, "internal") {
			p.NeedsRelease = false
			p.NextRelease = fmt.Sprintf("%s%s", p.TagPrefix, "v0.0.0")
		}
		return nil
	}

	fulltags := strings.Split(tagsout, "\n")
	var tags []string
	for _, tag := range fulltags {
		tags = append(tags, strings.TrimPrefix(tag, p.TagPrefix))
	}
	semver.Sort(tags)
	p.LastRelease = fmt.Sprintf("%s%s", p.TagPrefix, tags[len(tags)-1])
	return p.getChangelog(sess)
}

func (p *Package) getChangelog(sess *session.Context) error {
	var lastTagQuery = []string{"log"}
	if !p.FirstRelease {
		lastTagQuery = append(lastTagQuery, fmt.Sprintf("%s..HEAD", p.LastRelease))
	}
	localpath := strings.TrimSuffix(p.TagPrefix, "/")
	if len(localpath) == 0 {
		localpath = "."
	}
	lastTagQuery = append(lastTagQuery, []string{"--pretty=format::COMMIT_START:%nSHORT:%h%nLONG:%H%nAUTHOR:%an%nMESSAGE:%B:COMMIT_END:", "--", localpath}...)
	logcmd := exec.Command("git", lastTagQuery...)
	logcmd.Dir = sess.Get("releaser.wd").String()
	logout, err := cli.Exec(sess, logcmd)
	if err != nil {
		return err
	}
	changelog, err := changelog.ParseGitLog(sess, logout)
	if err != nil {
		return err
	}
	p.Changelog = changelog
	if p.Changelog.Empty() {
		sess.Log().Debug("no changelog", slog.String("package", p.Import))
		return nil
	}
	if p.Changelog.HasMajorUpdate() {
		nextver, err := bumpMajor(p.TagPrefix, p.LastRelease)
		if err != nil {
			return fmt.Errorf("failed to bump major version for(%s): %w", p.Import, err)
		}
		p.NextRelease = nextver
		p.NeedsRelease = true
	} else if p.Changelog.HasMinorUpdate() {
		nextver, err := bumpMinor(p.TagPrefix, p.LastRelease)
		if err != nil {
			return fmt.Errorf("failed to bump minor version for(%s): %w", p.Import, err)
		}
		p.NextRelease = nextver
		p.NeedsRelease = true
	} else if p.Changelog.HasPatchUpdate() {
		nextver, err := bumpPatch(p.TagPrefix, p.LastRelease)
		if err != nil {
			return fmt.Errorf("failed to bump patch version for(%s): %w", p.Import, err)
		}
		p.NextRelease = nextver
		p.NeedsRelease = true
	}
	return nil
}

func (p *Package) SetDep(dep string, version string) error {
	for _, require := range p.Modfile.Require {
		if require.Mod.Path == dep {
			version = path.Base(version)
			if semver.Compare(version, require.Mod.Version) == 1 {
				if err := p.Modfile.AddRequire(require.Mod.Path, version); err != nil {
					return err
				}
				p.NeedsRelease = true
				p.UpdateDeps = true
				if p.NextRelease == "" || p.LastRelease == p.NextRelease {
					nextver, err := bumpPatch(p.TagPrefix, p.LastRelease)
					if err != nil {
						return fmt.Errorf("failed to bump patch version for(%s): %w", p.Import, err)
					}
					p.NextRelease = nextver
				}
				break
			}
		}
	}
	p.Modfile.Cleanup()
	return nil
}

// TopologicalReleaseQueue performs a topological sort on the dependency graph
func TopologicalReleaseQueue(pkgs []*Package) ([]string, error) {
	pkgMap := make(map[string]*Package)
	for i := range pkgs {
		pkgMap[pkgs[i].Import] = pkgs[i]
	}

	// Build dependency graph
	depGraph := make(map[string][]string)
	for _, pkg := range pkgs {
		for _, require := range pkg.Modfile.Require {
			if dep, exists := pkgMap[require.Mod.Path]; exists {
				// Add dependency only if it's within our pkgs
				depGraph[pkg.Import] = append(depGraph[pkg.Import], require.Mod.Path)
				if err := pkg.SetDep(dep.Import, dep.NextRelease); err != nil {
					return nil, err
				}
			}
		}
	}

	// Topological Sort
	var queue []string
	visited := make(map[string]bool)
	var visit func(string) error
	visit = func(n string) error {
		if visited[n] {
			return nil
		}
		visited[n] = true
		for _, m := range depGraph[n] {
			if err := visit(m); err != nil {
				return err
			}
		}
		queue = append(queue, n)
		return nil
	}
	for name := range depGraph {
		if err := visit(name); err != nil {
			return nil, fmt.Errorf("Dependency resolution error: %v", err)
		}
	}
	return queue, nil
}

type Dependency struct {
	Import     string
	UsedBy     []string
	MaxVersion string
	MinVersion string
}

func GetCommonDeps(pkgs []*Package) ([]Dependency, error) {
	// Map to hold the count of each external dependency
	deps := make(map[string]Dependency)

	// Map to quickly check if a dependency is internal
	internalDeps := make(map[string]struct{})
	for _, pkg := range pkgs {
		internalDeps[pkg.Import] = struct{}{}
	}

	// Collect and count external dependencies
	for _, pkg := range pkgs {
		for _, require := range pkg.Modfile.Require {
			if _, internal := internalDeps[require.Mod.Path]; !internal {
				if dep, exists := deps[require.Mod.Path]; !exists {
					deps[require.Mod.Path] = Dependency{
						Import:     require.Mod.Path,
						UsedBy:     []string{pkg.Import},
						MaxVersion: require.Mod.Version,
						MinVersion: require.Mod.Version,
					}
				} else {
					if semver.Compare(require.Mod.Version, dep.MaxVersion) == 1 {
						dep.MaxVersion = require.Mod.Version
					}
					if semver.Compare(require.Mod.Version, dep.MinVersion) == -1 {
						dep.MinVersion = require.Mod.Version
					}
					dep.UsedBy = append(dep.UsedBy, pkg.Import)
					deps[require.Mod.Path] = dep
				}
			}
		}
	}

	// Filter dependencies that are referenced by at least two packages
	var commonDeps []Dependency
	for _, dep := range deps {
		if len(dep.UsedBy) >= 2 {
			commonDeps = append(commonDeps, dep)
		}
	}
	return commonDeps, nil
}

var statusMessageStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#f40202")).
	Render

var configTableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#ffed56")).
	Render

type ReleasablesTableView struct {
	Yes      bool
	answered bool
	err      string
	table    table.Model
}

func (m ReleasablesTableView) Init() tea.Cmd {
	return nil
}

func (m ReleasablesTableView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
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

func (m ReleasablesTableView) View() string {
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

func GetConfirmReleasablesView(sess *session.Context, pkgs []*Package, queue []string) (ReleasablesTableView, error) {
	var (
		longestPackage = 10
	)

	for _, pkg := range pkgs {
		fmt.Println(pkg.Import)
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

	for _, impr := range queue {
		for _, pkg := range pkgs {
			if pkg.Import == impr {
				action := "skip"
				if pkg.NeedsRelease {
					action = "release"
				}
				if pkg.FirstRelease {
					action = "initial"
				}

				rows = append(rows, table.Row{pkg.Import, action, path.Base(pkg.LastRelease), path.Base(pkg.NextRelease), fmt.Sprint(pkg.UpdateDeps)})
			}
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(20),
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
	m := ReleasablesTableView{
		table: t,
	}
	return m, nil
}

func bumpMajor(prefix, ver string) (string, error) {
	clean := strings.TrimPrefix(ver, prefix+"v")
	parts := strings.Split(clean, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version: %s", ver)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", prefix, fmt.Sprintf("v%d.0.0", major+1)), nil
}

func bumpMinor(prefix, ver string) (string, error) {
	clean := strings.TrimPrefix(ver, prefix+"v")
	parts := strings.Split(clean, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version: %s", ver)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", prefix, fmt.Sprintf("v%s.%d.0", parts[0], minor+1)), nil
}

func bumpPatch(prefix, ver string) (string, error) {
	clean := strings.TrimPrefix(ver, prefix+"v")
	parts := strings.Split(clean, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version: %s", ver)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", prefix, fmt.Sprintf("v%s.%s.%d", parts[0], parts[1], patch+1)), nil
}
