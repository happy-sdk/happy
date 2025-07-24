// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

const (
	progressTaskSteps float64 = 100.0
)

type Runner struct {
	tasks         []Task
	mu            sync.Mutex
	model         *model
	program       *tea.Program
	capturedLines []string
	failedTasks   []TaskID
	ctx           context.Context
	dur           time.Duration
}

func New(name string) *Runner {
	if name == "" {
		name = "taskrunner"
	}
	// Initialize model
	s := spinner.New(
		spinner.WithSpinner(spinner.Points),
	)
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffed56"))
	p := progress.New(
		progress.WithScaledGradient("#ff9800", "#4caf50"),
		progress.WithWidth(50),
		progress.WithFillCharacters('▰', '▱'),
		progress.WithSpringOptions(120.0, 1.0),
		progress.WithoutPercentage(),
	)

	tr := &Runner{
		tasks:         make([]Task, 0),
		capturedLines: make([]string, 0),
		model: &model{
			spinner:  s,
			progress: p,
		},
	}

	tr.Add(name, func(ex *Executor) (res Result) {
		return Success(fmt.Sprintf("running %d root tasks", len(tr.tasks)))
	})

	return tr
}

// AddTask adds a task to runner.
func (tr *Runner) AddTask(task Task) TaskID {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.tasks = append(tr.tasks, task)
	tr.model.totalTasks++
	if len(task.name) > tr.model.longestTaskNameLength {
		tr.model.longestTaskNameLength = len(task.name)
	}
	return task.id
}

// Add adds a task without any dependencies.
func (tr *Runner) Add(name string, action Action) TaskID {
	task := NewTask(name, action)
	tr.AddTask(task)
	return task.id
}

// AddD adds a task with a dependency on other task.
func (tr *Runner) AddD(dep TaskID, name string, action Action) TaskID {
	task := NewTask(name, action).DependsOn(dep)
	tr.AddTask(task)
	return task.id
}

func (tr *Runner) Run() error {
	ctx, done := context.WithCancel(context.Background())
	tr.ctx = ctx
	defer done()

	// Create pipe to capture stdout
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}

	originalStdout := os.Stdout

	// Start bubbletea program
	tr.model.progressTotalSteps = float64(tr.model.totalTasks) * progressTaskSteps

	tr.program = tea.NewProgram(tr.model, tea.WithFPS(120))

	// Start output capture goroutine
	os.Stdout = pw
	defer func() {
		// Restore stdout and clean up
		os.Stdout = originalStdout
		if err := pw.Close(); err != nil {
			slog.Error("error closing pipe writer", slog.String("err", err.Error()))
		}
		if err := pr.Close(); err != nil {
			slog.Error("error closing pipe reader", slog.String("err", err.Error()))
		}
	}()

	go tr.captureOutput(pr)

	// Start task execution goroutine
	go tr.executeTasks(originalStdout)

	// Run the program
	fm, err := tr.program.Run()

	if err != nil {
		return err
	}

	fmTyped, ok := fm.(model)
	if !ok {
		return fmt.Errorf("%w: unexpected model type", Error)
	}

	if fmTyped.failures > 0 {
		return fmt.Errorf("%w: failed tasks: %d/%d", ErrCompletedWithFailures, fmTyped.failures, fmTyped.totalTasks)
	}
	return nil
}

func (tr *Runner) captureOutput(pr *os.File) {
	scanner := bufio.NewScanner(pr)
	for {
		select {
		case <-tr.ctx.Done():
			return
		default:
			if !scanner.Scan() {
				return
			}
			output := strings.TrimRight(scanner.Text(), " ")
			if output != "" {
				for line := range strings.SplitSeq(output, "\n") {
					line := strings.TrimRight(line, " ")
					if line != "" {
						tr.program.Send(OutputMsg("\r" + line))
					}
				}
			}
		}
	}
}

func (tr *Runner) executeTasks(originalStdout *os.File) {
	for _, task := range tr.tasks {
		e := newExecutor(task.name, tr.program, originalStdout)
		if task.id == uuid.Nil {
			res := Failure("invalid task id")
			res.name = task.name
			tr.program.Send(res)
			continue
		}
		tr.program.Send(SetStatusMsg(task.name))
		if task.dependsOn != uuid.Nil && slices.Contains(tr.failedTasks, task.dependsOn) {
			res := Skip("skip").WithDesc("dependency not satisified")
			res.id = task.id
			res.name = task.name

			tr.failedTasks = append(tr.failedTasks, task.id)
			tr.program.Send(res)
			continue
		}

		e.program.Send(addTickMsg{})
		start := time.Now()

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			ticker := time.NewTicker(time.Second / 5)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					e.program.Send(addTickMsg{})
				}
			}
		}()

		res := task.action(e)
		cancel()
		tr.program.Send(addTickMsg{})

		res.dur = time.Since(start)
		tr.dur += res.dur

		res.id = task.id
		res.name = task.name
		if res.state == FAILURE || res.state == SKIPPED {
			tr.failedTasks = append(tr.failedTasks, task.id)
			tr.program.Send(res)
			continue
		}

		sstate, failedTasks := e.runSubtasks(tr.failedTasks)
		tr.dur += e.dur

		res.dur += e.dur
		tr.failedTasks = append(tr.failedTasks, failedTasks...)
		if sstate == FAILURE || sstate == SKIPPED {
			tr.failedTasks = append(tr.failedTasks, task.id)
		}
		if sstate != SUCCESS {
			res.state = sstate
		}

		tr.program.Send(res)
	}

	// Send all tasks complete message
	tr.program.Send(allTasksCompleteMsg{
		ExecDur: tr.dur,
	})
}
