// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package adapters

import (
	"context"
	"expvar"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strconv"

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/tui/ansicolor"
)

// consoleAdapterDroppedRecords tracks the number of log records dropped due to buffer overflow.
var consoleAdapterDroppedRecords = expvar.NewInt("logging.adapters.console_adapter.dropped_records")

// ConsoleAdapterTheme defines styling for ConsoleAdapter log output, using
// ansicolor.Style for levels, timestamps, attributes, and source links. Each
// level has a dedicated style, and the theme is built before use to format level
// strings with consistent padding.
type ConsoleAdapterTheme struct {
	lvlstrs    map[logging.Level]string
	TimeLabel  ansicolor.Style
	SourceLink ansicolor.Style
	Attrs      ansicolor.Style

	LevelHappy   ansicolor.Style
	LevelTrace   ansicolor.Style
	LevelDebug   ansicolor.Style
	LevelInfo    ansicolor.Style
	LevelNotice  ansicolor.Style
	LevelSuccess ansicolor.Style
	LevelNotImpl ansicolor.Style
	LevelWarn    ansicolor.Style
	LevelDepr    ansicolor.Style
	LevelError   ansicolor.Style
	LevelOut     ansicolor.Style
	LevelBUG     ansicolor.Style

	http map[string]ansicolor.Style
}

// LevelString returns the styled string representation of a logging level,
// padded to a consistent width. For unrecognized levels, it falls back to the
// level's string representation with default formatting.
func (t ConsoleAdapterTheme) LevelString(lvl logging.Level) string {
	lvlstr, ok := t.lvlstrs[lvl]
	if !ok {
		return fmt.Sprintf(" %-12s", lvl.String())
	}
	return lvlstr
}

func (t ConsoleAdapterTheme) HttpMethod(method string, code int) string {
	if style, ok := t.http[method]; ok {
		return style.String(fmt.Sprintf(" %-7s %d ", method, code))
	}
	return fmt.Sprintf(" %-7s %d ", method, code)
}

// Build composes the theme. It returns the built theme, ready for use by
// ConsoleAdapter.
func (t ConsoleAdapterTheme) Build() ConsoleAdapterTheme {
	t.lvlstrs = map[logging.Level]string{}
	t.lvlstrs[logging.LevelHappy] = t.LevelHappy.String(fmt.Sprintf(" %-12s", logging.LevelHappy.String()))
	t.lvlstrs[logging.LevelDebugPkg] = t.LevelTrace.String(fmt.Sprintf(" %-12s", logging.LevelDebugPkg.String()))
	t.lvlstrs[logging.LevelDebugAddon] = t.LevelTrace.String(fmt.Sprintf(" %-12s", logging.LevelDebugAddon.String()))
	t.lvlstrs[logging.LevelTrace] = t.LevelTrace.String(fmt.Sprintf(" %-12s", logging.LevelTrace.String()))
	t.lvlstrs[logging.LevelDebug] = t.LevelDebug.String(fmt.Sprintf(" %-12s", logging.LevelDebug.String()))
	t.lvlstrs[logging.LevelInfo] = t.LevelInfo.String(fmt.Sprintf(" %-12s", logging.LevelInfo.String()))
	t.lvlstrs[logging.LevelNotice] = t.LevelNotice.String(fmt.Sprintf(" %-12s", logging.LevelNotice.String()))
	t.lvlstrs[logging.LevelSuccess] = t.LevelSuccess.String(fmt.Sprintf(" %-12s", logging.LevelSuccess.String()))
	t.lvlstrs[logging.LevelNotImpl] = t.LevelNotImpl.String(fmt.Sprintf(" %-12s", logging.LevelNotImpl.String()))
	t.lvlstrs[logging.LevelWarn] = t.LevelWarn.String(fmt.Sprintf(" %-12s", logging.LevelWarn.String()))
	t.lvlstrs[logging.LevelDepr] = t.LevelDepr.String(fmt.Sprintf(" %-12s", logging.LevelDepr.String()))
	t.lvlstrs[logging.LevelError] = t.LevelError.String(fmt.Sprintf(" %-12s", logging.LevelError.String()))
	t.lvlstrs[logging.LevelOut] = t.LevelOut.String(fmt.Sprintf(" %-12s", logging.LevelOut.String()))
	t.lvlstrs[logging.LevelBUG] = t.LevelBUG.String(fmt.Sprintf(" %-12s", logging.LevelBUG.String()))

	t.http = make(map[string]ansicolor.Style)

	t.http[http.MethodGet] = ansicolor.Style{FG: ansicolor.RGB(69, 130, 246)}
	t.http[http.MethodHead] = ansicolor.Style{FG: ansicolor.RGB(113, 128, 147)}
	t.http[http.MethodPost] = ansicolor.Style{FG: ansicolor.RGB(69, 255, 153)}
	t.http[http.MethodPut] = ansicolor.Style{FG: ansicolor.RGB(251, 146, 69)}
	t.http[http.MethodPatch] = ansicolor.Style{FG: ansicolor.RGB(221, 199, 89)}
	t.http[http.MethodDelete] = ansicolor.Style{FG: ansicolor.RGB(255, 100, 100)}
	t.http[http.MethodConnect] = ansicolor.Style{FG: ansicolor.RGB(151, 162, 173)}
	t.http[http.MethodOptions] = ansicolor.Style{FG: ansicolor.RGB(150, 150, 150)}
	t.http[http.MethodTrace] = ansicolor.Style{FG: ansicolor.RGB(100, 69, 210)}

	return t
}

