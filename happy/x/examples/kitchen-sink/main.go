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

// Package main
package main

import (
	"embed"
	"fmt"
	"reflect"
	"time"

	"github.com/mkungla/happy"

	"github.com/mkungla/happy/x/contrib/addons/servers/fileserver"
	"github.com/mkungla/happy/x/contrib/addons/servers/proxyserver"
	"github.com/mkungla/happy/x/contrib/commands"
	"github.com/mkungla/happy/x/contrib/flags"
	"github.com/mkungla/happy/x/contrib/logging/loggers/stdlog"

	"github.com/mkungla/happy/x/sdk"
	"github.com/mkungla/happy/x/sdk/application"
	"github.com/mkungla/happy/x/sdk/config"
	"github.com/mkungla/happy/x/sdk/engine"
	"github.com/mkungla/happy/x/sdk/monitor"
	"github.com/mkungla/happy/x/sdk/session"
)

//go:embed assets/*
var assets embed.FS

func main() {
	conf := config.New(
		// Option for configurator should have one of the following prefixes
		// "app."       application config options
		// "log."       logger options
		// "session."   default session options
		// "settings."  default settings options
		// "addon[<addon-slug>]." addon specific default options
		sdk.Option("app.slug", "kitchen-sink"),
		sdk.Option("app.title", "Happy SDK Kitchen Sink"),
		sdk.Option("log.level", happy.LOG_DEBUG),
	)

	logger := stdlog.New(
		sdk.Option("colors", true),
		// following marks option as readonly however behaviour still depends
		// in on underlying implementation do respect readonly values.
		// This is good example why you should use github.com/mkungla/happy/x/testsuite
		// to test your implementations, since that will test against such things.
		sdk.ReadOnlyOption("prefix", "happy"),
	)
	// Logger to be used
	conf.UseLogger(logger)

	// Session (context) manager to be used
	conf.UseSession(session.New())
	// Application monitor to be used
	conf.UseMonitor(monitor.New())
	// Asset FS to be used
	conf.UseAssets(assets)
	// App engine to be used
	e := engine.New(
		sdk.Option("service.discovery.timeout", time.Second*30),
	)

	// We reqister "remote" peer resolution
	// so that we can make service calls to that remote instance
	// (see Do function)
	e.ResolvePeerTo(
		"com.github.mkungla.happy.x.examples.kitchen-sink.microservice",
		"localhost:9508",
	)

	conf.UseEngine(e)

	// Create application instance
	app, err := application.New(conf)

	// Apply configuration to application
	if err != nil {
		logger.Error(err)
		return
	}

	// ORDER OF FOLLOWING SETUP CALS DOES NOT MATTER

	// Add single custom addon
	app.AddAddon(customAddon())

	app.AddAddons(
		// simple proxy server to serve all services
		// under one port
		proxyserver.New(
			sdk.ReadOnlyOption("port", 9500),
			sdk.ReadOnlyOption("fileloggerservice.level", happy.LOG_INFO),
		),

		// simple static file server
		fileserver.New(
			sdk.ReadOnlyOption("directory", "./public/files"),
			sdk.ReadOnlyOption("port", 9501),
			sdk.ReadOnlyOption("fileloggerservice.level", happy.LOG_INFO),
		),

		// simple webserver file server
		// see ./webserver-addon-setup.go
		webserverAddonSetup(
			sdk.ReadOnlyOption("hostname", "localhost"), // default
			sdk.ReadOnlyOption("public.dir", "./public/www"),
			sdk.ReadOnlyOption("template.dir", "./templates/web"),
			sdk.ReadOnlyOption("port", 9502),
			sdk.ReadOnlyOption("fileloggerservice.level", happy.LOG_INFO),
		),

		// simple rest api server
		// see ./restapi-addon-setup.go
		restapiAddonSetup(
			sdk.ReadOnlyOption("hostname", "localhost"), // default
			sdk.ReadOnlyOption("port", 9503),
			sdk.ReadOnlyOption("fileloggerservice.level", happy.LOG_INFO),
		),

		// simple websocket server
		// see ./websocket-addon-setup.go
		websocketAddonSetup(
			sdk.ReadOnlyOption("hostname", "localhost"), // default
			sdk.ReadOnlyOption("port", 9504),
			sdk.ReadOnlyOption("fileloggerservice.level", happy.LOG_INFO),
		),

		// simple eventsource server
		// see ./eventsource-addon-setup.go
		eventsourceAddonSetup(
			sdk.ReadOnlyOption("hostname", "localhost"), // default
			sdk.ReadOnlyOption("port", 9505),
			sdk.ReadOnlyOption("fileloggerservice.level", happy.LOG_INFO),
		),

		// simple rpc server
		// see ./rpcserver-addon-setup.go
		rpcserverAddonSetup(
			sdk.ReadOnlyOption("hostname", "localhost"), // default
			sdk.ReadOnlyOption("port", 9506),
			sdk.ReadOnlyOption("fileloggerservice.level", happy.LOG_INFO),
		),

		// simple docs server similar to
		// similar to https://cs.opensource.google/go/x/pkgsite
		// but serving your application and dependency documentation.
		// with go play feature if you have ExampleTests.
		// Addons can provide their own documentation
		// e.g. restapi serves testable OpenAPI documentation aswell.
		// see ./docs-addon-setup.go
		docsAddonSetup(
			sdk.ReadOnlyOption("hostname", "localhost"), // default
			sdk.ReadOnlyOption("port", 9507),
			sdk.ReadOnlyOption("fileloggerservice.level", happy.LOG_INFO),
		),

		// WebAssambly
		// This addon adds wasm support to your application.
		// see ./wasm-addon-setup.go
		wasmAddonSetup(
			// This addon provides command wasm-build. Option output.dir sets
			// destination directory. For serving that wasm We could use
			// ./public/ diretctory and serve it with fileserver or webserver addon,
			// however in this example we output wasm into ./assets directory which
			// means that it would be budled into our main application.
			sdk.ReadOnlyOption("output.dir", "./assets/wasm"),
			// This log level applies only for wasm addon commands
			sdk.ReadOnlyOption("fileloggerservice.level", happy.LOG_INFO),
		),
	)

	// Add some common flags provided by SDK
	app.AddFlag(flags.VersionFlag()) // --version (print app version)
	app.AddFlag(flags.XFlag())       // -x (prints commands as they are executed)
	app.AddFlag(flags.HelpFlag())    // -h, --help (help for app and comands)

	app.AddFlagFuncs(
		// Add common log verbosity flags provided by SDK
		// -v -verbose, --debug, --system-debug
		flags.LoggerFlags()...,
	)

	app.AddSubCommand(commands.BashCompletion()) // adds bash completion support
	app.AddSubCommand(commands.Env())            // prints application env invormation

	app.AddSubCommands(nil)

	// application
	app.AddService(customService())
	// mark that service to be required always
	// regardless of sub command
	app.RequireServices("happy://internal/services/customservice")

	// behaves like AddService but enables you to add services in one go
	// using slice of happy.ServiceCreateFunc's
	app.AddServices(
		// Example passive logging service
		// Other services can direct logs to this service.
		fileloggerserviceServiceSetup(
			sdk.ReadOnlyOption("directory", "./logs"),
			sdk.ReadOnlyOption("archive", "./logs/archive"),
			// For any service we can use option autostart so
			// that we do not need to call.
			// app.RequireServices("happy://internal/services/fileloggerservice")
			sdk.ReadOnlyOption("autostart", true),
			// rotate logs every 10 minutes
			sdk.ReadOnlyOption("rotate", time.Minute*10),
		),
	)

	// Before is called always async while application is starting up.
	app.Before(func(ctx happy.Session, args happy.Variables, assets happy.FS) error {
		ctx.Log().Experimentalf("[app]: Before args(%d) fs(%s)", args.Len(), reflect.TypeOf(assets).String())
		// We can call session.Ready(), if we want to do something while being sure
		// that all services have been started without any problems.
		// Otherwise it is ensured that session is ready before app or cmd .Do function is called.
		// WARNING:
		// When you use this option, you probably want to check ctx.Err
		// since session.Ready channel is closed on initialization error
		// which case you can get that error using session.Err()
		<-ctx.Ready()

		if ctx.Err() != nil {
			ctx.Log().Experimentalf("[app]: Before cancelled because of session had error %q", ctx.Err())
			return nil
		}
		ctx.Log().Experimental("[app]: Before session.Ready")
		return nil
	})

	// Do function only called when .Before returns without error
	// and session is ready.
	app.Do(func(ctx happy.Session, args happy.Variables, assets happy.FS) error {
		ctx.Log().Experimentalf("[app]: Do args(%d) fs(%s)", args.Len(), reflect.TypeOf(assets).String())

		// You can also require remote services
		// Schema is always "happy://" Engine should choose how to communicate with services.
		// Peer: is DNS entry we added earlier with engine.ResolvePeerTo
		// This is experimental api, but the intention is to have some sort
		// of peer discover system, but it is unsolved topic at the moment
		// what would be best approach to provide univeral API.
		schema := "happy://"
		peer := "com.github.mkungla.happy.x.examples.kitchen-sink.microservice"
		request := "/addon/apiserver/services/server-service"
		remoteservice := schema + peer + request

		// Start internal services
		// These services will initialize and start in parallel
		// This call blocks until all services are running.
		// Calling RequireServices multiple times has no effect
		// so if service is already running it will return immediately
		if err := ctx.RequireServices(
			"happy://internal/addon/fileserver/services?all",
			"happy://internal/addon/restapi/services?all",
			"happy://internal/addon/websocket/services?all",
			"happy://internal/addon/eventsource/services?all",
			"happy://internal/addon/rpcserver/services?all",
			remoteservice,
		); err != nil {
			return err
		}

		// Just for example we start these services after previous
		// set of services are running.
		if err := ctx.RequireServices(
			"happy://internal/addon/webserver/services?all",
			"happy://internal/addon/docs/services?all",
		); err != nil {
			return err
		}

		// And lets start our proxy server last
		if err := ctx.RequireServices(
			"happy://internal/addon/proxyserver/services?all",
		); err != nil {
			return err
		}

		// addon.webserver.url is set by webserver Addon
		fmt.Println("URL: ", ctx.GetOptionOrDefault("addon.proxyserver.url", "failed to get web app url, see logs for errors"))
		fmt.Println("PRESS ctrl+c to shutdown.")

		// lets block until you press ctrl+c
		<-ctx.Done()
		return nil
	})

	// AfterSuccess is called only after .Do function returns without error
	app.AfterSuccess(func(ctx happy.Session) error {
		ctx.Log().Experimental("[app]: AfterSuccess")
		return nil
	})

	// AfterFailure is called only after .Do function returns with error
	// This action will recieve that error as second arg
	// so that you can do something based on this error.
	app.AfterFailure(func(ctx happy.Session, err happy.Error) error {
		ctx.Log().Experimentalf("[app]: AfterFailure err(%s)", err)
		return nil
	})

	// AfterAlways is called always after .Do function returns
	// This action will recieve that error as second arg
	// so that you can do something based on this error.
	app.AfterAlways(func(ctx happy.Session, err happy.Error) error {
		ctx.Log().Experimentalf("[app]: AfterAlways err(%s)", err)
		return nil
	})

	// Add Cron jobs to your application
	app.Cron(func(cs happy.CronScheduler) {
		// Define Job with time duration
		cs.Job(time.Second*5, func(ctx happy.Session, err happy.Error) error {
			ctx.Log().Experimentalf("[app]: Cronjobs.Job in every 5 seconds err(%s)", err)
			return nil
		})

		// Define job with Crontab expression
		cs.Job("*/10 * * * * *", func(ctx happy.Session, err happy.Error) error {
			ctx.Log().Experimentalf("[app]: Cronjobs.Job in every 10 seconds err(%s)", err)
			return nil
		})
	})

	// OnTick is like render loop
	app.OnTick(func(ctx happy.Session, ts time.Time, delta time.Duration) error {
		ctx.Log().Experimentalf("[app]: OnTick ts(%s) delta(%s)", ts, delta)
		time.Sleep(time.Second * 15) // this affects also OnTock
		return nil
	})

	// OnTock is called right after previous tick returns so that you can separate
	// post proccessing logic if needed
	app.OnTock(func(ctx happy.Session, ts time.Time, delta time.Duration) error {
		ctx.Log().Experimentalf("[app]: OnTock ts(%s) delta(%s)", ts, delta)
		return nil
	})

	// Main MUST be always called last to start the app.
	// Behaviour of Main depends on platform
	app.Main()
}
