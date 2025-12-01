// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package settings

import (
	"errors"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/version"
	"golang.org/x/text/language"
)

type SimpleSettings struct {
	Name  String `key:"name" default:"test"`
	Value Int    `key:"value" default:"42"`
}

func (s *SimpleSettings) Blueprint() (*Blueprint, error) {
	return New(s)
}

func TestBlueprint_AddSpec(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test",
		Kind:       KindString,
		Mutability: SettingMutable,
	}

	err := b.AddSpec(spec)
	testutils.NoError(t, err)
	testutils.Assert(t, len(b.specs) > 0, "expected spec to be added")
}

func TestBlueprint_AddSpec_Invalid(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test.key",
		Kind:       KindString,
		Mutability: SettingImmutable + 1, // Invalid
	}

	err := b.AddSpec(spec)
	testutils.Error(t, err)
}

func TestBlueprint_AddSpec_WithGroup(t *testing.T) {
	b := &Blueprint{
		specs:  make(map[string]SettingSpec),
		groups: make(map[string]*Blueprint),
	}

	spec := SettingSpec{
		Key:        "group.key",
		Kind:       KindString,
		Mutability: SettingMutable,
	}

	err := b.AddSpec(spec)
	testutils.NoError(t, err)
	testutils.Assert(t, len(b.groups) > 0, "expected group to be created")
}

func TestBlueprint_AddSpec_NestedSettings(t *testing.T) {
	b := &Blueprint{
		specs:  make(map[string]SettingSpec),
		groups: make(map[string]*Blueprint),
	}

	nested := &SimpleSettings{}
	nestedBP, err := New(nested)
	testutils.NoError(t, err)

	spec := SettingSpec{
		Key:        "nested",
		Kind:       KindSettings,
		Mutability: SettingImmutable, // Required for nested settings
		Settings:   nestedBP,
	}

	err = b.AddSpec(spec)
	testutils.NoError(t, err)
	testutils.Assert(t, len(b.groups) > 0, "expected nested settings group")
}

func TestBlueprint_AddSpec_DuplicateGroup(t *testing.T) {
	b := &Blueprint{
		specs:  make(map[string]SettingSpec),
		groups: make(map[string]*Blueprint),
	}

	nested := &SimpleSettings{}
	nestedBP, err := New(nested)
	testutils.NoError(t, err)

	spec1 := SettingSpec{
		Key:        "nested",
		Kind:       KindSettings,
		Mutability: SettingImmutable, // Required for nested settings
		Settings:   nestedBP,
	}

	err = b.AddSpec(spec1)
	testutils.NoError(t, err)

	// Try to add duplicate
	err = b.AddSpec(spec1)
	testutils.Error(t, err)
}

func TestBlueprint_Migrate(t *testing.T) {
	b := &Blueprint{
		migrations: make(map[string]string),
	}

	err := b.Migrate("old.key", "new.key")
	testutils.NoError(t, err)
	testutils.Assert(t, len(b.migrations) > 0, "expected migration to be added")
}

func TestBlueprint_Migrate_Duplicate(t *testing.T) {
	b := &Blueprint{
		migrations: make(map[string]string),
	}

	err := b.Migrate("old.key", "new.key")
	testutils.NoError(t, err)

	// Try to migrate same key again
	err = b.Migrate("old.key", "another.key")
	testutils.Error(t, err)
}

func TestBlueprint_GetSpec(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test",
		Kind:       KindString,
		Mutability: SettingMutable,
	}

	b.specs["test"] = spec

	got, err := b.GetSpec("test")
	testutils.NoError(t, err)
	testutils.Equal(t, spec.Key, got.Key)
}

func TestBlueprint_GetSpec_NotFound(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	_, err := b.GetSpec("nonexistent.key")
	testutils.Error(t, err)
}

func TestBlueprint_GetSpec_Group(t *testing.T) {
	b := &Blueprint{
		specs:  make(map[string]SettingSpec),
		groups: make(map[string]*Blueprint),
	}

	group := &Blueprint{
		specs: make(map[string]SettingSpec),
	}
	group.specs["key"] = SettingSpec{
		Key:        "key",
		Kind:       KindString,
		Mutability: SettingMutable,
	}
	b.groups["group"] = group

	got, err := b.GetSpec("group.key")
	testutils.NoError(t, err)
	testutils.Equal(t, "key", got.Key)
}

func TestBlueprint_AddValidator(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test.key",
		Kind:       KindString,
		Mutability: SettingMutable,
	}
	b.specs["test.key"] = spec

	b.AddValidator("test.key", "test validator", func(s Setting) error {
		return nil
	})

	updatedSpec, ok := b.specs["test.key"]
	testutils.Assert(t, ok, "expected spec to exist")
	testutils.Assert(t, len(updatedSpec.validators) > 0, "expected validator to be added")
}

func TestBlueprint_AddValidator_NotFound(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	b.AddValidator("nonexistent.key", "test", func(s Setting) error {
		return nil
	})

	testutils.Assert(t, len(b.errs) > 0, "expected error to be recorded")
}

