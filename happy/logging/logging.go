// Copyright 2023 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"context"
	"errors"
)

var (
	ErrAttrReplace      = errors.New("replace attr error")
	ErrUnknownLevelName = errors.New("unknown level name")
)

// Handler handles log records produced by a Logger.
//
// A typical handler may print log records to standard error,
// or write them to a file or database, or perhaps augment them
// with additional attributes and pass them on to another handler.
//
// Any of the Handler's methods may be called concurrently with itself or with other methods.
// It is the responsibility of the Handler to manage this concurrency.
//
// Users of the slog package should not invoke Handler methods directly.
// They should use the methods of Logger instead.
type Handler interface {
	// Enabled reports whether the handler handles records at the given level.
	// The handler ignores records whose level is lower.
	// It is called early, before any arguments are processed,
	// to save effort if the log event should be discarded.
	// The Logger's context is passed so Enabled can use its values
	// to make a decision. The context may be nil.
	Enabled(ctx context.Context, level Level) bool

	// Handle handles the Record.
	// It will only be called if Enabled returns true.
	//
	// The first argument is the context of the Logger that created the Record,
	// which may be nil.
	// It is present solely to provide Handlers access to the context's values.
	// Canceling the context should not affect record processing.
	// (Among other things, log messages may be necessary to debug a
	// cancellation-related problem.)
	//
	// Handle methods that produce output should observe the following rules:
	//   - If r.Time is the zero time, ignore the time.
	//   - If an Attr's key is the empty string and the value is not a group,
	//     ignore the Attr.
	//   - If a group's key is empty, inline the group's Attrs.
	//   - If a group has no Attrs (even if it has a non-empty key),
	//     ignore it.
	Handle(ctx context.Context, r Record) error

	// WithAttrs returns a new Handler whose attributes consist of
	// both the receiver's attributes and the arguments.
	// The Handler owns the slice: it may retain, modify or discard it.
	// [Logger.With] will resolve the Attrs.
	WithAttrs(attrs ...Attr) Handler

	// WithGroup returns a new Handler with the given group appended to
	// the receiver's existing groups.
	// The keys of all subsequent attributes, whether added by With or in a
	// Record, should be qualified by the sequence of group names.
	//
	// How this qualification happens is up to the Handler, so long as
	// this Handler's attribute keys differ from those of another Handler
	// with a different sequence of group names.
	//
	// A Handler should treat WithGroup as starting a Group of Attrs that ends
	// at the end of the log event. That is,
	//
	//     logger.WithGroup("s").LogAttrs(level, msg, slog.Int("a", 1), slog.Int("b", 2))
	//
	// should behave like
	//
	//     logger.LogAttrs(level, msg, slog.Group("s", slog.Int("a", 1), slog.Int("b", 2)))
	//
	// If the name is empty, WithGroup returns the receiver.
	WithGroup(name string) Handler

	// Dispose discards handlerer and frees resources used by handler
	Dispose() error

	// Flags returns record flags for handler.
	Flags() RecordFlag
}

type Config struct {
	// AddSource is set to true then the handler adds a ("source", "file:line")
	// attribute to the output indicating the source code position of the log
	// statement. AddSource is false by default to skip the cost of computing
	// this information.
	AddSource bool

	Colors bool

	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// Default: LevelInfo
	Level Level

	// Secrets is a collection of log attr keys whose values are masked in logs.
	Secrets []string

	ReplaceAttr []AttrReplaceFunc

	TimeLoc        string
	TimeLayout     string
	DefaultHandler HandlerKind
}

type HandlerKind uint8

const (
	WithDefaultHandler HandlerKind = 1 << iota
	WithTextHandler
	WithColoredHandler
	WithJSONHandler
)

func (c Config) attrReplacers() []AttrReplaceFunc {
	secrets := make(map[string]struct{})
	for _, secret := range c.Secrets {
		secrets[secret] = struct{}{}
	}
	return []AttrReplaceFunc{
		// Secrets
		func(groups []string, a Attr) (Attr, error) {
			if _, ok := secrets[a.Key]; ok {
				a.Value = replaceSecretAttrValue
			}
			return a, nil
		},
	}
}
