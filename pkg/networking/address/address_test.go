// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package address

import "testing"

// TestZeroValueAddressParseDoesNotPanic is a regression test: calling
// Parse on a zero-value Address (no underlying url.URL set) panicked with
// a nil pointer dereference inside net/url, since (*url.URL).Parse cannot
// be called on a nil receiver. Parse must instead fall back to parsing ref
// directly when there is no base URL to resolve against.
func TestZeroValueAddressParseDoesNotPanic(t *testing.T) {
	var a Address
	got, err := a.Parse("happy://example.com/foo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Host() != "example.com" {
		t.Errorf("expected host %q, got %q", "example.com", got.Host())
	}
}

func TestFromModuleEmptyModulePath(t *testing.T) {
	a, err := FromModule("example.com", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a.Host() != "example.com" {
		t.Errorf("expected host %q, got %q", "example.com", a.Host())
	}
}

// TestFromModuleSingleSegmentHasReverseDNS is a regression test: FromModule
// early-returned for a single-segment module path (e.g. "hello", as produced
// by a bare `go mod init hello`) without ever setting Module()/ReverseDNS(),
// even though reverseDns("hello") correctly computes "hello". Since
// ReverseDNS() is the common way callers derive an app identifier (see
// sdk/app/internal/initializer), this broke app identifier derivation for
// the single most common `go mod init` invocation.
func TestFromModuleSingleSegmentHasReverseDNS(t *testing.T) {
	a, err := FromModule("example.com", "hello")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a.Module() != "hello" {
		t.Errorf("expected module %q, got %q", "hello", a.Module())
	}
	if a.ReverseDNS() != "hello" {
		t.Errorf("expected non-empty ReverseDNS() %q, got %q", "hello", a.ReverseDNS())
	}
}

func TestFromModuleAndResolveService(t *testing.T) {
	a, err := FromModule("example.com", "github.com/happy-sdk/happy")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a.Module() != "github.com/happy-sdk/happy" {
		t.Errorf("expected module %q, got %q", "github.com/happy-sdk/happy", a.Module())
	}

	svc, err := a.ResolveService("mysvc")
	if err != nil {
		t.Fatalf("expected no error resolving service, got %v", err)
	}
	if svc.Host() != "example.com" {
		t.Errorf("expected service host %q, got %q", "example.com", svc.Host())
	}
}
