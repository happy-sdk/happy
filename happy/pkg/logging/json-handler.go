// Copyright 2023 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"context"
	"io"

	"encoding/json"
)

func newJSONHandler(l Level, flags RecordFlag, w io.Writer) *JSONHandler {
	return &JSONHandler{
		level: l,
		w:     w,
		flags: flags,
	}
}

type JSONHandler struct {
	flags RecordFlag
	level Level
	w     io.Writer

	groups []string // all groups started from WithGroup
	attrs  []Attr
}

func (h *JSONHandler) Enabled(ctx context.Context, level Level) bool {
	return true
}

const lf = '\n'

func (h *JSONHandler) Handle(ctx context.Context, r Record) error {
	entry := make(map[string]any)

	if r.TimeString.Kind() != AttrOmittedKind {
		entry[r.TimeString.Key] = r.TimeString.Value.value
	}

	if r.Level.Kind() != AttrOmittedKind {
		entry[r.Level.Key] = r.Level.Value.value
	}

	if r.Message.Kind() != AttrOmittedKind {
		entry[r.Message.Key] = r.Message.Value.value
	}

	if r.Error.Kind() != AttrOmittedKind {
		entry[r.Error.Key] = r.Error.Value.String()
	}

	if r.Source.Kind() != AttrOmittedKind {
		entry[r.Source.Key] = r.Source.Value.value
	}

	data := jsonObject(r.Data)

	if len(data) > 0 {
		entry[DataKey] = data[DataKey]
	}

	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	b = append(b, lf)
	_, err = h.w.Write(b)

	return err
}

func jsonObject(attr Attr) map[string]any {
	obj := make(map[string]any)
	switch attr.Kind() {
	case AttrSingleKind:
		obj[attr.Key] = attr.Value.value
	case AttrArrayKind:
		var data []any
		for _, el := range attr.Value.Array() {
			data = append(data, el.value)
		}
		obj[attr.Key] = data
	case AttrObjectKind:
		data := make(map[string]any)
		for _, el := range attr.Value.Object() {
			if el.Kind() == AttrSingleKind {
				data[el.Key] = el.Value.value
			} else {
				data[el.Key] = jsonObject(el)[el.Key]
			}
		}
		obj[attr.Key] = data
	}
	return obj
}

func (h *JSONHandler) WithAttrs(attrs ...Attr) Handler {
	h2 := newJSONHandler(h.level, h.flags, h.w)
	h2.groups = append(h2.groups, h.groups...)
	h2.attrs = append(h.attrs, attrs...)
	return h2
}

func (h *JSONHandler) WithGroup(group string) Handler {
	h2 := newJSONHandler(h.level, h.flags, h.w)
	h2.groups = append(h.groups, group)
	h2.attrs = append(h2.attrs, h.attrs...)
	return h2
}

func (h *JSONHandler) Dispose() error {
	return nil
}

func (h *JSONHandler) Flags() RecordFlag {
	return h.flags
}
