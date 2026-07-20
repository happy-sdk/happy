// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

// Release is the set of changes for a single release - either parsed from
// git log (see ParseGitLog/FromCommits) or built up programmatically via
// Add/AddBreakingChange.
type Release struct {
	entries  []Entry
	breaking []Entry
}

// NewRelease returns an empty Release ready for programmatic composition.
func NewRelease() *Release {
	return &Release{}
}

func (r *Release) Entries() []Entry {
	return r.entries
}

func (r *Release) Breaking() []Entry {
	return r.breaking
}

func (r *Release) Empty() bool {
	return len(r.entries) == 0 && len(r.breaking) == 0
}

func (r *Release) HasMajorUpdate() bool {
	if len(r.breaking) > 0 {
		return true
	}
	for _, entry := range r.entries {
		if entry.Typ.Kind == EntryKindMajor {
			return true
		}
	}
	return false
}

func (r *Release) HasMinorUpdate() bool {
	for _, entry := range r.entries {
		if entry.Typ.Kind == EntryKindMinor {
			return true
		}
	}
	return false
}

func (r *Release) HasPatchUpdate() bool {
	for _, entry := range r.entries {
		if entry.Typ.Kind == EntryKindPatch {
			return true
		}
	}
	return false
}

// AddBreakingChange records a breaking change entry.
func (r *Release) AddBreakingChange(shortHash, longHash, author, subject string) {
	r.breaking = append(r.breaking, Entry{
		ShortHash: shortHash,
		LongHash:  longHash,
		Author:    author,
		Subject:   subject,
		Typ:       breakingChangeType,
	})
}

// Add records a regular change entry.
func (r *Release) Add(shortHash, longHash, author, subject string, typ EntryType) {
	r.entries = append(r.entries, Entry{
		ShortHash: shortHash,
		LongHash:  longHash,
		Author:    author,
		Subject:   subject,
		Typ:       typ,
	})
}
