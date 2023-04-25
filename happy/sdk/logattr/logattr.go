// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

// Package logattr is providing some common slog.Attr to use in your logging.
package logattr

import (
	"fmt"
	"time"

	"golang.org/x/exp/slog"
)

func GithubIssue(no int, owner, repo string) slog.Attr {
	return slog.Group(
		"github",
		slog.Int("issue", no),
		slog.String("url", fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, no)),
	)
}

func GithubPR(pr int, owner, repo string) slog.Attr {
	return slog.Group(
		"github",
		slog.Int("pr", pr),
		slog.String("url", fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, pr)),
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