// ConsoleAdapterDefaultTheme returns a default theme with predefined ANSI color
// styles for levels, timestamps, attributes, and source links. It must be built
// before use with Build() when using it outside of the happy logging package.
// ConsoleAdapter will call it when building the adapter.
func ConsoleAdapterDefaultTheme() ConsoleAdapterTheme {
	return ConsoleAdapterTheme{
		TimeLabel:  ansicolor.Style{FG: ansicolor.RGB(151, 162, 173)},
		SourceLink: ansicolor.Style{FG: ansicolor.RGB(150, 150, 150)},
		Attrs:      ansicolor.Style{FG: ansicolor.RGB(221, 199, 89)},

		LevelHappy:   ansicolor.Style{FG: ansicolor.RGB(255, 237, 86)},
		LevelTrace:   ansicolor.Style{FG: ansicolor.RGB(113, 128, 147)},
		LevelDebug:   ansicolor.Style{FG: ansicolor.RGB(100, 69, 210)},
		LevelInfo:    ansicolor.Style{FG: ansicolor.RGB(45, 212, 255)},
		LevelNotice:  ansicolor.Style{FG: ansicolor.RGB(69, 130, 246)},
		LevelSuccess: ansicolor.Style{FG: ansicolor.RGB(69, 255, 153)},
		LevelNotImpl: ansicolor.Style{FG: ansicolor.RGB(69, 102, 241), BG: ansicolor.RGB(131, 141, 151)},
		LevelWarn:    ansicolor.Style{FG: ansicolor.RGB(251, 146, 69)},
		LevelDepr:    ansicolor.Style{BG: ansicolor.RGB(180, 150, 69)},
		LevelError:   ansicolor.Style{FG: ansicolor.RGB(255, 0, 0)},
		LevelBUG:     ansicolor.Style{FG: ansicolor.RGB(255, 100, 100), BG: ansicolor.RGB(128, 0, 0)},
		LevelOut:     ansicolor.Style{FG: ansicolor.RGB(69, 69, 69)},
	}
}

type ConsoleAdapter struct {
	config logging.Config
	writer *logging.Writer
	theme  ConsoleAdapterTheme
	attrs  []slog.Attr
	group  string
}

func NewConsoleAdapter(w io.Writer, theme ConsoleAdapterTheme) *logging.AdapterComposer[*ConsoleAdapter] {
	return logging.NewAdapter(w, func(w *logging.Writer, config logging.Config) *ConsoleAdapter {
		return &ConsoleAdapter{
			config: config,
			writer: w,
			theme:  theme.Build(),
		}
	})
}

func NewBufferedConsoleAdapter(w io.Writer, theme ConsoleAdapterTheme) *logging.AdapterComposer[*logging.BufferedAdapter[*ConsoleAdapter]] {
	return logging.NewAdapter(w, func(w *logging.Writer, config logging.Config) *logging.BufferedAdapter[*ConsoleAdapter] {
		adapter := &ConsoleAdapter{
			config: config,
			writer: w,
			theme:  theme.Build(),
		}
		return logging.NewBufferedAdapter(adapter, config.Adapter, consoleAdapterDroppedRecords)
	})
}

// Enabled reports whether the adapter handles records at the given level,
// based on the configured minimum level.
func (c *ConsoleAdapter) Enabled(_ context.Context, l slog.Level) bool {
	return l >= c.config.Level.Level()
}

