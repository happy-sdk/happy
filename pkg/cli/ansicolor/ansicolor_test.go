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

// colorsEqual checks if two color.RGBA values are equal.
func colorsEqual(a, b color.RGBA) bool {
	return a.R == b.R && a.G == b.G && a.B == b.B && a.A == b.A
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

// containsSubstring checks if a string contains another string.
func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}

func (c Color) Err() error {
	return c.err
}

func (c Color) Valid() bool {
	return c.valid
}
