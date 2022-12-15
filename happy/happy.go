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
	// Level used by implementations
	// to separate debug logs of application from implementation debug logs.
	LOG_SYSTEMDEBUG
)

// ApplicationVersion version is set
// then app.version is set to this value instead of
// reading it from debug.BuildInfo. This is only fo cases where you need
// version embed with go build and take full responsibility
// about setting correct version see:
// https://github.com/golang/go/issues/50603
var ApplicationVersion string

// A Kind represents the specific kind of kinde that a Value represents.
// The zero Kind is not a valid kind.
type ValueKind uint

const (
	KindInvalid ValueKind = iota
	KindBool
	KindInt
	KindInt8
	KindInt16
	KindInt32
	KindInt64
	KindUint
	KindUint8
	KindUint16
	KindUint32
	KindUint64
	KindUintptr
	KindFloat32
	KindFloat64
	KindComplex64
	KindComplex128
	KindArray
	KindChan
	KindFunc
	KindInterface
	KindMap
	KindPointer
	KindSlice
	KindString
	KindStruct
	KindUnsafePointer
)

///////////////////////////////////////////////////////////////////////////////
// COMMON
///////////////////////////////////////////////////////////////////////////////

type (
	LogPriority int

	LogEntry struct {
		Time     time.Time
		Priority LogPriority
		Message  string
		Label    string
	}

	// Action is common callback in happy framework.
	ActionFunc           func(sess Session) error
	ActionWithStatusFunc func(sess Session, status ApplicationStatus) error

	// ActionWithArgs is common callback in happy framework which has
	// privides arguments as vars.Collection.
	ActionWithArgsFunc  func(sess Session, args Variables) error
	ActionWithEventFunc func(sess Session, ev Event) error

	ActionCommandFunc func(
		sess Session,
		flags Flags,
		assets FS,
		status ApplicationStatus,
		apis []API,
	) error

	// ActionWithError is common callback in happy framework which has
	// second arguments as previous error id any otherwise nil.
	ActionWithErrorFunc func(sess Session, err Error) error

	// TickTock is operation set in given minimal time frame it can be executed.
	// You can throttle tick/tocks to cap FPS or for [C|G]PU throttling.
	//
	// Tock is helper called after each tick to separate
	// logic processed in tick and do post processing on tick.
	// Tocks are useful mostly for GPU ops which need to do post proccessing
	// of frames rendered in tick.
	ActionTickFunc func(sess Session, ts time.Time, delta time.Duration) error

	ActionCronSchedulerSetup func(CronScheduler)

	ActionCronFunc func(sess Session) error

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
		Err() Error
	}

	ApplicationStatus interface {
		Started() time.Time
		Elapsed() time.Duration
		TotalEvents() int

		Addons() []AddonInfo
		Services() []ServiceStatus
		GetServiceStatus(url URL) (ServiceStatus, Error)
		Dependencies() []DependencyInfo
		DebugInfo() Variables
	}

	Logger interface {
		// LogInitialization should be called once
		// and log all log entries matchin LogPriority.
		// e.g. Used in Application after setting LogPriority from flag.
		LogInitialization()

		GetPriority() LogPriority
		SetPriority(LogPriority)

		// RuntimePriority sets priority to priority.
		// You can call ResetPriority to reset it back to
		// initial priority. Or last lpriority set with SetPriority.
		SetRuntimePriority(LogPriority)

		// ResetPriority resets priority back to initial priority.
		ResetPriority()

		// Writer returns the output destination for the logger.
		Writer() io.Writer

		// SetOutput sets the output destination for the logger.
		SetOutput(w io.Writer)

		// Print calls Output to print to the standard logger.
		// Arguments are handled in the manner of fmt.Print.
		Print(v ...any)

		// Printf calls Output to print to the standard logger.
		// Arguments are handled in the manner of fmt.Printf.
		Printf(format string, v ...any)

		// Println calls Output to print to the standard logger.
		// Arguments are handled in the manner of fmt.Println.
		Println(v ...any)

		// Output writes the output for a logging event. The string s contains
		// the text to print after the prefix specified by the flags of the
		// Logger. A newline is appended if the last character of s is not
		// already a newline. Calldepth is the count of the number of
		// frames to skip when computing the file name and line number
		// if Llongfile or Lshortfile is set; a value of 1 will print the details
		// for the caller of Output.
		Output(calldepth int, s string) error

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

		// LOG_SYSTEMDEBUG
		SystemDebug(args ...any)
		SystemDebugf(template string, args ...any)

		OnEntry(func(LogEntry))

		OptionDefaultsSetter
	}

	Error interface {
		Error() string
		Code() int
		Wrap(error) Error
		WithText(string) Error
		WithTextf(template string, args ...any) Error
	}

	Slug interface {
		fmt.Stringer
		Valid() bool
	}

	URL interface {
		fmt.Stringer
		Args() Variables
		PeerService() string
	}

	Cron interface {
		Cron(ActionCronSchedulerSetup)
	}

	CronHandler interface {
		Job() ActionWithErrorFunc
		Expr() string
	}

	CronScheduler interface {
		Job(expr string, cb ActionCronFunc)
	}

	// FS could be e.g. embed.FS
	FS interface {
		fs.FS
		ReadDir(name string) ([]fs.DirEntry, error)
		ReadFile(name string) ([]byte, error)
	}

	TickerFuncs interface {
		// OnTick enables you to define func body for operation set
		// to call in minimal timeframe until session is valid and
		// service is running.
		OnTick(ActionTickFunc)

		// OnTock is helper called right after OnTick to separate
		// your primary operations and post prossesing logic.
		OnTock(ActionTickFunc)
	}

	EventListener interface {
		OnEvent(scope, key string, cb ActionWithEventFunc)
		OnAnyEvent(ActionWithEventFunc)
	}
)

