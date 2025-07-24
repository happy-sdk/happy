// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	statusMessage string

	running  bool
	finished bool

	spinner  spinner.Model
	progress progress.Model

	// statuses
	successes int
	failures  int
	warnings  int
	notices   int
	skipped   int

	executedTasks          int
	totalTasks             int
	progressTotalSteps     float64
	progressCompletedSteps float64

	longestTaskNameLength int
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
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
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case SetStatusMsg:
		m.statusMessage = string(msg)
		m.running = true
		return m, nil

	case addTickMsg:
		m.progressCompletedSteps++
		m.progressTotalSteps++
	case addSubTaskMsg:

		m.totalTasks++
		cmds = append(cmds, m.progress.SetPercent(float64(m.progressCompletedSteps)/float64(m.progressTotalSteps)))
	case subTaskProgressStepsMsg:
		m.progressTotalSteps += msg.steps
	case OutputMsg:
		m.progressCompletedSteps++
		m.progressTotalSteps++
		cmds = append(cmds, m.progress.SetPercent(float64(m.progressCompletedSteps)/float64(m.progressTotalSteps)))

		cmds = append(cmds,
			tea.Println("\033[2K\r", string(msg)),
		)
	case allTasksCompleteMsg:
		cmds = append(cmds,
			m.progress.SetPercent(float64(m.progressCompletedSteps)/float64(m.progressTotalSteps)),
			tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
				return finalExitMsg(msg)
			}),
		)

		return m, tea.Sequence(cmds...)
	case finalExitMsg:
		m.finished = true
		cmds = append(cmds, tea.Println(m.getFinalRaport(msg.ExecDur)), tea.Quit)
	case Result:
		m.executedTasks++
		if !msg.isSubtask {
			m.progressCompletedSteps += progressTaskSteps

			m.running = false
		} else {
			m.progressCompletedSteps += msg.subtaskProgressTaskSteps
		}
		cmds = append(cmds, m.progress.SetPercent(float64(m.progressCompletedSteps)/float64(m.progressTotalSteps)))

		var (
			mark string
		)

		switch msg.state {
		case SUCCESS:
			mark = successMark.String()

			m.successes++

		case NOTICE:
			mark = noticeMark.String()

			m.notices++

		case WARNING:
			mark = warnMark.String()

			m.warnings++

		case FAILURE:
			mark = failMark.String()

			m.failures++

		case SKIPPED:
			mark = skipMark.String()

			m.skipped++

		default:
			mark = infoMark.String()
		}
		cmds = append(cmds, tea.Println(fmt.Sprintf("%s %s %s %s %s",
			mark,
			nameStyle.Width(m.longestTaskNameLength+4).Render(msg.name),
			valueStyle.Render(msg.msg),
			descStyle.Render(msg.decription),
			descStyle.Render("["+infoStyle.Render(msg.dur.String()+"]")))))

	}
	return m, tea.Sequence(cmds...)
}

func (m model) View() string {
	// Show status line only if not finished
	if !m.finished {
		return m.getStatusMessage()
	}
	return ""
}

func (m model) getFinalRaport(execDur time.Duration) string {
	report := ""
	if m.failures > 0 {
		report += fmt.Sprintf("%s %s = %d", failStyle.Render("FAILURES"), failMark, m.failures)
	}
	if m.warnings > 0 {
		pre := " "
		if m.failures == 0 {
			pre = warnStyle.Render("WARN ")
		}
		report += fmt.Sprintf("%s%s = %d", pre, warnMark, m.warnings)
	}
	if m.notices > 0 {
		pre := " "
		if m.failures == 0 && m.warnings == 0 {
			pre = noticeStyle.Render("NOTICES ")
		}
		report += fmt.Sprintf("%s%s = %d", pre, noticeMark, m.notices)
	}
	if m.successes > 0 {
		pre := " "
		if m.failures == 0 && m.warnings == 0 && m.notices == 0 {
			pre = successStyle.Render("OK ")
		}
		report += fmt.Sprintf("%s%s = %d", pre, successMark, m.successes)
	}
	if m.skipped > 0 {
		report += fmt.Sprintf(" %s = %d", skipMark, m.skipped)
	}
	report += fmt.Sprintf(" total (%d) [%s]", m.totalTasks, execDur.String())
	return doneStyle.Render(report)
}

func (m model) getStatusMessage() string {
	taskCount := fmt.Sprintf(" %d/%d",
		m.executedTasks, m.totalTasks,
	)
	spin := m.spinner.View() + " "
	prog := m.progress.View()
	percentage := 100 / m.progressTotalSteps * float64(m.progressCompletedSteps)
	prog += fmt.Sprintf(" %.2f%%", percentage)
	statusMessage := statusMessageStyle.Render(
		fmt.Sprintf("%-*s", m.longestTaskNameLength+1, m.statusMessage),
	)
	// status := currentRunnerStyle.Render("running ") + statusMessage
	return fmt.Sprintf("%s%s%s%s",
		spin, statusMessage, prog, taskCount,
	)
}
