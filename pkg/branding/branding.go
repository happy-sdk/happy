// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

// Package branding provides a builder for an application's identity
// (Brand): its name, version, slug, description, terminal color theme, and
// color palette.
package branding

import "github.com/happy-sdk/happy/pkg/tui/ansicolor"

// New returns a new Brand Builder seeded with the given Info. Call Build to
// validate and obtain the resulting *Brand.
func New(info Info) *Builder {
	return &Builder{
		brand: &Brand{
			info: info,
			ansi: ansicolor.New(),
		},
	}
}
