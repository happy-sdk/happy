// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func commitBlock(short, long, author, message string) string {
	return strings.Join([]string{
		":COMMIT_START:",
		"SHORT:" + short,
		"LONG:" + long,
		"AUTHOR:" + author,
		"MESSAGE:" + message,
		":COMMIT_END:",
	}, "\n")
}

func TestParseGitLogBasicEntries(t *testing.T) {
	log := strings.Join([]string{
		commitBlock("abc1111", "abc1111full", "Jane Doe", "feat: add new widget"),
		commitBlock("abc2222", "abc2222full", "John Roe", "fix(cli): correct flag parsing"),
	}, "\n")

	r, err := ParseGitLog(log)
	testutils.NoError(t, err)
	testutils.Equal(t, 2, len(r.Entries()), "unexpected entry count")
	testutils.Equal(t, 0, len(r.Breaking()), "expected no breaking changes")

	testutils.Equal(t, "add new widget", r.Entries()[0].Subject, "unexpected first subject")
	testutils.Equal(t, "feat", r.Entries()[0].Typ.Typ, "unexpected first type")

	testutils.Equal(t, "correct flag parsing", r.Entries()[1].Subject, "unexpected second subject")
	testutils.Equal(t, "cli", r.Entries()[1].Typ.Scope, "unexpected second scope")
}

func TestParseGitLogBreakingChange(t *testing.T) {
	msg := "feat(api): rework client\n\nBREAKING CHANGE: client constructor signature changed"
	log := commitBlock("def3333", "def3333full", "Jane Doe", msg)

	r, err := ParseGitLog(log)
	testutils.NoError(t, err)
	testutils.Equal(t, 1, len(r.Entries()), "expected the feat line to also be recorded as a regular entry")
	testutils.Equal(t, 1, len(r.Breaking()), "expected one breaking change")
	testutils.Equal(t, "client constructor signature changed", r.Breaking()[0].Subject, "unexpected breaking subject")
	testutils.Assert(t, r.HasMajorUpdate(), "expected a major update")
}

func TestParseGitLogMultilineSubject(t *testing.T) {
	msg := "fix: correct a bug\nthat spans two lines"
	log := commitBlock("ghi4444", "ghi4444full", "Jane Doe", msg)

	r, err := ParseGitLog(log)
	testutils.NoError(t, err)
	testutils.Equal(t, 1, len(r.Entries()), "unexpected entry count")
	testutils.Equal(t, "correct a bug that spans two lines", r.Entries()[0].Subject, "expected multiline subject to be joined")
}

func TestParseGitLogUnrecognizedTypeSkipped(t *testing.T) {
	log := commitBlock("jkl5555", "jkl5555full", "Jane Doe", "notarealtype: should be skipped")

	r, err := ParseGitLog(log)
	testutils.NoError(t, err)
	testutils.Assert(t, r.Empty(), "expected an unrecognized commit type to be skipped entirely")
}

func TestParseGitLogEmpty(t *testing.T) {
	r, err := ParseGitLog("")
	testutils.NoError(t, err)
	testutils.Assert(t, r.Empty(), "expected an empty log to produce an empty release")
}

func TestFromCommitsDirect(t *testing.T) {
	commits := []Commit{
		{shortHash: "a", longHash: "afull", author: "Jane", message: "chore: tidy up"},
	}
	r, err := FromCommits(commits)
	testutils.NoError(t, err)
	testutils.Equal(t, 1, len(r.Entries()), "unexpected entry count")
	testutils.Assert(t, r.HasPatchUpdate(), "expected chore to be classified as a patch update")
}

// TestFromCommitsMultipleEntriesOneUnrecognized covers a single commit whose
// message contains two conventional-commit lines, the first with an
// unrecognized type. Only the previous (first) entry is looked up via
// ParseEntryType before starting the new one, so this exercises that
// skip-on-error path independently from the final pending-entry path.
func TestFromCommitsMultipleEntriesOneUnrecognized(t *testing.T) {
	commits := []Commit{
		{shortHash: "a", longHash: "afull", author: "Jane", message: "notarealtype: skipped\nfix: kept"},
	}
	r, err := FromCommits(commits)
	testutils.NoError(t, err)
	testutils.Equal(t, 1, len(r.Entries()), "expected only the recognized entry to be recorded")
	testutils.Equal(t, "kept", r.Entries()[0].Subject, "unexpected surviving subject")
}
