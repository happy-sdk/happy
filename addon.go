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

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/options"
)

var ErrAddon = fmt.Errorf("%w:addon", Error)

type Addon struct {
	info     AddonInfo
	opts     *options.Options
	settings settings.Settings
	errs     []error

	registerAction Action
	events         []Event

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

func NewAddon(name string, s settings.Settings, opts ...options.OptionSpec) *Addon {
	addon := &Addon{
		settings: s,
		info: AddonInfo{
			Name: name,
		},
	}
	var err error
	addon.opts, err = options.New(name, append(getDefaultAddonConfig(), opts...))
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

func getDefaultAddonConfig() []options.OptionSpec {
	addonOpts := []options.OptionSpec{
		options.NewOption("events.discard", false, "will discard all events this addon emits", options.KindReadOnly|options.KindConfig, nil),
	}
	return addonOpts
}

func (addon *Addon) OnRegister(action Action) {
	addon.registerAction = action
}

func (addon *Addon) Emits(scope, key, description string, example *vars.Map) {
	addon.EmitsEvent(registrableEvent(scope, key, description, example))
}

func (addon *Addon) EmitsEvent(event Event) {
	addon.events = append(addon.events, event)
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

func (addon *Addon) register(sess *Session) error {
	if addon.errs != nil {
		return errors.Join(addon.errs...)
	}

	if addon.registerAction != nil {
		if err := addon.registerAction(sess); err != nil {
			return fmt.Errorf("%w(%s): %s", ErrAddon, addon.info.Name, err)
		}
	}
	if err := addon.opts.Seal(); err != nil {
		return err
	}
	return nil
}
