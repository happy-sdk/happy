// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package cli

import "testing"

func TestBuiltinFlags(t *testing.T) {
	tests := []struct {
		name       string
		create     Flag
		wantName   string
		wantAlias  string
		wantHidden bool
	}{
		{"version", FlagVersion, "version", "", false},
		{"help", FlagHelp, "help", "h", false},
		{"show-exec", FlagX, "show-exec", "x", false},
		{"system-debug", FlagSystemDebug, "system-debug", "", false},
		{"debug", FlagDebug, "debug", "", false},
		{"verbose", FlagVerbose, "verbose", "v", false},
		{"x-prod", FlagXProd, "x-prod", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := tt.create()
			if err != nil {
				t.Fatalf("unexpected error creating flag: %v", err)
			}
			if f.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", f.Name(), tt.wantName)
			}
			if tt.wantAlias != "" {
				aliases := f.Aliases()
				found := false
				for _, a := range aliases {
					if a == tt.wantAlias {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected alias %q in %v", tt.wantAlias, aliases)
				}
			}
			if f.Hidden() != tt.wantHidden {
				t.Errorf("Hidden() = %v, want %v", f.Hidden(), tt.wantHidden)
			}
			// Default for all builtin flags is bool false.
			if f.Default().String() != "false" {
				t.Errorf("Default() = %q, want %q", f.Default().String(), "false")
			}
		})
	}
}

func TestNewStringFlag(t *testing.T) {
	f, err := NewStringFlag("name", "default", "usage", "n")()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Name() != "name" {
		t.Errorf("Name() = %q, want %q", f.Name(), "name")
	}
	if f.Default().String() != "default" {
		t.Errorf("Default() = %q, want %q", f.Default().String(), "default")
	}
}

func TestNewBoolFlag(t *testing.T) {
	f, err := NewBoolFlag("flag", true, "usage")()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Default().String() != "true" {
		t.Errorf("Default() = %q, want %q", f.Default().String(), "true")
	}
}

func TestNewUintFlag(t *testing.T) {
	f, err := NewUintFlag("flag", 42, "usage")()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Default().String() != "42" {
		t.Errorf("Default() = %q, want %q", f.Default().String(), "42")
	}
}

func TestNewIntFlag(t *testing.T) {
	f, err := NewIntFlag("flag", -7, "usage")()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Default().String() != "-7" {
		t.Errorf("Default() = %q, want %q", f.Default().String(), "-7")
	}
}

func TestNewFloat64Flag(t *testing.T) {
	f, err := NewFloat64Flag("flag", 3.14, "usage")()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Default().String() != "3.14" {
		t.Errorf("Default() = %q, want %q", f.Default().String(), "3.14")
	}
}

func TestNewDurationFlag(t *testing.T) {
	f, err := NewDurationFlag("flag", 0, "usage")()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Name() != "flag" {
		t.Errorf("Name() = %q, want %q", f.Name(), "flag")
	}
}

func TestNewOptionFlag(t *testing.T) {
	f, err := NewOptionFlag("flag", []string{"a"}, []string{"a", "b", "c"}, "usage")()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Name() != "flag" {
		t.Errorf("Name() = %q, want %q", f.Name(), "flag")
	}
}
