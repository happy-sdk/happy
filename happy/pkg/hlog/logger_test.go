// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package hlog

import (
	"bytes"
	"context"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

// source
// https://github.com/golang/exp/blob/dc92f86530134df267cc7e7ea686d509f7ca1163/slog/logger_test.go

func TestLogHandler(t *testing.T) {
	var buf bytes.Buffer

	removeTime := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}

	lvl := &slog.LevelVar{}
	l := New(Config{
		Options: slog.HandlerOptions{
			ReplaceAttr: removeTime,
			Level:       lvl,
		},
	}.NewHandler(&buf))

	check := func(want string) {
		t.Helper()
		checkLogOutput(t, buf.String(), want)
		buf.Reset()
	}

	l.Info("msg", "a", 1, "b", 2)
	check(` info     msg a=1 b=2`)

	// By default, debug messages are not printed.
	l.SystemDebug("bg", slog.Int("a", 1), "b", 2)
	check("")
	l.Debug("bg", slog.Int("a", 1), "b", 2)
	check("")

	lvl.Set(levelSystemDebug)
	SetDefault(l, false)

	SystemDebug("bg", slog.Int("a", 1), "b", 2)
	check(" system   bg a=1 b=2")
	Debug("bg", slog.Int("a", 1), "b", 2)
	check(" debug    bg a=1 b=2")

	Task("t", slog.Duration("dur", 3*time.Second))
	check(` task      task.name=t task.dur=3s`)
	l.Task("t", slog.Duration("dur", 3*time.Second))
	check(` task      task.name=t task.dur=3s`)

	Ok("o", slog.Duration("dur", 3*time.Second))
	check(` ok       o dur=3s`)
	l.Ok("o", slog.Duration("dur", 3*time.Second))
	check(` ok       o dur=3s`)

	Notice("n", slog.Duration("dur", 3*time.Second))
	check(` notice   n dur=3s`)
	l.Notice("n", slog.Duration("dur", 3*time.Second))
	check(` notice   n dur=3s`)

	NotImplemented("w", slog.Int("a", 1), slog.String("b", "two"))
	check(` notimpl  w a=1 b=two`)
	l.NotImplemented("w", slog.Int("a", 1), slog.String("b", "two"))
	check(` notimpl  w a=1 b=two`)

	Deprecated("d", slog.Int("a", 1), slog.String("b", "two"))
	check(` depr     d a=1 b=two`)
	l.Deprecated("d", slog.Int("a", 1), slog.String("b", "two"))
	check(` depr     d a=1 b=two`)

	Issue("i", slog.Int("a", 1), slog.String("b", "two"))
	check(` issue    i a=1 b=two`)
	l.Issue("i", slog.Int("a", 1), slog.String("b", "two"))
	check(` issue    i a=1 b=two`)

	Warn("w", slog.Duration("dur", 3*time.Second))
	check(` warn     w dur=3s`)
	l.Warn("w", slog.Duration("dur", 3*time.Second))
	check(` warn     w dur=3s`)

	Out("o", slog.Duration("dur", 3*time.Second))
	check(`o dur=3s`)
	l.Out("o", slog.Duration("dur", 3*time.Second))
	check(`o dur=3s`)

	Error("bad", io.EOF, "a", 1)
	check(` error    bad err=EOF a=1`)
	l.Error("bad", io.EOF, "a", 1)
	check(` error    bad err=EOF a=1`)

	l.LogAttrs(LevelTask, "a b c", slog.Int("a", 1), slog.String("b", "two"))
	check(` task     "a b c" a=1 b=two`)

	l.Info("info", "a", []slog.Attr{slog.Int("i", 1)})
	check(` info     info a.i=1`)

	l.Info("info", "a", slog.GroupValue(slog.Int("i", 1)))
	check(` info     info a.i=1`)
}

type wrappingHandler struct {
	h slog.Handler
}

func (h wrappingHandler) Enabled(level slog.Level) bool         { return h.h.Enabled(level) }
func (h wrappingHandler) WithGroup(name string) slog.Handler    { return h.h.WithGroup(name) }
func (h wrappingHandler) WithAttrs(as []slog.Attr) slog.Handler { return h.h.WithAttrs(as) }
func (h wrappingHandler) Handle(r slog.Record) error            { return h.h.Handle(r) }

