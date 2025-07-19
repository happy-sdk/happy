// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy

import (
	"errors"
	"fmt"
	"sync"

	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/slug"
	"golang.org/x/text/language"
)

type Settings struct {
	mu sync.Mutex

	// Appication info
	Name           settings.String `key:"app.name" default:"Happy Prototype" desc:"Application name"`
	Slug           settings.String `key:"app.slug" default:"" desc:"Application slug"`
	Identifier     settings.String `key:"app.identifier" desc:"Application identifier"`
	Description    settings.String `key:"app.description" default:"This application is built using the Happy-SDK to provide enhanced functionality and features." desc:"Application description"`
	CopyrightBy    settings.String `key:"app.copyright_by" default:"Anonymous" desc:"Application author"`
	CopyrightSince settings.Uint   `key:"app.copyright_since" default:"0" desc:"Application copyright since"`
	License        settings.String `key:"app.license" default:"NOASSERTION" desc:"Application license"`

	// Application settings
	Engine   EngineSettings   `key:"app.engine"`
	CLI      CliSettings      `key:"app.cli"`
	Profiles ProfileSettings  `key:"app.profiles"`
	DateTime DateTimeSettings `key:"app.datetime"`
	Instance InstanceSettings `key:"app.instance"`
	Logging  LoggingSettings  `key:"app.logging"`
	Services ServicesSettings `key:"app.services"`
	Stats    StatsSettings    `key:"app.stats"`
	Devel    DevelSettings    `key:"app.devel"`
	I18n     I18nSettings     `key:"app.i18n"`

	global     []settings.Settings
	migrations map[string]string
	errs       []error
}

func (s *Settings) Blueprint() (*settings.Blueprint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}
	const appSlug = "app.slug"
	b.AddValidator(appSlug, "", func(s settings.Setting) error {
		if s.Value().Empty() {
			return fmt.Errorf("%w: empty application slug", settings.ErrSetting)
		}
		if s.Value().Len() > 128 {
			return fmt.Errorf("%w: application slug is too long %d/128", settings.ErrSetting, s.Value().Len())
		}
		if !slug.IsValid(s.Value().String()) {
			return fmt.Errorf("%w: invalid application slug", settings.ErrSetting)
		}
		return nil
	})

	for _, ext := range s.global {
		if err := b.Extend("", ext); err != nil {
			return nil, err
		}
	}

	for keyfrom, keyto := range s.migrations {
		if err := b.Migrate(keyfrom, keyto); err != nil {
			s.errs = append(s.errs, err)
		}
	}
	return b, errors.Join(s.errs...)
}

// Migrate allows auto migrate old settings from keyfrom to keyto
// when applying preferences from deprecated keyfrom.
func (s *Settings) Migrate(keyfrom, keyto string) *Settings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.migrations == nil {
		s.migrations = make(map[string]string)
	}
	if to, ok := s.migrations[keyfrom]; ok {
		s.errs = append(s.errs, fmt.Errorf("%w: adding migration from %s to %s. from %s to %s already exists", settings.ErrSetting, keyfrom, keyto, keyfrom, to))
	}
	s.migrations[keyfrom] = keyto
	return s
}

// extend adds a new settings group to the application settings.
func (s *Settings) extend(ss settings.Settings) {
	s.global = append(s.global, ss)
}

func (s *Settings) GetFallbackLanguage() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.I18n.Language.String()
}

type LoggingSettings struct {
	Level           logging.Level   `key:"level,config" default:"ok" mutation:"mutable" desc:"logging level"`
	WithSource      settings.Bool   `key:"with_source,config" default:"false" mutation:"once" desc:"Show source location in log messages"`
	TimestampFormat settings.String `key:"timeestamp_format,config" default:"15:04:05.000" mutation:"once" desc:"Timestamp format for log messages"`
	NoTimestamp     settings.Bool   `key:"no_timestamp,config" default:"false" mutation:"once" desc:"Do not show timestamps"`
	NoSlogDefault   settings.Bool   `key:"no_slog_default" default:"false" mutation:"once" desc:"Do not set the default slog logger"`
}

func (s LoggingSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

type ProfileSettings struct {
	// Disabled is used to disable the configuration system. When set to true, the configuration
	// system is disabled and single runtime profile is used. This is useful when the application does not
	// require user configuration and only uses the default settings.
	// All settings below are ignored and set to default values when Disabled is set to true.
	Disabled    settings.Bool        `default:"false" desc:"Disabled presistent user configuration"`
	Additional  settings.StringSlice `desc:"Additional profiles provided by default."`
	Default     settings.String      `default:"default" mutation:"once" desc:"Default profile to use when no profile is specified."`
	AllowCustom settings.Bool        `default:"false" desc:"Are creation of custom profiles allowed."`

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
	EnableDevel settings.Bool `default:"false" desc:"Enable profile development mode."`
}

func (s ProfileSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

// Settings represents the configuration settings for internationalization.
// It provides a blueprint for configuring the default language and supported languages
// for Happy SDK Applications.
type I18nSettings struct {
	Language  settings.String      `key:"language,save" default:"en" mutation:"mutable"`
	Supported settings.StringSlice `key:"supported"`
}

func (s I18nSettings) Blueprint() (*settings.Blueprint, error) {

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
		if err := i18n.SetLanguage(lang); err != nil && !errors.Is(err, i18n.ErrDisabled) {
			return err
		}
		return nil
	})
	return b, nil
}

type CliSettings struct {
	settings.Settings
	MainMinArgs          settings.Uint `default:"0" desc:"Minimum number of arguments for a application main"`
	MainMaxArgs          settings.Uint `default:"0" desc:"Maximum number of arguments for a application main"`
	WithConfigCmd        settings.Bool `default:"false" desc:"Add the config command in the CLI"`
	WithGlobalFlags      settings.Bool `default:"false" desc:"Add the default global flags automatically in the CLI"`
	HideDisabledCommands settings.Bool `default:"false" desc:"Hide disabled commands"`
}

func (s CliSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

type EngineSettings struct {
	ThrottleTicks settings.Duration `key:"throttle_ticks,save" default:"1s" mutation:"once" desc:"Throttle engine ticks duration"`
}

func (s EngineSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

type DateTimeSettings struct {
	Location settings.String `key:"location,save" default:"Local" mutation:"once" desc:"The location to use for time operations."`
}

func (s DateTimeSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

type InstanceSettings struct {
	// How many instances of the applications can be booted at the same time.
	Max settings.Uint `key:"max" default:"0" desc:"Maximum number of instances of the application that can be booted at the same time"`
}

func (s InstanceSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

type ServicesSettings struct {
	LoaderTimeout  settings.Duration `key:"loader_timeout,save" default:"30s" mutation:"once" desc:"Service loader timeout"`
	RunCronOnStart settings.Bool     `key:"cron_on_service_start,save" default:"false" mutation:"once" desc:"Run cron jobs on service start"`
}

func (s ServicesSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

type StatsSettings struct {
	Enabled settings.Bool `key:"enabled,save" default:"false" mutation:"once"  desc:"Enable runtime statistics"`
}

func (s StatsSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

// Settings for the devel module.
// These settings are used to configure the behavior of the application when user
// compiles your application from source or uses go run .
type DevelSettings struct {
	AllowProd settings.Bool `default:"false" desc:"Allow set app into production mode when running from source."`
}

func (s DevelSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}
