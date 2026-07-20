// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/addons/devel/pkg/gomodule"
	"github.com/happy-sdk/happy/lib/devel/changelog"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func changelogWith(entries, breaking []string) *changelog.Release {
	cl := changelog.NewRelease()
	for _, e := range entries {
		cl.Add("abc1234", "abc1234full", "author", e, changelog.EntryType{})
	}
	for _, b := range breaking {
		cl.AddBreakingChange("def5678", "def5678full", "author", b)
	}
	return cl
}

// resultString renders a tr.Result's unexported message for assertions -
// tr.Result exposes no getters, but its fields are readable via fmt/reflect
// (the standard, well-known way to inspect unexported struct fields for
// diagnostic printing without calling any of their methods).
func resultString(res any) string {
	return fmt.Sprintf("%+v", res)
}

func TestWriteChangelogsRootModule(t *testing.T) {
	rootDir := t.TempDir()
	distDir := filepath.Join(t.TempDir(), ".happy", "build")

	prj := &Project{dir: DirInfo{Path: rootDir}, dist: distDir}

	gomodules := []*gomodule.Package{
		{
			Import:         "github.com/happy-sdk/happy",
			Dir:            rootDir,
			TagPrefix:      "",
			NeedsRelease:   true,
			LastReleaseTag: "v1.1.0",
			NextReleaseTag: "v1.2.0",
			Changelog:      changelogWith([]string{"add: new feature"}, nil),
		},
		{
			Import:         "github.com/happy-sdk/happy/pkg/sub",
			Dir:            filepath.Join(rootDir, "pkg", "sub"),
			TagPrefix:      "pkg/sub/",
			NeedsRelease:   true,
			LastReleaseTag: "pkg/sub/v0.1.0",
			NextReleaseTag: "pkg/sub/v0.2.0",
			Changelog:      changelogWith([]string{"fix: sub bug"}, []string{"remove: old API"}),
		},
		{
			// not releasing - must be excluded from the changelog entirely.
			Import:         "github.com/happy-sdk/happy/pkg/skip",
			Dir:            filepath.Join(rootDir, "pkg", "skip"),
			TagPrefix:      "pkg/skip/",
			NeedsRelease:   false,
			NextReleaseTag: "pkg/skip/v0.1.0",
			Changelog:      changelogWith([]string{"unused"}, nil),
		},
	}

	res := prj.writeChangelogs(gomodules)
	testutils.Assert(t, strings.Contains(resultString(res), "changelog saved"), "expected a success result, got %s", resultString(res))

	clPath := filepath.Join(distDir, "v1.2.0", "CHANGELOG.md")
	data, err := os.ReadFile(clPath)
	testutils.NoError(t, err, "expected changelog written to %s", clPath)

	content := string(data)
	for _, want := range []string{
		"github.com/happy-sdk/happy@v1.2.0",
		"add: new feature",
		"pkg/sub/v0.2.0",
		"fix: sub bug",
		"remove: old API",
	} {
		testutils.Assert(t, strings.Contains(content, want), "expected changelog to contain %q, got:\n%s", want, content)
	}
	testutils.Assert(t, !strings.Contains(content, "unused"), "expected non-releasing package to be excluded, got:\n%s", content)

	// No per-package fallback files should exist alongside the unified one.
	_, err = os.Stat(filepath.Join(distDir, "pkg/sub/v0.2.0"))
	testutils.Assert(t, os.IsNotExist(err), "expected no per-package changelog dir when a root module exists")
}

func TestWriteChangelogsNoRootModule(t *testing.T) {
	rootDir := t.TempDir()
	distDir := filepath.Join(t.TempDir(), ".happy", "build")

	prj := &Project{dir: DirInfo{Path: rootDir}, dist: distDir}

	gomodules := []*gomodule.Package{
		{
			Import:         "github.com/happy-sdk/addons/daemon",
			Dir:            filepath.Join(rootDir, "daemon"),
			TagPrefix:      "daemon/",
			NeedsRelease:   true,
			NextReleaseTag: "daemon/v1.0.0",
			Changelog:      changelogWith([]string{"feat: daemon thing"}, []string{"break: daemon API"}),
		},
		{
			Import:         "github.com/happy-sdk/addons/agentic",
			Dir:            filepath.Join(rootDir, "agentic"),
			TagPrefix:      "agentic/",
			NeedsRelease:   false,
			NextReleaseTag: "agentic/v0.1.0",
			Changelog:      changelogWith([]string{"unused"}, nil),
		},
	}

	res := prj.writeChangelogs(gomodules)
	testutils.Assert(t, strings.Contains(resultString(res), "1 package changelog"), "expected one package changelog saved, got %s", resultString(res))

	daemonCl := filepath.Join(distDir, "daemon/v1.0.0", "CHANGELOG.md")
	data, err := os.ReadFile(daemonCl)
	testutils.NoError(t, err, "expected per-package changelog written to %s", daemonCl)

	content := string(data)
	for _, want := range []string{
		"github.com/happy-sdk/addons/daemon@daemon/v1.0.0",
		"feat: daemon thing",
		"break: daemon API",
	} {
		testutils.Assert(t, strings.Contains(content, want), "expected changelog to contain %q, got:\n%s", want, content)
	}

	_, err = os.Stat(filepath.Join(distDir, "agentic/v0.1.0"))
	testutils.Assert(t, os.IsNotExist(err), "expected non-releasing package to have no changelog written")
}

func TestWritePackageChangelogsNoneToWrite(t *testing.T) {
	prj := &Project{dist: filepath.Join(t.TempDir(), ".happy", "build")}

	res := prj.writePackageChangelogs([]*gomodule.Package{
		{Import: "github.com/happy-sdk/addons/idle", NeedsRelease: false},
		{Import: "github.com/happy-sdk/addons/nilcl", NeedsRelease: true, Changelog: nil},
	})

	testutils.Assert(t, strings.Contains(resultString(res), "no changelogs to write"), "expected a skip result, got %s", resultString(res))
}
