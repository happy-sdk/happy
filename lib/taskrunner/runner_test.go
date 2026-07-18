// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestRunnerNewDefaultsName(t *testing.T) {
	tr := New("")
	testutils.Equal(t, 1, len(tr.tasks), "expected New to register one root reporter task")
	testutils.Equal(t, "taskrunner", tr.tasks[0].name, "expected default name fallback")
}

func TestRunnerNewCustomName(t *testing.T) {
	tr := New("my-runner")
	testutils.Equal(t, "my-runner", tr.tasks[0].name, "expected the given name to be used")
	testutils.Equal(t, 1, tr.model.totalTasks, "expected model.totalTasks to count the root task")
}

func TestRunnerAddTaskTracksLongestName(t *testing.T) {
	tr := New("root")
	before := tr.model.totalTasks
	id := tr.Add("a-longer-task-name", func(ex *Executor) (res Result) { return Success("ok") })

	testutils.NotEqual(t, uuid.Nil, id, "expected a non-nil task id")
	testutils.Equal(t, before+1, tr.model.totalTasks, "expected totalTasks to increment")
	testutils.Equal(t, len("a-longer-task-name"), tr.model.longestTaskNameLength, "expected longestTaskNameLength to track the longer name")

	tr.Add("x", func(ex *Executor) (res Result) { return Success("ok") })
	testutils.Equal(t, len("a-longer-task-name"), tr.model.longestTaskNameLength, "a shorter task name must not shrink longestTaskNameLength")
}

func TestRunnerAddD(t *testing.T) {
	tr := New("root")
	dep := tr.Add("first", func(ex *Executor) (res Result) { return Success("ok") })
	second := tr.AddD(dep, "second", func(ex *Executor) (res Result) { return Success("ok") })

	testutils.NotEqual(t, uuid.Nil, second, "expected a non-nil task id")
	last := tr.tasks[len(tr.tasks)-1]
	testutils.Equal(t, dep, last.dependsOn, "expected AddD to record the dependency")
}

// newTestRunnerProgram wires a Runner's program to a real (headless)
// bubbletea program without going through Run()'s stdout-piping machinery,
// so executeTasks/captureOutput can be exercised directly and
// deterministically.
func newRunnerWithHeadlessProgram(t *testing.T, tr *Runner, drive func(p *tea.Program)) model {
	t.Helper()
	return runHeadless(t, *tr.model, func(p *tea.Program) {
		tr.program = p
		drive(p)
	})
}

func TestRunnerExecuteTasksAllSucceed(t *testing.T) {
	tr := New("root")
	tr.Add("task1", func(ex *Executor) (res Result) { return Success("ok") })
	tr.Add("task2", func(ex *Executor) (res Result) { return Success("ok") })

	final := newRunnerWithHeadlessProgram(t, tr, func(p *tea.Program) {
		tr.executeTasks(os.Stdout)
	})

	testutils.Equal(t, 0, final.failures, "expected no failures")
	testutils.Equal(t, 0, len(tr.failedTasks), "expected no failed task ids recorded")
}

func TestRunnerExecuteTasksSkipsDependentOnFailure(t *testing.T) {
	tr := New("root")
	failID := tr.Add("fails", func(ex *Executor) (res Result) { return Failure("boom") })
	var ranDependent bool
	tr.AddD(failID, "dependent", func(ex *Executor) (res Result) {
		ranDependent = true
		return Success("should not run")
	})

	final := newRunnerWithHeadlessProgram(t, tr, func(p *tea.Program) {
		tr.executeTasks(os.Stdout)
	})

	testutils.Assert(t, !ranDependent, "expected the dependent task's action never to run")
	testutils.Assert(t, final.failures >= 1, "expected at least one recorded failure")
	testutils.Assert(t, final.skipped >= 1, "expected the dependent task to be recorded as skipped")
	testutils.Equal(t, 2, len(tr.failedTasks), "expected both the failing and the skipped task ids recorded")
}

func TestRunnerExecuteTasksInvalidTaskID(t *testing.T) {
	tr := New("root")
	// Task{} is unreachable through the public API (NewTask always sets a
	// real uuid), but tr.tasks is a plain slice in this same package, so
	// this exercises executeTasks' defensive "invalid task id" branch
	// directly.
	tr.tasks = append(tr.tasks, Task{name: "broken"})

	final := newRunnerWithHeadlessProgram(t, tr, func(p *tea.Program) {
		tr.executeTasks(os.Stdout)
	})

	testutils.Assert(t, final.failures >= 1, "expected the invalid task to be recorded as a failure")
}

