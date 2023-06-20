// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

// Package cli provides utilities for happy command line interfaces.
package cli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/happy-sdk/happy"
	"golang.org/x/exp/slog"
	"golang.org/x/term"
)

var (
	// ErrCommand        = happyx.NewError("command error")
	// ErrCommandAction  = happyx.NewError("command action error")
	ErrCommandInvalid = errors.New("invalid command definition")
	ErrCommandArgs    = errors.New("command arguments error")
	ErrCommandFlags   = errors.New("command flags error")
	ErrPanic          = errors.New("there was panic, check logs for more info")
)

// ExecCommand wraps ExecCommandRaw to return output as string.
func ExecCommand(sess *happy.Session, cmd *exec.Cmd) (string, error) {
	out, err := ExecCommandRaw(sess, cmd)
	return string(bytes.TrimSpace(out)), err
}

// ExecCommandRaw wraps and executes provided command and returns its
// CombinedOutput. It ensures that -x flag is taken into account and
// Command is Session Context aware.
func ExecCommandRaw(sess *happy.Session, cmd *exec.Cmd) ([]byte, error) {
	sess.Log().Debug("exec: ", slog.String("cmd", cmd.String()))

	if sess.Get("app.cli.x").Bool() {
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
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			fmt.Println(string(ee.Stderr))
			sess.Log().Error("cmd error", ee)
		}
		return out, err
	}
	sess.Log().Debug(cmd.String(), slog.Int("exit", 0))
	return out, nil
}

// RunCommand wraps and executes provided command and writes
// its Stdout and Stderr. It ensures that -x flag is taken
// into account and Command is Session Context aware.
func RunCommand(sess *happy.Session, cmd *exec.Cmd) error {
	sess.Log().Debug("exec: ", slog.String("cmd", cmd.String()))

	if sess.Get("app.cli.x").Bool() {
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

	go scanPipe(stdout, os.Stdout)
	go scanPipe(stderr, os.Stderr)

	if err := cmd.Start(); err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		errChan <- cmd.Wait()
	}()

	select {
	case <-sess.Done():
		// If the session is done, stop the command
		if err := cmd.Process.Kill(); err != nil {
			return err
		}

		if sess.Err() == context.Canceled {
			sess.Log().Debug(cmd.String(), slog.Int("exit", 0))
			return nil
		} else {
			return sess.Err()
		}
	case err = <-errChan:
		return err
	}
}

// AskForConfirmation gets (y/Y)es or (n/N)o from cli input.
func AskForConfirmation(q string) bool {
	var response string
	fmt.Fprintln(os.Stdout, q, "(y/Y)es or (n/N)o?")

	if _, err := fmt.Scanln(&response); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
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

func AskForInput(q string) string {
	fmt.Fprintln(os.Stdout, q)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	// Remove the newline character at the end
	response = strings.TrimSuffix(response, "\n")

	return response
}

func AskForSecret(q string) string {
	fmt.Fprintln(os.Stdout, q)
	bpasswd, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return ""
	}
	return strings.TrimSpace(string(bpasswd))
}

func scanPipe(pipe io.Reader, output io.Writer) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		fmt.Fprintln(output, scanner.Text())
	}
}
