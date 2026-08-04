// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

//go:build windows

package exec

import "os/exec"

// configureProcessGroup is a no-op on Windows, which has no process groups in
// the POSIX sense. Killing the child is the best available behaviour without
// pulling in Job Object handling.
func configureProcessGroup(cmd *exec.Cmd) {}

func terminateGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