func TestAttrs(t *testing.T) {
	check := func(got []slog.Attr, want ...slog.Attr) {
		t.Helper()
		if !attrsEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	l1 := New(&captureHandler{}).With("a", 1)
	l2 := New(l1.Handler()).With("b", 2)
	l2.Info("m", "c", 3)
	h := l2.Handler().(*captureHandler)
	check(h.attrs, slog.Int("a", 1), slog.Int("b", 2))
	check(attrsSlice(h.r), slog.Int("c", 3))
}

func TestCallDepth(t *testing.T) {
	h := &captureHandler{}
	var startLine int

	check := func(count int) {
		t.Helper()
		const wantFile = "logger_test.go"
		wantLine := startLine + count*2
		gotFile, gotLine := h.r.SourceLine()
		gotFile = filepath.Base(gotFile)
		if gotFile != wantFile || gotLine != wantLine {
			t.Errorf("got (%s, %d), want (%s, %d)", gotFile, gotLine, wantFile, wantLine)
		}
	}

	logger := New(h)
	SetDefault(logger, false)

	// Calls to check must be one line apart.
	// Determine line where calls start.
	f, _ := runtime.CallersFrames([]uintptr{pc(2)}).Next()
	startLine = f.Line + 4
	// Do not change the number of lines between here and the call to check(0).

	logger.Log(LevelInfo, "")
	check(0)
	logger.LogAttrs(LevelInfo, "")
	check(1)
	logger.Debug("")
	check(2)
	logger.Info("")
	check(3)
	logger.Warn("")
	check(4)
	logger.Error("", nil)
	check(5)
	Debug("")
	check(6)
	Info("")
	check(7)
	Warn("")
	check(8)
	Error("", nil)
	check(9)
	Log(LevelInfo, "")
	check(10)
	LogAttrs(LevelInfo, "")
	check(11)
}

func TestAlloc(t *testing.T) {
	dl := New(discardHandler{})
	defer func(d *Logger) { SetDefault(d, false) }(Default())
	SetDefault(dl, false)

	t.Run("Info", func(t *testing.T) {
		wantAllocs(t, 0, func() { Info("hello") })
	})
	t.Run("Error", func(t *testing.T) {
		wantAllocs(t, 2, func() { Error("hello", io.EOF) })
	})
	t.Run("logger.Info", func(t *testing.T) {
		wantAllocs(t, 0, func() { dl.Info("hello") })
	})
	t.Run("logger.Log", func(t *testing.T) {
		wantAllocs(t, 0, func() { dl.Log(LevelDebug, "hello") })
	})
	t.Run("2 pairs", func(t *testing.T) {
		s := "abc"
		i := 2000
		wantAllocs(t, 2, func() {
			dl.Info("hello",
				"n", i,
				"s", s,
			)
		})
	})
	t.Run("2 pairs disabled inline", func(t *testing.T) {
		l := New(discardHandler{disabled: true})
		s := "abc"
		i := 2000
		wantAllocs(t, 2, func() {
			l.Log(LevelInfo, "hello",
				"n", i,
				"s", s,
			)
		})
	})
	t.Run("2 pairs disabled", func(t *testing.T) {
		l := New(discardHandler{disabled: true})
		s := "abc"
		i := 2000
		wantAllocs(t, 0, func() {
			if l.Enabled(LevelInfo) {
				l.Log(LevelInfo, "hello",
					"n", i,
					"s", s,
				)
			}
		})
	})
	t.Run("9 kvs", func(t *testing.T) {
		s := "abc"
		i := 2000
		d := time.Second
		wantAllocs(t, 11, func() {
			dl.Info("hello",
				"n", i, "s", s, "d", d,
				"n", i, "s", s, "d", d,
				"n", i, "s", s, "d", d)
		})
	})
	t.Run("pairs", func(t *testing.T) {
		wantAllocs(t, 0, func() { dl.Info("", "error", io.EOF) })
	})
	t.Run("attrs1", func(t *testing.T) {
		wantAllocs(t, 0, func() { dl.LogAttrs(LevelInfo, "", slog.Int("a", 1)) })
		wantAllocs(t, 0, func() { dl.LogAttrs(LevelInfo, "", slog.Any("error", io.EOF)) })
	})
	t.Run("attrs3", func(t *testing.T) {
		wantAllocs(t, 0, func() {
			dl.LogAttrs(LevelInfo, "hello", slog.Int("a", 1), slog.String("b", "two"), slog.Duration("c", time.Second))
		})
	})
	t.Run("attrs3 disabled", func(t *testing.T) {
		logger := New(discardHandler{disabled: true})
		wantAllocs(t, 0, func() {
			logger.LogAttrs(LevelInfo, "hello", slog.Int("a", 1), slog.String("b", "two"), slog.Duration("c", time.Second))
		})
	})
	t.Run("attrs6", func(t *testing.T) {
		wantAllocs(t, 1, func() {
			dl.LogAttrs(LevelInfo, "hello",
				slog.Int("a", 1), slog.String("b", "two"), slog.Duration("c", time.Second),
				slog.Int("d", 1), slog.String("e", "two"), slog.Duration("f", time.Second))
		})
	})
	t.Run("attrs9", func(t *testing.T) {
		wantAllocs(t, 1, func() {
			dl.LogAttrs(LevelInfo, "hello",
				slog.Int("a", 1), slog.String("b", "two"), slog.Duration("c", time.Second),
				slog.Int("d", 1), slog.String("e", "two"), slog.Duration("f", time.Second),
				slog.Int("d", 1), slog.String("e", "two"), slog.Duration("f", time.Second))
		})
	})
}

func TestSetDefault(t *testing.T) {
	// Verify that setting the default to itself does not result in deadlock.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer func(w io.Writer) { log.SetOutput(w) }(log.Writer())
	log.SetOutput(io.Discard)
	go func() {
		Info("A")
		SetDefault(Default(), true)
		Info("B")
		cancel()
	}()
	<-ctx.Done()
	if err := ctx.Err(); err != context.Canceled {
		t.Errorf("wanted canceled, got %v", err)
	}
}

func TestLoggerError(t *testing.T) {
	var buf bytes.Buffer

	removeTime := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}

	l := New(Config{
		Options: slog.HandlerOptions{
			ReplaceAttr: removeTime,
		},
	}.NewHandler(&buf))

	l.Error("msg", io.EOF, "a", 1)
	checkLogOutput(t, buf.String(), ` error    msg err=EOF a=1`)

	buf.Reset()
	l.Error("msg", io.EOF, "a")
	checkLogOutput(t, buf.String(), ` error    msg err=EOF !BADKEY=a`)
}

