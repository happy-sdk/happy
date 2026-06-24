// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package api_test

import (
	"testing"

	"github.com/happy-sdk/happy/sdk/api"
)

// MyAPI mirrors the exact embedding pattern documented on api.Provider:
// embedding the interface (left as its zero value) satisfies it for any
// consumer, since Provider's method is unexported and can't be implemented
// directly outside the api package.
type MyAPI struct {
	api.Provider
	Greeting string
}

// TestProviderEmbeddingSatisfiesInterface guards the documented embedding
// pattern: if api.Provider's definition ever changed in a way that broke
// this (e.g. requiring an exported method instead), this would fail to
// compile, catching the regression before it reaches addon authors who
// rely on the package doc's example.
func TestProviderEmbeddingSatisfiesInterface(t *testing.T) {
	var p api.Provider = &MyAPI{Greeting: "hello"}

	myAPI, ok := p.(*MyAPI)
	if !ok {
		t.Fatal("expected to assert back to *MyAPI")
	}
	if myAPI.Greeting != "hello" {
		t.Errorf("Greeting = %q, want %q", myAPI.Greeting, "hello")
	}
}

// TestProviderEmbeddingValueReceiver confirms the pattern also works for a
// value (non-pointer) type embedding api.Provider. The assertion is the
// compile itself: ValueAPI must satisfy api.Provider via the embedded zero
// value.
func TestProviderEmbeddingValueReceiver(t *testing.T) {
	type ValueAPI struct {
		api.Provider
	}
	var p api.Provider = ValueAPI{}
	if _, ok := p.(ValueAPI); !ok {
		t.Fatal("expected to assert back to ValueAPI")
	}
}
