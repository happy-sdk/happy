// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package i18n

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/i18n"
	"golang.org/x/text/language"
)

// TestTranslationsRegistered confirms this package's init() successfully
// embedded and registered its translation JSON files: a known key from
// en.json must resolve to its real translation, not silently fall back to
// the raw key (which is what an unregistered/failed-to-load key returns).
func TestTranslationsRegistered(t *testing.T) {
	i18n.Initialize(language.English)

	const key = "com.github.happy-sdk.happy.sdk.cli.cmd.config.category"
	got := i18n.T(key)
	if got == key {
		t.Errorf("T(%q) returned the raw key, expected a registered translation", key)
	}
	if got != "Configuration" {
		t.Errorf("T(%q) = %q, want %q", key, got, "Configuration")
	}
}
