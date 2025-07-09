// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gomodule

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

	"github.com/happy-sdk/happy/cmd/gohappy/pkg/changelog"
	"github.com/happy-sdk/happy/cmd/gohappy/pkg/git"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

type Package struct {
	ModFilePath    string
	Dir            string
	TagPrefix      string
	Import         string
	Modfile        *modfile.File
	FirstRelease   bool
	NeedsRelease   bool
	PendingRelease bool
	IsInternal     bool
	UpdateDeps     bool
	NextReleaseTag string
	LastReleaseTag string
	Changelog      *changelog.Changelog
}

func LoadAll(sess *session.Context, wd string) ([]*Package, error) {
	var pkgs []*Package

	if err := filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
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

		pkg, err := Load(sess, wd, goModPath)
		if err != nil {
			return err
		}
		pkgs = append(pkgs, pkg)
		return nil
	}); err != nil {
		return nil, err
	}
	return pkgs, nil
}

func Load(sess *session.Context, root, path string) (pkg *Package, err error) {
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

	pkg.TagPrefix = strings.TrimPrefix(pkg.Dir+"/", root+"/")

	return pkg, nil
}

func (p *Package) LoadReleaseInfo(sess *session.Context, rootPath, origin string, checkRemote bool) error {

	sess.Log().Debug(
		"getting latest release",
		slog.String("package", p.Modfile.Module.Mod.Path),
		slog.String("tag.prefix", p.TagPrefix),
	)
	tagscmd := exec.Command("git", "tag", "--list", p.TagPrefix+"*")
	tagscmd.Dir = rootPath
	tagsout, err := cli.Exec(sess, tagscmd)
	if err != nil {
		return err
	}

	var nextVersion version.Version = "v0.1.0"
	nextVersionFile := filepath.Join(p.Dir, "VERSION")
	nextVersionBytes, err := os.ReadFile(nextVersionFile)
	if err == nil {
		nextVersionStr := strings.TrimSpace(string(nextVersionBytes))
		nextVersion, err = version.Parse(nextVersionStr)
		if err != nil {
			nextVersion = "v0.1.0"
		}
	}

	defer func() {
		if !version.IsValid(nextVersion) {
			return
		}
		nextReleaseTagVersion := version.Version(path.Base(p.NextReleaseTag))
		lastReleaseTagVersion := version.Version(path.Base(p.LastReleaseTag))
		if version.IsValid(nextReleaseTagVersion) {
			if version.Compare(nextVersion, nextReleaseTagVersion) == 1 {
				p.NextReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, nextVersion)
				p.NeedsRelease = true
			}
		} else if version.IsValid(lastReleaseTagVersion) {
			if version.Compare(nextVersion, lastReleaseTagVersion) == 1 {
				p.NextReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, nextVersion)
				p.NeedsRelease = true
			}
		} else {
			p.NextReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, nextVersion)
			p.NeedsRelease = true
		}
	}()

	if tagsout == "" {
		// First release
		p.FirstRelease = true
		p.NeedsRelease = true
		p.NextReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, nextVersion)
		p.LastReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, ".")
		if strings.Contains(p.Import, "internal") {
			p.FirstRelease = false
			p.NeedsRelease = false
			p.IsInternal = true
			p.LastReleaseTag = "."
			p.NextReleaseTag = "."
		}
		return nil
	}

	fulltags := strings.Split(tagsout, "\n")
	var tags []string
	for _, tag := range fulltags {

		ntag := strings.TrimPrefix(tag, p.TagPrefix)
		// skip nested package
		if strings.Contains(ntag, "/") {
			continue
		}
		tags = append(tags, ntag)
	}
	semver.Sort(tags)
	p.LastReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, tags[len(tags)-1])

	// Handle pending release
	if !checkRemote {
		return p.getChangelog(sess, rootPath)
	}
	if git.RemoteTagExists(sess, rootPath, origin, p.LastReleaseTag) {
		return p.getChangelog(sess, rootPath)
	}
	p.NextReleaseTag = p.LastReleaseTag
	p.LastReleaseTag = ""
	p.NeedsRelease = true
	p.PendingRelease = true

	tags = tags[:len(tags)-1]
	for i := len(tags) - 1; i >= 0; i-- {
		tag := fmt.Sprintf("%s%s", p.TagPrefix, tags[i])
		if git.RemoteTagExists(sess, rootPath, origin, tag) {
			p.LastReleaseTag = tag
			break
		}
	}
	if p.LastReleaseTag == "" {
		p.LastReleaseTag = fmt.Sprintf("%s%s", p.TagPrefix, ".")
		p.FirstRelease = true
	}
	return p.getChangelog(sess, rootPath)
}

func (p *Package) Tidy(sess *session.Context) error {
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = p.Dir
	_, err := cli.ExecRaw(sess, tidyCmd)
	return err
}

func (p *Package) SetDep(dep string, ver version.Version) error {
	if p.IsInternal {
		return nil
	}
	for _, require := range p.Modfile.Require {
		if require.Mod.Path == dep {
			requireModVersion, err := version.Parse(require.Mod.Version)
			if err != nil {
				return err
			}
			if version.Compare(ver, requireModVersion) == 0 {
				return nil
			}
		}
	}

	if err := p.Modfile.AddRequire(dep, ver.String()); err != nil {
		return err
	}
	p.NeedsRelease = true
	p.UpdateDeps = true

	if p.NextReleaseTag == "" || p.LastReleaseTag == p.NextReleaseTag {
		nextver, err := bumpPatch(p.TagPrefix, p.LastReleaseTag)
		if err != nil {
			return fmt.Errorf("failed to bump patch version for(%s): %w", p.Import, err)
		}
		p.NextReleaseTag = nextver
	}

	p.Modfile.Cleanup()
	return nil
}

func (p *Package) getChangelog(sess *session.Context, rootPath string) error {
	if p.IsInternal {
		return nil
	}
	var lastTagQuery = []string{"log"}
	upto := "HEAD"
	if p.PendingRelease {
		upto = p.NextReleaseTag
	}
	if !p.FirstRelease {
		lastTagQuery = append(lastTagQuery, fmt.Sprintf("%s..%s", p.LastReleaseTag, upto))
	}

	localpath := strings.TrimSuffix(p.TagPrefix, "/")
	if len(localpath) == 0 {
		localpath = "."
	}
	lastTagQuery = append(lastTagQuery, []string{"--pretty=format::COMMIT_START:%nSHORT:%h%nLONG:%H%nAUTHOR:%an%nMESSAGE:%B:COMMIT_END:", "--", localpath}...)
	logcmd := exec.Command("git", lastTagQuery...)
	logcmd.Dir = rootPath
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
		nextTag, err := bumpMajor(p.TagPrefix, p.LastReleaseTag)
		if err != nil {
			return fmt.Errorf("failed to bump major version for(%s): %w", p.Import, err)
		}
		p.NextReleaseTag = nextTag
		p.NeedsRelease = true
	} else if p.Changelog.HasMinorUpdate() {
		nextTag, err := bumpMinor(p.TagPrefix, p.LastReleaseTag)
		if err != nil {
			return fmt.Errorf("failed to bump minor version for(%s): %w", p.Import, err)
		}
		p.NextReleaseTag = nextTag
		p.NeedsRelease = true
	} else if p.Changelog.HasPatchUpdate() {
		nextTag, err := bumpPatch(p.TagPrefix, p.LastReleaseTag)
		if err != nil {
			return fmt.Errorf("failed to bump patch version for(%s): %w", p.Import, err)
		}
		p.NextReleaseTag = nextTag
		p.NeedsRelease = true
	}
	return nil
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
