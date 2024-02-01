// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy

import (
	"errors"
	"fmt"

	"github.com/happy-sdk/happy/sdk/instance"
	"github.com/happy-sdk/happy/sdk/settings"
	"golang.org/x/text/language"
)

type Settings struct {
	Name                 settings.String   `key:"app.name" default:"Happy Prototype"`
	Slug                 settings.String   `key:"app.slug" default:""`
	Description          settings.String   `key:"app.description" default:""`
	CopyrightBy          settings.String   `key:"app.copyright.by"`
	CopyrightSince       settings.Uint     `key:"app.copyright.since" default:"0"`
	License              settings.String   `key:"app.license"`
	MainArgcMax          settings.Uint     `key:"app.main.argn_max" default:"0"`
	TimeLocation         settings.String   `key:"app.datetime.location,save" default:"Local" mutation:"once"`
	EngineThrottleTicks  settings.Duration `key:"app.engine.throttle_ticks,save" default:"1s" mutation:"once"`
	ServiceLoaderTimeout settings.Duration `key:"app.service_loader.timeout" default:"30s" mutation:"once"`
	Instance             instance.Settings `key:"app.instance"`
	global               []settings.Settings
	migrations           map[string]string
	errs                 []error
}

// Blueprint returns a blueprint for the settings.
func (s Settings) Blueprint() (*settings.Blueprint, error) {

	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}
	const appSlug = "app.slug"
	b.Describe(appSlug, language.English, "Application slug")
	b.AddValidator(appSlug, "", func(s settings.Setting) error {
		if s.Value().Empty() {
			return fmt.Errorf("%w: empty application slug", settings.ErrSetting)
		}
		if s.Value().Len() > 128 {
			return fmt.Errorf("%w: application slug is too long %d/128", settings.ErrSetting, s.Value().Len())
		}
		return nil
	})

	s.Migrate("app.throttle.ticks", "app.engine.throttle_ticks")
	if s.migrations != nil {
		for from, to := range s.migrations {
			if err := b.Migrate(from, to); err != nil {
				return nil, err
			}
		}
	}
	return b, errors.Join(s.errs...)
}

// Migrate allows auto migrate old settigns from keyfrom to keyto
// when applying preferences from deprecated keyfrom.
func (s *Settings) Migrate(keyfrom, keyto string) {
	if s.migrations == nil {
		s.migrations = make(map[string]string)
	}
	if to, ok := s.migrations[keyfrom]; ok {
		s.errs = append(s.errs, fmt.Errorf("%w: adding migration from %s to %s. from %s to %s already exists", settings.ErrSetting, keyfrom, keyto, keyfrom, to))
	}
	s.migrations[keyfrom] = keyto
}

// Extend adds a new settings group to the settings.
func (s *Settings) Extend(ss settings.Settings) {
	s.global = append(s.global, ss)
}
