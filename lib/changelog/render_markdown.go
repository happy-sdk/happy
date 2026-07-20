// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

func (d *Document) renderMarkdown(o renderOptions) ([]byte, error) {
	if o.template != "" {
		tmpl, err := template.New("changelog-markdown").Parse(o.template)
		if err != nil {
			return nil, fmt.Errorf("changelog: parsing markdown template: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, d); err != nil {
			return nil, fmt.Errorf("changelog: executing markdown template: %w", err)
		}
		return buf.Bytes(), nil
	}
	return d.renderMarkdownDefault(), nil
}

// renderMarkdownDefault is the built-in Markdown template: a top-level
// title heading followed by one "### <section title>" heading per section,
// with a bulleted "**Breaking Changes**" list (if any) followed by a
// bulleted "**Changes**" list (if any).
func (d *Document) renderMarkdownDefault() []byte {
	var b strings.Builder

	fmt.Fprintf(&b, "## %s\n", d.Title)

	for _, s := range d.Sections {
		fmt.Fprintf(&b, "\n### %s\n", s.Title)
		if s.Release == nil {
			continue
		}
		if breaking := s.Release.Breaking(); len(breaking) > 0 {
			b.WriteString("\n**Breaking Changes**\n")
			for _, e := range breaking {
				fmt.Fprintf(&b, "* %s %s\n", e.ShortHash, e.Subject)
			}
		}
		if changes := s.Release.Entries(); len(changes) > 0 {
			b.WriteString("\n**Changes**\n")
			for _, e := range changes {
				fmt.Fprintf(&b, "* %s %s\n", e.ShortHash, e.Subject)
			}
		}
	}

	return []byte(b.String())
}
