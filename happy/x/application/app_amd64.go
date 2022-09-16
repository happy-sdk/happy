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
	"fmt"
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/pkg/varflag"
	"os"
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

	if a.flag("version").Present() {
		fmt.Println("version")
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
