// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package main

import (
	"github.com/happy-sdk/happy/tools/happyvet/passes/logging"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		logging.Analyzer,
	)
}
