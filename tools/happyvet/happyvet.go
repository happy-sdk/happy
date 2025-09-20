// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package happyvet

import (
	"github.com/happy-sdk/happy/tools/happyvet/passes/logging"
	"golang.org/x/tools/go/analysis"
)

// New returns the list of Happy SDK analyzers.
func New(conf any) ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		logging.Analyzer,
	}, nil
}
