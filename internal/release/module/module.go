// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package module

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy-go/internal/release/changelog"
	"github.com/happy-sdk/happy-go/internal/release/git"
	"github.com/happy-sdk/happy/sdk/cli"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

var ErrSkipPackage = fmt.Errorf("skip package")

type Package struct {
	wd           string
	minGoVer     string
	modfile      *modfile.File
	firstRelease bool // true if this is the first release of the package

	ModFilePath  string // full path to go mod file
	LocalPath    string // relative path to the monorepo root directory
	TagPrefix    string // tag prefix for the package
	Import       string // import path
	NeedsRelease bool
	LastRelease  string
	NextRelease  string
	Changelog    *changelog.Changelog
}

func Load(sess *happy.Session, wd, goModPath, minGoVer string) (pkg *Package, err error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}
	pkg = &Package{
		ModFilePath: goModPath,
	}
	pkg.modfile, err = modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}
	pkg.Import = pkg.modfile.Module.Mod.Path
	pkgpath := strings.TrimPrefix(pkg.modfile.Module.Mod.Path, "github.com/happy-sdk/happy-go/")
	if pkgpath == "github.com/happy-sdk/happy-go" {
		sess.Log().Info("skipping package", slog.String("package", "github.com/happy-sdk/happy-go"))
		return nil, fmt.Errorf("%w: %s", ErrSkipPackage, pkg.Import)
	}
	pkg.LocalPath = pkgpath
	pkg.TagPrefix = pkgpath + "/"

	if err := pkg.getLatestRelease(sess, wd); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (p *Package) Prepare(sess *happy.Session, wd string) error {
	sess.Log().Debug("prepare", slog.String("package", p.Import))

	if semver.Compare("v"+p.modfile.Go.Version, "v"+p.minGoVer) == -1 {
		oldver := p.modfile.Go.Version
		sess.Log().Ok("updated go version",
			slog.String("module", p.modfile.Module.Mod.Path),
			slog.String("from", oldver),
			slog.String("to", p.minGoVer),
		)
		if err := p.modfile.AddGoStmt(p.minGoVer); err != nil {
			return err
		}
		p.modfile.Cleanup()
		// Write the updated file back
		updatedModFile, err := p.modfile.Format()
		if err != nil {
			return err
		}
		if err := os.WriteFile(p.ModFilePath, updatedModFile, 0644); err != nil {
			return err
		}
		gomodtidy := exec.Command("go", "mod", "tidy")
		gomodtidy.Dir = filepath.Dir(p.ModFilePath)
		if err := cli.RunCommand(sess, gomodtidy); err != nil {
			return err
		}
		if err := git.AddAndCommit(sess, wd, "dep", p.LocalPath, fmt.Sprintf("update go version from %s to %s", oldver, p.minGoVer)); err != nil {
			return err
		}
	} else {
		sess.Log().Debug("go version ok",
			slog.String("module", p.modfile.Module.Mod.Path),
			slog.String("go.version", p.modfile.Go.Version),
		)
	}
	return p.getChangelog(sess, wd)
}

func (p *Package) getLatestRelease(sess *happy.Session, wd string) error {
	sess.Log().Debug("getting latest release", slog.String("package", p.Import))
	tagscmd := exec.Command("git", "tag", "--list", p.TagPrefix+"*")
	tagscmd.Dir = wd
	tagsout, err := cli.ExecCommand(sess, tagscmd)
	if err != nil {
		return err
	}
	if tagsout == "" {
		// First release
		p.firstRelease = true
		p.NeedsRelease = true
		p.NextRelease = fmt.Sprintf("%s%s", p.TagPrefix, "v0.1.0")
		p.LastRelease = fmt.Sprintf("%s%s", p.TagPrefix, "v0.0.0")
		return nil
	}

	fulltags := strings.Split(tagsout, "\n")
	var tags []string
	for _, tag := range fulltags {
		tags = append(tags, strings.TrimPrefix(tag, p.TagPrefix))
	}
	semver.Sort(tags)
	p.LastRelease = fmt.Sprintf("%s%s", p.TagPrefix, tags[len(tags)-1])
	return nil
}

func (p *Package) getChangelog(sess *happy.Session, wd string) error {
	var lastTagQuery = []string{"log"}
	if !p.firstRelease {
		lastTagQuery = append(lastTagQuery, fmt.Sprintf("%s..HEAD", p.LastRelease))
	}
	lastTagQuery = append(lastTagQuery, []string{"--pretty=format::COMMIT_START:%nSHORT:%h%nLONG:%H%nAUTHOR:%an%nMESSAGE:%B:COMMIT_END:", "--", p.LocalPath}...)

	logcmd := exec.Command("git", lastTagQuery...)
	logcmd.Dir = wd
	logout, err := cli.ExecCommand(sess, logcmd)
	if err != nil {
		return err
	}

	changelog, err := changelog.ParseGitLog(sess, logout)
	if err != nil {
		return err
	}
	p.Changelog = changelog

	if p.Changelog.Empty() {
		return nil
	}
	if p.Changelog.IsBreaking() {
		nextver, err := bumpMajor(p.TagPrefix, p.LastRelease)
		if err != nil {
			return fmt.Errorf("failed to bump major version for(%s): %w", p.Import, err)
		}
		p.NextRelease = nextver
		p.NeedsRelease = true
	} else if p.Changelog.IsFeature() {
		nextver, err := bumpMinor(p.TagPrefix, p.LastRelease)
		if err != nil {
			return fmt.Errorf("failed to bump minor version for(%s): %w", p.Import, err)
		}
		p.NextRelease = nextver
		p.NeedsRelease = true
	} else if p.Changelog.IsPatch() {
		nextver, err := bumpPatch(p.TagPrefix, p.LastRelease)
		if err != nil {
			return fmt.Errorf("failed to bump patch version for(%s): %w", p.Import, err)
		}
		p.NextRelease = nextver
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
