// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cli provides sdk to add cli commands to happy application.
package cli

import (
	"bytes"
	"errors"
	"os/exec"

	"github.com/mkungla/happy"
)

var (
	ErrCommand        = errors.New("command error")
	ErrCommandInvalid = errors.New("invalid command definition")
	ErrCommandArgs    = errors.New("command arguments error")
	ErrCommandFlags   = errors.New("command flags error")
	ErrCommandExec    = errors.New("command execution error")
	ErrPanic          = errors.New("there was panic, check logs for more info")
)

// ExecCommand wraps ExecCommandRaw to return output as string.
func ExecCommand(ctx happy.Session, cmd *exec.Cmd) (string, error) {
	out, err := ExecCommandRaw(ctx, cmd)
	return string(bytes.TrimSpace(out)), err
}

// ExecCommandRaw wraps and executes provided command and returns its
// CombinedOutput. It ensures that -x flag is taken into account and
// Command is RuntimeContext aware.
func ExecCommandRaw(ctx happy.Session, cmd *exec.Cmd) ([]byte, error) {
	return execCommandRaw(ctx, cmd)
}

// RunCommand wraps and executes provided command and prints
// its Stdin and Stdout with logger.Line. It ensures that -x flag is taken
// into account and Command is RuntimeContext aware.
func RunCommand(ctx happy.Session, cmd *exec.Cmd) error {
	return runCommand(ctx, cmd)
}
