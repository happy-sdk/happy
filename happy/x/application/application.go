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
)

type Application struct {
}

func New(conf happy.Configurator) (*Application, happy.Error) {
	app := &Application{}
	return app, happyx.Errorf("application: %w", happyx.ErrNotImplemented)
}

// happy.Application interface
func (a *Application) Configure(conf happy.Configurator) happy.Error {
	return happyx.ErrNotImplemented
}
func (a *Application) AddAddon(happy.Addon)                             {}
func (a *Application) AddAddonCreateFuncs(...happy.AddonCreateFunc)     {}
func (a *Application) AddService(happy.Service)                         {}
func (a *Application) AddServiceCreateFuncs(...happy.ServiceCreateFunc) {}
func (a *Application) Log() happy.Logger                                { return nil }
func (a *Application) Main()                                            {}

// happy.Command interface (root command)
func (a *Application) Slug() happy.Slug                                   { return nil }
func (a *Application) AddFlag(happy.Flag)                                 {}
func (a *Application) AddFlagCreateFunc(...happy.FlagCreateFunc)          {}
func (a *Application) AddSubCommand(happy.Command)                        {}
func (a *Application) AddSubCommandCreateFunc(...happy.CommandCreateFunc) {}
func (a *Application) Before(happy.ActionWithArgsAndAssetsFunc)           {}
func (a *Application) Do(happy.ActionWithArgsAndAssetsFunc)               {}
func (a *Application) AfterSuccess(happy.ActionFunc)                      {}
func (a *Application) AfterFailure(happy.ActionWithErrorFunc)             {}
func (a *Application) AfterAlways(happy.ActionWithErrorFunc)              {}
func (a *Application) RequireServices(svcs ...string)                     {}

// happy.TickerFuncs interface
func (a *Application) OnTick(happy.ActionTickFunc) {}
func (a *Application) OnTock(happy.ActionTickFunc) {}

// happy.Cron interface
func (a *Application) Cron(happy.ActionCronSchedulerSetup) {}
