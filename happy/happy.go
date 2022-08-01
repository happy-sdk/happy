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

// Package happy makes developers happy by providing simple and and powerful
// sdk to build cross-platform cli, gui and mobile applications.
package happy

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/mkungla/happy/config"
	"github.com/mkungla/varflag/v5"
	"github.com/mkungla/vars/v5"
)

const (
	LevelSystemDebug LogLevel = iota
	LevelDebug
	LevelVerbose

	LevelTask

	LevelNotice
	LevelOk
	LevelIssue

	LevelWarn
	LevelDeprecated
	LevelNotImplemented
	LevelExperimental

	LevelError
	LevelCritical
	LevelAlert
	LevelEmergency

	LevelOut
	LevelQuiet
)

var (
	ErrMissingImplementation = errors.New("missing implementation")
)

type (
	Application interface {
		AddCommand(cmd Command)
		AddCommandFn(fn func() (Command, error))
		Commands() map[string]Command
		Command() Command
		Session() Session
		Dispose(code int)
		Exit(code int, err error)
		Flag(name string) varflag.Flag
		Flags() varflag.Flags
		Run()
		Setup(action Action)
		Before(action Action)
		Do(action Action)
		AfterSuccess(action Action)
		AfterFailure(action Action)
		AfterAlways(action Action)
		Stats() Stats
		Config() config.Config
		ServiceManager() ServiceManager
		AddonManager() AddonManager
		Set(key string, val any) error
		Store(key string, val any) error
		AddExitFunc(exitFn func(code int, ctx Session))
		RegisterAddons(addonFns ...func() (Addon, error))
		RegisterServices(serviceFns ...func() (Service, error))
	}

	ServiceManager interface {
		Register(ns string, services ...Service) error
		Initialize(ctx Session, keepAlive bool) error
		Start(ctx Session, srvurls ...string)
		Stop() error
		Len() int
		Tick(ctx Session)
		OnEvent(ctx Session, ev string, payload []vars.Variable)
		StartService(ctx Session, id string)
	}

	Service interface {
		Name() string
		Slug() string
		Description() string
		Version() Version
		Multiple() bool

		// service
		Initialize(ctx Session) error
		Start(ctx Session) error
		Stop(ctx Session) error
		Tick(ctx Session, ts time.Time, delta time.Duration) error
		OnEvent(ctx Session, ev string, payload vars.Collection)
		Call(fn string) (any, error)
	}

	AddonManager interface {
		Register(addons ...Addon) error
		Addons() []Addon
	}

	Addon interface {
		Name() string
		Slug() string
		Description() string
		Version() Version
		DefaultSettings(ctx Session) config.Settings
		// Configured returning false
		// prevents addon and it's services to be loaded.
		// expecting user to configure the addon.
		Configured(ctx Session) bool
		Services() []Service
		Commands() ([]Command, error)
	}

	Stats interface {
		Dispose()
		Elapsed() time.Duration
		Start() context.CancelFunc
		Next() <-chan struct{}
		Now() time.Time
		FPS() int
		MaxFPS() int
		Frame()
	}

	Command interface {
		String() string
		SetCategory(category string)
		Category() string
		SetShortDesc(description string)
		ShortDesc() string
		LongDesc(desc ...string) string
		Usage(usage ...string) string

		Before(action Action)
		Do(action Action)
		AfterSuccess(action Action)
		AfterFailure(action Action)
		AfterAlways(action Action)

		AddFlag(f varflag.Flag) error
		Flags() varflag.Flags
		AcceptsFlags() bool
		Args() []vars.Value
		AcceptsArgs() bool

		SetParents(parents []string)
		Parents() []string
		Parent() Command
		SetParent(Command)

		AddSubCommand(cmd Command)
		SubCommand(name string) (cmd Command, exists bool)
		HasSubcommands() bool
		Subcommands() (scmd []Command)

		Verify() error

		ExecuteBeforeFn(ctx Session) error
		ExecuteDoFn(ctx Session) error
		ExecuteAfterSuccessFn(ctx Session) error
		ExecuteAfterFailureFn(ctx Session) error
		ExecuteAfterAlwaysFn(ctx Session) error

		RequireServices(...string) // services to enable before the command
		ServiceLoaders() []string
	}

	Session interface {
		Options
		Log() Logger
		Settings() Settings
		NotifyContext(signals ...os.Signal)
		Deadline() (deadline time.Time, ok bool)
		Done() <-chan struct{}
		Err() error
		Value(key any) any
		Destroy(err error)
		Args() []vars.Value
		Arg(pos int) vars.Value
		Flags() varflag.Flags
		Flag(name string) varflag.Flag
		Out(response any)
		Start(cmd string, args []vars.Value, flags varflag.Flags) error

		// TaskAdd adds task to session
		TaskAdd(string)
		TaskAddf(string, ...any)
		// TaskDone decrements the internal WaitGroup counter by one.
		TaskDone()
		// Ready blocks until the internal  WaitGroup counter is zero.
		Ready()

		RequireService(string)
		Dispatch(ev string, val ...vars.Variable) error
		Events() []string
		GetEventPayload(ev string) ([]vars.Variable, error)
	}

	Settings interface {
		Options
		OptionGetterFallback
		LoadFile() error
		SaveFile() error
	}

	Logger interface {
		SystemDebug(args ...any)
		SystemDebugf(template string, args ...any)

		Debug(args ...any)
		Debugf(template string, args ...any)

		Info(args ...any)
		Infof(template string, args ...any)

		Notice(args ...any)
		Noticef(template string, args ...any)

		Ok(args ...any)
		Okf(template string, args ...any)

		Issue(nr int, args ...any)
		Issuef(nr int, template string, args ...any)

		Task(args ...any)
		Taskf(template string, args ...any)

		Warn(args ...any)
		Warnf(template string, args ...any)

		Experimental(args ...any)
		Experimentalf(template string, args ...any)

		Deprecated(args ...any)
		Deprecatedf(template string, args ...any)

		NotImplemented(args ...any)
		NotImplementedf(template string, args ...any)

		Error(args ...any)
		Errorf(template string, args ...any)

		Critical(args ...any)
		Criticalf(template string, args ...any)

		Alert(args ...any)
		Alertf(template string, args ...any)

		Emergency(args ...any)
		Emergencyf(template string, args ...any)

		Out(args ...any)
		Outf(template string, args ...any)

		Sync() error

		Level() LogLevel
		SetLevel(LogLevel)

		Write(data []byte) (n int, err error)
	}

	Plugin interface {
		Name() string
		Slug() string
		Enabled() bool
		Command(cmd string, args ...any) (string, error)
	}

	// Action represents user callable function
	Action  func(ctx Session) error
	Options interface {
		OptionSetter
		OptionGetter
		Delete(key string)
		Range(f func(key string, value vars.Value) bool)
	}
	OptionSetter interface {
		Set(key string, val any) error
		Store(key string, val any) error
	}
	OptionGetter interface {
		Get(key string) vars.Value
		Has(key string) bool
	}

	OptionGetterFallback interface {
		Getd(key string, defval ...any) (val vars.Value)
	}

	Option func(opts OptionSetter) error

	// Level for logger.
	LogLevel int8

	Version interface {
		String() string
	}
)
