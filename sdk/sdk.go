// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package sdk

import (
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/migration"
	"github.com/happy-sdk/happy/sdk/settings"
)

type Action func(sess *Session) error
type ActionWithArgs func(sess *Session, args Args) error
type ActionTick func(sess *Session, ts time.Time, delta time.Duration) error
type ActionTock func(sess *Session, delta time.Duration, tps int) error
type ActionWithPrevErr func(sess *Session, err error) error

type Session struct{}
type Command struct{}
type Service struct{}

func New(s settings.Settings) *Main {
	return &Main{}
}

func (s *Session) Log() logging.Logger {
	return nil
}

type Main struct {
	mu     sync.RWMutex
	sealed bool

	log logging.Logger
}

type Addon struct{}

// WithAddons adds addons to the prototype.
func (m *Main) WithAddon(addons *Addon) *Main {
	return m
}

func (m *Main) WithMigrations(mm *migration.Manager) *Main {
	return m
}

// OnSetup runs at first execution.
func (m *Main) WithService(svc *Service) *Main {
	return m
}

func (m *Main) WithCommand(cmd *Command) *Main {
	return m
}

func (m *Main) WithFlag(flags varflag.Flag) *Main {
	return m
}

func (m *Main) WithLogger(l logging.Logger) *Main {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.log != nil {
		m.log.Error("logger already set")
		return m
	}
	m.log = l
	return m
}

// Run runs the prototype.
func (m *Main) Run() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// initialize
	if m.sealed {
		m.log.Error("can not call .Run application is aleady sealed")
		return
	}
	m.sealed = true
	if m.log == nil {
		m.log = logging.Default(logging.LevelOk)
	}

}

func (m *Main) Before(a ActionWithArgs)          {}
func (m *Main) Do(a ActionWithArgs)              {}
func (m *Main) AfterSuccess(a Action)            {}
func (m *Main) AfterFailure(a ActionWithPrevErr) {}
func (m *Main) AfterAlways(a ActionWithPrevErr)  {}
func (m *Main) Tick(a ActionTick)                {}
func (m *Main) Tock(a ActionTock)                {}
