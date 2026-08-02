// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

// newDashboardTestSession registers a couple of app translation keys (one
// fully translated, one missing German) plus a dependency key, and returns
// a session in a temp module rooted appropriately for getAppModulePrefix/
// getDependencyIdentifiers to resolve against it.
func newDashboardTestSession(t *testing.T) *session.Context {
	t.Helper()
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/banctl",
		"github.com/some/dependency v1.0.0",
	)
	t.Cleanup(cleanup)
	ensureI18nInitialized(t)

	mustRegister := func(lang language.Tag, key, value string) {
		if err := i18n.RegisterTranslation(lang, key, value); err != nil {
			t.Fatalf("RegisterTranslation(%s, %s) failed: %v", lang, key, err)
		}
	}
	mustRegister(language.English, "com.github.happy-sdk.banctl.greeting", "Hello")
	mustRegister(language.French, "com.github.happy-sdk.banctl.greeting", "Bonjour")
	mustRegister(language.English, "com.github.some.dependency.label", "Label")

	return sess
}

func TestComputeDashboardRowsAppOnly(t *testing.T) {
	sess := newDashboardTestSession(t)

	rows := computeDashboardRows(sess, false)
	if len(rows) == 0 {
		t.Fatal("expected at least one row")
	}
	for _, row := range rows {
		if row.Total == 0 {
			continue
		}
		wantPct := float64(row.Translated) / float64(row.Total) * 100.0
		if row.Translated+row.Missing != row.Total {
			t.Errorf("%s: translated(%d)+missing(%d) != total(%d)", row.Language, row.Translated, row.Missing, row.Total)
		}
		if diff := row.Percentage - wantPct; diff > 0.01 || diff < -0.01 {
			t.Errorf("%s: percentage = %v, want %v", row.Language, row.Percentage, wantPct)
		}
	}
}

// TestComputeDashboardRowsReflectsRegisteredTranslation proves rows are
// computed fresh from the current i18n state rather than cached: since
// this test package's own tests (and the rest of the process it links
// against) register many real translation keys against the same global
// i18n registry, absolute counts aren't reliable here - the only safe
// assertion is the delta a single new registration produces.
func TestComputeDashboardRowsReflectsRegisteredTranslation(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/dashboardrowsdelta")
	defer cleanup()
	ensureI18nInitialized(t)

	// A key unique to this run: pkg/i18n's registry is process-global, and
	// -count > 1 (or a shared `go test` process across packages) re-runs
	// this test function in the same process, where a fixed literal key
	// would already exist from a prior iteration and never register as
	// "new" again.
	key := fmt.Sprintf("com.github.happy-sdk.dashboardrowsdelta.unique.greeting%d", time.Now().UnixNano())
	frenchTotal := func(rows []dashboardRow) (total, missing int) {
		for _, r := range rows {
			if r.Language == language.French {
				return r.Total, r.Missing
			}
		}
		return 0, 0
	}

	beforeTotal, _ := frenchTotal(computeDashboardRows(sess, false))

	if err := i18n.RegisterTranslation(language.English, key, "Hello"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	afterEnglishTotal, afterEnglishMissing := frenchTotal(computeDashboardRows(sess, false))
	if afterEnglishTotal != beforeTotal+1 {
		t.Fatalf("expected total to grow by exactly 1 after registering a new key, got %d -> %d", beforeTotal, afterEnglishTotal)
	}
	if afterEnglishMissing == 0 {
		t.Fatal("expected the new key to count as missing for French until French is registered too")
	}

	if err := i18n.RegisterTranslation(language.French, key, "Bonjour"); err != nil {
		t.Fatalf("RegisterTranslation failed: %v", err)
	}
	_, afterFrenchMissing := frenchTotal(computeDashboardRows(sess, false))
	if afterFrenchMissing != afterEnglishMissing-1 {
		t.Errorf("expected missing count to drop by exactly 1 once French was registered, got %d -> %d", afterEnglishMissing, afterFrenchMissing)
	}
}

func TestComputeDashboardRowsWithDepsIncludesMore(t *testing.T) {
	sess := newDashboardTestSession(t)

	appOnly := computeDashboardRows(sess, false)
	withDeps := computeDashboardRows(sess, true)

	var appTotal, depTotal int
	for _, r := range appOnly {
		if r.Language == language.French {
			appTotal = r.Total
		}
	}
	for _, r := range withDeps {
		if r.Language == language.French {
			depTotal = r.Total
		}
	}
	if depTotal <= appTotal {
		t.Errorf("expected --with-deps total (%d) to exceed app-only total (%d)", depTotal, appTotal)
	}
}

func TestDashboardModelToggleDepsKey(t *testing.T) {
	sess := newDashboardTestSession(t)
	m := newDashboardModel(sess, false)

	if m.withDeps {
		t.Fatal("expected withDeps to start false")
	}

	newM, cmd := m.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	if cmd != nil {
		t.Error("expected no command from toggling deps")
	}
	if !newM.withDeps {
		t.Error("expected 'd' to toggle withDeps on")
	}
}

func TestDashboardModelViewRendersLanguagesAndPercentages(t *testing.T) {
	sess := newDashboardTestSession(t)
	m := newDashboardModel(sess, false)

	content := m.View().Content
	if !strings.Contains(content, "%") {
		t.Errorf("expected view to render a percentage, got:\n%s", content)
	}
	if !strings.Contains(content, "fr") {
		t.Errorf("expected view to mention the French language row, got:\n%s", content)
	}
}

func TestDashboardModelViewNoLanguages(t *testing.T) {
	sess, _, cleanup, err := session.CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// A dashboard for a language set with nothing registered still renders
	// without panicking, just with an explanatory message instead of rows.
	m := dashboardModel{sess: sess, bar: newDashboardModel(sess, false).bar}
	content := m.View().Content
	if content == "" {
		t.Error("expected a non-empty view even with zero rows")
	}
}

func TestDashboardModelWindowSizeMsg(t *testing.T) {
	sess := newDashboardTestSession(t)
	m := newDashboardModel(sess, false)

	newM, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if newM.width != 120 {
		t.Errorf("width = %d, want 120", newM.width)
	}
}
