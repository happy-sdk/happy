// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"github.com/happy-sdk/happy/pkg/settings"
	"golang.org/x/text/language"
)

// Settings represents the configuration settings for internationalization.
// It provides a blueprint for configuring the default language and supported languages
// for Happy SDK Applications.
type Settings struct {
	Language       settings.String      `key:"language,save" default:"en" mutation:"mutable"`
	Supported      settings.StringSlice `key:"supported"`
	WithGlobalFlag settings.Bool        `key:"add_global_flag"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {

	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	b.Describe("language", language.English, "Global language")
	b.Describe("supported", language.English, "Available languages")
	b.AddValidator("language", "Validate language tag", func(s settings.Setting) error {
		lang, err := language.Parse(s.String())
		if err != nil {
			return err
		}
		if err := SetLanguage(lang); err != nil {
			return err
		}
		return nil
	})
	return b, nil
}
