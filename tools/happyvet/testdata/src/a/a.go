// Package a is an analysistest fixture for the happy_logging analyzer.
package a

import (
	"happy/pkg/logging"
	"log/slog"
)

func _() {
	var l logging.Logger

	// Valid: alternating string key / value pairs.
	l.Info("msg", "k1", "v1", "k2", 2)

	// Valid: a slog.Attr in key position is self-contained.
	l.Info("msg", slog.String("k1", "v1"), "k2", 2)
	l.Error("msg", slog.String("k1", "v1"))
	l.Log(nil, slog.LevelInfo, "msg", slog.Int("k1", 1), "k2", "v2")

	// Invalid: a non-string, non-Attr key.
	l.Info("msg", 11, "k") // want `log/slog\.Logger\.Info arg "\\"k\\"" should be a string \(previous arg "11" cannot be a key\)`

	// Invalid: missing final value.
	l.Info("msg", "k1", "v1", "k2") // want `call to log/slog\.Logger\.Info missing a final value`

	// Invalid: a bad key followed by an orphaned value.
	l.Warn("msg", 11, "k", "v") // want `log/slog\.Logger\.Warn arg "\\"k\\"" should be a string \(previous arg "11" cannot be a key\)`

	// Valid: With takes alternating pairs too.
	_ = l.With("k1", "v1")

	// Invalid: ... args are skipped entirely (can't be checked statically).
	args := []any{"k1"}
	l.Info("msg", args...)
}
