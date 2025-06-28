// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2017 The Happy Authors

package ansicolor

import (
	"image/color"
	"strings"
	"testing"
)

func TestHEX(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		wantErr bool
		want    Color
	}{
		{"Valid HEX Short #000", "#000", false, RGB(0, 0, 0)},
		{"Valid HEX Short #FFF", "#FFF", false, RGB(255, 255, 255)},
		{"Valid HEX Long #FFFFFF", "#FFFFFF", false, RGB(255, 255, 255)},
		{"Invalid HEX No #", "FFFFFF", true, InvalidColor},
		{"Invalid HEX Wrong Length", "#FFFF", true, InvalidColor},
		{"Invalid HEX Non-Hex Characters", "#ZZZZZZ", true, InvalidColor},
		{"Invalid HEX Short Non-Hex Characters", "#GGG", true, InvalidColor},
		{"Invalid HEX Lowercase", "#ffffff", false, RGB(255, 255, 255)},
		{"Invalid HEX Short Lowercase", "#fff", false, RGB(255, 255, 255)},
		{"Valid HEX Mixed Case", "#FfFfFf", false, RGB(255, 255, 255)},
		{"Invalid HEX Too Long", "#FFFFFFFF", true, InvalidColor},
		{"Valid HEX Digits", "#012345", false, RGB(1, 35, 69)},
		{"Valid HEX Digit and Letter Combo", "#0A1B2C", false, RGB(10, 27, 44)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HEX(tt.hex)
			if (got.Err() != nil) != tt.wantErr {
				t.Errorf("HEX() error = %v, wantErr %v", got.err, tt.wantErr)
			}
			if !tt.wantErr && !colorsEqual(got.RGB(), tt.want.RGB()) {
				t.Errorf("HEX() got = %v, want %v", got.RGB(), tt.want.RGB())
			}
		})
	}
}

func TestRGB(t *testing.T) {
	c := RGB(255, 255, 255)
	if !c.Valid() {
		t.Error("RGB() should create a valid color")
	}
	if c.RGB().R != 255 || c.RGB().G != 255 || c.RGB().B != 255 {
		t.Errorf("RGB() produced incorrect color: got %v", c.RGB())
	}
}

func TestText(t *testing.T) {
	theme := New()
	txt := "Test"
	fg := theme.Primary
	bg := theme.Dark
	flags := Reset // Add more flags to test other cases

	formattedText := Text(txt, fg, bg, flags)
	if formattedText == "" || !containsSubstring(formattedText, txt) {
		t.Errorf("Text() did not format correctly, got %v", formattedText)
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		Text   string
		Flag   Flag
		Result string
	}{
		{"Bold", Bold, "\x1b[1mTest\x1b[0m"},
		{"Italic", Italic, "\x1b[3mTest\x1b[0m"},
		{"Underline", Underline, "\x1b[4mTest\x1b[0m"},
		{"CrossedOut", Strike, "\x1b[9mTest\x1b[0m"},
		{"Bold and Italic", Bold | Italic, "\x1b[1;3mTest\x1b[0m"},
		{"Bold and Underline", Bold | Underline, "\x1b[1;4mTest\x1b[0m"},
		{"Bold and Strike", Bold | Strike, "\x1b[1;9mTest\x1b[0m"},
		{"Italic and Underline", Italic | Underline, "\x1b[3;4mTest\x1b[0m"},
		{"Italic and Strike", Italic | Strike, "\x1b[3;9mTest\x1b[0m"},
		{"Underline and Strike", Underline | Strike, "\x1b[4;9mTest\x1b[0m"},
		{"Bold, Italic, Underline, and Strike", Bold | Italic | Underline | Strike, "\x1b[1;3;4;9mTest\x1b[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.Text, func(t *testing.T) {
			got := Format("Test", tt.Flag)
			if tt.Result != got {
				t.Errorf("Format() got = %q, want %q", got, tt.Result)
			}
		})
	}
}

// colorsEqual checks if two color.RGBA values are equal.
func colorsEqual(a, b color.RGBA) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A
}

// containsSubstring checks if a string contains another string.
func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}

var (
	redColor  = RGB(255, 0, 0)
	blueColor = RGB(0, 0, 255)
)

func BenchmarkText_ShortString(b *testing.B) {
	text := "Hello"
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = Text(text, redColor, blueColor, Bold|Underline)
	}
}

func BenchmarkText_MediumString(b *testing.B) {
	text := "This is a medium length string for benchmarking purposes"
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = Text(text, redColor, blueColor, Bold|Underline)
	}
}

func BenchmarkText_LongString(b *testing.B) {
	text := strings.Repeat("This is a long string used for benchmarking memory allocations and performance. ", 10)
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = Text(text, redColor, blueColor, Bold|Underline)
	}
}

// Benchmark different scenarios
func BenchmarkText_NoColors(b *testing.B) {
	text := "Hello World"
	emptyColor := Color{}
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = Text(text, emptyColor, emptyColor, Bold)
	}
}

func BenchmarkText_OnlyForeground(b *testing.B) {
	text := "Hello World"
	emptyColor := Color{}
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = Text(text, redColor, emptyColor, Bold)
	}
}

func BenchmarkText_OnlyBackground(b *testing.B) {
	text := "Hello World"
	emptyColor := Color{}
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = Text(text, emptyColor, blueColor, Bold)
	}
}

func BenchmarkText_NoFlags(b *testing.B) {
	text := "Hello World"
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = Text(text, redColor, blueColor, 0)
	}
}

func BenchmarkText_ManyFlags(b *testing.B) {
	text := "Hello World"
	flags := Bold | Italic | Underline | Strike | Reverse
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Text(text, redColor, blueColor, flags)
	}
}

func BenchmarkText_Parallel(b *testing.B) {
	text := "Hello World"
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Text(text, redColor, blueColor, Bold|Underline)
		}
	})
}
