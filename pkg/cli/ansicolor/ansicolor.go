// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2017 The Happy Authors

package ansicolor

import (
	"errors"
	"fmt"
	"image/color"
)

var ErrInvalidHex = errors.New("invalid HEX color code")

var InvalidColor = Color{valid: true}

type Color struct {
	valid bool
	rgb   color.RGBA
	fg    string
	bg    string
	err   error
}

type Theme struct {
	Primary        Color // Primary color for standard text
	Secondary      Color // Secondary color for accentuating text
	Accent         Color // Accent color for highlighting text
	Success        Color // Color for success messages
	Info           Color // Color for informational messages
	Warning        Color // Color for warning messages
	Error          Color // Color for error messages
	Debug          Color // Color for debugging messages
	Notice         Color // Color for notice messages
	NotImplemented Color // Color for not implemented features
	Deprecated     Color // Color for deprecated features or elements
	BUG            Color // Color for bug reports or critical issues
	Light          Color // Light color
	Dark           Color // Dark color
	Muted          Color // Muted color
}

func New() Theme {
	return Theme{
		Primary:        RGB(255, 237, 86),
		Secondary:      RGB(221, 199, 89),
		Accent:         RGB(221, 199, 89),
		Success:        RGB(76, 175, 80),
		Info:           RGB(173, 216, 230),
		Warning:        RGB(255, 152, 0),
		Error:          RGB(213, 0, 0),
		Debug:          RGB(177, 188, 199),
		Notice:         RGB(33, 150, 243),
		NotImplemented: RGB(156, 39, 176),
		Deprecated:     RGB(150, 115, 19),
		BUG:            RGB(244, 67, 54),
		Light:          RGB(245, 245, 245),
		Dark:           RGB(28, 28, 28),
		Muted:          RGB(150, 150, 150),
	}
}

type Flag uint32

const (
	Reset Flag = 1 << iota
	Bold
	Faint
	Italic
	Underline
	Reverse
	Conceal
	CrossedOut
	Overlined
	BrightForeground
	BrightBackground
)

var ansiflags = map[Flag]string{
	Reset:            "0",
	Bold:             "1",
	Faint:            "2",
	Italic:           "3",
	Underline:        "4",
	Reverse:          "7",
	Conceal:          "8",
	CrossedOut:       "9",
	Overlined:        "53",
	BrightForeground: "90",
	BrightBackground: "100",
}

func Text(text string, fg, bg Color, flags Flag) string {
	// Initialize with the ANSI reset code to ensure a clean state
	var str = "\033[0m"
	for flag, code := range ansiflags {
		if flags&flag != 0 {
			str += "\033[" + code + "m"
		}
	}

	// If the foreground color is valid, append its ANSI code
	if fg.valid {
		str += "\033[" + fg.fg + "m"
	}

	// If the background color is valid, append its ANSI code
	if bg.valid {
		str += "\033[" + bg.bg + "m"
	}

	// Append the text and reset the formatting at the end
	str += text + "\033[0m"

	return str
}

func Format(text string, fmtf Flag) string {
	return Style{Format: fmtf}.String(text)
}

type Style struct {
	FG     Color
	BG     Color
	Format Flag
}

func (s Style) String(text string) string {
	return Text(text, s.FG, s.BG, s.Format)
}

// HEX converts a hex color code to an Color.
func HEX(hex string) (c Color) {
	if hex[0] != '#' {
		c = InvalidColor
		c.err = fmt.Errorf("%w: %s", ErrInvalidHex, hex)
		return
	}

	hex2b := func(b byte) byte {
		if c.err != nil {
			return 0
		}
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		c = InvalidColor
		c.err = fmt.Errorf("%w: %c found in %s", ErrInvalidHex, b, hex)
		return 0
	}

	var rgb [3]byte

	switch len(hex) {
	case 4:
		for i := range 3 {
			// scale each nibble
			rgb[i] = hex2b(hex[i+1]) * 0x11
		}
	case 7:
		for i := range 3 {
			rgb[i] = hex2b(hex[i*2+1])<<4 + hex2b(hex[i*2+2])
		}
	default:
		c = InvalidColor
		c.err = fmt.Errorf("%w: %s", ErrInvalidHex, hex)
	}
	if c.err != nil {
		return
	}
	c = RGB(rgb[0], rgb[1], rgb[2])
	return
}

func RGB(r, g, b byte) Color {
	c := Color{rgb: color.RGBA{r, g, b, 0xff}}
	c.fg = toAnsi(c.rgb, '3')
	c.bg = toAnsi(c.rgb, '4')
	c.valid = true
	return c
}

func (c Color) RGB() color.RGBA {
	return c.rgb
}

func toAnsi(rgba color.RGBA, base byte) string {
	return string(base) + "8;2;" + coloritoa(rgba.R) + ";" + coloritoa(rgba.G) + ";" + coloritoa(rgba.B)
}

// coloritoa converts a byte to a string. Used in constructing ANSI color codes.
func coloritoa(t byte) string {
	var (
		a [3]byte
		j = 2
	)
	for i := 0; i < 3; i, j = i+1, j-1 {
		a[j] = '0' + t%10
		if t = t / 10; t == 0 {
			break
		}
	}
	return string(a[j:])
}
