// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package branding

import (
	"github.com/happy-sdk/happy/pkg/tui/ansicolor"
)

// Brand is an application's identity: its Info, terminal color theme, and
// color palette. Construct one via New(...).Build().
type Brand struct {
	info   Info
	ansi   ansicolor.Theme
	colors ColorPalette
}

// Info identifies an application: its display name, version, slug (a
// URL/filesystem-safe identifier, see pkg/strings/slug), and a short
// description.
type Info struct {
	Name        string
	Version     string
	Slug        string
	Description string
}

// Info returns the brand's Info.
func (b *Brand) Info() Info {
	return b.info
}

// ANSI returns the brand's terminal color theme, used to render colored CLI
// output (help text, banners, etc).
func (b *Brand) ANSI() ansicolor.Theme {
	return b.ansi
}

// Colors returns the brand's color palette.
func (b *Brand) Colors() ColorPalette {
	return b.colors
}

// ColorPalette each Color specifies a color by hex or ANSI value. For example:
//
//	ansiColor := colorPalette.Primary = "21"
//	hexColor := colorPalette.Primary = "#0000ff"
type ColorPalette struct {
	Primary   string
	Secondary string
	Accent    string
	Highlight string
	Info      string
	Success   string
	Warning   string
	Danger    string
}
