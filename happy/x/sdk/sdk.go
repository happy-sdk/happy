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

package sdk

import (
	"os"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/addon"
	"github.com/mkungla/happy/x/application"
	"github.com/mkungla/happy/x/cli"
	"github.com/mkungla/happy/x/config"
	"github.com/mkungla/happy/x/contrib/loggers/console"
	"github.com/mkungla/happy/x/engine"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/monitor"
	"github.com/mkungla/happy/x/pkg/varflag"
	"github.com/mkungla/happy/x/pkg/version"
	"github.com/mkungla/happy/x/session"
)

var (
	ErrAddon = addon.ErrAddon
	ErrFlag  = happyx.NewError("flag error")
)

type Slug struct {
	s string
}

func (s Slug) Valid() bool {
	return false
}

func (s Slug) String() string {
	return s.s
}

func NewAddon(slug, name, ver string, defaultOptions ...happy.OptionSetFunc) (happy.Addon, happy.Error) {
	v, err := version.Parse(ver)
	if err != nil {
		return nil, ErrAddon.Wrap(err)
	}

	return addon.New(slug, name, v, defaultOptions...)
}

func NewConfig(opts ...happy.OptionSetFunc) happy.Configurator {
	conf, err := config.New(opts...)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
		return nil
	}

	// default logger with default options
	logger := console.New(
		os.Stderr,
		happyx.Option("level", happy.LOG_NOTICE),
		happyx.Option("colors", true),
		happyx.Option("scope", ""),
		// following marks option as readonly however behaviour still depends
		// in on underlying implementation do respect readonly values.
		// This is good example why you should use github.com/mkungla/happy/x/testsuite
		// to test your implementations, since that will test against such things.
		happyx.ReadOnlyOption("filenames.level", happy.LOG_NOTICE),
		happyx.ReadOnlyOption("filenames.long", false),
		happyx.ReadOnlyOption("filenames.pre", ""),
		happyx.ReadOnlyOption("ts.date", false),
		happyx.ReadOnlyOption("ts.time", true),
		happyx.ReadOnlyOption("ts.microseconds", false),
		happyx.ReadOnlyOption("ts.utc", false), // utc true will show UTC time instead
	)
	logger.SystemDebug("attached default logger")
	// Logger to be used
	conf.UseLogger(logger)

	// Session (context) manager to be used
	conf.UseSession(NewSession(logger))

	// Application monitor to be used
	conf.UseMonitor(NewMonitor())

	// App engine to be used
	e := NewEngine(
		happyx.Option("services.discovery.timeout", time.Second*30),
	)

	conf.UseEngine(e)
	return conf
}

func NewApplication(conf happy.Configurator) happy.Application {
	app, err := application.New(conf)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	return app
}

func NewSession(logger happy.Logger, opts ...happy.OptionSetFunc) happy.Session {
	return session.New(logger, opts...)
}

func NewEngine(opts ...happy.OptionSetFunc) happy.Engine {
	return engine.New(opts...)
}

func NewMonitor(opts ...happy.OptionSetFunc) happy.Monitor {
	return monitor.New(opts...)
}

func NewCommand(cmd string, opts ...happy.OptionSetFunc) (happy.Command, happy.Error) {
	return cli.NewCommand(cmd, opts...)
}

func NewStringFlag(name string, value string, usage string, aliases ...string) (happy.Flag, happy.Error) {
	f, err := varflag.New(name, value, usage, aliases...)
	if err != nil {
		return nil, ErrFlag.Wrap(err)
	}
	return varflag.AsFlag[happy.Flag, happy.Variable, happy.Value](f), nil
}
