// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"os"
	"slices"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

// runHeadless runs a real (but headless - no real terminal/input needed)
// bubbletea program against m and lets drive interact with it via p while it
// runs concurrently, then quits it and returns the final model. *tea.Program
// only actually drains its message channel once Run() is executing, so
// every test that exercises an Executor/Runner method that calls
// program.Send(...) must run its program this way - calling Send on a
// program that was never Run() blocks forever. This also exercises
// Executor/Runner's message-sending logic against the real bubbletea
// message loop, instead of needing a fake in place of *tea.Program (which
// the exported Executor.Program() method must keep returning as-is, since
// real callers use *tea.Program-specific methods like
// ReleaseTerminal/RestoreTerminal on it).
func runHeadless(t *testing.T, m tea.Model, drive func(p *tea.Program)) model {
	t.Helper()
	p := tea.NewProgram(m, tea.WithInput(nil), tea.WithoutRenderer())
	done := make(chan tea.Model, 1)
	go func() {
		fm, err := p.Run()
		testutils.NoError(t, err)
		done <- fm
	}()
	drive(p)
	p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	fm := <-done
	return fm.(model)
}

func newTestProgressModel() model {
	return model{spinner: spinner.New(), progress: progress.New()}
}

func TestNewExecutorAccessors(t *testing.T) {
	runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task-a", p, os.Stdout)
		testutils.Equal(t, p, e.Program(), "Program() should return the exact *tea.Program given to newExecutor")
		testutils.Equal(t, os.Stdout, e.Stdout(), "Stdout() should return the exact file given to newExecutor")
	})
}

func TestExecutorAddTick(t *testing.T) {
	final := runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task", p, os.Stdout)
		e.AddTick()
	})
	testutils.Assert(t, final.progressCompletedSteps >= 1, "expected AddTick to bump progressCompletedSteps")
}

func TestExecutorSubtaskAndSubtaskD(t *testing.T) {
	var subtaskCount int
	var dep1, id1, id2 uuid.UUID

	runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task", p, os.Stdout)

		id1 = e.Subtask("sub1", func(ex *Executor) (res Result) { return Success("ok") })
		id2 = e.SubtaskD(id1, "sub2", func(ex *Executor) (res Result) { return Success("ok") })

		subtaskCount = len(e.subtasks)
		dep1 = e.subtasks[1].dependsOn
	})

	testutils.NotEqual(t, uuid.Nil, id1, "expected a non-nil subtask id")
	testutils.NotEqual(t, uuid.Nil, id2, "expected a non-nil subtask id")
	testutils.Equal(t, 2, subtaskCount, "expected two queued subtasks")
	testutils.Equal(t, id1, dep1, "SubtaskD should record the dependency")
}

func TestExecutorRunSubtasksEmpty(t *testing.T) {
	var state State
	var failedLen int
	var sealed bool

	runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task", p, os.Stdout)
		var failed []TaskID
		state, failed = e.runSubtasks(nil)
		failedLen = len(failed)
		sealed = e.sealed
	})

	testutils.Equal(t, SUCCESS, state, "expected SUCCESS for no subtasks")
	testutils.Equal(t, 0, failedLen, "expected no failed tasks")
	testutils.Assert(t, sealed, "expected executor to be sealed after runSubtasks")
}

func TestExecutorRunSubtasksAllSucceed(t *testing.T) {
	var state State
	var failedLen, remainingSubtasks int

	runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task", p, os.Stdout)
		e.Subtask("sub1", func(ex *Executor) (res Result) { return Success("ok") })
		e.Subtask("sub2", func(ex *Executor) (res Result) { return Success("ok") })

		var failed []TaskID
		state, failed = e.runSubtasks(nil)
		failedLen = len(failed)
		remainingSubtasks = len(e.subtasks)
	})

	testutils.Equal(t, SUCCESS, state, "expected overall SUCCESS")
	testutils.Equal(t, 0, failedLen, "expected no failed tasks")
	testutils.Equal(t, 0, remainingSubtasks, "expected subtasks slice to be reset")
}

