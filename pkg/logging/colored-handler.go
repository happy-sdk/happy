// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"context"
	"fmt"
	"io"
	"strings"
)

func newColoredHandler(l LevelIface, flags RecordFlag, w io.Writer) *ColoredHandler {
	return &ColoredHandler{
		level: l,
		w:     w,
		flags: flags,
	}
}

type ColoredHandler struct {
	flags RecordFlag
	level LevelIface
	w     io.Writer

	groups []string // all groups started from WithGroup
	attrs  []Attr
}

func (h *ColoredHandler) Enabled(ctx context.Context, level Level) bool {
	return true
}

func (h *ColoredHandler) Handle(ctx context.Context, r Record) error {
	var (
		line colorLine
	)

	if r.Level.Kind() != AttrOmittedKind {
		line.writeLevel(r.level)
	}

	if r.TimeString.Kind() != AttrOmittedKind {
		line.writeTimestamp(r.TimeString.Value.String())
	}
	if r.Message.Kind() != AttrOmittedKind {
		line.writeString(r.Message.Value.String())
	}

	if r.Error.Kind() != AttrOmittedKind {
		if err := line.writeError(r.Error); err != nil {
			return err
		}
	}

	groups := strings.Join(h.groups, ".")
	for _, attr := range h.attrs {
		if attr.Kind() != AttrOmittedKind {
			if err := line.writeColorAttr(groups, attr); err != nil {
				return err
			}
		}
	}

	if r.Data.Kind() != AttrOmittedKind {
		if err := line.writeColorAttr(groups, r.Data); err != nil {
			return err
		}
	}

	if r.Source.Kind() != AttrOmittedKind {
		if err := line.writeColorAttr("", r.Source); err != nil {
			return err
		}
	}

	if err := line.writeRune('\n'); err != nil {
		return err
	}

	if _, err := io.WriteString(h.w, line.String()); err != nil {
		return err
	}

	return nil
}

func (h *ColoredHandler) WithAttrs(attrs ...Attr) Handler {
	h2 := newColoredHandler(h.level, h.flags, h.w)
	h2.groups = append(h2.groups, h.groups...)
	h2.attrs = append(h.attrs, attrs...)
	return h2
}

func (h *ColoredHandler) WithGroup(group string) Handler {
	h2 := newColoredHandler(h.level, h.flags, h.w)
	h2.groups = append(h.groups, group)
	h2.attrs = append(h2.attrs, h.attrs...)
	return h2
}

func (h *ColoredHandler) Dispose() error {
	return nil
}

func (h *ColoredHandler) Flags() RecordFlag {
	return h.flags
}

type colorLine struct {
	logLine
}

func (l *colorLine) writeError(attr Attr) error {
	l.buf.WriteString(colorize(attr.Key, fgRed, 0, 1))
	l.buf.WriteRune('=')
	l.buf.Write([]byte{'\033', '[', '0', 'm'})
	n, err := l.buf.WriteString(attr.Value.String())
	l.len += n
	return err
}

func (l *colorLine) writeLevel(lvl Level) error {
	return l.writeString(lvl.colorLabel())
}

func (l *colorLine) writeTimestamp(ts string) error {
	l.buf.Write([]byte{'\033', '[', '2', 'm'})
	l.buf.WriteString(ts)
	l.buf.Write([]byte{'\033', '[', '0', 'm'})
	return nil
}

func (l *colorLine) writeKey(key string) error {
	l.buf.Write([]byte{'\033', '[', '2', 'm'})
	l.buf.WriteString(" " + key + "=")
	l.buf.Write([]byte{'\033', '[', '0', 'm'})

	return nil
}

func (l *colorLine) writeColorAttr(groups string, attr Attr) error {
	switch attr.Kind() {
	case AttrSingleKind:
		var key string
		if groups != "" {
			key = groups + "."
		}
		key += attr.Key

		l.writeKey(key)
		entry := attr.Value.String()
		l.buf.WriteRune('"')
		if _, err := l.buf.WriteString(entry); err != nil {
			return err
		}
		l.buf.WriteRune('"')
	case AttrArrayKind:
		for i, a := range attr.Value.Array() {
			if err := l.writeArrayValue(groups, i, a); err != nil {
				return err
			}
		}
	case AttrObjectKind:
		for _, el := range attr.Value.Object() {
			key := el.Key
			if groups != "" {
				key = groups + "." + key
			}
			if err := l.writeColoredAttrValue(key, el.Value); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *colorLine) writeColoredAttrValue(key string, value AttrValue) error {
	if key == "" {
		key = "data"
	}
	switch value.kind {
	case AttrObjectKind:
		for _, el := range value.Object() {
			if err := l.writeColoredAttrValue(key+"."+el.Key, el.Value); err != nil {
				return err
			}
		}
		return nil
	}
	l.writeKey(key)

	l.buf.WriteRune('"')
	n, err := l.buf.WriteString(fmt.Sprint(value.value))
	l.buf.WriteRune('"')
	l.len += n
	return err
}

func (l Level) colorLabel() string {
	var (
		fg, bg Color
	)
	switch l {
	case LevelSystemDebug:
		fg, bg = fgBlue, 0
	case LevelDebug:
		fg, bg = fgWhite, 0
	case LevelInfo:
		fg, bg = fgCyan, 0
	case LevelTask:
		fg, bg = fgBlue, bgBlack
	case LevelOk:
		fg, bg = fgGreen, 0
	case LevelNotice:
		fg, bg = fgBlack, bgCyan
	case LevelWarn:
		fg, bg = fgYellow, 0
	case LevelNotImplemented:
		fg, bg = fgYellow, 0
	case LevelDeprecated:
		fg, bg = fgYellow, 0
	case LevelIssue:
		fg, bg = fgBlack, bgYellow
	case LevelError:
		fg, bg = fgRed, bgBlack
	case LevelBUG:
		fg, bg = fgRed, bgBlack
	case LevelAlways:
		fg, bg = fgWhite, 0
	default:
		fg, bg = fgRed, bgBlack
	}
	return colorize(fmt.Sprintf("%-8s", l.String()), fg, bg, 1)
}

func colorize(s string, fg, bg Color, format uint) (str string) {
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

	return esc + fmt.Sprint(format, ";", string(fgs), string(bgs), "m ", s) + clear

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
