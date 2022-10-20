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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/pkg/peeraddr"
	"github.com/mkungla/happy/x/pkg/version"
	"github.com/mkungla/happy/x/session"
)

func (a *APP) setup() happy.Error {
	defer func() {
		dur := time.Since(a.initialized)
		a.logger.SystemDebugf(
			"initialization of %s %s took: %s",
			a.opts.Get("slug"),
			a.opts.Get("version"),
			dur,
		)
	}()

	// Verify command chain
	// Fail fast if command or one of the sub commands has errors
	if err := a.rootCmd.Verify(); err != nil {
		return err
	}

	if e := a.rootCmd.Flags().Parse(os.Args); e != nil {
		return ErrApplication.Wrap(e)
	}

	// change log level after user potenially added own logger
	if a.rootCmd.Flag("verbose").Present() {
		a.Log().SetPriority(happy.LOG_INFO)
		a.session.Store("flags.verbose", true)
	} else if a.rootCmd.Flag("debug").Present() {
		a.Log().SetPriority(happy.LOG_DEBUG)
		a.session.Store("flags.debug", true)
	} else if a.rootCmd.Flag("system-debug").Present() {
		a.session.Store("flags.system-debug", true)
		a.Log().SetPriority(happy.LOG_SYSTEMDEBUG)
	}
	a.Log().LogInitialization() // logs init log entires if needed

	if a.rootCmd.Flag("version").Present() {
		a.printVersion()
		a.Exit(0)
	}

	if a.rootCmd.Flag("x").Present() {
		a.session.Store("flags.x", true)
	}

	if err := a.loadHostInfo(); err != nil {
		return err
	}

	a.opts.Range(func(v happy.Variable) bool {
		a.session.Store("app."+v.Key(), v.Value())
		return true
	})

	if err := a.setActiveCommand(); err != nil {
		return err
	}

	return nil
}

func (a *APP) setActiveCommand() happy.Error {
	settree := a.rootCmd.Flags().GetActiveSets()
	name := settree[len(settree)-1].Name()

	if name == "/" {
		a.activeCmd = a.rootCmd
		return nil
	}

	var (
		activeCmd happy.Command
		exists    bool
	)

	// skip root cmd
	for _, set := range settree[1:] {
		slug := set.Name()
		if activeCmd == nil {
			activeCmd, exists = a.rootCmd.SubCommand(slug)
			if !exists {
				return ErrApplication.WithTextf("unknown command: %s", slug)
			}
			continue
		}
		activeCmd, exists = activeCmd.SubCommand(set.Name())
		if !exists {
			return ErrApplication.WithTextf("unknown subcommand: %s for %s", slug, activeCmd.Slug())
		}
		break
	}

	a.activeCmd = activeCmd

	// only set app tick tock if current command is root command
	if a.rootCmd == a.activeCmd {
		if a.tickAction != nil {
			a.engine.OnTick(a.tickAction)
		}
		if a.tockAction != nil {
			a.engine.OnTock(a.tockAction)
		}
	}
	return nil
}

func (a *APP) exit(code int) {
	a.logger.SystemDebug("shutting down")

	if err := a.engine.Stop(a.session); err != nil {
		a.Log().Error(err)
	}

	if err := a.engine.Monitor().Stop(); err != nil {
		a.Log().Error(err)
	}

	// Destroy session
	a.session.Destroy(nil)

	if err := a.session.Err(); err != nil && !errors.Is(err, session.ErrSession) {
		a.logger.Error(err)
	}

	if a.opts.Get("runtime.ensure.paths").Bool() {
		tmdir := a.opts.Get("path.tmp")
		if err := os.RemoveAll(tmdir.String()); err != nil {
			a.logger.Warn(err)
		}
	}

	a.logger.SystemDebugf("uptime: %s", a.engine.Monitor().Status().Elapsed())
	os.Exit(code)
}

func (a *APP) loadModuleInfo() error {
	var (
		ver string
		err error
	)
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		ver = fmt.Sprintf("0.0.1-devel+%d", time.Now().UnixMilli())
	} else {

		// version
		if bi.Main.Version == "(devel)" {
			fmt.Println("bi.Main.Version is ", bi.Main)
			slug, ok := a.opts.LoadOrDefault("slug", "happy-app")
			if !ok {
				return errors.New("APP.loadModuleInfo slug not set")
			}
			moduleVersion := strings.Trim(filepath.Ext(slug.String()), ".")
			major := "v1"
			if strings.HasPrefix(moduleVersion, "v") {
				majorint, err := strconv.Atoi(strings.TrimPrefix(moduleVersion, "v"))
				if err != nil {
					return err
				}
				major = fmt.Sprintf("v%d", majorint+1)
			}
			ver = fmt.Sprintf("%s.0.0-alpha.%d", major, time.Now().UnixMilli())
		} else {
			ver = bi.Main.Version
		}
	}

	version, err := version.Parse(ver)
	if err != nil {
		return err
	}
	a.opts.Store("version", version)

	if !a.opts.Has("peer.addr") {
		a.opts.Store("peer.addr", peeraddr.Current())
	}

	a.version = version
	return nil
}

