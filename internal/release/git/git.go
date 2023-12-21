// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package git

import (
	"fmt"
	"os/exec"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/cli"
)

func AddAndCommit(sess *happy.Session, wd, typ, scope, msg string) error {
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
