// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package sarg

import (
	"fmt"
	"time"

	"golang.org/x/exp/slog"
)

func GithubIssue(nr int, owner, repo string) slog.Attr {
	return slog.Group(
		"github",
		slog.Int("issue", nr),
		slog.String("url", fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, nr)),
	)
}

type Task struct {
	Name    string
	Started time.Time
	Args    []slog.Attr
}

func (t Task) LogAttr() slog.Attr {
	args := append([]slog.Attr{
		slog.String("name", t.Name),
		slog.Duration("elapsed", time.Since(t.Started)),
		slog.Time("started", t.Started),
	}, t.Args...)

	return slog.Group("task", args...)
}
