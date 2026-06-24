// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package analysisinternal provides gopls' internal analyses with a
// number of helper functions that operate on typed syntax trees.
package analysisinternal

// src: https://cs.opensource.google/go/x/tools/+/refs/tags/v0.36.0:internal/analysisinternal/analysis.go
//
// Only Format is used by this module (see passes/logging); the rest of the
// upstream file's helpers (AddImport, MatchingIdents, ValidateFixes, etc.)
// have no callers here and were removed rather than carried as dead code.

import (
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

// Format returns a string representation of the node n.
func Format(fset *token.FileSet, n ast.Node) string {
	var buf strings.Builder
	_ = printer.Fprint(&buf, fset, n)
	return buf.String()
}
