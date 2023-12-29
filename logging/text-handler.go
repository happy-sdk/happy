// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"context"
	"io"
	"strings"
)

func newTextHandler(l LevelIface, flags RecordFlag, w io.Writer) *TextHandler {
	return &TextHandler{
		level: l,
		w:     w,
		flags: flags,
	}
}

type TextHandler struct {
	flags RecordFlag
	level LevelIface
	w     io.Writer

	groups []string // all groups started from WithGroup
	attrs  []Attr
}

func (h *TextHandler) Enabled(ctx context.Context, level Level) bool {
	return true
}

func (h *TextHandler) Handle(ctx context.Context, r Record) error {
	var (
		line logLine
	)

	if r.TimeString.Kind() != AttrOmittedKind {
		if err := line.writeAttr("", r.TimeString); err != nil {
			return err
		}
	}

	if r.Level.Kind() != AttrOmittedKind {
		if err := line.writeAttr("", r.Level); err != nil {
			return err
		}
	}

	if r.Message.Kind() != AttrOmittedKind {
		if err := line.writeAttr("", r.Message); err != nil {
			return err
		}
	}

	if r.Error.Kind() != AttrOmittedKind {
		if err := line.writeAttr(":", r.Error); err != nil {
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
		if err := line.writeAttr("", r.Source); err != nil {
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

func (h *TextHandler) WithAttrs(attrs ...Attr) Handler {
	h2 := newTextHandler(h.level, h.flags, h.w)
	h2.groups = append(h2.groups, h.groups...)
	h2.attrs = append(h.attrs, attrs...)
	return h2
}

func (h *TextHandler) WithGroup(group string) Handler {
	h2 := newTextHandler(h.level, h.flags, h.w)
	h2.groups = append(h.groups, group)
	h2.attrs = append(h2.attrs, h.attrs...)
	return h2
}

func (h *TextHandler) Dispose() error {
	return nil
}

func (h *TextHandler) Flags() RecordFlag {
	return h.flags
}
