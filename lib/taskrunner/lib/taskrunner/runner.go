// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"bufio"
	"context"
	"os"
	"slices"
	"strings"
	"sync"

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
	captureMu     sync.Mutex
	failedTasks   []TaskID
	ctx           context.Context
}

func New() *Runner {
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

	return &Runner{
		tasks:         make([]Task, 0),
		capturedLines: make([]string, 0),
		model: &model{
			spinner:  s,
			progress: p,
		},
	}
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

	original := os.Stdout

	// Start bubbletea program
	tr.model.progressTotalSteps = float64(tr.model.totalTasks) * progressTaskSteps

	tr.program = tea.NewProgram(tr.model, tea.WithFPS(120))

	// Start output capture goroutine
	os.Stdout = pw
	defer func() {
		// Restore stdout and clean up
		os.Stdout = original
		pw.Close()
		pr.Close()
	}()

	go tr.captureOutput(pr)

	// Start task execution goroutine
	go tr.executeTasks(original)

	// Run the program
	_, err = tr.program.Run()

	return err
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
			output := strings.TrimSpace(scanner.Text())
			if output != "" {
				for line := range strings.SplitSeq(output, "\n") {
					if strings.TrimSpace(line) != "" {
						tr.program.Send(OutputMsg(strings.TrimSpace(line)))
					}
				}
			}
		}
	}
}

func (tr *Runner) executeTasks(out *os.File) {
	for _, task := range tr.tasks {
		e := newExecutor(task.name, tr.program, out)
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

		res := task.action(e)
		res.id = task.id
		res.name = task.name
		if res.state == FAILURE || res.state == SKIPPED {
			tr.failedTasks = append(tr.failedTasks, task.id)
			tr.program.Send(res)
			continue
		}

		sstate, failedTasks := e.runSubtasks(tr.failedTasks)
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
	tr.program.Send(allTasksCompleteMsg{})
}
