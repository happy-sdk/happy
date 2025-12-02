// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package varflag

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

// TestPositionalGlobalFlag tests that global flags work in all positions:
// 1. Before command: -v cmd
// 2. After command: cmd -v
// 3. After subcommand: cmd subcmd -v
func TestPositionalGlobalFlag(t *testing.T) {
	binName := filepath.Base(os.Args[0])

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "flag before command",
			args:     []string{binName, "-v", "cmd", "arg"},
			expected: true,
		},
		{
			name:     "flag after command",
			args:     []string{binName, "cmd", "-v", "arg"},
			expected: true,
		},
		{
			name:     "flag after subcommand",
			args:     []string{binName, "cmd", "subcmd", "-v", "arg"},
			expected: true,
		},
		{
			name:     "flag with alias before command",
			args:     []string{binName, "--verbose", "cmd", "arg"},
			expected: true,
		},
		{
			name:     "flag with alias after command",
			args:     []string{binName, "cmd", "--verbose", "arg"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create root flag set using "/" which gets converted to binary name
			root, err := NewFlagSet("/", 0)
			testutils.NoError(t, err)

			// Create global verbose flag
			verbose, err := Bool("verbose", false, "enable verbose output", "v")
			testutils.NoError(t, err)

			// Add to root
			testutils.NoError(t, root.Add(verbose))

			// Create command flag set
			cmd, err := NewFlagSet("cmd", 1)
			testutils.NoError(t, err)

			// Add global flag to command (simulating how global flags are added to subcommands)
			testutils.NoError(t, cmd.Add(verbose))

			// Create subcommand flag set
			subcmd, err := NewFlagSet("subcmd", 1)
			testutils.NoError(t, err)

			// Add global flag to subcommand
			testutils.NoError(t, subcmd.Add(verbose))

			// Add subcommand to command
			testutils.NoError(t, cmd.AddSet(subcmd))

			// Add command to root
			testutils.NoError(t, root.AddSet(cmd))

			// Parse args
			err = root.Parse(tt.args)
			testutils.NoError(t, err)

			// Check if verbose flag is present
			if verbose.Present() != tt.expected {
				t.Errorf("expected verbose.Present() to be %v, got %v", tt.expected, verbose.Present())
			}

			if verbose.Present() && !verbose.Value() {
				t.Error("expected verbose.Value() to be true when present")
			}
		})
	}
}

// TestPositionalCommandFlag tests that flags attached to commands work when they appear after the command
func TestPositionalCommandFlag(t *testing.T) {
	binName := filepath.Base(os.Args[0])

	tests := []struct {
		name        string
		args        []string
		expected    bool
		expectError bool
	}{
		{
			name:        "command flag after command",
			args:        []string{binName, "cmd", "--cmdflag", "value"},
			expected:    true,
			expectError: false,
		},
		{
			name:        "command flag before command (should error)",
			args:        []string{binName, "--cmdflag", "value", "cmd"},
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create root flag set using "/" which gets converted to binary name
			root, err := NewFlagSet("/", 0)
			testutils.NoError(t, err)

			// Create command flag set
			cmd, err := NewFlagSet("cmd", 1)
			testutils.NoError(t, err)

			// Create flag attached to command
			cmdflag, err := New("cmdflag", "", "flag for cmd")
			testutils.NoError(t, err)
			cmdflag.AttachTo("cmd")

			// Add flag to command
			testutils.NoError(t, cmd.Add(cmdflag))

			// Add command to root
			testutils.NoError(t, root.AddSet(cmd))

			// Parse args
			err = root.Parse(tt.args)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			testutils.NoError(t, err)

			// Check if flag is present
			if cmdflag.Present() != tt.expected {
				t.Errorf("expected cmdflag.Present() to be %v, got %v", tt.expected, cmdflag.Present())
			}
		})
	}
}

