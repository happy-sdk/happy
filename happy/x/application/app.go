// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package application

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/cli"
	"github.com/mkungla/happy/x/happyx"
)

type APP struct {
	initialized time.Time
	configured  bool
	logger      happy.Logger
	session     happy.Session
	assets      happy.FS
	engine      happy.Engine

	opts happy.Variables

	rootCmd   happy.Command
	activeCmd happy.Command

	tickAction happy.ActionTickFunc
	tockAction happy.ActionTickFunc

	apis    []happy.API
	slug    happy.Slug
	version happy.Version
	// hasRootDo is used to show general help when rootCmd == activeCmd
	// and rootCmd does not have Do action set.
	hasRootDo bool
}

func (a *APP) Version() happy.Version {
	return a.version
}

// happy.Application interface
func (a *APP) Configure(conf happy.Configurator) (err happy.Error) {
	if a.configured {
		return happyx.Errorf("%w: application already configured", happyx.ErrConfiguration)
	}

	defer func() {
		a.configured = true
	}()

	a.opts = conf.GetApplicationOptions()

	// also sets app version
	if err := a.loadModuleInfo(); err != nil {
		return ErrApplication.Wrap(err)
	}

	slug, _ := a.opts.LoadOrDefault("slug", os.Args[0])
	a.slug = happyx.Slug(slug.String())

	a.logger, err = conf.GetLogger()
	if err != nil {
		return err
	}

	a.session, err = conf.GetSession()
	if err != nil {
		return err
	}

	a.assets, err = conf.GetAssets()
	if err != nil {
		return err
	}

	a.engine, err = conf.GetEngine()
	if err != nil {
		return err
	}
	monitor, err := conf.GetMonitor()
	if err != nil {
		return err
	}
	if err := a.engine.AttachMonitor(monitor); err != nil {
		return err
	}

	rootCmd, err := cli.NewCommand(filepath.Base(os.Args[0]))
	if err != nil {
		return err
	}
	a.rootCmd = rootCmd
	return nil
}

func (a *APP) RegisterAddon(addon happy.Addon) {

	if addon == nil {
		a.logger.Debugf("RegisterAddon got <nil> addon")
		return
	}

	enabled, _ := addon.GetOption("enabled")

	if enabled != nil && !enabled.Bool() {
		a.logger.Debugf(
			"addon: %s, version: %s disabled",
			addon.Slug(),
			addon.Version(),
		)
		return
	}

	for _, cmd := range addon.Commands() {
		a.logger.SystemDebugf("addon: %s provided command %s", addon.Slug(), cmd.Slug())
		a.AddSubCommand(cmd)
	}
	if len(addon.Commands()) == 0 {
		a.logger.SystemDebugf("addon: %s provided no commands", addon.Slug())
	}

	for _, svc := range addon.Services() {
		a.logger.SystemDebugf("addon: %s provided service %s", addon.Slug(), svc.Slug())
		a.RegisterService(svc)
	}
	if len(addon.Services()) == 0 {
		a.logger.SystemDebugf("addon: %s provided no services", addon.Slug())
	}

	info := happy.AddonInfo{
		Name:    addon.Name(),
		Slug:    addon.Slug(),
		Version: addon.Version(),
	}

	a.engine.Monitor().RegisterAddon(info)

	if api := addon.API(); api != nil {
		a.apis = append(a.apis, api)
	}
	a.logger.Debugf(
		"addon: %s, version: %s registered",
		addon.Slug(),
		addon.Version(),
	)
}

func (a *APP) RegisterAddons(acfunc ...happy.AddonCreateFunc) {
	for _, acf := range acfunc {
		if acf == nil {
			a.logger.Warn("RegisterAddons got <nil> arg")
			continue
		}
		addon, err := acf()
		if err != nil {
			a.logger.Emergency(err)
			a.Exit(2)
			return
		}
		a.RegisterAddon(addon)
	}
}

func (a *APP) RegisterService(svc happy.Service) {
	if svc == nil {
		a.logger.Alert("adding <nil> service")
		a.Exit(2)
		return
	}
	if err := a.engine.Register(svc); err != nil {
		a.logger.Alert(err)
		a.Exit(2)
		return
	}

	a.logger.SystemDebugf("added service %s:", svc.URL())
}

