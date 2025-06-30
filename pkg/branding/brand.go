// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package branding

import (
	"github.com/happy-sdk/happy/pkg/tui/ansicolor"
)

type Brand struct {
	info   Info
	ansi   ansicolor.Theme
	colors ColorPalette
}

type Info struct {
	Name        string
	Version     string
	Slug        string
	Description string
}

func (b *Brand) Info() Info {
	return b.info
}

func (b *Brand) ANSI() ansicolor.Theme {
	return b.ansi
}

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
