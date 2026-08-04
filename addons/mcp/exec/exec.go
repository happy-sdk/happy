// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

// Package exec runs the commands declared by a repository's agent manifest.
//
// Manifests declare commands that execute on a maintainer's machine. That is
// the point of them, and it is also the risk, so this package is deliberately
// narrow:
//
//   - argv only, never a shell
//   - inputs interpolate into separate argv entries, never into a command line
//   - only fields declared in the tool's input schema may be referenced
//   - the working directory must resolve inside the declaring repository
//   - every run is bounded by a timeout and killed as a process group
package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	Error          = errors.New("exec")
	ErrOutsideRepo = fmt.Errorf("%w: path escapes repository", Error)
	ErrUnknownArg  = fmt.Errorf("%w: unknown input", Error)
	ErrTimeout     = fmt.Errorf("%w: timed out", Error)
)

// DefaultTimeout bounds a tool that declares none.
const DefaultTimeout = 2 * time.Minute

// MaxOutput caps captured output so a runaway command cannot exhaust memory or
// blow past what a client will accept in one response.
const MaxOutput = 1 << 20 // 1 MiB

// placeholder matches {{ .field }} with optional surrounding spaces.
var placeholder = regexp.MustCompile(`\{\{\s*\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// Request is one command to run on behalf of a declarative tool.
type Request struct {
	// RepoDir is the repository root. Everything must resolve inside it.
	RepoDir string
	// Command is argv. Entries may contain {{ .field }} placeholders.
	Command []string
	// Cwd is repository-relative and may contain placeholders.
	Cwd string
	// Args are conditional argv additions.
	Args []ArgRule
	// Inputs are the validated tool arguments available to placeholders.
	Inputs map[string]any
	// Declared is the set of input names the tool's schema declares. A
	// placeholder naming a declared input the caller omitted renders empty;
	// one naming anything outside this set is a manifest bug and fails. When
	// nil every unsupplied placeholder is treated as a bug.
	Declared map[string]bool
	// Timeout bounds the run.
	Timeout time.Duration
}

// ArgRule appends Add when If interpolates to a non-empty value.
type ArgRule struct {
	If  string
	Add []string
}

// Result is the outcome of a run. A non-zero exit is a result, not an error:
// the output is what the caller wanted, and an agent needs to see it.
type Result struct {
	Output    string
	ExitCode  int
	Truncated bool
	Duration  time.Duration
}

// Run executes the request. It returns an error only when the request itself
// is invalid or the command could not be started or completed - a command that
// runs and fails reports through Result.ExitCode.
func Run(ctx context.Context, req Request) (*Result, error) {
	if len(req.Command) == 0 {
		return nil, fmt.Errorf("%w: empty command", Error)
	}

	argv, err := interpolateAll(req.Command, req.Inputs, req.Declared)
	if err != nil {
		return nil, err
	}
	for _, rule := range req.Args {
		cond, err := interpolate(rule.If, req.Inputs, req.Declared)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(cond) == "" {
			continue
		}
		add, err := interpolateAll(rule.Add, req.Inputs, req.Declared)
		if err != nil {
			return nil, err
		}
		argv = append(argv, add...)
	}

	cwd, err := interpolate(req.Cwd, req.Inputs, req.Declared)
	if err != nil {
		return nil, err
	}
	dir, err := resolveCwd(req.RepoDir, cwd)
	if err != nil {
		return nil, err
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	// Kill the whole group: a build or test command spawns children, and
	// cancelling only the parent leaves them running past the timeout.
	configureProcessGroup(cmd)
	cmd.Cancel = func() error { return terminateGroup(cmd) }

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	started := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(started)

	res := &Result{Duration: elapsed}
	out := buf.Bytes()
	if len(out) > MaxOutput {
		out = out[:MaxOutput]
		res.Truncated = true
	}
	res.Output = string(out)

	if runErr != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			return res, fmt.Errorf("%w after %s", ErrTimeout, timeout)
		case errors.As(runErr, &exitErr):
			res.ExitCode = exitErr.ExitCode()
		default:
			return res, fmt.Errorf("%w: %s", Error, runErr.Error())
		}
	}
	return res, nil
}

// resolveCwd confines the working directory to the repository. A manifest that
// points outside its own root is rejected rather than clamped, because the
// intent is unrecoverable at that point.
func resolveCwd(repoDir, cwd string) (string, error) {
	root, err := filepath.Abs(repoDir)
	if err != nil {
		return "", err
	}
	if cwd == "" {
		cwd = "."
	}
	if filepath.IsAbs(cwd) {
		return "", fmt.Errorf("%w: cwd %q must be repository-relative", ErrOutsideRepo, cwd)
	}
	joined := filepath.Clean(filepath.Join(root, cwd))
	rel, err := filepath.Rel(root, joined)
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrOutsideRepo, cwd)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q resolves outside %s", ErrOutsideRepo, cwd, root)
	}
	return joined, nil
}

func interpolateAll(in []string, inputs map[string]any, declared map[string]bool) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, s := range in {
		v, err := interpolate(s, inputs, declared)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// interpolate substitutes {{ .field }} from inputs.
//
// An omitted but declared input renders empty, which is how optional arguments
// stay optional. Referencing a field the schema never declares is an error
// rather than an empty string: that is a manifest bug, and silently producing
// a command with a hole in it and running it anyway is the worst way to
// surface one.
func interpolate(s string, inputs map[string]any, declared map[string]bool) (string, error) {
	if s == "" || !strings.Contains(s, "{{") {
		return s, nil
	}
	var bad string
	out := placeholder.ReplaceAllStringFunc(s, func(match string) string {
		name := placeholder.FindStringSubmatch(match)[1]
		v, ok := inputs[name]
		if !ok {
			if !declared[name] && bad == "" {
				bad = name
			}
			return ""
		}
		return valueString(v)
	})
	if bad != "" {
		return "", fmt.Errorf("%w: %q is not among the declared inputs", ErrUnknownArg, bad)
	}
	if strings.Contains(out, "{{") {
		return "", fmt.Errorf("%w: unresolved placeholder in %q", Error, s)
	}
	return out, nil
}

func valueString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		// JSON numbers decode as float64; render integers without a suffix so
		// a threshold of 95 becomes "95" rather than "95.000000".
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	default:
		return fmt.Sprint(t)
	}
}
