// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package settings

import (
	"fmt"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/i18n"
	"golang.org/x/text/language"
)

func TestMutability_String(t *testing.T) {
	tests := []struct {
		name string
		m    Mutability
		want string
	}{
		{"Immutable", SettingImmutable, "immutable"},
		{"Once", SettingOnce, "once"},
		{"Mutable", SettingMutable, "mutable"},
		{"Unknown", Mutability(0), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m.String()
			testutils.Equal(t, tt.want, got)
		})
	}
}

func TestSettingSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    SettingSpec
		wantErr bool
	}{
		{
			name: "Valid Immutable",
			spec: SettingSpec{
				Key:        "test.key",
				Mutability: SettingImmutable,
			},
			wantErr: false,
		},
		{
			name: "Valid Once",
			spec: SettingSpec{
				Key:        "test.key",
				Mutability: SettingOnce,
			},
			wantErr: false,
		},
		{
			name: "Valid Mutable",
			spec: SettingSpec{
				Key:        "test.key",
				Mutability: SettingMutable,
			},
			wantErr: false,
		},
		{
			name: "Invalid - too high",
			spec: SettingSpec{
				Key:        "test.key",
				Mutability: SettingImmutable + 1,
			},
			wantErr: true,
		},
		{
			name: "Invalid - too low",
			spec: SettingSpec{
				Key:        "test.key",
				Mutability: SettingMutable - 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				testutils.Error(t, err)
			} else {
				testutils.NoError(t, err)
			}
		})
	}
}

func TestString_MarshalSetting(t *testing.T) {
	s := String("test value")
	data, err := s.MarshalSetting()
	testutils.NoError(t, err)
	testutils.Equal(t, "test value", string(data))
}

func TestString_UnmarshalSetting(t *testing.T) {
	var s String
	err := s.UnmarshalSetting([]byte("test value"))
	testutils.NoError(t, err)
	testutils.Equal(t, String("test value"), s)
}

func TestString_String(t *testing.T) {
	s := String("test")
	testutils.Equal(t, "test", s.String())
}

func TestString_SettingKind(t *testing.T) {
	var s String
	testutils.Equal(t, KindString, s.SettingKind())
}

func TestBool_MarshalSetting(t *testing.T) {
	b := Bool(true)
	data, err := b.MarshalSetting()
	testutils.NoError(t, err)
	testutils.Equal(t, "true", string(data))
}

func TestBool_UnmarshalSetting(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    Bool
		wantErr bool
	}{
		{"true", []byte("true"), Bool(true), false},
		{"false", []byte("false"), Bool(false), false},
		{"empty", []byte(""), Bool(false), false},
		{"invalid", []byte("invalid"), Bool(false), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b Bool
			err := b.UnmarshalSetting(tt.data)
			if tt.wantErr {
				testutils.Error(t, err)
			} else {
				testutils.NoError(t, err)
				testutils.Equal(t, tt.want, b)
			}
		})
	}
}

func TestBool_String(t *testing.T) {
	testutils.Equal(t, "true", Bool(true).String())
	testutils.Equal(t, "false", Bool(false).String())
}

func TestBool_SettingKind(t *testing.T) {
	var b Bool
	testutils.Equal(t, KindBool, b.SettingKind())
}

func TestInt_MarshalSetting(t *testing.T) {
	i := Int(42)
	data, err := i.MarshalSetting()
	testutils.NoError(t, err)
	testutils.Equal(t, "42", string(data))
}

func TestInt_UnmarshalSetting(t *testing.T) {
	var i Int
	err := i.UnmarshalSetting([]byte("42"))
	testutils.NoError(t, err)
	testutils.Equal(t, Int(42), i)
}

func TestInt_String(t *testing.T) {
	testutils.Equal(t, "42", Int(42).String())
}

func TestInt_SettingKind(t *testing.T) {
	var i Int
	testutils.Equal(t, KindInt, i.SettingKind())
}

func TestUint_MarshalSetting(t *testing.T) {
	u := Uint(42)
	data, err := u.MarshalSetting()
	testutils.NoError(t, err)
	testutils.Equal(t, "42", string(data))
}

