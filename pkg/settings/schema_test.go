// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package settings

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/version"
)

func TestSchema_Set(t *testing.T) {
	s := &Schema{
		settings: make(map[string]SettingSpec),
	}
	spec := SettingSpec{
		Key:        "test.key",
		Kind:       KindString,
		Mutability: SettingMutable,
	}
	err := s.set("test.key", spec)
	testutils.NoError(t, err)
	testutils.Assert(t, len(s.settings) > 0, "expected settings to be added")
}

func TestSchema_Set_InvalidSpec(t *testing.T) {
	s := &Schema{
		settings: make(map[string]SettingSpec),
	}
	spec := SettingSpec{
		Key:        "test.key",
		Kind:       KindString,
		Mutability: SettingImmutable + 1, // Invalid mutability
	}
	err := s.set("test.key", spec)
	testutils.Error(t, err)
}

func TestSchema_Profile(t *testing.T) {
	s := &Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Value:      "default",
			},
		},
	}
	
	profile, err := s.Profile("test", nil)
	testutils.NoError(t, err)
	testutils.NotNil(t, profile)
	testutils.Equal(t, "test", profile.Name())
}

func TestSchema_Profile_WithPreferences(t *testing.T) {
	s := &Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Value:      "default",
				Persistent: true,
			},
		},
	}
	
	prefs := NewPreferences(version.Version("v1.0.0"))
	prefs.Set("test.key", "preference value")
	
	profile, err := s.Profile("test", prefs)
	testutils.NoError(t, err)
	testutils.NotNil(t, profile)
}

func TestSchema_Profile_VersionMismatch(t *testing.T) {
	s := &Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}
	
	prefs := NewPreferences(version.Version("v2.0.0"))
	
	profile, err := s.Profile("test", prefs)
	testutils.Error(t, err)
	testutils.Nil(t, profile)
}

func TestSchema_SetID(t *testing.T) {
	s := &Schema{
		pkgSettingsStructName: "TestSettings",
		module:                "test.module",
		mode:                  ModeTesting,
	}
	s.setID()
	testutils.Assert(t, s.id != "", "expected ID to be set")
}