func checkLogOutput(t *testing.T, got, wantRegexp string) {
	t.Helper()
	got = clean(got)
	wantRegexp = "^" + wantRegexp + "$"
	matched, err := regexp.MatchString(wantRegexp, got)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("\ngot  %s\nwant %s", got, wantRegexp)
	}
}

// clean prepares log output for comparison.
func clean(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return strings.ReplaceAll(s, "\n", "~")
}

type captureHandler struct {
	r     slog.Record
	attrs []slog.Attr
}

func (h *captureHandler) Handle(r slog.Record) error {
	h.r = r
	return nil
}

func (*captureHandler) Enabled(slog.Level) bool { return true }

func (c *captureHandler) WithAttrs(as []slog.Attr) slog.Handler {
	c2 := *c
	c2.attrs = concat(c2.attrs, as)
	return &c2
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	panic("unimplemented")
}

type discardHandler struct {
	disabled bool
	attrs    []slog.Attr
}

func (d discardHandler) Enabled(slog.Level) bool { return !d.disabled }
func (discardHandler) Handle(slog.Record) error  { return nil }
func (d discardHandler) WithAttrs(as []slog.Attr) slog.Handler {
	d.attrs = concat(d.attrs, as)
	return d
}
func (h discardHandler) WithGroup(name string) slog.Handler {
	return h
}

// concat returns a new slice with the elements of s1 followed
// by those of s2. The slice has no additional capacity.
func concat[T any](s1, s2 []T) []T {
	s := make([]T, len(s1)+len(s2))
	copy(s, s1)
	copy(s[len(s1):], s2)
	return s
}

func attrsEqual(as1, as2 []slog.Attr) bool {
	return slices.EqualFunc(as1, as2, slog.Attr.Equal)
}

func attrsSlice(r slog.Record) []slog.Attr {
	s := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) { s = append(s, a) })
	return s
}
