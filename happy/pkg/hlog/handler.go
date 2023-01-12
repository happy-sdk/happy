// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package hlog

import (
	"encoding"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mkungla/happy/pkg/hlog/internal/buffer"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

var groupPool = sync.Pool{New: func() any {
	s := make([]string, 0, 10)
	return &s
}}

func NewHandler(w io.Writer) slog.Handler {
	return Config{}.NewHandler(w)
}

type Handler struct {
	opts              slog.HandlerOptions
	preformattedAttrs []byte
	groupPrefix       string   // for text: prefix of groups opened in preformatting
	groups            []string // all groups started from WithGroup
	nOpenGroups       int      // the number of groups opened in preformattedAttrs
	mu                sync.Mutex
	w                 io.Writer
	colors            bool
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
func (h *Handler) Enabled(level slog.Level) bool {
	return h.enabled(level)
}

// With returns a new JSONHandler whose attributes consists
// of h's attributes followed by attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.withAttrs(attrs)
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return h.withGroup(name)
}

// Handle formats its argument Record as a JSON object on a single line.
//
// If the Record's time is zero, the time is omitted.
// Otherwise, the key is "time"
// and the value is output as with json.Marshal.
//
// If the Record's level is zero, the level is omitted.
// Otherwise, the key is "level"
// and the value of [Level.String] is output.
//
// If the AddSource option is set and source information is available,
// the key is "source"
// and the value is output as "FILE:LINE".
//
// The message's key is "msg".
//
// To modify these or other attributes, or remove them from the output, use
// [HandlerOptions.ReplaceAttr].
//
// Values are formatted as with encoding/json.Marshal, with the following
// exceptions:
//   - Floating-point NaNs and infinities are formatted as one of the strings
//     "NaN", "+Inf" or "-Inf".
//   - Levels are formatted as with Level.String.
//
// Each call to Handle results in a single serialized call to io.Writer.Write.
func (h *Handler) Handle(r slog.Record) error {
	if !h.enabled(r.Level) {
		return nil
	}
	return h.handle(r)
}

func (h *Handler) clone() *Handler {
	// We can't use assignment because we can't copy the mutex.
	return &Handler{
		opts:              h.opts,
		preformattedAttrs: slices.Clip(h.preformattedAttrs),
		groupPrefix:       h.groupPrefix,
		groups:            slices.Clip(h.groups),
		nOpenGroups:       h.nOpenGroups,
		w:                 h.w,
		colors:            h.colors,
	}
}

// Enabled reports whether l is greater than or equal to the
// minimum level.
func (h *Handler) enabled(l slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return l >= minLevel
}

func (h *Handler) withAttrs(as []slog.Attr) *Handler {
	h2 := h.clone()
	// Pre-format the attributes as an optimization.
	prefix := buffer.New()
	defer prefix.Free()
	prefix.WriteString(h.groupPrefix)
	state := h2.newHandleState((*buffer.Buffer)(&h2.preformattedAttrs), false, "", prefix)
	defer state.free()
	if len(h2.preformattedAttrs) > 0 {
		state.sep = h.attrSep()
	}
	state.openGroups()
	var lvl Level
	if h.opts.Level != nil {
		lvl = Level(h.opts.Level.Level())
	}
	for _, a := range as {
		state.appendAttr(lvl, a)
	}
	// Remember the new prefix for later keys.
	h2.groupPrefix = state.prefix.String()
	// Remember how many opened groups are in preformattedAttrs,
	// so we don't open them again when we handle a Record.
	h2.nOpenGroups = len(h2.groups)
	return h2
}

// attrSep returns the separator between attributes.
func (h *Handler) attrSep() string {
	return " "
}

func (h *Handler) withGroup(name string) *Handler {
	h2 := h.clone()
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *Handler) handle(r slog.Record) error {
	state := h.newHandleState(buffer.New(), true, "", nil)
	defer state.free()

	// Built-in attributes. They are not in a group.
	stateGroups := state.groups
	state.groups = nil // So ReplaceAttrs sees no groups instead of the pre groups.
	// rep := h.opts.ReplaceAttr

	lvl := Level(r.Level)

	if lvl != LevelOut {
		rep := h.opts.ReplaceAttr
		// level
		var level string
		if state.colors {
			level = lvl.ColorLabel()
		} else {
			level = lvl.Label()
		}
		state.buf.WriteString(level)

		// time
		if !r.Time.IsZero() {
			val := r.Time.Round(0) // strip monotonic to match Attr behavior
			if rep == nil {
				state.appendTimeMicro(val)
			} else {
				state.appendTime(val)
			}
		}
	}

	state.appendString(r.Message)
	state.buf.WriteByte(' ')

	state.groups = stateGroups // Restore groups passed to ReplaceAttrs.
	state.appendNonBuiltIns(r)

	// source
	if h.opts.AddSource {
		file, line := r.SourceLine()
		if file != "" {
			state.appendSource(file, line)
		}
	}
	state.buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(*state.buf)
	return err
}

func (h *Handler) newHandleState(buf *buffer.Buffer, freeBuf bool, sep string, prefix *buffer.Buffer) handleState {
	s := handleState{
		h:       h,
		buf:     buf,
		freeBuf: freeBuf,
		sep:     sep,
		prefix:  prefix,
		colors:  h.colors,
	}
	if h.opts.ReplaceAttr != nil {
		s.groups = groupPool.Get().(*[]string)
		*s.groups = append(*s.groups, h.groups[:h.nOpenGroups]...)
	}
	return s
}

// handleState holds state for a single call to commonHandler.handle.
// The initial value of sep determines whether to emit a separator
// before the next key, after which it stays true.
type handleState struct {
	colors  bool
	h       *Handler
	buf     *buffer.Buffer
	freeBuf bool           // should buf be freed?
	sep     string         // separator to write before next key
	prefix  *buffer.Buffer // for text: key prefix
	groups  *[]string      // pool-allocated slice of active groups, for ReplaceAttr
}

func (s *handleState) free() {
	if s.freeBuf {
		s.buf.Free()
	}
	if gs := s.groups; gs != nil {
		*gs = (*gs)[:0]
		groupPool.Put(gs)
	}
}

func (s *handleState) openGroups() {
	for _, n := range s.h.groups[s.h.nOpenGroups:] {
		s.openGroup(n)
	}
}

// Separator for group names and keys.
const keyComponentSep = '.'

// openGroup starts a new group of attributes
// with the given name.
func (s *handleState) openGroup(name string) {
	s.prefix.WriteString(name)
	s.prefix.WriteByte(keyComponentSep)
	// Collect group names for ReplaceAttr.
	if s.groups != nil {
		*s.groups = append(*s.groups, name)
	}

}

// closeGroup ends the group with the given name.
func (s *handleState) closeGroup(name string) {
	(*s.prefix) = (*s.prefix)[:len(*s.prefix)-len(name)-1 /* for keyComponentSep */]
	s.sep = s.h.attrSep()
	if s.groups != nil {
		*s.groups = (*s.groups)[:len(*s.groups)-1]
	}
}

// appendAttr appends the Attr's key and value using app.
// If sep is true, it also prepends a separator.
// It handles replacement and checking for an empty key.
// It sets sep to true if it actually did the append (if the key was non-empty
// after replacement).
func (s *handleState) appendAttr(lvl Level, a slog.Attr) {
	if a.Key == "" {
		return
	}
	v := a.Value.Resolve()
	if rep := s.h.opts.ReplaceAttr; rep != nil && v.Kind() != slog.GroupKind {
		var gs []string
		if s.groups != nil {
			gs = *s.groups
		}
		a = rep(gs, slog.Attr{Key: a.Key, Value: v})
		if a.Key == "" {
			return
		}
		v = a.Value.Resolve()
	}
	if v.Kind() == slog.GroupKind {
		s.openGroup(a.Key)
		for _, aa := range v.Group() {
			s.appendAttr(lvl, aa)
		}
		s.closeGroup(a.Key)
	} else {
		s.appendKey(lvl, a.Key)
		s.appendValue(v)
	}
}

func (s *handleState) appendError(err error) {
	s.appendString(fmt.Sprintf("!ERROR:%v", err))
}

func (s *handleState) appendKey(lvl Level, key string) {
	if s.colors {
		s.buf.Write(lvl.color())
	}

	s.buf.WriteString(s.sep)
	if s.prefix != nil {
		// TODO: optimize by avoiding allocation.
		s.appendString(string(*s.prefix) + key)
	} else {
		s.appendString(key)
	}
	if s.colors {
		s.buf.Write([]byte{'\033', '[', '9', '0', 'm'})
	}
	s.buf.WriteByte('=')

	if s.colors {
		s.buf.Write([]byte{'\033', '[', '0', 'm'})
	}
	s.sep = s.h.attrSep()
}

func (s *handleState) appendSource(file string, line int) {
	if s.colors {
		s.buf.Write([]byte{'\033', '[', '2', 'm'})
		defer s.buf.Write([]byte{'\033', '[', '0', 'm'})
	}
	s.buf.WriteByte(' ')
	// text
	// common case: no quoting needed.
	s.appendString(file)
	s.buf.WriteByte(':')
	s.buf.WritePosInt(line)
}

func (s *handleState) appendString(str string) {
	// text
	if needsQuoting(str) {
		*s.buf = strconv.AppendQuote(*s.buf, str)
	} else {
		s.buf.WriteString(str)
	}
}

func (s *handleState) appendValue(v slog.Value) {
	if err := coloredAppendTextValue(s, v); err != nil {
		s.appendError(err)
	}
}
func (s *handleState) appendTimeMicro(t time.Time) {
	if s.colors {
		s.buf.Write([]byte{'\033', '[', '2', 'm'})
		defer s.buf.Write([]byte{'\033', '[', '0', 'm'})
	}

	hour, min, sec := t.Clock()

	s.buf.WritePosIntWidth(hour, 2)
	s.buf.WriteByte(':')
	s.buf.WritePosIntWidth(min, 2)
	s.buf.WriteByte(':')
	s.buf.WritePosIntWidth(sec, 2)
	ns := t.Nanosecond()
	s.buf.WriteByte('.')
	s.buf.WritePosIntWidth(ns/1e3, 6)
	s.buf.WriteByte(' ')

}

func (s *handleState) appendNonBuiltIns(r slog.Record) {
	// preformatted Attrs
	if len(s.h.preformattedAttrs) > 0 {
		s.buf.WriteString(s.sep)
		s.buf.Write(s.h.preformattedAttrs)
		s.sep = s.h.attrSep()
	}
	// Attrs in Record -- unlike the built-in ones, they are in groups started
	// from WithGroup.
	s.prefix = buffer.New()
	defer s.prefix.Free()
	s.prefix.WriteString(s.h.groupPrefix)
	s.openGroups()
	lvl := Level(r.Level)
	r.Attrs(func(a slog.Attr) {
		s.appendAttr(lvl, a)
	})
}

func coloredAppendTextValue(s *handleState, v slog.Value) error {
	if s.colors {
		s.buf.Write([]byte{'\033', '[', '3', 'm'})
		defer s.buf.Write([]byte{'\033', '[', '0', 'm'})
	}

	switch v.Kind() {
	case slog.StringKind:
		s.appendString(v.String())
	case slog.TimeKind:
		writeTimeRFC3339Millis(s.buf, v.Time())
	case slog.AnyKind:
		aany := v.Any()
		if tm, ok := aany.(encoding.TextMarshaler); ok {
			data, err := tm.MarshalText()
			if err != nil {
				return err
			}
			// TODO: avoid the conversion to string.
			s.appendString(string(data))
			return nil
		}
		if bs, ok := byteSlice(aany); ok {
			// As of Go 1.19, this only allocates for strings longer than 32 bytes.
			s.buf.WriteString(strconv.Quote(string(bs)))
			return nil
		}
		s.appendString(fmt.Sprint(v.Any()))
	default:
		*s.buf = valueAppend(v, *s.buf)
	}
	return nil
}

// append appends a text representation of v to dst.
// v is formatted as with fmt.Sprint.
func valueAppend(v slog.Value, dst []byte) []byte {
	switch v.Kind() {
	case slog.Int64Kind:
		return strconv.AppendInt(dst, v.Int64(), 10)
	case slog.Uint64Kind:
		return strconv.AppendUint(dst, v.Uint64(), 10)
	case slog.Float64Kind:
		return strconv.AppendFloat(dst, v.Float64(), 'g', -1, 64)
	case slog.BoolKind:
		return strconv.AppendBool(dst, v.Bool())
	case slog.DurationKind:
		return append(dst, v.Duration().String()...)
	case slog.TimeKind:
		return append(dst, v.Time().String()...)
	case slog.AnyKind, slog.GroupKind, slog.LogValuerKind:
		return append(dst, fmt.Sprint(v.Any())...)
	default:
		panic(fmt.Sprintf("bad kind: %s", v.Kind()))
	}
}

// byteSlice returns its argument as a []byte if the argument's
// underlying type is []byte, along with a second return value of true.
// Otherwise it returns nil, false.
func byteSlice(a any) ([]byte, bool) {
	if bs, ok := a.([]byte); ok {
		return bs, true
	}
	// Like Printf's %s, we allow both the slice type and the byte element type to be named.
	t := reflect.TypeOf(a)
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return reflect.ValueOf(a).Bytes(), true
	}
	return nil, false
}

