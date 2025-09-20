// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package typesinternal provides access to internal go/types APIs that are not
// yet exported.
package typesinternal

// src: https://cs.opensource.google/go/x/tools/+/refs/tags/v0.36.0:internal/typesinternal/types.go
import "go/types"

// IsPackageLevel reports whether obj is a package-level symbol.
func IsPackageLevel(obj types.Object) bool {
	return obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope()
}