func (a *APP) printVersion() {
	fmt.Println(a.opts.Get("version"))
}

func (a *APP) loadHostInfo() happy.Error {

	// env, err := vars.ParseMapFromSlice(os.Environ())
	// if err != nil {
	// 	return ErrApplication.Wrap(err)
	// }
	// env.Range(func(v vars.Variable) bool {
	// 	a.session.Store("env."+v.Key(), v.String())
	// 	return true
	// })

	wd, err := os.Getwd()
	if err != nil {
		return ErrApplication.Wrap(err)
	}
	a.opts.Store("path.wd", wd)

	hostname, err := os.Hostname()
	if err != nil {
		return ErrApplication.Wrap(err)
	}
	a.opts.Store("hostname", hostname)

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return ErrApplication.Wrap(err)
	}
	a.opts.Store("path.home", userHomeDir)

	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", a.Slug().String(), time.Now().UnixMilli()))
	a.opts.Store("path.tmp", tempDir)

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return ErrApplication.Wrap(err)
	}
	a.opts.Store("path.cache", filepath.Join(userCacheDir, a.Slug().String()))

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return ErrApplication.Wrap(err)
	}
	a.opts.Store("path.config", filepath.Join(userConfigDir, a.Slug().String()))

	return nil
}

func appmain() {
	select {}
}

// executed in go routine
func (a *APP) execute() {

	if err := a.session.Start(); err != nil {
		a.logger.Emergency(err)
		a.Exit(1)
		return
	}

	if a.opts.Get("runtime.ensure.paths").Bool() {
		dirs := a.opts.ExtractWithPrefix("path.")
		dirs.Range(func(v happy.Variable) bool {
			if v.Key() == "home" {
				return true
			}
			if err := os.MkdirAll(v.String(), 0750); err != nil {
				a.logger.Error(err)
			}
			a.session.Log().SystemDebugf("create dir: %s", v.String())
			return true
		})
	}

	if err := a.engine.Start(a.session); err != nil {
		a.Log().Emergency(err)
		a.Exit(1)
		return
	}

	// execute before action chain
	if err := a.executeBeforeActions(); err != nil {
		a.Log().Alert(err)
		a.Exit(1)
		return
	}

	// block until session is ready
	a.logger.SystemDebug("waiting session...")
	<-a.session.Ready()
	if a.session.Err() != nil {
		a.Exit(1)
		return
	}

	cmdtree := strings.Join(a.activeCmd.Parents(), ".") + "." + a.activeCmd.Slug().String()
	a.logger.SystemDebugf("session ready: execute %s.Do action", cmdtree)
	err := a.activeCmd.ExecuteDoAction(a.session, a.assets, a.engine.Monitor().Status(), a.apis)
	if err != nil {
		a.executeAfterFailureActions(err)
	} else {
		a.executeAfterSuccessActions()
	}
	a.executeAfterAlwaysActions(err)
}

func (a *APP) executeBeforeActions() happy.Error {
	a.logger.SystemDebug("execute before actions")
	if a.rootCmd != a.activeCmd {
		if err := a.rootCmd.ExecuteBeforeAction(a.session, a.assets, a.engine.Monitor().Status(), a.apis); err != nil {
			return err
		}
	}
	if err := a.activeCmd.ExecuteBeforeAction(a.session, a.assets, a.engine.Monitor().Status(), a.apis); err != nil {
		return err
	}
	return nil
}

func (a *APP) executeAfterFailureActions(err happy.Error) {
	a.logger.SystemDebug("execute after failure actions")
	a.logger.Error(err)

	if err := a.activeCmd.ExecuteAfterFailureAction(a.session, err); err != nil {
		a.logger.Error(err)
	}

	if a.rootCmd != a.activeCmd {
		if err := a.rootCmd.ExecuteAfterFailureAction(a.session, err); err != nil {
			a.logger.Error(err)
		}
	}
}

func (a *APP) executeAfterSuccessActions() {
	a.logger.SystemDebug("execute after success actions")
	if err := a.activeCmd.ExecuteAfterSuccessAction(a.session); err != nil {
		a.logger.Error(err)
	}

	if a.rootCmd != a.activeCmd {
		if err := a.rootCmd.ExecuteAfterSuccessAction(a.session); err != nil {
			a.logger.Error(err)
		}
	}
}

func (a *APP) executeAfterAlwaysActions(err happy.Error) {
	a.logger.SystemDebug("execute after always actions")

	if err := a.activeCmd.ExecuteAfterAlwaysAction(a.session, err); err != nil {
		a.logger.Error(err)
	}

	if a.rootCmd != a.activeCmd {
		if err := a.rootCmd.ExecuteAfterAlwaysAction(a.session, err); err != nil {
			a.logger.Error(err)
		}
	}

	if err != nil {
		a.Exit(1)
	} else {
		a.Exit(0)
	}
}
