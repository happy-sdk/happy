// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/settings"
)

// configProfile builds the project config schema the same way loadConfig does,
// minus the git lookup, and applies prefs as if they had been read from a
// .happy.yaml.
func configProfile(t *testing.T, prefs map[string]string) (*settings.Profile, error) {
	t.Helper()

	bp, err := (&Config{}).Blueprint()
	if err != nil {
		t.Fatal(err)
	}
	schema, err := bp.Schema("project", ConfigVersion)
	if err != nil {
		t.Fatal(err)
	}

	pref := settings.NewPreferences(ConfigVersion)
	for k, v := range prefs {
		pref.Set(k, v)
	}
	return schema.Profile("test", pref)
}

func TestAgentConfigDefaults(t *testing.T) {
	profile, err := configProfile(t, nil)
	testutils.NoError(t, err)

	testutils.Equal(t, true, profile.Get("agent.enabled").Value().Bool(),
		"agent support is on by default; the files are the opt-in signal")
	testutils.Equal(t, ".happy/AGENTS.md", profile.Get("agent.instructions").String())
	testutils.Equal(t, ".happy/skills", profile.Get("agent.skills").String())
	testutils.Equal(t, ".happy/mcp.yaml", profile.Get("agent.mcp").String())
}

// The schema is strict: before agent: existed, a .happy.yaml declaring it
// failed to load with "preferences provided key(agent.…) not found", which
// breaks every happyctl command in that repository rather than being ignored.
func TestAgentConfigAcceptsManifestKeys(t *testing.T) {
	profile, err := configProfile(t, map[string]string{
		"agent.enabled":      "true",
		"agent.instructions": ".happy/AGENTS.md",
		"agent.skills":       "docs/skills",
		"agent.mcp":          "docs/mcp.yaml",
	})
	testutils.NoError(t, err)

	testutils.Equal(t, "docs/skills", profile.Get("agent.skills").String(),
		"an explicitly set path must override the default")
	testutils.Equal(t, "docs/mcp.yaml", profile.Get("agent.mcp").String())
}

func TestAgentConfigCanBeDisabled(t *testing.T) {
	profile, err := configProfile(t, map[string]string{"agent.enabled": "false"})
	testutils.NoError(t, err)

	testutils.Equal(t, false, profile.Get("agent.enabled").Value().Bool(),
		"a repository shipping the files must still be able to withhold them")
}

// Guards the property that made this section a toggle-and-paths schema: an
// unknown key is a hard error, not a warning, so structured definitions can
// never be inlined into .happy.yaml.
func TestConfigRejectsUnknownKey(t *testing.T) {
	_, err := configProfile(t, map[string]string{"agent.mcp.tools": "[]"})
	testutils.Error(t, err, "an unknown key must fail the whole profile")
}
