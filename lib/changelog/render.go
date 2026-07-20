// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import "fmt"

// Format selects which renderer Document.Render uses.
type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
	FormatJSON     Format = "json"
	FormatPDF      Format = "pdf"
)

type renderOptions struct {
	template string
}

// RenderOption customizes a single Document.Render call.
type RenderOption func(*renderOptions)

// WithTemplate supplies a custom template source for the Markdown or HTML
// renderers, overriding their built-in default template. It has no effect
// on FormatJSON or FormatPDF.
func WithTemplate(src string) RenderOption {
	return func(o *renderOptions) { o.template = src }
}

// Render produces the document in the requested Format.
func (d *Document) Render(format Format, opts ...RenderOption) ([]byte, error) {
	var o renderOptions
	for _, opt := range opts {
		opt(&o)
	}

	switch format {
	case FormatMarkdown:
		return d.renderMarkdown(o)
	case FormatHTML:
		return d.renderHTML(o)
	case FormatJSON:
		return d.renderJSON()
	case FormatPDF:
		return d.renderPDF()
	default:
		return nil, fmt.Errorf("changelog: unsupported format %q", format)
	}
}
