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

// Package cli is provides implementations of happy.Application
// command line interfaces
package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrCommand        = happyx.NewError("command error")
	ErrCommandAction  = happyx.NewError("command action error")
	ErrCommandInvalid = happyx.NewError("invalid command definition")
	ErrCommandArgs    = happyx.NewError("command arguments error")
	ErrCommandFlags   = happyx.NewError("command flags error")
	ErrPanic          = happyx.NewError("there was panic, check logs for more info")
)

// ExecCommand wraps ExecCommandRaw to return output as string.
func ExecCommand(sess happy.Session, cmd *exec.Cmd) (string, error) {
	out, err := ExecCommandRaw(sess, cmd)
	return string(bytes.TrimSpace(out)), err
}

// ExecCommandRaw wraps and executes provided command and returns its
// CombinedOutput. It ensures that -x flag is taken into account and
// Command is Session Context aware.
func ExecCommandRaw(sess happy.Session, cmd *exec.Cmd) ([]byte, error) {
	return execCommandRaw(sess, cmd)
}

// RunCommand wraps and executes provided command and writes
// its Stdout and Stderr. It ensures that -x flag is taken
// into account and Command is Session Context aware.
func RunCommand(sess happy.Session, cmd *exec.Cmd) error {
	return runCommand(sess, cmd)
}

// AskForConfirmation gets (y/Y)es or (n/N)o from cli input.
func AskForConfirmation(q string) bool {
	var response string
	fmt.Fprintln(os.Stdout, q, "(y/Y)es or (n/N)o?")

	if _, err := fmt.Scanln(&response); err != nil {
		return false
	}

	switch strings.ToLower(response) {
	case "y", "Y", "yes":
		return true
	case "n", "N", "no":
		return false
	default:
		fmt.Fprintln(
			os.Stdout,
			"I'm sorry but I didn't get what you meant, please type (y/Y)es or (n/N)o and then press enter:")

		return AskForConfirmation(q)
	}
}

func runCommand(sess happy.Session, cmd *exec.Cmd) error {
	sess.Log().Debugf("exec: %s", cmd.String())

	if sess.Get("flags.x").Bool() {
		fmt.Fprintln(os.Stdout, "cmd: "+cmd.String())
	}

	scmd := exec.CommandContext(sess, cmd.Path, cmd.Args[1:]...) //nolint: gosec
	scmd.Env = cmd.Env
	scmd.Dir = cmd.Dir
	scmd.Stdin = cmd.Stdin
	scmd.Stdout = cmd.Stdout
	scmd.Stderr = cmd.Stderr
	scmd.ExtraFiles = cmd.ExtraFiles
	cmd = scmd

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stdopipe := bufio.NewScanner(stdout)
	go func() {
		for stdopipe.Scan() {
			fmt.Fprintln(os.Stdout, stdopipe.Text())
		}
	}()
	stdepipe := bufio.NewScanner(stderr)
	go func() {
		for stdepipe.Scan() {
			fmt.Fprintln(os.Stderr, stdepipe.Text())
		}
	}()

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		//nolint: forbidigo
		fmt.Println("")
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			// ee.ExitCode(),
			sess.Log().Warnf("%s %s", ee.Error(), string(ee.Stderr))
		}

		return ErrCommand.Wrap(err)
	}
	sess.Log().Debugf("%s done", cmd.String())
	return nil
}

func execCommandRaw(sess happy.Session, cmd *exec.Cmd) ([]byte, error) {
	sess.Log().Debugf("exec: %s", cmd.String())

	if sess.Get("flags.x").Bool() {
		fmt.Fprintln(os.Stdout, "cmd: "+cmd.String())
	}

	scmd := exec.CommandContext(sess, cmd.Path, cmd.Args[1:]...) //nolint: gosec
	scmd.Env = cmd.Env
	scmd.Dir = cmd.Dir
	scmd.Stdin = cmd.Stdin
	scmd.Stdout = cmd.Stdout
	scmd.Stderr = cmd.Stderr
	scmd.ExtraFiles = cmd.ExtraFiles
	cmd = scmd

	out, err := cmd.CombinedOutput()
	if err == nil {
		sess.Log().Debugf("%s done", cmd.String())
		return out, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		sess.Log().Warnf("%s %s", ee.Error(), string(ee.Stderr))
		return nil, ErrCommand.Wrap(err)
	}
	return nil, err
}
