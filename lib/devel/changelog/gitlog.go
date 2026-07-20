// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"bufio"
	"regexp"
	"strings"
)

// Commit is a single raw commit record as read from `git log`, before it has
// been split into conventional-commit Entries.
type Commit struct {
	message   string
	shortHash string
	longHash  string
	author    string
}

// ParseGitLog parses the output of a `git log` invocation formatted as:
//
//	--pretty=format::COMMIT_START:%nSHORT:%h%nLONG:%H%nAUTHOR:%an%nMESSAGE:%B:COMMIT_END:
//
// into a Release, classifying each conventional-commit message into regular
// changes or breaking changes.
func ParseGitLog(log string) (*Release, error) {
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

	return FromCommits(commits)
}

var commitRegex = regexp.MustCompile(`^(?P<Type>[^\(]+)(?:\((?P<Scope>[^\)]*)\))?: (?P<Subject>.+)$`)

// FromCommits classifies a slice of raw Commits into a Release.
func FromCommits(commits []Commit) (*Release, error) {
	release := &Release{}

	breakingChangePrefix := "BREAKING CHANGE:"

	for _, commit := range commits {
		lines := strings.Split(commit.message, "\n")
		var currentSubject, currentType, currentScope string

		for _, line := range lines {
			line = strings.TrimSpace(line)

			if subject, ok := strings.CutPrefix(line, breakingChangePrefix); ok {
				release.AddBreakingChange(commit.shortHash, commit.longHash, commit.author, strings.TrimSpace(subject))
				continue
			}

			if matches := commitRegex.FindStringSubmatch(line); matches != nil {
				if currentSubject != "" {
					// Add the previous entry
					eTyp, err := ParseEntryType(currentType, currentScope)
					if err == nil {
						release.Add(commit.shortHash, commit.longHash, commit.author, currentSubject, eTyp)
					}
				}
				// Start a new entry
				currentType, currentScope, currentSubject = matches[1], matches[2], matches[3]
			} else if currentSubject != "" {
				// Append to the current subject if it's a multiline commit message
				currentSubject += " " + line
			}
		}

		// Add the last entry if there's one pending
		if currentSubject != "" {
			eTyp, err := ParseEntryType(currentType, currentScope)
			if err == nil {
				release.Add(commit.shortHash, commit.longHash, commit.author, currentSubject, eTyp)
			}
		}
	}

	return release, nil
}
