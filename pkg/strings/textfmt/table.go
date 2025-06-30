// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

// Package textfmt offers a collection of utils for plain text formatting.
package textfmt

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Table struct {
	Title         string
	WithHeader    bool
	SkipEmptyRows bool
	rows          [][]string
	cols          int
	dividers      map[int]bool
}

func (t *Table) AddRow(cols ...string) {
	if len(cols) == 0 && !t.SkipEmptyRows {
		t.rows = append(t.rows, make([]string, t.cols))
		return
	}
	t.rows = append(t.rows, cols)
	if len(cols) > t.cols {
		t.cols = len(cols)
	}
}

func (t *Table) AddDivider() {
	if t.dividers == nil {
		t.dividers = make(map[int]bool)
	}
	t.dividers[len(t.rows)] = true
}

func (t *Table) String() string {
	if len(t.rows) == 0 || (t.WithHeader && len(t.rows) == 1) {
		return ""
	}

	var b strings.Builder
	maxColWidth, tableWidth := t.calculateMaxColumnWidths()

	if t.Title != "" {
		title := fmt.Sprint(t.Title)
		b.WriteString(t.buildBorder('┌', '─', '┐', maxColWidth))
		suffixlen := tableWidth - utf8.RuneCountInString(title) - 4
		suffix := ""
		if suffixlen > 0 {
			suffix = strings.Repeat(" ", suffixlen)
		}
		b.WriteString("│ " + title + suffix + " │\n")
		b.WriteString(t.buildBorder('├', '┬', '┤', maxColWidth))
	} else {
		b.WriteString(t.buildBorder('┌', '┬', '┐', maxColWidth))
	}

	for i, row := range t.rows {
		// divider
		if t.dividers != nil {
			if _, ok := t.dividers[i]; ok {
				b.WriteString(t.buildBorder('├', '┼', '┤', maxColWidth))
			}
		}
		b.WriteString(t.formatRow(row, maxColWidth))
		// header
		if i == 0 && t.WithHeader {
			b.WriteString(t.buildBorder('├', '┼', '┤', maxColWidth))
		}
	}
	b.WriteString(t.buildBorder('└', '┴', '┘', maxColWidth))
	b.WriteRune('\n')
	return b.String()
}

func (t *Table) calculateMaxColumnWidths() (cw []int, total int) {
	maxColWidth := make([]int, t.cols)

	// Calculate initial column widths based on content
	for _, row := range t.rows {
		for i, col := range row {
			colLen := utf8.RuneCountInString(col) + 2 // +2 for padding spaces
			if colLen > maxColWidth[i] {
				maxColWidth[i] = colLen
			}
		}
	}

	// Ensure minimum column width of 3 (1 char + 2 spaces)
	for i := range maxColWidth {
		if maxColWidth[i] < 3 {
			maxColWidth[i] = 3
		}
	}

	// Calculate total width: borders + column separators + column content
	totalWidthOfCols := 1 // left border
	for i, w := range maxColWidth {
		totalWidthOfCols += w
		if i < len(maxColWidth)-1 {
			totalWidthOfCols += 1 // column separator
		}
	}
	totalWidthOfCols += 1 // right border

	// If we have a title, ensure table is wide enough
	if t.Title != "" && t.cols > 0 {
		titleLen := utf8.RuneCountInString(t.Title)
		minRequiredWidth := titleLen + 4 // title + "│ " + " │"

		if totalWidthOfCols < minRequiredWidth {
			// Extend the last column to accommodate the title
			deficit := minRequiredWidth - totalWidthOfCols
			maxColWidth[len(maxColWidth)-1] += deficit
			totalWidthOfCols = minRequiredWidth
		}
	}

	return maxColWidth, totalWidthOfCols
}

func (t *Table) buildBorder(left, middle, right rune, clens []int) string {
	var b strings.Builder
	b.WriteRune(left)
	for i, l := range clens {
		b.WriteString(strings.Repeat("─", l))
		if i < len(clens)-1 {
			b.WriteRune(middle)
		}
	}
	b.WriteRune(right)
	b.WriteRune('\n')
	return b.String()
}

func (t *Table) formatRow(row []string, colWidths []int) string {
	var b strings.Builder
	for i := 0; i < t.cols; i++ {
		b.WriteRune('│')
		col := ""
		if i < len(row) {
			col = row[i]
		}
		colDisplayWidth := utf8.RuneCountInString(col)
		padding := colWidths[i] - colDisplayWidth - 1 // -1 for space before the text
		if padding < 0 {
			padding = 0
		}
		b.WriteString(" " + col + strings.Repeat(" ", padding))
	}
	b.WriteRune('│')
	b.WriteRune('\n')
	return b.String()
}
