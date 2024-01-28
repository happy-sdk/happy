// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/settings"
)

var (
	ErrSession          = fmt.Errorf("%w:session", Error)
	ErrSessionDestroyed = fmt.Errorf("%w:destroyed", ErrSession)
)

type Session struct {
	mu sync.RWMutex

	logger  logging.Logger
	profile *settings.Profile
	opts    *Options

	ready         context.Context
	readyCancel   context.CancelFunc
	terminate     context.Context // SIGINT or SIGTERM listener
	terminateStop context.CancelFunc
	kill          context.Context // SIGKILL listener
	killStop      context.CancelFunc
	err           error

	done   chan struct{}
	closed chan struct{}
	evch   chan Event
	svss   map[string]*ServiceInfo
	apis   map[string]API

	allowUserCancel bool
	terminated      bool
	disposed        bool
	valid           bool
}

// Ready returns channel which blocks until session considers application to be ready.
// It is ensured that Ready closes before root or command Do function is called.
func (s *Session) Ready() <-chan struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d := s.ready.Done()
	s.Log().LogDepth(3, logging.LevelSystemDebug, "waiting session")
	return d
}

// Err returns session error if any or nil
// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
// After Err returns a non-nil error, successive calls to Err return the same error.
func (s *Session) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	err := s.err
	return err
}

func (s *Session) ServiceInfo(svcurl string) (*ServiceInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	svcinfo, ok := s.svss[svcurl]
	if !ok {
		return nil, fmt.Errorf("%w: unknown service %s", ErrService, svcurl)
	}
	return svcinfo, nil
}

func (s *Session) Destroy(err error) {
	if s.Err() != nil {
		// prevent Destroy to be called multiple times
		// e.g. by sig release or other contexts.
		return
	}

	s.mu.Lock()
	s.disposed = true

	// s.err is nil otherwise we would not be here
	s.err = err

	if s.readyCancel != nil {
		s.readyCancel()
	}
	if s.err == nil {
		s.err = ErrSessionDestroyed
	}

	s.mu.Unlock()

	if s.terminateStop != nil {
		s.terminateStop()
		s.terminateStop = nil
	}
	if s.killStop != nil {
		s.killStop()
		s.terminateStop = nil
	}

	s.mu.Lock()
	if s.evch != nil {
		close(s.evch)
	}

	if s.closed != nil {
		close(s.closed)
	}
	if s.done != nil {
		close(s.done)
	}

	s.mu.Unlock()
}

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (s *Session) Deadline() (deadline time.Time, ok bool) {
	return
}

func (s *Session) Log() logging.Logger {
	s.mu.RLock()
	if s.logger != nil {
		s.mu.RUnlock()
		return s.logger
	}
	s.mu.RUnlock()

	return s.logger
}

// Done enables you to hook into chan to know when application exits
// however DO NOT use that for graceful shutdown actions.
// Use Application.AddExitFunc or Cloesed instead.
func (s *Session) Done() <-chan struct{} {
	s.mu.Lock()
	if s.done == nil {
		s.done = make(chan struct{})
	}
	d := s.done
	s.mu.Unlock()
	return d
}

// Closed returns channel which blocks until session is closed.
// It is ensured that Closed closes before root or command
// "Do" after functions are called. This is useful for graceful shutdown actions.
func (s *Session) Closed() <-chan struct{} {
	s.mu.Lock()
	if s.closed == nil {
		s.closed = make(chan struct{})
	}
	d := s.closed
	s.mu.Unlock()
	return d
}

// UserClosed allows user to cancel application by pressing Ctrl+C
// or sending SIGINT or SIGTERM while application is running.
// By default this is not allowed. If you want to allow user to cancel
// application, you call this method any point at application runtime.
// Calling this method multiple times has no effect and triggers Warning
// log message.
func (s *Session) UserClosed() <-chan struct{} {
	s.mu.Lock()
	s.allowUserCancel = true
	s.mu.Unlock()
	return s.Closed()
}

// Value returns the value associated with this context for key, or nil
func (s *Session) Value(key any) any {
	switch k := key.(type) {
	case string:
		if v, ok := s.opts.Load(k); ok {
			return v
		}
	case *int:
		if s.terminate != nil && s.terminate.Err() != nil {
			if s.allowUserCancel {
				s.mu.Lock()
				s.terminateStop()
				s.terminate = nil
				s.terminated = true
				s.terminateStop = nil
				if s.closed != nil {
					close(s.closed)
					s.closed = nil
				}
				s.mu.Unlock()
				return nil
			}

			s.Destroy(s.terminate.Err())
		}
		if s.kill != nil && s.kill.Err() != nil {
			s.Destroy(s.kill.Err())
		}
		return nil
	}
	return nil
}

func (s *Session) String() string {
	return "happy.Session"
}

func (s *Session) Get(key string) vars.Variable {
	if !s.Has(key) {
		s.logger.LogDepth(3, logging.LevelWarn, "accessing non existing session option", slog.String("key", key))
		return vars.EmptyVariable
	}
	if s.profile != nil && s.profile.Has(key) {
		return s.profile.Get(key).Value()
	}
	return s.opts.Get(key)
}

