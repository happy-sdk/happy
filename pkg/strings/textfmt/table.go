// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package textfmt

import (
	"strings"
	"sync"
	"unicode/utf8"
)

// Pre-allocated slices pool for reuse
var (
	intSlicePool = sync.Pool{
		New: func() any {
			slice := make([]int, 0, 16)
			return &slice
		},
	}

	builderPool = sync.Pool{
		New: func() any {
			return &strings.Builder{}
		},
	}
)

type TableOption func(*Table)

// TableTitle sets the table title
func TableTitle(title string) TableOption {
	return func(t *Table) {
		t.title = title
	}
}

// TableWithHeader sets whether the table has a header row
func TableWithHeader() TableOption {
	return func(t *Table) {
		t.withHeader = true
	}
}

// TableSkipEmptyRows sets whether to skip empty rows
func TableSkipEmptyRows(skip bool) TableOption {
	return func(t *Table) {
		t.skipEmptyRows = skip
	}
}

// Table represents a thread-safe plain text table structure
type Table struct {
	mu            sync.RWMutex
	title         string
	withHeader    bool
	skipEmptyRows bool
	rows          [][]string
	cols          int
	dividers      map[int]struct{}
	children      []*Table

	// Cache frequently computed values
	cachedGlobalWidth int
	cacheValid        bool
}

// NewTable creates a new Table with proper initialization
func NewTable(opts ...TableOption) *Table {
	t := &Table{
		dividers: make(map[int]struct{}),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// AddRow adds a row to the table
func (t *Table) AddRow(cols ...string) {
	if len(cols) == 0 {
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.skipEmptyRows {
			return
		}
		t.rows = append(t.rows, []string{})
		t.invalidateCache()
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Pre-allocate if we know the exact size
	row := make([]string, len(cols))
	copy(row, cols)

	t.rows = append(t.rows, row)
	if len(cols) > t.cols {
		t.cols = len(cols)
	}
	t.invalidateCache()
}

// AddRows adds multiple rows efficiently
func (t *Table) AddRows(rows [][]string) {
	if len(rows) == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.addRows(rows)
}

func (t *Table) addRows(rows [][]string) {
	if len(rows) == 0 {
		return
	}
	// Pre-allocate capacity
	newSize := len(t.rows) + len(rows)
	if cap(t.rows) < newSize {
		newRows := make([][]string, len(t.rows), newSize)
		copy(newRows, t.rows)
		t.rows = newRows
	}

	for _, row := range rows {
		if len(row) == 0 {
			if !t.skipEmptyRows {
				t.rows = append(t.rows, []string{})
			}
			continue
		}

		// Copy to avoid sharing slices
		newRow := make([]string, len(row))
		copy(newRow, row)
		t.rows = append(t.rows, newRow)

		if len(row) > t.cols {
			t.cols = len(row)
		}
	}
	t.invalidateCache()
}

// AddDivider adds a divider at the current row position
func (t *Table) AddDivider() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.dividers[len(t.rows)] = struct{}{}
	t.invalidateCache()
}

// Append adds a child table
func (t *Table) Append(child *Table) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.children = append(t.children, child)
	t.invalidateCache()
}

// String returns the string representation of the table
func (t *Table) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.hasContent() {
		return ""
	}

	globalWidth := t.calculateGlobalWidth()

	b := builderPool.Get().(*strings.Builder)
	defer func() {
		b.Reset()
		builderPool.Put(b)
	}()

	lastCw, bottomBorder := t.writeContent(b, globalWidth, false, "")
	if lastCw == nil {
		return b.String()
	}

	b.WriteString(bottomBorder)
	b.WriteRune('\n')
	return b.String()
}

// invalidateCache marks the cache as invalid
func (t *Table) invalidateCache() {
	t.cacheValid = false
}

// hasContent checks if table has content
func (t *Table) hasContent() bool {
	if len(t.rows) > 0 && (!t.withHeader || len(t.rows) != 1) {
		return true
	}
	for _, c := range t.children {
		if c.hasContent() {
			return true
		}
	}
	return false
}

// calculateGlobalWidth calculates the global width
func (t *Table) calculateGlobalWidth() int {
	if t.cacheValid {
		return t.cachedGlobalWidth
	}

	maxW := 0
	if t.title != "" {
		maxW = utf8.RuneCountInString(t.title) + 4
	}

	cw := t.calculateLocalColumnWidths()
	contentW := 0
	if t.cols > 0 {
		contentW = t.cols + 1
		for _, w := range cw {
			contentW += w
		}
	}
	if contentW > maxW {
		maxW = contentW
	}

	for _, c := range t.children {
		childW := c.calculateGlobalWidth()
		if childW > maxW {
			maxW = childW
		}
	}

	t.cachedGlobalWidth = maxW
	t.cacheValid = true
	return maxW
}

