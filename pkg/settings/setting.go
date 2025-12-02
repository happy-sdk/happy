// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"fmt"

	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/vars"
	"golang.org/x/text/language"
)

type Mutability uint8

const (
	// SettingImmutable can not be changed on runtime.
	SettingImmutable Mutability = 254
	// SettingOnce can be set only once on runtime.
	// When changed typically requires a reload of application.
	SettingOnce Mutability = 253
	// SettingMutable can be changed on runtime.
	SettingMutable Mutability = 252
)

func (m Mutability) String() string {
	switch m {
	case SettingImmutable:
		return "immutable"
	case SettingOnce:
		return "once"
	case SettingMutable:
		return "mutable"
	}
	return "unknown"
}

type SettingSpec struct {
	IsSet       bool
	Kind        Kind
	Key         string
	Default     string
	Mutability  Mutability
	Value       string
	Required    bool
	Persistent  bool
	Unmarchaler Unmarshaller
	Marchaler   Marshaller
	Settings    *Blueprint
	Secret      bool // true if field has secret:"true" tag
	validators  []validator
	i18n        map[language.Tag]string
	isI18n      bool   // true if field has i18n:"true" tag
	i18nKey     string // the i18n key to use for translation
}

func (s SettingSpec) Validate() error {
	if s.Mutability > SettingImmutable || s.Mutability < SettingMutable {
		return fmt.Errorf("%w: invalid mutability setting %d for %s", ErrSpec, s.Mutability, s.Key)
	}
	return nil
}

func (s SettingSpec) ValidateValue(value string) error {
	s.Value = value
	setting, err := s.setting()
	if err != nil {
		return err
	}
	for _, v := range s.validators {
		if err := v.fn(setting); err != nil {
			return fmt.Errorf("%s: %s", v.desc, err.Error())
		}
	}
	return nil
}

func (s SettingSpec) Setting(lang language.Tag) (Setting, error) {
	setting, err := s.setting()
	if err != nil {
		return setting, err
	}
	if s.i18n != nil {

		if desc, ok := s.i18n[lang]; ok {
			setting.desc = desc
		}
	}
	return setting, nil
}

func (s SettingSpec) setting() (Setting, error) {
	setting := Setting{
		key:        s.Key,
		kind:       s.Kind,
		isSet:      s.IsSet,
		mutability: s.Mutability,
		persistent: s.Persistent,
		desc:       s.i18n[language.English],
		isI18n:     s.isI18n,
		i18nKey:    s.i18nKey,
		isSecret:   s.Secret,
	}

	var err error
	setting.vv, err = vars.NewAs(s.Key, s.Value, true, vars.Kind(s.Kind))
	if err != nil {
		return Setting{}, fmt.Errorf("%w: key(%s)  %s", ErrProfile, s.Key, err.Error())
	}
	setting.dvv, err = vars.NewAs(s.Key, s.Default, true, vars.Kind(s.Kind))
	if err != nil {
		return Setting{}, fmt.Errorf("%w: key(%s)  %s", ErrProfile, s.Key, err.Error())
	}
	return setting, nil
}

type Setting struct {
	key        string
	vv         vars.Variable
	dvv        vars.Variable
	kind       Kind
	isSet      bool
	mutability Mutability
	persistent bool
	desc       string
	isI18n     bool   // true if this setting uses i18n
	i18nKey    string // the i18n key to use for translation
	isSecret   bool   // true if this setting is marked as secret via struct tag
}

func (s Setting) String() string {
	return s.vv.String()
}

// Display returns the translated value for display purposes.
// If the setting uses i18n and i18n is initialized, it returns the translated string.
// Otherwise, it returns the raw value (same as String()).
// For Persistent settings that are user-defined (isSet=true), i18n is not applied.
func (s Setting) Display() string {
	if !s.isI18n || s.i18nKey == "" {
		return s.vv.Display()
	}

	// Don't apply i18n to user-defined Persistent settings
	if s.persistent && s.isSet {
		return s.vv.Display()
	}

	// Try to resolve via i18n if available
	// This uses a build-tagged helper function
	translated := resolveI18n(s.i18nKey)
	// If translation was found (different from key), use it
	// Otherwise, if the value itself is the i18n key, return the translation attempt
	// (which might be the key if not found, but i18n.T() will try fallback)
	if translated != s.i18nKey {
		return translated
	}

	// If translation not found and value is the i18n key, return the translation attempt
	// (i18n.T() already tried fallback, so this is the best we can do)
	rawValue := s.vv.String()
	if rawValue == s.i18nKey {
		return translated
	}

	return s.vv.Display()
}

func (s Setting) Key() string {
	return s.key
}

func (s Setting) IsSet() bool {
	return s.isSet
}

func (s Setting) Value() vars.Variable {
	return s.vv
}

func (s Setting) Default() vars.Variable {
	return s.dvv
}

func (s Setting) Kind() Kind {
	return s.kind
}

func (s Setting) Persistent() bool {
	return s.persistent
}

func (s Setting) Mutability() Mutability {
	return s.mutability
}

func (s Setting) Description() string {
	return s.desc
}

// IsSecret reports whether this setting has been marked as secret via struct tag.
// Secrets are intended to be redacted by higher-level tooling (e.g. CLI config commands).
func (s Setting) IsSecret() bool {
	return s.isSecret
}

// IsI18n returns true if this setting uses i18n translation.
func (s Setting) IsI18n() bool {
	return s.isI18n
}

// I18nKey returns the i18n key for this setting, or empty string if not using i18n.
func (s Setting) I18nKey() string {
	return s.i18nKey
}

// resolveI18n resolves an i18n key to a translated string using the i18n package.
func resolveI18n(key string) string {
	return i18n.T(key)
}

func UnmarshalValue[T Value](data []byte, v T) error {
	return v.UnmarshalSetting(data)
}