func TestRunnerExecuteTasksAggregatesSubtaskFailure(t *testing.T) {
	tr := New("root")
	tr.Add("parent", func(ex *Executor) (res Result) {
		ex.Subtask("child", func(ex *Executor) (res Result) { return Failure("child failed") })
		return Success("parent ok")
	})

	final := newRunnerWithHeadlessProgram(t, tr, func(p *tea.Program) {
		tr.executeTasks(os.Stdout)
	})

	testutils.Assert(t, final.failures >= 1, "expected the subtask failure to be reflected")
	// executeTasks records both the failing subtask's own id (via
	// runSubtasks' returned failedTasks) and the parent task's id (since
	// sstate == FAILURE), so both end up in tr.failedTasks.
	testutils.Equal(t, 2, len(tr.failedTasks), "expected both the subtask's and the parent task's ids recorded")
}

// TestRunnerExecuteTasksAllSubtasksSkippedElevatesParent is the end-to-end
// version of the runSubtasks fix: a parent task that itself runs fine but
// whose only subtask never actually runs (because that subtask depends on
// an unrelated, already-failed earlier task) must have its own result
// elevated to SKIPPED - not silently reported as if it were a plain success.
func TestRunnerExecuteTasksAllSubtasksSkippedElevatesParent(t *testing.T) {
	tr := New("root")
	failID := tr.Add("fails", func(ex *Executor) (res Result) { return Failure("boom") })
	tr.Add("parent", func(ex *Executor) (res Result) {
		ex.SubtaskD(failID, "child", func(ex *Executor) (res Result) {
			return Success("should not run")
		})
		return Success("parent ok")
	})

	final := newRunnerWithHeadlessProgram(t, tr, func(p *tea.Program) {
		tr.executeTasks(os.Stdout)
	})

	testutils.Assert(t, final.skipped >= 1, "expected the parent's result to be elevated to SKIPPED")
}

func TestRunnerExecuteTasksTicksDuringSlowAction(t *testing.T) {
	tr := New("root")
	tr.Add("slow", func(ex *Executor) (res Result) {
		// Long enough for executeTasks' 1/5s ticker to fire at least once
		// before the action returns and cancels it.
		time.Sleep(250 * time.Millisecond)
		return Success("ok")
	})

	final := newRunnerWithHeadlessProgram(t, tr, func(p *tea.Program) {
		tr.executeTasks(os.Stdout)
	})
	testutils.Assert(t, final.progressCompletedSteps > 0, "expected at least one tick to have been processed")
}

// recorderModel is a minimal tea.Model that just records every message it
// receives, for tests that need to inspect exact message content (e.g.
// captureOutput's line-splitting) rather than the real model's derived
// counters.
type recorderModel struct {
	mu *recorderState
}

type recorderState struct {
	messages []tea.Msg
}

func newRecorderModel() (*recorderModel, *recorderState) {
	st := &recorderState{}
	return &recorderModel{mu: st}, st
}

func (m *recorderModel) Init() tea.Cmd { return nil }
func (m *recorderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.mu.messages = append(m.mu.messages, msg)
	if km, ok := msg.(tea.KeyMsg); ok && km.String() == "q" {
		return m, tea.Quit
	}
	return m, nil
}
func (m *recorderModel) View() string { return "" }

func TestRunnerCaptureOutputSplitsAndTrimsLines(t *testing.T) {
	rec, st := newRecorderModel()
	p := tea.NewProgram(rec, tea.WithInput(nil), tea.WithoutRenderer())

	pr, pw, err := os.Pipe()
	testutils.NoError(t, err)

	tr := &Runner{}
	ctx, cancel := context.WithCancel(context.Background())
	tr.ctx = ctx
	tr.program = p

	done := make(chan tea.Model, 1)
	go func() {
		fm, err := p.Run()
		testutils.NoError(t, err)
		done <- fm
	}()

	go tr.captureOutput(pr)

	_, err = pw.WriteString("first line  \nsecond line\n\nthird\n")
	testutils.NoError(t, err)
	testutils.NoError(t, pw.Close())

	// Give captureOutput a moment to drain, then stop it via ctx (belt and
	// braces alongside the pipe EOF, which alone already makes Scan return
	// false) and quit the program.
	time.Sleep(50 * time.Millisecond)
	cancel()
	p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	<-done
	_ = pr.Close()

	var got []string
	for _, msg := range st.messages {
		if om, ok := msg.(OutputMsg); ok {
			got = append(got, strings.TrimPrefix(string(om), "\r"))
		}
	}
	testutils.Equal(t, 3, len(got), "expected 3 non-empty output lines, got %v", got)
	if len(got) == 3 {
		testutils.Equal(t, "first line", got[0], "expected trailing spaces trimmed")
		testutils.Equal(t, "second line", got[1], "unexpected line")
		testutils.Equal(t, "third", got[2], "expected the blank line to be dropped")
	}
}

