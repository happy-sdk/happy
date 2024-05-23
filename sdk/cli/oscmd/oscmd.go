// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package oscmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/logging"
)

// Exec wraps ExecRaw to return output as string.
func Exec(sess *happy.Session, cmd *exec.Cmd) (string, error) {
	out, err := ExecRaw(sess, cmd)
	return string(bytes.TrimSpace(out)), err
}

// ExecRaw wraps and executes provided command and returns its
// CombinedOutput. It ensures that -x flag is taken into account and
// Command is Session Context aware.
func ExecRaw(sess *happy.Session, cmd *exec.Cmd) ([]byte, error) {
	return execCommandRaw(sess, cmd)
}

// Run wraps and executes provided command and writes
// its Stdout and Stderr. It ensures that -x flag is taken
// into account and Command is Session Context aware.
func Run(sess *happy.Session, cmd *exec.Cmd) error {
	return run(sess, cmd)
}

func run(sess *happy.Session, cmd *exec.Cmd) error {
	sess.Log().Debug("exec: ", slog.String("cmd", cmd.String()))

	if sess.Get("app.main.exec.x").Bool() {
		sess.Log().LogDepth(4, logging.LevelAlways, cmd.String())
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
			fmt.Println(string(ee.Stderr))
			sess.Log().Error(ee.Error())
		}

		return err
	}
	sess.Log().Debug(cmd.String(), slog.Int("exit", 0))
	return nil
}

func execCommandRaw(sess *happy.Session, cmd *exec.Cmd) ([]byte, error) {
	sess.Log().Debug("exec: ", slog.String("cmd", cmd.String()))

	if sess.Get("app.main.exec.x").Bool() {
		sess.Log().LogDepth(4, logging.LevelAlways, cmd.String())
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
		sess.Log().Debug(cmd.String(), slog.Int("exit", 0))
		return out, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		fmt.Println(string(ee.Stderr))
		sess.Log().Error(ee.Error())
		return out, err
	}
	return out, err
}
