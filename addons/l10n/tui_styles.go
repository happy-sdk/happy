// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import "charm.land/lipgloss/v2"

// Coverage color thresholds shared by every view that renders a
// per-language percentage (currently just the dashboard): green reads as
// "done enough to ship", yellow as "in progress", red as "needs attention".
const (
	coverageGoodThreshold = 95.0
	coverageOkThreshold   = 70.0
)

var (
	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffed56")).
			Underline(true)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f5f"))

	coverageGoodStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4caf50"))
	coverageOkStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffed56"))
	coverageBadStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f"))
)

// coverageStyle picks the color-coded style for a translation percentage,
// per the thresholds documented above.
func coverageStyle(percentage float64) lipgloss.Style {
	switch {
	case percentage >= coverageGoodThreshold:
		return coverageGoodStyle
	case percentage >= coverageOkThreshold:
		return coverageOkStyle
	default:
		return coverageBadStyle
	}
}
