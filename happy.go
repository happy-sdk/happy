// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

// Package happy provides a modular framework for rapid prototyping in Go. With this SDK, developers
// of all levels can easily bring their ideas to life. Whether you're a hacker or a creator, Package
// happy has everything you need to tackle your domain problems and create working prototypes or MVPs
// with minimal technical knowledge and infrastructure planning.
//
// Its modular design enables you to package your commands and services into reusable addons, so you're
// not locked into any vendor tools. It also fits well into projects where different components are written
// in different programming languages.
//
// Let Package happy help you bring your projects from concept to reality and make you happy along the way.
package happy

import (
	"errors"
	"fmt"

	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/slug"
	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/app"
	"github.com/happy-sdk/happy/sdk/app/engine"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/config"
	"github.com/happy-sdk/happy/sdk/datetime"
	"github.com/happy-sdk/happy/sdk/devel"
	"github.com/happy-sdk/happy/sdk/instance"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/services"
	"github.com/happy-sdk/happy/sdk/session"
	"github.com/happy-sdk/happy/sdk/stats"
)

func New(s Settings) *app.Main {
	return app.New(s)
}

type Settings struct {
	// Appication info
	Name           settings.String `key:"app.name" default:"Happy Prototype" desc:"Application name"`
	Slug           settings.String `key:"app.slug" default:"" desc:"Application slug"`
	Identifier     settings.String `key:"app.identifier" desc:"Application identifier"`
	Description    settings.String `key:"app.description" default:"This application is built using the Happy-SDK to provide enhanced functionality and features." desc:"Application description"`
	CopyrightBy    settings.String `key:"app.copyright_by" default:"Anonymous" desc:"Application author"`
	CopyrightSince settings.Uint   `key:"app.copyright_since" default:"0" desc:"Application copyright since"`
	License        settings.String `key:"app.license" default:"NOASSERTION" desc:"Application license"`

	// Application settings
	Engine     engine.Settings   `key:"app.engine"`
	CLI        cli.Settings      `key:"app.cli"`
	Config     config.Settings   `key:"app.config"`
	DateTime   datetime.Settings `key:"app.datetime"`
	Instance   instance.Settings `key:"app.instance"`
	Logging    logging.Settings  `key:"app.logging"`
	Services   services.Settings `key:"app.services"`
	Stats      stats.Settings    `key:"app.stats"`
	Devel      devel.Settings    `key:"app.devel"`
	I18n       i18n.Settings     `key:"app.i18n"`
	global     []settings.Settings
	migrations map[string]string
	errs       []error
}

// Blueprint returns a blueprint for the settings.
func (s Settings) Blueprint() (*settings.Blueprint, error) {

	b, err := settings.New(&s)
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

// Extend adds a new settings group to the application settings.
func (s *Settings) Extend(ss settings.Settings) {
	s.global = append(s.global, ss)
}

// API returns the API for the given addon slug if addon has given API registered.
func API[API api.Provider](sess *session.Context, addonSlug string) (api API, err error) {
	return session.API[API](sess, addonSlug)
}
