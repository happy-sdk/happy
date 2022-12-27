// Package color adds coloring functionality for TTY output.
//nolint:gochecknoglobals
package color

import "fmt"

// // Foreground colors.
// const (
// 	Black Color = iota + 30
// 	Red
// 	Green
// 	Yellow
// 	Blue
// 	Magenta
// 	Cyan
// 	White
// )

var (
	// escape char.
	esc = byte(27)
	// ascii reset.
	asciiReset = []byte{esc, 91, 48, 109}

	Red      = []byte{esc, 91, 51, 49, 109}
	Green    = []byte{esc, 91, 51, 50, 109}
	White    = []byte{esc, 91, 49, 59, 51, 55, 109}
	Gray     = []byte{esc, 91, 48, 59, 51, 55, 109}
	GrayDark = []byte{esc, 91, 48, 59, 51, 48, 109}
	Yellow   = []byte{esc, 91, 51, 51, 109}
	Cyan     = []byte{esc, 91, 51, 54, 109}
	Blue     = []byte{esc, 91, 51, 52, 109}
	Black    = []byte{esc, 91, 51, 48, 109}
)

// Color represents a text color.
type Color uint8

// Text colorize.
func Text(color string, str string) string {
	var label []byte
	switch color {
	case "red":
		label = Red
	case "green":
		label = Green
	case "white":
		label = White
	case "gray":
		label = Gray
	case "darkgray":
		label = GrayDark
	case "yellow":
		label = Yellow
	case "cyan":
		label = Cyan
	case "blue":
		label = Blue
	case "black":
		label = Black
	default:
		label = White
	}

	return fmt.Sprintf("%s%s%s", label, str, asciiReset)
}
