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
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/pkg/varflag"
	"os"
	"time"
)

type APP struct {
	initialized time.Time
	configured  bool
	logger      happy.Logger
	session     happy.Session
	monitor     happy.ApplicationMonitor
	assets      happy.FS
	engine      happy.Engine

	opts happy.Variables

	flags happy.Flags
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

	a.logger, err = conf.GetLogger()
	if err != nil {
		return err
	}

	a.session, err = conf.GetSession()
	if err != nil {
		return err
	}

	a.monitor, err = conf.GetMonitor()
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

	flags, ferr := varflag.NewFlagSetAs[
		happy.Flags,    // flagset
		happy.Flags,    // sub flagset
		happy.Flag,     // flag
		happy.Variable, // flag values
		happy.Value,    // arguements
	](os.Args[0], 0)
	if ferr != nil {
		return err
	}
	a.flags = flags

	return nil
}

func (a *APP) RegisterAddon(addon happy.Addon) {
	if addon == nil {
		a.logger.Warn("RegisterAddon got <nil> addon")
		return
	}

	a.logger.SystemDebugf(
		"registered addon name: %s, version: %s",
		addon.Name(),
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

func (a *APP) RegisterService(happy.Service) {
	a.logger.SystemDebug("app.RegisterService")
}

func (a *APP) RegisterServices(...happy.ServiceCreateFunc) {
	a.logger.SystemDebug("app.RegisterServices")
}

func (a *APP) Log() happy.Logger { return a.logger }

func (a *APP) Main() {
	if err := a.setup(); err != nil {
		a.Log().Emergency(err)
		a.Exit(1)
		return
	}

	if err := a.engine.Start(); err != nil {
		a.Log().Emergency(err)
		a.Exit(1)
		return
	}

	a.Exit(0)
}

// happy.Command interface (root command)
func (a *APP) Slug() happy.Slug { return nil }

// AddFlag to application. Invalid flag or when flag is shadowing
// existing flag log Alert.
func (a *APP) AddFlag(flag happy.Flag) {
	if flag == nil {
		a.logger.Alert("adding <nil> flag")
		return
	}

	if err := a.flags.Add(flag); err != nil {
		a.logger.Alertf("failed to add flag %s:", err)
	}
	a.logger.SystemDebugf("added global flag %s:", flag.Name())
}

func (a *APP) AddFlags(flagFuncs ...happy.FlagCreateFunc) {
	for _, flagFunc := range flagFuncs {
		flag, err := flagFunc()
		if err != nil {
			a.logger.Error(err)
		}
		a.AddFlag(flag)
	}
}

func (a *APP) AddSubCommand(happy.Command) {
	a.logger.SystemDebug("app.AddSubCommand")
}
func (a *APP) AddSubCommands(...happy.CommandCreateFunc) {
	a.logger.SystemDebug("app.AddSubCommands")
}
func (a *APP) Before(happy.ActionWithArgsAndAssetsFunc) {
	a.logger.SystemDebug("app.Before")
}
func (a *APP) Do(happy.ActionWithArgsAndAssetsFunc) {
	a.logger.SystemDebug("app.Do")
}
func (a *APP) AfterSuccess(happy.ActionFunc) {
	a.logger.SystemDebug("app.AfterSuccess")
}
func (a *APP) AfterFailure(happy.ActionWithErrorFunc) {
	a.logger.SystemDebug("app.AfterFailure")
}
func (a *APP) AfterAlways(happy.ActionWithErrorFunc) {
	a.logger.SystemDebug("app.AfterAlways")
}
func (a *APP) RequireServices(svcs ...string) {
	a.logger.SystemDebug("app.RequireServices")
}

// happy.TickerFuncs interface
func (a *APP) OnTick(happy.ActionTickFunc) {
	a.logger.SystemDebug("app.OnTick")
}
func (a *APP) OnTock(happy.ActionTickFunc) {
	a.logger.SystemDebug("app.OnTock")
}

// happy.Cron interface
func (a *APP) Cron(happy.ActionCronSchedulerSetup) {
	a.logger.SystemDebug("app.Cron")
}

func (a *APP) Exit(code int) {
	a.exit(code)
}
