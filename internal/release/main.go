// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/commands"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
	"golang.org/x/text/language"
)

type Releaser struct {
	WD           string
	SkipPackages []string
	Package      map[string]*Package
}

type Settings struct {
	happy.Settings
	GithubToken  settings.String `key:"github.token" mutation:"once"`
	MinGoVersion settings.String `key:"go.version.min" mutation:"once"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	blueprint, err := settings.NewBlueprint(s)
	if err != nil {
		return nil, err
	}
	blueprint.Describe("github.token", language.English, "Github token")
	return blueprint, nil
}

func main() {
	settings := happy.Settings{
		Name:           "Happy-Go - releaser",
		CopyrightBy:    "The Happy Authors",
		CopyrightSince: 2023,
		License:        "Apache License 2.0",
		Logger: logging.Settings{
			Secrets: "token",
		},
	}
	settings.Extend("happy-go", Settings{
		GithubToken:  "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",
		MinGoVersion: "1.21.5",
	})

	app := happy.New(settings)

	// Happy CLI commands
	app.AddCommand(commands.Info())
	app.AddCommand(commands.Reset())

	releaser := &Releaser{
		SkipPackages: []string{
			"internal/release",
		},
	}

	app.Before(func(sess *happy.Session, args happy.Args) error {
		if !sess.Profile().Get("happy-go.github.token").IsSet() {
			return nil // not retruning error here so that we can call other subcommands
		}

		gitstatus := exec.Command("git", "diff-index", "--quiet", "HEAD")
		if err := cli.RunCommand(sess, gitstatus); err != nil {
			sess.Log().NotImplemented("git is in a dirty state", slog.String("err", err.Error()))
			// return errors.New("git is in a dirty state")
		}
		return releaser.Before(sess)
	})

	app.Do(func(sess *happy.Session, args happy.Args) error {
		if !sess.Profile().Get("happy-go.github.token").IsSet() {
			return errors.New("github.token is not set")
		}
		sess.Log().Info("using GITHUB_TOKEN", slog.String("token", sess.Profile().Get("happy-go.github.token").String()))
		sess.Log().Info("do")
		return nil
	})
	app.Main()
}

func (r *Releaser) Before(sess *happy.Session) error {
	r.WD = sess.Get("app.fs.path.pwd").String()
	if strings.HasSuffix(r.WD, "internal/release") {
		r.WD = filepath.Join(r.WD, "../../")
	}
	if !strings.HasSuffix(r.WD, "happy-go") {
		return fmt.Errorf("can noit call release from location %s", r.WD)
	}

	sess.Log().Msg("working directory", slog.String("dir", r.WD))

	if err := r.loadModules(sess); err != nil {
		return err
	}
	return nil
}

func (r *Releaser) loadModules(sess *happy.Session) error {
	// var modules []Package
	minGoVer := sess.Profile().Get("happy-go.go.version.min").String()
	sess.Log().Msg("Min Go Version", slog.String("version", minGoVer))
	if err := filepath.Walk(r.WD, func(path string, info os.FileInfo, err error) error {
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
		return r.loadModule(sess, goModPath, minGoVer)
	}); err != nil {
		return err
	}
	return nil
}

func (r *Releaser) loadModule(sess *happy.Session, goModPath, minGoVer string) error {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}
	pkg := &Package{
		ModFilePath: goModPath,
	}
	modFile, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return err
	}
	pkg.Import = modFile.Module.Mod.Path

	pkgpath := strings.TrimPrefix(modFile.Module.Mod.Path, "github.com/happy-sdk/happy-go/")
	if pkgpath == "github.com/happy-sdk/happy-go" {
		sess.Log().Info("skipping package", slog.String("package", "github.com/happy-sdk/happy-go"))
		return nil
	}
	pkg.LocalPath = pkgpath
	pkg.TagPrefix = pkgpath + "/"

	for _, skip := range r.SkipPackages {
		if skip == pkgpath {
			sess.Log().Info("skipping package", slog.String("package", pkgpath))
			return nil
		}
	}

	// update go version if needed
	if semver.Compare("v"+modFile.Go.Version, "v"+minGoVer) == -1 {
		oldver := modFile.Go.Version
		sess.Log().Ok("updated go version",
			slog.String("module", modFile.Module.Mod.Path),
			slog.String("from", oldver),
			slog.String("to", minGoVer),
		)

		if err := modFile.AddGoStmt(minGoVer); err != nil {
			return err
		}
		modFile.Cleanup()
		// Write the updated file back
		updatedModFile, err := modFile.Format()
		if err != nil {
			return err
		}
		if err := os.WriteFile(goModPath, updatedModFile, 0644); err != nil {
			return err
		}
		if err := goModTidy(sess, filepath.Dir(goModPath)); err != nil {
			return err
		}
		if err := gitAddAndCommit(sess, r.WD, "dep", pkg.LocalPath, fmt.Sprintf("update go version from %s to %s", oldver, minGoVer)); err != nil {
			return err
		}
	}

	sess.Log().Info(
		"loading package",
		slog.String("module", pkg.Import),
		slog.String("path", pkg.LocalPath),
	)

	// check if package needs release
	if err := pkg.CheckNeedsRelease(); err != nil {
		return err
	}

	if pkg.NeedsRelease {
		sess.Log().Ok(
			"loaded package",
			slog.String("module", pkg.Import),
			slog.String("path", pkg.LocalPath),
		)

	} else {
		sess.Log().Msg(
			"skiped package - no changes",
			slog.String("module", pkg.Import),
			slog.String("path", pkg.LocalPath),
		)
	}

	return nil
}

type Package struct {
	ModFilePath  string // full path to go mod file
	LocalPath    string // relative path to the monorepo root directory
	TagPrefix    string // tag prefix for the package
	Import       string // import path
	NeedsRelease bool
}

func (p *Package) CheckNeedsRelease() error {
	return nil
}

func gitAddAndCommit(sess *happy.Session, wd, typ, scope, msg string) error {
	gitadd := exec.Command("git", "add", ".")
	gitadd.Dir = wd
	if err := cli.RunCommand(sess, gitadd); err != nil {
		return err
	}
	commitMsg := fmt.Sprintf("%s(%s): %s", typ, scope, msg)
	gitcommit := exec.Command("git", "commit", "-sm", commitMsg)
	gitcommit.Dir = wd
	if err := cli.RunCommand(sess, gitcommit); err != nil {
		return err
	}
	return nil
}

func goModTidy(sess *happy.Session, wd string) error {
	gomodtidy := exec.Command("go", "mod", "tidy")
	gomodtidy.Dir = wd
	if err := cli.RunCommand(sess, gomodtidy); err != nil {
		return err
	}
	return nil
}