///////////////////////////////////////////////////////////////////////////////
// OPTIONS
///////////////////////////////////////////////////////////////////////////////

type (
	// OptionSetFunc is callback function to apply configurable options.
	OptionSetFunc func(opts OptionSetter) Error

	// Options interface enables you to apply configuration options in
	// different parts of your application.
	Options interface {
		// DeleteOption allows you remove option value and it's key from
		// your option set. return ErrOption when key does not exist or
		// ErrOptionRO if key is read only.
		DeleteOption(key string) Error

		// ResetOptions clears all options and restores defaults
		// if defaults have been set
		ResetOptions() Error

		// OptionGetter's embedded interface allows you to control how your
		// implementation makes configuration options readable. For example you
		// can implement your own middleware for option getter.
		OptionGetter

		// OptionSetter's embedded interface allows you to control how your
		// implementation makes changes to configuration options. For example you
		// can implement your own middleware for option setter.
		OptionSetter
	}

	// OptionGetter is read only representation of Options
	OptionGetter interface {

		// GetOption return option by key or ErrOption,
		// if non existingOption was requested.
		GetOption(key string) (Variable, Error)

		// GetOptionOrDefault looks option and returns
		// provided default value if key does not exists.
		// This call does not set that default to options!
		GetOptionOrDefault(key string, defval any) (val Variable)

		// HasOption reports whether option set has option with key.
		HasOption(key string) bool

		// Range calls f sequentially for each key -> value present in the options.
		// If f returns false, range stops the iteration.
		RangeOptions(f func(opt Variable) bool)

		// Get all options as vars.Collection
		GetOptions() Variables

		GetOptionSetFunc(srcKey, destKey string) OptionSetFunc
	}

	OptionSetter interface {
		SetOption(Variable) Error
		SetOptionKeyValue(key string, val any) Error
		SetOptionValue(key string, val Value) Error
	}

	OptionDefaultsSetter interface {
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
		Has(key string) bool
		Delete(key string)
		Store(key string, value any)
		Get(key string) (v Variable)
		LoadWithPrefix(prfx string) (Variables, bool)
		// ExtractWithPrefix returns Variables with prefix left trimed
		ExtractWithPrefix(prfx string) Variables
		Load(key string) (v Variable, ok bool)
		LoadAndDelete(key string) (v Variable, loaded bool)
		LoadOrDefault(key string, value any) (v Variable, loaded bool)
		LoadOrStore(key string, value any) (actual Variable, loaded bool)
		Range(f func(v Variable) bool)
		All() []Variable
	}

	// Variable is key=value pair
	Variable interface {
		String() string
		Key() string
		Value() Value
		ReadOnly() bool
		Underlying() any
		Len() int
		Bool() bool
		Int() int
		Int8() int8
		Int16() int16
		Int32() int32
		Int64() int64
		Uint() uint
		Uint8() uint8
		Uint16() uint16
		Uint32() uint32
		Uint64() uint64
		Float32() float32
		Float64() float64
		Complex64() complex64
		Complex128() complex128
		Uintptr() uintptr
		Fields() []string
	}

	Value interface {
		// String MUST return string value Value
		String() string
		// Underlying MUST return original value
		// from what this Value was created.
		Underlying() any
		Len() int
		Bool() (bool, error)
		Int() (int, error)
		Int8() (int8, error)
		Int16() (int16, error)
		Int32() (int32, error)
		Int64() (int64, error)
		Uint() (uint, error)
		Uint8() (uint8, error)
		Uint16() (uint16, error)
		Uint32() (uint32, error)
		Uint64() (uint64, error)
		Float32() (float32, error)
		Float64() (float64, error)
		Complex64() (complex64, error)
		Complex128() (complex128, error)
		Uintptr() (uintptr, error)
		Fields() []string
	}
)

