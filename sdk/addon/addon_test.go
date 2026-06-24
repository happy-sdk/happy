// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package addon

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/services"
	"github.com/happy-sdk/happy/sdk/services/service"
)

// helloAddon mirrors how a real addon is constructed: New is called
// directly from this package's own top-level helper function, so
// loadPackageInfo's caller-detection attributes the addon to this test
// package.
func helloAddon(name string) *Addon {
	return New(name)
}

func TestNewLoadsPackageInfo(t *testing.T) {
	a := helloAddon("Hello")
	if len(a.errs) > 0 {
		t.Fatalf("unexpected errors from New: %v", a.errs)
	}
	if a.info.Module == "" {
		t.Error("expected info.Module to be set")
	}
	if a.info.Name != "Hello" {
		t.Errorf("info.Name = %q, want %q", a.info.Name, "Hello")
	}
	if a.info.Slug == "" {
		t.Error("expected info.Slug to be auto-derived")
	}
}

func TestWithConfigSetsSlug(t *testing.T) {
	a := helloAddon("Hello")
	a.WithConfig(Config{Slug: "custom-slug"})
	if a.info.Slug != "custom-slug" {
		t.Errorf("info.Slug = %q, want %q", a.info.Slug, "custom-slug")
	}
}

func TestProvideAPI(t *testing.T) {
	a := helloAddon("Hello")
	a.ProvideAPI(&testAPI{})
	if a.api == nil {
		t.Fatal("expected api to be set")
	}

	// A second ProvideAPI call must be rejected.
	a.ProvideAPI(&testAPI{})
	if len(a.errs) == 0 {
		t.Error("expected an error for setting a second API")
	}
}

func TestProvideAPINil(t *testing.T) {
	a := helloAddon("Hello")
	a.ProvideAPI(nil)
	if len(a.errs) == 0 {
		t.Error("expected an error for a nil API")
	}
}

func TestProvideCommandsNil(t *testing.T) {
	a := helloAddon("Hello")
	a.ProvideCommands(nil)
	if len(a.errs) == 0 {
		t.Error("expected an error for a nil command")
	}
}

func TestProvideServicesNil(t *testing.T) {
	a := helloAddon("Hello")
	a.ProvideServices(nil)
	if len(a.errs) == 0 {
		t.Error("expected an error for a nil service")
	}
}

func TestWithEventsNil(t *testing.T) {
	a := helloAddon("Hello")
	a.WithEvents(nil)
	if len(a.errs) == 0 {
		t.Error("expected an error for a nil event")
	}
}

// TestMethodsNoOpAfterAttached is a regression-style test for
// tryConfigureAttached: once an addon is attached to a Manager,
// configuration methods must reject further changes instead of silently
// applying them.
func TestMethodsNoOpAfterAttached(t *testing.T) {
	a := helloAddon("Hello")
	m := NewManager()
	if err := m.Add(a); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	preErrs := len(a.errs)
	a.WithConfig(Config{Slug: "changed"})
	if a.info.Slug == "changed" {
		t.Error("WithConfig should not apply after attached")
	}
	if len(a.errs) <= preErrs {
		t.Error("expected an additional error from WithConfig after attached")
	}
}

func TestDeprecated(t *testing.T) {
	a := helloAddon("Hello")
	a.Deprecated("use NewV2 instead")
	if len(a.deprecations) != 1 {
		t.Fatalf("expected 1 deprecation, got %d", len(a.deprecations))
	}
}

type testAPI struct {
	api.Provider
}

func newTestCommand(t *testing.T, name string) *command.Command {
	t.Helper()
	return command.New(name, command.Config{Description: "test command"})
}

func newTestService(name string) *services.Service {
	return services.New(service.Config{Name: settings.String(name)})
}
