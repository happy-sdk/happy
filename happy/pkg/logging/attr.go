// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"fmt"

	"log/slog"

	"github.com/happy-sdk/vars"
)

// Keys for "built-in" attributes.
const (
	// TimeKey is the key used by the built-in handlers for the time
	// when the log method is called. The associated Value is a [time.Time].
	TimeKey = "time"
	// LevelKey is the key used by the built-in handlers for the level
	// of the log call. The associated value is a [Level].
	LevelKey = "level"
	// MessageKey is the key used by the built-in handlers for the
	// message of the log call. The associated value is a string.
	MessageKey = "message"
	// SourceKey is the key used by the built-in handlers for the source file
	// and line of the log call. The associated value is a string.
	SourceKey = "source"
	// ErrorKey is the key used for errors by Logger.Error.
	// The associated value is an [error].
	ErrorKey = "error"
	// DataKey is the key used for data constructed from logger arguments.
	DataKey = "data"
)

var replaceSecretAttrValue = AttrValue{
	kind:  AttrSingleKind,
	value: "****",
}

func attrsFromArgs(args []any) []Attr {
	var attrs []Attr

	for _, arg := range args {
		switch a := arg.(type) {
		case Attr:
			attrs = append(attrs, a)
		case slog.Attr:
			if a.Value.Kind() == slog.KindGroup {
				var els []any
				for _, el := range a.Value.Group() {
					els = append(els, el)
				}
				g := attrsFromArgs(els)
				attrs = append(attrs, Attr{
					Key: a.Key,
					Value: AttrValue{
						kind:  AttrObjectKind,
						value: g,
					},
				})
			} else {
				attrs = append(attrs, NewAttr(a.Key, a.Value.Any()))
			}
		case vars.Variable:
			attrs = append(attrs, NewAttr(a.Name(), a.Any()))
		default:
			attrs = append(attrs, NewAttr("", a))
		}
	}
	return attrs
}

func NewAttr(key string, val any) Attr {
	return Attr{
		Key:   key,
		Value: NewValue(val),
	}
}

func NewValue(arg any) (val AttrValue) {
	switch v := arg.(type) {
	case slog.Attr:
		if v.Value.Kind() == slog.KindGroup {
			val.kind = AttrObjectKind
			var els []any
			for _, el := range v.Value.Group() {
				els = append(els, el)
			}
			val.value = attrsFromArgs(els)
		} else {
			val.kind = AttrSingleKind
			val.value = v
		}
	case []Attr:
		val.kind = AttrObjectKind
		val.value = v
	case []AttrValue:
		val.kind = AttrArrayKind
		val.value = v
	default:
		val.kind = AttrSingleKind
		val.value = v
	}
	return
}

type AttrKind uint8

const (
	AttrOmittedKind AttrKind = iota
	AttrEmptyKind
	AttrSingleKind
	AttrArrayKind
	AttrObjectKind
)

type AttrReplaceFunc func(groups []string, a Attr) (Attr, error)

type Attr struct {
	Key   string
	Value AttrValue
}

func (a Attr) Kind() AttrKind {
	return a.Value.kind
}

type AttrValue struct {
	kind  AttrKind
	value any
}

func (a AttrValue) Array() []AttrValue {
	if a.kind == AttrArrayKind {
		values, ok := a.value.([]AttrValue)
		if ok {
			return values
		}
	}

	return []AttrValue{}
}

func (a AttrValue) Object() []Attr {
	if a.kind == AttrObjectKind {
		values, ok := a.value.([]Attr)
		if ok {
			return values
		}
	}

	return []Attr{}
}

func (a AttrValue) String() string {
	if a.value == nil {
		return ""
	}

	if stringer, ok := a.value.(fmt.Stringer); ok {
		return stringer.String()
	}
	if str, ok := a.value.(string); ok {
		return str
	}
	if err, ok := a.value.(error); ok {
		return err.Error()
	}
	return fmt.Sprint(a.value)
}
