// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/language"
)

func TestLoadL10nFileConfigNoFile(t *testing.T) {
	dir := t.TempDir()

	cnf, err := loadL10nFileConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cnf.isUnified() {
		t.Error("expected per-lang layout by default")
	}
	if got := cnf.filenamePattern(); got != defaultPerLangFilenamePattern {
		t.Errorf("filenamePattern() = %q, want %q", got, defaultPerLangFilenamePattern)
	}
}

func TestLoadL10nFileConfigNoL10nKey(t *testing.T) {
	dir := t.TempDir()
	writeHappyYAML(t, dir, `
releaser:
  enabled: true
linter:
  golangci-lint:
    disable:
      - staticcheck
`)

	cnf, err := loadL10nFileConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cnf.isUnified() {
		t.Error("expected per-lang layout when l10n: key is absent")
	}
}

func TestLoadL10nFileConfigUnifiedLayout(t *testing.T) {
	dir := t.TempDir()
	writeHappyYAML(t, dir, `
releaser:
  enabled: true

l10n:
  layout: unified
`)

	cnf, err := loadL10nFileConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cnf.isUnified() {
		t.Error("expected unified layout")
	}
	if got := cnf.filenamePattern(); got != defaultUnifiedFilename {
		t.Errorf("filenamePattern() = %q, want %q", got, defaultUnifiedFilename)
	}
	want := filepath.Join(dir, "l10n", defaultUnifiedFilename)
	if got := cnf.appFilePath(filepath.Join(dir, "l10n"), language.German); got != want {
		t.Errorf("appFilePath() = %q, want %q", got, want)
	}
}

func TestLoadL10nFileConfigCustomFilename(t *testing.T) {
	dir := t.TempDir()
	writeHappyYAML(t, dir, `
l10n:
  layout: per-lang
  filename: "app.{lang}.strings.json"
`)

	cnf, err := loadL10nFileConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "l10n", "app.et.strings.json")
	if got := cnf.appFilePath(filepath.Join(dir, "l10n"), language.Estonian); got != want {
		t.Errorf("appFilePath() = %q, want %q", got, want)
	}
}

func TestLoadL10nFileConfigMalformedFileDoesNotError(t *testing.T) {
	dir := t.TempDir()
	writeHappyYAML(t, dir, "l10n: [this is not: a valid: mapping")

	cnf, err := loadL10nFileConfig(dir)
	if err != nil {
		t.Fatalf("expected malformed .happy.yaml to fall back to defaults, got error: %v", err)
	}
	if cnf.isUnified() {
		t.Error("expected per-lang default after malformed file")
	}
}

func TestAppFilePathPerLangDefault(t *testing.T) {
	cnf := defaultL10nFileConfig()
	want := filepath.Join("l10n", "fr.json")
	if got := cnf.appFilePath("l10n", language.French); got != want {
		t.Errorf("appFilePath() = %q, want %q", got, want)
	}
}

func writeHappyYAML(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".happy.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write .happy.yaml: %v", err)
	}
}
