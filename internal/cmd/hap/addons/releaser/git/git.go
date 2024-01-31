// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package git

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/cli"
)

func AddAndCommit(sess *happy.Session, wd, typ, scope, msg string) error {
	// Check for uncommitted changes
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = wd
	status, err := cli.ExecCommandRaw(sess, statusCmd)
	if err != nil {
		return err
	}
	if bytes.TrimSpace(status) == nil {
		return nil
	}

	gitadd := exec.Command("git", "add", "-A")
	gitadd.Dir = wd
	if err := cli.RunCommand(sess, gitadd); err != nil {
		return err
	}
	commitMsg := fmt.Sprintf("%s(%s): %s", typ, scope, msg)
	if scope == "" {
		commitMsg = fmt.Sprintf("%s: %s", typ, msg)
	}
	gitcommit := exec.Command("git", "commit", "-sm", commitMsg)
	gitcommit.Dir = wd
	if err := cli.RunCommand(sess, gitcommit); err != nil {
		return err
	}
	return nil
}