func TestBlueprint_Describe(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test.key",
		Kind:       KindString,
		Mutability: SettingMutable,
	}
	b.specs["test.key"] = spec

	b.Describe("test.key", language.French, "Description en français")

	updatedSpec, ok := b.specs["test.key"]
	testutils.Assert(t, ok, "expected spec to exist")
	testutils.Assert(t, updatedSpec.i18n != nil, "expected i18n map to be created")
	testutils.ContainsKey(t, updatedSpec.i18n, language.French)
}

func TestBlueprint_Describe_Duplicate(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test.key",
		Kind:       KindString,
		Mutability: SettingMutable,
		i18n: map[language.Tag]string{
			language.French: "Existing description",
		},
	}
	b.specs["test.key"] = spec

	b.Describe("test.key", language.French, "New description")

	testutils.Assert(t, len(b.errs) > 0, "expected error for duplicate description")
}

func TestBlueprint_PackageSettingsStructName(t *testing.T) {
	b := &Blueprint{
		pkgSettingsStructName: "TestSettings",
	}
	testutils.Equal(t, "TestSettings", b.PackageSettingsStructName())
}

func TestBlueprint_SetDefault(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test",
		Kind:       KindString,
		Mutability: SettingMutable,
		Value:      "",
	}
	b.specs["test"] = spec

	value := String("new default")
	err := b.SetDefault("test", &value)
	testutils.NoError(t, err)

	updatedSpec := b.specs["test"]
	testutils.Equal(t, "new default", updatedSpec.Default)
	testutils.Equal(t, "new default", updatedSpec.Value)
}

func TestBlueprint_SetDefault_WithExistingValue(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test",
		Kind:       KindString,
		Mutability: SettingMutable,
		Value:      "existing value",
	}
	b.specs["test"] = spec

	value := String("new default")
	err := b.SetDefault("test", &value)
	testutils.NoError(t, err)

	updatedSpec := b.specs["test"]
	testutils.Equal(t, "new default", updatedSpec.Default)
	testutils.Equal(t, "existing value", updatedSpec.Value) // Should not change
}

func TestBlueprint_SetDefault_WithZeroValue(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test",
		Kind:       KindString,
		Mutability: SettingMutable,
		Value:      "0",
	}
	b.specs["test"] = spec

	value := String("new default")
	err := b.SetDefault("test", &value)
	testutils.NoError(t, err)

	updatedSpec := b.specs["test"]
	testutils.Equal(t, "new default", updatedSpec.Value)
}

func TestBlueprint_SetDefault_WithGroup(t *testing.T) {
	b := &Blueprint{
		specs:  make(map[string]SettingSpec),
		groups: make(map[string]*Blueprint),
	}

	group := &Blueprint{
		specs: make(map[string]SettingSpec),
	}
	group.specs["key"] = SettingSpec{
		Key:        "key",
		Kind:       KindString,
		Mutability: SettingMutable,
	}
	b.groups["group"] = group

	value := String("group default")
	err := b.SetDefault("group.key", &value)
	testutils.NoError(t, err)

	updatedSpec := group.specs["key"]
	testutils.Equal(t, "group default", updatedSpec.Default)
}

func TestBlueprint_SetDefault_GroupNotFound(t *testing.T) {
	b := &Blueprint{
		specs:  make(map[string]SettingSpec),
		groups: make(map[string]*Blueprint),
	}

	value := String("default")
	err := b.SetDefault("nonexistent.key", &value)
	testutils.Error(t, err)
}

func TestBlueprint_SetDefault_KeyNotFound(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	value := String("default")
	err := b.SetDefault("nonexistent", &value)
	testutils.Error(t, err)
}

func TestBlueprint_SetDefaultFromString(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test",
		Kind:       KindString,
		Mutability: SettingMutable,
		Value:      "",
	}
	b.specs["test"] = spec

	err := b.SetDefaultFromString("test", "string default")
	testutils.NoError(t, err)

	updatedSpec := b.specs["test"]
	testutils.Equal(t, "string default", updatedSpec.Default)
	testutils.Equal(t, "string default", updatedSpec.Value)
}

func TestBlueprint_SetDefaultFromString_WithExistingValue(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test",
		Kind:       KindString,
		Mutability: SettingMutable,
		Value:      "existing",
	}
	b.specs["test"] = spec

	err := b.SetDefaultFromString("test", "new default")
	testutils.NoError(t, err)

	updatedSpec := b.specs["test"]
	testutils.Equal(t, "new default", updatedSpec.Default)
	testutils.Equal(t, "existing", updatedSpec.Value)
}

func TestBlueprint_SetDefaultFromString_WithZeroValue(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	spec := SettingSpec{
		Key:        "test",
		Kind:       KindString,
		Mutability: SettingMutable,
		Value:      "0",
	}
	b.specs["test"] = spec

	err := b.SetDefaultFromString("test", "new default")
	testutils.NoError(t, err)

	updatedSpec := b.specs["test"]
	testutils.Equal(t, "new default", updatedSpec.Value)
}

