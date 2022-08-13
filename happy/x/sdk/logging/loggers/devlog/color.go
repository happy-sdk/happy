// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build (linux && !android) || freebsd || windows || openbsd || darwin || !js

package devlog

import (
	"fmt"
)

const (
	fgColor color = 1 << 14 // 14th bit
	bgColor color = 1 << 15 // 15th bit

	fgShift = 16 // from 16th bit
	bgShift = 24 // from 24th bit
)

// Foreground colors
// [  0;   7] - 30-37
// [  8;  15] - 90-97
// [ 16; 231] - RGB
// [232; 255] - grayscale
const (
	blackFg   color = (iota << fgShift) | fgColor // 30, 90
	redFg                                         // 31, 91
	greenFg                                       // 32, 92
	yellowFg                                      // 33, 93
	blueFg                                        // 34, 94
	magentaFg                                     // 35, 95
	cyanFg                                        // 36, 96
	whiteFg                                       // 37, 97
	fgMask    = (0xff << fgShift) | fgColor
)

// Background colors
// [  0;   7] - 40-47
// [  8;  15] - 100-107
// [ 16; 231] - RGB
// [232; 255] - grayscale
const (
	blackBg   color = (iota << bgShift) | bgColor // 40, 100
	redBg                                         // 41, 101
	greenBg                                       // 42, 102
	yellowBg                                      // 43, 103
	blueBg                                        // 44, 104
	magentaBg                                     // 45, 105
	cyanBg                                        // 46, 106
	whiteBg                                       // 47, 107
	bgMask    = (0xff << bgShift) | bgColor
)

const (
	esc   = "\033["
	clear = esc + "0m"
)

func colorize(s string, fg, bg color, format uint) (str string) {
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
