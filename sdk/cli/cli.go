// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

// Package cli provides utilities for happy command line interfaces.
package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/term"

	_ "github.com/happy-sdk/happy/sdk/cli/i18n"
)

var (
	ErrCommandInvalid = errors.New("invalid command definition")
	ErrCommandArgs    = errors.New("command arguments error")
	ErrCommandFlags   = errors.New("command flags error")
	ErrPanic          = errors.New("there was panic, check logs for more info")
)

// AskForConfirmation gets (y/Y)es or (n/N)o from cli input.
func AskForConfirmation(q string) bool {
	var response string
	_, _ = fmt.Fprintln(os.Stdout, q, "(y/Y)es or (n/N)o?")

	if _, err := fmt.Scanln(&response); err != nil {
		return false
	}

	switch strings.ToLower(response) {
	case "y", "Y", "yes":
		return true
	case "n", "N", "no":
		return false
	default:
		_, _ = fmt.Fprintln(
			os.Stdout,
			"I'm sorry but I didn't get what you meant, please type (y/Y)es or (n/N)o and then press enter:")

		return AskForConfirmation(q)
	}
}

func AskForInput(q string) string {
	_, _ = fmt.Fprintln(os.Stdout, q)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(response)
}

func AskForSecret(q string) string {
	_, _ = fmt.Fprintln(os.Stdout, q)

	var secret []byte
	for {
		char, err := getChar()
		if err != nil {
			return ""
		}

		switch char {
		case '\r', '\n':
			fmt.Println()
			return string(secret)
		case 127, 8:
			if len(secret) > 0 {
				secret = secret[:len(secret)-1]
				fmt.Print("\b \b") // Move back, print space, move back again
			}
		case 3: // Ctrl+C
			fmt.Println()
			return ""
		default:
			if char >= 32 && char <= 126 {
				secret = append(secret, char)
				fmt.Print("*")
			}
		}
	}
}

func getChar() (byte, error) {
	// Save original terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}()

	var buf [1]byte
	_, err = os.Stdin.Read(buf[:])
	return buf[0], err
}

// Exec wraps ExecRaw to return output as string.
func Exec(sess *session.Context, cmd *exec.Cmd) (string, error) {
	out, err := ExecRaw(sess, cmd)
	return string(bytes.TrimSpace(out)), err
}

// ExecRaw wraps and executes provided command and returns its
// CombinedOutput. It ensures that -x flag is taken into account and
// Command is Session Context aware.
func ExecRaw(sess *session.Context, cmd *exec.Cmd) ([]byte, error) {
	return execCommandRaw(sess, cmd)
}

// Run wraps and executes provided command and writes
// its Stdout and Stderr. It ensures that -x flag is taken
// into account and Command is Session Context aware.
func Run(sess *session.Context, cmd *exec.Cmd) error {
	return run(sess, cmd)
}

func run(sess *session.Context, cmd *exec.Cmd) error {
	sess.Log().Debug("exec: ", slog.String("cmd", cmd.String()))

	if sess.Get("app.main.exec.x").Bool() {
		_ = sess.Log().LogDepth(4, logging.LevelOut, cmd.String())
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
			_, _ = fmt.Fprintln(os.Stdout, stdopipe.Text())
		}
	}()
	stdepipe := bufio.NewScanner(stderr)
	go func() {
		for stdepipe.Scan() {
			_, _ = fmt.Fprintln(os.Stderr, stdepipe.Text())
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

func execCommandRaw(sess *session.Context, cmd *exec.Cmd) ([]byte, error) {
	sess.Log().Debug("exec: ", slog.String("cmd", cmd.String()))

	if sess.Get("app.main.exec.x").Bool() {
		_ = sess.Log().LogDepth(4, logging.LevelOut, cmd.String())
	}

	scmd := exec.CommandContext(sess, cmd.Path, cmd.Args[1:]...) //nolint: gosec
	scmd.Env = cmd.Env
	scmd.Dir = cmd.Dir

	// Create buffers to capture stdout and stderr separately
	var stdoutBuf, stderrBuf bytes.Buffer

	// Always redirect stdout and stderr to our buffers, ignoring any previously set values
	// This ensures we don't write to the original stdout/stderr
	scmd.Stdout = &stdoutBuf
	scmd.Stderr = &stderrBuf

	scmd.ExtraFiles = cmd.ExtraFiles
	cmd = scmd

	// Execute command
	err := cmd.Run()

	// Get stdout content
	stdoutBytes := stdoutBuf.Bytes()

	// If no error, just return the stdout
	if err == nil {
		return stdoutBytes, nil
	}

	// On error, clean stderr by replacing newlines with spaces
	cleanedStderr := strings.ReplaceAll(stderrBuf.String(), "\n", " ")

	var ee *exec.ExitError
	if errors.As(err, &ee) {
		// Create new error with original error and cleaned stderr appended
		enhancedErr := fmt.Errorf("%w: %s", err, cleanedStderr)
		return stdoutBytes, enhancedErr
	}

	// For other types of errors, also append cleaned stderr
	enhancedErr := fmt.Errorf("%w: %s", err, cleanedStderr)
	return stdoutBytes, enhancedErr
}
