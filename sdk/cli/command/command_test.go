// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package command

import (
	"context"
	"log/slog"
	"testing"
)

// recordingHandler is a minimal slog.Handler that records whether it
// received any log record, used to detect leakage to the global slog
// default logger.
type recordingHandler struct {
	called bool
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *recordingHandler) Handle(context.Context, slog.Record) error {
	h.called = true
	return nil
}
func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(name string) slog.Handler       { return h }

// TestToInvalidDoesNotLeakToGlobalSlog is a regression test: toInvalid's
// deferred logging called the global slog.Error(...) directly, bypassing
// the application's configured logging entirely and unconditionally
// writing to the global slog default logger (and therefore, by default,
// to stderr) regardless of the app's log level/output. It now logs
// through the command's own queue logger (c.cnflog) instead, consistent
// with error()'s convention elsewhere in this package.
func TestToInvalidDoesNotLeakToGlobalSlog(t *testing.T) {
	prev := slog.Default()
	rec := &recordingHandler{}
	slog.SetDefault(slog.New(rec))
	defer slog.SetDefault(prev)

	// An invalid command name (starting with an uppercase letter/digit,
	// per varflag's FlagRe) fails FlagSet creation in New, which triggers
	// toInvalid().
	cmd := New("Invalid_Name", Config{})

	if cmd.Err() == nil {
		t.Fatal("expected New to mark the command invalid for a bad name")
	}
	if rec.called {
		t.Error("expected no log records on the global slog default logger, but toInvalid logged to it")
	}
}

func TestNewValidCommand(t *testing.T) {
	cmd := New("valid-name", Config{Description: "a valid command"})
	if cmd.Err() != nil {
		t.Fatalf("expected no error for a valid command, got %v", cmd.Err())
	}
	if cmd.Name() != "valid-name" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "valid-name")
	}
}