func TestExecutorRunSubtasksAggregatesWorstState(t *testing.T) {
	var state State
	var failed []TaskID
	var failID uuid.UUID

	runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task", p, os.Stdout)
		e.Subtask("sub1", func(ex *Executor) (res Result) { return Success("ok") })
		failID = e.Subtask("sub2", func(ex *Executor) (res Result) { return Failure("boom") })
		e.Subtask("sub3", func(ex *Executor) (res Result) { return Warn("careful") })

		state, failed = e.runSubtasks(nil)
	})

	testutils.Equal(t, FAILURE, state, "expected worst-state (FAILURE) to win")
	testutils.Equal(t, 1, len(failed), "expected exactly one failed task id")
	if len(failed) == 1 {
		testutils.Equal(t, failID, failed[0], "expected the failing subtask's id to be recorded")
	}
}

// TestExecutorRunSubtasksSkipsOnUnmetDependencyFromCaller documents a subtle
// behavior: a subtask skipped because its dependency was already in
// trFailedTasks (the caller-supplied list) hits a `continue` before the
// `if res.state > state` aggregation runs, so it's recorded in the returned
// failedTasks slice but does NOT elevate the returned state - unlike a
// subtask that runs its own action and returns FAILURE/SKIPPED, which does.
// If every subtask in a run is skipped this way, runSubtasks reports
// SUCCESS even though nothing actually ran. Not something this test suite
// changes - just pinning down what the current code actually does.
func TestExecutorRunSubtasksSkipsOnUnmetDependencyFromCaller(t *testing.T) {
	var state State
	var failed []TaskID
	var ranAction bool
	upstreamFailedID := uuid.New()

	runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task", p, os.Stdout)
		e.SubtaskD(upstreamFailedID, "dependent", func(ex *Executor) (res Result) {
			ranAction = true
			return Success("should not run")
		})
		state, failed = e.runSubtasks([]TaskID{upstreamFailedID})
	})

	testutils.Equal(t, SUCCESS, state, "a dependency-skip alone does not elevate the returned state (see comment above)")
	testutils.Assert(t, !ranAction, "expected the dependent subtask's action never to run")
	testutils.Equal(t, 1, len(failed), "expected the skipped subtask to be recorded as failed")
}

func TestExecutorRunSubtasksSkipsOnDependencyFailedWithinSameRun(t *testing.T) {
	var state State
	var failed []TaskID
	var ranAction bool
	var failID uuid.UUID

	runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task", p, os.Stdout)
		failID = e.Subtask("upstream", func(ex *Executor) (res Result) { return Failure("boom") })
		e.SubtaskD(failID, "downstream", func(ex *Executor) (res Result) {
			ranAction = true
			return Success("should not run")
		})

		state, failed = e.runSubtasks(nil)
	})

	testutils.Equal(t, FAILURE, state, "expected overall FAILURE")
	testutils.Assert(t, !ranAction, "expected the downstream subtask's action never to run")
	testutils.Assert(t, slices.Contains(failed, failID), "expected the upstream failure id to be recorded")
	testutils.Equal(t, 2, len(failed), "expected both the upstream failure and the skipped downstream task recorded")
}

func TestExecutorRunSubtasksTicksDuringSlowAction(t *testing.T) {
	final := runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task", p, os.Stdout)
		e.Subtask("slow", func(ex *Executor) (res Result) {
			// Long enough for runSubtasks' 1/30s ticker to fire at least
			// once before the action returns and cancels it.
			time.Sleep(60 * time.Millisecond)
			return Success("ok")
		})
		e.runSubtasks(nil)
	})
	testutils.Assert(t, final.progressCompletedSteps > 0, "expected at least one tick to have been processed")
}

func TestExecutorPrintln(t *testing.T) {
	final := runHeadless(t, newTestProgressModel(), func(p *tea.Program) {
		e := newExecutor("task", p, os.Stdout)
		e.Println("hello", "world")
	})
	// Println's OutputMsg is consumed by model.Update, which folds it into
	// progress steps; there's no direct text capture on model, so just
	// confirm the message was processed (progress accounting moved).
	testutils.Assert(t, final.progressCompletedSteps >= 1, "expected Println's OutputMsg to be processed")
}
