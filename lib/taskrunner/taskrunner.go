// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

var (
	Error                    = errors.New("taskrunner")
	ErrCompletedWithFailures = fmt.Errorf("%w: completed with failures", Error)
)

type SetStatusMsg string
type OutputMsg string

type addTickMsg struct {
}

type allTasksCompleteMsg struct {
	ExecDur time.Duration
}

type finalExitMsg struct {
	ExecDur time.Duration
}

type addSubTaskMsg struct {
}

type subTaskProgressStepsMsg struct {
	steps float64
}

var (
	doneStyle = lipgloss.NewStyle().Margin(1, 2)

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("47"))
	successMark  = successStyle.SetString("✓")
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	failMark     = failStyle.SetString("✗")
	noticeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	noticeMark   = noticeStyle.SetString("⚠")
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	warnMark     = warnStyle.SetString("⚠")
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	infoMark     = infoStyle.SetString("☉")
	skipMark     = lipgloss.NewStyle().Foreground(lipgloss.Color("144")).SetString("⚐")

	statusMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ddc759"))

	nameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA"))

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))
)

type State uint

const (
	SKIPPED State = iota
	SUCCESS
	INFO
	NOTICE
	WARNING
	FAILURE
)

type Result struct {
	id                       uuid.UUID
	name                     string
	msg                      string
	dur                      time.Duration
	state                    State
	decription               string
	isSubtask                bool
	subtaskProgressTaskSteps float64
}

func (r Result) WithDesc(desc string) Result {
	r.decription = normalize(desc)
	return r
}

// Success utility function to create Result with state SUCCESS
func Success(msg string) Result {
	return Result{
		msg:   normalize(msg),
		state: SUCCESS,
	}
}

// Info utility function to create Result with state INFO
func Info(msg string) Result {
	return Result{
		msg:   normalize(msg),
		state: INFO,
	}
}

// Notice utility function to create Result with state INFO
func Notice(msg string) Result {
	return Result{
		msg:   normalize(msg),
		state: NOTICE,
	}
}

// Warn utility function to create Result with state WARNING
func Warn(msg string) Result {
	return Result{
		msg:   normalize(msg),
		state: WARNING,
	}
}

// Failure utility function to create Result with state FAILURE
func Failure(msg string) Result {
	return Result{
		msg:   normalize(msg),
		state: FAILURE,
	}
}

// Skip utility function to create Result with state FAILURE
func Skip(msg string) Result {
	return Result{
		msg:   normalize(msg),
		state: SKIPPED,
	}
}

func normalize(in string) (out string) {
	return strings.ReplaceAll(in, "\n", "")
}
