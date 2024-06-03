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
	"time"

	"github.com/happy-sdk/happy/pkg/cli/ansicolor"
)

type consoleTheme struct {
	attrs      ansicolor.Style
	muted      ansicolor.Style
	sysdebug   ansicolor.Style
	debug      ansicolor.Style
	info       ansicolor.Style
	notice     ansicolor.Style
	success    ansicolor.Style
	warn       ansicolor.Style
	notimpl    ansicolor.Style
	deprecated ansicolor.Style
	error      ansicolor.Style
	bug        ansicolor.Style
	light      ansicolor.Style
}

type ConsoleOptions struct {
	Level           Level
	Theme           ansicolor.Theme
	ReplaceAttr     func(groups []string, a slog.Attr) slog.Attr
	AddSource       bool
	TimeLocation    *time.Location
	TimestampFormat string
}

func ConsoleDefaultOptions() ConsoleOptions {
	return ConsoleOptions{
		Level:       LevelInfo,
		Theme:       ansicolor.New(),
		ReplaceAttr: nil,
		AddSource:   true,
	}
}

func Console(opts ConsoleOptions) *DefaultLogger {
	var tsloc *time.Location
	if opts.TimeLocation != nil {
		tsloc = opts.TimeLocation
	} else {
		tsloc = time.Local
	}

	l := &DefaultLogger{
		lvl:   new(slog.LevelVar),
		ctx:   context.Background(),
		tsloc: tsloc,
	}
	l.lvl.Set(slog.Level(opts.Level))

	replaceAttr := opts.ReplaceAttr

	tsfmt := "15:04:05.000"
	if opts.TimestampFormat != "" {
		tsfmt = opts.TimestampFormat
	}
	h := &ConsoleHandler{
		styles: consoleTheme{
			attrs:      ansicolor.Style{FG: opts.Theme.Secondary},
			muted:      ansicolor.Style{FG: opts.Theme.Muted},
			sysdebug:   ansicolor.Style{FG: ansicolor.RGB(96, 125, 139)},
			debug:      ansicolor.Style{FG: opts.Theme.Debug},
			info:       ansicolor.Style{FG: opts.Theme.Info},
			notice:     ansicolor.Style{FG: opts.Theme.Notice},
			success:    ansicolor.Style{FG: opts.Theme.Success},
			warn:       ansicolor.Style{FG: opts.Theme.Warning},
			notimpl:    ansicolor.Style{FG: opts.Theme.NotImplemented},
			deprecated: ansicolor.Style{FG: opts.Theme.Deprecated},
			error:      ansicolor.Style{FG: opts.Theme.Error},
			bug:        ansicolor.Style{BG: opts.Theme.BUG},
			light:      ansicolor.Style{FG: opts.Theme.Light},
		},
		src: opts.AddSource,
		Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: l.lvl,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					level := a.Value.Any().(slog.Level)
					a.Value = slog.StringValue(Level(level).String())
				}
				if replaceAttr != nil {
					a = replaceAttr(groups, a)
				}
				return a
			},
			AddSource: opts.AddSource,
		}),
		l:     log.New(os.Stdout, "", 0),
		tsfmt: tsfmt,
	}

	l.log = slog.New(h)
	return l
}

type ConsoleHandler struct {
	slog.Handler
	styles consoleTheme
	src    bool
	l      *log.Logger
	tsfmt  string
}

func (h *ConsoleHandler) getLevelStr(lvl slog.Level) string {
	l := Level(lvl)
	if l == LevelQuiet {
		return ""
	}

	var c ansicolor.Style
	switch l {
	case levelHappy, levelInit:
		c = h.styles.sysdebug
	case LevelDebug:
		c = h.styles.debug
	case LevelInfo:
		c = h.styles.info
	case LevelOk:
		c = h.styles.success
	case LevelNotice:
		c = h.styles.notice
	case LevelWarn:
		c = h.styles.warn
	case LevelNotImplemented:
		c = h.styles.notimpl
	case LevelDeprecated:
		c = h.styles.deprecated
	case LevelError:
		c = h.styles.error
	case LevelAlways:
		return ""
		// case LevelBUG:
	// 	c = h.styles.bug
	default:
		c = h.styles.bug
	}
	return c.String(fmt.Sprintf(" %-11s", l.String()))
}

func (h *ConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	lvlstr := h.getLevelStr(r.Level)
	lvl := Level(r.Level)

	var (
		msg,
		payload string
	)

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
		if lvl >= LevelDebug {
			payload = h.styles.attrs.String(string(b))
		} else {
			payload = string(b)
		}
	}

	timeStr := h.styles.muted.String(r.Time.Format(h.tsfmt))

	if lvl < LevelDebug {
		msg = h.styles.sysdebug.String(r.Message)
	} else if lvl == LevelDebug {
		msg = h.styles.debug.String(r.Message)
	} else {
		msg = h.styles.light.String(r.Message)
	}

	if h.src && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			payload += " " + h.styles.muted.String(f.File+":"+strconv.Itoa(f.Line))
		}
	}
	if lvl == LevelAlways {
		h.l.Println(msg, payload)
	} else {
		h.l.Println(lvlstr, timeStr, msg, payload)
	}

	return nil
}

func (h *ConsoleHandler) http(status int, method, p string, attrs ...slog.Attr) {

	var (
		state,
		payload string
	)

	state = fmt.Sprintf("[%-6s %d]", method, status)
	switch {
	case status < 200:
		state = h.styles.info.String(state)
	case status < 300:
		state = h.styles.success.String(state)
	case status < 400:
		state = h.styles.warn.String(state)
	case status < 500:
		state = h.styles.error.String(state)
	default:
		state = h.styles.bug.String(state)
	}
	if len(attrs) > 0 {
		fields := make(map[string]any, len(attrs))
		for _, a := range attrs {
			fields[a.Key] = a.Value.Any()
		}
		b, err := json.Marshal(fields)
		if err != nil {
			return
		}
		payload = h.styles.attrs.String(string(b))
	}
	if h.src {
		var pcs [1]uintptr
		runtime.Callers(4, pcs[:])
		fs := runtime.CallersFrames([]uintptr{pcs[0]})
		f, _ := fs.Next()
		if f.File != "" {
			payload += " " + h.styles.muted.String(f.File+":"+strconv.Itoa(f.Line))
		}
	}

	timeStr := h.styles.muted.String(time.Now().Format(h.tsfmt))

	h.l.Println(state, timeStr, p, payload)
}
