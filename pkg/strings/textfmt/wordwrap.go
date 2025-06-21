// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package textfmt

import (
	"strings"
	"unicode"
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

	firstPrefixLen := len([]rune(firstPrefix))
	contPrefixLen := len([]rune(contPrefix))

	var result strings.Builder
	result.Grow(len(text) + len(text)/lineWidth + len(contPrefix)*len(text)/lineWidth)

	currentLineLength := 0
	var currentWords []string
	isFirstLine := true

	// Split on whitespace and process
	start := 0
	for i, r := range text {
		if unicode.IsSpace(r) {
			if i > start {
				word := text[start:i]
				wordLen := len([]rune(word))

				// Calculate effective width for current line
				effectiveWidth := lineWidth
				if isFirstLine {
					effectiveWidth -= firstPrefixLen
				} else {
					effectiveWidth -= contPrefixLen
				}

				if effectiveWidth <= 0 {
					// Prefix too long, just return original
					return text
				}

				if currentLineLength == 0 {
					currentWords = append(currentWords, word)
					currentLineLength = wordLen
				} else if currentLineLength+1+wordLen <= effectiveWidth {
					currentWords = append(currentWords, word)
					currentLineLength += 1 + wordLen
				} else {
					// Write current line with appropriate prefix
					if !isFirstLine {
						result.WriteByte('\n')
					}
					if isFirstLine {
						result.WriteString(firstPrefix)
					} else {
						result.WriteString(contPrefix)
					}
					result.WriteString(strings.Join(currentWords, " "))
					isFirstLine = false

					// Start new line
					currentWords = currentWords[:0]
					currentWords = append(currentWords, word)
					currentLineLength = wordLen
				}
			}
			start = i + 1
		}
	}

	// Handle last word
	if start < len(text) {
		word := text[start:]
		wordLen := len([]rune(word))

		effectiveWidth := lineWidth
		if isFirstLine {
			effectiveWidth -= firstPrefixLen
		} else {
			effectiveWidth -= contPrefixLen
		}

		if currentLineLength == 0 {
			currentWords = append(currentWords, word)
		} else if currentLineLength+1+wordLen <= effectiveWidth {
			currentWords = append(currentWords, word)
		} else {
			// Write current line
			if !isFirstLine {
				result.WriteByte('\n')
			}
			if isFirstLine {
				result.WriteString(firstPrefix)
			} else {
				result.WriteString(contPrefix)
			}
			result.WriteString(strings.Join(currentWords, " "))
			isFirstLine = false

			currentWords = currentWords[:0]
			currentWords = append(currentWords, word)
		}
	}

	// Write final line
	if len(currentWords) > 0 {
		if !isFirstLine {
			result.WriteByte('\n')
		}
		if isFirstLine {
			result.WriteString(firstPrefix)
		} else {
			result.WriteString(contPrefix)
		}
		result.WriteString(strings.Join(currentWords, " "))
	}

	return result.String()
}
