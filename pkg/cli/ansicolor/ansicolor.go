// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2017 The Happy Authors

// Package ansicolor provides utilities for colorizing text in console applications.
package ansicolor

type Color uint32

const (
	fgBit Color = 1 << 24
	bgBit Color = 1 << 25

	fgShift = 16
	bgShift = 8
)

// Foreground color constants
const (
	FgBlack Color = iota | fgBit
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

// Background color constants
const (
	BgBlack Color = iota | bgBit
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
)

type Flag uint8

const (
	Reset     Flag = 0
	Bold      Flag = 1 << iota // 1
	Dim                        // 2
	Italic                     // 4
	Underline                  // 8
	Blink                      // 16
	Reverse                    // 32
	Hidden                     // 64
)

const (
	esc   = "\033["
	clear = esc + "0m"
)

// Text styles a given string with foreground and background colors.
func Text(text string, fg, bg Color, format Flag) string {
	style := buildStyleCode(fg, bg, format)
	if style == "" {
		return text
	}
	return style + text + clear
}

// TextPadded styles and pads the text with a space on both sides.
func TextPadded(text string, fg, bg Color, format Flag) string {
	return Text(" "+text+" ", fg, bg, format)
}

// FgRGB creates a foreground color from RGB values.
func FgRGB(r, g, b byte) Color {
	return Color(r)<<fgShift | Color(g)<<bgShift | Color(b) | fgBit
}

// BgRGB creates a background color from RGB values.
func BgRGB(r, g, b byte) Color {
	return Color(r)<<fgShift | Color(g)<<bgShift | Color(b) | bgBit
}

func buildStyleCode(fg, bg Color, format Flag) string {
	var parts []string

	if format != 0 {
		parts = append(parts, coloritoa(byte(format)))
	}

	if fg&fgBit != 0 {
		parts = append(parts, buildColorCode(fg, '3'))
	}

	if bg&bgBit != 0 {
		parts = append(parts, buildColorCode(bg, '4'))
	}

	if len(parts) == 0 {
		return ""
	}

	return esc + stringJoin(parts, ";") + "m"
}

func buildColorCode(color Color, baseChar byte) string {
	if color&0xFF0000 != 0 { // RGB color
		r := (color >> fgShift) & 0xFF
		g := (color >> bgShift) & 0xFF
		b := color & 0xFF
		return string(baseChar) + "8;2;" + coloritoa(byte(r)) + ";" + coloritoa(byte(g)) + ";" + coloritoa(byte(b))
	}

	// Standard color
	colorValue := color & 0xFF
	return string(baseChar) + coloritoa(byte(colorValue))
}

func stringJoin(elements []string, delimiter string) string {
	switch len(elements) {
	case 0:
		return ""
	case 1:
		return elements[0]
	default:
		return elements[0] + delimiter + stringJoin(elements[1:], delimiter)
	}
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
