// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import "testing"

func TestDefaultCommandConfig(t *testing.T) {
	cnf := DefaultCommandConfig()
	if cnf.Name != "i18n" {
		t.Errorf("Name = %q, want %q", cnf.Name, "i18n")
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
	if cmd.Name() != "i18n" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "i18n")
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
	t.Run("i18nReport", func(t *testing.T) {
		cmd := i18nReport()
		if err := cmd.Err(); err != nil {
			t.Fatalf("i18nReport() returned error: %v", err)
		}
	})
	t.Run("i18nList", func(t *testing.T) {
		cmd := i18nList()
		if err := cmd.Err(); err != nil {
			t.Fatalf("i18nList() returned error: %v", err)
		}
	})
	t.Run("i18nTranslate", func(t *testing.T) {
		cmd := i18nTranslate()
		if err := cmd.Err(); err != nil {
			t.Fatalf("i18nTranslate() returned error: %v", err)
		}
	})
}
