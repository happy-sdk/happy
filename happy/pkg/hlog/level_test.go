// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package hlog

import (
	"testing"

	"golang.org/x/exp/slog"
)

func TestLevelString(t *testing.T) {
	for _, test := range []struct {
		in   Level
		want string
	}{
		{0, "info"},
		{LevelError, "error"},
		{LevelError + 2, "ERROR+2"},
		{LevelError - 2, "depr"},
		{LevelWarn, "warn"},
		{LevelWarn - 1, "notice"},
		{LevelInfo, "info"},
		{LevelInfo + 1, "task"},
		{LevelInfo - 3, "DEBUG+1"},
		{LevelDebug, "debug"},
		{LevelDebug - 2, "DEBUG-2"},
	} {
		got := test.in.String()
		if got != test.want {
			t.Errorf("%d: got %s, want %s", test.in, got, test.want)
		}
	}
}

func TestLevelVar(t *testing.T) {
	var al slog.LevelVar
	if got, want := al.Level(), levelInfo; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	al.Set(slog.Level(LevelWarn))
	if got, want := al.Level(), levelWarn; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	al.Set(slog.Level(LevelInfo))
	if got, want := al.Level(), levelInfo; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
