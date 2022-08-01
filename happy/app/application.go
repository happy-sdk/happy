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
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/addon"
	"github.com/mkungla/happy/cli"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/happy/internal/jsonlog"
	"github.com/mkungla/happy/internal/stats"
	"github.com/mkungla/happy/internal/stdlog"
	"github.com/mkungla/happy/service"
	"github.com/mkungla/happy/version"
	"github.com/mkungla/varflag/v5"
)

var (
	ErrInitialization = errors.New("app initialization failed")
	ErrAppOption      = errors.New("invalid app option")
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

	sm *service.Manager
	am *addon.Manager

	errors *list.List

	stats *stats.Stats

	version version.Version

	disposed bool
	started  bool
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
			a.Exit(2, ErrInitialization)
		}
		a.Log().SystemDebugf("%s %s initialized", a.config.Slug, a.version)
	}()

	if !a.initFlags() {
		return nil
	}

	if !a.applyOptions(options...) {
		return nil
	}

	// change log level after user potenially added own logger
	if a.Flag("verbose").Present() {
		a.Log().SetLevel(happy.LevelVerbose)
	} else if a.Flag("debug").Present() {
		a.Log().SetLevel(happy.LevelDebug)
	} else if a.Flag("system-debug").Present() {
		a.Log().SetLevel(happy.LevelSystemDebug)
	}

	a.loadModuleInfo()

	if a.Flag("version").Present() {
		a.Log().SetLevel(happy.LevelQuiet)
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

	return a
}

func (a *Application) Session() happy.Session {
	return a.session
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

// Log returns logger.
func (a *Application) Log() happy.Logger {
	// bug when app fails early e.g. failed flag parser
	if a.logger == nil {
		a.logger = stdlog.New(happy.LevelWarn)
	}
	return a.logger
}

func (a *Application) Dispose(code int) {
	if a.disposed {
		a.Log().SystemDebug("dispose called multiple times")
		return
	}
	a.disposed = true
	a.Log().SystemDebug("app disposing")

	if a.errors.Len() > 0 {
		for e := a.errors.Front(); e != nil; e = e.Next() {
			a.Log().Error(e.Value)
		}
	}

	if len(a.exitFns) > 0 {
		for _, exitFn := range a.exitFns {
			exitFn(code, a.session)
		}
		a.Log().SystemDebug("exit funcs completed")
	}

	if a.sm != nil {
		a.Log().SystemDebug("stopping services...")
		a.sm.Stop()
	}

	if a.session != nil {
		tmpDir := a.session.Get("host.path.tmp").String()
		if _, err := os.Stat(tmpDir); len(tmpDir) > 0 && err == nil {
			if err := a.removeDir("tmp", tmpDir); err != nil {
				a.Log().Error(err)
			}
		}

		a.session.Destroy(nil)
		// <-a.session.Done()
	}

	if a.stats != nil {
		elapsed := a.Stats().Elapsed()
		a.Stats().Dispose()

		a.Log().SystemDebugf("shut down complete - uptime was %s", elapsed)
	}

	_ = a.Log().Sync()
	if a.Flag("json").Present() {
		log, ok := a.logger.(*jsonlog.Logger)
		if ok {
			res := log.GetOutput(nil)
			var (
				pl  []byte
				err error
			)

			if a.Flag("json").String() == "pretty" {
				pl, err = json.MarshalIndent(res, "", "  ")
			} else {
				pl, err = json.Marshal(res)
			}
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(pl))
		}
	}

}

func (a *Application) Exit(code int, err error) {
	a.Log().SystemDebug("shutting down")
	if err != nil {
		a.Log().Error(err)
	}
	a.Dispose(code)
	if a.Flag("json").Present() {
		os.Exit(0)
	}
	os.Exit(code)
}

// Run executes instance based on it's configuration.
func (a *Application) Run() {
	// initialize and configure localhost
	a.initLocalhost()

	if a.errors.Len() > 0 {
		a.Exit(1, errors.New("failed to start the app"))
	}

	if a.Flag("services-keep-alive").Present() && a.sm == nil {
		a.Log().Warn("flag --services-keep-alive has no effect! there are no background services")
		a.Flag("services-keep-alive").Unset()
	}

	if err := a.start(); err != nil {
		if !a.Flag("json").Present() {
			cli.Banner(a.session)
		}

		if errors.Is(err, cli.ErrCommand) {
			a.Exit(127, err)
		} else {
			a.Exit(1, err)
		}
	}
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
	f, err := a.flags.Get(name)
	if err != nil && !errors.Is(err, varflag.ErrNoNamedFlag) {
		a.Log().Error(err)
	}

	if err == nil {
		return f
	}

	// thes could be predefined
	f, err = varflag.Bool(config.CreateSlug(name), false, "")
	if err != nil {
		a.Log().Error(err)
	}
	return f
}

func (a *Application) Flags() varflag.Flags {
	return a.flags
}

func (a *Application) Commands() map[string]happy.Command {
	return a.commands
}

func (a *Application) Command() happy.Command {
	return a.currentCmd
}

// Stats returns application runtime statistics.
func (a *Application) Stats() happy.Stats {
	return a.stats
}

// Config returns application config.
func (a *Application) Config() config.Config {
	return a.config
}

func (a *Application) ServiceManager() happy.ServiceManager {
	if a.sm == nil {
		a.sm = service.NewManager()
	}

	return a.sm
}

// RegisterServices allows you to register individual services to application.
func (a *Application) RegisterServices(serviceFns ...func() (happy.Service, error)) {
	a.Log().Experimental("RegisterServices: services (%d)", len(serviceFns))
	for _, serviceFn := range serviceFns {
		service, err := serviceFn()
		a.addAppErr(err)
		if err == nil {
			a.addAppErr(a.ServiceManager().Register(a.Session().Get("app.slug").String(), service))
			a.Log().Experimental("RegisterServices: registered (%s)", service.Slug())
		}
	}
}

func (a *Application) AddonManager() happy.AddonManager {
	if a.am == nil {
		a.am = addon.NewManager()
	}
	return a.am
}

func (a *Application) RegisterAddons(addonFns ...func() (happy.Addon, error)) {
	for _, addonFn := range addonFns {
		addon, err := addonFn()
		a.addAppErr(err)
		if err == nil {
			a.addAppErr(a.AddonManager().Register(addon))
		}
	}
}

func (a *Application) Store(key string, val any) error {
	a.Log().NotImplementedf("can not set key on app", key)
	return nil
}

func (a *Application) Set(key string, val any) error {
	switch key {
	case "logger":
		if logger, ok := val.(happy.Logger); ok {
			if a.Flag("json").Present() {
				a.logger.SetLevel(logger.Level())
			} else {
				a.logger = logger
			}
		} else {
			return fmt.Errorf("%w: %x", ErrAppOption, val)
		}
	}
	return nil
}

// AddExitFunc append exit function called before application exits.
func (a *Application) AddExitFunc(exitFn func(code int, ctx happy.Session)) {
	a.exitFns = append(a.exitFns, exitFn)
}
