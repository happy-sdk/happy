// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestDocumentEmpty(t *testing.T) {
	d := NewDocument("Changelog")
	testutils.Assert(t, d.Empty(), "expected a document with no sections to be empty")

	d.AddSection("github.com/happy-sdk/happy/pkg/idle@v0.1.0", NewRelease())
	testutils.Assert(t, d.Empty(), "expected a document whose only section has an empty release to be empty")

	r := NewRelease()
	r.Add("h", "h", "a", "add a feature", EntryType{Typ: "feat", Kind: EntryKindMinor})
	d.AddSection("github.com/happy-sdk/happy@v1.1.0", r)
	testutils.Assert(t, !d.Empty(), "expected a document with a non-empty release to be non-empty")
}

func TestDocumentAddSectionChains(t *testing.T) {
	d := NewDocument("Changelog").
		AddSection("a", NewRelease()).
		AddSection("b", NewRelease())
	testutils.Equal(t, 2, len(d.Sections), "expected AddSection to be chainable")
}

// fixtureDocument builds a small, deterministic multi-section document
// reused by the render_*_test.go files.
func fixtureDocument() *Document {
	root := NewRelease()
	root.Add("aaa1111", "aaa1111full", "Jane Doe", "add a new feature", EntryType{Typ: "feat", Kind: EntryKindMinor})
	root.AddBreakingChange("bbb2222", "bbb2222full", "Jane Doe", "removed deprecated Foo()")

	sub := NewRelease()
	sub.Add("ccc3333", "ccc3333full", "John Roe", "fix a bug in sub", EntryType{Typ: "fix", Kind: EntryKindPatch})

	return NewDocument("Changelog").
		AddSection("github.com/happy-sdk/happy@v1.2.0", root).
		AddSection("github.com/happy-sdk/happy/pkg/sub@v0.2.0", sub)
}
