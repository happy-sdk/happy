// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package console

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/tui/ansicolor"
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

type Adapter struct {
	ctx     context.Context
	opts    *logging.Options
	handler slog.Handler
}

func New(ctx context.Context, w io.Writer, opts *logging.Options, theme ansicolor.Theme) logging.Adapter {

	handler := NewHandler(ctx, w, opts, theme)
	return &Adapter{
		ctx:     ctx,
		opts:    opts,
		handler: handler,
	}
}

func (ta *Adapter) Handler() slog.Handler {
	return ta.handler
}

func (ta *Adapter) Options() *logging.Options {
	return ta.opts
}

func (ta *Adapter) Context() context.Context {
	return ta.ctx
}

func (ta *Adapter) Dispose() error {
	return nil
}

type ConsoleHandler struct {
	slog.Handler
	styles consoleTheme
	src    bool
	l      *log.Logger
	tsfmt  string
	nots   bool
}

func NewHandler(ctx context.Context, w io.Writer, opts *logging.Options, theme ansicolor.Theme) *ConsoleHandler {
	if opts.LevelVar == nil {
		opts.LevelVar = new(slog.LevelVar)
	}
	replaceAttr := opts.ReplaceAttr
	tsfmt := "15:04:05.000"
	if opts.TimestampFormat != "" {
		tsfmt = opts.TimestampFormat
	}
	h := &ConsoleHandler{
		styles: consoleTheme{
			attrs:      ansicolor.Style{FG: theme.Secondary},
			muted:      ansicolor.Style{FG: theme.Muted},
			sysdebug:   ansicolor.Style{FG: ansicolor.RGB(96, 125, 139)},
			debug:      ansicolor.Style{FG: theme.Debug},
			info:       ansicolor.Style{FG: theme.Info},
			notice:     ansicolor.Style{FG: theme.Notice},
			success:    ansicolor.Style{FG: theme.Success},
			warn:       ansicolor.Style{FG: theme.Warning},
			notimpl:    ansicolor.Style{FG: theme.NotImplemented},
			deprecated: ansicolor.Style{FG: theme.Deprecated},
			error:      ansicolor.Style{FG: theme.Error},
			bug:        ansicolor.Style{BG: theme.BUG},
			light:      ansicolor.Style{FG: theme.Light},
		},
		src: opts.AddSource,
		Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: opts.LevelVar,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					level := a.Value.Any().(slog.Level)
					a.Value = slog.StringValue(logging.Level(level).String())
				}
				if replaceAttr != nil {
					a = replaceAttr(groups, a)
				}
				return a
			},
			AddSource: opts.AddSource,
		}),
		l:     log.New(os.Stderr, "", 0),
		tsfmt: tsfmt,
		nots:  opts.NoTimestamp,
	}

	return h
}

func (h *ConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	lvlstr := h.getLevelStr(r.Level)
	lvl := logging.Level(r.Level)

	var (
		msg,
		payload string
	)

	var src string

	if r.NumAttrs() > 0 {
		fields := make(map[string]any, r.NumAttrs())
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == slog.SourceKey && a.Value.Kind() == slog.KindString {
				src = a.Value.String()
				return true
			}
			fields[a.Key] = a.Value.Any()
			return true
		})

		if len(fields) > 0 {
			b, err := json.Marshal(fields)
			if err != nil {
				return err
			}
			if lvl >= logging.LevelDebug {
				payload = h.styles.attrs.String(string(b))
			} else {
				payload = string(b)
			}
		}
	}

	var timeStr string
	if !h.nots {
		timeStr = h.styles.muted.String(r.Time.Format(h.tsfmt))
	}

	if lvl < logging.LevelDebug {
		msg = h.styles.sysdebug.String(r.Message)
	} else if lvl == logging.LevelDebug {
		msg = h.styles.debug.String(r.Message)
	} else {
		msg = h.styles.light.String(r.Message)
	}

	if h.src {
		if src != "" {
			payload += " " + h.styles.muted.String(src)
		} else if r.PC != 0 {
			fs := runtime.CallersFrames([]uintptr{r.PC})
			f, _ := fs.Next()
			if f.File != "" {
				payload += " " + h.styles.muted.String(f.File+":"+strconv.Itoa(f.Line))
			}
		}
	}

	if lvl == logging.LevelAlways {
		h.l.Println(msg, payload)
	} else {
		h.l.Println(lvlstr, timeStr, msg, payload)
	}

	return nil
}

func (h *ConsoleHandler) getLevelStr(lvl slog.Level) string {
	l := logging.Level(lvl)
	if l == logging.LevelQuiet {
		return ""
	}

	var c ansicolor.Style
	switch l {
	case logging.LevelDebug:
		c = h.styles.debug
	case logging.LevelInfo:
		c = h.styles.info
	case logging.LevelOk:
		c = h.styles.success
	case logging.LevelNotice:
		c = h.styles.notice
	case logging.LevelWarn:
		c = h.styles.warn
	case logging.LevelNotImplemented:
		c = h.styles.notimpl
	case logging.LevelDeprecated:
		c = h.styles.deprecated
	case logging.LevelError:
		c = h.styles.error
	case logging.LevelAlways:
		return ""
	case logging.LevelBUG:
		c = h.styles.bug
	default:
		c = h.styles.sysdebug
	}
	return c.String(fmt.Sprintf(" %-11s", l.String()))
}

//nolint:unused
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
