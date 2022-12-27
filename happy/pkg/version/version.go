// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package version

import (
	"fmt"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	// "golang.org/x/mod/semver"
)

type Version string

func (v Version) String() string {
	return string(v)
}

func Current() Version {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return Version(fmt.Sprintf("v0.0.1-devel+%d", time.Now().UnixMilli()))
	}

	if bi.Main.Version != "(devel)" && len(bi.Main.Version) > 0 {
		return Version(bi.Main.Version)
	}

	var (
		major    = "v1"
		revision = ""
		vcs      = ""
		date     = ""
		modified = false
		dirty    = ""
	)
	if mp := path.Base(bi.Main.Path); strings.HasPrefix(mp, "v") {
		if m, err := strconv.Atoi(mp[1:]); err == nil {
			major = fmt.Sprintf("v%d", m+1)
		}
	}
	for _, setting := range bi.Settings {
		if setting.Key == "vcs" {
			vcs = setting.Value
		}
		if setting.Key == "vcs.revision" {
			revision = "+" + setting.Value[0:6]
		}
		if setting.Key == "vcs.modified" && setting.Value == "true" {
			modified = true
		}
		if setting.Key == "vcs.time" {
			d, _ := time.Parse(time.RFC3339, setting.Value)
			date = fmt.Sprint(d.Unix())
		}
	}

	if modified {
		dirty = "." + vcs + "." + date
	}

	return Version(fmt.Sprintf("%s.0.0-alpha%s%s", major, revision, dirty))
}