func TestBlueprint_SetDefaultFromString_WithGroup(t *testing.T) {
	b := &Blueprint{
		specs:  make(map[string]SettingSpec),
		groups: make(map[string]*Blueprint),
	}

	group := &Blueprint{
		specs: make(map[string]SettingSpec),
	}
	group.specs["key"] = SettingSpec{
		Key:        "key",
		Kind:       KindString,
		Mutability: SettingMutable,
	}
	b.groups["group"] = group

	err := b.SetDefaultFromString("group.key", "group default")
	testutils.NoError(t, err)

	updatedSpec := group.specs["key"]
	testutils.Equal(t, "group default", updatedSpec.Default)
}

func TestBlueprint_SetDefaultFromString_GroupNotFound(t *testing.T) {
	b := &Blueprint{
		specs:  make(map[string]SettingSpec),
		groups: make(map[string]*Blueprint),
	}

	err := b.SetDefaultFromString("nonexistent.key", "default")
	testutils.Error(t, err)
}

func TestBlueprint_SetDefaultFromString_KeyNotFound(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
	}

	err := b.SetDefaultFromString("nonexistent", "default")
	testutils.Error(t, err)
}

func TestBlueprint_Schema(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
		mode:  ModeTesting,
	}

	spec := SettingSpec{
		Key:        "test",
		Kind:       KindString,
		Mutability: SettingMutable,
	}
	b.specs["test"] = spec

	schema, err := b.Schema("test.module", version.Version("v1.0.0"))
	testutils.NoError(t, err)
	testutils.Equal(t, "test.module", schema.module)
	testutils.Equal(t, version.Version("v1.0.0"), schema.version)
	testutils.Assert(t, len(schema.settings) > 0, "expected settings in schema")
}

func TestBlueprint_Schema_WithErrors(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
		errs:  []error{errors.New("test error")},
	}

	_, err := b.Schema("test.module", version.Version("v1.0.0"))
	testutils.Error(t, err)
}

func TestBlueprint_Schema_WithNestedSettings(t *testing.T) {
	b := &Blueprint{
		specs: make(map[string]SettingSpec),
		mode:  ModeTesting,
	}

	nested := &SimpleSettings{}
	nestedBP, err := New(nested)
	testutils.NoError(t, err)

	spec := SettingSpec{
		Key:        "nested",
		Kind:       KindSettings,
		Mutability: SettingImmutable,
		Settings:   nestedBP,
	}
	b.specs["nested"] = spec

	schema, err := b.Schema("test.module", version.Version("v1.0.0"))
	testutils.NoError(t, err)
	// Should have nested settings expanded
	testutils.Assert(t, len(schema.settings) > 1, "expected nested settings to be expanded")
}

func TestBlueprint_Schema_WithGroups(t *testing.T) {
	b := &Blueprint{
		specs:  make(map[string]SettingSpec),
		groups: make(map[string]*Blueprint),
		mode:   ModeTesting,
	}

	group := &Blueprint{
		specs: make(map[string]SettingSpec),
		mode:  ModeTesting,
	}
	group.specs["key"] = SettingSpec{
		Key:        "key",
		Kind:       KindString,
		Mutability: SettingMutable,
	}
	b.groups["group"] = group

	schema, err := b.Schema("test.module", version.Version("v1.0.0"))
	testutils.NoError(t, err)
	// Should have group settings with prefix
	testutils.ContainsKey(t, schema.settings, "group.key")
}

func TestBlueprint_Extend_Nil(t *testing.T) {
	b := &Blueprint{
		groups: make(map[string]*Blueprint),
	}

	err := b.Extend("group", nil)
	testutils.Error(t, err)
}

func TestBlueprint_Extend_PanicRecovery(t *testing.T) {
	b := &Blueprint{
		groups: make(map[string]*Blueprint),
	}

	// Create a Settings that panics on Blueprint()
	panicSettings := &PanicSettings{}

	err := b.Extend("group", panicSettings)
	// Should recover from panic and try New() instead
	testutils.NoError(t, err)
	testutils.Assert(t, len(b.groups) > 0, "expected group to be added after panic recovery")
}

// PanicSettings panics when Blueprint() is called
type PanicSettings struct {
	Value String `key:"value" default:"test"`
}

func (p *PanicSettings) Blueprint() (*Blueprint, error) {
	panic("intentional panic for testing")
}

func TestBlueprint_Extend_AlreadyExists(t *testing.T) {
	b := &Blueprint{
		groups: make(map[string]*Blueprint),
	}

	existing := &Blueprint{}
	b.groups["group"] = existing

	settings := &SimpleSettings{}
	err := b.Extend("group", settings)
	testutils.Error(t, err)
}

func TestBlueprint_Extend_BlueprintReturnsNil(t *testing.T) {
	b := &Blueprint{
		groups: make(map[string]*Blueprint),
	}

	nilSettings := &NilBlueprintSettings{}

	err := b.Extend("group", nilSettings)
	testutils.Error(t, err)
}

// NilBlueprintSettings returns nil from Blueprint()
type NilBlueprintSettings struct{}

func (n *NilBlueprintSettings) Blueprint() (*Blueprint, error) {
	return nil, nil
}
