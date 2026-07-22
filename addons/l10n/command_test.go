// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import "testing"

func TestDefaultCommandConfig(t *testing.T) {
	cnf := DefaultCommandConfig()
	if cnf.Name != "l10n" {
		t.Errorf("Name = %q, want %q", cnf.Name, "l10n")
	}
	if cnf.WithoutReport || cnf.WithoutList || cnf.WithoutTranslate {
		t.Error("expected all subcommands to be enabled by default")
	}
}

func TestCommandBuildsWithoutError(t *testing.T) {
	cmd := Command(DefaultCommandConfig())
	if err := cmd.Err(); err != nil {
		t.Fatalf("Command() with default config returned error: %v", err)
	}
	if cmd.Name() != "l10n" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "l10n")
	}
}

func TestCommandWithoutSubcommands(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*CommandConfig)
	}{
		{"without report", func(c *CommandConfig) { c.WithoutReport = true }},
		{"without list", func(c *CommandConfig) { c.WithoutList = true }},
		{"without translate", func(c *CommandConfig) { c.WithoutTranslate = true }},
		{"without all", func(c *CommandConfig) {
			c.WithoutReport = true
			c.WithoutList = true
			c.WithoutTranslate = true
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cnf := DefaultCommandConfig()
			tt.modify(&cnf)
			cmd := Command(cnf)
			if err := cmd.Err(); err != nil {
				t.Fatalf("Command() returned error: %v", err)
			}
		})
	}
}

func TestSubcommandConstructors(t *testing.T) {
	t.Run("l10nReport", func(t *testing.T) {
		cmd := l10nReport()
		if err := cmd.Err(); err != nil {
			t.Fatalf("l10nReport() returned error: %v", err)
		}
	})
	t.Run("l10nList", func(t *testing.T) {
		cmd := l10nList()
		if err := cmd.Err(); err != nil {
			t.Fatalf("l10nList() returned error: %v", err)
		}
	})
	t.Run("l10nTranslate", func(t *testing.T) {
		cmd := l10nTranslate()
		if err := cmd.Err(); err != nil {
			t.Fatalf("l10nTranslate() returned error: %v", err)
		}
	})
}
