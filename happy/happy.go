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

// Package happy is the ultimate Go prototyping SDK. This makes developers
// happy by giving them a great SDK to solve any domain problem and quickly
// create a working prototype (maybe even MVP). It's a tool for hackers and
// creators to realize their ideas when a software architect is not at hand
// or technical knowledge about infrastructure planning is minimal.
package happy

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"time"
	// "bytes"
	// "regexp"
	// "runtime"
	// "runtime/debug"
	// "sort"
	// "strings"
	// "time"
	// "unicode"
	// "github.com/mkungla/happy/pkg/varflag"
	// "github.com/mkungla/happy/pkg/vars"
)

const (
	// Same as log/syslog and here since
	// log/syslog pkg is not available
	// on windows, plan9, js
	LOG_EMERG LogPriority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
	LOG_SYSTEMDEBUG
)

///////////////////////////////////////////////////////////////////////////////
// COMMON
///////////////////////////////////////////////////////////////////////////////

type (
	LogPriority int

	// Action is common callback in happy framework.
	ActionFunc func(ctx Session) error

	// ActionWithArgs is common callback in happy framework which has
	// privides arguments as vars.Collection.
	ActionWithArgsFunc  func(ctx Session, args Variables) error
	ActionWithEventFunc func(ctx Session, ev Event) error

	ActionWithArgsAndAssetsFunc func(ctx Session, args Variables, assets FS) error

	// ActionWithError is common callback in happy framework which has
	// second arguments as previous error id any otherwise nil.
	ActionWithErrorFunc func(ctx Session, err Error) error

	// TickTock is operation set in given minimal time frame it can be executed.
	// You can throttle tick/tocks to cap FPS or for [C|G]PU throttling.
	//
	// Tock is helper called after each tick to separate
	// logic processed in tick and do post processing on tick.
	// Tocks are useful mostly for GPU ops which need to do post proccessing
	// of frames rendered in tick.
	ActionTickFunc func(ctx Session, ts time.Time, delta time.Duration) error

	// Event is consumable by local instance of application components.
	Event interface {
		// Time is UTC time when event was created.
		Time() time.Time

		// Scope is event scope it is string matching always.
		// ^[a-z0-9][a-z0-9.-]*[a-z0-9]$
		Scope() string
		// Key is event identifier it is string matching always.
		// ^[a-z0-9][a-z0-9.-]*[a-z0-9]$
		Key() string

		// Payload returns of the event
		Payload() Variables
	}

	ActionStats interface {
		Started() time.Time
		Elapsed() time.Duration
		Dispose()
	}

	Logger interface {
		io.WriteCloser
		GetPriority() LogPriority
		SetPriority(LogPriority)
		RuntimePriority(LogPriority)
		ResetPriority(LogPriority)

		// LOG_EMERG
		Emergency(args ...any)
		Emergencyf(template string, args ...any)

		// LOG_ALERT
		Alert(args ...any)
		Alertf(template string, args ...any)

		// LOG_CRIT
		Critical(args ...any)
		Criticalf(template string, args ...any)
		BUG(args ...any)
		BUGF(template string, args ...any)

		// LOG_ERR
		Error(args ...any)
		Errorf(template string, args ...any)

		// LOG_WARNING
		Warn(args ...any)
		Warnf(template string, args ...any)
		Deprecated(args ...any)
		Deprecatedf(template string, args ...any)

		// LOG_NOTICE
		Notice(args ...any)
		Noticef(template string, args ...any)
		Ok(args ...any)
		Okf(template string, args ...any)
		Issue(nr int, args ...any)
		Issuef(nr int, template string, args ...any)

		// LOG_INFO
		Info(args ...any)
		Infof(template string, args ...any)
		Experimental(args ...any)
		Experimentalf(template string, args ...any)
		NotImplemented(args ...any)
		NotImplementedf(template string, args ...any)

		// LOG_DEBUG
		Debug(args ...any)
		Debugf(template string, args ...any)
		SystemDebug(args ...any)
		SystemDebugf(template string, args ...any)
	}

	Error interface {
		error
		App() Slug
		Code() int
		Time() time.Time
	}

	Slug interface {
		fmt.Stringer
		Valid() bool
	}

	URL interface {
		fmt.Stringer
	}

	Version interface {
		fmt.Stringer
	}

	Cron interface {
		Cron(func(CronScheduler))
	}

	CronHandler interface {
		Job() ActionWithErrorFunc
		Expr() string
	}

	CronScheduler interface {
		Job(expr any, cb ActionWithErrorFunc)
	}

	// FS could be e.g. embed.FS
	FS interface {
		fs.FS
		ReadDir(name string) ([]fs.DirEntry, error)
		ReadFile(name string) ([]byte, error)
	}

	TickTocker interface {
		// OnTick enables you to define func body for operation set
		// to call in minimal timeframe until session is valid and
		// service is running.
		OnTick(ActionTickFunc)

		// OnTock is helper called right after OnTick to separate
		// your primary operations and post prossesing logic.
		OnTock(ActionTickFunc)
	}
)

