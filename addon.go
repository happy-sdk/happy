// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy

import (
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/settings"
	"golang.org/x/mod/semver"
)

var ErrAddon = fmt.Errorf("%w:addon", Error)

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
	Module      string
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
		addon.perr(err)
	}
	addon.setAddonPackageInfo()
	return addon
}

func (addon *Addon) setAddonPackageInfo() {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		addon.perr(fmt.Errorf("%w: failed to get addon caller info", ErrAddon))
		return
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		addon.perr(fmt.Errorf("%w: failed to get addon caller info for pc %d", ErrAddon, pc))
		return
	}
	fnName := fn.Name()

	lastDotIndex := strings.LastIndex(fnName, ".")
	if lastDotIndex == -1 {
		addon.info.Module = fnName
	} else {
		addon.info.Module = fnName[:lastDotIndex]
	}

	// In test mode, the path may include "_test" suffix, which we should strip.
	// if b.mode == settings.ModeTesting {
	// 	pkgPath, _, _ = strings.Cut(pkgPath, "_test")
	// }

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		addon.perr(fmt.Errorf("%w: failed to get addon build info", ErrAddon))
		return
	}
	for _, dep := range buildInfo.Deps {
		if dep.Path == addon.info.Module {
			v, err := version.Parse(dep.Version)
			if err != nil {
				if strings.Contains(dep.Version, "devel") {
					break
				}
				addon.perr(fmt.Errorf("%w: failed to parse addon version %q", ErrAddon, dep.Version))
				continue
			}
			addon.info.Version = v
			return
		}
	}

	// if we got here then addon is probably sub module of calling package
	addon.info.Version = version.Current()
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
					return fmt.Errorf("%w: %q, version must be valid semantic version", errors.Join(Error, version.Error), val)
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
	addon.EmitsEvent(registrableEvent(scope, key, description, example))
}

func (addon *Addon) EmitsEvent(event Event) {
	addon.events = append(addon.events, event)
}

func (addon *Addon) Option(key string, value any, description string, validator OptionValueValidator) {
	addon.acceptsOpts = append(addon.acceptsOpts, OptionArg{
		key:       key,
		value:     value,
		desc:      description,
		kind:      RuntimeOption,
		validator: validator,
	})
}

func (addon *Addon) ProvidesCommand(cmd *Command) {
	if cmd == nil {
		addon.perr(fmt.Errorf("%w: %s provided <nil> command", ErrAddon, addon.info.Name))
		return
	}
	addon.cmds = append(addon.cmds, cmd)
}

func (addon *Addon) ProvidesService(svc *Service) {
	if svc == nil {
		addon.perr(fmt.Errorf("%w: %s provided <nil> service", ErrAddon, addon.info.Name))
		return
	}
	addon.svcs = append(addon.svcs, svc)
}

func (addon *Addon) ProvidesAPI(api API) {
	if api == nil {
		addon.perr(fmt.Errorf("%w: %s provided <nil> API", ErrAddon, addon.info.Name))
		return
	}
	addon.api = api
}

// add pending error
func (addon *Addon) perr(err error) {
	addon.errs = append(addon.errs, err)
}
