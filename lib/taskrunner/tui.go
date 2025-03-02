// SPDX-License-Identifier: Apache-2.0
//
// Copyright (C) 2025 The Happy SDK Authors.
// See the LICENSE file for full licensing details.

package taskrunner

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tui struct {
	taskGroups        []*Group
	groupIndex        int
	executingTaskName string
	width             int
	height            int
	spinner           spinner.Model
	progress          progress.Model

	done          bool
	successes     int
	failures      int
	warnings      int
	skipped       int
	executedTasks int
	totalTasks    int

	failedDeps map[string]bool
}

func (m tui) Init() tea.Cmd {
	group := m.taskGroups[m.groupIndex]
	task := group.getNextTask()

	out := groupTitleStyle.Render(group.title) + "\n"
	out += dividerStyle.Render(strings.Repeat("─", 60))

	return tea.Batch(tea.Println(out), m.executeTask(task, false), m.spinner.Tick)
}

func (m tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}
	case Result:
		var (
			hideOutput bool
			mark       string
			res        string
			skipTask   bool
			currentOut tea.Cmd
		)

		switch msg.State {
		case SUCCESS:
			mark = successMark.String()
			m.successes++

		case WARNING:
			mark = warnMark.String()
			m.warnings++
		case FAILURE:
			mark = failMark.String()
			m.failures++
			m.failedDeps[msg.uuid] = true
		case SKIPPED:
			m.skipped++
			hideOutput = true
			m.failedDeps[msg.uuid] = true
		default:
			mark = infoMark.String()
		}
		if !hideOutput {
			res = fmt.Sprintf("%s %s %s %s",
				mark,
				keyStyle.Render(msg.task),
				valueStyle.Render(msg.Status),
				descStyle.Render(msg.Decription))
		}

		if m.executedTasks >= m.totalTasks-1 {
			// Everything's been installed. We're done!
			m.done = true

			return m, tea.Sequence(
				tea.Println(res), // print the last success message
				tea.Quit,         // exit the program
			)
		}

		// Update progress bar
		m.executedTasks++
		progressCmd := m.progress.SetPercent(float64(m.executedTasks) / float64(m.totalTasks))

		group := m.taskGroups[m.groupIndex]
		task := group.getNextTask()

		if task == nil {
			m.groupIndex++
			group = m.taskGroups[m.groupIndex]
			if group.justStarted() {
				res += "\n\n" + groupTitleStyle.Render(group.title) + "\n"
				res += dividerStyle.Render(strings.Repeat("─", 60))
			}
			task = group.getNextTask()
		} else {
			m.executingTaskName = task.name
		}

		if len(task.dependsOn) > 0 {
			if _, ok := m.failedDeps[task.dependsOn]; ok {
				skipTask = true
			}
		}

		if len(res) > 0 {
			currentOut = tea.Println(res)
		}

		return m, tea.Batch(
			progressCmd,
			currentOut,
			m.executeTask(task, skipTask),
		)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		newModel, cmd := m.progress.Update(msg)
		if newModel, ok := newModel.(progress.Model); ok {
			m.progress = newModel
		}
		return m, cmd
	}
	return m, nil
}

func (m tui) View() string {

	w := lipgloss.Width(fmt.Sprintf("%d", m.totalTasks))

	if m.done {
		return m.checkStatusFinalReport()
	}

	pkgCount := fmt.Sprintf(" %*d/%*d", w, m.executedTasks, w, m.totalTasks)

	spin := m.spinner.View() + " "
	prog := m.progress.View()
	// cellsAvail := max(0, m.width-lipgloss.Width(spin+prog+pkgCount))

	taskName := currentTaskNameStyle.Render(m.executingTaskName)
	// info := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).MaxWidth(cellsAvail).Render("running " + taskName)
	info := currentRunnerStyle.Render("running ") + taskName

	// cellsRemaining := max(0, m.width-lipgloss.Width(spin+info+prog+pkgCount))
	// gap := strings.Repeat("", cellsRemaining)

	return spin + info + "\n\n" + prog + pkgCount
}

func (m tui) checkStatusFinalReport() string {

	freport := ""
	if m.failures > 0 {
		freport += fmt.Sprintf("%s %s = %d", failStyle.Render("FAILURES"), failMark, m.failures)
	}
	if m.warnings > 0 {
		pre := " "
		if m.failures == 0 {
			pre = warnStyle.Render("WARN ")
		}
		freport += fmt.Sprintf("%s%s = %d", pre, warnMark, m.warnings)
	}
	if m.successes > 0 {
		pre := " "
		if m.failures == 0 && m.warnings == 0 {
			pre = successStyle.Render("OK ")
		}
		freport += fmt.Sprintf("%s%s = %d", pre, successMark, m.successes)
	}
	if m.skipped > 0 {
		freport += fmt.Sprintf(" %s = %d", skipMark, m.skipped)
	}
	freport += fmt.Sprintf(" total (%d)", m.totalTasks)
	return doneStyle.Render(freport)
}

func (m tui) executeTask(task *Task, skip bool) tea.Cmd {
	if task == nil {
		return func() tea.Msg {
			return Result{
				State: INFO,
				uuid:  task.uuid.String(),
			}
		}
	}
	if skip {
		return func() tea.Msg {
			return Result{
				State: SKIPPED,
			}
		}
	}
	return func() tea.Msg {
		var res Result
		res.uuid = task.uuid.String()
		if task.action != nil {
			res = task.action()
		}
		res.task = task.name
		return res
	}
}

// func max(a, b int) int {
// 	if a > b {
// 		return a
// 	}
// 	return b
// }
