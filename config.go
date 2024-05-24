// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/happy-sdk/happy/pkg/branding"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/networking/address"

	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/datetime"
	"github.com/happy-sdk/happy/sdk/instance"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/services"
	"github.com/happy-sdk/happy/sdk/stats"
	"golang.org/x/text/language"
)

type Settings struct {
	// Appication info
	Name           settings.String `key:"app.name" default:"Happy Prototype"`
	Slug           settings.String `key:"app.slug" default:"" mutation:"once"`
	Identifier     settings.String `key:"app.identifier"`
	Description    settings.String `key:"app.description" default:"This application is built using the Happy-SDK to provide enhanced functionality and features."`
	CopyrightBy    settings.String `key:"app.copyright_by" default:"Anonymous"`
	CopyrightSince settings.Uint   `key:"app.copyright_since" default:"0" mutation:"once"`
	License        settings.String `key:"app.license" default:"NOASSERTION"`

	// Application settings
	CLI      cli.Settings      `key:"app.cli"`
	DateTime datetime.Settings `key:"app.datetime"`
	Instance instance.Settings `key:"app.instance"`
	Logging  logging.Settings  `key:"app.logging"`
	Services services.Settings `key:"app.services"`
	Stats    stats.Settings    `key:"app.stats"`

	global     []settings.Settings
	migrations map[string]string
	errs       []error
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

func (i *initializer) unsafeInitSettings(m *Main, settingsb *settings.Blueprint) error {
	for _, opt := range i.mainOptSpecs {
		if err := m.sess.Opts().Add(opt); err != nil {
			return err
		}
	}

	mainArgcMaxSpec, mainArgcMaxErr := settingsb.GetSpec("app.cli.argn_max")
	if mainArgcMaxErr != nil {
		return mainArgcMaxErr
	}
	argnmax, err := strconv.ParseUint(mainArgcMaxSpec.Value, 10, 64)
	if err != nil {
		return err
	}
	if err := m.root.setArgcMax(uint(argnmax)); err != nil {
		return err
	}

	slugSpec, slugErr := settingsb.GetSpec("app.slug")
	if slugErr != nil {
		return slugErr
	}
	insRevDNSSpec, insRevDNSErr := settingsb.GetSpec("app.identifier")
	if insRevDNSErr != nil {
		return insRevDNSErr
	}
	rdns := insRevDNSSpec.Value

	m.slug = slugSpec.Value

	// Set the root name to the slug early so that it can be used by help menu
	m.root.name = m.slug

	curr, err := address.Current()
	if err != nil {
		return err
	}

	if len(m.slug) == 0 {
		if testing.Testing() {
			tmpaddr, err := address.CurrentForDepth(2)
			if err != nil {
				return err
			}
			m.slug = tmpaddr.Instance() + "-test"

			if err := m.sess.opts.Set("app.module", tmpaddr.Module()); err != nil {
				return err
			}
			rdns = tmpaddr.ReverseDNS() + ".test"
		} else {
			m.slug = curr.Instance()
		}
		if err := settingsb.SetDefault("app.slug", m.slug); err != nil {
			return err
		}
	}
	if len(rdns) == 0 {
		rdns = curr.ReverseDNS()
		if len(rdns) == 0 {
			return fmt.Errorf("could not find app.identifier")
		}
	}
	if err := settingsb.SetDefault("app.identifier", rdns); err != nil {
		return err
	}

	inst, err := instance.New(m.slug, rdns)
	if err != nil {
		return err
	}
	m.instance = inst

	if err := slugSpec.ValidateValue(m.slug); err != nil {
		return err
	}

	if err := m.sess.opts.Set("app.address", inst.Address().String()); err != nil {
		return err
	}

	return nil
}

func getConfig() []options.OptionSpec {
	ver := version.Current()
	addr, _ := address.Current()

	opts := []options.OptionSpec{
		// {
		// 	Key:   "*",
		// 	Value: "",
		// 	Kind:  ReadOnlyOption | RuntimeOption,
		// 	validator: func(key string, val vars.Value) error {
		// 		if strings.HasPrefix(key, "app.") || strings.HasPrefix(key, "log.") || strings.HasPrefix(key, "happy.") || strings.HasPrefix(key, "fs.") {
		// 			return fmt.Errorf("%w: unknown application option %s", ErrOptionValidation, key)
		// 		}
		// 		return nil
		// 	},
		// },
		options.NewOption(
			"app.devel",
			version.IsDev(ver.String()),
			"Is application in development mode",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.version",
			ver.String(),
			"Application version",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.pwd",
			"",
			"Current working directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.home",
			"",
			"Current user home directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.tmp",
			"",
			"Runtime tmp directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.cache",
			"",
			"Application cache directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.config",
			"",
			"Application configuration directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.pids",
			"",
			"Application pids directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.main.exec.x",
			"",
			"-x flag is set to print all commands as executed",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.profile.name",
			"",
			"name of current settings profile",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.profile.preferences",
			"",
			"file path of current settings profile file",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.firstuse",
			false,
			"application first use",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.module",
			addr.Module(),
			"application module",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.address",
			"",
			"application address",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.pid",
			0,
			"application process id",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
	}
	return opts
}

func (i *initializer) unsafeInitBrand(m *Main, settingsb *settings.Blueprint) error {
	if i.brand != nil {
		m.brand = i.brand
	} else {
		nameSpec, nameErr := settingsb.GetSpec("app.name")
		if nameErr != nil {
			return nameErr
		}
		builder := branding.New(branding.Info{
			Name:    nameSpec.Value,
			Slug:    m.slug,
			Version: m.sess.Get("app.version").String(),
		})
		brand, err := builder.Build()
		if err != nil {
			return err
		}
		m.brand = brand
	}
	return nil
}
