// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gomodule

import "testing"

func TestParseBumpKind(t *testing.T) {
	tests := []struct {
		in      string
		want    BumpKind
		wantErr bool
	}{
		{"major", BumpKindMajor, false},
		{"minor", BumpKindMinor, false},
		{"patch", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseBumpKind(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBumpStrategy(t *testing.T) {
	tests := []struct {
		in      string
		want    BumpStrategy
		wantErr bool
	}{
		{"single", BumpStrategySingle, false},
		{"hundred", BumpStrategyHundred, false},
		{"thousand", BumpStrategyThousand, false},
		{"million", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseBumpStrategy(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNextBumpValue(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		strategy BumpStrategy
		want     int
	}{
		{"single increments by one", 55, BumpStrategySingle, 56},
		{"hundred jumps to next full hundred", 55, BumpStrategyHundred, 100},
		{"hundred on exact boundary jumps to the next one", 100, BumpStrategyHundred, 200},
		{"hundred at zero jumps to 100", 0, BumpStrategyHundred, 100},
		{"thousand jumps to next full thousand", 55, BumpStrategyThousand, 1000},
		{"thousand on exact boundary jumps to the next one", 1000, BumpStrategyThousand, 2000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextBumpValue(tt.current, tt.strategy); got != tt.want {
				t.Errorf("nextBumpValue(%d, %q) = %d, want %d", tt.current, tt.strategy, got, tt.want)
			}
		})
	}
}

func TestBumpSignificant(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		tag      string
		kind     BumpKind
		strategy BumpStrategy
		want     string
	}{
		{"major single", "", "v1.55.3", BumpKindMajor, BumpStrategySingle, "v2.0.0"},
		{"minor single", "", "v1.55.3", BumpKindMinor, BumpStrategySingle, "v1.56.0"},
		{"minor hundred", "", "v1.55.3", BumpKindMinor, BumpStrategyHundred, "v1.100.0"},
		{"minor thousand", "", "v1.55.3", BumpKindMinor, BumpStrategyThousand, "v1.1000.0"},
		{"major hundred", "", "v55.3.1", BumpKindMajor, BumpStrategyHundred, "v100.0.0"},
		{"with tag prefix", "pkg/mod/", "pkg/mod/v1.55.3", BumpKindMinor, BumpStrategyHundred, "pkg/mod/v1.100.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bumpSignificant(tt.prefix, tt.tag, tt.kind, tt.strategy)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("bumpSignificant(%q, %q, %q, %q) = %q, want %q", tt.prefix, tt.tag, tt.kind, tt.strategy, got, tt.want)
			}
		})
	}
}

func TestBumpSignificantInvalidTag(t *testing.T) {
	_, err := bumpSignificant("", "not-a-version", BumpKindMajor, BumpStrategySingle)
	if err == nil {
		t.Fatal("expected an error for a malformed tag")
	}
}

func TestBumpSignificantInvalidComponents(t *testing.T) {
	t.Run("non-numeric major", func(t *testing.T) {
		_, err := bumpSignificant("", "vX.1.0", BumpKindMajor, BumpStrategySingle)
		if err == nil {
			t.Fatal("expected an error for a non-numeric major component")
		}
	})
	t.Run("non-numeric minor", func(t *testing.T) {
		_, err := bumpSignificant("", "v1.X.0", BumpKindMinor, BumpStrategySingle)
		if err == nil {
			t.Fatal("expected an error for a non-numeric minor component")
		}
	})
}
