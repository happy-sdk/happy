// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package version

import (
	"errors"
	"fmt"
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
func Current() Version {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return Version(fmt.Sprintf("v0.0.1-devel+%d", time.Now().UnixMilli()))
	}

	// If we're on a full tag, return it directly
	if semver.IsValid(bi.Main.Version) && !strings.HasSuffix(bi.Main.Version, "-devel") {
		return Version(bi.Main.Version)
	}

	var (
		major    = 1
		commit   = ""
		modified = false
		date     = ""
	)

	if mp := path.Base(bi.Main.Path); strings.HasPrefix(mp, "v") {
		if m, err := strconv.Atoi(mp[1:]); err == nil {
			major = m + 1
		}
	}

	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			commit = setting.Value[:7] // Short commit hash
		case "vcs.modified":
			modified = (setting.Value == "true")
		case "vcs.time":
			if d, err := time.Parse(time.RFC3339, setting.Value); err == nil {
				date = fmt.Sprintf("%d", d.Unix())
			}
		}
	}

	// If we have no commit info, fallback to a basic devel version
	if commit == "" {
		return Version(fmt.Sprintf("v%d.0.0-devel+%d", major, time.Now().UnixMilli()))
	}

	// Construct pre-release suffix
	suffix := fmt.Sprintf("+git.%s.%s", commit, date)
	if modified {
		suffix += ".dirty"
	}

	// If there's a version but it's not a full tag, append commit info
	return Version(fmt.Sprintf("%d.0.1-devel%s", major, suffix))
}

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
