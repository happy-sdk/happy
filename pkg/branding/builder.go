// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package branding

import "github.com/happy-sdk/happy/pkg/tui/ansicolor"

type Builder struct {
	brand *Brand
}

// Build returns the configured Brand. The error return exists for forward
// compatibility (e.g. future validation); it is currently always nil. An
// empty Info (no name, no slug) is accepted, since some bootstrap paths
// (e.g. under testing.Testing()) construct a Brand before a name or slug
// has been resolved.
func (b *Builder) Build() (*Brand, error) {
	return b.brand, nil
}

func (b *Builder) WithANSI(ansi ansicolor.Theme) *Builder {
	b.brand.ansi = ansi
	return b
}

func (b *Builder) WithPalette(palette ColorPalette) *Builder {
	b.brand.colors = palette
	return b
}