///////////////////////////////////////////////////////////////////////////////
// OPTIONS
///////////////////////////////////////////////////////////////////////////////

type (
	// OptionWriteFunc is callback function to apply configurable options.
	OptionWriteFunc func(opts OptionWriter) Error

	// Options interface enables you to apply configuration options in
	// different parts of your application.
	Options interface {
		// OptionReadCloser's embedded interface allows you to control how your
		// implementation makes configuration options readable. For example you
		// can implement your own middleware for option getter.
		OptionReader

		// OptionWriteCloser's embedded interface allows you to control how your
		// implementation makes changes to configuration options. For example you
		// can implement your own middleware for option setter.
		OptionWriter

		// DeleteOption allows you remove option value and it's key from
		// your option set. return ErrOption when key dows not exist or
		// ErrOptionRO if key is read only.
		DeleteOption(key string) Error

		// ResetOptions clears all options and restores defaults
		// if defaults have been set
		ResetOptions() Error
	}

	// OptionGetter is read only representation of Options
	OptionReader interface {
		io.Reader

		// GetOption return option by key or ErrOption,
		// if non existingOption was requested.
		GetOption(key string) (Variable, Error)

		// GetOptionOrDefault looks session value and returns
		// provided default value if key does not exists.
		// This call does not set that default to session!
		GetOptionOrDefault(key string, defval ...any) (val Variable)

		// HasOption reports whether option set has option with key.
		HasOption(key string) bool

		// Range calls f sequentially for each key -> value present in the options.
		// If f returns false, range stops the iteration.
		RangeOptions(f func(key string, value Value) bool)

		// Get all options as vars.Collection
		GetAllOptions() Variables
	}

	OptionWriter interface {
		io.Writer
		SetOption(Variable) Error
		SetOptionKeyValue(key string, val any) Error
		SetOptionValue(key string, val Value) Error
	}

	OptionDefaultsWriter interface {
		SetOptionDefault(Variable) Error
		SetOptionDefaultKeyValue(key string, val any) Error
		SetOptionsDefaultFuncs(vfuncs ...VariableParseFunc) Error
	}
)

type (
	VariableParseFunc func() (Variable, Error)

	ValueParseFunc func() (Value, Error)

	// Variables are collection of key=value pairs
	Variables interface {
		Len() int
	}

	// Variable is key=value pair
	Variable interface {
		fmt.Stringer // alias to Value().String()
		Value() Value
		ReadOnly() bool
	}

	Value interface {
		fmt.Stringer
	}
)

///////////////////////////////////////////////////////////////////////////////
// APPLICATION
///////////////////////////////////////////////////////////////////////////////

type (
	Configurator interface {
		OptionDefaultsWriter // Sets application configuration options

		UseLogger(logger Logger)
		GetLogger(logger Logger) (Logger, Error)

		UseSession(session Session)
		GetSession(session Session) (Session, Error)

		UseMonitor(monitor ApplicationMonitor)
		GetMonitor(ApplicationMonitor) (ApplicationMonitor, Error)

		UseAssets(fs.FS)
		GetAssets(fs.FS) (fs.FS, Error)

		UseEngine(Engine)
		GetEngine(Engine) (Engine, Error)
	}

	// Application is interface for application runtime.
	Application interface {
		Configure(Configurator) Error

		AddAddon(Addon)
		AddAddons(...AddonCreateFunc)

		AddService(Service)
		AddServices(...ServiceCreateFunc)

		Log() Logger

		Command
		Cron
		TickTocker

		Main()
	}

	Session interface {
		// Done() behaves like context.Done and indcates that session was destroyed
		// Err() returns last error in session
		context.Context

		Options

		// Ready blocks until the session is ready to use.
		Ready() <-chan struct{}

		// Destroy session
		Destroy(err Error)

		// Log returns application logger.
		Log() Logger

		Settings() Settings

		Dispatch(Event)

		// simple context you can use as parent context
		// instead of passing session it self
		// even tho it implements context aswell.
		Context() context.Context

		RequireServices(svcs ...string) Error
	}

	Settings interface {
		Options
		OptionDefaultsWriter

		// Load loads settings from underlying storage driver.
		Load() Error

		// Save saves settings using underlying storage driver.
		Save() Error
	}

	ApplicationMonitor interface {
		Start() Error
		Stop() Error
		OnEvent(Event)
		Stats() ActionStats
	}
)

///////////////////////////////////////////////////////////////////////////////
// CLI
///////////////////////////////////////////////////////////////////////////////

