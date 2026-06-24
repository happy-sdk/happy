// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 The Happy Authors

package logging_test

import (
	"testing"

	"github.com/happy-sdk/happy/tools/happyvet/internal"
	"github.com/happy-sdk/happy/tools/happyvet/passes/logging"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	analysistest.Run(t, internal.TestData(), logging.Analyzer, "a")
}
