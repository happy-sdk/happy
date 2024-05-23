// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package module

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/internal/cmd/hap/addons/releaser/git"
	"github.com/happy-sdk/happy/sdk/cli/oscmd"
)

func (p *Package) Release(sess *happy.Session) error {
	if !p.NeedsRelease {
		return nil
	}

	p.Modfile.Cleanup()
	// Write the updated file back
	updatedModFile, err := p.Modfile.Format()
	if err != nil {
		return err
	}
	if err := os.WriteFile(p.ModFilePath, updatedModFile, 0644); err != nil {
		return err
	}
	sess.Log().Info("updated go.mod", slog.String("package", p.Import))

	gomodtidy := exec.Command("go", "mod", "tidy")
	gomodtidy.Dir = p.Dir
	if err := oscmd.Run(sess, gomodtidy); err != nil {
		return err
	}
	localpath := strings.TrimSuffix(p.TagPrefix, "/")

	if err := git.AddAndCommit(sess, sess.Get("releaser.working.directory").String(), "dep", localpath, "update go.mod deps"); err != nil {
		return err
	}

	origin := sess.Get("releaser.git.remote.name").String()
	branch := sess.Get("releaser.git.branch").String()

	gitpush := exec.Command("git", "push", origin, branch)
	gitpush.Dir = sess.Get("releaser.working.directory").String()
	if err := oscmd.Run(sess, gitpush); err != nil {
		return err
	}

	if strings.Contains(p.Import, "internal") {
		sess.Log().Warn("skipping internal package release", slog.String("package", p.Import))
		return nil
	}

	gitag := exec.Command("git", "tag", "-sm", fmt.Sprintf("%q", p.NextRelease), p.NextRelease)
	gitag.Dir = sess.Get("releaser.working.directory").String()
	if err := oscmd.Run(sess, gitag); err != nil {
		return err
	}

	gitpushtag := exec.Command("git", "push", origin, p.NextRelease)
	gitpushtag.Dir = sess.Get("releaser.working.directory").String()
	if err := oscmd.Run(sess, gitpushtag); err != nil {
		return err
	}

	sess.Log().Ok("released package", slog.String("package", p.Import), slog.String("version", p.NextRelease))
	return nil
}
