// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

// This file contains tests for the logging checker.

package a

import (
	"fmt"

	"github.com/happy-sdk/happy/pkg/logging"
)

// Used in tests by package b.
var MyLogger = logging.New(logging.DefaultConfig())

func F() {
	// Unrelated call.
	fmt.Println("ok")
}
