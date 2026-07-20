// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

// Package changelog composes and renders changelogs. A Release holds the
// changes for one release, either parsed from git log (ParseGitLog) or built
// up programmatically (NewRelease + Add/AddBreakingChange). A Document
// composes one or more titled Releases into a renderable unit, and can be
// rendered as Markdown, static HTML, JSON (with an accompanying JSON
// Schema), or PDF.
package changelog

// Section is a titled Release within a Document, e.g. "github.com/happy-sdk/happy@v1.2.0".
type Section struct {
	Title   string
	Release *Release
}

// Document composes one or more Sections into a single renderable changelog.
type Document struct {
	// Title is the overall document title/heading, e.g. "Changelog".
	Title    string
	Sections []Section
}

// NewDocument returns an empty Document with the given title.
func NewDocument(title string) *Document {
	return &Document{Title: title}
}

// AddSection appends a titled Release to the document and returns the
// Document so calls can be chained.
func (d *Document) AddSection(title string, release *Release) *Document {
	d.Sections = append(d.Sections, Section{Title: title, Release: release})
	return d
}

// Empty reports whether the document has no sections with any changes.
func (d *Document) Empty() bool {
	for _, s := range d.Sections {
		if s.Release != nil && !s.Release.Empty() {
			return false
		}
	}
	return true
}
