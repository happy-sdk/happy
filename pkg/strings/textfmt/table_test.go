// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package textfmt

import (
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestTable_Basic(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("Name", "Age", "City")
	tbl.AddRow("Alice", "30", "New York")
	tbl.AddRow("Bob", "25", "London")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Alice"), "table should contain data")
	testutils.Assert(t, strings.Contains(result, "Bob"), "table should contain data")
}

func TestTable_WithHeader(t *testing.T) {
	tbl := NewTable(TableWithHeader())
	tbl.AddRow("Name", "Age")
	tbl.AddRow("Alice", "30")
	tbl.AddRow("Bob", "25")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Name"), "table should contain header")
	testutils.Assert(t, strings.Contains(result, "Age"), "table should contain header")
	// Should have divider after header
	testutils.Assert(t, strings.Contains(result, "├"), "table should have header divider")
}

func TestTable_WithTitle(t *testing.T) {
	tbl := NewTable(TableTitle("Employee List"))
	tbl.AddRow("Name", "Age")
	tbl.AddRow("Alice", "30")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Employee List"), "table should contain title")
}

func TestTable_Russian(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("Имя", "Возраст", "Город")
	tbl.AddRow("Анна", "25", "Москва")
	tbl.AddRow("Иван", "30", "Санкт-Петербург")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Анна"), "table should contain Russian text")
	testutils.Assert(t, strings.Contains(result, "Иван"), "table should contain Russian text")

	// Verify columns are properly aligned (Russian characters are single-width)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, "│") && !strings.Contains(line, "─") {
			// Count separators to verify structure
			separators := strings.Count(line, "│")
			testutils.Assert(t, separators >= 3, "row should have proper column separators")
		}
	}
}

func TestTable_Chinese(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("姓名", "年龄", "城市")
	tbl.AddRow("张三", "25", "北京")
	tbl.AddRow("李四", "30", "上海")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "张三"), "table should contain Chinese text")
	testutils.Assert(t, strings.Contains(result, "李四"), "table should contain Chinese text")

	// Chinese characters are wide (2 columns each), so columns should be wider
	// Verify the table structure is correct
	lines := strings.Split(result, "\n")
	hasData := false
	for _, line := range lines {
		if strings.Contains(line, "│") && strings.Contains(line, "张三") {
			hasData = true
			// Verify proper formatting
			testutils.Assert(t, strings.Count(line, "│") >= 3, "row should have proper column separators")
		}
	}
	testutils.Assert(t, hasData, "table should contain formatted Chinese data")
}

func TestTable_Japanese(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("名前", "年齢", "都市")
	tbl.AddRow("太郎", "25", "東京")
	tbl.AddRow("花子", "30", "大阪")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "太郎"), "table should contain Japanese text")
	testutils.Assert(t, strings.Contains(result, "花子"), "table should contain Japanese text")
}

func TestTable_Korean(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("이름", "나이", "도시")
	tbl.AddRow("철수", "25", "서울")
	tbl.AddRow("영희", "30", "부산")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "철수"), "table should contain Korean text")
	testutils.Assert(t, strings.Contains(result, "영희"), "table should contain Korean text")
}

func TestTable_MixedLanguages(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("Name/姓名", "Age/年龄", "City/城市")
	tbl.AddRow("Alice/爱丽丝", "30", "New York/纽约")
	tbl.AddRow("Bob/鲍勃", "25", "London/伦敦")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Alice"), "table should contain English text")
	testutils.Assert(t, strings.Contains(result, "爱丽丝"), "table should contain Chinese text")
}

func TestTable_Emoji(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("Name", "Status", "City")
	tbl.AddRow("Alice 😀", "Active ✅", "New York 🌆")
	tbl.AddRow("Bob 🎉", "Inactive ❌", "London 🇬🇧")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "😀"), "table should contain emoji")
	testutils.Assert(t, strings.Contains(result, "✅"), "table should contain emoji")
}

