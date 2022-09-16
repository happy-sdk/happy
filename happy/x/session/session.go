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

package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
)

var ErrSession = happyx.NewError("session error")

type Session struct {
	mu       sync.RWMutex
	logger   happy.Logger
	settings *Settings

	done      chan struct{}
	err       error
	ready     context.Context
	readyFunc context.CancelFunc
	sig       context.Context
}

func New(logger happy.Logger, opts ...happy.OptionWriteFunc) *Session {
	return &Session{
		logger:   logger,
		settings: &Settings{},
	}
}

// API
func (s *Session) Ready() <-chan struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d := s.ready.Done()
	return d
}

func (s *Session) Start() happy.Error {
	s.ready, s.readyFunc = context.WithCancel(context.Background())
	return nil
}

func (s *Session) Destroy(err happy.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		s.logger.SystemDebugf("%s: already canceled", s.String())
		return
	}

	s.err = err
	// ensure chan
	if s.done != nil {
		close(s.done)
	}
	if s.readyFunc != nil {
		s.readyFunc()
	}

	if err := s.settings.Save(); err != nil && !errors.Is(err, happyx.ErrNotImplemented) {
		s.logger.Errorf("failed to save session settings: %s", err)
	}
}

func (s *Session) Log() happy.Logger {
	return s.logger
}
func (s *Session) Settings() happy.Settings {
	return s.settings
}

func (s *Session) Dispatch(happy.Event) {
	s.logger.NotImplemented("Session.Dispatch")
}

func (s *Session) Context() context.Context {
	s.logger.NotImplemented("Session.Context")
	return nil
}

func (s *Session) RequireServices(svcs ...string) happy.Error {
	return happyx.NotImplementedError("Session.RequireServices")
}

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (s *Session) Deadline() (deadline time.Time, ok bool) {
	s.logger.NotImplemented("Session.Deadline")
	return
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

// Value returns the value associated with this context for key, or nil
func (s *Session) Value(key any) any {
	switch k := key.(type) {
	case string:
		if v, ok := s.settings.Load(k); ok {
			return v
		}
	case *int:
		if s.sig != nil && s.sig.Err() != nil {
			s.Destroy(ErrSession.Wrap(s.sig.Err()))
		}
		return nil
	}
	return nil
}

func (s *Session) String() string {
	return "happyx.Session"
}

// // happy.Options interface
// func (s *Session) DeleteOption(key string) happy.Error {
// 	return happyx.NotImplementedError("Session.DeleteOption")
// }
// func (s *Session) ResetOptions() happy.Error {
// 	return happyx.NotImplementedError("Session.DeleteOption")
// }

// // happy.OptionReader interface
// func (s *Session) Read(p []byte) (n int, err error) {
// 	return 0, happyx.NotImplementedError("Session.Read")
// }

// func (s *Session) GetOption(key string) (happy.Variable, happy.Error) {
// 	return nil, happyx.NotImplementedError("Session.GetOption")
// }

// func (s *Session) GetOptionOrDefault(key string, defval ...any) (val happy.Variable) {
// 	s.logger.NotImplemented("Session.GetOptionOrDefault")
// 	return
// }

// func (s *Session) HasOption(key string) bool {
// 	s.logger.NotImplemented("Session.HasOption")
// 	return false
// }

// func (s *Session) RangeOptions(f func(v happy.Variable) bool) {
// 	s.logger.NotImplemented("Session.RangeOptions")
// }

// func (sess *Session) GetAllOptions() happy.Variables {
// 	return nil
// }

// // happy.OptionWriter interface
// func (sess *Session) Write(p []byte) (n int, err error)    { return 0, happyx.ErrNotImplemented }
// func (sess *Session) SetOption(happy.Variable) happy.Error { return happyx.ErrNotImplemented }
// func (sess *Session) SetOptionKeyValue(key string, val any) happy.Error {
// 	return happyx.ErrNotImplemented
// }
// func (sess *Session) SetOptionValue(key string, val happy.Value) happy.Error {
// 	return happyx.ErrNotImplemented
// }
