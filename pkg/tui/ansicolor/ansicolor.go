// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2017 The Happy Authors

package ansicolor

import (
	"errors"
	"fmt"
	"image/color"
	"sync"
)

var (
	ErrInvalidHex = errors.New("invalid HEX color code")
	InvalidColor  = Color{valid: false}
)

type Flag uint32

const (
	Reset Flag = 1 << iota
	Bold
	Faint
	Italic
	Underline
	Reverse
	Conceal
	Strike // CrossedOut
	Overlined
	BrightForeground
	BrightBackground
)

const (
	semi    = ';'
	m       = 'm'
	control = '\033'
	lsqb    = '['
)

var flagOrder = []Flag{
	Reset,
	Bold,
	Faint,
	Italic,
	Underline,
	Reverse,
	Conceal,
	Strike,
	Overlined,
	BrightForeground,
	BrightBackground,
}

var ansiflags map[Flag]rune

func init() {
	ansiflags = sync.OnceValue(func() map[Flag]rune {
		flags := map[Flag]rune{
			Reset:            '0',
			Bold:             '1',
			Faint:            '2',
			Italic:           '3',
			Underline:        '4',
			Reverse:          '7',
			Conceal:          '8',
			Strike:           '9',
			Overlined:        53,
			BrightForeground: 90,
			BrightBackground: 100,
		}
		return flags
	})()
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

func Text(text string, fg, bg Color, flags Flag) string {
	// Calculate exact size needed
	size := len(text) + 7 // \033[ + m + \033[0m

	if flags > 0 {
		for _, flag := range flagOrder {
			if flags&flag != 0 {
				size += 2 // flag digit + semicolon
			}
		}
	}

	if fg.valid {
		size += len(fg.fg) + 1 // +1 for semicolon
	}
	if bg.valid {
		size += len(bg.bg) + 1 // +1 for semicolon
	}

	// Single allocation
	result := make([]byte, 0, size)
	result = append(result, control, lsqb)

	// Build the sequence
	needsSemi := false

	if flags > 0 {
		for _, flag := range flagOrder {
			if flags&flag != 0 {
				if needsSemi {
					result = append(result, semi)
				}
				result = append(result, byte(ansiflags[flag]))
				needsSemi = true
			}
		}
	}

	if fg.valid {
		result = append(result, fg.fg...)
	}

	if bg.valid {
		result = append(result, bg.bg...)
	}
	lastIndex := len(result) - 1
	if result[lastIndex] == semi {
		result[lastIndex] = m
	} else {
		result = append(result, m)
	}

	result = append(result, text...)
	result = append(result, control, lsqb, '0', m)

	return string(result)
}

func Format(text string, fmtf Flag) string {
	return Style{Format: fmtf}.String(text)
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

type Style struct {
	FG     Color
	BG     Color
	Format Flag
}

func (s Style) String(text string) string {
	return Text(text, s.FG, s.BG, s.Format)
}

type Color struct {
	valid bool
	rgb   color.RGBA
	fg    []byte
	bg    []byte
	err   error
}

func (c Color) RGB() color.RGBA {
	return c.rgb
}

func (c Color) Err() error {
	return c.err
}

func (c Color) Valid() bool {
	return c.valid
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

func toAnsi(rgba color.RGBA, base byte) []byte {
	ansi := []byte{'\033', '[', base}
	ansi = append(ansi, "8;2;"...)
	ansi = append(ansi, coloritoa[rgba.R]...)
	ansi = append(ansi, semi)
	ansi = append(ansi, coloritoa[rgba.G]...)
	ansi = append(ansi, semi)
	ansi = append(ansi, coloritoa[rgba.B]...)
	ansi = append(ansi, semi)
	return ansi
}

// coloritoa converts a byte to a string. Used in constructing ANSI color codes.

var coloritoa [256][]byte

func init() {
	coloritoa = sync.OnceValue(func() [256][]byte {
		fn := func(t byte) []byte {
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
			return a[j:]
		}
		var res [256][]byte
		for i := range 256 {
			res[i] = fn(byte(i))
		}
		return res
	})()
}
