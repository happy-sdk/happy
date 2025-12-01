// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package settings

import (
	"fmt"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/version"
	"golang.org/x/text/language"
)

func TestProfile_Name(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	profile, err := schema.Profile("test-profile", nil)
	testutils.NoError(t, err)
	testutils.Equal(t, "test-profile", profile.Name())
}

func TestProfile_Lang(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)
	testutils.Equal(t, language.English, profile.Lang())
}

func TestProfile_Loaded(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)
	testutils.Assert(t, profile.Loaded(), "expected profile to be loaded")
}

func TestProfile_Version(t *testing.T) {
	v := version.Version("v1.0.0")
	schema := Schema{
		version: v,
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)
	testutils.Equal(t, v, profile.Version())
}

func TestProfile_Get(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Value:      "default value",
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	setting := profile.Get("test.key")
	testutils.Assert(t, setting.Key() != "", "expected setting to have key")
}

func TestProfile_Get_NotFound(t *testing.T) {
	schema := Schema{
		version:  version.Version("v1.0.0"),
		settings: map[string]SettingSpec{},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	setting := profile.Get("nonexistent.key")
	testutils.Equal(t, "", setting.Key())
}

func TestProfile_Has(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	testutils.Assert(t, profile.Has("test.key"), "expected profile to have key")
	testutils.Assert(t, !profile.Has("nonexistent.key"), "expected profile not to have nonexistent key")
}

func TestProfile_Set(t *testing.T) {
	schema := Schema{
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

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.Set("test.key", "new value")
	testutils.NoError(t, err)
	testutils.Assert(t, profile.Changed(), "expected profile to be changed")
}

func TestProfile_Set_NotFound(t *testing.T) {
	schema := Schema{
		version:  version.Version("v1.0.0"),
		settings: map[string]SettingSpec{},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.Set("nonexistent.key", "value")
	testutils.Error(t, err)
}

func TestProfile_Set_Immutable(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingImmutable,
				Value:      "default",
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.Set("test.key", "new value")
	testutils.Error(t, err)
}

func TestProfile_Set_SettingOnce(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingOnce,
				Value:      "default",
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	// First set should succeed
	err = profile.Set("test.key", "first value")
	testutils.NoError(t, err)

	// Second set should fail
	err = profile.Set("test.key", "second value")
	testutils.Error(t, err)
}

func TestProfile_Changed(t *testing.T) {
	schema := Schema{
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

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	testutils.Assert(t, !profile.Changed(), "expected profile not to be changed initially")

	err = profile.Set("test.key", "new value")
	testutils.NoError(t, err)
	testutils.Assert(t, profile.Changed(), "expected profile to be changed after Set")
}

func TestProfile_ValidatePreference(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Persistent: true,
				Value:      "default",
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.ValidatePreference("test.key", "preference value")
	testutils.NoError(t, err)
}

func TestProfile_ValidatePreference_NotPersistent(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Persistent: false,
				Value:      "default",
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.ValidatePreference("test.key", "value")
	testutils.Error(t, err)
}

func TestProfile_Preferences(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Persistent: true,
				Value:      "default",
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.Set("test.key", "user value")
	testutils.NoError(t, err)

	prefs := profile.Preferences()
	testutils.NotNil(t, prefs)
	testutils.Equal(t, schema.version, prefs.SchemaVersion())
}

func TestProfile_All(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"key1": {
				Key:        "key1",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
			"key2": {
				Key:        "key2",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	count := 0
	for range profile.All() {
		count++
	}
	testutils.Equal(t, 2, count)
}

func TestProfile_PackageSettingsStructName(t *testing.T) {
	schema := Schema{
		version:               version.Version("v1.0.0"),
		pkgSettingsStructName: "TestSettings",
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)
	testutils.Equal(t, "TestSettings", profile.PackageSettingsStructName())
}

func TestProfile_Module(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		module:  "test.module",
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)
	testutils.Equal(t, "test.module", profile.Module())
}

func TestProfile_Load_AlreadyLoaded(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	// Try to load again
	err = profile.load(nil)
	testutils.Error(t, err)
}

func TestProfile_Load_WithMigrations(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"new.key": {
				Key:        "new.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Persistent: true,
			},
		},
		migrations: map[string]string{
			"old.key": "new.key",
		},
	}

	prefs := NewPreferences(version.Version("v1.0.0"))
	prefs.Set("old.key", "migrated value")

	profile, err := schema.Profile("test", prefs)
	testutils.NoError(t, err)

	setting := profile.Get("new.key")
	testutils.Equal(t, "migrated value", setting.String())
}

func TestProfile_Load_WithPreferences_NotFound(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
			},
		},
	}

	prefs := NewPreferences(version.Version("v1.0.0"))
	prefs.Set("nonexistent.key", "value")

	_, err := schema.Profile("test", prefs)
	// Should not error, but should clear the nonexistent key
	testutils.NoError(t, err)
	testutils.Equal(t, "", prefs.data["nonexistent.key"])
}