// calculateLocalColumnWidths calculates column widths
func (t *Table) calculateLocalColumnWidths() []int {
	if t.cols == 0 {
		return nil
	}

	cwPtr := intSlicePool.Get().(*[]int)
	cw := *cwPtr
	defer func() {
		*cwPtr = (*cwPtr)[:0] // Reset slice but keep capacity
		intSlicePool.Put(cwPtr)
	}()

	// Ensure we have enough capacity
	if cap(cw) < t.cols {
		cw = make([]int, t.cols)
	} else {
		cw = cw[:t.cols]
		// Clear existing values
		for i := range cw {
			cw[i] = 0
		}
	}

	for _, row := range t.rows {
		for i, col := range row {
			if i >= len(cw) {
				break
			}
			colLen := utf8.RuneCountInString(col) + 2
			if colLen > cw[i] {
				cw[i] = colLen
			}
		}
	}

	// Apply minimum width
	for i := range cw {
		if cw[i] < 3 {
			cw[i] = 3
		}
	}

	// Copy result since we're returning the slice from pool
	result := make([]int, len(cw))
	copy(result, cw)
	return result
}

// adjustColumnWidths adjusts column widths to match global width
func (t *Table) adjustColumnWidths(localCw []int, globalWidth int) []int {
	if len(localCw) == 0 {
		return nil
	}

	adjusted := make([]int, len(localCw))
	copy(adjusted, localCw)

	contentW := len(adjusted) + 1
	for _, w := range adjusted {
		contentW += w
	}

	deficit := globalWidth - contentW
	if deficit > 0 {
		adjusted[len(adjusted)-1] += deficit
	}

	return adjusted
}

// writeContent writes table content to buffer
func (t *Table) writeContent(b *strings.Builder, globalWidth int, isSub bool, parentBottomBorder string) ([]int, string) {
	localCw := t.calculateLocalColumnWidths()
	adjustedCw := t.adjustColumnWidths(localCw, globalWidth)
	lastCw := adjustedCw

	if isSub {
		var topBorder string
		if t.title != "" {

			if parentBottomBorder != "" {
				topBorder = strings.ReplaceAll(parentBottomBorder, "└", "├")
				topBorder = strings.ReplaceAll(topBorder, "┘", "┤")
			}
			b.WriteString(topBorder)
			suffixlen := globalWidth - utf8.RuneCountInString(t.title) - 4
			if suffixlen >= 0 {
				b.WriteString("│ ")
				b.WriteString(t.title)
				b.WriteString(strings.Repeat(" ", suffixlen))
				b.WriteString(" │\n")
			}
		}

		if len(t.rows) > 0 {
			childTopBorder := t.buildBorder('├', '┬', '┤', adjustedCw)
			cborder := []rune(childTopBorder)
			pborder := []rune(parentBottomBorder)
			lastcharix := len(cborder) - 1

			hasTitle := t.title != ""
			if len(cborder) == len(pborder) {
				for i, crune := range cborder {
					if i >= len(pborder) {
						break
					}
					prune := pborder[i]
					if prune == crune || i == 0 || i == lastcharix {
						continue
					}
					if crune == '─' && !hasTitle {
						cborder[i] = prune
					}

				}
				b.WriteString(string(cborder))
			} else {
				// NOTE: Potential issue with border length mismatch
				b.WriteString(childTopBorder)
			}
		}
	} else {
		if t.title != "" {
			b.WriteString(t.buildBorder('┌', '─', '┐', []int{globalWidth - 2}))
			suffixlen := globalWidth - utf8.RuneCountInString(t.title) - 4
			if suffixlen >= 0 {
				b.WriteString("│ ")
				b.WriteString(t.title)
				b.WriteString(strings.Repeat(" ", suffixlen))
				b.WriteString(" │\n")
			}
			if len(t.rows) > 0 {
				b.WriteString(t.buildBorder('├', '┬', '┤', adjustedCw))
			}
		} else if len(t.rows) > 0 {
			b.WriteString(t.buildBorder('┌', '┬', '┐', adjustedCw))
		}
	}

	for i, row := range t.rows {
		if t.skipEmptyRows && len(row) == 0 {
			continue
		}
		if _, ok := t.dividers[i]; ok {
			b.WriteString(t.buildBorder('├', '┼', '┤', adjustedCw))
		}
		b.WriteString(t.formatRow(row, adjustedCw))
		if i == 0 && t.withHeader {
			b.WriteString(t.buildBorder('├', '┼', '┤', adjustedCw))
		}
	}

	var childLastCw []int
	// Use full width border if no columns, otherwise use adjusted column widths
	if len(adjustedCw) == 0 {
		parentBottomBorder = t.buildBorder('└', '─', '┘', []int{globalWidth - 2})
	} else {
		parentBottomBorder = t.buildBorder('└', '┴', '┘', adjustedCw)
	}
	for _, child := range t.children {
		childLastCw, parentBottomBorder = child.writeContent(b, globalWidth, true, parentBottomBorder)
		if childLastCw != nil {
			lastCw = childLastCw
		}
	}

	return lastCw, t.buildBorder('└', '┴', '┘', lastCw)
}

