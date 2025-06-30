// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package branding

import "github.com/happy-sdk/happy/pkg/tui/ansicolor"

// New returns a new Brand Builder with the given slug.
func New(info Info) *Builder {
	return &Builder{
		brand: &Brand{
			info: info,
			ansi: ansicolor.New(),
		},
	}
}
