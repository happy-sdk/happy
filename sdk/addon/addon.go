// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package addon

import (
	"errors"
	"fmt"
	"path"
	"reflect"
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
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/custom"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/services"
)

var (
	Error = errors.New("addon")
)

type Config struct {
	Name string
	// DiscardEvents tells application to discard all events this addon emits
	DiscardEvents   bool
	WithoutCommands bool
	WithoutServices bool
	Settings        settings.Settings
}

type Info struct {
	Name        string
	Slug        string
	Description string
	Version     version.Version
	Module      string
}

func Option(key string, dval any, desc string, ro bool, vfunc options.ValueValidator) options.Spec {
	kind := options.KindRuntime
	if ro {
		kind |= options.KindReadOnly
	}
	return options.NewOption(key, dval, desc, kind, vfunc)
}

type Addon struct {
	mu             sync.Mutex
	info           Info
	config         Config
	api            custom.API
	registerAction action.Register

	events []events.Event
	cmds   []*command.Command
	svcs   []*services.Service
	opts   *options.Options

	errs []error
}

func New(c Config, opts ...options.Spec) *Addon {
	addon := &Addon{
		config: c,
		info: Info{
			Name: c.Name,
		},
	}

	if c.Settings != nil && reflect.TypeOf(c.Settings).Kind() == reflect.Ptr {
		addon.perr(fmt.Errorf("%w: %s.Settings must not be a pointer - provide a struct or nil", Error, c.Name))
	}
	addon.loadPackageInfo()

	var err error
	addon.opts, err = options.New(addon.info.Slug, opts)
	addon.perr(err)
	return addon
}

func (addon *Addon) OnRegister(action action.Register) {
	addon.mu.Lock()
	defer addon.mu.Unlock()
	addon.registerAction = action
}

func (addon *Addon) Emits(evs ...events.Event) {
	addon.mu.Lock()
	defer addon.mu.Unlock()
	for _, ev := range evs {
		if ev == nil {
			addon.perr(fmt.Errorf("%w: %s provided <nil> event", Error, addon.info.Name))
			continue
		}
		addon.events = append(addon.events, ev)
	}
}

func (addon *Addon) ProvideCommands(cmds ...*command.Command) {
	addon.mu.Lock()
	defer addon.mu.Unlock()
	for _, cmd := range cmds {
		if cmd == nil {
			addon.perr(fmt.Errorf("%w: %s provided <nil> command", Error, addon.info.Name))
			return
		}
		addon.cmds = append(addon.cmds, cmd)
	}
}

func (addon *Addon) ProvideServices(svcs ...*services.Service) {
	addon.mu.Lock()
	defer addon.mu.Unlock()
	for _, svc := range svcs {
		if svc == nil {
			addon.perr(fmt.Errorf("%w: %s provided <nil> service", Error, addon.info.Name))
			return
		}
		addon.svcs = append(addon.svcs, svc)
	}
}

func (addon *Addon) ProvideAPI(api custom.API) {
	addon.mu.Lock()
	defer addon.mu.Unlock()
	if api == nil {
		addon.perr(fmt.Errorf("%w: %s provided <nil> API", Error, addon.info.Name))
		return
	}
	addon.api = api
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
