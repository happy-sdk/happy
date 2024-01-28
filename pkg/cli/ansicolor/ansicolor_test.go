// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2017 The Happy Authors

package ansicolor

import (
	"regexp"
	"strings"
	"testing"
)

func TestText(t *testing.T) {
	cases := []struct {
		text     string
		fg, bg   Color
		format   uint
		expected string
	}{
		{"No Color", 0, 0, 0, "No Color"},
		{"Hello, world!", FgRed, BgBlack, 1, "\033[1;31;40mHello, world!\033[0m"},
		{"Test", FgGreen, BgWhite, 0, "\033[32;47mTest\033[0m"},
		{"", FgBlue, BgYellow, 0, "\033[34;43m\033[0m"},                   // Test with empty string
		{"Sample", FgCyan, 0, 0, "\033[36mSample\033[0m"},                 // Test with only foreground color
		{"Text", 0, BgMagenta, 4, "\033[4;45mText\033[0m"},                // Test with only background color and underline
		{"Bold Text", FgWhite, BgRed, 1, "\033[1;37;41mBold Text\033[0m"}, // Test with bold formatting
	}

	for _, c := range cases {
		got := Text(c.text, c.fg, c.bg, c.format)
		if got != c.expected {
			t.Errorf("Text(%q, %v, %v, %d) == %q, want %q", c.text, c.fg, c.bg, c.format, got, c.expected)
		}
	}
}

func TestTextPadded(t *testing.T) {
	cases := []struct {
		text     string
		fg, bg   Color
		format   uint
		expected string
	}{
		// Basic test cases
		{"Hello, world!", FgRed, BgBlack, 1, "\033[1;31;40m Hello, world! \033[0m"},
		{"Test", FgGreen, BgWhite, 0, "\033[32;47m Test \033[0m"},

		// Testing with no color and format
		{"No Color", 0, 0, 0, " No Color "},
	}

	for _, c := range cases {
		got := TextPadded(c.text, c.fg, c.bg, c.format)
		if got != c.expected {
			t.Errorf("TextPadded(%q, %v, %v, %d) == %q, want %q", c.text, c.fg, c.bg, c.format, got, c.expected)
		}
	}
}

func TestFgRGB(t *testing.T) {
	cases := []struct {
		r, g, b  byte
		expected Color
	}{
		// Test cases with different RGB values
		{255, 0, 0, Color(255)<<8 | fgRGB},                                       // Red
		{0, 255, 0, Color(255)<<16 | fgRGB},                                      // Green
		{0, 0, 255, Color(255)<<24 | fgRGB},                                      // Blue
		{255, 255, 255, Color(255)<<8 | Color(255)<<16 | Color(255)<<24 | fgRGB}, // White
	}

	for _, c := range cases {
		got := FgRGB(c.r, c.g, c.b)
		if got != c.expected {
			t.Errorf("FgRGB(%d, %d, %d) == %v, want %v", c.r, c.g, c.b, got, c.expected)
		}
	}
}

func TestBgRGB(t *testing.T) {
	cases := []struct {
		r, g, b  byte
		expected Color
	}{
		// Test cases with different RGB values
		{255, 0, 0, Color(255)<<8 | bgRGB},                                       // Red
		{0, 255, 0, Color(255)<<16 | bgRGB},                                      // Green
		{0, 0, 255, Color(255)<<24 | bgRGB},                                      // Blue
		{255, 255, 255, Color(255)<<8 | Color(255)<<16 | Color(255)<<24 | bgRGB}, // White
	}

	for _, c := range cases {
		got := BgRGB(c.r, c.g, c.b)
		if got != c.expected {
			t.Errorf("BgRGB(%d, %d, %d) == %v, want %v", c.r, c.g, c.b, got, c.expected)
		}
	}
}

func TestBrightColorCodes(t *testing.T) {
	cases := []struct {
		color    Color
		expected string // Expected ANSI code for bright colors
	}{
		{FgBlack | FgRed, "\033[31m"}, // Combining FgBlack and FgRed to ensure colorValue <= 15
	}

	for _, c := range cases {
		got := Text("test", c.color, 0, 0) // Text function will use buildColorCode internally
		if !strings.Contains(got, c.expected) {
			t.Errorf("Expected ANSI code %q not found in %q", c.expected, got)
		}
	}
}

func TestColoritoa(t *testing.T) {
	cases := []struct {
		input    byte
		expected string
	}{
		{0, "0"},
		{9, "9"},
		{10, "10"},
		{255, "255"},
		// Add more cases as needed
	}

	for _, c := range cases {
		got := coloritoa(c.input)
		if got != c.expected {
			t.Errorf("coloritoa(%d) == %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestTextFormatting(t *testing.T) {
	// Test various formatting options
	formattingCases := []struct {
		format   uint
		expected string // Expected ANSI formatting code
	}{
		{1, "\033[1m"}, // Bold
		{4, "\033[4m"}, // Underline
		// Add more formatting options as needed
	}

	for _, c := range formattingCases {
		got := Text("test", 0, 0, c.format)
		if !strings.Contains(got, c.expected) {
			t.Errorf("Expected formatting code %q not found in %q", c.expected, got)
		}
	}
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(str string) string {
	// Regular expression to match ANSI escape codes
	ansiRegex := regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")
	// Replace all ANSI escape codes with an empty string
	return ansiRegex.ReplaceAllString(str, "")
}