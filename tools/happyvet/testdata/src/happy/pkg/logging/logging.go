// Package logging is a minimal stand-in for
// github.com/happy-sdk/happy/pkg/logging, used only by analysistest
// fixtures: it embeds *slog.Logger the same way the real Logger does, so
// the analyzer's handling of promoted methods can be tested without
// depending on the real module.
package logging

import "log/slog"

type Logger struct {
	*slog.Logger
}
