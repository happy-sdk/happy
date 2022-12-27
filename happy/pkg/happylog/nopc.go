// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

// Use the nopc flag for benchmarks, on the assumption
// that retrieving the pc will become cheap.

//go:build nopc

package happylog

// source
// https://github.com/golang/exp/blob/dc92f86530134df267cc7e7ea686d509f7ca1163/slog/nopc.go

// pc returns 0 to avoid incurring the cost of runtime.Callers.
func pc(depth int) uintptr { return 0 }
