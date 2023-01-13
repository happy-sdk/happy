// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/mkungla/happy/pkg/hlog"
	"github.com/mkungla/happy/pkg/vars"
	"golang.org/x/exp/slog"
)

type Session struct {
	mu sync.RWMutex

	logger *hlog.Logger
	opts   *Options

	ready      context.Context
	readyFunc  context.CancelFunc
	sig        context.Context
	sigRelease context.CancelFunc
	err        error

	done chan struct{}
	evch chan Event
	svss map[string]*ServiceInfo
	apis map[string]API

	disposed bool
}

func (s *Session) Ready() <-chan struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d := s.ready.Done()
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

func (s *Session) setServiceInfo(info *ServiceInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.svss == nil {
		s.svss = make(map[string]*ServiceInfo)
	}

	s.svss[info.addr.String()] = info
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

	if s.readyFunc != nil {
		s.readyFunc()
	}
	if s.err == nil {
		s.err = ErrSessionDestroyed
	}

	s.mu.Unlock()

	if s.sigRelease != nil {
		s.sigRelease()
		s.sigRelease = nil
	}

	s.mu.Lock()
	if s.evch != nil {
		close(s.evch)
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

func (s *Session) Log() *hlog.Logger {
	return s.logger
}

// Done enables you to hook into chan to know when application exits
// however DO NOT use that for graceful shutdown actions.
// Use Application.AddExitFunc instead.
func (s *Session) Done() <-chan struct{} {
	s.mu.Lock()
	if s.done == nil {
		s.done = make(chan struct{})
	}
	d := s.done
	s.mu.Unlock()
	return d
}

// Value returns the value associated with this context for key, or nil
func (s *Session) Value(key any) any {
	switch k := key.(type) {
	case string:
		if v, ok := s.opts.Load(k); ok {
			return v
		}
	case *int:
		if s.sig != nil && s.sig.Err() != nil {
			s.Destroy(s.sig.Err())
		}
		return nil
	}
	return nil
}

func (s *Session) String() string {
	return "happyx.Session"
}

func (s *Session) Get(key string) vars.Variable {
	return s.opts.Get(key)
}

func (s *Session) Set(key string, val any) error {
	return s.opts.Set(key, val)
}

func (s *Session) Has(key string) bool {
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
	} else {
		s.Log().SystemDebug(
			"session is disposed - skipping event dispatch",
			slog.String("scope", ev.Scope()),
			slog.String("key", ev.Key()),
		)
	}
	s.mu.Unlock()
}

func (s *Session) start() error {
	s.ready, s.readyFunc = context.WithCancel(context.Background())
	s.sig, s.sigRelease = signal.NotifyContext(s, os.Interrupt, os.Kill)
	s.evch = make(chan Event, 100)
	s.Log().SystemDebug("session started")
	return nil
}

func (s *Session) setReady() {
	s.mu.Lock()
	s.readyFunc()
	s.mu.Unlock()
	s.Log().SystemDebug("session ready")
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

func (s *Session) API(addonName string) (API, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	api, ok := s.apis[addonName]
	if !ok {
		return nil, fmt.Errorf("no api fo addon: %s", addonName)
	}
	return api, nil
}
