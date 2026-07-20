// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestNewReleaseEmpty(t *testing.T) {
	r := NewRelease()
	testutils.Assert(t, r.Empty(), "expected a fresh release to be empty")
	testutils.Assert(t, !r.HasMajorUpdate(), "expected no major update")
	testutils.Assert(t, !r.HasMinorUpdate(), "expected no minor update")
	testutils.Assert(t, !r.HasPatchUpdate(), "expected no patch update")
}

func TestReleaseAdd(t *testing.T) {
	r := NewRelease()
	feat, err := ParseEntryType("feat", "")
	testutils.NoError(t, err)
	r.Add("abc1234", "abc1234full", "author", "add a feature", feat)

	testutils.Assert(t, !r.Empty(), "expected release with an entry to be non-empty")
	testutils.Equal(t, 1, len(r.Entries()), "unexpected entry count")
	testutils.Equal(t, 0, len(r.Breaking()), "unexpected breaking count")
	testutils.Assert(t, r.HasMinorUpdate(), "expected a minor update for a feat entry")
	testutils.Assert(t, !r.HasMajorUpdate(), "expected no major update")
	testutils.Assert(t, !r.HasPatchUpdate(), "expected no patch update")

	e := r.Entries()[0]
	testutils.Equal(t, "abc1234", e.ShortHash, "unexpected ShortHash")
	testutils.Equal(t, "abc1234full", e.LongHash, "unexpected LongHash")
	testutils.Equal(t, "author", e.Author, "unexpected Author")
	testutils.Equal(t, "add a feature", e.Subject, "unexpected Subject")
}

func TestReleaseAddBreakingChange(t *testing.T) {
	r := NewRelease()
	r.AddBreakingChange("def5678", "def5678full", "author", "removed old API")

	testutils.Assert(t, !r.Empty(), "expected release with a breaking change to be non-empty")
	testutils.Equal(t, 1, len(r.Breaking()), "unexpected breaking count")
	testutils.Assert(t, r.HasMajorUpdate(), "expected a major update for a breaking change")

	b := r.Breaking()[0]
	testutils.Equal(t, "def5678", b.ShortHash, "unexpected ShortHash")
	testutils.Equal(t, EntryKindMajor, b.Typ.Kind, "expected breaking change entries to be classified major")
}

func TestReleasePatchUpdate(t *testing.T) {
	r := NewRelease()
	fix, err := ParseEntryType("fix", "")
	testutils.NoError(t, err)
	r.Add("h", "h", "a", "fix a bug", fix)

	testutils.Assert(t, r.HasPatchUpdate(), "expected a patch update for a fix entry")
	testutils.Assert(t, !r.HasMinorUpdate(), "expected no minor update")
	testutils.Assert(t, !r.HasMajorUpdate(), "expected no major update")
}

func TestReleaseHasMajorUpdatePrefersBreakingOverEntries(t *testing.T) {
	r := NewRelease()
	r.AddBreakingChange("h", "h", "a", "breaking")
	testutils.Assert(t, r.HasMajorUpdate(), "expected breaking changes alone to signal a major update")
}

// TestReleaseHasMajorUpdateFromEntryKind covers a major-kind entry added
// directly (programmatic composition, not via git-log parsing where only
// AddBreakingChange ever produces EntryKindMajor) with no breaking changes
// present, exercising the entries-loop fallback in HasMajorUpdate.
func TestReleaseHasMajorUpdateFromEntryKind(t *testing.T) {
	r := NewRelease()
	r.Add("h", "h", "a", "manually classified as major", EntryType{Typ: "custom", Kind: EntryKindMajor})
	testutils.Assert(t, r.HasMajorUpdate(), "expected a major-kind entry to signal a major update even with no breaking changes")
}
