// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package addon

import (
	"testing"

	"github.com/happy-sdk/happy/sdk/session"
)

func TestManagerAdd(t *testing.T) {
	m := NewManager()
	a := helloAddon("First")
	if err := m.Add(a); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if !a.attached {
		t.Error("expected addon to be marked attached")
	}
}

func TestManagerAddNil(t *testing.T) {
	m := NewManager()
	if err := m.Add(nil); err == nil {
		t.Error("expected an error adding a nil addon")
	}
}

func TestManagerAddInvalidSlug(t *testing.T) {
	m := NewManager()
	a := helloAddon("")
	a.info.Slug = "" // force an invalid slug regardless of auto-derivation
	if err := m.Add(a); err == nil {
		t.Error("expected an error adding an addon with an invalid slug")
	}
}

func TestManagerAddDuplicateSlug(t *testing.T) {
	m := NewManager()
	a1 := helloAddon("Dup")
	a1.WithConfig(Config{Slug: "dup"})
	a2 := helloAddon("Dup")
	a2.WithConfig(Config{Slug: "dup"})

	if err := m.Add(a1); err != nil {
		t.Fatalf("first Add failed: %v", err)
	}
	if err := m.Add(a2); err == nil {
		t.Error("expected an error adding a second addon with the same slug")
	}
}

// TestManagerPreservesRegistrationOrder is a regression test: every
// iteration method on Manager ranged over the addons map directly, and Go
// map iteration order is randomized, so the caller-supplied order from
// app.WithAddons(a, b, c) was lost. This adds many addons (enough that
// random map iteration would very likely disagree with insertion order at
// least once across these assertions) and checks every order-exposing
// method reports them in the order they were Add-ed.
func TestManagerPreservesRegistrationOrder(t *testing.T) {
	m := NewManager()
	var want []string
	for i := range 20 {
		a := helloAddon("Addon")
		slug := "addon-" + string(rune('a'+i))
		a.WithConfig(Config{Slug: slug})
		a.ProvideCommands(newTestCommand(t, slug+"-cmd"))
		a.ProvideServices(newTestService(slug + "-svc"))
		if err := m.Add(a); err != nil {
			t.Fatalf("Add(%s) failed: %v", slug, err)
		}
		want = append(want, slug)
	}

	var gotCmds []string
	for _, cmd := range m.Commands() {
		gotCmds = append(gotCmds, cmd.Name())
	}
	for i, slug := range want {
		expected := slug + "-cmd"
		if gotCmds[i] != expected {
			t.Fatalf("Commands()[%d] = %q, want %q (registration order not preserved)", i, gotCmds[i], expected)
		}
	}

	ordered := m.ordered()
	for i, slug := range want {
		if ordered[i].info.Slug != slug {
			t.Fatalf("ordered()[%d].info.Slug = %q, want %q (registration order not preserved)", i, ordered[i].info.Slug, slug)
		}
	}
}

func TestManagerExtendSettings(t *testing.T) {
	m := NewManager()
	a := helloAddon("Hello")
	if err := m.Add(a); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	// No addon.settings configured; Extend should simply be a no-op.
	if err := m.ExtendSettings(nil); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestManagerRegister(t *testing.T) {
	m := NewManager()
	a := helloAddon("Hello")

	called := false
	a.OnRegister(func(sess session.Register) error {
		called = true
		return nil
	})
	a.Deprecated("this addon is deprecated")

	if err := m.Add(a); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	sess, _, cleanup, err := session.CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	if err := m.Register(sess); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !called {
		t.Error("expected registerAction to be called")
	}
}

func TestManagerRegisterPropagatesPendingErrors(t *testing.T) {
	m := NewManager()
	a := helloAddon("Hello")
	a.ProvideAPI(nil) // records a pending error

	if err := m.Add(a); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	sess, _, cleanup, err := session.CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	if err := m.Register(sess); err == nil {
		t.Error("expected Register to propagate the addon's pending error")
	}
}

func TestManagerGetAPIs(t *testing.T) {
	m := NewManager()
	a := helloAddon("Hello")
	a.ProvideAPI(&testAPI{})
	if err := m.Add(a); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	apis := m.GetAPIs()
	if _, ok := apis[a.info.Slug]; !ok {
		t.Errorf("expected GetAPIs to include slug %q", a.info.Slug)
	}
}

func TestManagerConfigFiltersCommandsServicesEvents(t *testing.T) {
	m := NewManager()
	a := helloAddon("Hello")
	a.WithConfig(Config{
		WithoutCommands: true,
		WithoutServices: true,
		DiscardEvents:   true,
	})
	a.ProvideCommands(newTestCommand(t, "should-be-filtered"))
	a.ProvideServices(newTestService("should-be-filtered"))

	if err := m.Add(a); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if len(m.Commands()) != 0 {
		t.Error("expected Commands() to be empty when WithoutCommands is set")
	}
	if len(m.Services()) != 0 {
		t.Error("expected Services() to be empty when WithoutServices is set")
	}
	if len(m.Events()) != 0 {
		t.Error("expected Events() to be empty when DiscardEvents is set")
	}
}
