// Copyright 2023 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"context"
	"fmt"
	"io"
	"strings"
)

func newDefaultHandler(l Level, flags RecordFlag, w io.Writer) *DefaultHandler {
	return &DefaultHandler{
		level: l,
		w:     w,
		flags: flags,
	}
}

type DefaultHandler struct {
	flags RecordFlag
	level Level
	w     io.Writer

	groups []string // all groups started from WithGroup
	attrs  []Attr
}

func (h *DefaultHandler) Enabled(ctx context.Context, level Level) bool {
	return true
}

func (h *DefaultHandler) Handle(ctx context.Context, r Record) error {
	var line logLine

	if r.TimeString.Kind() != AttrOmittedKind {
		if err := line.writeString(r.TimeString.Value.String()); err != nil {
			return err
		}
	}

	if r.Level.Kind() != AttrOmittedKind {
		if err := line.writeString(r.Level.Value.String()); err != nil {
			return err
		}
	}

	if r.Message.Kind() != AttrOmittedKind {
		if err := line.writeString(r.Message.Value.String()); err != nil {
			return err
		}
	}

	if r.Error.Kind() != AttrOmittedKind {
		if err := line.writeString(r.Error.Value.String()); err != nil {
			return err
		}
	}

	groups := strings.Join(h.groups, ".")

	for _, attr := range h.attrs {
		if attr.Kind() != AttrOmittedKind {
			if err := line.writeAttr(groups, attr); err != nil {
				return err
			}
		}
	}

	if r.Data.Kind() != AttrOmittedKind {
		if err := line.writeAttr(groups, r.Data); err != nil {
			return err
		}
	}

	if r.Source.Kind() != AttrOmittedKind {
		if err := line.writeString(r.Source.Value.String()); err != nil {
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

func (h *DefaultHandler) WithAttrs(attrs ...Attr) Handler {
	h2 := newDefaultHandler(h.level, h.flags, h.w)
	h2.groups = append(h2.groups, h.groups...)
	h2.attrs = append(h.attrs, attrs...)
	return h2
}

func (h *DefaultHandler) WithGroup(group string) Handler {
	h2 := newDefaultHandler(h.level, h.flags, h.w)
	h2.groups = append(h.groups, group)
	h2.attrs = append(h2.attrs, h.attrs...)
	return h2
}

func (h *DefaultHandler) Dispose() error {
	return nil
}

func (h *DefaultHandler) Flags() RecordFlag {
	return h.flags
}

type logLine struct {
	len int
	buf strings.Builder
}

func (l *logLine) writeString(s string) error {
	if l.len > 0 {
		l.buf.WriteRune(' ')
	}
	n, err := l.buf.WriteString(s)
	if err != nil {
		return err
	}
	l.len += n
	return nil
}

func (l *logLine) writeAttrValue(key string, value AttrValue) error {
	if key == "" {
		key = "data"
	}

	switch value.kind {
	case AttrObjectKind:
		for _, el := range value.Object() {
			if err := l.writeAttrValue(key+"."+el.Key, el.Value); err != nil {
				return err
			}
		}
		return nil
	}
	entry := fmt.Sprintf("%s=%v", key, value.value)
	return l.writeString(entry)
}

func (l *logLine) writeArrayValue(key string, ix int, value AttrValue) error {
	if key == "" {
		key = "data"
	}

	entry := fmt.Sprintf("%s[%d]=%v", key, ix, value.value)
	return l.writeString(entry)
}

func (l *logLine) writeAttr(groups string, attr Attr) error {
	switch attr.Kind() {
	case AttrSingleKind:
		var entry string
		if groups != "" {
			entry = groups + "."
		}
		entry += attr.Key
		entry += "="
		entry += attr.Value.String()
		if err := l.writeString(entry); err != nil {
			return err
		}
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
			if err := l.writeAttrValue(key, el.Value); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *logLine) writeRune(r rune) error {
	n, err := l.buf.WriteRune(r)
	l.len += n
	return err
}

func (l *logLine) String() string {
	return l.buf.String()
}
