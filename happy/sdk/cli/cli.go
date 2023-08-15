// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

// Package cli provides utilities for happy command line interfaces.
package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"log/slog"

	"github.com/happy-sdk/happy"
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

// ExecCommand wraps ExecCommandRaw to trim the output and return it as a string.
// If an error occurs during execution, it is returned along with the output.
func ExecCommand(sess *happy.Session, cmd *exec.Cmd) (string, error) {
	out, err := ExecCommandRaw(sess, cmd)
	return string(bytes.TrimSpace(out)), err
}

// ExecCommandRaw wraps and executes the provided command, returning its combined
// output as a byte slice. It ensures that the -x flag is taken into account, and
// that the command is context-aware in relation to the session.
// If an error occurs during execution, it is logged and returned along with the output.
func ExecCommandRaw(sess *happy.Session, cmd *exec.Cmd) ([]byte, error) {
	sess.Log().Debug("exec: ", slog.String("cmd", cmd.String()))

	xcmd := prepareCommand(sess, cmd)

	out, err := xcmd.CombinedOutput()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			fmt.Println(string(ee.Stderr))
			sess.Log().Error("cmd error", ee)
		}
		return out, err
	}
	sess.Log().Debug(xcmd.String(), slog.Int("exit", 0))
	return out, nil
}

// RunCommand wraps and executes the provided command, writing its Stdout and Stderr.
// It ensures that the -x flag is taken into account, and that the command is
// context-aware in relation to the session. If an error occurs during execution,
// the command is stopped, and the error is returned.
func RunCommand(sess *happy.Session, cmd *exec.Cmd) error {
	sess.Log().Debug("exec: ", slog.String("cmd", cmd.String()))

	xcmd := prepareCommand(sess, cmd)

	stderr, err := xcmd.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := xcmd.StdoutPipe()
	if err != nil {
		return err
	}

	go scanPipe(stdout, os.Stdout)
	go scanPipe(stderr, os.Stderr)

	if err := xcmd.Start(); err != nil {
		return err
	}

	exiterr := xcmd.Wait()
	if exiterr != nil {
		var ee *exec.ExitError
		if errors.As(exiterr, &ee) {
			// Print stderr if command failed and it has output
			if len(ee.Stderr) > 0 {
				fmt.Println("stderr:\n" + string(ee.Stderr))
			}
		}
		if xcmd.Terminated() {
			fmt.Println("terminated", exiterr)
			return nil
		}
		return exiterr
	}
	return nil
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

type Cmd struct {
	*exec.Cmd
	mu         sync.Mutex
	terminated bool
}

func (cmd *Cmd) Terminated() bool {
	cmd.mu.Lock()
	defer cmd.mu.Unlock()
	return cmd.terminated
}

func prepareCommand(sess *happy.Session, cmd *exec.Cmd) *Cmd {
	if sess == nil {
		panic("nil Session")
	}

	if sess.Get("app.cli.x").Bool() {
		fmt.Fprintln(os.Stdout, "cmd: "+cmd.String())
	}

	xcmd := &Cmd{
		Cmd: exec.CommandContext(sess, cmd.Path, cmd.Args[1:]...),
	}
	xcmd.Env = cmd.Env
	xcmd.Dir = cmd.Dir
	xcmd.Stdin = cmd.Stdin
	xcmd.Stdout = cmd.Stdout
	xcmd.Stderr = cmd.Stderr
	xcmd.ExtraFiles = cmd.ExtraFiles

	xcmd.Cancel = func() error {
		xcmd.mu.Lock()
		xcmd.terminated = true
		xcmd.mu.Unlock()

		err := xcmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			return err
		}
		return nil
	}

	return xcmd
}

func scanPipe(pipe io.Reader, output io.Writer) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		fmt.Fprintln(output, scanner.Text())
	}
}