func TestUint_UnmarshalSetting(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    Uint
		wantErr bool
	}{
		{"valid", []byte("42"), Uint(42), false},
		{"empty", []byte(""), Uint(0), false},
		{"invalid", []byte("invalid"), Uint(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u Uint
			err := u.UnmarshalSetting(tt.data)
			if tt.wantErr {
				testutils.Error(t, err)
			} else {
				testutils.NoError(t, err)
				testutils.Equal(t, tt.want, u)
			}
		})
	}
}

func TestUint_String(t *testing.T) {
	testutils.Equal(t, "42", Uint(42).String())
}

func TestUint_SettingKind(t *testing.T) {
	var u Uint
	testutils.Equal(t, KindUint, u.SettingKind())
}

func TestDuration_MarshalSetting(t *testing.T) {
	d := Duration(3600000000000) // 1 hour
	data, err := d.MarshalSetting()
	testutils.NoError(t, err)
	testutils.Equal(t, "1h0m0s", string(data))
}

func TestDuration_UnmarshalSetting(t *testing.T) {
	var d Duration
	err := d.UnmarshalSetting([]byte("1h"))
	testutils.NoError(t, err)
	testutils.Equal(t, Duration(3600000000000), d)
}

func TestDuration_String(t *testing.T) {
	d := Duration(3600000000000)
	testutils.Equal(t, "1h0m0s", d.String())
}

func TestDuration_SettingKind(t *testing.T) {
	var d Duration
	testutils.Equal(t, KindDuration, d.SettingKind())
}

func TestStringSlice_MarshalSetting(t *testing.T) {
	ss := StringSlice{"a", "b", "c"}
	data, err := ss.MarshalSetting()
	testutils.NoError(t, err)
	testutils.Equal(t, "a\x1fb\x1fc", string(data))
}

func TestStringSlice_UnmarshalSetting(t *testing.T) {
	var ss StringSlice
	err := ss.UnmarshalSetting([]byte("a\x1fb\x1fc"))
	testutils.NoError(t, err)
	want := StringSlice{"a", "b", "c"}
	testutils.Equal(t, len(want), len(ss))
	for i := range want {
		testutils.Equal(t, want[i], ss[i])
	}
}

func TestStringSlice_String(t *testing.T) {
	ss := StringSlice{"a", "b", "c"}
	testutils.Equal(t, "a\x1fb\x1fc", ss.String())
}

func TestStringSlice_SettingKind(t *testing.T) {
	var ss StringSlice
	testutils.Equal(t, KindStringSlice, ss.SettingKind())
}

func TestSetting_String(t *testing.T) {
	// This test requires a properly initialized Setting
	// We'll test this more thoroughly in integration tests
}

