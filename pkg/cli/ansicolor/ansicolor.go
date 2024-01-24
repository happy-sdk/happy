// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2017 The Happy Authors

// Package ansicolor provides utilities for colorizing text in console applications.
package ansicolor

import (
	"strconv"
)

type Color uint

const (
	fgColor Color = 1 << 14 // 14th bit
	bgColor Color = 1 << 15 // 15th bit

	fgShift = 16 // from 16th bit
	bgShift = 24 // from 24th bit

	fgRGB Color = 1 << 16 // New flag for RGB foreground
	bgRGB Color = 1 << 17 // New flag for RGB background
)

// Foreground color constants
const (
	FgBlack Color = (iota << fgShift) | fgColor
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
	fgMask = (0xff << fgShift) | fgColor
)

// Background color constants
const (
	BgBlack Color = (iota << bgShift) | bgColor
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
	bgMask = (0xff << bgShift) | bgColor
)

const (
	esc   = "\033["
	clear = esc + "0m"
)

// Text styles a given string with foreground and background colors.
// It applies ANSI color codes based on the provided foreground (fg) and background (bg) colors.
// The 'format' parameter can be used to specify additional formatting options, such as bold or underline, using ANSI codes.
// If fg and bg are both zero, the original text is returned without any styling.
//
// Example:
//
//	styledText := ansicolor.Text("Hello, world!", ansicolor.FgRed, ansicolor.BgBlack, 1)
//	fmt.Println(styledText) // Prints "Hello, world!" in red with a black background
//
// Parameters:
//
//	text   - The string to be styled.
//	fg     - The foreground color. Use the Fg* constants defined in the package.
//	bg     - The background color. Use the Bg* constants defined in the package.
//	format - Additional formatting options (e.g., 1 for bold, 4 for underline).
//
// Returns:
//
//	A string with ANSI color styling applied.
//
// Text styles a given string with foreground and background colors.
// It applies ANSI color codes based on the provided foreground (fg) and background (bg) colors.
// The 'format' parameter can be used to specify additional formatting options, such as bold or underline, using ANSI codes.
// If fg and bg are both zero, the original text is returned without any styling.
func Text(text string, fg, bg Color, format uint) string {
	fgCode := ""
	bgCode := ""
	formatCode := ""

	if format != 0 {
		formatCode = strconv.Itoa(int(format))
	}

	if fg+bg != 0 { // Check if either fg or bg color is set
		fgCode = buildColorCode(fg, fgMask, fgShift, '3', '9')
		bgCode = buildColorCode(bg, bgMask, bgShift, '4', '1')
	}

	if formatCode == "" && fgCode == "" && bgCode == "" {
		return text
	}

	combinedCode := ""
	if formatCode != "" {
		combinedCode = formatCode
		if fgCode != "" || bgCode != "" {
			combinedCode += ";" // Add semicolon only if there are color codes
		}
	}
	combinedCode += fgCode
	if fgCode != "" && bgCode != "" {
		combinedCode += ";" // Add semicolon between fg and bg codes
	}
	combinedCode += bgCode

	return esc + combinedCode + "m" + text + clear
}

// TextPadded styles a given string with foreground and background colors and pads it with a space on both sides.
// This function is similar to Text but adds a single space before and after the text for better readability in certain contexts.
// The 'format' parameter allows specifying additional formatting options like bold or underline using ANSI codes.
//
// Example:
//
//	paddedText := ansicolor.TextPadded("Hello, world!", ansicolor.FgGreen, ansicolor.BgWhite, 1)
//	fmt.Println(paddedText) // Prints " Hello, world! " with green text on a white background, in bold.
//
// Parameters:
//
//	text   - The string to be styled and padded.
//	fg     - The foreground color. Use the Fg* constants defined in the package.
//	bg     - The background color. Use the Bg* constants defined in the package.
//	format - Additional formatting options (e.g., 1 for bold, 4 for underline).
//
// Returns:
//
//	A string with ANSI color styling and padding applied.
func TextPadded(text string, fg, bg Color, format uint) string {
	const sp = " "
	return Text(sp+text+sp, fg, bg, format)
}

// FgRGB creates a foreground color from RGB values.
// This function enables users to specify custom colors using RGB (red, green, blue) values.
// Each color component (r, g, b) should be a byte value ranging from 0 to 255.
//
// The function combines these values into a single Color type, marked with an internal flag (fgRGB)
// indicating that the color is specified in RGB format.
//
// Example:
//
//	redColor := ansicolor.FgRGB(255, 0, 0) // Creates a bright red foreground color
//
// Parameters:
//
//	r - Red component of the color, a byte value from 0 to 255.
//	g - Green component of the color, a byte value from 0 to 255.
//	b - Blue component of the color, a byte value from 0 to 255.
//
// Returns:
//
//	A Color value representing the specified RGB color for the foreground.
func FgRGB(r, g, b byte) Color {
	return Color(r)<<8 | Color(g)<<16 | Color(b)<<24 | fgRGB
}

// BgRGB creates a background color from RGB values.
// Similar to FgRGB, this function allows specifying background colors using RGB values.
// It takes three byte values (r, g, b), each representing the intensity of the red, green, and blue components.
//
// The function encodes these values into a Color type, with an internal flag (bgRGB)
// to indicate RGB format for background color.
//
// Example:
//
//	blueBackground := ansicolor.BgRGB(0, 0, 255) // Creates a bright blue background color
//
// Parameters:
//
//	r - Red component of the color, a byte value from 0 to 255.
//	g - Green component of the color, a byte value from 0 to 255.
//	b - Blue component of the color, a byte value from 0 to 255.
//
// Returns:
//
//	A Color value representing the specified RGB color for the background.
func BgRGB(r, g, b byte) Color {
	return Color(r)<<8 | Color(g)<<16 | Color(b)<<24 | bgRGB
}

// buildColorCode constructs the ANSI color code for a given color.
// buildColorCode constructs the ANSI color code for a given color.
func buildColorCode(color, mask Color, shift uint, base, brightBase byte) string {
	if color == 0 {
		return ""
	}

	// Check if the color is a standard ANSI color
	if color&fgColor != 0 || color&bgColor != 0 {
		colorValue := (color & mask) >> shift
		switch {
		case colorValue >= 8 && colorValue <= 15: // Bright colors
			return string([]byte{brightBase, '0' + byte(colorValue-8)})
		case colorValue >= 0 && colorValue <= 7: // Normal colors
			return string([]byte{base, '0' + byte(colorValue)})
		}
	}

	// Check for RGB color flag
	if color&fgRGB != 0 || color&bgRGB != 0 {
		r, g, b := byte(color>>16), byte(color>>8), byte(color)
		colorType := "38" // Default to foreground
		if color&bgRGB != 0 {
			colorType = "48" // Use background
		}
		return "\033[" + colorType + ";2;" + strconv.Itoa(int(r)) + ";" + strconv.Itoa(int(g)) + ";" + strconv.Itoa(int(b)) + "m"
	}

	return ""
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