///////////////////////////////////////////////////////////////////////////////
// APPLICATION
///////////////////////////////////////////////////////////////////////////////

type (
	Configurator interface {
		UseLogger(Logger)
		GetLogger() (Logger, Error)

		UseSession(Session)
		GetSession() (Session, Error)

		UseMonitor(Monitor)
		GetMonitor() (Monitor, Error)

		UseAssets(FS)
		GetAssets() (FS, Error)

		UseEngine(Engine)
		GetEngine() (Engine, Error)

		GetApplicationOptions() Variables

		OptionDefaultsSetter // Sets application configuration options
	}

	// Application is interface for application runtime.
	Application interface {
		Configure(Configurator) Error

		RegisterAddon(Addon)
		RegisterAddons(...AddonCreateFunc)

		RegisterService(Service)
		RegisterServices(...ServiceCreateFunc)

		Log() Logger

		Main()
		Version() Version
		Exit(code int)
		CommandActionSetter
		CommandSubCommandSetter
		CommandFlagSetter
		TickerFuncs
		Cron
	}

	// Maybe name it Context?
	Session interface {
		fmt.Stringer

		// Ready blocks until the session is ready to use.
		Ready() <-chan struct{}

		// Start session
		Start() Error

		// Destroy session
		Destroy(err Error)

		// Log returns application logger.
		Log() Logger

		Settings() Settings

		Get(key string) Variable
		Store(key string, value any)
		Opts() Variables

		Dispatch(Event)

		// simple context you can use as parent context
		// instead of passing session it self
		// even tho it implements context aswell.
		Context() context.Context

		RequireServices(status ApplicationStatus, urls ...string) ServiceLoader

		Events() <-chan Event

		// Done() behaves like context.Done and indcates that session was destroyed
		// Err() returns last error in session
		context.Context

		// Options
	}

	Settings interface {
		Variables

		// Save saves settings.
		Save() Error
	}

	Monitor interface {
		Start(Session) Error
		Stop() Error
		Status() ApplicationStatus

		RegisterAddon(AddonInfo)
		SetServiceStatus(url, key string, val any)

		EventListener
	}
)

