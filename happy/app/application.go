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

package app

import (
	"bytes"
	"container/list"
	"fmt"
	"os"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/cli"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/happy/internal/session"
	"github.com/mkungla/happy/internal/stats"
	"github.com/mkungla/happy/internal/stdlog"
	"github.com/mkungla/happy/internal/version"
	"github.com/mkungla/varflag/v5"
	"github.com/mkungla/vars/v5"
)

type Application struct {
	config  config.Config
	exitFns []func(code int, session happy.Session)

	logger happy.Logger

	session happy.Session // application runtime context
	flags   varflag.Flags

	commands    map[string]happy.Command // commands
	currentCmd  happy.Command
	rootCmd     happy.Command
	setupAction happy.Action

	sm happy.ServiceManager
	am happy.AddonManager

	errors *list.List

	stats *stats.Stats

	version version.Version
}

func New(options ...happy.Option) happy.Application {
	var (
		lasterr error
	)
	a := &Application{
		errors: list.New(),
	}

	// exit app if there were initialization errors
	// and try to log these errors to appropriate logger.
	defer func() {
		if a.errors.Len() > 0 {
			a.Exit(2)
		}
		a.Log().SystemDebugf("%s %s initialized", a.config.Slug, a.version)
	}()

	a.session = session.New()

	if !a.applyOptions(options...) || !a.initFlags() {
		return nil
	}

	a.loadVersion()

	if a.Flag("version").Present() {
		fmt.Fprintln(os.Stdout, a.version.String())
		a.Exit(0, nil)
	}

	// Stats
	a.stats = stats.New()

	// Add NotifyContext only if services-keep-alive is not present
	if !a.Flag("services-keep-alive").Present() {
		a.session.NotifyContext(os.Interrupt, os.Kill)
	}

	if a.rootCmd, lasterr = cli.NewCommand(a.config.Slug, 0); lasterr != nil {
		a.errors.PushBack(lasterr)
		return nil
	}

	a.Log().SystemDebugf("init service manager")
	a.Log().SystemDebugf("init addon manager")

	var env bytes.Buffer
	env.WriteString("SESSION\n")

	a.session.Range(func(key string, val vars.Value) bool {
		env.WriteString(fmt.Sprintf("%-25s %10s = %s\n", key, "("+val.Type().String()+")", val.String()))
		return true
	})
	a.Log().SystemDebug(env.String())
	return a
}

// AddCommand to application.
//
// Added Commands and command flags will be verified
// upon application startup and will prevent application
// to start if command was invalid or command introduces
// any flag shadowing.
func (a *Application) AddCommand(c happy.Command) {
	a.Log().SystemDebug("a.AddCommand: ", c)
	a.addCommand(c)
}

func (a *Application) AddCommandFn(fn func() (happy.Command, error)) {
	cmd, err := fn()
	if err != nil {
		a.Exit(1, err)
		return
	}
	a.AddCommand(cmd)
}

func (a *Application) Log() happy.Logger {
	if a.logger == nil {
		a.logger = stdlog.New()
	}
	return a.logger
}

func (a *Application) Dispose(code int, errs ...error) {
	a.Log().SystemDebug("a.Dispose: ")
	if len(errs) > 0 {
		for _, err := range errs {
			a.Log().Error(err)
		}
	}

	if a.errors.Len() > 0 {
		for e := a.errors.Front(); e != nil; e = e.Next() {
			a.Log().Error(e.Value)
		}
	}
	a.Stats().Close()
}

func (a *Application) Exit(code int, errs ...error) {
	a.Log().SystemDebug("a.Exit: ")
	a.Dispose(code, errs...)
	os.Exit(code)
}

// Run executes instance based on it's configuration.
func (a *Application) Run() {
	a.Log().SystemDebug("a.Run: ")
}

// Setup function enables you add pre hook for application.
// Useful to perform user environment checks and trigger
// initial configuration or configuration wizards.
func (a *Application) Setup(action happy.Action) {
	a.setupAction = action
}

// Before function for root cmd.
func (a *Application) Before(action happy.Action) {
	a.rootCmd.Before(action)
}

// Do function for root cmd.
func (a *Application) Do(action happy.Action) {
	a.rootCmd.Do(action)
}

// AfterSuccess root function runs when Application exits with exit code 0.
func (a *Application) AfterSuccess(action happy.Action) {
	a.rootCmd.AfterSuccess(action)
}

// AfterFailure root function runs before os.Exit
// when application exits with non 0 status.
func (a *Application) AfterFailure(action happy.Action) {
	a.rootCmd.AfterFailure(action)
}

// AfterAlways root function runs always as last callback
// before application exits and session context is canceled.
func (a *Application) AfterAlways(action happy.Action) {
	a.rootCmd.AfterAlways(action)
}

// Flag looks up flag with given name and returns flags.Interface.
// If no flag was found empty bool flag will be returned.
// Instead of returning error you can check returned flags .IsPresent.
func (a *Application) Flag(name string) varflag.Flag {
	if f, err := a.flags.Get(name); err != nil {
		a.Log().Error(err)
		return f
	}

	// thes could be predefined
	f, err := varflag.Bool(config.CreateSlug(name), false, "")
	if err != nil {
		a.Log().Error(err)
	}
	return f
}

// Stats returns application runtime statistics.
func (a *Application) Stats() happy.Stats {
	return a.stats
}

// Config returns application config.
func (a *Application) Config() config.Config {
	return a.config
}