func (a *APP) RegisterServices(svcFuncs ...happy.ServiceCreateFunc) {
	for _, svcFunc := range svcFuncs {
		svc, err := svcFunc()
		if err != nil {
			a.logger.Error(err)
			continue
		}
		a.RegisterService(svc)
	}
}

func (a *APP) Log() happy.Logger { return a.logger }

func (a *APP) Main() {
	if err := a.setup(); err != nil {
		a.Log().Emergency(err)
		a.Exit(1)
		return
	}

	showGeneralHelp := a.rootCmd.Flag("help").Present()
	if !showGeneralHelp && (a.rootCmd == a.activeCmd && !a.hasRootDo) {
		showGeneralHelp = true
	}

	// Shall we display default help if so print it and exit with 0
	if showGeneralHelp {
		cli.Help(a.session, showGeneralHelp, a.rootCmd, a.activeCmd)
		a.Exit(0)
		return
	}

	// Start application main process
	go a.execute()

	// block if needed
	osmain()
}

// Slug returns application slug
func (a *APP) Slug() happy.Slug {
	return a.slug
}

// AddFlag to application. Invalid flag or when flag is shadowing
// existing flag log Alert.
func (a *APP) AddFlag(flag happy.Flag) {
	if flag == nil {
		a.logger.Alert("adding <nil> flag")
		a.Exit(2)
		return
	}

	a.logger.SystemDebugf("adding global flag %s:", flag.Name())
	a.rootCmd.AddFlag(flag)
}

func (a *APP) AddFlags(flagFuncs ...happy.FlagCreateFunc) {
	a.rootCmd.AddFlags(flagFuncs...)
}

func (a *APP) AddSubCommand(cmd happy.Command) {
	if cmd == nil {
		a.logger.Alert("adding <nil> command")
		a.Exit(2)
		return
	}
	a.rootCmd.AddSubCommand(cmd)
	a.logger.SystemDebugf("added command %s:", cmd.Slug())
}

func (a *APP) AddSubCommands(cmdFuncs ...happy.CommandCreateFunc) {
	for _, cmdFunc := range cmdFuncs {
		cmd, err := cmdFunc()
		if err != nil {
			a.logger.Error(err)
			a.Exit(2)
			return
		}
		a.AddSubCommand(cmd)
	}
}

func (a *APP) Before(action happy.ActionCommandFunc) {
	a.rootCmd.Before(action)
	a.logger.SystemDebug("set app.Before")
}

func (a *APP) Do(action happy.ActionCommandFunc) {
	a.rootCmd.Do(action)
	a.logger.SystemDebug("set app.Do")
	a.hasRootDo = action != nil
}

func (a *APP) AfterSuccess(action happy.ActionFunc) {
	a.rootCmd.AfterSuccess(action)
	a.logger.SystemDebug("set app.AfterSuccess")
}

func (a *APP) AfterFailure(action happy.ActionWithErrorFunc) {
	a.rootCmd.AfterFailure(action)
	a.logger.SystemDebug("set app.AfterFailure")
}

func (a *APP) AfterAlways(action happy.ActionWithErrorFunc) {
	a.rootCmd.AfterAlways(action)
	a.logger.SystemDebug("set app.AfterAlways")
}

func (a *APP) RequireServices(svcs ...string) {
	a.logger.SystemDebug("app.RequireServices")
}

// happy.TickerFuncs interface
func (a *APP) OnTick(action happy.ActionTickFunc) {
	a.tickAction = action
	a.logger.SystemDebug("set app.OnTick")
}

func (a *APP) OnTock(action happy.ActionTickFunc) {
	a.tockAction = action
	a.logger.SystemDebug("set app.OnTock")
}

// happy.Cron interface
func (a *APP) Cron(happy.ActionCronSchedulerSetup) {
	a.logger.NotImplemented("app.Cron")
}

func (a *APP) Exit(code int) {
	a.exit(code)
}
