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

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/session"
)

var (
	ErrCommandInvalid = errors.New("invalid command definition")
	ErrCommandArgs    = errors.New("command arguments error")
	ErrCommandFlags   = errors.New("command flags error")
	ErrPanic          = errors.New("there was panic, check logs for more info")
)

// Common CLI flags which are automatically attached to the CLI ubnless disabled ins settings.
// You still can manually add them to your CLI if you want to.
var (
	FlagVersion     = varflag.BoolFunc("version", false, "print application version")
	FlagHelp        = varflag.BoolFunc("help", false, "display help or help for the command. [...command --help]", "h")
	FlagX           = varflag.BoolFunc("x", false, "the -x flag prints all the cli commands as they are executed.")
	FlagSystemDebug = varflag.BoolFunc("system-debug", false, "enable system debug log level (very verbose)")
	FlagDebug       = varflag.BoolFunc("debug", false, "enable debug log level")
	FlagVerbose     = varflag.BoolFunc("verbose", false, "enable verbose log level", "v")
)

type Settings struct {
	Name            settings.String `default:"" desc:"Name of executable file"`
	MainMinArgs     settings.Uint   `default:"0" desc:"Minimum number of arguments for a application main"`
	MainMaxArgs     settings.Uint   `default:"0" desc:"Maximum number of arguments for a application main"`
	WithConfigCmd   settings.Bool   `default:"false" desc:"Add the config command in the CLI"`
	WithGlobalFlags settings.Bool   `default:"false" desc:"Add the default global flags automatically in the CLI"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

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
		sess.Log().LogDepth(4, logging.LevelAlways, cmd.String())
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
