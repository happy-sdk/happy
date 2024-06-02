// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/branding"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/addon"
	"github.com/happy-sdk/happy/sdk/app/internal/application"
	"github.com/happy-sdk/happy/sdk/app/internal/initializer"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/migration"
	"github.com/happy-sdk/happy/sdk/services"
)

type Main struct {
	mu     sync.RWMutex
	init   *initializer.Initializer
	rt     application.Runtime
	log    *logging.QueueLogger
	booted bool
}

func New[S settings.Settings](s S) *Main {
	m := &Main{
		log: logging.NewQueueLogger(),
	}
	m.init = initializer.New(s, &m.rt, m.log)
	return m
}

func (m *Main) AddInfo(paragraph string) *Main {
	if !m.canConfigure("add extra info") {
		return m
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init.MainAddInfo(paragraph)
	return m
}

func (m *Main) AfterAlways(a action.WithPrevErr) *Main {
	if !m.canConfigure("adding AfterAlways action") {
		return m
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init.MainAfterAlways(a)
	return m
}

func (m *Main) AfterFailure(a action.WithPrevErr) *Main {
	if !m.canConfigure("adding AfterFailure action") {
		return m
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init.MainAfterFailure(a)
	return m
}

func (m *Main) AfterSuccess(a action.Action) *Main {
	if !m.canConfigure("adding AfterSuccess action") {
		return m
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init.MainAfterSuccess(a)
	return m
}

func (m *Main) Before(a action.WithArgs) *Main {
	if !m.canConfigure("adding Before action") {
		return m
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init.MainBefore(a)
	return m
}

func (m *Main) BeforeAlways(a action.WithArgs) *Main {
	if !m.canConfigure("adding BeforeAlways action") {
		return m
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init.MainBeforeAlways(&m.rt, a)
	return m
}

// Run starts the Application.
func (m *Main) Run() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.booted {
		m.log.LogDepth(1, logging.LevelWarn, "application already booted")
		m.mu.Unlock()
		return
	}
	m.booted = true
	m.log.LogDepth(1, logging.LevelDebug, "preparing runtime")

	defer func() {
		if r := recover(); r != nil {
			// Log the panic message
			var errMessage string
			if err, ok := r.(error); ok {
				errMessage = err.Error()
			} else {
				errMessage = fmt.Sprintf("%v", r)
			}

			// Obtain and log the stack trace
			stackTrace := string(debug.Stack())
			m.log.LogDepth(1, logging.LevelBUG, "panic (recovered)", slog.String("msg", errMessage))
			fmt.Println(stackTrace)
			m.rt.Exit(1)
		}
	}()

	if m.init == nil {
		m.log.BUG("initializer is nil, not set correctly")
		return
	}

	if err := m.init.Configure(); err != nil {
		if errors.Is(err, initializer.ErrExitWithSuccess) {
			m.rt.Exit(0)
			return
		}
		m.log.Error("app configuration failed", slog.String("error", err.Error()))
		{
			// rare case where logger is not available, then use slog
			// to consume the log queue if it is not already consumed.
			if m.init != nil {
				for _, r := range m.log.Consume() {
					slogr := r.Record(time.Local)
					slog.Default().Handler().Handle(context.Background(), slogr)
				}
			}
		}

		m.rt.Exit(1)
		return
	}

	// dispose the initializer
	{
		if err := m.init.Finalize(); err != nil {
			m.log.Error("disposing initializer failed", slog.String("error", err.Error()))
			m.rt.Exit(1)
			return
		}
		m.init = nil
	}

	go func() {
		m.rt.Start()
	}()
	osmain(m.rt.ExitCh())
}

func (m *Main) Do(a action.WithArgs) *Main {
	if m.canConfigure("setting do action") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.init.MainDo(a)
	}
	return m
}

func (m *Main) SetOptions(a ...options.Arg) *Main {
	if m.canConfigure("setting options") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.init.SetOptions(a...)
	}
	return m
}

func (m *Main) Tick(a action.Tick) *Main {
	if m.canConfigure("setting Tick action") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.init.MainTick(a)
	}
	return m
}

func (m *Main) Tock(a action.Tock) *Main {
	if m.canConfigure("setting Tick action") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.init.MainTock(a)
	}
	return m
}

func (m *Main) WithAddon(addon *addon.Addon) *Main {
	if !m.canConfigure("attaching addon") {
		return m
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.init.WithAddon(addon)
	return m
}

func (m *Main) WithBrand(b *branding.Builder) *Main { return m }

func (m *Main) WithCommand(cmd *command.Command) *Main {
	if m.canConfigure("add subcommand") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.init.MainAddCommand(cmd)
	}
	return m
}

func (m *Main) WithFlags(ffns ...varflag.FlagCreateFunc) *Main {
	if m.canConfigure("adding flags") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.init.MainAddFlags(ffns)
	}
	return m
}

func (m *Main) WithLogger(logger logging.Logger) *Main {
	if m.canConfigure("setting logger") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.init.SetLogger(logger)
	}
	return m
}

func (m *Main) WithMigrations(mm *migration.Manager) *Main { return m }

func (m *Main) WithOptions(opts ...options.Spec) *Main {
	if m.canConfigure("setting logger") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.init.WithOptions(opts)
	}
	return m
}

func (m *Main) WithService(svc *services.Service) *Main {
	if m.canConfigure("setting service") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.rt.AddService(svc)
	}
	return m
}

func (m *Main) Setup(setup action.Action) *Main {
	if m.canConfigure("set setup action") {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.init.WithSetup(setup)
	}
	return m
}

func (m *Main) canConfigure(errmsg string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.booted {
		slog.Error("application is already booted", slog.String("msg", errmsg))
		return false
	}

	if m.init == nil {
		slog.Error("initializer is nil, not set correctly", slog.String("msg", errmsg))
		return false
	}
	return !m.init.HasFailed()
}
