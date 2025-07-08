// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package version

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// PRE is the default pre-release identifier for development versions.
const PRE = "devel"

// Error represents a versioning error.
var Error = errors.New("version")

// Version is a semantic version string.
type Version string

// String returns the string representation of the version.
func (v Version) String() string {
	return string(v)
}

// Build returns the build metadata of the version, if any.
func (v Version) Build() string {
	return strings.Trim(semver.Build(v.String()), "+")
}

// Current attempts to generate the closest realistic semantic version
// without requiring -ldflags, using Go module build info.

// Parse attempts to parse a version string and returns a valid `Version` type.
func Parse(v string) (Version, error) {
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return Version(""), fmt.Errorf("%w: invalid version: %s", Error, v)
	}
	return Version(v), nil
}

// Prerelease extracts the pre-release part of a semantic version.
func Prerelease(v string) string {
	return strings.Trim(semver.Prerelease(v), "-")
}

// IsDev checks if the version is a development version.
func IsDev(v string) bool {
	return Prerelease(v) == PRE
}

// Current attempts to generate the closest realistic semantic version
// without requiring -ldflags, using Go module build info and Git fallbacks.
func Current() Version {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return fallbackVersion()
	}

	// If we're on a full tag, return it directly
	if semver.IsValid(bi.Main.Version) && !strings.HasSuffix(bi.Main.Version, "-devel") {
		return Version(bi.Main.Version)
	}

	// Check if we have a pseudo-version (like v0.40.2-0.20250622225330-8992fbf30020)
	// This happens when -buildvcs is used
	if strings.Contains(bi.Main.Version, "-") && !strings.HasSuffix(bi.Main.Version, "-devel") {
		version := bi.Main.Version

		// Check if modified from VCS settings
		for _, setting := range bi.Settings {
			if setting.Key == "vcs.modified" && setting.Value == "true" {
				version += "+dirty"
				break
			}
		}

		return Version(version)
	}

	var (
		major    = 1
		commit   = ""
		modified = false
		date     = ""
	)

	// Extract major version from module path
	if mp := path.Base(bi.Main.Path); strings.HasPrefix(mp, "v") {
		if m, err := strconv.Atoi(mp[1:]); err == nil {
			major = m + 1
		}
	}

	// Try to get VCS info from build info first
	vcsInfoFound := false
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			commit = setting.Value
			if len(commit) > 7 {
				commit = commit[:7] // Short commit hash
			}
			vcsInfoFound = true
		case "vcs.modified":
			modified = (setting.Value == "true")
		case "vcs.time":
			if d, err := time.Parse(time.RFC3339, setting.Value); err == nil {
				date = fmt.Sprintf("%d", d.Unix())
			}
		}
	}

	// If no VCS info from build info, try Git commands as fallback
	if !vcsInfoFound || commit == "" {
		if gitCommit, gitModified, gitDate := getGitInfo(); gitCommit != "" {
			commit = gitCommit
			modified = gitModified
			date = gitDate
		}
	}

	// If we still have no commit info, fallback to a basic devel version
	if commit == "" {
		return fallbackVersion()
	}

	// Try to construct the same pseudo-version format as -buildvcs
	if baseVersion := getLatestTag(); baseVersion != "" {
		// Get commit timestamp in the exact format Go uses
		timestamp := ""
		if date != "" {
			if ts, err := strconv.ParseInt(date, 10, 64); err == nil {
				t := time.Unix(ts, 0).UTC()
				timestamp = t.Format("20060102150405")
			}
		} else {
			// Fallback to git log for commit time
			if commitTime := getCommitTime(); commitTime != "" {
				timestamp = commitTime
			}
		}

		// Get 12-character commit hash (same as Go uses)
		commit12 := commit
		if fullCommit := getFullCommitHash(); fullCommit != "" && len(fullCommit) >= 12 {
			commit12 = fullCommit[:12]
		}

		if timestamp != "" && commit12 != "" {
			version := fmt.Sprintf("%s-0.%s-%s", baseVersion, timestamp, commit12)
			if modified {
				version += "+dirty"
			}
			return Version(version)
		}
	}

	// Fallback to original format if we can't get tag info
	suffix := fmt.Sprintf("+git.%s", commit)
	if date != "" {
		suffix += "." + date
	}
	if modified {
		suffix += ".dirty"
	}

	return Version(fmt.Sprintf("v%d.0.1-devel%s", major, suffix))
}

