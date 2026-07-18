// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func newTestModel() model {
	return model{
		spinner:            spinner.New(),
		progress:           progress.New(),
		totalTasks:         2,
		progressTotalSteps: 200,
	}
}

func TestModelInit(t *testing.T) {
	m := newTestModel()
	cmd := m.Init()
	testutils.Assert(t, cmd != nil, "expected Init to return a non-nil command")
}

func TestModelUpdateSpinnerTick(t *testing.T) {
	m := newTestModel()
	msg := m.spinner.Tick()
	newM, cmd := m.Update(msg)
	nm, ok := newM.(model)
	testutils.Assert(t, ok, "expected model type back")
	_ = nm
	testutils.Assert(t, cmd != nil, "expected a follow-up tick command")
}

func TestModelUpdateProgressFrame(t *testing.T) {
	m := newTestModel()
	// Drive a real FrameMsg through progress.SetPercent so Update sees a
	// genuine progress.FrameMsg rather than a hand-built one.
	cmd := m.progress.SetPercent(0.5)
	testutils.Assert(t, cmd != nil, "expected SetPercent to return a command")
	msg := cmd()
	newM, _ := m.Update(msg)
	_, ok := newM.(model)
	testutils.Assert(t, ok, "expected model type back")
}

func TestModelUpdateKeyMsgQuits(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"q", tea.KeyPressMsg{Text: "q", Code: 'q'}},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel()
			_, cmd := m.Update(tt.key)
			testutils.Assert(t, cmd != nil, "expected a quit command for key %q", tt.name)
			msg := cmd()
			_, isQuit := msg.(tea.QuitMsg)
			testutils.Assert(t, isQuit, "expected tea.QuitMsg for key %q, got %T", tt.name, msg)
		})
	}
}

func TestModelUpdateOtherKeyDoesNotQuit(t *testing.T) {
	m := newTestModel()
	newM, cmd := m.Update(tea.KeyPressMsg{Text: "x", Code: 'x'})
	nm := newM.(model)
	testutils.Equal(t, m.finished, nm.finished, "unrelated key must not change finished")
	if cmd != nil {
		msg := cmd()
		_, isQuit := msg.(tea.QuitMsg)
		testutils.Assert(t, !isQuit, "unrelated key must not quit")
	}
}

func TestModelUpdateSetStatusMsg(t *testing.T) {
	m := newTestModel()
	newM, _ := m.Update(SetStatusMsg("building"))
	nm := newM.(model)
	testutils.Equal(t, "building", nm.statusMessage, "unexpected status message")
	testutils.Assert(t, nm.running, "expected running to be true")
}

func TestModelUpdateAddTickMsg(t *testing.T) {
	m := newTestModel()
	m.progressCompletedSteps = 5
	before := m.progressTotalSteps
	newM, _ := m.Update(addTickMsg{})
	nm := newM.(model)
	testutils.Equal(t, 6.0, nm.progressCompletedSteps, "expected completed steps to increment")
	testutils.Equal(t, before+1, nm.progressTotalSteps, "expected total steps to increment")
}

func TestModelUpdateAddSubTaskMsg(t *testing.T) {
	m := newTestModel()
	m.progressTotalSteps = 100
	m.progressCompletedSteps = 50
	before := m.totalTasks
	newM, cmd := m.Update(addSubTaskMsg{})
	nm := newM.(model)
	testutils.Equal(t, before+1, nm.totalTasks, "expected totalTasks to increment")
	testutils.Assert(t, cmd != nil, "expected a progress update command")
}

func TestModelUpdateSubTaskProgressStepsMsg(t *testing.T) {
	m := newTestModel()
	before := m.progressTotalSteps
	newM, _ := m.Update(subTaskProgressStepsMsg{steps: 25})
	nm := newM.(model)
	testutils.Equal(t, before+25, nm.progressTotalSteps, "expected total steps to grow by the given amount")
}

func TestModelUpdateOutputMsg(t *testing.T) {
	m := newTestModel()
	m.progressTotalSteps = 100
	m.progressCompletedSteps = 10
	newM, cmd := m.Update(OutputMsg("hello"))
	nm := newM.(model)
	testutils.Equal(t, 11.0, nm.progressCompletedSteps, "expected completed steps to increment")
	testutils.Assert(t, cmd != nil, "expected commands to be returned")
}

func TestModelUpdateAllTasksCompleteMsg(t *testing.T) {
	m := newTestModel()
	m.progressTotalSteps = 100
	m.progressCompletedSteps = 100
	newM, cmd := m.Update(allTasksCompleteMsg{ExecDur: time.Second})
	nm := newM.(model)
	testutils.Assert(t, !nm.finished, "allTasksCompleteMsg itself must not mark finished yet")
	testutils.Assert(t, cmd != nil, "expected a sequence command")

	// Executing the returned command chain should eventually deliver a
	// finalExitMsg (via tea.Tick), same as the real runtime would.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if ok {
		var found bool
		for _, c := range batch {
			if c == nil {
				continue
			}
			if _, isFinal := c().(finalExitMsg); isFinal {
				found = true
			}
		}
		testutils.Assert(t, found, "expected a finalExitMsg to eventually be produced")
	}
}