func TestTable_EmptyRows(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("Name", "Age")
	tbl.AddRow() // Empty row
	tbl.AddRow("Alice", "30")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")

	// With skipEmptyRows=false (default), empty row should be included
	tbl2 := NewTable(TableSkipEmptyRows(true))
	tbl2.AddRow("Name", "Age")
	tbl2.AddRow() // Empty row - should be skipped
	tbl2.AddRow("Alice", "30")

	result2 := tbl2.String()
	testutils.Assert(t, result2 != "", "table should not be empty")
}

func TestTable_Dividers(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("Name", "Age")
	tbl.AddRow("Alice", "30")
	tbl.AddDivider()
	tbl.AddRow("Bob", "25")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "├"), "table should contain divider")
	testutils.Assert(t, strings.Contains(result, "┼"), "table should contain divider")
}

func TestTable_AddRows(t *testing.T) {
	tbl := NewTable()
	rows := [][]string{
		{"Name", "Age"},
		{"Alice", "30"},
		{"Bob", "25"},
	}
	tbl.AddRows(rows)

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Alice"), "table should contain data from AddRows")
	testutils.Assert(t, strings.Contains(result, "Bob"), "table should contain data from AddRows")
}

func TestTable_Nested(t *testing.T) {
	parent := NewTable(TableTitle("Parent Table"))
	parent.AddRow("Name", "Details")
	parent.AddRow("Alice", "See below")

	child := NewTable(TableTitle("Child Table"))
	child.AddRow("Key", "Value")
	child.AddRow("Age", "30")
	child.AddRow("City", "New York")

	parent.Append(child)

	result := parent.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Parent Table"), "table should contain parent title")
	testutils.Assert(t, strings.Contains(result, "Child Table"), "table should contain child title")
}

func TestTable_ColumnWidthCalculation(t *testing.T) {
	// Test that column widths are calculated correctly for wide characters
	tbl := NewTable()
	tbl.AddRow("Short", "Very Long Column Header", "中")
	tbl.AddRow("A", "B", "中文内容")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")

	// Verify that columns are properly aligned
	// The "中" column should accommodate "中文内容" (4 display columns)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, "│") && strings.Contains(line, "中文内容") {
			// Verify proper formatting
			testutils.Assert(t, strings.Count(line, "│") >= 3, "row should have proper column separators")
		}
	}
}

func TestTable_MixedWidthColumns(t *testing.T) {
	// Test table with mixed ASCII and wide characters
	tbl := NewTable()
	tbl.AddRow("ASCII", "中文", "Mixed/混合")
	tbl.AddRow("Hello", "测试", "Hi你好")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Hello"), "table should contain ASCII text")
	testutils.Assert(t, strings.Contains(result, "测试"), "table should contain Chinese text")
	testutils.Assert(t, strings.Contains(result, "Hi你好"), "table should contain mixed text")
}

// TestTable_RaggedRow is a regression test: formatRow's first loop and its
// "last column" block both processed the same final row index whenever a
// row had fewer columns than the table (e.g. a 1-column row in a 5-column
// table), rendering that cell's content twice and corrupting the
// right-hand box-drawing border.
func TestTable_RaggedRow(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("a", "b", "c", "d", "e")
	tbl.AddRow("short")

	result := tbl.String()
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")

	testutils.Equal(t, 4, len(lines), "expected border, full row, short row, border")

	shortRowLine := lines[2]
	testutils.Equal(t, 1, strings.Count(shortRowLine, "short"), "short row content must not be duplicated")

	// Every content line must have exactly two "│" (the row's own left and
	// right borders) -- a duplicated short-row render leaves a stray
	// mid-line "│" from the first loop's own closing border.
	testutils.Equal(t, 2, strings.Count(shortRowLine, "│"), "short row must have exactly 2 border characters")

	// All rendered lines (borders use ─ joined by ┌/┬/┐ etc, content lines
	// use │) must be the same display width, confirming alignment.
	width := len([]rune(lines[0]))
	for i, line := range lines {
		testutils.Equal(t, width, len([]rune(line)), "line %d must match table width", i)
	}
}

