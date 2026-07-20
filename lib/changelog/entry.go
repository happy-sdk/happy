// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import "fmt"

// Entry is a single change record - either a regular change or a breaking
// change, depending on which slice of a Release it lives in.
type Entry struct {
	ShortHash string
	LongHash  string
	Author    string
	Subject   string
	Typ       EntryType
}

// EntryKind classifies an EntryType by the semver bump it implies.
type EntryKind int

const (
	EntryKindPatch EntryKind = iota
	EntryKindMinor
	EntryKindMajor
)

// EntryType describes the conventional-commit type/scope an Entry was
// parsed from (or was constructed with, when composed programmatically).
type EntryType struct {
	Typ   string
	Scope string
	Kind  EntryKind
}

var breakingChangeType = EntryType{
	Typ:  "BREAKING CHANGE",
	Kind: EntryKindMajor,
}

// ParseEntryType maps a conventional-commit type (e.g. "feat", "fix") and
// its optional scope to an EntryType. It returns an error for unrecognized
// types so callers parsing free-form commit messages can skip lines that
// aren't conventional-commit entries at all.
func ParseEntryType(typ, scope string) (EntryType, error) {
	etyp := EntryType{}
	switch typ {
	case "feat":
		etyp.Typ = "feat"
		etyp.Kind = EntryKindMinor
	case "fix":
		etyp.Typ = "fix"
		etyp.Kind = EntryKindPatch
	case "docs":
		etyp.Typ = "docs"
		etyp.Kind = EntryKindPatch
	case "deps":
		etyp.Typ = "deps"
		etyp.Kind = EntryKindPatch
	case "style":
		etyp.Typ = "style"
		etyp.Kind = EntryKindPatch
	case "refactor":
		etyp.Typ = "refactor"
		etyp.Kind = EntryKindPatch
	case "perf":
		etyp.Typ = "perf"
		etyp.Kind = EntryKindPatch
	case "test":
		etyp.Typ = "test"
		etyp.Kind = EntryKindPatch
	case "chore":
		etyp.Typ = "chore"
		etyp.Kind = EntryKindPatch
	case "revert":
		etyp.Typ = "revert"
		etyp.Kind = EntryKindPatch
	case "ci":
		etyp.Typ = "ci"
		etyp.Kind = EntryKindPatch
	case "devops":
		etyp.Typ = "devops"
		etyp.Kind = EntryKindPatch
	case "dev":
		etyp.Typ = "dev"
		etyp.Kind = EntryKindPatch
	case "wip":
		etyp.Typ = "wip"
		etyp.Kind = EntryKindPatch
	default:
		return etyp, fmt.Errorf("invalid commit message type: %s", typ)
	}
	etyp.Scope = scope
	return etyp, nil
}
