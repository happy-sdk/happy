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
		Dispose(code int)
		Exit(code int, err error)
		Flag(name string) varflag.Flag
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
		AddExitFunc(exitFn func(code int, ctx Session))
		RegisterAddons(addonFns ...func() (Addon, error))
		RegisterServices(serviceFns ...func() (Service, error))
	}

	ServiceManager interface {
		Register(services ...Service) error
		Initialize(ctx Session, log Logger, keepAlive bool) error
		Stop() error
		Len() int
	}

	Service interface {
		Name() string
		Slug() string
		Description() string
		Version() Version
		Multiple() bool
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
		SetShortDesc(description string)
		Before(action Action)
		Do(action Action)
		AfterSuccess(action Action)
		AfterFailure(action Action)
		AfterAlways(action Action)
		Verify() error
		Flags() varflag.Flags
		GetSubCommand(name string) (cmd Command, exists bool)
		Args() []vars.Value

		ExecuteBeforeFn(ctx Session) error
		ExecuteDoFn(ctx Session) error
		ExecuteAfterSuccessFn(ctx Session) error
		ExecuteAfterFailureFn(ctx Session) error
		ExecuteAfterAlwaysFn(ctx Session) error
		SetParents(parents []string)
		AddSubCommand(cmd Command)
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
