// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"context"
	"io"
	"log/slog"
)

type Adapter interface {
	Handler() slog.Handler
	Options() *Options
	Context() context.Context
	Dispose() error
}

type TextAdapter struct {
	ctx     context.Context
	opts    *Options
	handler slog.Handler
}

func NewTextAdapter(ctx context.Context, w io.Writer, opts *Options) Adapter {
	if opts == nil {
		opts = DefaultOptions()
	}
	if opts.LevelVar == nil {
		opts.LevelVar = new(slog.LevelVar)
	}

	replaceAttr := opts.ReplaceAttr
	tsfmt := "15:04:05.000"
	if opts.TimestampFormat != "" {
		tsfmt = opts.TimestampFormat
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: opts.LevelVar,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				a.Value = slog.StringValue(Level(level).String())
				return a
			}
			if a.Key == slog.TimeKey {
				// Format the timestamp however you want
				return slog.String(slog.TimeKey, a.Value.Time().Format(tsfmt))
			}
			if replaceAttr != nil {
				a = replaceAttr(groups, a)
			}
			return a
		},
		AddSource: opts.AddSource,
	})
	return &TextAdapter{
		ctx:     ctx,
		opts:    opts,
		handler: handler,
	}
}

func (ta *TextAdapter) Handler() slog.Handler {
	return ta.handler
}

func (ta *TextAdapter) Options() *Options {
	return ta.opts
}

func (ta *TextAdapter) Context() context.Context {
	return ta.ctx
}

func (ta *TextAdapter) Dispose() error {
	return nil
}
