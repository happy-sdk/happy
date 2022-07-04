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

//go:build (linux && !android) || freebsd || windows || openbsd || darwin || !js

package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/mkungla/happy"
)

func execCommandRaw(ctx happy.Session, cmd *exec.Cmd) ([]byte, error) {
	ctx.Log().Debugf("exec: %s", cmd.String())
	if ctx.Flag("x").Present() {
		fmt.Fprintln(os.Stdout, "cmd: "+cmd.String())
	}

	scmd := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...) //nolint: gosec
	scmd.Env = cmd.Env
	scmd.Dir = cmd.Dir
	scmd.Stdin = cmd.Stdin
	scmd.Stdout = cmd.Stdout
	scmd.Stderr = cmd.Stderr
	scmd.ExtraFiles = cmd.ExtraFiles
	cmd = scmd

	out, err := cmd.CombinedOutput()
	if err == nil {
		ctx.Log().Debugf("%s done", cmd.String())
		return out, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		ctx.Log().Errorf("%s %s", ee.Error(), string(ee.Stderr))
		return nil, fmt.Errorf("%w: %s", ErrCommandExec, err)
	}
	return nil, err
}

func runCommand(ctx happy.Session, cmd *exec.Cmd) error {
	ctx.Log().Debugf("exec: %s", cmd.String())
	if ctx.Flag("x").Present() {
		fmt.Fprintln(os.Stdout, "cmd: "+cmd.String())
	}

	scmd := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...) //nolint: gosec
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
			fmt.Fprintln(os.Stdout, stdepipe.Text())
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
			ctx.Log().Errorf("%s %s", ee.Error(), string(ee.Stderr))
		}

		return fmt.Errorf("%w: %s", ErrCommandExec, err)
	}
	//nolint: forbidigo
	fmt.Println("")
	ctx.Log().Debugf("%s done", cmd.String())
	return nil
}