func TestModelUpdateFinalExitMsg(t *testing.T) {
	m := newTestModel()
	m.successes = 1
	newM, cmd := m.Update(finalExitMsg{ExecDur: time.Second})
	nm := newM.(model)
	testutils.Assert(t, nm.finished, "expected finished to be true")
	testutils.Assert(t, cmd != nil, "expected commands (println + quit)")
}

func TestModelUpdateResultSuccess(t *testing.T) {
	m := newTestModel()
	m.progressTotalSteps = 200
	newM, _ := m.Update(Result{name: "t1", msg: "ok", state: SUCCESS})
	nm := newM.(model)
	testutils.Equal(t, 1, nm.executedTasks, "expected executedTasks to increment")
	testutils.Equal(t, 1, nm.successes, "expected successes to increment")
	testutils.Equal(t, progressTaskSteps, nm.progressCompletedSteps, "non-subtask result should add a full task step")
	testutils.Assert(t, !nm.running, "expected running to be cleared for a non-subtask result")
}

func TestModelUpdateResultSubtask(t *testing.T) {
	m := newTestModel()
	m.progressTotalSteps = 200
	m.running = true
	newM, _ := m.Update(Result{name: "sub", msg: "ok", state: SUCCESS, isSubtask: true, subtaskProgressTaskSteps: 33})
	nm := newM.(model)
	testutils.Equal(t, 33.0, nm.progressCompletedSteps, "subtask result should add its own step size")
	testutils.Assert(t, nm.running, "a subtask result must not clear running")
}

func TestModelUpdateResultStateCounts(t *testing.T) {
	tests := []struct {
		state State
		check func(t *testing.T, m model)
	}{
		{SUCCESS, func(t *testing.T, m model) { testutils.Equal(t, 1, m.successes, "successes") }},
		{NOTICE, func(t *testing.T, m model) { testutils.Equal(t, 1, m.notices, "notices") }},
		{WARNING, func(t *testing.T, m model) { testutils.Equal(t, 1, m.warnings, "warnings") }},
		{FAILURE, func(t *testing.T, m model) { testutils.Equal(t, 1, m.failures, "failures") }},
		{SKIPPED, func(t *testing.T, m model) { testutils.Equal(t, 1, m.skipped, "skipped") }},
		{INFO, func(t *testing.T, m model) {
			testutils.Equal(t, 0, m.successes+m.notices+m.warnings+m.failures+m.skipped, "INFO should not bump any counter")
		}},
	}
	for _, tt := range tests {
		m := newTestModel()
		m.progressTotalSteps = 200
		newM, _ := m.Update(Result{name: "t", msg: "m", state: tt.state})
		tt.check(t, newM.(model))
	}
}

func TestModelViewFinishedIsEmpty(t *testing.T) {
	m := newTestModel()
	m.finished = true
	testutils.Equal(t, "", m.View().Content, "expected empty view once finished")
}

func TestModelViewNotFinished(t *testing.T) {
	m := newTestModel()
	m.statusMessage = "working"
	testutils.Assert(t, m.View().Content != "", "expected a non-empty status view")
}

func TestGetFinalRaport(t *testing.T) {
	tests := []struct {
		name  string
		m     model
		parts []string
	}{
		{
			name:  "all zero",
			m:     model{totalTasks: 3},
			parts: []string{"total (3)"},
		},
		{
			name:  "only successes",
			m:     model{totalTasks: 2, successes: 2},
			parts: []string{"OK", "= 2", "total (2)"},
		},
		{
			name:  "failures and warnings",
			m:     model{totalTasks: 4, failures: 1, warnings: 2},
			parts: []string{"FAILURES", "= 1", "= 2"},
		},
		{
			name:  "only warnings",
			m:     model{totalTasks: 2, warnings: 2},
			parts: []string{"WARN", "= 2", "total (2)"},
		},
		{
			name:  "only notices",
			m:     model{totalTasks: 2, notices: 3},
			parts: []string{"NOTICES", "= 3", "total (2)"},
		},
		{
			name:  "failures warnings notices successes skipped all present",
			m:     model{totalTasks: 5, failures: 1, warnings: 1, notices: 1, successes: 1, skipped: 1},
			parts: []string{"FAILURES", "= 1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := tt.m.getFinalRaport(time.Second)
			for _, part := range tt.parts {
				testutils.Assert(t, strings.Contains(report, part), "expected report to contain %q, got %q", part, report)
			}
		})
	}
}

func TestGetStatusMessage(t *testing.T) {
	m := newTestModel()
	m.executedTasks = 1
	m.totalTasks = 4
	m.progressTotalSteps = 100
	m.progressCompletedSteps = 25
	m.statusMessage = "building"

	msg := m.getStatusMessage()
	testutils.Assert(t, strings.Contains(msg, "1/4"), "expected task counter in status message, got %q", msg)
}
