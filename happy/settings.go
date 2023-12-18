// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"fmt"

	"github.com/happy-sdk/happy/pkg/address"
	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/version"
	"golang.org/x/mod/semver"
	"golang.org/x/text/language"
)

type Settings struct {
	Name           settings.String `key:"app.name" default:"Happy Prototype" mutation:"once"`
	Slug           settings.String `key:"app.slug" default:"" mutation:"once"`
	Description    settings.String `key:"app.description" default:"Happy application prototype" mutation:"once"`
	Usage          settings.String `key:"app.usage" default:"" mutation:"once"`
	CopyrightBy    settings.String `key:"app.copyright.by" mutation:"once"`
	CopyrightSince settings.Uint   `key:"app.copyright.since" default:"0" mutation:"once"`
	License        settings.String `key:"app.license" mutation:"once"`
	TimeLocation   settings.String `key:"app.datetime.location" default:"Local" mutation:"once"`

	Logger logging.Settings `key:"logger"`
	I18n   i18n.Settings    `key:"i18n"`

	// MainArgcMax is number of arguments what root coomand accepts.
	// when arg does not match a subcommand.
	MainArgcMax   settings.Uint     `key:"app.main.argc.max" default:"0" mutation:"once"`
	ThrottleTicks settings.Duration `key:"app.throttle.ticks" default:"1s" mutation:"once"`
	embedSettings map[string]settings.Settings
	errs          []error
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	blueprint, err := settings.NewBlueprint(s)
	if err != nil {
		return nil, err
	}

	const appName = "app.name"
	blueprint.Describe(appName, language.English, "Application output verbosity")
	blueprint.AddValidator(appName, "application name validation", func(s settings.Setting) error {
		if s.Value().Empty() {
			return fmt.Errorf("%w: empty application name", settings.ErrSetting)
		}
		return nil
	})

	const appSlug = "app.slug"
	blueprint.Describe(appSlug, language.English, "Application slug")
	blueprint.AddValidator(appSlug, "", func(s settings.Setting) error {
		if s.Value().Empty() {
			return fmt.Errorf("%w: empty application slug", settings.ErrSetting)
		}
		if s.Value().Len() > 128 {
			return fmt.Errorf("%w: application slug is too long %d/128", settings.ErrSetting, s.Value().Len())
		}
		return nil
	})

	const appDescription = "app.description"
	blueprint.Describe(appDescription, language.English, "Set application slug")
	blueprint.AddValidator(appSlug, "", func(s settings.Setting) error {
		if s.Value().Empty() {
			return fmt.Errorf("%w: empty application slug", settings.ErrSetting)
		}
		if s.Value().Len() > 255 {
			return fmt.Errorf("%w: application slug is too long %d/255", settings.ErrSetting, s.Value().Len())
		}
		return nil
	})

	const appVersion = "app.version"
	ver := version.Current()
	blueprint.AddSpec(settings.SettingSpec{
		IsSet:      true,
		Kind:       settings.KindString,
		Mutability: settings.SettingImmutable,
		Key:        appVersion,
		Value:      ver.String(),
	})
	blueprint.Describe(appVersion, language.English, "Application version")
	blueprint.AddValidator(appVersion, "", func(s settings.Setting) error {
		if s.Value().Empty() {
			return fmt.Errorf("%w: empty application version", settings.ErrSetting)
		}
		if !semver.IsValid(s.String()) {
			return fmt.Errorf("%w %q, version must be valid semantic version", ErrInvalidVersion, s.String())
		}
		return nil
	})

	const appAddress = "app.address"
	var (
		addr *address.Address
	)

	if addr, err = address.Current(); err != nil {
		return nil, err
	}

	blueprint.AddSpec(settings.SettingSpec{
		IsSet:      true,
		Key:        appAddress,
		Kind:       settings.KindString,
		Mutability: settings.SettingImmutable,
		Value:      addr.String(),
	})
	blueprint.Describe(appAddress, language.English, "Application address")
	blueprint.AddValidator(appAddress, "", func(s settings.Setting) error {
		if s.Value().Empty() {
			return fmt.Errorf("%w: empty application address", settings.ErrSetting)
		}
		_, err := address.Parse(s.String())
		if err != nil {
			return fmt.Errorf(
				"%w: invalid application address (%q)",
				settings.ErrSetting, s.String())
		}
		return nil
	})
	const appModule = "app.module"
	blueprint.AddSpec(settings.SettingSpec{
		IsSet:      true,
		Key:        appModule,
		Kind:       settings.KindString,
		Mutability: settings.SettingImmutable,
		Value:      addr.Module,
	})

	const appDevel = "app.devel"
	blueprint.AddSpec(settings.SettingSpec{
		IsSet:      true,
		Key:        appDevel,
		Kind:       settings.KindBool,
		Mutability: settings.SettingImmutable,
		Value:      settings.Bool(version.IsDev(ver.String())).String(),
	})

	// Add user settings
	for group, extSettings := range s.embedSettings {
		if err := blueprint.Extend(group, extSettings); err != nil {
			return nil, err
		}
	}

	// configOpts := []OptionArg{

	// 	{
	// 		key:       "app.cron.on.service.start",
	// 		value:     false,
	// 		desc:      "Execute Cronjobs first time when service starts",
	// 		kind:      ReadOnlyOption | ConfigOption,
	// 		validator: noopvalidator,
	// 	},
	// 	{
	// 		key:       "app.fs.enabled",
	// 		value:     false,
	// 		desc:      "enable and load filesystem paths for application",
	// 		kind:      ReadOnlyOption | ConfigOption,
	// 		validator: noopvalidator,
	// 	},

	// 	{
	// 		key:       "app.cli.x",
	// 		value:     false,
	// 		desc:      "indicate that external commands should be printed.",
	// 		kind:      ConfigOption,
	// 		validator: noopvalidator,
	// 	},
	// 	{
	// 		key:   "app.throttle.ticks",
	// 		value: time.Duration(time.Second / 60),
	// 		desc:  "Interfal target for system and service ticks",
	// 		kind:  ReadOnlyOption | SettingsOption,
	// 		validator: func(key string, val vars.Value) error {
	// 			v, err := val.Int64()
	// 			if err != nil {
	// 				return err
	// 			}
	// 			if v < 1 {
	// 				return fmt.Errorf(
	// 					"%w: invalid throttle value %s(%d - %v), must be greater that 1",
	// 					ErrOptionValidation, val.Kind(), v, val.Any())
	// 			}
	// 			return nil
	// 		},
	// 	},

	// 	{
	// 		key:   "app.settings.persistent",
	// 		value: false,
	// 		desc:  "persist settings across restarts",
	// 		kind:  ReadOnlyOption | ConfigOption,
	// 		validator: func(key string, val vars.Value) error {
	// 			if val.Kind() != vars.KindBool {
	// 				return fmt.Errorf("%w: %s must be boolean got %s(%s)", ErrOptionValidation, key, val.Kind(), val.String())
	// 			}
	// 			return nil
	// 		},
	// 	},

	// }
	// return configOpts, nil
	return blueprint, nil
}

func (s *Settings) Extend(group string, ss settings.Settings) {
	if s.embedSettings == nil {
		s.embedSettings = make(map[string]settings.Settings)
	}
	if _, ok := s.embedSettings[group]; ok {
		s.errs = append(s.errs, fmt.Errorf("%w: settings group %q already exists", settings.ErrSettings, group))
		return
	}
	s.embedSettings[group] = ss
}
