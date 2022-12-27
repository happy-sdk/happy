// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happylog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

// source
// https://github.com/golang/exp/blob/dc92f86530134df267cc7e7ea686d509f7ca1163/slog/handler_test.go

// Verify the common parts of TextHandler and JSONHandler.
func TestJSONAndTextHandlers(t *testing.T) {

	// ReplaceAttr functions

	// remove all Attrs
	removeAll := func(_ []string, a slog.Attr) slog.Attr { return slog.Attr{} }

	attrs := []slog.Attr{slog.String("a", "one"), slog.Int("b", 2), slog.Any("", "ignore me")}
	preAttrs := []slog.Attr{slog.Int("pre", 3), slog.String("x", "y")}

	for _, test := range []struct {
		name     string
		replace  func([]string, slog.Attr) slog.Attr
		with     func(slog.Handler) slog.Handler
		preAttrs []slog.Attr
		attrs    []slog.Attr
		wantText string
		wantJSON string
	}{
		{
			name:     "basic",
			attrs:    attrs,
			wantText: " info     03:04:05.000000 message a=one b=2",
			wantJSON: `{"time":"2000-01-02T03:04:05Z","level":"info","msg":"message","a":"one","b":2}`,
		},
		{
			name:     "cap keys",
			replace:  upperCaseKey,
			attrs:    attrs,
			wantText: " info     03:04:05.000000 message A=one B=2",
			wantJSON: `{"TIME":"2000-01-02T03:04:05Z","LEVEL":"INFO","MSG":"message","A":"one","B":2}`,
		},
		{
			name:     "remove all",
			replace:  removeAll,
			attrs:    attrs,
			wantText: " info     message ",
			wantJSON: `{}`,
		},
		{
			name:     "preformatted",
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: " info     03:04:05.000000 message pre=3 x=y a=one b=2",
			wantJSON: `{"time":"2000-01-02T03:04:05Z","level":"info","msg":"message","pre":3,"x":"y","a":"one","b":2}`,
		},
		{
			name:     "preformatted cap keys",
			replace:  upperCaseKey,
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: " info     03:04:05.000000 message PRE=3 X=y A=one B=2",
			wantJSON: `{"TIME":"2000-01-02T03:04:05Z","LEVEL":"INFO","MSG":"message","PRE":3,"X":"y","A":"one","B":2}`,
		},
		{
			name:     "preformatted remove all",
			replace:  removeAll,
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: " info     message ",
			wantJSON: "{}",
		},
		{
			name:     "remove built-in",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:    attrs,
			wantText: " info     message a=one b=2",
			wantJSON: `{"a":"one","b":2}`,
		},
		{
			name:     "preformatted remove built-in",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			attrs:    attrs,
			wantText: " info     message pre=3 x=y a=one b=2",
			wantJSON: `{"pre":3,"x":"y","a":"one","b":2}`,
		},
		{
			name:    "groups",
			replace: removeKeys(slog.TimeKey, slog.LevelKey), // to simplify the result
			attrs: []slog.Attr{
				slog.Int("a", 1),
				slog.Group("g",
					slog.Int("b", 2),
					slog.Group("h", slog.Int("c", 3)),
					slog.Int("d", 4)),
				slog.Int("e", 5),
			},
			wantText: " info     message a=1 g.b=2 g.h.c=3 g.d=4 e=5",
			wantJSON: `{"msg":"message","a":1,"g":{"b":2,"h":{"c":3},"d":4},"e":5}`,
		},
		{
			name:     "empty group",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{slog.Group("g"), slog.Group("h", slog.Int("a", 1))},
			wantText: " info     message  h.a=1",
			wantJSON: `{"msg":"message","g":{},"h":{"a":1}}`,
		},
		{
			name:    "escapes",
			replace: removeKeys(slog.TimeKey, slog.LevelKey),
			attrs: []slog.Attr{
				slog.String("a b", "x\t\n\000y"),
				slog.Group(" b.c=\"\\x2E\t",
					slog.String("d=e", "f.g\""),
					slog.Int("m.d", 1)), // dot is not escaped
			},
			wantText: ` info     message "a b"="x\t\n\x00y" " b.c=\"\\x2E\t.d=e"="f.g\"" " b.c=\"\\x2E\t.m.d"=1`,
			wantJSON: `{"msg":"message","a b":"x\t\n\u0000y"," b.c=\"\\x2E\t":{"d=e":"f.g\"","m.d":1}}`,
		},
		{
			name:    "LogValuer",
			replace: removeKeys(slog.TimeKey, slog.LevelKey),
			attrs: []slog.Attr{
				slog.Int("a", 1),
				slog.Any("name", logValueName{"Ren", "Hoek"}),
				slog.Int("b", 2),
			},
			wantText: " info     message a=1 name.first=Ren name.last=Hoek b=2",
			wantJSON: `{"msg":"message","a":1,"name":{"first":"Ren","last":"Hoek"},"b":2}`,
		},
		{
			name:     "with-group",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs).WithGroup("s") },
			attrs:    attrs,
			wantText: " info     message pre=3 x=y s.a=one s.b=2",
			wantJSON: `{"msg":"message","pre":3,"x":"y","s":{"a":"one","b":2}}`,
		},
		{
			name:    "preformatted with-groups",
			replace: removeKeys(slog.TimeKey, slog.LevelKey),
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{slog.Int("p1", 1)}).
					WithGroup("s1").
					WithAttrs([]slog.Attr{slog.Int("p2", 2)}).
					WithGroup("s2")
			},
			attrs:    attrs,
			wantText: " info     message p1=1 s1.p2=2 s1.s2.a=one s1.s2.b=2",
			wantJSON: `{"msg":"message","p1":1,"s1":{"p2":2,"s2":{"a":"one","b":2}}}`,
		},
		{
			name:    "two with-groups",
			replace: removeKeys(slog.TimeKey, slog.LevelKey),
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{slog.Int("p1", 1)}).
					WithGroup("s1").
					WithGroup("s2")
			},
			attrs:    attrs,
			wantText: " info     message p1=1 s1.s2.a=one s1.s2.b=2",
			wantJSON: `{"msg":"message","p1":1,"s1":{"s2":{"a":"one","b":2}}}`,
		},
		{
			name:     "GroupValue as Attr value",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{{"v", slog.AnyValue(slog.IntValue(3))}},
			wantText: " info     message v=3",
			wantJSON: `{"msg":"message","v":3}`,
		},
		{
			name:     "byte slice",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{slog.Any("bs", []byte{1, 2, 3, 4})},
			wantText: ` info     message bs="\x01\x02\x03\x04"`,
			wantJSON: `{"msg":"message","bs":"AQIDBA=="}`,
		},
		{
			name:     "json.RawMessage",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{slog.Any("bs", json.RawMessage([]byte("1234")))},
			wantText: ` info     message bs="1234"`,
			wantJSON: `{"msg":"message","bs":1234}`,
		},
	} {
		r := slog.NewRecord(testTime, slog.LevelInfo, "message", 1, nil)
		r.AddAttrs(test.attrs...)
		var buf bytes.Buffer
		opts := slog.HandlerOptions{ReplaceAttr: test.replace}
		t.Run(test.name, func(t *testing.T) {
			for _, handler := range []struct {
				name string
				h    slog.Handler
				want string
			}{
				{"text", Config{Options: opts}.NewHandler(&buf), test.wantText},
				{"json", Config{Options: opts, JSON: true}.NewHandler(&buf), test.wantJSON},
			} {
				t.Run(handler.name, func(t *testing.T) {
					h := handler.h
					if test.with != nil {
						h = test.with(h)
					}
					buf.Reset()
					if err := h.Handle(r); err != nil {
						t.Fatal(err)
					}
					got := strings.TrimSuffix(buf.String(), "\n")
					if got != handler.want {
						t.Errorf("\ngot  %s\nwant %s\n", got, handler.want)
					}
				})
			}
		})
	}
}

