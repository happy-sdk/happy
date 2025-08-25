// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
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

func NewTextAdapter(ctx context.Context, w io.Writer, opts *Options) *TextAdapter {
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

type JSONAdapter struct {
	ctx     context.Context
	opts    *Options
	handler slog.Handler
}

func NewJSONAdapter(ctx context.Context, w io.Writer, opts *Options) *JSONAdapter {
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

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
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
	return &JSONAdapter{
		ctx:     ctx,
		opts:    opts,
		handler: handler,
	}
}

func (ja *JSONAdapter) Handler() slog.Handler {
	return ja.handler
}

func (ja *JSONAdapter) Options() *Options {
	return ja.opts
}

func (ja *JSONAdapter) Context() context.Context {
	return ja.ctx
}

func (ja *JSONAdapter) Dispose() error {
	return nil
}

type CombinedAdapter struct {
	ctx      context.Context
	opts     *Options
	adapters []Adapter
	handler  slog.Handler
}

func NewCombinedAdapter(aa ...Adapter) *CombinedAdapter {
	if len(aa) == 0 {
		return nil
	}
	var handlers []slog.Handler
	for _, h := range aa {
		handlers = append(handlers, h.Handler())
	}
	handler := NewCombinedHandler(handlers...)

	return &CombinedAdapter{
		adapters: aa,
		ctx:      aa[0].Context(),
		opts:     aa[0].Options(),
		handler:  handler,
	}
}

func (ca *CombinedAdapter) Handler() slog.Handler {
	return ca.handler
}

func (ca *CombinedAdapter) Options() *Options {
	return ca.opts
}

func (ca *CombinedAdapter) Context() context.Context {
	return ca.ctx
}

func (ca *CombinedAdapter) Dispose() (err error) {
	for _, adapter := range ca.adapters {
		if e := adapter.Dispose(); e != nil {
			err = errors.Join(err, e)
		}
	}
	return
}

type CombinedHandler struct {
	mu       sync.Mutex
	handlers []slog.Handler
}

func NewCombinedHandler(handlers ...slog.Handler) *CombinedHandler {
	if len(handlers) == 0 {
		return nil
	}
	return &CombinedHandler{
		handlers: handlers,
	}
}

func (ch *CombinedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range ch.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (ch *CombinedHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range ch.handlers {
		if err := h.Handle(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (ch *CombinedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	newHandlers := make([]slog.Handler, len(ch.handlers))
	for i, h := range ch.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return NewCombinedHandler(newHandlers...)
}

func (ch *CombinedHandler) WithGroup(name string) slog.Handler {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	newHandlers := make([]slog.Handler, len(ch.handlers))
	for i, h := range ch.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return NewCombinedHandler(newHandlers...)
}
