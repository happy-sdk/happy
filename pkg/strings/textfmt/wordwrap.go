// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package textfmt

import (
	"strings"
)

// WordWrap returns string wrapped string to a specified line width.
func WordWrap(text string, lineWidth int) string {
	return WordWrapWithPrefixes(text, lineWidth, "", "")
}

// WordWrapWithPrefix with prefix for each line.
func WordWrapWithPrefix(text string, lineWidth int, prefix string) string {
	return WordWrapWithPrefixes(text, lineWidth, prefix, prefix)
}

// WordWrapWithPrefixes returns string with different
// prefix for first line vs continuation lines.
func WordWrapWithPrefixes(text string, lineWidth int, firstPrefix, contPrefix string) string {
	if lineWidth <= 0 || text == "" {
		return text
	}

	firstPrefixWidth := displayWidth(firstPrefix)
	contPrefixWidth := displayWidth(contPrefix)

	var result strings.Builder
	result.Grow(len(text) + len(text)/lineWidth + len(contPrefix)*len(text)/lineWidth)

	currentLineLength := 0
	var currentWords []string
	linesWritten := 0

	// Helper function to get current effective width
	getCurrentEffectiveWidth := func() int {
		if linesWritten == 0 {
			return lineWidth - firstPrefixWidth
		}
		return lineWidth - contPrefixWidth
	}

	// Helper function to write current line
	writeCurrentLine := func() {
		if linesWritten > 0 {
			result.WriteByte('\n')
		}
		if linesWritten == 0 {
			result.WriteString(firstPrefix)
		} else {
			result.WriteString(contPrefix)
		}
		lineContent := strings.Join(currentWords, " ")
		result.WriteString(lineContent)
		linesWritten++
		currentWords = currentWords[:0]
		currentLineLength = 0
	}

	// Split on whitespace and process words
	for word := range strings.FieldsSeq(text) {
		wordWidth := len(word)
		effectiveWidth := getCurrentEffectiveWidth()

		if effectiveWidth <= 0 {
			// Prefix too long, just return original
			return text
		}

		if currentLineLength == 0 {
			// First word on line
			currentWords = append(currentWords, word)
			currentLineLength = wordWidth
		} else if currentLineLength+1+wordWidth <= effectiveWidth {
			// Word fits on current line (1 is for the space)
			currentWords = append(currentWords, word)
			currentLineLength += 1 + wordWidth
		} else {
			// Word doesn't fit, write current line and start new one
			writeCurrentLine()
			currentWords = append(currentWords, word)
			currentLineLength = wordWidth
		}
	}

	// Write final line
	if len(currentWords) > 0 {
		writeCurrentLine()
	}

	return result.String()
}

// displayWidth calculates the visual/print width of a string
// accounting for tabs and other special characters
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		switch r {
		case '\t':
			width += 2
		default:
			width += 1
		}
	}
	return width
}