// removeKeys returns a function suitable for HandlerOptions.ReplaceAttr
// that removes all Attrs with the given keys.
func removeKeys(keys ...string) func([]string, slog.Attr) slog.Attr {
	return func(_ []string, a slog.Attr) slog.Attr {
		for _, k := range keys {
			if a.Key == k {
				return slog.Attr{}
			}
		}
		return a
	}
}

type logValueName struct {
	first, last string
}

func (n logValueName) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("first", n.first),
		slog.String("last", n.last))
}

func TestHandler(t *testing.T) {
	for _, test := range []struct {
		name             string
		attr             slog.Attr
		wantKey, wantVal string
	}{
		{
			"unquoted",
			slog.Int("a", 1),
			"a", "1",
		},
		{
			"quoted",
			slog.String("x = y", `qu"o`),
			`"x = y"`, `"qu\"o"`,
		},
		{
			"Sprint",
			slog.Any("name", name{"Ren", "Hoek"}),
			`name`, `"Hoek, Ren"`,
		},
		{
			"TextMarshaler",
			slog.Any("t", text{"abc"}),
			`t`, `"text{\"abc\"}"`,
		},
		{
			"TextMarshaler error",
			slog.Any("t", text{""}),
			`t`, `"!ERROR:text: empty string"`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, opts := range []struct {
				name       string
				opts       Config
				wantPrefix string
				modKey     func(string) string
			}{
				{
					"none",
					Config{},
					` info     03:04:05.000000 "a message"`,
					func(s string) string { return s },
				},
				{
					"replace",
					Config{Options: slog.HandlerOptions{ReplaceAttr: upperCaseKey}},
					` info     03:04:05.000000 "a message"`,
					strings.ToUpper,
				},
			} {
				t.Run(opts.name, func(t *testing.T) {
					var buf bytes.Buffer
					h := opts.opts.NewHandler(&buf)
					r := slog.NewRecord(testTime, levelInfo, "a message", 0, nil)
					r.AddAttrs(test.attr)
					if err := h.Handle(r); err != nil {
						t.Fatal(err)
					}
					got := buf.String()
					// Remove final newline.
					got = got[:len(got)-1]
					want := opts.wantPrefix + " " + opts.modKey(test.wantKey) + "=" + test.wantVal
					if got != want {
						t.Errorf("\ngot  %s\nwant %s", got, want)
					}
				})
			}
		})
	}
}

