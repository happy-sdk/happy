// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happylog

import (
	"fmt"

	"golang.org/x/exp/slog"
)

type Color uint

const (
	fgColor Color = 1 << 14 // 14th bit
	bgColor Color = 1 << 15 // 15th bit

	fgShift = 16 // from 16th bit
	bgShift = 24 // from 24th bit
)

// Foreground colors
// [  0;   7] - 30-37
// [  8;  15] - 90-97
// [ 16; 231] - RGB
// [232; 255] - grayscale
const (
	FgBlack   Color = (iota << fgShift) | fgColor // 30, 90
	FgRed                                         // 31, 91
	FgGreen                                       // 32, 92
	FgYellow                                      // 33, 93
	FgBlue                                        // 34, 94
	FgMagenta                                     // 35, 95
	FgCyan                                        // 36, 96
	FgWhite                                       // 37, 97
	fgMask    = (0xff << fgShift) | fgColor
)

// Background colors
// [  0;   7] - 40-47
// [  8;  15] - 100-107
// [ 16; 231] - RGB
// [232; 255] - grayscale
const (
	BgBlack   Color = (iota << bgShift) | bgColor // 40, 100
	BgRed                                         // 41, 101
	BgGreen                                       // 42, 102
	BgYellow                                      // 43, 103
	BgBlue                                        // 44, 104
	BgMagenta                                     // 45, 105
	BgCyan                                        // 46, 106
	BgWhite                                       // 47, 107
	bgMask    = (0xff << bgShift) | bgColor
)

const (
	esc   = "\033["
	clear = esc + "0m"
)

func ColorLevel(lvlArg slog.Attr) slog.Attr {
	// lvlArg.Value.Int64()

	return slog.Attr{
		Value: slog.StringValue(Colorize(lvlArg.Value.String(), FgGreen, BgBlack, 1)),
	}
}

func Colorize(s string, fg, bg Color, format uint) (str string) {
	if fg+bg == 0 {
		return s
	}
	var fgs, bgs []byte
	if fg > 0 {
		// 0- 7 :  30-37
		// 8-15 :  90-97
		// > 15 : 38;5;val

		switch fgc := (fg & fgMask) >> fgShift; {
		case fgc <= 7:
			// '3' and the value itself
			fgs = append(fgs, '3', '0'+byte(fgc))
		case fg <= 15:
			// '9' and the value itself
			fgs = append(fgs, '9', '0'+byte(fgc&^0x08)) // clear bright flag
		default:
			fgs = append(fgs, '3', '8', ';', '5', ';')
			fgs = append(fgs, coloritoa(byte(fgc))...)
		}
	}

	if bg > 0 {
		if fg > 0 {
			bgs = append(bgs, ';')
		}
		// 0- 7 :  40- 47
		// 8-15 : 100-107
		// > 15 : 48;5;val
		switch bgc := (bg & bgMask) >> bgShift; {
		case fg <= 7:
			// '3' and the value itself
			bgs = append(bgs, '4', '0'+byte(bgc))
		case fg <= 15:
			// '1', '0' and the value itself
			bgs = append(bgs, '1', '0', '0'+byte(bgc&^0x08)) // clear bright flag
		default:
			bgs = append(bgs, '4', '8', ';', '5', ';')
			bgs = append(bgs, coloritoa(byte(bgc))...)
		}
	}

	return esc + fmt.Sprint(format, ";", string(fgs), string(bgs), "m ", s) + clear + " "

}

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
