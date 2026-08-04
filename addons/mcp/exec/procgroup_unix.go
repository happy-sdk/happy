// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

//go:build !windows

package exec

import (
	"os/exec"
	"syscall"
)

// configureProcessGroup puts the command in its own process group so the whole
// tree can be signalled on timeout. Build and test commands spawn children,
// and killing only the parent leaves them running.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminateGroup signals the process group, giving it a chance to exit
// cleanly. The negative pid addresses the group rather than the leader.
func terminateGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		// The group may already be gone, or was never created; fall back to
		// the process itself rather than leaving it running.
		return cmd.Process.Kill()
	}
	return nil
}