func TestRunnerCaptureOutputStopsOnContextDone(t *testing.T) {
	rec, _ := newRecorderModel()
	p := tea.NewProgram(rec, tea.WithInput(nil), tea.WithoutRenderer())

	pr, pw, err := os.Pipe()
	testutils.NoError(t, err)
	defer func() { _ = pw.Close() }()
	defer func() { _ = pr.Close() }()

	tr := &Runner{}
	ctx, cancel := context.WithCancel(context.Background())
	tr.ctx = ctx
	tr.program = p

	done := make(chan tea.Model, 1)
	go func() {
		fm, err := p.Run()
		testutils.NoError(t, err)
		done <- fm
	}()

	captureDone := make(chan struct{})
	go func() {
		tr.captureOutput(pr)
		close(captureDone)
	}()

	cancel()
	select {
	case <-captureDone:
	case <-time.After(2 * time.Second):
		t.Fatal("captureOutput did not stop after ctx cancellation")
	}

	p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	<-done
}

// withHeadlessNewProgram makes Run() build its *tea.Program headlessly (no
// real controlling terminal, which most CI/sandboxed test environments
// don't have) for the duration of the test.
func withHeadlessNewProgram(t *testing.T) {
	t.Helper()
	old := newProgram
	newProgram = func(m tea.Model, opts ...tea.ProgramOption) *tea.Program {
		opts = append(opts, tea.WithInput(nil), tea.WithoutRenderer())
		return old(m, opts...)
	}
	t.Cleanup(func() { newProgram = old })
}

func TestRunnerRunAllSucceed(t *testing.T) {
	withHeadlessNewProgram(t)
	tr := New("root")
	tr.Add("task", func(ex *Executor) (res Result) { return Success("ok") })

	err := tr.Run()
	testutils.NoError(t, err)
}

func TestRunnerRunReportsFailures(t *testing.T) {
	withHeadlessNewProgram(t)
	tr := New("root")
	tr.Add("task", func(ex *Executor) (res Result) { return Failure("boom") })

	err := tr.Run()
	testutils.Error(t, err)
	testutils.Assert(t, errors.Is(err, ErrCompletedWithFailures), "expected ErrCompletedWithFailures")
}

// quitsImmediatelyModel is a tea.Model unrelated to this package's model
// type, used to exercise Run()'s "unexpected model type" defensive branch:
// tr.program.Run() legitimately returns whatever model the *tea.Program
// was actually constructed with, so substituting a different one via
// newProgram is enough to trigger it without needing production changes.
type quitsImmediatelyModel struct{}

func (quitsImmediatelyModel) Init() tea.Cmd                         { return tea.Quit }
func (m quitsImmediatelyModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, tea.Quit }
func (quitsImmediatelyModel) View() string                          { return "" }

func TestRunnerRunUnexpectedModelType(t *testing.T) {
	old := newProgram
	newProgram = func(m tea.Model, opts ...tea.ProgramOption) *tea.Program {
		opts = append(opts, tea.WithInput(nil), tea.WithoutRenderer())
		return old(quitsImmediatelyModel{}, opts...)
	}
	t.Cleanup(func() { newProgram = old })

	tr := New("root")
	tr.Add("task", func(ex *Executor) (res Result) { return Success("ok") })

	err := tr.Run()
	testutils.Error(t, err)
	testutils.Assert(t, errors.Is(err, Error), "expected the base taskrunner Error to be wrapped")
}

// brokenReader always fails, used to make the real *tea.Program's Run()
// return a genuine (non-model-related) error deterministically.
type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errors.New("broken read") }

func TestRunnerRunProgramError(t *testing.T) {
	old := newProgram
	newProgram = func(m tea.Model, opts ...tea.ProgramOption) *tea.Program {
		opts = append(opts, tea.WithInput(brokenReader{}), tea.WithoutRenderer())
		return old(m, opts...)
	}
	t.Cleanup(func() { newProgram = old })

	tr := New("root")
	tr.Add("task", func(ex *Executor) (res Result) { return Success("ok") })

	err := tr.Run()
	testutils.Error(t, err)
	testutils.Assert(t, !errors.Is(err, Error), "expected a raw bubbletea error, not one of taskrunner's own")
}

func TestRunnerRunPipeError(t *testing.T) {
	old := newPipe
	wantErr := errors.New("pipe failure")
	newPipe = func() (*os.File, *os.File, error) { return nil, nil, wantErr }
	t.Cleanup(func() { newPipe = old })

	tr := New("root")
	err := tr.Run()
	testutils.Assert(t, errors.Is(err, wantErr), "expected the pipe error to be returned as-is")
}

func TestRunnerRunLogsPipeCloseError(t *testing.T) {
	withHeadlessNewProgram(t)

	old := newPipe
	newPipe = func() (*os.File, *os.File, error) {
		pr, pw, err := old()
		if err != nil {
			return pr, pw, err
		}
		// Close both ends up front so Run()'s own deferred Close() calls
		// hit "file already closed", exercising both error-logging branches.
		_ = pw.Close()
		_ = pr.Close()
		return pr, pw, nil
	}
	t.Cleanup(func() { newPipe = old })

	tr := New("root")
	tr.Add("task", func(ex *Executor) (res Result) { return Success("ok") })

	// Only asserting this doesn't panic/hang: the double-close is logged via
	// slog, not returned as an error, so Run()'s own return value here isn't
	// the point of this test.
	_ = tr.Run()
}
