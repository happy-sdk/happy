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
		wordWidth := displayWidth(word)
		effectiveWidth := getCurrentEffectiveWidth()

		if effectiveWidth <= 0 {
			// Prefix too long, just return original
			return text
		}

		// If word is too long to fit on a line, break it at character boundaries
		if wordWidth > effectiveWidth {
			// Break long word into chunks that fit (using runes to handle multi-byte characters)
			runes := []rune(word)
			remainingRunes := runes

			for len(remainingRunes) > 0 {
				chunkRunes := []rune{}
				chunkWidth := 0

				// Build chunk rune by rune until it fits
				for _, r := range remainingRunes {
					runeWidth := 1
					if isWideRune(r) {
						runeWidth = 2
					}

					if chunkWidth+runeWidth > effectiveWidth && chunkWidth > 0 {
						// Current chunk is full, start new line
						break
					}

					chunkRunes = append(chunkRunes, r)
					chunkWidth += runeWidth
				}

				if len(chunkRunes) == 0 {
					// Single rune is too wide, add it anyway
					chunkRunes = []rune{remainingRunes[0]}
					chunkWidth = displayWidth(string(chunkRunes))
					remainingRunes = remainingRunes[1:]
				} else {
					remainingRunes = remainingRunes[len(chunkRunes):]
				}

				chunk := string(chunkRunes)

				// Write current line if needed
				if currentLineLength > 0 {
					writeCurrentLine()
				}

				// Add chunk to new line
				currentWords = append(currentWords, chunk)
				currentLineLength = chunkWidth
			}
			continue
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
// accounting for tabs, wide characters (East Asian), and other special characters.
// Wide characters (CJK, emoji, etc.) take 2 display columns.
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		switch r {
		case '\t':
			width += 2
		default:
			// Check if rune is a wide character (East Asian Wide, East Asian Fullwidth, etc.)
			if isWideRune(r) {
				width += 2
			} else {
				width += 1
			}
		}
	}
	return width
}

// isWideRune determines if a rune takes 2 display columns.
// This includes East Asian Wide, East Asian Fullwidth, and other wide characters.
// Based on Unicode East Asian Width property (Wide and Fullwidth characters).
func isWideRune(r rune) bool {
	// East Asian Wide and Fullwidth characters take 2 display columns
	// These include: CJK (Chinese, Japanese, Korean), some symbols, and emojis

	// CJK Unified Ideographs
	if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
		(r >= 0x2A700 && r <= 0x2B73F) || // CJK Extension C
		(r >= 0x2B740 && r <= 0x2B81F) || // CJK Extension D
		(r >= 0x2B820 && r <= 0x2CEAF) || // CJK Extension E
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
		(r >= 0x2F800 && r <= 0x2FA1F) { // CJK Compatibility Ideographs Supplement
		return true
	}

	// Hiragana and Katakana (Japanese)
	if (r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0x31F0 && r <= 0x31FF) { // Katakana Phonetic Extensions
		return true
	}

	// Hangul (Korean)
	if (r >= 0xAC00 && r <= 0xD7AF) || // Hangul Syllables
		(r >= 0x1100 && r <= 0x11FF) || // Hangul Jamo
		(r >= 0x3130 && r <= 0x318F) { // Hangul Compatibility Jamo
		return true
	}

	// Fullwidth and Halfwidth forms
	if r >= 0xFF00 && r <= 0xFFEF { // Halfwidth and Fullwidth Forms
		// Fullwidth characters (0xFF01-0xFF5E, 0xFFE0-0xFFE6) are wide
		if (r >= 0xFF01 && r <= 0xFF5E) || (r >= 0xFFE0 && r <= 0xFFE6) {
			return true
		}
	}

	// Emojis and symbols that are typically wide
	if isEmojiWide(r) {
		return true
	}

	return false
}

// isEmojiWide checks if a rune is an emoji or other wide symbol that takes 2 columns.
func isEmojiWide(r rune) bool {
	return (r >= 0x1F300 && r <= 0x1F9FF) || // Emoticons, symbols, pictographs
		(r >= 0x2600 && r <= 0x26FF) || // Miscellaneous Symbols
		(r >= 0x2700 && r <= 0x27BF) || // Dingbats
		(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
		(r >= 0x1FA00 && r <= 0x1FAFF) || // Symbols and Pictographs Extended-A
		(r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
		(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map Symbols
		(r >= 0x1F700 && r <= 0x1F77F) || // Alchemical Symbols
		(r >= 0x1F780 && r <= 0x1F7FF) || // Geometric Shapes Extended
		(r >= 0x1F800 && r <= 0x1F8FF) // Supplemental Arrows-C
}