// Handle processes a log record
func (c *ConsoleAdapter) Handle(_ context.Context, record slog.Record) error {
	buf := logging.NewLineBuffer()
	defer buf.Free()
	c.buildLine(buf, record)
	_, err := c.writer.Write(*buf)
	return err
}

func (c *ConsoleAdapter) BatchHandle(records []logging.Record) error {
	buf := logging.NewLineBuffer()
	defer buf.Free()

	for _, record := range records {
		c.buildLine(buf, record.Record)
	}

	_, err := c.writer.Write(*buf)
	return err
}

// WithAttrs returns a new slog.Handler with the specified attributes added.
// The original adapter is not modified.
func (c *ConsoleAdapter) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := c.clone()
	clone.attrs = slices.Clone(c.attrs)
	clone.attrs = append(clone.attrs, attrs...)
	return clone
}

// WithGroup returns a new slog.Handler with the specified group name.
// The original adapter is not modified.
func (c *ConsoleAdapter) WithGroup(name string) slog.Handler {
	clone := c.clone()
	if c.group == "" {
		clone.group = name
	} else {
		clone.group = c.group + "." + name
	}
	return clone
}

// Dispose is noop
func (c *ConsoleAdapter) Dispose() error { return nil }

// Ready is noop
func (c *ConsoleAdapter) Ready() {}

// Flush is noop
func (c *ConsoleAdapter) Flush() error { return nil }

func (c *ConsoleAdapter) HTTP(_ context.Context, method string, statusCode int, path string, record slog.Record) error {
	buf := logging.NewLineBuffer()
	defer buf.Free()
	c.buildHttpLine(buf, method, statusCode, path, record)
	_, err := c.writer.Write(*buf)
	return err
}

func (c *ConsoleAdapter) HTTPBatchHandle(records []logging.HttpRecord) error {
	buf := logging.NewLineBuffer()
	defer buf.Free()

	for _, record := range records {
		c.buildHttpLine(buf, record.Method, record.Code, record.Path, record.Record)
	}

	_, err := c.writer.Write(*buf)
	return err
}

// clone creates a new ConsoleAdapter with the same configuration.
func (c *ConsoleAdapter) clone() *ConsoleAdapter {
	clone := &ConsoleAdapter{
		config: c.config,
		writer: c.writer,
		theme:  c.theme,
		attrs:  slices.Clone(c.attrs),
		group:  c.group,
	}
	return clone
}

// buildLine formats a log record into a LineBuffer, applying the theme for levels,
// timestamps, messages, attributes, and source information. It returns an error
// if attribute marshaling fails.
func (c *ConsoleAdapter) buildLine(buf *logging.LineBuffer, record slog.Record) {
	_, _ = buf.WriteString(c.theme.LevelString(logging.Level(record.Level)))

	if !c.config.NoTimestamp {
		_ = buf.WriteByte(SP)
		_, _ = buf.WriteString(c.theme.TimeLabel.String(record.Time.Format(c.config.TimeFormat)))
	}

	var msg string
	msg = record.Message
	_ = buf.WriteByte(SP)
	if record.Level < logging.LevelDebug.Level() {
		msg = c.theme.LevelTrace.String(msg)
	} else if record.Level == logging.LevelDebug.Level() {
		msg = c.theme.LevelDebug.String(msg)
	} else {
		msg = record.Message
	}
	_, _ = buf.WriteString(msg)

	// Merge adapter attributes with record attributes
	attrs := NewAttrMap()
	if len(c.attrs) > 0 {
		for _, a := range c.attrs {
			if c.group != "" {
				// Prefix attribute keys with group name
				if a.Value.Kind() == slog.KindGroup {
					attrs.Set(c.group+"."+a.Key, AttrGroupToMap(a.Value.Group()))
				} else {
					attrs.Set(c.group+"."+a.Key, a.Value.Any())
				}
			} else {
				if a.Value.Kind() == slog.KindGroup {
					attrs.Set(a.Key, AttrGroupToMap(a.Value.Group()))
				} else {
					attrs.Set(a.Key, a.Value.Any())
				}
			}
		}
	}

	var src *slog.Source
	if record.NumAttrs() > 0 {
		record.Attrs(func(a slog.Attr) bool {
			if a.Key == slog.SourceKey {
				srcattr := a.Value.Any()
				var ok bool
				src, ok = srcattr.(*slog.Source)
				if ok {
					return true
				}
			}
			if c.group != "" && a.Value.Kind() != slog.KindGroup {
				// Prefix non-group record attributes with group name
				attrs.Set(c.group+"."+a.Key, a.Value.Any())
			} else if a.Value.Kind() == slog.KindGroup {
				attrs.Set(a.Key, AttrGroupToMap(a.Value.Group()))
			} else {
				attrs.Set(a.Key, a.Value.Any())
			}
			return true
		})
	}

	var attrErr error
	if attrs.Len() > 0 {
		_ = buf.WriteByte(SP)
		b, err := attrs.MarshalJSON()
		if err != nil {
			attrErr = fmt.Errorf("ConsoleAdapter failed to marshal attributes: %w", err)
		} else {
			_, _ = buf.WriteString(c.theme.Attrs.String(string(b)))
		}
	}

	if src == nil {
		src = record.Source()
	}

	if c.config.AddSource && src != nil {
		_ = buf.WriteByte(SP)
		_, _ = buf.WriteString(c.theme.SourceLink.String(src.File + ":" + strconv.Itoa(src.Line)))
		_ = buf.WriteByte(SP)
		_, _ = buf.WriteString(c.theme.SourceLink.String(path.Base(src.Function)))
	}

	_ = buf.WriteByte(NL)
	if attrErr != nil {
		_, _ = buf.WriteString(attrErr.Error())
		_ = buf.WriteByte(NL)
	}
	return
}