type (
	CommandCreateFunc func() (Command, Error)

	CommandCreateFlagFunc func() (CommandFlag, Error)

	Command interface {
		// String returns command name.
		fmt.Stringer
		// RequireServices ensures that services by their URL
		// will be started before Command.Main call. These services will be started
		// in parallel with Command.Init. If you need to have services ready
		// inside Command.Before then you can call ctx.Ready which will block
		// until all services are running.
		RequireServices(svcs ...string)

		AddFlag(CommandFlag)
		AddFlagFuncs(flags ...CommandCreateFlagFunc)

		AddSubCommand(Command)
		AddSubCommands(...CommandCreateFunc)

		// Before is optional action to execute before Main.
		// Useful to check preconidtion for your command Main.
		// Since any required services by command will be initialized in parallel
		// with this action you can call ctx.Ready if you need to use services
		// inside the Init.
		Before(ActionWithArgsAndAssetsFunc)

		// Main is action where your commands main logic should live.
		// Main function body is required unless it is wrapper (parent) command
		// for it's subcommand you want to have your application logic.
		Do(ActionWithArgsAndAssetsFunc)

		// AfterSuccess is optional callback called when Main returns without error.
		AfterSuccess(action ActionFunc)

		// AfterFailure is optional callback when Main returns error.
		// Second argument will be error returned.
		AfterFailure(action ActionWithErrorFunc)

		// AfterAlways is optional callback to execute reqardless did Main
		// succeed or not.
		AfterAlways(action ActionWithErrorFunc)
	}

	CommandFlag interface{}
)

///////////////////////////////////////////////////////////////////////////////
// ADDON
///////////////////////////////////////////////////////////////////////////////

type (
	// AddonCreateFunc implementation is function used to register
	// addons to your application.
	AddonCreateFunc func() (Addon, Error)
	// Addon enables you to bundle set of commands, services and other features
	// into single package and make it sharable accross applications and domains.
	Addon interface {
		Cronjobs()
		Name() string
		Slug() Slug
		Version() Version
		Options() OptionDefaultsWriter
		Commands(...Command)
		Services(...Service)
	}

	AddonFactory interface {
		Addon() (Addon, Error)
	}
)

///////////////////////////////////////////////////////////////////////////////
// SERVICE
///////////////////////////////////////////////////////////////////////////////

type (
	ServiceCreateFunc func() (Service, Error)
	// ServiceHandler is callback to be called when Service recieves a Request.
	ServiceHandler func(ctx Session, req ServiceRequest) ServiceResponse

	// Service interface
	Service interface {
		TickTocker

		Slug() Slug

		// OnInitialize is called when app is preparing runtime and attaching
		// services.
		OnInitialize(ActionFunc)

		// OnStart is called when service is requested to be started.
		// For instace when command is requiring this service or whenever
		// service is required on runtime via ctx.RequireService call.
		//
		// Start can be called multiple times in case of service restarts.
		// If you do not want to allow service restarts you should implement
		// your logic in OnStop when it's called first time and check that
		// state OnStart.
		OnStart(ActionWithArgsFunc)

		// OnStop is called once application exits, session is destroyed or
		// when service restart is requested.
		OnStop(ActionFunc)

		// OnEvent is simple local event system
		OnEvent(ActionWithEventFunc)

		// OnRequest enables you to define routes for your
		// service to respond when it is requested inernally or by some of
		// tthe connected peers.
		OnRequest(r ServiceRouter)
	}

	// Engine is application runtime managing services etc.
	Engine interface {
		// Register enables you to sergiste individual services
		// to application when having Addon would be too much overhead
		// in design.
		Register(...Service)

		Start() Error
		Stop() Error

		// ResolvePeerTo adds record into
		// internal name resolution registry
		//
		ResolvePeerTo(ns, ipport string)
	}

	// ServiceRouter is managing how your services communicate with each other.
	ServiceRouter interface {
		Handle(path string, handler ServiceHandler)
	}

	// ServiceResponse response returned by service.
	ServiceResponse interface {
		// Duration returns time.Duration representing the time it took reciever
		// to fulfill the request.
		Duration() time.Duration

		// Payload is retuns fully read concrete type representing
		// response payload.
		Payload() any
	}

	ServiceRequest interface {
		// Payload returns request payload.
		Payload() any

		// URL specifies either the URI being requested (for service
		// requests) or the URL to access (for client requests).
		//
		// For server requests, the URL is parsed from the URI
		// supplied on the Request-Line as stored in RequestURI.  For
		// most requests, fields other than Path and RawQuery will be
		// empty. (See RFC 7230, Section 5.3)
		//
		// For client requests, the URL's Host specifies the server to
		// connect to, while the Request's Host field optionally
		// specifies the Host header value to send in the HTTP
		// request.
		URL() URL
	}

	ServiceFactory interface {
		Service() (Service, Error)
	}
)
