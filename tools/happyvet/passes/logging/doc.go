// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// Package logging defines an Analyzer that checks for
// mismatched key-value pairs in github.com/happy-sdk/happy/pkg/logging calls.
//
// # Analyzer logging
//
// logging: check for invalid structured logging calls
//
// The slog checker looks for calls to functions from the
// github.com/happy-sdk/happy/pkg/logging package that take
// alternating key-value pairs. It reports calls
// where an argument in a key position is neither a string nor a
// slog.Attr, and where a final key is missing its value.
// For example,it would report
//
//	logging.Warn("message", 11, "k") // logging.Warn arg "11" should be a string or a slog.Attr
//
// and
//
//	logging.Info("message", "k1", v1, "k2") // call to logging.Info missing a final value
package logging
