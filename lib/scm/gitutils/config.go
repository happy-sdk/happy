// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gitutils

import (
	"os/exec"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/session"
)

// Config is a reusable, persistable settings block for a project's git
// identity, current branch, and remote. Any Happy project's own settings
// tree can embed this directly (e.g. a field tagged key:"git" of type
// Config) instead of re-declaring the same fields.
type Config struct {
	CommitterName  settings.String `key:"committer.name,save"`
	CommitterEmail settings.String `key:"committer.email,save"`
	AuthorName     settings.String `key:"author.name,save"`
	AuthorEmail    settings.String `key:"author.email,save"`
	Branch         settings.String `key:"branch,save" default:"main"`
	RemoteName     settings.String `key:"remote.name,save" default:"origin"`
	RemoteURL      settings.String `key:"remote.url,save"`
}

func (c *Config) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}

// LoadConfig discovers a project's git committer identity, current branch,
// and remote for dir. Every field is optional best-effort data: a project
// dir without a configured git identity, a detached HEAD, no remote, or no
// git binary at all must not stop a caller from booting, so each lookup
// degrades independently to its zero value on failure instead of aborting
// the whole call.
func LoadConfig(sess *session.Context, dir string) (cnf Config, err error) {
	if _, lookErr := exec.LookPath("git"); lookErr != nil {
		return cnf, nil
	}

	if name, email, ierr := CommitterIdentity(sess, dir); ierr == nil {
		cnf.CommitterName = settings.String(name)
		cnf.CommitterEmail = settings.String(email)
		cnf.AuthorName = cnf.CommitterName
		cnf.AuthorEmail = cnf.CommitterEmail
	}

	if branch, berr := CurrentBranch(sess, dir); berr == nil {
		cnf.Branch = settings.String(branch)
	}

	if remoteName, remoteURL, rerr := CurrentRemote(sess, dir); rerr == nil {
		cnf.RemoteName = settings.String(remoteName)
		cnf.RemoteURL = settings.String(remoteURL)
	}

	return cnf, nil
}