func TestHandlerPreformatted(t *testing.T) {
	var buf bytes.Buffer
	var h slog.Handler = NewHandler(&buf)
	h = h.WithAttrs([]slog.Attr{slog.Duration("dur", time.Minute), slog.Bool("b", true)})
	// Also test omitting time.
	r := slog.NewRecord(time.Time{}, 0 /* 0 Level is INFO */, "m", 0, nil)
	r.AddAttrs(slog.Int("a", 1))
	if err := h.Handle(r); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSuffix(buf.String(), "\n")
	want := ` info     m dur=1m0s b=true a=1`
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// Should get these allocs to 0
func TestHandlerAlloc(t *testing.T) {
	r := slog.NewRecord(time.Now(), levelInfo, "msg", 0, nil)
	for i := 0; i < 10; i++ {
		r.AddAttrs(slog.Int("x = y", i))
	}
	var h slog.Handler = NewHandler(io.Discard)
	wantAllocs(t, 2, func() { h.Handle(r) })

	h = h.WithGroup("s")
	r.AddAttrs(slog.Group("g", slog.Int("a", 1)))
	wantAllocs(t, 2, func() { h.Handle(r) })
}

func upperCaseKey(_ []string, a slog.Attr) slog.Attr {
	a.Key = strings.ToUpper(a.Key)
	return a
}

var testTime = time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)

// for testing fmt.Sprint
type name struct {
	First, Last string
}

func (n name) String() string { return n.Last + ", " + n.First }

// for testing TextMarshaler
type text struct {
	s string
}

func (t text) String() string { return t.s } // should be ignored

func (t text) MarshalText() ([]byte, error) {
	if t.s == "" {
		return nil, errors.New("text: empty string")
	}
	return []byte(fmt.Sprintf("text{%q}", t.s)), nil
}

func TestHandlerEnabled(t *testing.T) {
	levelVar := func(l Level) *slog.LevelVar {
		var al slog.LevelVar
		al.Set(slog.Level(l))
		return &al
	}

	for _, test := range []struct {
		leveler slog.Leveler
		want    bool
	}{
		{nil, true},
		{levelWarn, false},
		{&slog.LevelVar{}, true}, // defaults to Info
		{levelVar(LevelWarn), false},
		{levelDebug, true},
		{levelVar(LevelDebug), true},
	} {
		h := Config{Options: slog.HandlerOptions{Level: test.leveler}}.NewHandler(nil)
		got := h.Enabled(slog.LevelInfo)
		if got != test.want {
			t.Errorf("%v: got %t, want %t", test.leveler, got, test.want)
		}
	}
}

func TestSecondWith(t *testing.T) {
	// Verify that a second call to Logger.With does not corrupt
	// the original.
	var buf bytes.Buffer
	h := Config{Options: slog.HandlerOptions{ReplaceAttr: removeKeys(slog.TimeKey)}}.NewHandler(&buf)
	logger := New(h).With(
		slog.String("app", "playground"),
		slog.String("role", "tester"),
		slog.Int("data_version", 2),
	)
	appLogger := logger.With("type", "log") // this becomes type=met
	_ = logger.With("type", "metric")
	appLogger.Info("foo")
	got := strings.TrimSpace(buf.String())
	want := `info     foo app=playground role=tester data_version=2 type=log`
	if got != want {
		t.Errorf("\ngot  %s\nwant %s", got, want)
	}
}

func TestReplaceAttrGroups(t *testing.T) {
	// Verify that ReplaceAttr is called with the correct groups.
	type ga struct {
		groups string
		key    string
		val    string
	}

	var got []ga

	h := Config{Options: slog.HandlerOptions{ReplaceAttr: func(gs []string, a slog.Attr) slog.Attr {
		v := a.Value.String()
		if a.Key == slog.TimeKey {
			v = "<now>"
		}
		got = append(got, ga{strings.Join(gs, ","), a.Key, v})
		return a
	}}}.NewHandler(io.Discard)
	New(h).
		With(slog.Int("a", 1)).
		WithGroup("g1").
		With(slog.Int("b", 2)).
		WithGroup("g2").
		With(
			slog.Int("c", 3),
			slog.Group("g3", slog.Int("d", 4)),
			slog.Int("e", 5)).
		Warn("mdddddddddddddddddd",
			slog.Int("f", 6),
			slog.Group("g4", slog.Int("h", 7)),
			slog.Int("i", 8))

	want := []ga{
		{"", "a", "1"},
		{"g1", "b", "2"},
		{"g1,g2", "c", "3"},
		{"g1,g2,g3", "d", "4"},
		{"g1,g2", "e", "5"},
		{"", "time", "<now>"},
		{"g1,g2", "f", "6"},
		{"g1,g2,g4", "h", "7"},
		{"g1,g2", "i", "8"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("\ngot  %v\nwant %v", got, want)
	}
}
