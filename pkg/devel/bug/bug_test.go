// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package bug_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/bug"
	"github.com/happy-sdk/happy/pkg/logging"
)

func TestLogSlog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)
	bug.Log("test_bug")

	out := buf.String()
	if !strings.Contains(out, "test_bug") {
		t.Errorf("expected out to contain %q, got %q", "test_bug\n", out)
	}

	if !strings.Contains(out, "level=ERROR+1") {
		t.Errorf("expected out to contain %q, got %q", "level=ERROR+1\n", out)
	}
}

func TestLogLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewTextLogger(context.Background(), &buf, logging.DefaultOptions())
	slog.SetDefault(logger.Logger())

	bug.Log("test_bug", logger)
	out := buf.String()
	if !strings.Contains(out, "test_bug") {
		t.Errorf("expected out to contain %q, got %q", "test_bug\n", out)
	}

	if !strings.Contains(out, "level=bug") {
		t.Errorf("expected out to contain %q, got %q", "level=bug\n", out)
	}
}
