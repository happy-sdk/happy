// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

type Executor struct {
	mu       sync.RWMutex
	program  *tea.Program
	stdout   *os.File
	subtasks []Task
	taskName string
	sealed   bool
	dur      time.Duration
}

func newExecutor(taskName string, program *tea.Program, stdout *os.File) *Executor {
	return &Executor{
		program:  program,
		stdout:   stdout,
		taskName: taskName,
	}
}

func (e *Executor) Program() *tea.Program {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.program
}

func (e *Executor) Stdout() *os.File {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stdout
}

// AddTick adds a tick to the executor which is affecting the executor's progress bar.
func (e *Executor) AddTick() {
	e.mu.RLock()
	defer e.mu.RUnlock()
	e.program.Send(addTickMsg{})
}

func (e *Executor) Subtask(name string, action Action) TaskID {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.program.Send(addSubTaskMsg{})
	task := NewTask(name, action)
	e.subtasks = append(e.subtasks, task)
	return task.id
}

func (e *Executor) SubtaskD(dep TaskID, name string, action Action) TaskID {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.program.Send(addSubTaskMsg{})

	task := NewTask(name, action).DependsOn(dep)
	e.subtasks = append(e.subtasks, task)
	return task.id
}

func (e *Executor) runSubtasks(trFailedTasks []TaskID) (state State, failedTasks []TaskID) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.sealed = true

	defer func() {
		e.program.Send(SetStatusMsg(e.taskName))
	}()

	if len(e.subtasks) == 0 {
		return SUCCESS, nil
	}

	state = SUCCESS

	subtaskProgressTaskSteps := progressTaskSteps / float64(len(e.subtasks))
	for _, task := range e.subtasks {
		ex := newExecutor(task.name, e.program, e.stdout)
		e.program.Send(SetStatusMsg(task.name))
		e.program.Send(subTaskProgressStepsMsg{steps: subtaskProgressTaskSteps})

		if task.dependsOn != uuid.Nil &&
			(slices.Contains(trFailedTasks, task.dependsOn) ||
				slices.Contains(failedTasks, task.dependsOn)) {
			res := Skip("skip").WithDesc("dependency not satisified")
			res.id = task.id
			res.name = task.name
			res.isSubtask = true
			res.subtaskProgressTaskSteps = subtaskProgressTaskSteps
			failedTasks = append(failedTasks, task.id)
			e.program.Send(res)
			continue
		}

		start := time.Now()

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			ticker := time.NewTicker(time.Second / 30)
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

		res := task.action(ex)
		cancel()
		e.program.Send(addTickMsg{})

		res.dur = time.Since(start)
		e.dur += res.dur

		res.name = task.name
		res.isSubtask = true
		res.subtaskProgressTaskSteps = subtaskProgressTaskSteps

		e.program.Send(res)

		if res.state > state {
			state = res.state
		}
		if res.state == FAILURE || res.state == SKIPPED {
			failedTasks = append(failedTasks, task.id)
		}
	}
	e.subtasks = nil

	return state, failedTasks
}

func (e *Executor) Println(a ...any) {
	builder := bytes.Buffer{}
	_, _ = fmt.Fprintln(&builder, a...)
	e.program.Send(OutputMsg(builder.String()))
}
