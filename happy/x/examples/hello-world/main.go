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

package main

import (
	"errors"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/contrib/flags"
	"github.com/mkungla/happy/x/examples/hello-world/hello"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/sdk"
)

func main() {
	conf := sdk.NewConfig(
		happyx.Option("app.title", "Hello World"),
	)
	app := sdk.NewApplication(conf)

	// app.AddFlag(flags.VersionFlag()) // --version (print app version)
	// app.AddFlag(flags.XFlag())       // -x (prints commands as they are executed)
	app.AddFlag(flags.HelpFlag()) // -h, --help (help for app and comands)
	app.AddFlags(
		// Add common log verbosity flags provided by SDK
		// -v -verbose, --debug, --system-debug
		flags.LoggerFlags()...,
	)

	app.RegisterAddon(hello.New())

	app.Before(func(sess happy.Session, f happy.Flags, assets happy.FS, status happy.ApplicationStatus, apis []happy.API) error {
		sess.Log().Notice("app.before")

		loader := sess.RequireServices(
			status,
			"/hello-service",
		)

		sess.Log().Notice("app.before service loading")
		<-loader.Loaded()
		sess.Log().Notice("app.before service loaded")

		// <-sess.Ready()
		// auth

		return nil
	})

	app.Do(func(sess happy.Session, f happy.Flags, assets happy.FS, status happy.ApplicationStatus, apis []happy.API) error {
		// say hello in background
		time.Sleep(time.Second * 2)

		sess.Dispatch(happyx.NewEvent("say", "hello.world", nil, nil))
		time.Sleep(time.Second * 2)
		<-sess.Done()
		return errors.New("before")
	})

	app.AfterFailure(func(sess happy.Session, err happy.Error) error {
		sess.Log().Notice("AfterFailure")
		return nil
	})

	app.AfterSuccess(func(sess happy.Session) error {
		sess.Log().Notice("AfterSuccess")
		return nil
	})

	app.AfterAlways(func(sess happy.Session, status happy.ApplicationStatus) error {
		sess.Log().Noticef("AfterAlways: elapsed %s", status.Elapsed())
		return nil
	})

	app.Main()
}
