// Copyright 2023 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"time"

	"golang.org/x/exp/slog"
)

// TaskInfo is used to log task info
type TaskInfo struct {
	// Name of the task
	Name string
	// Description of the task
	Description string

	startedAt time.Time
}

// Failed returns slog.Attr for task failed
func (ti TaskInfo) Failed() slog.Attr {
	return slog.Group(
		"task",
		slog.String("name", ti.Name),
		slog.String("status", "failed"),
		slog.Duration("elapsed", time.Since(ti.startedAt)),
	)
}

// Done returns slog.Attr for task done
func (ti TaskInfo) Success() slog.Attr {
	return slog.Group(
		"task",
		slog.String("name", ti.Name),
		slog.String("status", "success"),
		slog.Duration("elapsed", time.Since(ti.startedAt)),
	)
}