func TestTable_EmptyTable(t *testing.T) {
	tbl := NewTable()
	result := tbl.String()
	testutils.Equal(t, "", result, "empty table should return empty string")
}

func TestTable_OnlyHeader(t *testing.T) {
	tbl := NewTable(TableWithHeader())
	tbl.AddRow("Name", "Age")
	// No data rows

	result := tbl.String()
	// Table with only header and no data should be empty
	testutils.Equal(t, "", result, "table with only header and no data should be empty")
}

func TestTable_BatchOperations(t *testing.T) {
	tbl := NewTable()
	batch := NewTableBatchOp()
	batch.AddRow("Name", "Age")
	batch.AddRow("Alice", "30")
	batch.AddDivider()
	batch.AddRow("Bob", "25")

	tbl.Batch(batch)

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Alice"), "table should contain batch data")
	testutils.Assert(t, strings.Contains(result, "Bob"), "table should contain batch data")
	testutils.Assert(t, strings.Contains(result, "├"), "table should contain divider from batch")
}

func TestTable_VeryLongContent(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("Short", "Very Long Column Content That Should Wrap")
	tbl.AddRow("A", "B")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "Very Long"), "table should contain long content")
}

func TestTable_UnicodeInTitle(t *testing.T) {
	tbl := NewTable(TableTitle("员工列表 / Employee List"))
	tbl.AddRow("Name", "Age")
	tbl.AddRow("Alice", "30")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	testutils.Assert(t, strings.Contains(result, "员工列表"), "table should contain Chinese in title")
	testutils.Assert(t, strings.Contains(result, "Employee List"), "table should contain English in title")
}

func TestTable_ColumnAlignment(t *testing.T) {
	// Test that columns with different content types align correctly
	tbl := NewTable()
	tbl.AddRow("ASCII", "中文", "Mixed")
	tbl.AddRow("A", "测试", "Hi你好")
	tbl.AddRow("Very Long ASCII Text", "短", "Test")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")

	// Verify all rows have the same number of column separators
	lines := strings.Split(result, "\n")
	separatorCounts := []int{}
	for _, line := range lines {
		if strings.Contains(line, "│") && !strings.Contains(line, "─") && !strings.Contains(line, "┼") {
			count := strings.Count(line, "│")
			if count > 0 {
				separatorCounts = append(separatorCounts, count)
			}
		}
	}

	// All data rows should have the same number of separators
	if len(separatorCounts) > 1 {
		firstCount := separatorCounts[0]
		for i := 1; i < len(separatorCounts); i++ {
			testutils.Equal(t, firstCount, separatorCounts[i], "all rows should have same number of columns")
		}
	}
}

func TestTable_SpecialCharacters(t *testing.T) {
	tbl := NewTable()
	tbl.AddRow("Name", "Value")
	tbl.AddRow("Test", "Value with\nnewline")
	tbl.AddRow("Another", "Value with\ttab")

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	// Table should handle special characters gracefully
}

func TestTable_ConcurrentAccess(t *testing.T) {
	// Test that table is thread-safe
	tbl := NewTable()

	// Simulate concurrent access
	done := make(chan bool, 2)

	go func() {
		tbl.AddRow("Name", "Age")
		tbl.AddRow("Alice", "30")
		done <- true
	}()

	go func() {
		tbl.AddRow("Bob", "25")
		tbl.AddRow("Charlie", "35")
		done <- true
	}()

	<-done
	<-done

	result := tbl.String()
	testutils.Assert(t, result != "", "table should not be empty")
	// Should have at least some rows
	testutils.Assert(t, strings.Contains(result, "│"), "table should have proper structure")
}
