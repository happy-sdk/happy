// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package addon

import (
	"errors"
	"fmt"

	"github.com/happy-sdk/happy/pkg/logging"
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

// Manager collects every addon registered with an application (via
// app.WithAddon(s)) and aggregates their commands, services, settings,
// options, events, and APIs into the application's own. The application
// constructs and owns a Manager internally; addon authors don't need to
// interact with it directly.
type Manager struct {
	// addons is a map of all registered addons, keyed by slug.
	addons map[string]*Addon
	// order records the slugs of registered addons in the order Add was
	// called, since map iteration order is randomized and callers (e.g.
	// app.WithAddons(a, b, c)) may rely on addons being processed in the
	// order they were supplied -- for example, two addons with
	// order-dependent setup, or commands/services that should appear in a
	// predictable, caller-controlled order in --help output.
	order []string
}

// NewManager returns a new, empty Manager.
func NewManager() *Manager {
	return &Manager{
		addons: make(map[string]*Addon),
	}
}

// Add registers addon, attaching it (rejecting further configuration calls
// on it) and finalizing its options. It returns an error if addon is nil,
// has an invalid or already-registered slug.
func (m *Manager) Add(addon *Addon) error {
	if addon == nil {
		return fmt.Errorf("%w: attempt to add nil addon", Error)
	}
	if !slug.IsValid(addon.info.Slug) {
		return fmt.Errorf("%w: %q", ErrInvalidAddonName, addon.info.Slug)
	}
	if _, ok := m.addons[addon.info.Slug]; ok {
		return fmt.Errorf("%w: %sq addon already attached", Error, addon.info.Slug)
	}
	addon.attached = true
	m.addons[addon.info.Slug] = addon
	m.order = append(m.order, addon.info.Slug)
	var err error
	addon.opts, err = options.New(addon.info.Slug, addon.pendingOpts...)
	addon.pendingOpts = nil
	addon.perr(err)

	return nil
}

// ExtendSettings extends sb with every registered addon's settings,
// namespaced under each addon's slug, in registration order.
func (m *Manager) ExtendSettings(sb *settings.Blueprint) error {
	for _, addon := range m.ordered() {
		if addon.settings != nil {
			if err := sb.Extend(addon.info.Slug, addon.settings); err != nil {
				return fmt.Errorf("%w: %s", Error, err)
			}
		}
	}
	return nil
}

// ExtendOptions extends opts with every registered addon's options,
// namespaced under each addon's slug, in registration order.
func (m *Manager) ExtendOptions(opts *options.Spec) error {
	for _, addon := range m.ordered() {
		if addon.opts != nil {
			if err := opts.Extend(addon.opts); err != nil {
				return fmt.Errorf("%w: %s", Error, err)
			}
		}
	}
	return nil
}

// Commands returns every command provided by a registered addon (skipping
// addons configured with Config.WithoutCommands), in registration order.
func (m *Manager) Commands() []*command.Command {
	var cmds []*command.Command
	for _, addon := range m.ordered() {
		if addon.config.WithoutCommands {
			continue
		}
		cmds = append(cmds, addon.cmds...)
	}
	return cmds
}

// Services returns every service provided by a registered addon (skipping
// addons configured with Config.WithoutServices), in registration order.
func (m *Manager) Services() []*services.Service {
	var svcs []*services.Service
	for _, addon := range m.ordered() {
		if addon.config.WithoutServices {
			continue
		}
		svcs = append(svcs, addon.svcs...)
	}
	return svcs
}

// Events returns every event a registered addon may emit (skipping addons
// configured with Config.DiscardEvents), in registration order.
func (m *Manager) Events() []events.Event {
	var evts []events.Event
	for _, addon := range m.ordered() {
		if addon.config.DiscardEvents {
			continue
		}
		evts = append(evts, addon.events...)
	}
	return evts
}

// Register finalizes every registered addon, in registration order: it
// returns any pending configuration error recorded by an addon's
// With*/Provide* methods, logs each addon's deprecation notices, and
// invokes each addon's OnRegister callback, if any.
func (m *Manager) Register(sess session.Register) error {
	for _, addon := range m.ordered() {
		err := errors.Join(addon.errs...)
		if err != nil {
			return fmt.Errorf("%w(%s): %s", Error, addon.info.Slug, err.Error())
		}
		for _, deprecation := range addon.deprecations {
			sess.Log().Log(sess.Context(), logging.LevelDepr.Level(), deprecation)
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

// GetAPIs returns every registered addon's custom API (see
// Addon.ProvideAPI), keyed by addon slug.
func (m *Manager) GetAPIs() map[string]api.Provider {
	apis := make(map[string]api.Provider)
	for _, addon := range m.ordered() {
		if addon.api == nil {
			continue
		}
		apis[addon.info.Slug] = addon.api
	}
	return apis
}

// ordered returns all registered addons in the order Add was called.
func (m *Manager) ordered() []*Addon {
	addons := make([]*Addon, 0, len(m.order))
	for _, slug := range m.order {
		addons = append(addons, m.addons[slug])
	}
	return addons
}
