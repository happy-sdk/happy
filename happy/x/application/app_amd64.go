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
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/pkg/varflag"
	"github.com/mkungla/happy/x/pkg/version"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

func (a *APP) setup() happy.Error {
	err := a.monitor.Start()
	if err != nil {
		return err
	}

	if err := a.session.Start(); err != nil {
		return err
	}

	if e := a.flags.Parse(os.Args); e != nil {
		return ErrApplication.Wrap(e)
	}

	// change log level after user potenially added own logger
	if a.flag("verbose").Present() {
		a.Log().SetPriority(happy.LOG_INFO)
	} else if a.flag("debug").Present() {
		a.Log().SetPriority(happy.LOG_DEBUG)
	} else if a.flag("system-debug").Present() {
		a.Log().SetPriority(happy.LOG_SYSTEMDEBUG)
	}
	a.Log().LogInitialization() // logs init log entires if needed

	if err := a.loadModuleInfo(); err != nil {
		return ErrApplication.Wrap(err)
	}
	if a.flag("version").Present() {
		a.printVersion()
		a.Exit(0)
	}

	dur := time.Since(a.initialized)

	a.logger.SystemDebug("initialization took: ", dur)
	return nil
}

func (a *APP) flag(name string) happy.Flag {
	f, err := a.flags.Get(name)
	if err != nil {
		a.Log().Error(err)
		vf, err := varflag.Bool(name, false, "")
		if err != nil {
			a.Log().Error(err)
		}
		f = varflag.AsFlag[happy.Flag, happy.Variable, happy.Value](vf)
	}
	if f == nil {
		a.logger.Emergency("app returned")
	}
	return f
}

func (a *APP) exit(code int) {
	a.logger.SystemDebug("shutting down")

	if err := a.engine.Stop(); err != nil {
		a.Log().Error(err)
	}

	// Destroy session
	a.session.Destroy(nil)

	// Stop monitor
	if err := a.monitor.Stop(); err != nil {
		a.Log().Error(err)
	}
	a.logger.SystemDebugf("uptime: %s", a.monitor.Stats().Elapsed())
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
		a.opts.Store("go.version", bi.GoVersion)
		a.opts.Store("go.path", bi.Path)

		// The module containing the main package
		a.opts.Store("go.module.path", bi.Main.Path)
		a.opts.Store("go.module.version", bi.Main.Version)
		a.opts.Store("go.module.sum", bi.Main.Sum)

		if bi.Main.Replace != nil {
			a.opts.Store("go.module.replace.path", bi.Main.Replace.Path)
			a.opts.Store("go.module.replace.version", bi.Main.Replace.Version)
			a.opts.Store("go.module.replace.sum", bi.Main.Replace.Sum)
		} else {
			a.opts.Store("go.module.replace", nil)
		}
		if bi.Deps != nil {
			for _, dep := range bi.Deps {
				key := "go.module.deps.[" + dep.Path + "]"
				a.opts.Store(key+".path", dep.Path)
				a.opts.Store(key+".version", dep.Version)
				a.opts.Store(key+".sum", dep.Sum)
			}
		} else {
			a.opts.Store("go.module.deps", nil)
		}

		if bi.Settings != nil {
			for _, setting := range bi.Settings {
				a.opts.Store(fmt.Sprintf("go.module.settings.%s", setting.Key), setting.Value)
			}
		} else {
			a.opts.Store("go.module.settings", nil)
		}

		// version
		if bi.Main.Version == "(devel)" {
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

	a.version, err = version.Parse(ver)
	if err != nil {
		return err
	}

	return nil
}

func (a *APP) printVersion() {
	fmt.Println(a.version)
}
