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
// The logging checker looks for calls to *github.com/happy-sdk/happy/pkg/logging.Logger
// methods that take alternating key-value pairs (Debug, Info, Warn, Error,
// their *Context variants, Log, and With). These methods are promoted from
// Logger's embedded *slog.Logger, so the checker follows the same rules as
// slog itself: it reports calls where an argument in a key position is
// neither a string nor a slog.Attr, and where a final key is missing its
// value. For example, it would report
//
//	logger.Warn("message", 11, "k") // ...Logger.Warn arg "11" should be a string or a slog.Attr
//
// and
//
//	logger.Info("message", "k1", v1, "k2") // call to ...Logger.Info missing a final value
package logging
