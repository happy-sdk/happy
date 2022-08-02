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
	"fmt"
	"os"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/cli"
	"github.com/mkungla/happy/internal/jsonlog"
	"github.com/mkungla/varflag/v6"
)

// AddFlag to application. Invalid flag will add error to multierror and
// prevents application to stacli.
func (a *Application) AddFlag(f varflag.Flag) {
	if f == nil {
		a.addAppErr(fmt.Errorf("%w: %s", cli.ErrCommandFlags, "adding <nil> flag"))
		return
	}

	// assign the flag
	a.flags.Add(f)
}

// initialize flags and add builtin flags.
func (a *Application) initFlags() bool {
	// Global flags are always allowed and can be used to configure
	// background jobs or frontend behaviour
	var err error
	if a.flags, err = varflag.NewFlagSet(os.Args[0], 0); err != nil {
		a.addAppErr(err)
		return false
	}

	bashCompletion, err := varflag.Bool("show-bash-completion", false, "")
	if err == nil {
		bashCompletion.Hide()
		a.AddFlag(bashCompletion)
	}
	a.addAppErr(err)

	systemDebug, err := varflag.Bool(
		"system-debug",
		false,
		"enable system debug log level (very verbose)",
	)
	a.addAppErr(err)

	if err == nil {
		a.AddFlag(systemDebug)
	}

	debug, err := varflag.Bool(
		"debug",
		false,
		"enable debug log level. when debug flag is after the command then debugging will be enabled only for that command",
	)
	a.addAppErr(err)

	if err == nil {
		a.AddFlag(debug)
	}

	verbose, err := varflag.Bool(
		"verbose",
		false,
		"enable verbose log level",
		"v",
	)
	a.addAppErr(err)
	if err == nil {
		a.AddFlag(verbose)
	}

	help, err := varflag.Bool(
		"help",
		false,
		"display help or help for the command. [...command --help]",
		"h",
	)
	a.addAppErr(err)
	if err == nil {
		a.AddFlag(help)
	}

	version, err := varflag.Bool(
		"version",
		false,
		"display application version",
	)
	a.addAppErr(err)
	if err == nil {
		a.AddFlag(version)
	}

	bgka, err := varflag.Bool(
		"services-keep-alive",
		false,
		"if services are defined then ones with 'KeepAlive() true' alive when UI exits with 0 status until os interrupt or other kill signal is received.", //nolint:lll
	)
	a.addAppErr(err)
	if err == nil {
		a.AddFlag(bgka)
	}

	jsonflag, err := varflag.Option(
		"json",
		[]string{"compact"},
		"if json flag is set app will not log anything to sdtout and sdterr, It will exit on first call to ctx.JSON() or with auto response", //nolint:lll
		[]string{"compact", "pretty"},
	)
	a.addAppErr(err)
	if err == nil {
		a.AddFlag(jsonflag)
	}

	x, err := varflag.Bool(
		"x",
		false,
		"The -x flag prints all the external commands as they are executed by the Application.Exec",
	)
	if err == nil {
		a.AddFlag(x)
	}
	a.addAppErr(err)

	// we can ignore err??
	a.addAppErr(a.flags.Parse(os.Args))
	if a.Flag("json").Present() {
		a.logger = jsonlog.New(happy.LevelTask)
	}

	return a.errors.Len() == 0
}
