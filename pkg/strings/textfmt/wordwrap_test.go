// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package textfmt

import (
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestWordWrap_Basic(t *testing.T) {
	text := "This is a simple test string that should wrap correctly."
	result := WordWrap(text, 20)

	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 1, "expected text to be wrapped into multiple lines")

	// Verify no line exceeds width
	for _, line := range lines {
		testutils.Assert(t, displayWidth(line) <= 20, "line should not exceed width: %q (width: %d)", line, displayWidth(line))
	}
}

func TestWordWrap_Russian(t *testing.T) {
	// Russian text with Cyrillic characters
	text := "Это тестовая строка на русском языке для проверки переноса слов."
	result := WordWrap(text, 20)

	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 1, "expected Russian text to be wrapped")

	// Verify words are not broken incorrectly
	for _, line := range lines {
		testutils.Assert(t, displayWidth(line) <= 20, "line should not exceed width: %q (width: %d)", line, displayWidth(line))
	}

	// Verify all words are preserved (no broken words)
	allWords := strings.Fields(text)
	resultWords := strings.Fields(result)
	testutils.Equal(t, len(allWords), len(resultWords), "all words should be preserved")
}

func TestWordWrap_Chinese(t *testing.T) {
	// Chinese text with CJK characters (each character takes 2 display columns)
	text := "这是一个测试字符串用于检查单词换行功能。"
	result := WordWrap(text, 20)

	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 1, "expected Chinese text to be wrapped")

	// Verify display width (Chinese characters are wide)
	for _, line := range lines {
		testutils.Assert(t, displayWidth(line) <= 20, "line should not exceed display width: %q (width: %d)", line, displayWidth(line))
	}
}

func TestWordWrap_Japanese(t *testing.T) {
	// Japanese text with Hiragana, Katakana, and Kanji
	text := "これは単語の折り返し機能をテストするためのテスト文字列です。"
	result := WordWrap(text, 20)

	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 1, "expected Japanese text to be wrapped")

	for _, line := range lines {
		testutils.Assert(t, displayWidth(line) <= 20, "line should not exceed display width: %q (width: %d)", line, displayWidth(line))
	}
}

func TestWordWrap_Korean(t *testing.T) {
	// Korean text with Hangul
	text := "이것은 단어 줄바꿈 기능을 테스트하기 위한 테스트 문자열입니다."
	result := WordWrap(text, 20)

	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 1, "expected Korean text to be wrapped")

	for _, line := range lines {
		testutils.Assert(t, displayWidth(line) <= 20, "line should not exceed display width: %q (width: %d)", line, displayWidth(line))
	}
}

func TestWordWrap_MixedLanguages(t *testing.T) {
	// Mixed English and Russian
	text := "Hello Привет World Мир"
	result := WordWrap(text, 15)

	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 1, "expected mixed language text to be wrapped")

	for _, line := range lines {
		testutils.Assert(t, displayWidth(line) <= 15, "line should not exceed width: %q (width: %d)", line, displayWidth(line))
	}
}

func TestWordWrap_Emoji(t *testing.T) {
	// Text with emojis (wide characters)
	text := "Hello 😀 World 🌍 Test 🎉"
	result := WordWrap(text, 20)

	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 1, "expected text with emojis to be wrapped")

	for _, line := range lines {
		testutils.Assert(t, displayWidth(line) <= 20, "line should not exceed display width: %q (width: %d)", line, displayWidth(line))
	}
}

func TestWordWrapWithPrefix(t *testing.T) {
	text := "This is a test string that should wrap with a prefix."
	prefix := "  "
	result := WordWrapWithPrefix(text, 20, prefix)

	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 1, "expected text to be wrapped")

	// Verify all lines have prefix
	for _, line := range lines {
		testutils.Assert(t, strings.HasPrefix(line, prefix), "line should have prefix: %q", line)
		// Verify line width including prefix
		testutils.Assert(t, displayWidth(line) <= 20, "line should not exceed width: %q (width: %d)", line, displayWidth(line))
	}
}

func TestWordWrapWithPrefixes(t *testing.T) {
	text := "This is a test string that should wrap with different prefixes."
	firstPrefix := "> "
	contPrefix := "  "
	result := WordWrapWithPrefixes(text, 20, firstPrefix, contPrefix)

	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 1, "expected text to be wrapped")

	// Verify first line has firstPrefix
	testutils.Assert(t, strings.HasPrefix(lines[0], firstPrefix), "first line should have first prefix: %q", lines[0])

	// Verify continuation lines have contPrefix
	for i := 1; i < len(lines); i++ {
		testutils.Assert(t, strings.HasPrefix(lines[i], contPrefix), "continuation line should have continuation prefix: %q", lines[i])
		testutils.Assert(t, displayWidth(lines[i]) <= 20, "line should not exceed width: %q (width: %d)", lines[i], displayWidth(lines[i]))
	}
}

func TestWordWrap_EmptyString(t *testing.T) {
	result := WordWrap("", 20)
	testutils.Equal(t, "", result, "empty string should return empty string")
}

