// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"flag"
	"strings"
	"testing"
)

// for slog compatibility tests are matching slog tests
// https://cs.opensource.google/go/x/exp/+/24139beb:slog/level_test.go
func TestLevelString(t *testing.T) {
	for _, test := range []struct {
		in   Level
		want string
	}{
		{LevelSystemDebug, "system"},
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelTask, "task"},
		{LevelOk, "ok"},
		{LevelNotice, "notice"},
		{LevelWarn, "warn"},
		{LevelNotImplemented, "notimpl"},
		{LevelDeprecated, "depr"},
		{LevelIssue, "issue"},
		{LevelError, "error"},
		{LevelBUG, "bug"},
		{LevelAlways, "msg"},
		{LevelQuiet, "quiet"},
		{0, "info"},
		{LevelError + 2, "error+2"},
		{LevelError - 2, "depr"},
		{LevelWarn - 1, "notice"},
		{LevelInfo + 1, "task"},
		{LevelInfo - 3, "debug+1"},
		{LevelDebug - 1, "system+5"},
		{LevelSystemDebug - 2, "system-2"},
	} {
		got := test.in.String()
		if got != test.want {
			t.Errorf("%d: got %s, want %s", test.in, got, test.want)
		}
	}
}

func TestLevelVar(t *testing.T) {
	var al levelVar
	if got, want := al.Level(), LevelInfo; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	al.Set(LevelWarn)
	if got, want := al.Level(), LevelWarn; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	al.Set(LevelInfo)
	if got, want := al.Level(), LevelInfo; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMarshalJSON(t *testing.T) {
	want := LevelWarn - 3
	data, err := want.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var got Level
	if err := got.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLevelMarshalText(t *testing.T) {
	want := LevelWarn - 3
	data, err := want.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var got Level
	if err := got.UnmarshalText(data); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLevelParse(t *testing.T) {
	for _, test := range []struct {
		in   string
		want Level
	}{
		{"system", LevelSystemDebug},
		{"SYSTEM", LevelSystemDebug},
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"task", LevelTask},
		{"TASK", LevelTask},
		{"ok", LevelOk},
		{"OK", LevelOk},
		{"notice", LevelNotice},
		{"NOTICE", LevelNotice},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"notimpl", LevelNotImplemented},
		{"NOTIMPL", LevelNotImplemented},
		{"depr", LevelDeprecated},
		{"DEPR", LevelDeprecated},
		{"issue", LevelIssue},
		{"ISSUE", LevelIssue},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"bug", LevelBUG},
		{"BUG", LevelBUG},
		{"msg", LevelAlways},
		{"MSG", LevelAlways},
		{"quiet", LevelQuiet},
		{"QUIET", LevelQuiet},
		{"iNfo", LevelInfo},
		{"INFO+87", LevelInfo + 87},
		{"Error-18", LevelError - 18},
		{"Error-8", LevelInfo},
	} {
		var got Level
		if err := got.parse(test.in); err != nil {
			t.Fatalf("%q: %v", test.in, err)
		}
		if got != test.want {
			t.Errorf("%q: got %s, want %s", test.in, got, test.want)
		}
	}
}

func TestLevelParseError(t *testing.T) {
	for _, test := range []struct {
		in   string
		want string // error string should contain this
	}{
		{"", "unknown level name"},
		{"dbg", "unknown level name"},
		{"INFO+", "invalid syntax"},
		{"INFO-", "invalid syntax"},
		{"ERROR+23x", "invalid syntax"},
	} {
		var l Level
		err := l.parse(test.in)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%q: got %v, want string containing %q", test.in, err, test.want)
		}

		var lv levelVar
		err2 := lv.UnmarshalText([]byte(test.in))
		if err2 == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%q: got %v, want string containing %q", test.in, err, test.want)
		}
	}
}

func TestLevelFlag(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	lf := LevelInfo
	fs.TextVar(&lf, "level", lf, "set level")
	err := fs.Parse([]string{"-level", "WARN+3"})
	if err != nil {
		t.Fatal(err)
	}
	if g, w := lf, LevelWarn+3; g != w {
		t.Errorf("got %v, want %v", g, w)
	}
}

func TestLevelVarMarshalText(t *testing.T) {
	var v levelVar
	v.Set(LevelWarn)
	data, err := v.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var v2 levelVar
	if err := v2.UnmarshalText(data); err != nil {
		t.Fatal(err)
	}
	if g, w := v2.Level(), LevelWarn; g != w {
		t.Errorf("got %s, want %s", g, w)
	}
}

func TestLevelVarFlag(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	v := &levelVar{}
	v.Set(LevelWarn + 3)
	fs.TextVar(v, "level", v, "set level")
	err := fs.Parse([]string{"-level", "WARN+3"})
	if err != nil {
		t.Fatal(err)
	}
	if g, w := v.Level(), LevelWarn+3; g != w {
		t.Errorf("got %v, want %v", g, w)
	}
}
