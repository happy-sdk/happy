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

type (
	Application interface {
		AddCommand(cmd Command)
		AddCommandFn(fn func() (Command, error))
		Exit(code int, err ...error)
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
	}

	ServiceManager interface{}
	AddonManager   interface{}

	Stats interface {
		Dispose()
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
)
