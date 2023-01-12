// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package hlog

import (
	"io"

	"golang.org/x/exp/slog"
)

type Config struct {
	Options slog.HandlerOptions
	JSON    bool
	Colors  bool
	Secrets []string
}

// NewJSONHandler creates a JSONHandler that writes to w,
// using the default options.
func (cnf Config) NewHandler(w io.Writer) slog.Handler {
	if cnf.JSON {
		opts := slog.HandlerOptions{
			Level:     cnf.Options.Level,
			AddSource: cnf.Options.AddSource,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if rep := cnf.Options.ReplaceAttr; rep != nil {
					a = cnf.Options.ReplaceAttr(groups, a)
				}

				for _, secret := range cnf.Secrets {
					if a.Key == secret {
						a.Value = slog.StringValue("*****")
					}
				}
				if a.Key == slog.LevelKey {
					lvl, ok := a.Value.Any().(slog.Level)
					if ok {
						return slog.String("level", Level(lvl).String())
					}
				}
				return a
			},
		}
		return opts.NewJSONHandler(w)

	}
	h := &Handler{
		w:      w,
		colors: cnf.Colors,
	}
	h.opts = slog.HandlerOptions{
		Level:     cnf.Options.Level,
		AddSource: cnf.Options.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if rep := cnf.Options.ReplaceAttr; rep != nil {
				a = cnf.Options.ReplaceAttr(groups, a)
			}

			for _, secret := range cnf.Secrets {
				if a.Key == secret {
					a.Value = slog.StringValue("*****")
				}
			}
			return a
		},
	}

	return h
}
