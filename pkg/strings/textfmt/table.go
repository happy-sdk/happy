// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

// Package textfmt offers a collection of utils for plain text formatting.
package textfmt

import (
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
		b.WriteString(t.buildBorder('┌', '─', '┐', maxColWidth))
		suffixlen := tableWidth - utf8.RuneCountInString(t.Title)

		suffix := ""
		if suffixlen > 0 {
			suffix = strings.Repeat(" ", suffixlen)
		}
		b.WriteString("│ " + t.Title + suffix + " │\n")
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

	for _, row := range t.rows {
		for i, col := range row {
			colLen := utf8.RuneCountInString(col) + 2
			if colLen > maxColWidth[i] {
				maxColWidth[i] = colLen
			}
		}
	}

	totalWidthOfCols := 0
	for _, w := range maxColWidth {
		totalWidthOfCols += w
	}
	totalWidthOfCols++
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