// /////////////////////////////////////////////////////////////////////////////
// CLI
// /////////////////////////////////////////////////////////////////////////////
type (
	CommandCreateFunc func() (Command, Error)

	FlagCreateFunc func() (Flag, Error)

	CommandActionSetter interface {
		// Before is optional action to execute before Main.
		// Useful to check preconidtion for your command Main.
		// Since any required services by command will be initialized in parallel
		// with this action you can call sess.Ready if you need to use services
		// inside the Init.
		Before(ActionCommandFunc)

		// Main is action where your commands main logic should live.
		// Main function body is required unless it is wrapper (parent) command
		// for it's subcommand you want to have your application logic.
		Do(ActionCommandFunc)

		// AfterSuccess is optional callback called when Main returns without error.
		AfterSuccess(action ActionFunc)

		// AfterFailure is optional callback when Main returns error.
		// Second argument will be error returned.
		AfterFailure(action ActionWithErrorFunc)

		// AfterAlways is optional callback to execute reqardless did Main
		// succeed or not.
		AfterAlways(action ActionWithStatusFunc)
	}

	CommandSubCommandSetter interface {
		AddSubCommand(Command)
		AddSubCommands(...CommandCreateFunc)
	}

	CommandFlagSetter interface {
		AddFlag(Flag)
		AddFlags(flags ...FlagCreateFunc)
	}

	Command interface {
		// Slug String returns command name.
		Slug() Slug

		// RequireServices ensures that services by their URL
		// will be started before Command.Main call. These services will be started
		// in parallel with Command.Before. If you need to have services ready
		// inside Command.Before then you can call sess.Ready which will block
		// until all services are running.
		WithServices(urls ...URL)

		AttachParent(parent Command)
		Parent() Command
		SubCommands() []Command
		Flags() Flags
		Flag(name string) Flag
		SetParents(parents []string)
		Parents() (parents []string)
		Err() Error
		Verify() Error
		Category() string
		HasSubcommands() bool
		SubCommand(name string) (cmd Command, exists bool)

		Description() string
		UsageDescription() string

		ExecuteBeforeAction(sess Session, assets FS, status ApplicationStatus, apis []API) Error
		ExecuteDoAction(sess Session, assets FS, status ApplicationStatus, apis []API) Error
		ExecuteAfterFailureAction(sess Session, err Error) Error
		ExecuteAfterSuccessAction(sess Session) Error
		ExecuteAfterAlwaysAction(sess Session, status ApplicationStatus) Error

		CommandActionSetter
		CommandSubCommandSetter
		CommandFlagSetter
	}

	Flags interface {

		// Name of the flag set
		Name() string

		// Len returns number of flags in this set
		// not including subset flags.
		Len() int

		// Add flag to flag set
		Add(...Flag) error

		// Get named flag
		Get(name string) (Flag, error)
		// Add sub set of flags to flag set
		AddSet(...Flags) error

		// GetActiveSets.
		GetActiveSets() []Flags

		// Position of flag set
		Pos() int

		// Was flagset (sub command present)
		Present() bool

		Args() []Value
		// AcceptsArgs returns true if set accepts any arguments.
		AcceptsArgs() bool

		// Flags returns slice of flags in this set
		Flags() []Flag

		// Sets retruns subsets of flags under this flagset.
		Sets() []Flags

		// Parse all flags and sub sets
		Parse(args []string) error
	}

	Flag interface {
		// Get primary name for the flag. Usually that is long option
		Name() string

		// Get flag default value
		Default() Variable

		// Usage returns a usage description for that flag
		Usage() string
		UsageAliases() string

		// Flag returns flag with leading - or --
		// useful for help menus
		Flag() string

		// Return flag aliases
		Aliases() []string

		// IsHidden reports whether to show that flag in help menu or not.
		Hidden() bool

		// Hide flag from help menu.
		Hide()

		// IsGlobal reports whether this flag was global and was set before any command or arg
		Global() bool

		// BelongsTo marks flag non global and belonging to provided named command.
		AttachTo(cmdname string)

		// CommandName returns empty string if command is not set with .BelongsTo
		// When BelongsTo is set to wildcard "*" then this function will return
		// name of the command which triggered this flag to be parsed.
		BelongsTo() string

		// Pos returns flags position after command. In case of mulyflag first position is reported
		Pos() int
		// Unset unsets the value for the flag if it was parsed, handy for cases where
		// one flag cancels another like --debug cancels --verbose
		Unset()

		// Present reports whether flag was set in commandline
		Present() bool

		// Var returns vars.Variable for this flag.
		// where key is flag and Value flags value.
		Var() Variable

		// Required sets this flag as required
		MarkAsRequired()

		// IsRequired returns true if this flag is required
		Required() bool

		// Parse value for the flag from given string. It returns true if flag
		// was found in provided args string and false if not.
		// error is returned when flag was set but had invalid value.
		Parse([]string) (bool, error)

		// String calls Value().String()
		String() string

		Input() []string
	}
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
		Name() string
		Slug() Slug
		Cronjobs()
		Version() Version
		Commands() []Command
		Services() []Service

		Options
		API() API
	}

	API interface {
		// So that you dont return addon it self as API
		// Commands() error
		// Services() error
	}

	AddonInfo struct {
		Name    string
		Slug    Slug
		Version Version
	}

	DependencyInfo struct {
		Path    string // module path
		Version string // module version
		Sum     string // checksum
	}

	ServiceStatus struct {
		URL        string
		Registered bool
		Running    bool
		Failed     bool
		StartedAt  time.Time
		StoppedAt  time.Time
		Err        string
	}
	// AddonFactory interface {
	// 	GetAddonCreateFunc() AddonCreateFunc
	// }
)

