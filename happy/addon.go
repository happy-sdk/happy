// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"fmt"

	"github.com/happy-sdk/happy-go/devel/version"
	"github.com/happy-sdk/happy-go/settings"
	"github.com/happy-sdk/happy-go/vars"
	"golang.org/x/mod/semver"
)

type Addon struct {
	info     AddonInfo
	opts     *Options
	settings settings.Settings
	errs     []error

	registerAction ActionWithOptions
	events         []Event
	acceptsOpts    []OptionArg

	cmds []*Command
	svcs []*Service

	api API
}

type AddonInfo struct {
	Name        string
	Description string
	Version     version.Version
}

func NewAddon(name string, s settings.Settings, opts ...OptionArg) *Addon {
	addon := &Addon{
		settings: s,
		info: AddonInfo{
			Name: name,
		},
	}
	var err error
	addon.opts, err = NewOptions("config", getDefaultAddonConfig())
	if err != nil {
		addon.errs = append(addon.errs, err)
	}
	return addon
}

func getDefaultAddonConfig() []OptionArg {
	addonOpts := []OptionArg{
		{
			key:   "description",
			value: "",
			desc:  "Short description for addon",
			kind:  ReadOnlyOption | ConfigOption,
		},
		{
			key:   "version",
			value: version.Current(),
			desc:  "Addon version",
			kind:  ReadOnlyOption | ConfigOption,
			validator: func(key string, val vars.Value) error {
				if !semver.IsValid(val.String()) {
					return fmt.Errorf("%w %q, version must be valid semantic version", ErrInvalidVersion, val)
				}
				return nil
			},
		},
	}
	return addonOpts
}

func (addon *Addon) OnRegister(action ActionWithOptions) {
	addon.registerAction = action
}

func (addon *Addon) Emits(scope, key, description string, example *vars.Map) {
	addon.EmitsEvent(registerEvent(scope, key, description, example))
}

func (addon *Addon) EmitsEvent(event Event) {
	addon.events = append(addon.events, event)
}

func (addon *Addon) Option(key string, value any, description string, validator OptionValueValidator) {
	addon.acceptsOpts = append(addon.acceptsOpts, OptionArg{
		key:       key,
		value:     value,
		desc:      description,
		kind:      SettingsOption,
		validator: validator,
	})
}

func (addon *Addon) ProvidesCommand(cmd *Command) {
	if cmd == nil {
		addon.errs = append(addon.errs, fmt.Errorf("%w: %s provided <nil> command", ErrAddon, addon.info.Name))
		return
	}
	addon.cmds = append(addon.cmds, cmd)
}

func (addon *Addon) ProvidesService(svc *Service) {
	if svc == nil {
		addon.errs = append(addon.errs, fmt.Errorf("%w: %s provided <nil> service", ErrAddon, addon.info.Name))
		return
	}
	addon.svcs = append(addon.svcs, svc)
}

func (addon *Addon) ProvidesAPI(api API) {
	if svc == nil {
		addon.errs = append(addon.errs, fmt.Errorf("%w: %s provided <nil> API", ErrAddon, addon.info.Name))
		return
	}
	addon.api = api
}
