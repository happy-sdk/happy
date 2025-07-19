// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package addon

import (
	"errors"
	"fmt"
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/slug"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/services"
)

var (
	Error = errors.New("addon")
)

type Config struct {
	// Slug is the unique identifier for this addon.
	Slug string
	// DiscardEvents tells application to discard all events this addon emits.
	DiscardEvents bool
	// WithoutCommands tells application to discard all commands this addon registers.
	WithoutCommands bool
	// WithoutServices tells application to discard all services this addon registers.
	WithoutServices bool
}

type Info struct {
	Name        string
	Slug        string
	Description string
	Version     version.Version
	Module      string
}

func Option(key string, dval any) *options.OptionSpec {
	return options.NewOption(key, dval)
}

type Addon struct {
	mu             sync.Mutex
	info           Info
	config         Config
	settings       settings.Settings
	api            api.Provider
	registerAction action.Register

	events      []events.Event
	cmds        []*command.Command
	svcs        []*services.Service
	pendingOpts []*options.OptionSpec
	opts        *options.Spec

	errs         []error
	deprecations []string
	attached     bool
}

func New(name string) *Addon {
	addon := &Addon{
		info: Info{
			Name: name,
		},
	}
	addon.loadPackageInfo()
	return addon
}

func (addon *Addon) WithConfig(cfg Config) *Addon {
	addon.mu.Lock()
	defer addon.mu.Unlock()

	if addon.tryConfigureAttached() {
		return addon
	}
	addon.config = cfg
	if cfg.Slug != "" {
		addon.info.Slug = cfg.Slug
	}
	return addon
}

func (addon *Addon) WithSettings(s settings.Settings) *Addon {
	addon.mu.Lock()
	defer addon.mu.Unlock()

	if addon.tryConfigureAttached() {
		return addon
	}

	addon.settings = s
	return addon
}

func (addon *Addon) WithOptions(opts ...*options.OptionSpec) *Addon {
	addon.mu.Lock()
	defer addon.mu.Unlock()

	if addon.tryConfigureAttached() {
		return addon
	}
	addon.pendingOpts = opts
	return addon
}

func (addon *Addon) WithEvents(evs ...events.Event) *Addon {
	addon.mu.Lock()
	defer addon.mu.Unlock()

	if addon.tryConfigureAttached() {
		return addon
	}

	for _, ev := range evs {
		if ev == nil {
			addon.perr(fmt.Errorf("%w: %s provided <nil> event", Error, addon.info.Name))
			continue
		}
		addon.events = append(addon.events, ev)
	}
	return addon
}

func (addon *Addon) ProvideAPI(api api.Provider) *Addon {
	addon.mu.Lock()
	defer addon.mu.Unlock()

	if addon.tryConfigureAttached() {
		return addon
	}

	if api == nil {
		addon.perr(fmt.Errorf("%w: %s provided <nil> API", Error, addon.info.Name))
		return addon
	}

	if addon.api != nil {
		addon.perr(fmt.Errorf("%w: %s already has an API", Error, addon.info.Name))
		return addon
	}
	addon.api = api
	return addon
}

func (addon *Addon) ProvideCommands(cmds ...*command.Command) *Addon {
	addon.mu.Lock()
	defer addon.mu.Unlock()
	for _, cmd := range cmds {
		if cmd == nil {
			addon.perr(fmt.Errorf("%w: %s provided <nil> command", Error, addon.info.Name))
			return addon
		}
		addon.cmds = append(addon.cmds, cmd)
	}
	return addon
}

func (addon *Addon) ProvideServices(svcs ...*services.Service) *Addon {
	addon.mu.Lock()
	defer addon.mu.Unlock()
	for _, svc := range svcs {
		if svc == nil {
			addon.perr(fmt.Errorf("%w: %s provided <nil> service", Error, addon.info.Name))
			return addon
		}
		addon.svcs = append(addon.svcs, svc)
	}
	return addon
}

func (addon *Addon) OnRegister(action action.Register) *Addon {
	addon.mu.Lock()
	defer addon.mu.Unlock()
	addon.registerAction = action
	return addon
}

// Deprecated allows developers to mark an addon or some of its as deprecated.
func (addon *Addon) Deprecated(msg string) *Addon {
	addon.mu.Lock()
	defer addon.mu.Unlock()
	addon.deprecations = append(addon.deprecations, msg)
	return addon
}

func (addon *Addon) tryConfigureAttached() bool {
	if addon.attached {
		addon.perr(fmt.Errorf("%w: %s already attached", Error, addon.info.Name))
		return addon.attached
	}
	return addon.attached
}

func (addon *Addon) loadPackageInfo() {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		addon.perr(fmt.Errorf("%w: failed to get addon caller info", Error))
		return
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		addon.perr(fmt.Errorf("%w: failed to get addon caller info for pc %d", Error, pc))
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
		addon.perr(fmt.Errorf("%w: failed to get addon build info", Error))
		return
	}
	for _, dep := range buildInfo.Deps {
		if dep.Path == addon.info.Module {
			v, err := version.Parse(dep.Version)
			if err != nil {
				if strings.Contains(dep.Version, "devel") {
					break
				}
				addon.perr(fmt.Errorf("%w: failed to parse addon version %q", Error, dep.Version))
				continue
			}
			addon.info.Version = v
			return
		}
	}

	// if we got here then addon is probably sub module of calling package
	addon.info.Version = version.Current()

	pkgName := path.Base(addon.info.Module)
	if strings.HasPrefix(pkgName, "v") {
		if _, err := strconv.Atoi(pkgName[1:]); err == nil {
			pkgName = path.Base(path.Dir(addon.info.Module))
		}
	}

	if addon.info.Name == "" {
		addon.info.Name = cases.Title(language.English).String(pkgName)
	}
	if addon.info.Slug == "" {
		addon.info.Slug = slug.Create(addon.info.Name)
	}
}

// add pending error
func (addon *Addon) perr(err error) {
	addon.errs = append(addon.errs, err)
}
