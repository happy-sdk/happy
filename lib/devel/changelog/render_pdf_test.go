// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestRenderPDF(t *testing.T) {
	out, err := fixtureDocument().Render(FormatPDF)
	testutils.NoError(t, err)
	testutils.HasPrefix(t, string(out), "%PDF-", "expected a valid PDF header")
	testutils.Assert(t, strings.Contains(string(out), "%%EOF"), "expected a valid PDF trailer")
	testutils.Assert(t, len(out) > 200, "expected a non-trivial PDF byte size, got %d bytes", len(out))
}

func TestRenderPDFEmptyDocument(t *testing.T) {
	out, err := NewDocument("Changelog").Render(FormatPDF)
	testutils.NoError(t, err)
	testutils.HasPrefix(t, string(out), "%PDF-", "expected a valid PDF header even for an empty document")
}

func TestRenderPDFSectionWithNilRelease(t *testing.T) {
	d := &Document{Title: "Changelog", Sections: []Section{{Title: "pkg@v0.1.0", Release: nil}}}
	out, err := d.Render(FormatPDF)
	testutils.NoError(t, err)
	testutils.HasPrefix(t, string(out), "%PDF-", "expected a valid PDF header for a section with a nil release")
}
