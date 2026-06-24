// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package config

import "testing"

func TestDefaultCommandConfig(t *testing.T) {
	cnf := DefaultCommandConfig()

	if cnf.Name != "config" {
		t.Errorf("Name = %q, want %q", cnf.Name, "config")
	}
	if cnf.HideDefaultUsage {
		t.Error("expected HideDefaultUsage to default to false")
	}
	for _, without := range []bool{
		cnf.WithoutLsCommand, cnf.WithoutGetCommand, cnf.WithoutSetCommand,
		cnf.WithoutAddCommand, cnf.WithoutRemoveCommand, cnf.WithoutResetCommand,
	} {
		if without {
			t.Error("expected all subcommands to be enabled by default")
		}
	}
}

func TestCommandBuildsWithoutError(t *testing.T) {
	cmd := Command(DefaultCommandConfig())
	if err := cmd.Err(); err != nil {
		t.Fatalf("Command() with default config returned error: %v", err)
	}
	if cmd.Name() != "config" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "config")
	}
}

func TestCommandWithCustomName(t *testing.T) {
	cnf := DefaultCommandConfig()
	cnf.Name = "settings"
	cmd := Command(cnf)
	if err := cmd.Err(); err != nil {
		t.Fatalf("Command() with custom name returned error: %v", err)
	}
	if cmd.Name() != "settings" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "settings")
	}
}

// TestCommandWithoutSubcommands confirms every WithoutXCommand flag is
// honored at construction time without causing a build error -- a
// regression here would mean a future addon/app disabling a subcommand
// breaks command construction entirely instead of just omitting it.
func TestCommandWithoutSubcommands(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*CommandConfig)
	}{
		{"without ls", func(c *CommandConfig) { c.WithoutLsCommand = true }},
		{"without get", func(c *CommandConfig) { c.WithoutGetCommand = true }},
		{"without set", func(c *CommandConfig) { c.WithoutSetCommand = true }},
		{"without add", func(c *CommandConfig) { c.WithoutAddCommand = true }},
		{"without remove", func(c *CommandConfig) { c.WithoutRemoveCommand = true }},
		{"without reset", func(c *CommandConfig) { c.WithoutResetCommand = true }},
		{"without all", func(c *CommandConfig) {
			c.WithoutLsCommand = true
			c.WithoutGetCommand = true
			c.WithoutSetCommand = true
			c.WithoutAddCommand = true
			c.WithoutRemoveCommand = true
			c.WithoutResetCommand = true
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

func TestCommandWithHiddenAndDisabledKeys(t *testing.T) {
	cnf := DefaultCommandConfig()
	cnf.HideKeys = []string{"app.secret.key"}
	cnf.DisableKeys = []string{"app.internal.key"}
	cnf.Secrets = []string{"app.password"}
	cnf.SecretsPassword = "test-password"

	cmd := Command(cnf)
	if err := cmd.Err(); err != nil {
		t.Fatalf("Command() with hidden/disabled/secret keys returned error: %v", err)
	}
}