// This takes half the time of Time.AppendFormat.
func writeTimeRFC3339Millis(buf *buffer.Buffer, t time.Time) {
	year, month, day := t.Date()
	buf.WritePosIntWidth(year, 4)
	buf.WriteByte('-')
	buf.WritePosIntWidth(int(month), 2)
	buf.WriteByte('-')
	buf.WritePosIntWidth(day, 2)
	buf.WriteByte('T')
	hour, min, sec := t.Clock()
	buf.WritePosIntWidth(hour, 2)
	buf.WriteByte(':')
	buf.WritePosIntWidth(min, 2)
	buf.WriteByte(':')
	buf.WritePosIntWidth(sec, 2)
	ns := t.Nanosecond()
	buf.WriteByte('.')
	buf.WritePosIntWidth(ns/1e6, 3)
	_, offsetSeconds := t.Zone()
	if offsetSeconds == 0 {
		buf.WriteByte('Z')
	} else {
		offsetMinutes := offsetSeconds / 60
		if offsetMinutes < 0 {
			buf.WriteByte('-')
			offsetMinutes = -offsetMinutes
		} else {
			buf.WriteByte('+')
		}
		buf.WritePosIntWidth(offsetMinutes/60, 2)
		buf.WriteByte(':')
		buf.WritePosIntWidth(offsetMinutes%60, 2)
	}
}

func (s *handleState) appendTime(t time.Time) {
	if rep := s.h.opts.ReplaceAttr; rep != nil {
		var gs []string
		if s.groups != nil {
			gs = *s.groups
		}
		a := rep(gs, slog.Attr{Key: slog.TimeKey, Value: slog.TimeValue(t)})
		if a.Key == "" {
			return
		}

		v := a.Value.Resolve()
		if v.Kind() == slog.TimeKind {
			s.appendTimeMicro(v.Time())
		} else {
			s.appendValue(v)
		}
	}
}

func needsQuoting(s string) bool {
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if needsQuotingSet[b] {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || unicode.IsSpace(r) || !unicode.IsPrint(r) {
			return true
		}
		i += size
	}
	return false
}

var needsQuotingSet = [utf8.RuneSelf]bool{
	'"': true,
	'=': true,
}

func init() {
	for i := 0; i < utf8.RuneSelf; i++ {
		r := rune(i)
		if unicode.IsSpace(r) || !unicode.IsPrint(r) {
			needsQuotingSet[i] = true
		}
	}
}
