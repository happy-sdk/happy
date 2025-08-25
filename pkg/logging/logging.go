// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

var (
	Error                    = errors.New("logger")
	ErrLoggerAlreadyDisposed = fmt.Errorf("%w already disposed", Error)
)

type Logger interface {
	Debug(msg string, attrs ...slog.Attr)
	Info(msg string, attrs ...slog.Attr)
	Ok(msg string, attrs ...slog.Attr)
	Notice(msg string, attrs ...slog.Attr)
	NotImplemented(msg string, attrs ...slog.Attr)
	Warn(msg string, attrs ...slog.Attr)
	Deprecated(msg string, attrs ...slog.Attr)
	Error(msg string, attrs ...slog.Attr)
	Errors(err error, attrs ...slog.Attr)
	BUG(msg string, attrs ...slog.Attr)
	Println(msg string, attrs ...slog.Attr)
	Printf(format string, v ...any)
	HTTP(status int, method, path string, attrs ...slog.Attr)
	Handle(r slog.Record) error

	Enabled(lvl Level) bool
	Level() Level
	SetLevel(lvl Level)

	LogDepth(depth int, lvl Level, msg string, attrs ...slog.Attr)

	Logger() *slog.Logger

	ConsumeQueue(queue *QueueLogger) error
	Dispose() error

	AttachAdapter(adapter Adapter) error
	SetAdapter(adapter Adapter) error
	Options() (*Options, error)
}

type Options struct {
	LevelVar        *slog.LevelVar
	Level           Level
	ReplaceAttr     func(groups []string, a slog.Attr) slog.Attr
	AddSource       bool
	TimeLocation    *time.Location
	TimestampFormat string
	NoTimestamp     bool
	SetSlogOutput   bool
}

func DefaultOptions() *Options {
	return &Options{
		LevelVar:        new(slog.LevelVar),
		Level:           LevelInfo,
		ReplaceAttr:     nil,
		AddSource:       true,
		TimeLocation:    time.Local,
		TimestampFormat: "15:04:05.000",
		NoTimestamp:     false,
		SetSlogOutput:   true,
	}
}

func NewTextLogger(ctx context.Context, w io.Writer, opts *Options) *DefaultLogger {
	return New(NewTextAdapter(ctx, w, opts))
}

type httpHandler interface {
	http(status int, method, path string, attrs ...slog.Attr)
}
