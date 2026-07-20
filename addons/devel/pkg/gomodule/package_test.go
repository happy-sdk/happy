// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gomodule

import (
	"testing"

	"golang.org/x/mod/modfile"
)

func newTestPackage(t *testing.T, goVersion, tagPrefix, lastReleaseTag string, internal bool) *Package {
	t.Helper()
	mf, err := modfile.Parse("go.mod", []byte("module example.com/mod\n\ngo "+goVersion+"\n"), nil)
	if err != nil {
		t.Fatalf("failed to parse fixture go.mod: %v", err)
	}
	return &Package{
		Import:         "example.com/mod",
		TagPrefix:      tagPrefix,
		Modfile:        mf,
		IsInternal:     internal,
		LastReleaseTag: lastReleaseTag,
	}
}

func TestPackage_SyncGoVersion(t *testing.T) {
	t.Run("no-op when already in sync", func(t *testing.T) {
		p := newTestPackage(t, "1.25.0", "pkg/mod/", "pkg/mod/v1.2.3", false)
		if err := p.SyncGoVersion("1.25.0", BumpKindMajor, BumpStrategySingle); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.NeedsRelease {
			t.Error("expected NeedsRelease to stay false when go version already matches")
		}
		if p.Modfile.Go.Version != "1.25.0" {
			t.Errorf("expected go version to remain 1.25.0, got %s", p.Modfile.Go.Version)
		}
	})

	t.Run("updates and bumps major when out of sync", func(t *testing.T) {
		p := newTestPackage(t, "1.25.0", "pkg/mod/", "pkg/mod/v1.2.3", false)
		if err := p.SyncGoVersion("1.26.4", BumpKindMajor, BumpStrategySingle); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !p.NeedsRelease {
			t.Error("expected NeedsRelease to become true when go version differs from root")
		}
		if p.Modfile.Go.Version != "1.26.4" {
			t.Errorf("expected go version updated to 1.26.4, got %s", p.Modfile.Go.Version)
		}
		if p.NextReleaseTag != "pkg/mod/v2.0.0" {
			t.Errorf("expected major-bumped next release tag pkg/mod/v2.0.0, got %s", p.NextReleaseTag)
		}
	})

	t.Run("bumps minor to the next full hundred when configured", func(t *testing.T) {
		p := newTestPackage(t, "1.25.0", "pkg/mod/", "pkg/mod/v1.55.3", false)
		if err := p.SyncGoVersion("1.27.0", BumpKindMinor, BumpStrategyHundred); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.NextReleaseTag != "pkg/mod/v1.100.0" {
			t.Errorf("expected next release tag pkg/mod/v1.100.0, got %s", p.NextReleaseTag)
		}
	})

	t.Run("does not override an already-set next release tag that is already higher", func(t *testing.T) {
		p := newTestPackage(t, "1.25.0", "pkg/mod/", "pkg/mod/v1.2.3", false)
		p.NextReleaseTag = "pkg/mod/v2.0.0" // e.g. already bumped minor/major by changelog
		if err := p.SyncGoVersion("1.26.4", BumpKindMajor, BumpStrategySingle); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.NextReleaseTag != "pkg/mod/v2.0.0" {
			t.Errorf("expected existing next release tag to be preserved, got %s", p.NextReleaseTag)
		}
	})

	// Regression: a package with ordinary (non-breaking) commits since its
	// last tag gets an ordinary +1 minor bump from getChangelog before
	// SyncGoVersion ever runs. SyncGoVersion used to only apply its own
	// bump when NextReleaseTag was still unset/unchanged, so that smaller
	// ordinary bump silently won even when the Go version sync's bump
	// (e.g. a "hundred" jump) was actually larger. It must compare the two
	// and take whichever is bigger.
	t.Run("overrides an already-set next release tag that is lower", func(t *testing.T) {
		p := newTestPackage(t, "1.25.0", "pkg/mod/", "pkg/mod/v1.55.3", false)
		p.NextReleaseTag = "pkg/mod/v1.56.0" // e.g. ordinary minor bump from a feat commit
		if err := p.SyncGoVersion("1.27.0", BumpKindMinor, BumpStrategyHundred); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.NextReleaseTag != "pkg/mod/v1.100.0" {
			t.Errorf("expected the larger hundred-bump v1.100.0 to win, got %s", p.NextReleaseTag)
		}
	})

	t.Run("skips internal modules", func(t *testing.T) {
		p := newTestPackage(t, "1.25.0", "pkg/mod/", "pkg/mod/v1.2.3", true)
		if err := p.SyncGoVersion("1.26.4", BumpKindMajor, BumpStrategySingle); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.NeedsRelease {
			t.Error("expected internal modules to be skipped")
		}
		if p.Modfile.Go.Version != "1.25.0" {
			t.Errorf("expected go version to remain unchanged for internal module, got %s", p.Modfile.Go.Version)
		}
	})

	t.Run("no-op when root version is unknown", func(t *testing.T) {
		p := newTestPackage(t, "1.25.0", "pkg/mod/", "pkg/mod/v1.2.3", false)
		if err := p.SyncGoVersion("", BumpKindMajor, BumpStrategySingle); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.NeedsRelease {
			t.Error("expected no-op when root go version could not be determined")
		}
	})
}