func TestProfile_Load_WithPreferences_ValidationError(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Persistent: true,
				validators: []validator{
					{
						desc: "test validator",
						fn: func(s Setting) error {
							if s.vv.String() == "invalid" {
								return fmt.Errorf("invalid value")
							}
							return nil
						},
					},
				},
			},
		},
	}

	prefs := NewPreferences(version.Version("v1.0.0"))
	prefs.Set("test.key", "invalid")

	_, err := schema.Profile("test", prefs)
	testutils.Error(t, err)
}

func TestProfile_Load_SettingError(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindInvalid, // Invalid kind will cause error
				Mutability: SettingMutable,
			},
		},
	}

	_, err := schema.Profile("test", nil)
	testutils.Error(t, err)
}

func TestProfile_Load_DefaultValidationError(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Value:      "default",
				validators: []validator{
					{
						desc: "test validator",
						fn: func(s Setting) error {
							return fmt.Errorf("default validation failed")
						},
					},
				},
			},
		},
	}

	_, err := schema.Profile("test", nil)
	testutils.Error(t, err)
}

func TestProfile_Set_ValidationError(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				validators: []validator{
					{
						desc: "test validator",
						fn: func(s Setting) error {
							if s.vv.String() == "invalid" {
								return fmt.Errorf("invalid value")
							}
							return nil
						},
					},
				},
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.Set("test.key", "invalid")
	testutils.Error(t, err)
}

func TestProfile_Set_SameValue(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Value:      "value",
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	// Set to same value
	err = profile.Set("test.key", "value")
	testutils.NoError(t, err)
	testutils.Assert(t, !profile.Changed(), "expected profile not to be changed when setting same value")
}

func TestProfile_ValidatePreference_NotFound(t *testing.T) {
	schema := Schema{
		version:  version.Version("v1.0.0"),
		settings: map[string]SettingSpec{},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.ValidatePreference("nonexistent.key", "value")
	testutils.Error(t, err)
}

func TestProfile_ValidatePreference_Immutable(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingImmutable,
				Persistent: true,
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.ValidatePreference("test.key", "value")
	testutils.Error(t, err)
}

func TestProfile_ValidatePreference_ValidationError(t *testing.T) {
	schema := Schema{
		version: version.Version("v1.0.0"),
		settings: map[string]SettingSpec{
			"test.key": {
				Key:        "test.key",
				Kind:       KindString,
				Mutability: SettingMutable,
				Persistent: true,
				validators: []validator{
					{
						desc: "test validator",
						fn: func(s Setting) error {
							if s.vv.String() == "invalid" {
								return fmt.Errorf("invalid value")
							}
							return nil
						},
					},
				},
			},
		},
	}

	profile, err := schema.Profile("test", nil)
	testutils.NoError(t, err)

	err = profile.ValidatePreference("test.key", "invalid")
	testutils.Error(t, err)
}
