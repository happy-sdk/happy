// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

// Package bexp implements Brace Expansion mechanism to generate arbitrary strings.
// Deprecated: This module is no longer maintained.
// Development has moved to github.com/happy-sdk/happy-go/strings/bexp.
// Users are encouraged to use the new module location for future updates and bug fixes.
package bexp

import "time"

const (
	// Deprecated is a marker for deprecated code.
	Deprecated = true

	// DeprecatedBy is the name entity who deprecated this package.
	DeprecatedBy = "The Happy Authors"

	// NewLocation is the new location of this package.
	NewLocation = "github.com/happy-sdk/happy-go/strings/bexp"
)

// DeprecatedAt is the date when this package was deprecated.
func DeprecatedAt() time.Time {
	return time.Date(2023, time.December, 27, 14, 25, 0, 0, time.UTC)
}