func (s *Session) Set(key string, val any) error {
	if !s.opts.Accepts(key) {
		s.Log().Warn("setting non existing runtime options", slog.String("key", key))
		return fmt.Errorf("setting non existing runtime options: %s", key)
	}
	if strings.HasPrefix(key, "app.") {
		s.Log().Warn(
			"setting app.* variables can lead to unexpected behaviour",
			slog.String("key", key),
			slog.Any("value", val),
		)
	}
	if strings.HasPrefix(key, "fs.") {
		s.Log().Warn(
			"setting fs.* variables is not allowed",
			slog.String("key", key),
			slog.Any("value", val),
		)
		return fmt.Errorf("setting fs.* variables is not allowed, attempt to set %s = %v", key, val)
	}
	if err := s.opts.Set(key, val); err != nil {
		s.Log().Warn("setting runtime options failed", slog.String("key", key), slog.Any("value", val), slog.String("err", err.Error()))
		return err
	}
	return nil
}

func (s *Session) Has(key string) bool {
	if s.profile != nil && s.profile.Has(key) {
		return true
	}
	return s.opts.Has(key)
}

func (s *Session) Dispatch(ev Event) {
	if ev == nil {
		s.Log().Warn("received <nil> event")
		return
	}
	s.mu.Lock()
	if !s.disposed {
		s.evch <- ev
		s.mu.Unlock()
	} else {
		s.mu.Unlock()
		s.Log().SystemDebug(
			"session is disposed - skipping event dispatch",
			slog.String("scope", ev.Scope()),
			slog.String("key", ev.Key()),
		)
	}
}

func (s *Session) Setting(key string) settings.Setting {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.profile == nil || !s.profile.Loaded() {
		s.logger.Warn("session profile not loaded, while accessing settings", slog.String("key", key))
		return settings.Setting{}
	}
	if !s.profile.Has(key) {
		s.logger.Warn("accessing non existing setting", slog.String("key", key))
	}

	return s.profile.Get(key)
}

func (s *Session) API(addonName string) (API, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	api, ok := s.apis[addonName]
	if !ok {
		return nil, fmt.Errorf("no api for addon: %s", addonName)
	}
	return api, nil
}

// Settings returns a map of all settings which are defined by application
// and are user configurable.
func (s *Session) Profile() *settings.Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.profile == nil || !s.profile.Loaded() {
		s.logger.Warn("session profile not loaded, while accessing settings profile")
	}
	profile := s.profile
	return profile
}

// Config returns a map of all config options which are defined by application
func (s *Session) Config() *vars.Map {
	s.mu.RLock()
	defer s.mu.RUnlock()
	config := &vars.Map{}
	for _, cnf := range s.opts.config {
		if cnf.kind&ConfigOption != 0 {
			config.Store(cnf.key, s.opts.Get(cnf.key).Value())
		}
	}
	return config
}

// Opts returns a map of all options which are defined by application
// turing current session life cycle.
func (s *Session) Opts() *vars.Map {
	s.mu.RLock()
	defer s.mu.RUnlock()
	opts := &vars.Map{}
	for _, opt := range s.opts.db.All() {
		cnf, ok := s.opts.config[opt.Name()]
		if ok {
			if cnf.kind&RuntimeOption != 0 {
				opts.Store(cnf.key, s.opts.Get(cnf.key).Value())
			}
		} else {
			opts.Store(opt.Name(), opt.Value())
		}
	}
	return opts
}

// ServiceLoader calls NewServiceLoader with current session as first argument.
func (s *Session) ServiceLoader(svcs ...string) *ServiceLoader {
	return NewServiceLoader(s, svcs...)
}

func (s *Session) start() (err error) {
	s.ready, s.readyCancel = context.WithCancel(context.Background())
	s.terminate, s.terminateStop = signal.NotifyContext(s, os.Interrupt)
	s.kill, s.killStop = signal.NotifyContext(s, os.Kill)
	s.evch = make(chan Event, 100)
	s.Log().SystemDebug("session started")
	return err
}

func (s *Session) setReady() {
	s.mu.Lock()
	s.readyCancel()
	s.valid = true
	s.mu.Unlock()
	s.Log().SystemDebug("session ready")
}

func (s *Session) isValid() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.valid
}

func (s *Session) setProfile(profile *settings.Profile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// p := profile.All()

	// for _, setting := range p {
	// 	if setting.Persistent() {
	// 		fmt.Println("key:", setting.Key(), " val:", setting.Value().String())
	// 	}
	// }
	s.profile = profile
}

func (s *Session) canRecover(err error) bool {
	if err == nil {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.allowUserCancel && s.terminated {
		s.Log().Warn("session terminated by user")
		return true
	}
	return false
}

func (s *Session) registerAPI(addonName string, api API) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.apis == nil {
		s.apis = make(map[string]API)
	}
	if _, ok := s.apis[addonName]; ok {
		return fmt.Errorf("addon api already registered: %s", addonName)
	}
	s.apis[addonName] = api
	return nil
}

func (s *Session) setServiceInfo(info *ServiceInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.svss == nil {
		s.svss = make(map[string]*ServiceInfo)
	}

	s.svss[info.addr.String()] = info
}

func newSession(logger logging.Logger) *Session {
	s := &Session{
		logger: logger,
	}
	var err error
	s.opts, err = NewOptions("app", getRuntimeConfig())
	s.err = errors.Join(s.err, err)
	return s
}
