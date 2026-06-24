// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package branding

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/tui/ansicolor"
)

func TestBuildBasic(t *testing.T) {
	brand, err := New(Info{
		Name:        "Test App",
		Version:     "1.0.0",
		Slug:        "test-app",
		Description: "a test application",
	}).Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if brand.Info().Name != "Test App" {
		t.Errorf("Info().Name = %q, want %q", brand.Info().Name, "Test App")
	}
	if brand.Info().Slug != "test-app" {
		t.Errorf("Info().Slug = %q, want %q", brand.Info().Slug, "test-app")
	}
}

// TestBuildEmptyInfoSucceeds documents that Build deliberately accepts an
// empty Info: some bootstrap paths (e.g. under testing.Testing(), see
// sdk/app/internal/initializer) construct a Brand before a name or slug
// has been resolved, and Build erroring on that case breaks them.
func TestBuildEmptyInfoSucceeds(t *testing.T) {
	brand, err := New(Info{}).Build()
	if err != nil {
		t.Fatalf("expected no error for empty Info, got %v", err)
	}
	if brand == nil {
		t.Fatal("expected a non-nil brand")
	}
}

func TestBuildNameOnlyOrSlugOnlySucceeds(t *testing.T) {
	if _, err := New(Info{Name: "Test"}).Build(); err != nil {
		t.Errorf("expected no error for name-only Info, got %v", err)
	}
	if _, err := New(Info{Slug: "test"}).Build(); err != nil {
		t.Errorf("expected no error for slug-only Info, got %v", err)
	}
}

func TestBuilderWithANSI(t *testing.T) {
	theme := ansicolor.Theme{Primary: ansicolor.RGB(1, 2, 3)}
	brand, err := New(Info{Name: "Test"}).WithANSI(theme).Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if brand.ANSI().Primary.RGB() != theme.Primary.RGB() {
		t.Error("ANSI() did not return the theme set via WithANSI")
	}
}

func TestBuilderWithPalette(t *testing.T) {
	palette := ColorPalette{Primary: "#0000ff", Success: "#00ff00"}
	brand, err := New(Info{Name: "Test"}).WithPalette(palette).Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if brand.Colors() != palette {
		t.Error("Colors() did not return the palette set via WithPalette")
	}
}

func TestDefaultANSITheme(t *testing.T) {
	brand, err := New(Info{Name: "Test"}).Build()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// New() seeds the builder with a default, valid ANSI theme even if
	// WithANSI is never called.
	if !brand.ANSI().Primary.Valid() {
		t.Error("expected default ANSI theme to have a valid Primary color")
	}
}
