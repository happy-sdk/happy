// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
	"strconv"

	"github.com/happy-sdk/happy/pkg/cli/ansicolor"
)

type ConsoleHandler struct {
	slog.Handler
	src bool
	l   *log.Logger
}

func (h *ConsoleHandler) getLevelStr(lvl slog.Level) string {
	// l := Level(lvl)
	// if l == LevelQuiet {
	// 	return ""
	// }
	// var (
	// 	fg, bg Color
	// )

	// switch l {
	// case LevelSystemDebug:
	// 	fg, bg = fgBlue, 0
	// case LevelDebug:
	// 	fg, bg = fgWhite, 0
	// case LevelInfo:
	// 	fg, bg = fgCyan, 0
	// case LevelOk:
	// 	fg, bg = fgGreen, 0
	// case LevelNotice:
	// 	fg, bg = fgBlack, bgCyan
	// case LevelWarn:
	// 	fg, bg = fgYellow, 0
	// case LevelNotImplemented:
	// 	fg, bg = fgYellow, 0
	// case LevelDeprecated:
	// 	fg, bg = fgYellow, 0
	// case LevelError:
	// 	fg, bg = fgRed, 0
	// case LevelBUG:
	// 	fg, bg = fgWhite, bgRed
	// case LevelAlways:
	// 	return ""
	// default:
	// 	fg, bg = fgRed, 0
	// }
	// return colorize(fmt.Sprintf("%-8s", l.String()), fg, bg, 1, true)
	l := Level(lvl)
	if l == LevelQuiet {
		return ""
	}
	var (
		fg, bg ansicolor.Color
	)

	switch l {
	case LevelSystemDebug:
		fg, bg = ansicolor.FgBlue, 0
	case LevelDebug:
		fg, bg = ansicolor.FgWhite, 0
	case LevelInfo:
		fg, bg = ansicolor.FgCyan, 0
	case LevelOk:
		fg, bg = ansicolor.FgGreen, 0
	case LevelNotice:
		fg, bg = ansicolor.FgBlack, ansicolor.BgCyan
	case LevelWarn:
	case LevelNotImplemented, LevelDeprecated:
		fg, bg = ansicolor.FgBlack, ansicolor.BgYellow
	case LevelError:
		fg, bg = ansicolor.FgRed, 0
	case LevelBUG:
		fg, bg = ansicolor.FgWhite, ansicolor.BgRed
	case LevelAlways:
		return ""
	default:
		fg, bg = ansicolor.FgRed, 0
	}
	return ansicolor.TextPadded(fmt.Sprintf("%-8s", l.String()), fg, bg, 1)
}

func (h *ConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	lvlstr := h.getLevelStr(r.Level)
	lvl := Level(r.Level)

	var payload string

	if r.NumAttrs() > 0 {
		fields := make(map[string]any, r.NumAttrs())
		r.Attrs(func(a slog.Attr) bool {
			fields[a.Key] = a.Value.Any()
			return true
		})
		b, err := json.Marshal(fields)
		if err != nil {
			return err
		}
		payload = string(b)
	}

	timeStr := "\033[2m" + r.Time.Format("15:05:05.000") + "\033[0m"

	var msg string
	if lvl <= LevelDebug {
		msg = colorize(r.Message, fgWhite, 0, 0, false)
		payload = colorize(payload, fgCyan, 0, 2, false)
	} else {
		payload = colorize(payload, fgCyan, 0, 0, false)
		msg = colorize(r.Message, fgWhite, 0, 1, false)
	}

	if h.src && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			payload += colorize(f.File+":"+strconv.Itoa(f.Line), fgWhite, 0, 2, true)
		}
	}
	if lvl == LevelAlways {
		h.l.Println(msg, payload, clear)
	} else {
		h.l.Println(lvlstr, timeStr, msg, payload, clear)
	}

	return nil
}

func (h *ConsoleHandler) http(method, path string, status int, attrs ...slog.Attr) {
	var (
		fg, bg Color
	)
	switch {
	case status < 200:
		fg, bg = fgBlue, 0
	case status >= 200 && status < 300:
		fg, bg = fgGreen, 0
	case status >= 300 && status < 400:
		fg, bg = fgYellow, 0
	case status >= 400 && status < 500:
		fg, bg = fgRed, 0
	case status >= 500:
		fg, bg = fgBlack, bgRed
	default:
		fg, bg = fgRed, 0
	}
	statusStr := colorize(fmt.Sprintf("%-8s", method), fg, bg, 1, true)

	var payload string

	if len(attrs) > 0 {
		fields := make(map[string]any, len(attrs))
		for _, a := range attrs {
			fields[a.Key] = a.Value.Any()
		}
		b, err := json.Marshal(fields)
		if err != nil {
			return
		}
		payload = colorize(string(b), fgCyan, 0, 0, false)
	}

	if h.src {
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		fs := runtime.CallersFrames([]uintptr{pcs[0]})
		f, _ := fs.Next()
		if f.File != "" {
			payload += colorize(f.File+":"+strconv.Itoa(f.Line), fgWhite, 0, 2, true)
		}
	}

	h.l.Println(statusStr, path, payload, clear)
}

func Console(lvl Level) *DefaultLogger {
	l := &DefaultLogger{
		lvl: new(slog.LevelVar),
		ctx: context.Background(),
	}
	l.lvl.Set(slog.Level(lvl))

	addSource := true
	h := &ConsoleHandler{
		src: addSource,
		Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: l.lvl,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					level := a.Value.Any().(slog.Level)
					a.Value = slog.StringValue(Level(level).String())
				}
				return a
			},
			AddSource: addSource,
		}),
		l: log.New(os.Stdout, "", 0),
	}

	l.log = slog.New(h)
	return l
}

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
	fgBlack   Color = (iota << fgShift) | fgColor // 30, 90
	fgRed                                         // 31, 91
	fgGreen                                       // 32, 92
	fgYellow                                      // 33, 93
	fgBlue                                        // 34, 94
	fgMagenta                                     // 35, 95
	fgCyan                                        // 36, 96
	fgWhite                                       // 37, 97
	fgMask    = (0xff << fgShift) | fgColor
)

// Background colors
// [  0;   7] - 40-47
// [  8;  15] - 100-107
// [ 16; 231] - RGB
// [232; 255] - grayscale
const (
	bgBlack   Color = (iota << bgShift) | bgColor // 40, 100
	bgRed                                         // 41, 101
	bgGreen                                       // 42, 102
	bgYellow                                      // 43, 103
	bgBlue                                        // 44, 104
	bgMagenta                                     // 45, 105
	bgCyan                                        // 46, 106
	bgWhite                                       // 47, 107
	bgMask    = (0xff << bgShift) | bgColor
)

const (
	esc   = "\033["
	clear = esc + "0m"
)

func colorize(s string, fg, bg Color, format uint, padleft bool) (str string) {
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

	m := "m"
	if padleft {
		m = "m "
	}
	return esc + fmt.Sprint(format, ";", string(fgs), string(bgs), m, s) + clear
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
