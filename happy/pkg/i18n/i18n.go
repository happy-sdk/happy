// Copyright 2023 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package i18n

import (
	"strings"

	"github.com/happy-sdk/happy/pkg/settings"
)

type Settings struct {
	// DefaultLocale is default locale for application
	Default Locale `key:"default" default:"en" mutation:"immutable"`
	// SupportedLocales is list of supported locales
	Supported Supported `key:"supported" default:"en" mutation:"immutable"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	blueprint, err := settings.NewBlueprint(s)
	if err != nil {
		return nil, err
	}
	return blueprint, nil
}

type Locale string

// MarshalSetting converts the Bool setting to a byte slice, representing "true" or "false".
func (l Locale) MarshalSetting() ([]byte, error) {
	return []byte(l), nil
}

// UnmarshalSetting updates the Bool setting from a byte slice, interpreting it as "true" or "false".
func (l *Locale) UnmarshalSetting(data []byte) error {
	*l = Locale(data)
	return nil
}

func (l Locale) SettingKind() settings.Kind {
	return settings.KindString
}

func (l Locale) String() string {
	return string(l)
}

type Supported []string

// MarshalSetting converts the Bool setting to a byte slice, representing "true" or "false".
func (s Supported) MarshalSetting() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s Supported) String() string {
	return strings.Join(s, ",")
}

func (s Supported) SettingKind() settings.Kind {
	return settings.KindString
}

// UnmarshalSetting updates the Bool setting from a byte slice, interpreting it as "true" or "false".
func (s *Supported) UnmarshalSetting(data []byte) error {
	*s = Supported(strings.Split(string(data), ","))
	return nil
}
