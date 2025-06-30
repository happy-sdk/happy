// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package bug

import (
	"context"
	"log/slog"
)

var logLevel = slog.Level(9)

func Log(msg string, args ...any) {
	slog.Log(context.Background(), logLevel, msg, args...)
}
