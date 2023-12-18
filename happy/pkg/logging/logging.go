// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/happy-sdk/happy/pkg/settings"
	"golang.org/x/text/language"
)

var (
	ErrAttrReplace      = errors.New("replace attr error")
	ErrUnknownLevelName = errors.New("unknown level name")
)

type Settings struct {
	Level             Level           `key:"level" default:"ok" mutation:"mutable"`
	Secrets           settings.String `key:"secrets" mutation:"once"`
	Source            settings.Bool   `key:"source" default:"false" mutation:"once"`
	Handler           HandlerKind     `key:"default.handler" default:"colored" mutation:"once"`
	TimestampEnabled  settings.Bool   `key:"timestamp.enabled" default:"true" mutation:"once"`
	TimestampFormat   settings.String `key:"timestamp.format" default:"15:04:05.999" mutation:"once"`
	TimestampLocation settings.String `key:"timestamp.location" default:"Local" mutation:"once"`
	SlogGlobal        settings.Bool   `key:"slog.global" default:"false" mutation:"once"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	blueprint, err := settings.NewBlueprint(s)
	if err != nil {
		return nil, err
	}
	blueprint.Describe("logging.level", language.English, "Set application output verbosity")

	return blueprint, nil
}

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

func (hk HandlerKind) SettingKind() settings.Kind {
	return settings.KindString
}

func (hk HandlerKind) String() string {
	switch hk {
	case WithDefaultHandler:
		return "default"
	case WithTextHandler:
		return "text"
	case WithColoredHandler:
		return "colored"
	case WithJSONHandler:
		return "json"
	default:
		return fmt.Sprintf("unknown handler kind: %d", hk)
	}
}

func (hk HandlerKind) MarshalSetting() ([]byte, error) {
	return []byte(hk.String()), nil
}

func (hk *HandlerKind) UnmarshalSetting(data []byte) error {
	return hk.UnmarshalText(data)
}

func (hk *HandlerKind) UnmarshalText(data []byte) error {
	return hk.parse(string(data))
}

func (hk *HandlerKind) parse(s string) (err error) {
	if val, err := strconv.ParseUint(s, 10, 8); err == nil {
		*hk = HandlerKind(val)
		return nil
	}

	switch strings.ToLower(s) {
	case "default":
		*hk = WithDefaultHandler
	case "text":
		*hk = WithTextHandler
	case "colored":
		*hk = WithColoredHandler
	case "json":
		*hk = WithJSONHandler
	default:
		err = ErrUnknownLevelName
	}
	return err
}

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

func Simple(level Level) *DefaultLogger[Level] {
	cnf := Config{
		Level:          level,
		AddSource:      true,
		DefaultHandler: WithTextHandler,
	}
	logger, err := New(cnf)
	if err != nil {
		// there is no case where this should happen
		// but if it does then we panic
		panic(err)
	}
	return logger
}

type LevelIface interface {
	String() string
	Int() int
	MarshalText() ([]byte, error)
}

type LoggerIface[LVL LevelIface] interface {
	Level() LevelIface
	SetLevel(LevelIface)
	SystemDebug(msg string, args ...any)
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Ok(msg string, args ...any)
	Notice(msg string, args ...any)
	Warn(msg string, args ...any)
	NotImplemented(msg string, args ...any)
	Deprecated(msg string, args ...any)
	Issue(msg string, args ...any)
	Error(msg string, err error, args ...any)
	BUG(msg string, args ...any)
	Msg(msg string, args ...any)
	Msgf(format string, args ...any)
	Log(level LevelIface, msg string, args ...any)
	HTTP(status int, path string, args ...any)
	LogDepth(ctx context.Context, calldepth int, level LevelIface, msg string, err error, args ...any)
	With(args ...any) Logger
	WithGroup(group string) Logger
}
