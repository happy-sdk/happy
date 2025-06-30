// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package addon

import (
	"errors"
	"fmt"

	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/slug"
	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/services"
	"github.com/happy-sdk/happy/sdk/session"
)

var (
	ErrInvalidAddonName = fmt.Errorf("%s: invalid addon name", Error)
)

type Manager struct {
	// Addons is a map of all registered addons.
	addons map[string]*Addon
}

func NewManager() *Manager {
	return &Manager{
		addons: make(map[string]*Addon),
	}
}

func (m *Manager) Add(addon *Addon) error {
	if !slug.IsValid(addon.info.Slug) {
		return fmt.Errorf("%w: %q", ErrInvalidAddonName, addon.info.Slug)
	}
	if _, ok := m.addons[addon.info.Slug]; ok {
		return fmt.Errorf("%w: %sq addon already attached", Error, addon.info.Slug)
	}
	addon.attached = true
	m.addons[addon.info.Slug] = addon
	var err error
	addon.opts, err = options.New(addon.info.Slug, addon.pendingOpts...)
	addon.pendingOpts = nil
	addon.perr(err)

	return nil
}

func (m *Manager) ExtendSettings(sb *settings.Blueprint) error {
	for _, addon := range m.addons {
		if addon.settings != nil {
			if err := sb.Extend(addon.info.Slug, addon.settings); err != nil {
				return fmt.Errorf("%w: %s", Error, err)
			}
		}
	}
	return nil
}

func (m *Manager) ExtendOptions(opts *options.Spec) error {
	for _, addon := range m.addons {
		if addon.opts != nil {
			if err := opts.Extend(addon.opts); err != nil {
				return fmt.Errorf("%w: %s", Error, err)
			}
		}
	}
	return nil
}

func (m *Manager) Commands() []*command.Command {
	var cmds []*command.Command
	for _, addon := range m.addons {
		if addon.config.WithoutCommands {
			continue
		}
		cmds = append(cmds, addon.cmds...)
	}
	return cmds
}

func (m *Manager) Services() []*services.Service {
	var svcs []*services.Service
	for _, addon := range m.addons {
		if addon.config.WithoutServices {
			continue
		}
		svcs = append(svcs, addon.svcs...)
	}
	return svcs
}

func (m *Manager) Events() []events.Event {
	var evts []events.Event
	for _, addon := range m.addons {
		if addon.config.DiscardEvents {
			continue
		}
		evts = append(evts, addon.events...)
	}
	return evts
}

func (m *Manager) Register(sess session.Register) error {
	for _, addon := range m.addons {
		err := errors.Join(addon.errs...)
		if err != nil {
			return fmt.Errorf("%w(%s): %s", Error, addon.info.Slug, err.Error())
		}
		for _, deprecation := range addon.deprecations {
			sess.Log().Deprecated(deprecation)
		}
		if addon.registerAction == nil {
			continue
		}
		if err := addon.registerAction(sess); err != nil {
			return fmt.Errorf("%w: %s", Error, err)
		}
	}
	return nil
}

func (m *Manager) GetAPIs() map[string]api.Provider {
	apis := make(map[string]api.Provider)
	for _, addon := range m.addons {
		if addon.api == nil {
			continue
		}
		apis[addon.info.Slug] = addon.api
	}
	return apis
}
