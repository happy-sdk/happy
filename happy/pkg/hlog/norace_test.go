// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

//go:build !race

package hlog

import "testing"

// source
// https://github.com/golang/exp/blob/dc92f86530134df267cc7e7ea686d509f7ca1163/slog/race_test.go
func wantAllocs(t *testing.T, want int, f func()) {
	t.Helper()
	got := int(testing.AllocsPerRun(5, f))
	if got != want {
		t.Errorf("got %d allocs, want %d", got, want)
	}
}
