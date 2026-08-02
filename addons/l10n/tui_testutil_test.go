// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/i18n"
	"golang.org/x/text/language"
)

// ensureI18nInitialized makes sure pkg/i18n has run Initialize at least
// once in this test binary. RegisterTranslation only applies immediately
// once the manager is initialized - before that it silently queues (see
// manager.registerTranslation), so a test that registers a key and expects
// i18n.GetAllTranslations()/GetTranslationReport to see it right away must
// call this first. Safe to call from multiple tests: Initialize is
// idempotent enough for this purpose (a manager already initialized just
// re-runs bundle loading/pending-queue flushing).
func ensureI18nInitialized(t *testing.T) {
	t.Helper()
	i18n.Initialize(language.English)
}