func Compare(v Version, w Version) int {
	return semver.Compare(v.String(), w.String())
}

// getGitInfo attempts to get Git information using command line
func getGitInfo() (commit string, modified bool, date string) {
	// Check if we're in a Git repository
	if !isGitRepo() {
		return "", false, ""
	}

	// Get commit hash
	if cmd := exec.Command("git", "rev-parse", "--short=7", "HEAD"); cmd.Err == nil {
		if output, err := cmd.Output(); err == nil {
			commit = strings.TrimSpace(string(output))
		}
	}

	// Check if working directory is dirty
	if cmd := exec.Command("git", "diff-index", "--quiet", "HEAD", "--"); cmd.Err == nil {
		if err := cmd.Run(); err != nil {
			modified = true
		}
	}

	// Get commit date
	if commit != "" {
		if cmd := exec.Command("git", "show", "-s", "--format=%ct", commit); cmd.Err == nil {
			if output, err := cmd.Output(); err == nil {
				date = strings.TrimSpace(string(output))
			}
		}
	}

	return commit, modified, date
}

// getLatestTag attempts to get the base version for pseudo-version (matching Go's logic)
func getLatestTag() string {
	if !isGitRepo() {
		return ""
	}

	// Get the most recent tag that's an ancestor of HEAD
	if cmd := exec.Command("git", "describe", "--tags", "--abbrev=0", "--match=v*.*.*"); cmd.Err == nil {
		if output, err := cmd.Output(); err == nil {
			tag := strings.TrimSpace(string(output))
			if semver.IsValid(tag) {
				// Check if we have commits after this tag
				if hasCommitsAfterTag(tag) {
					// Increment patch version (Go's pseudo-version behavior)
					return incrementPatchVersion(tag)
				}
				return tag
			}
		}
	}

	return ""
}

// hasCommitsAfterTag checks if there are commits after the given tag
func hasCommitsAfterTag(tag string) bool {
	if !isGitRepo() {
		return false
	}

	// Check if HEAD is different from the tag
	cmd := exec.Command("git", "rev-list", "--count", tag+"..HEAD")
	if output, err := cmd.Output(); err == nil {
		count := strings.TrimSpace(string(output))
		if c, err := strconv.Atoi(count); err == nil && c > 0 {
			return true
		}
	}

	return false
}

// incrementPatchVersion increments the patch version of a semver tag
func incrementPatchVersion(tag string) string {
	if !strings.HasPrefix(tag, "v") {
		return tag
	}

	// Parse version like v0.40.1
	parts := strings.Split(tag[1:], ".")
	if len(parts) != 3 {
		return tag
	}

	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])

	if err1 != nil || err2 != nil || err3 != nil {
		return tag
	}

	// Increment patch version
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch+1)
}

// getCommitTime gets the commit time in Go's pseudo-version format (YYYYMMDDHHMMSS)
func getCommitTime() string {
	if !isGitRepo() {
		return ""
	}

	// Get commit time for current HEAD in UTC
	if cmd := exec.Command("git", "show", "-s", "--format=%ct", "HEAD"); cmd.Err == nil {
		if output, err := cmd.Output(); err == nil {
			if ts, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64); err == nil {
				t := time.Unix(ts, 0).UTC()
				return t.Format("20060102150405")
			}
		}
	}

	return ""
}

// getFullCommitHash gets the full commit hash
func getFullCommitHash() string {
	if !isGitRepo() {
		return ""
	}

	if cmd := exec.Command("git", "rev-parse", "HEAD"); cmd.Err == nil {
		if output, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(output))
		}
	}

	return ""
}

// isGitRepo checks if current directory is in a Git repository
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// fallbackVersion returns a basic development version when no VCS info is available
func fallbackVersion() Version {
	return Version(fmt.Sprintf("v0.0.1-devel+%d", time.Now().UnixMilli()))
}

// IsGoRun detects if the program is running via 'go run'
func IsGoRun() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}

	return strings.Contains(executable, os.TempDir()) ||
		strings.Contains(executable, "go-build")
}
