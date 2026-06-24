// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

// newTestSessionInModule creates a test session whose app.module and
// app.fs.path.wd point at a temp directory containing a real go.mod with
// the given module path and require lines.
func newTestSessionInModule(t *testing.T, modulePath string, requireLines ...string) (*session.Context, func()) {
	t.Helper()

	dir := t.TempDir()
	gomod := "module " + modulePath + "\n\ngo 1.25\n"
	if len(requireLines) > 0 {
		gomod += "\nrequire (\n"
		for _, r := range requireLines {
			gomod += "\t" + r + "\n"
		}
		gomod += ")\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	sess, _, cleanup, err := session.CreateTestSession(nil,
		options.NewOption("app.module", modulePath),
		options.NewOption("app.fs.path.wd", dir),
	)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	return sess, cleanup
}

func TestFindModuleRoot(t *testing.T) {
	t.Run("module at working directory", func(t *testing.T) {
		sess, cleanup := newTestSessionInModule(t, "github.com/example/test")
		defer cleanup()

		root, err := findModuleRoot(sess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wd := sess.Opts().Get("app.fs.path.wd").String()
		if root != wd {
			t.Errorf("root = %q, want %q", root, wd)
		}
	})

	t.Run("app.module is a subdirectory of the module", func(t *testing.T) {
		sess, cleanup := newTestSessionInModule(t, "github.com/example/test")
		defer cleanup()

		// Override app.module to a subpackage of the same module; the go.mod
		// found at the working directory should still match via the
		// "subdirectory of module path" rule.
		if err := sess.Opts().Set("app.module", "github.com/example/test/cmd/sub"); err != nil {
			t.Fatalf("failed to override app.module: %v", err)
		}
		root, err := findModuleRoot(sess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root == "" {
			t.Error("expected a non-empty module root")
		}
	})

	t.Run("no app.module set", func(t *testing.T) {
		sess, _, cleanup, err := session.CreateTestSession(nil)
		if err != nil {
			t.Fatalf("failed to create test session: %v", err)
		}
		defer cleanup()

		_, err = findModuleRoot(sess)
		if err == nil {
			t.Error("expected an error when app.module is not set")
		}
	})

	t.Run("no go.mod found", func(t *testing.T) {
		dir := t.TempDir()
		sess, _, cleanup, err := session.CreateTestSession(nil,
			options.NewOption("app.module", "github.com/example/test"),
			options.NewOption("app.fs.path.wd", dir),
		)
		if err != nil {
			t.Fatalf("failed to create test session: %v", err)
		}
		defer cleanup()

		_, err = findModuleRoot(sess)
		if err == nil {
			t.Error("expected an error when no go.mod is found")
		}
	})
}

func TestGetAppModulePrefix(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/banctl")
	defer cleanup()

	prefix, err := getAppModulePrefix(sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "com.github.happy-sdk.banctl"
	if prefix != want {
		t.Errorf("getAppModulePrefix() = %q, want %q", prefix, want)
	}
}

func TestGetDependencyIdentifiers(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/banctl",
		"github.com/some/dependency v1.0.0",
		"github.com/another/dep v1.0.0",
	)
	defer cleanup()

	deps, err := getDependencyIdentifiers(sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d: %v", len(deps), deps)
	}
	if !deps["com.github.some.dependency"] {
		t.Errorf("expected dependency identifier for github.com/some/dependency, got %v", deps)
	}
}

func TestGetDependencyIdentifiersExcludesOwnModule(t *testing.T) {
	// A module that (unusually) requires itself should not list itself as
	// a dependency.
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/banctl",
		"github.com/happy-sdk/banctl v1.0.0",
		"github.com/real/dependency v1.0.0",
	)
	defer cleanup()

	deps, err := getDependencyIdentifiers(sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps["com.github.happy-sdk.banctl"] {
		t.Error("expected the module's own identifier to be excluded from dependencies")
	}
	if !deps["com.github.real.dependency"] {
		t.Errorf("expected real dependency to be present, got %v", deps)
	}
}

func TestIsDependencyKey(t *testing.T) {
	sess, cleanup := newTestSessionInModule(t, "github.com/happy-sdk/banctl",
		"github.com/some/dependency v1.0.0",
	)
	defer cleanup()

	isDep, err := isDependencyKey(sess, "com.github.some.dependency.feature.key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isDep {
		t.Error("expected key to be identified as a dependency key")
	}

	isDep, err = isDependencyKey(sess, "com.github.happy-sdk.banctl.own.key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isDep {
		t.Error("expected own-module key to not be identified as a dependency key")
	}
}

func TestGetAppSupportedLanguages(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		sess, _, cleanup, err := session.CreateTestSession(nil)
		if err != nil {
			t.Fatalf("failed to create test session: %v", err)
		}
		defer cleanup()

		got := getAppSupportedLanguages(sess)
		if got != nil {
			t.Errorf("expected nil when not set, got %v", got)
		}
	})

	t.Run("configured via settings", func(t *testing.T) {
		sess, _, cleanup, err := session.CreateTestSession(&appI18nSettings{})
		if err != nil {
			t.Fatalf("failed to create test session: %v", err)
		}
		defer cleanup()

		got := getAppSupportedLanguages(sess)
		want := []language.Tag{language.English, language.French, language.German}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got[%d] = %v, want %v", i, got[i], want[i])
			}
		}
	})
}

// appI18nSettings mirrors the real app.i18n.supported nesting (see the root
// happy.Settings.I18n field) with the minimum needed for
// getAppSupportedLanguages, without requiring the full happy.Settings
// schema (which has unrelated mandatory fields like app.slug).
type appI18nSettings struct {
	I18n appI18nSubSettings `key:"app.i18n"`
}

func (s appI18nSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(&s)
}

type appI18nSubSettings struct {
	Supported settings.StringSlice `key:"supported" default:"en\x1ffr\x1fde"`
}

func (s appI18nSubSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(&s)
}
