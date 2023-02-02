// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

//go:build race

package hlog

import "testing"

func wantAllocs(t *testing.T, want int, f func()) {
	t.Log("skipping allocation tests with race detector")
}
