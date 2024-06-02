// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package config

import "github.com/happy-sdk/happy/pkg/settings"

type Settings struct {
	// Disabled is used to disable the configuration system. When set to true, the configuration
	// system is disabled and single runtime profile is used. This is useful when the application does not
	// require user configuration and only uses the default settings.
	// All settings below are ignored and set to default values when Disabled is set to true.
	Disabled            settings.Bool        `default:"false" desc:"Enabled presistent user configuration"`
	AdditionalProfiles  settings.StringSlice `desc:"Additional profiles provided by default."`
	DefaultProfile      settings.String      `default:"default" mutation:"once" desc:"Default profile to use when no profile is specified."`
	AllowCustomProfiles settings.Bool        `desc:"Are creation of custom profiles allowed."`

	// EnableProfileDevel enables profile development mode. This mode allows different settings
	// for development and release versions for a named profile. When this flag is set to true,
	// a profile named "default" will also have a corresponding "default-devel" profile.
	//
	// In development mode (e.g., when using `go run` or a locally compiled binary), the "-devel"
	// profile is used. In release mode, the standard profile is used.
	//
	// If application Devel.AllowProd is set to true, this behavior can be overridden by using
	// the -x-prod flag, which is added by the devel package when the AllowProd option is enabled.
	// This allows to load the standard profile even when running in development mode e.g. go run.
	EnableProfileDevel settings.Bool `default:"false" desc:"Enable profile development mode."`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	if s.Disabled {
		return settings.New(Settings{
			Disabled: true,
		})
	}
	return settings.New(s)
}
