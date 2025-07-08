// SPDX-License-Identifier: Apache-2.0
//
// Copyright (C) 2025 The Happy SDK Authors
// See the LICENSE file for full licensing details.

package taskrunner

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/happy-sdk/happy/pkg/branding"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	Error = errors.New("taskrunner")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#307a58")).
			Padding(0, 1).
			Align(lipgloss.Center).
			Width(80)

	currentTaskNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	currentRunnerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

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

	groupTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#164730")).
			Padding(0, 1).
			Width(80)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Width(80)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Width(48)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Width(60)
)

func normalize(in string) (out string) {
	return strings.ReplaceAll(in, "\n", "")
}

// Info utility function to create Result with state INFO
func Info(status, desc string) Result {
	return Result{
		Status:     normalize(status),
		State:      INFO,
		Decription: normalize(desc),
	}
}

// Notice utility function to create Result with state INFO
func Notice(status, desc string) Result {
	return Result{
		Status:     normalize(status),
		State:      NOTICE,
		Decription: normalize(desc),
	}
}

// Success utility function to create Result with state SUCCESS
func Success(status, desc string) Result {
	return Result{
		Status:     normalize(status),
		State:      SUCCESS,
		Decription: normalize(desc),
	}
}

// Warn utility function to create Result with state WARNING
func Warn(status, desc string) Result {
	return Result{
		Status:     normalize(status),
		State:      WARNING,
		Decription: normalize(desc),
	}
}

// Failure utility function to create Result with state FAILURE
func Failure(status, desc string) Result {
	return Result{
		Status:     normalize(status),
		State:      FAILURE,
		Decription: normalize(desc),
	}
}

type State uint

const (
	SKIPPED State = iota
	INFO
	NOTICE
	SUCCESS
	WARNING
	FAILURE
)

type Result struct {
	uuid       string
	task       string
	Status     string
	State      State
	Decription string
}

func New(title string) *Runner {
	return &Runner{
		title: title,
	}
}

type Runner struct {
	mu     sync.Mutex
	title  string
	groups []*Group
}

func (r *Runner) WithColors(colors branding.ColorPalette) *Runner {
	r.mu.Lock()
	defer r.mu.Unlock()

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Success))
	successMark = successStyle.SetString("✓")
	failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Danger))
	failMark = failStyle.SetString("✗")
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Warning))
	warnMark = warnStyle.SetString("⚠")
	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Info))
	infoMark = infoStyle.SetString("☉")

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color(colors.Primary)).
		Padding(0, 1).
		Align(lipgloss.Center).
		Width(80)

	groupTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color(colors.Secondary)).
		Padding(0, 1).
		Width(80)
	return r
}

func (r *Runner) Add(group *Group) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, g := range r.groups {
		if group.id.String() == g.id.String() {
			return fmt.Errorf("%w: group %s already registered", Error, group.title)
		}
	}
	r.groups = append(r.groups, group)
	return nil
}

// Run executes all tasks based on what pattern <group>.<task>
func (r *Runner) Run(what string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Println(titleStyle.Render(cases.Upper(language.English).String(r.title)))

	if len(r.groups) == 0 || len(r.groups[0].tasks) == 0 {
		return errors.New("no groups with tasks added")
	}
	p := progress.New(
		progress.WithScaledGradient("#32595e", "#239940"),
		progress.WithWidth(90),
	)
	s := spinner.New(
		spinner.WithSpinner(spinner.Pulse),
	)
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

	model := &tui{
		spinner:    s,
		progress:   p,
		failedDeps: make(map[string]bool),
	}

	if what == "*" {
		model.taskGroups = r.groups
	} else {
		return fmt.Errorf("%w: %s not supported what argument", Error, what)
	}

	model.executingTaskName = model.taskGroups[0].tasks[0].name
	for _, g := range model.taskGroups {
		model.totalTasks += len(g.tasks)
	}

	if _, err := tea.NewProgram(*model).Run(); err != nil {
		return err
	}
	return nil
}

type Task struct {
	uuid      uuid.UUID
	name      string
	action    Action
	dependsOn string
}

type Action func() (res Result)

type Group struct {
	id       uuid.UUID
	title    string
	tasks    []*Task
	executed int
}

func NewGroup(title string) *Group {
	return &Group{
		id:    uuid.New(),
		title: title,
	}
}

func (g *Group) task(name string, action Action) *Task {
	taskUUID := uuid.New()
	task := &Task{
		uuid: taskUUID,
		name: name,
		action: func() (res Result) {
			time.Sleep(time.Millisecond * 20)
			return action()
		},
	}
	g.tasks = append(g.tasks, task)
	return task
}

func (g *Group) Task(name string, action Action) uuid.UUID {
	task := g.task(name, action)
	return task.uuid
}

// TaskD adds task which depends on another tasks to be succsessful
func (g *Group) TaskD(dependency uuid.UUID, name string, action Action) uuid.UUID {
	task := g.task(name, action)
	task.dependsOn = dependency.String()
	return task.uuid
}

func (g *Group) getNextTask() *Task {
	if g.executed >= len(g.tasks) {
		return nil
	}
	next := g.tasks[g.executed]
	g.executed++
	return next
}

func (g *Group) justStarted() bool {
	return g.executed == 0 && len(g.tasks) > 0
}