func TestSetting_Key(t *testing.T) {
	spec := SettingSpec{
		Key:   "test.key",
		Value: "test value",
		Kind:  KindString,
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	testutils.Equal(t, "test.key", setting.Key())
}

func TestSetting_IsSet(t *testing.T) {
	spec := SettingSpec{
		Key:   "test.key",
		Value: "test value",
		Kind:  KindString,
		IsSet: true,
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	testutils.Assert(t, setting.IsSet(), "expected setting to be set")
}

func TestSetting_Kind(t *testing.T) {
	spec := SettingSpec{
		Key:  "test.key",
		Kind: KindBool,
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	testutils.Equal(t, KindBool, setting.Kind())
}

func TestSetting_Mutability(t *testing.T) {
	spec := SettingSpec{
		Key:        "test.key",
		Mutability: SettingMutable,
		Kind:       KindString,
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	testutils.Equal(t, SettingMutable, setting.Mutability())
}

func TestSetting_Persistent(t *testing.T) {
	spec := SettingSpec{
		Key:        "test.key",
		Persistent: true,
		Kind:       KindString,
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	testutils.Assert(t, setting.Persistent(), "expected setting to be persistent")
}

func TestSetting_Description(t *testing.T) {
	spec := SettingSpec{
		Key:  "test.key",
		Kind: KindString,
		i18n: map[language.Tag]string{
			language.English: "Test description",
		},
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	testutils.Equal(t, "Test description", setting.Description())
}

func TestSetting_IsI18n(t *testing.T) {
	spec := SettingSpec{
		Key:    "test.key",
		Kind:   KindString,
		isI18n: true,
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	testutils.Assert(t, setting.IsI18n(), "expected setting to use i18n")
}

func TestSetting_I18nKey(t *testing.T) {
	spec := SettingSpec{
		Key:     "test.key",
		Kind:    KindString,
		isI18n:  true,
		i18nKey: "i18n.test.key",
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	testutils.Equal(t, "i18n.test.key", setting.I18nKey())
}

func TestUnmarshalValue(t *testing.T) {
	var s String
	err := UnmarshalValue([]byte("test"), &s)
	testutils.NoError(t, err)
	testutils.Equal(t, String("test"), s)
}

func TestSetting_Display_NoI18n(t *testing.T) {
	spec := SettingSpec{
		Key:    "test.key",
		Value:  "test value",
		Kind:   KindString,
		isI18n: false,
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	testutils.Equal(t, "test value", setting.Display())
}

func TestSetting_Display_WithI18nKey(t *testing.T) {
	// Initialize i18n to avoid panic
	i18n.Initialize(language.English)

	spec := SettingSpec{
		Key:     "test.key",
		Value:   "i18n.key",
		Kind:    KindString,
		isI18n:  true,
		i18nKey: "i18n.key",
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	// Display will try to resolve via i18n
	result := setting.Display()
	testutils.Assert(t, result != "", "expected non-empty display value")
}

func TestSetting_Display_PersistentUserDefined(t *testing.T) {
	spec := SettingSpec{
		Key:        "test.key",
		Value:      "user value",
		Kind:       KindString,
		isI18n:     true,
		i18nKey:    "i18n.key",
		Persistent: true,
		IsSet:      true, // User-defined
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	// Should not apply i18n to user-defined persistent settings
	result := setting.Display()
	testutils.Equal(t, "user value", result)
}

func TestSetting_Display_ValueEqualsI18nKey(t *testing.T) {
	// Initialize i18n to avoid panic
	i18n.Initialize(language.English)

	spec := SettingSpec{
		Key:     "test.key",
		Value:   "i18n.key",
		Kind:    KindString,
		isI18n:  true,
		i18nKey: "i18n.key",
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)
	result := setting.Display()
	testutils.Assert(t, result != "", "expected non-empty display value")
}

func TestSetting_Default(t *testing.T) {
	spec := SettingSpec{
		Key:     "test.key",
		Value:   "value",
		Default: "default value",
		Kind:    KindString,
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)

	defaultVar := setting.Default()
	testutils.NotNil(t, defaultVar)
	testutils.Equal(t, "default value", defaultVar.String())
}

func TestSetting_Value(t *testing.T) {
	spec := SettingSpec{
		Key:   "test.key",
		Value: "test value",
		Kind:  KindString,
	}
	setting, err := spec.Setting(language.English)
	testutils.NoError(t, err)

	valueVar := setting.Value()
	testutils.NotNil(t, valueVar)
	testutils.Equal(t, "test value", valueVar.String())
}

func TestSettingSpec_ValidateValue(t *testing.T) {
	spec := SettingSpec{
		Key:        "test.key",
		Kind:       KindString,
		Mutability: SettingMutable,
	}

	err := spec.ValidateValue("valid value")
	testutils.NoError(t, err)
}

func TestSettingSpec_ValidateValue_WithValidator(t *testing.T) {
	spec := SettingSpec{
		Key:        "test.key",
		Kind:       KindString,
		Mutability: SettingMutable,
		validators: []validator{
			{
				desc: "test validator",
				fn: func(s Setting) error {
					if s.vv.String() == "invalid" {
						return fmt.Errorf("value cannot be 'invalid'")
					}
					return nil
				},
			},
		},
	}

	err := spec.ValidateValue("valid value")
	testutils.NoError(t, err)

	err = spec.ValidateValue("invalid")
	testutils.Error(t, err)
}

func TestSettingSpec_ValidateValue_InvalidKind(t *testing.T) {
	spec := SettingSpec{
		Key:        "test.key",
		Kind:       KindInvalid,
		Mutability: SettingMutable,
	}

	err := spec.ValidateValue("value")
	testutils.Error(t, err)
}

func TestResolveI18n(t *testing.T) {
	// Initialize i18n to avoid panic
	i18n.Initialize(language.English)

	// This is tested indirectly through Display() tests
	// The actual i18n resolution depends on i18n package initialization
	result := resolveI18n("test.key")
	testutils.Assert(t, result != "", "expected non-empty result")
}