func (c *ConsoleAdapter) buildHttpLine(buf *logging.LineBuffer, method string, statusCode int, p string, record slog.Record) {
	_, _ = buf.WriteString(c.theme.HttpMethod(method, statusCode))

	if !c.config.NoTimestamp {
		_ = buf.WriteByte(SP)
		_, _ = buf.WriteString(c.theme.TimeLabel.String(record.Time.Format(c.config.TimeFormat)))
	}
	_ = buf.WriteByte(SP)
	_, _ = buf.WriteString(p)

	// Merge adapter attributes with record attributes
	attrs := NewAttrMap()
	if len(c.attrs) > 0 {
		for _, a := range c.attrs {
			if c.group != "" {
				// Prefix attribute keys with group name
				if a.Value.Kind() == slog.KindGroup {
					attrs.Set(c.group+"."+a.Key, AttrGroupToMap(a.Value.Group()))
				} else {
					attrs.Set(c.group+"."+a.Key, a.Value.Any())
				}
			} else {
				if a.Value.Kind() == slog.KindGroup {
					attrs.Set(a.Key, AttrGroupToMap(a.Value.Group()))
				} else {
					attrs.Set(a.Key, a.Value.Any())
				}
			}
		}
	}

	var src *slog.Source
	if record.NumAttrs() > 0 {
		record.Attrs(func(a slog.Attr) bool {
			if a.Key == slog.SourceKey {
				srcattr := a.Value.Any()
				var ok bool
				src, ok = srcattr.(*slog.Source)
				if ok {
					return true
				}
			}
			if c.group != "" && a.Value.Kind() != slog.KindGroup {
				// Prefix non-group record attributes with group name
				attrs.Set(c.group+"."+a.Key, a.Value.Any())
			} else if a.Value.Kind() == slog.KindGroup {
				attrs.Set(a.Key, AttrGroupToMap(a.Value.Group()))
			} else {
				attrs.Set(a.Key, a.Value.Any())
			}
			return true
		})
	}

	var attrErr error
	if attrs.Len() > 0 {
		_ = buf.WriteByte(SP)
		b, err := attrs.MarshalJSON()
		if err != nil {
			attrErr = fmt.Errorf("ConsoleAdapter failed to marshal attributes: %w", err)
		} else {
			_, _ = buf.WriteString(c.theme.Attrs.String(string(b)))
		}
	}

	if src == nil {
		src = record.Source()
	}

	if c.config.AddSource && src != nil {
		_ = buf.WriteByte(SP)
		_, _ = buf.WriteString(c.theme.SourceLink.String(src.File + ":" + strconv.Itoa(src.Line)))
		_ = buf.WriteByte(SP)
		_, _ = buf.WriteString(c.theme.SourceLink.String(path.Base(src.Function)))
	}

	_ = buf.WriteByte(NL)
	if attrErr != nil {
		_, _ = buf.WriteString(attrErr.Error())
		_ = buf.WriteByte(NL)
	}
	return
}
