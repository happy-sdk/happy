// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package version

import (
	"regexp"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"v1.2.3", "v1.2.3", false},
		{"1.2.3", "v1.2.3", false}, // Should add "v" prefix
		{"v0.0.1-alpha", "v0.0.1-alpha", false},
		{"invalid", "", true},   // Invalid version
		{"v1.2", "v1.2", false}, // Missing patch version
		{"1.x.3", "", true},     // Invalid characters
	}

	for _, tt := range tests {
		v, err := Parse(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if v.String() != tt.expected {
			t.Errorf("Parse(%q) = %v, want %v", tt.input, v, tt.expected)
		}
	}
}

func TestVersionString(t *testing.T) {
	v := Version("v1.2.3+build.123")
	if v.String() != "v1.2.3+build.123" {
		t.Errorf("String() = %v, want %v", v.String(), "v1.2.3+build.123")
	}
}

func TestBuildMetadata(t *testing.T) {
	tests := []struct {
		input    Version
		expected string
	}{
		{"v1.2.3+build.123", "build.123"},
		{"v1.2.3", ""},
		{"v0.0.1-dev+git.abc123", "git.abc123"},
	}

	for _, tt := range tests {
		if got := tt.input.Build(); got != tt.expected {
			t.Errorf("Build() = %v, want %v", got, tt.expected)
		}
	}
}

func TestCurrentVersionFormat(t *testing.T) {
	v := Current().String()

	// The version should start with "v0.0.1-devel" or be a valid semver tag
	validVersion := regexp.MustCompile(`^v\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?(\+.*)?$`)
	if !validVersion.MatchString(v) {
		t.Errorf("Current() = %v, which is not a valid version format", v)
	}
}

func TestPrerelease(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.2.3-alpha", "alpha"},
		{"v1.2.3", ""},
		{"v0.0.1-dev", "dev"},
	}

	for _, tt := range tests {
		if got := Prerelease(tt.input); got != tt.expected {
			t.Errorf("Prerelease(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestIsDev(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"v0.0.1-devel", true},
		{"v1.2.3-alpha", false},
		{"v1.2.3", false},
	}

	for _, tt := range tests {
		if got := IsDev(tt.input); got != tt.expected {
			t.Errorf("IsDev(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