///////////////////////////////////////////////////////////////////////////////
// SERVICE
///////////////////////////////////////////////////////////////////////////////

type (
	ServiceCreateFunc func() (Service, Error)
	// ServiceHandler is callback to be called when Service recieves a Request.
	ServiceHandler func(sess Session, req ServiceRequest) ServiceResponse

	// Service interface
	Service interface {
		Slug() Slug
		Name() string

		URL() URL

		// OnInitialize is called when app is preparing runtime and attaching
		// services.
		OnInitialize(ActionWithStatusFunc)

		// OnStart is called when service is requested to be started.
		// For instace when command is requiring this service or whenever
		// service is required on runtime via sess.RequireService call.
		//
		// Start can be called multiple times in case of service restarts.
		// If you do not want to allow service restarts you should implement
		// your logic in OnStop when it's called first time and check that
		// state OnStart.
		OnStart(ActionWithArgsFunc)

		// OnStop is called once application exits, session is destroyed or
		// when service restart is requested.
		OnStop(ActionFunc)

		// OnRequest enables you to define routes for your
		// service to respond when it is requested inernally or by some of
		// tthe connected peers.
		OnRequest(r ServiceRouter)

		Register(Session) (BackgroundService, Error)

		TickerFuncs

		EventListener

		Cron
	}

	BackgroundService interface {
		Initialize(Session, ApplicationStatus) Error
		Start(sess Session, args Variables) Error
		Stop(Session) Error
		Tick(sess Session, ts time.Time, delta time.Duration) Error
		Tock(sess Session, ts time.Time, delta time.Duration) Error
		HandleEvent(sess Session, ev Event) error
	}

	ServiceLoader interface {
		Loaded() <-chan struct{}
		Err() Error
	}

	// Engine is application runtime managing services etc.
	Engine interface {
		// Register enables you to sergiste individual services
		// to application when having Addon would be too much overhead
		// in design.
		Register(...Service) Error

		Start(Session) Error
		Stop(Session) Error

		ListenEvents(<-chan Event)

		AttachMonitor(monitor Monitor) Error
		Monitor() Monitor

		// ResolvePeerTo adds record into
		// internal name resolution registry
		// API not confirmed
		// ResolvePeerTo(ns, ipport string)

		TickerFuncs
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

	// ServiceFactory interface {
	// 	Service() (Service, Error)
	// }
)
