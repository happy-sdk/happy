// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package changelog

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/happy-sdk/happy"
)

func ParseGitLog(sess *happy.Session, log string) (*Changelog, error) {
	var commits []Commit
	scanner := bufio.NewScanner(strings.NewReader(log))
	var currentCommit Commit
	var currentField *string

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, ":COMMIT_START:"):
			currentCommit = Commit{}
			currentField = nil
		case strings.HasPrefix(line, "SHORT:"):
			currentField = &currentCommit.shortHash
			*currentField = strings.TrimPrefix(line, "SHORT:")
		case strings.HasPrefix(line, "LONG:"):
			currentField = &currentCommit.longHash
			*currentField = strings.TrimPrefix(line, "LONG:")
		case strings.HasPrefix(line, "AUTHOR:"):
			currentField = &currentCommit.author
			*currentField = strings.TrimPrefix(line, "AUTHOR:")
		case strings.HasPrefix(line, "MESSAGE:"):
			currentField = &currentCommit.message
			*currentField = strings.TrimPrefix(line, "MESSAGE:")
		case strings.HasPrefix(line, ":COMMIT_END:"):
			commits = append(commits, currentCommit)
		case currentField != nil:
			*currentField += "\n" + line
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return FromCommits(sess, commits)
}

func FromCommits(sess *happy.Session, commits []Commit) (*Changelog, error) {
	changelog := &Changelog{}
	for _, commit := range commits {
		introduces := strings.Split(commit.message, "\n")
		for _, introduce := range introduces {
			if introduce == "" {
				continue
			}
			if strings.HasPrefix(introduce, "BREAKING CHANGE:") {
				introduce = strings.TrimPrefix(introduce, "BREAKING CHANGE:")
				introduce = strings.TrimSpace(introduce)
				changelog.AddBreakingChange(commit.shortHash, commit.longHash, commit.author, introduce)
				continue
			} else {
				r := regexp.MustCompile(`^(?P<Type>[^\(]+)\((?P<Scope>[^\)]+)\): (?P<Subject>.+)$`)
				matches := r.FindStringSubmatch(introduce)
				if matches == nil {
					continue
				}
				result := make(map[string]string)
				for i, name := range r.SubexpNames() {
					if i != 0 && name != "" {
						result[name] = matches[i]
					}
				}
				cTyp := result["Type"]
				cScope := result["Scope"]
				cSubject := result["Subject"]
				if cTyp == "" {
					continue
				}
				eTyp, err := ParseEntryType(cTyp, cScope)
				if err != nil {
					return nil, err
				}
				changelog.Add(commit.shortHash, commit.longHash, commit.author, cSubject, eTyp)
			}
		}
	}
	return changelog, nil
}

type Changelog struct {
	entries  []Entry
	breaking []Entry
}

func (c *Changelog) Empty() bool {
	return c.entries == nil && c.breaking == nil
}

func (c *Changelog) IsBreaking() bool {
	if c.breaking != nil {
		return len(c.breaking) > 0
	}
	return false
}

func (c *Changelog) IsFeature() bool {
	if c.entries != nil {
		for _, entry := range c.entries {
			if entry.Typ.Kind == EntryKindMinor {
				return true
			}
		}
	}
	return false
}

func (c *Changelog) IsPatch() bool {
	if c.entries != nil {
		for _, entry := range c.entries {
			if entry.Typ.Kind == EntryKindPatch {
				return true
			}
		}
	}
	return false
}

var breakingChangeType = EntryType{
	Typ:  "BREAKING CHANGE",
	Kind: EntryKindMajor,
}

func (c *Changelog) AddBreakingChange(shortHash, longHash, author, subject string) {
	c.breaking = append(c.breaking, Entry{
		ShortHash: shortHash,
		LongHash:  longHash,
		Author:    author,
		Subject:   subject,
		Typ:       breakingChangeType,
	})
}

func (c *Changelog) Add(shortHash, longHash, author, subject string, typ EntryType) {
	c.entries = append(c.entries, Entry{
		ShortHash: shortHash,
		LongHash:  longHash,
		Author:    author,
		Subject:   subject,
		Typ:       typ,
	})
}

type Entry struct {
	ShortHash string
	LongHash  string
	Author    string
	Subject   string
	Typ       EntryType
}

type EntryKind int

const (
	EntryKindPatch EntryKind = iota
	EntryKindMinor
	EntryKindMajor
)

type EntryType struct {
	Typ   string
	Scope string
	Kind  EntryKind
}

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
		etyp.Typ = "revert"
		etyp.Kind = EntryKindPatch
	case "devops":
		etyp.Typ = "devops"
		etyp.Kind = EntryKindPatch
	default:
		return etyp, fmt.Errorf("invalid commit message type: %s", typ)
	}
	etyp.Scope = scope
	return etyp, nil
}

type Commit struct {
	message   string
	shortHash string
	longHash  string
	author    string
}

var entryTypes = []EntryType{
	{Typ: "feat", Scope: "", Kind: EntryKindMinor},
	{Typ: "fix", Scope: "", Kind: EntryKindPatch},
	{Typ: "deps", Scope: "", Kind: EntryKindPatch},
	{Typ: "docs", Scope: "", Kind: EntryKindPatch},
	{Typ: "style", Scope: "", Kind: EntryKindPatch},
	{Typ: "refactor", Scope: "", Kind: EntryKindPatch},
	{Typ: "perf", Scope: "", Kind: EntryKindPatch},
	{Typ: "test", Scope: "", Kind: EntryKindPatch},
	{Typ: "devops", Scope: "", Kind: EntryKindPatch},
	{Typ: "chore", Scope: "", Kind: EntryKindPatch},
	{Typ: "revert", Scope: "", Kind: EntryKindPatch},
}
