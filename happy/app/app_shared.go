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
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/cli"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/varflag/v6"
	"github.com/mkungla/vars/v6"
)

func (a *Application) addAppErr(err error) {
	if err == nil {
		return
	}
	a.errors.PushBack(err)
}

func (a *Application) printEnv() {
	var (
		sessionKeys        []string
		settingKeys        []string
		longestSessionKey  int
		longestSettingsKey int
	)
	sessionVars := make(map[string]string)
	settings := make(map[string]string)

	a.session.Range(func(key string, val vars.Value) bool {
		sessionVars[key] = fmt.Sprintf("%-8s = %s\n", "("+val.Type().String()+")", val.String())
		sessionKeys = append(sessionKeys, key)
		if len(key) > longestSessionKey {
			longestSessionKey = len(key)
		}
		return true
	})
	a.session.Settings().Range(func(key string, val vars.Value) bool {
		settings[key] = fmt.Sprintf("%-8s = %s\n", "("+val.Type().String()+")", val.String())
		settingKeys = append(settingKeys, key)

		if len(key) > longestSettingsKey {
			longestSettingsKey = len(key)
		}
		return true
	})

	sessionRowTmpl := fmt.Sprintf("%%-%ds%%s", longestSessionKey+1)
	sort.Strings(sessionKeys)
	settingsRowTmpl := fmt.Sprintf("%%-%ds%%s", longestSettingsKey+1)
	sort.Strings(settingKeys)

	var env bytes.Buffer
	env.WriteString("\nSESSION\n")
	for _, k := range sessionKeys {
		env.WriteString(fmt.Sprintf(sessionRowTmpl, k, sessionVars[k]))
	}
	env.WriteString("\nSETTINGS\n")
	for _, k := range settingKeys {

		env.WriteString(fmt.Sprintf(settingsRowTmpl, k, settings[k]))
	}

	fmt.Fprintln(os.Stdout, env.String())
}

func (a *Application) run() error {
	// initialized application can proceed to initial setup

	a.Log().SystemDebugf("application startup took %s", a.Stats().Elapsed())

	// call app setup fn to
	// enables to override some setting defaults
	if a.setupAction != nil {
		if err := a.setupAction(a.session); err != nil {
			return err
		}
	}

	// Start application main process
	go a.execute()

	// block if needed
	appmain()
	return nil
}

func (a *Application) start() error {
	if a.started {
		return errors.New("app is already running")
	}
	a.started = true

	// should these un after a.flags.Parse?
	if a.Flag("show-bash-completion").Present() {
		bashcompletion(a.commands, a.config.Slug)
		a.Exit(0, nil)
		return nil
	}

	for _, addon := range a.AddonManager().Addons() {
		a.Log().SystemDebugf("loading addon %s %s", addon.Slug(), addon.Version())
		settings := addon.DefaultSettings(a.session)
		for i := range settings.General.Settings {
			key := strings.Join([]string{"addon", addon.Slug(), settings.General.Settings[i].Key}, ".")
			if !config.ValidSettingKey(key) {
				a.Exit(1, fmt.Errorf("%w: %s", config.ErrInvalidSettingsKey, key))
			}
			if !a.session.Settings().Has(key) {
				a.Log().SystemDebugf("setting default for %s", key)
				if err := a.session.Settings().Set(key, settings.General.Settings[i].Default()); err != nil {
					return err
				}
			}

		}

		if addon.Configured(a.session) {
			cmds, err := addon.Commands()
			if err != nil {
				return err
			}
			for _, cmd := range cmds {
				a.AddCommand(cmd)
			}

			if err := a.ServiceManager().Register("addon."+addon.Slug(), addon.Services()...); err != nil {
				a.Exit(1, err)
			}

		} else {
			a.Log().SystemDebugf("addon %s is not configured", addon.Slug())
		}
	}

	// verify configuration of commands
	for _, cmd := range a.commands {
		if err := cmd.Verify(); err != nil {
			a.addAppErr(fmt.Errorf("%s %w", cmd.String(), err))
		} else {
			a.flags.AddSet(cmd.Flags())
		}
	}

	if err := a.flags.Parse(os.Args); err != nil && !errors.Is(err, varflag.ErrFlagAlreadyParsed) {
		if errors.Is(err, varflag.ErrMissingRequired) && !a.Flag("help").Present() {
			return err
		}
	}

	a.Log().Issue(28, "Check for invalid global flags")

	a.prepareCommand()

	if a.errors.Len() > 0 {
		return errors.New("failed to prepare command")
	}

	// run root command if do fn exits
	if a.currentCmd == nil {
		a.currentCmd = a.rootCmd
	}

	// Shall we display default help if so print it and exit with 0
	cli.Help(a)

	if a.currentCmd == nil {
		return errors.New("no command, see (--help) for available commands")
	}

	curflags := a.flags
	curflags.Add(a.currentCmd.Flags().Flags()...)

	for p := a.currentCmd.Parent(); p != nil; p = p.Parent() {
		for _, f := range p.Flags().Flags() {
			if f.CommandName() == "*" {
				curflags.Add(f)
			}
		}
	}

	if err := a.session.Start(
		a.currentCmd.String(),
		a.currentCmd.Args(),
		curflags,
	); err != nil {
		return err
	}

	return a.run()
}

