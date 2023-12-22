// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package releaser

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy-go/internal/release/module"
	"github.com/happy-sdk/happy/sdk/cli"
)

type Releaser struct {
	wd           string
	skipPackages []string
	packages     []*module.Package
}

func New(wd string, skipPkgs []string) *Releaser {

	if strings.HasSuffix(wd, "internal/release") {
		wd = filepath.Join(wd, "../../")
	}

	return &Releaser{
		wd:           wd,
		skipPackages: skipPkgs,
	}
}

func (r *Releaser) Before(sess *happy.Session, args happy.Args) error {
	sess.Log().Debug("preparing releaser")

	if !strings.HasSuffix(r.wd, "happy-go") {
		return fmt.Errorf("can noit call release from location %s", r.wd)
	}

	sess.Log().Msg("working directory", slog.String("dir", r.wd))

	if err := r.loadModules(sess); err != nil {
		return err
	}
	if len(r.packages) == 0 {
		return fmt.Errorf("no packages need to release")
	}
	sess.Log().Msg("loaded modules", slog.Int("count", len(r.packages)))

	return nil
}

func (r *Releaser) Do(sess *happy.Session, args happy.Args) error {
	sess.Log().NotImplemented("releasing packages")

	for _, pkg := range r.packages {
		if pkg.NeedsRelease {
			sess.Log().Msg(
				"new release",
				slog.String("package", pkg.Import),
				slog.Group("version",
					slog.String("current", pkg.LastRelease),
					slog.String("next", pkg.NextRelease),
				),
			)
		} else {
			sess.Log().Msg(
				"no release",
				slog.String("package", pkg.Import),
				slog.Group("version",
					slog.String("current", pkg.LastRelease),
				),
			)
		}
	}
	if !cli.AskForConfirmation("Do you want to continue?") {
		return errors.New("user aborted")
	}

	return r.release(sess)
}

func (r *Releaser) loadModules(sess *happy.Session) error {
	minGoVer := sess.Profile().Get("happy-go.go.version.min").String()
	sess.Log().Msg("Min Go Version", slog.String("version", minGoVer))
	if err := filepath.Walk(r.wd, func(path string, info os.FileInfo, err error) error {
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
		pkg, err := module.Load(sess, r.wd, goModPath, minGoVer)
		if err != nil {
			if errors.Is(err, module.ErrSkipPackage) {
				return nil
			}
			return err
		}
		for _, skip := range r.skipPackages {
			if pkg.LocalPath == skip {
				return nil
			}
		}
		r.packages = append(r.packages, pkg)
		return nil
	}); err != nil {
		return err
	}

	for _, pkg := range r.packages {
		if err := pkg.Prepare(sess, r.wd); err != nil {
			return err
		}
	}
	return nil
}

func (r *Releaser) release(sess *happy.Session) error {
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = r.wd
	if err := cli.RunCommand(sess, pushCmd); err != nil {
		return err
	}

	for _, pkg := range r.packages {
		if !pkg.NeedsRelease {
			continue
		}
		if err := pkg.Release(sess, r.wd); err != nil {
			return err
		}
	}
	return nil
}
