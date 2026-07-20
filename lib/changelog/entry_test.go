// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestParseEntryType(t *testing.T) {
	tests := []struct {
		typ      string
		wantKind EntryKind
	}{
		{"feat", EntryKindMinor},
		{"fix", EntryKindPatch},
		{"docs", EntryKindPatch},
		{"deps", EntryKindPatch},
		{"style", EntryKindPatch},
		{"refactor", EntryKindPatch},
		{"perf", EntryKindPatch},
		{"test", EntryKindPatch},
		{"chore", EntryKindPatch},
		{"revert", EntryKindPatch},
		{"ci", EntryKindPatch},
		{"devops", EntryKindPatch},
		{"dev", EntryKindPatch},
		{"wip", EntryKindPatch},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			et, err := ParseEntryType(tt.typ, "scope")
			testutils.NoError(t, err)
			testutils.Equal(t, tt.typ, et.Typ, "unexpected Typ")
			testutils.Equal(t, "scope", et.Scope, "unexpected Scope")
			testutils.Equal(t, tt.wantKind, et.Kind, "unexpected Kind")
		})
	}
}

func TestParseEntryTypeInvalid(t *testing.T) {
	_, err := ParseEntryType("notarealtype", "")
	testutils.Error(t, err, "expected an error for an unrecognized commit type")
}