func TestWordWrap_ZeroWidth(t *testing.T) {
	result := WordWrap("test", 0)
	testutils.Equal(t, "test", result, "zero width should return original string")
}

func TestWordWrap_NegativeWidth(t *testing.T) {
	result := WordWrap("test", -1)
	testutils.Equal(t, "test", result, "negative width should return original string")
}

func TestWordWrap_SingleWord(t *testing.T) {
	text := "Supercalifragilisticexpialidocious"
	result := WordWrap(text, 10)

	// Single long word should still be wrapped (or at least not error)
	testutils.Assert(t, result != "", "result should not be empty")
}

func TestWordWrap_NoSpaces(t *testing.T) {
	// Text without spaces (like a URL or long identifier)
	text := "https://example.com/very/long/path/that/has/no/spaces"
	result := WordWrap(text, 20)

	// Should handle gracefully
	testutils.Assert(t, result != "", "result should not be empty")
}

func TestWordWrap_PreservesWords(t *testing.T) {
	text := "This is a test with multiple words that should be preserved."
	result := WordWrap(text, 15)

	// Split result and verify all original words are present
	originalWords := strings.Fields(text)
	resultText := strings.ReplaceAll(result, "\n", " ")
	resultWords := strings.Fields(resultText)

	testutils.Equal(t, len(originalWords), len(resultWords), "all words should be preserved")
}

func TestDisplayWidth_ASCII(t *testing.T) {
	testutils.Equal(t, 5, displayWidth("hello"), "ASCII string width")
	testutils.Equal(t, 11, displayWidth("hello world"), "ASCII string with space")
}

func TestDisplayWidth_Russian(t *testing.T) {
	// Russian characters are single-width (Cyrillic)
	testutils.Equal(t, 6, displayWidth("Привет"), "Russian string width (6 characters)")
}

func TestDisplayWidth_Chinese(t *testing.T) {
	// Chinese characters are wide (2 columns each)
	testutils.Equal(t, 4, displayWidth("你好"), "Chinese string width (2 chars * 2 = 4)")
	testutils.Equal(t, 4, displayWidth("测试"), "Chinese string width (2 chars * 2 = 4)")
}

func TestDisplayWidth_Japanese(t *testing.T) {
	// Japanese Katakana characters are wide (2 columns each)
	testutils.Equal(t, 6, displayWidth("テスト"), "Japanese string width (3 chars * 2 = 6)")
}

func TestDisplayWidth_Mixed(t *testing.T) {
	// Mixed ASCII and wide characters
	testutils.Equal(t, 6, displayWidth("Hi你好"), "Mixed ASCII and Chinese (2 + 2*2 = 6)")
	testutils.Equal(t, 10, displayWidth("Hello 世界"), "Mixed ASCII and Chinese with space (5 + 1 + 2*2 = 10)")
}

func TestDisplayWidth_Tabs(t *testing.T) {
	testutils.Equal(t, 2, displayWidth("\t"), "Tab width")
	testutils.Equal(t, 4, displayWidth("a\tb"), "String with tab (1 + 2 + 1 = 4)")
}

func TestDisplayWidth_Emoji(t *testing.T) {
	// Emojis are typically wide (2 columns)
	testutils.Equal(t, 2, displayWidth("😀"), "Emoji width")
	testutils.Equal(t, 4, displayWidth("Hi😀"), "Mixed ASCII and emoji (2 + 2 = 4)")
}

func TestWordWrap_VeryLongWord(t *testing.T) {
	// Word longer than line width
	text := "Supercalifragilisticexpialidocious is a very long word"
	result := WordWrap(text, 10)

	// Should handle gracefully - word will be on its own line
	testutils.Assert(t, result != "", "result should not be empty")
	lines := strings.Split(result, "\n")
	testutils.Assert(t, len(lines) > 0, "should have at least one line")
}

func TestWordWrap_MultipleSpaces(t *testing.T) {
	text := "This    has    multiple    spaces"
	result := WordWrap(text, 15)

	// Should handle multiple spaces correctly
	testutils.Assert(t, result != "", "result should not be empty")
	// Verify words are preserved
	originalWords := strings.Fields(text)
	resultText := strings.ReplaceAll(result, "\n", " ")
	resultWords := strings.Fields(resultText)
	testutils.Equal(t, len(originalWords), len(resultWords), "all words should be preserved")
}

func TestWordWrap_Newlines(t *testing.T) {
	text := "Line one\nLine two\nLine three"
	result := WordWrap(text, 10)

	// Should handle existing newlines
	testutils.Assert(t, result != "", "result should not be empty")
}

func TestWordWrap_PrefixTooLong(t *testing.T) {
	text := "Short text"
	longPrefix := "This is a very long prefix that exceeds the line width"
	result := WordWrapWithPrefix(text, 10, longPrefix)

	// Should return original text when prefix is too long
	testutils.Equal(t, text, result, "should return original when prefix too long")
}
