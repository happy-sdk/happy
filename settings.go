// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy

import (
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
	MainArgcMax          settings.Uint     `key:"app.main.argn.max" default:"0"`
	TimeLocation         settings.String   `key:"app.datetime.location,save" default:"Local" mutation:"once"`
	ThrottleTicks        settings.Duration `key:"app.throttle.ticks,save" default:"1s" mutation:"once"`
	ServiceLoaderTimeout settings.Duration `key:"app.service.loader.timeout" default:"30s" mutation:"once"`
	Instance             instance.Settings `group:"instance"`
	extended             map[string]settings.Settings
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

	return b, nil
}

// Extend adds a new settings group to the settings.
func (s *Settings) Extend(ss settings.Settings) {

}