// buildBorder builds a border string with given characters
func (t *Table) buildBorder(left, middle, right rune, clens []int) string {
	if len(clens) == 0 {
		return string([]rune{left, right, '\n'})
	}

	// Pre-calculate capacity to avoid reallocations
	capacity := 3 // left + right + newline
	for _, l := range clens {
		capacity += l
	}
	capacity += len(clens) - 1 // middle characters

	b := builderPool.Get().(*strings.Builder)
	defer func() {
		b.Reset()
		builderPool.Put(b)
	}()

	b.Grow(capacity)
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

// formatRow formats a single row
func (t *Table) formatRow(row []string, colWidths []int) string {
	if len(colWidths) == 0 {
		return "│\n"
	}

	// Pre-calculate capacity
	capacity := 2 // start and end │
	for _, w := range colWidths {
		capacity += w + 1 // width + │
	}

	b := builderPool.Get().(*strings.Builder)
	defer func() {
		b.Reset()
		builderPool.Put(b)
	}()

	b.Grow(capacity)
	b.WriteRune('│')

	if len(row) == 0 {
		sumCw := 0
		for _, w := range colWidths {
			sumCw += w
		}
		b.WriteString(strings.Repeat(" ", sumCw+len(colWidths)-1))
	} else {
		for i := 0; i < len(row) && i < len(colWidths)-1; i++ {
			col := row[i]
			colDisplayWidth := utf8.RuneCountInString(col)
			padding := colWidths[i] - colDisplayWidth - 1

			b.WriteString(" ")
			b.WriteString(col)
			if padding > 0 {
				b.WriteString(strings.Repeat(" ", padding))
			}
			b.WriteRune('│')
		}

		// Handle last column
		if len(row) > 0 {
			lastIdx := min(len(row)-1, len(colWidths)-1)
			col := row[lastIdx]
			colDisplayWidth := utf8.RuneCountInString(col)

			lastWidth := 0
			for j := lastIdx; j < len(colWidths); j++ {
				lastWidth += colWidths[j]
			}
			lastWidth += (len(colWidths) - lastIdx - 1)

			padding := lastWidth - colDisplayWidth - 1
			b.WriteString(" ")
			b.WriteString(col)
			if padding > 0 {
				b.WriteString(strings.Repeat(" ", padding))
			}
		}
	}

	b.WriteRune('│')
	b.WriteRune('\n')
	return b.String()
}

// TableBatchOp Batch operations when adding rows to table for better performance
type TableBatchOp struct {
	rows     [][]string
	dividers []int
}

// NewTableBatchOp creates a new table batch operation
func NewTableBatchOp() *TableBatchOp {
	return &TableBatchOp{
		rows:     make([][]string, 0),
		dividers: make([]int, 0),
	}
}

// AddRow adds a row to the table batch op
func (bo *TableBatchOp) AddRow(cols ...string) {
	row := make([]string, len(cols))
	copy(row, cols)
	bo.rows = append(bo.rows, row)
}

// AddDivider adds a divider at the current position
func (bo *TableBatchOp) AddDivider() {
	bo.dividers = append(bo.dividers, len(bo.rows))
}

// Batch executes the batch operation on the table
func (t *Table) Batch(batch *TableBatchOp) {
	t.mu.Lock()
	defer t.mu.Unlock()

	startRow := len(t.rows)

	// Add rows
	t.addRows(batch.rows)

	// Add dividers
	for _, divPos := range batch.dividers {
		adjustedPos := startRow + divPos
		t.dividers[adjustedPos] = struct{}{}
	}

	t.invalidateCache()
}