func (a *Application) prepareCommand() {
	settree := a.flags.GetActiveSetTree()
	name := settree[len(settree)-1].Name()

	if name != "/" {
		var (
			cmd    happy.Command
			exists bool
		)
		for _, set := range settree {
			// skip root
			if set.Name() == "/" {
				continue
			}
			if cmd == nil {
				cmd, exists = a.commands[set.Name()]
				if !exists {
					a.addAppErr(fmt.Errorf("%w: unknown command (%s)", cli.ErrCommand, name))
					return
				}
			} else {
				cmd, exists = cmd.SubCommand(set.Name())
				if !exists {
					a.addAppErr(fmt.Errorf("%w: unknown subcommand (%s) for %s", cli.ErrCommand, name, cmd.String()))
					return
				}
			}
			a.currentCmd = cmd
		}
	} else {
		cmd, exists := a.commands[a.config.Slug]
		if !exists {
			// Not having root command is not a error.
			// It is treated as no command was provided
			return
		}
		a.currentCmd = cmd
	}
}

func (a *Application) execute() {

	// initialize services
	a.Log().Experimentalf("initialize %d services", a.ServiceManager().Len())
	if a.ServiceManager().Len() > 0 {
		err := a.ServiceManager().Initialize(
			a.session,
			a.Flag("services-keep-alive").Present(),
		)
		if err != nil {
			a.Exit(1, err)
		}
	}

	a.session.TaskAdd("start services if needed")

	cancelStats := a.Stats().Start()
	defer cancelStats()

	go func() {
		a.ServiceManager().Start(a.session, a.currentCmd.ServiceLoaders()...)
		a.session.TaskDone()

		a.Log().Experimental("starting service manager runtime")
	serviceTicker:
		for {
			select {
			case <-a.session.Done():
				a.Log().Experimental("stopping service manager runtime")
				break serviceTicker
			case <-a.Stats().Next():
				a.ServiceManager().Tick(a.session)
			}
		}
	}()

	// app before runs always
	a.executeBeforeFn()

	// block
	a.session.Ready()

	// log env if
	if a.Log().Level() == happy.LevelSystemDebug && !a.Flag("json").Present() {
		a.printEnv()
	}
	if a.session.Err() != nil {
		a.Exit(1, a.session.Err())
	}

	var code int
	isRootCmd := a.currentCmd == a.rootCmd
	if err := a.currentCmd.ExecuteDoFn(a.session); err != nil { //nolint:nestif // yes deeply nested

		a.addAppErr(err)
		code = 2
		if err := a.currentCmd.ExecuteAfterFailureFn(a.session); err != nil {
			a.addAppErr(err)
			code = 1
		}

		if a.rootCmd != nil && !isRootCmd {
			if err := a.rootCmd.ExecuteAfterFailureFn(a.session); err != nil {
				a.addAppErr(err)
			}
		}
	} else {
		code = 0
		if err := a.currentCmd.ExecuteAfterSuccessFn(a.session); err != nil {
			a.addAppErr(err)
			code = 1
		}
		if a.rootCmd != nil && !isRootCmd {
			a.addAppErr(a.rootCmd.ExecuteAfterSuccessFn(a.session))
		}
	}

	a.executeAfterAlwaysFn(code)
}

func (a *Application) executeBeforeFn() {
	if a.rootCmd != nil && a.currentCmd != a.rootCmd {
		if err := a.rootCmd.ExecuteBeforeFn(a.session); err != nil {
			a.Exit(1, err)
		}
	}

	if err := a.currentCmd.ExecuteBeforeFn(a.session); err != nil {
		a.Exit(1, err)
	}
}

func (a *Application) executeAfterAlwaysFn(code int) {
	if err := a.currentCmd.ExecuteAfterAlwaysFn(a.session); err != nil {
		a.Log().Error(err)
		code = 1
	}

	if a.rootCmd != nil && a.currentCmd != a.rootCmd {
		if err := a.rootCmd.ExecuteAfterAlwaysFn(a.session); err != nil {
			a.Exit(1, err)
		}
	}

	if a.Flag("services-keep-alive").Present() && code == 0 {
		a.Flag("services-keep-alive").Unset()
		a.Log().SystemDebug("UI exited with code(0), but service is in keep-alive mode, continue exec")
		<-a.Session().Done()
	}
	a.Exit(code, nil)
}
