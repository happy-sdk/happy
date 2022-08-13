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
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"time"
)

type Session struct {
}

func New(opts ...happy.OptionWriteFunc) *Session {
	return &Session{}
}

// API
func (sess *Session) Ready() <-chan struct{}                     { return nil }
func (sess *Session) Destroy(err happy.Error)                    {}
func (sess *Session) Log() happy.Logger                          { return nil }
func (sess *Session) Settings() happy.Settings                   { return nil }
func (sess *Session) Dispatch(happy.Event)                       {}
func (sess *Session) Context() context.Context                   { return nil }
func (sess *Session) RequireServices(svcs ...string) happy.Error { return happyx.ErrNotImplemented }

// context.Context interface
func (sess *Session) Deadline() (deadline time.Time, ok bool) { return }
func (sess *Session) Done() <-chan struct{}                   { return nil }
func (sess *Session) Err() error                              { return happyx.ErrNotImplemented }
func (sess *Session) Value(key any) any                       { return nil }

// happy.Options interface
func (sess *Session) DeleteOption(key string) happy.Error { return nil }
func (sess *Session) ResetOptions() happy.Error           { return nil }

// happy.OptionReader interface
func (sess *Session) Read(p []byte) (n int, err error)                                  { return 0, happyx.ErrNotImplemented }
func (sess *Session) GetOption(key string) (happy.Variable, happy.Error)                { return nil, nil }
func (sess *Session) GetOptionOrDefault(key string, defval ...any) (val happy.Variable) { return }
func (sess *Session) HasOption(key string) bool                                         { return false }
func (sess *Session) RangeOptions(f func(key string, value happy.Value) bool)           {}
func (sess *Session) GetAllOptions() happy.Variables                                    { return nil }

// happy.OptionWriter interface
func (sess *Session) Write(p []byte) (n int, err error)    { return 0, happyx.ErrNotImplemented }
func (sess *Session) SetOption(happy.Variable) happy.Error { return happyx.ErrNotImplemented }
func (sess *Session) SetOptionKeyValue(key string, val any) happy.Error {
	return happyx.ErrNotImplemented
}
func (sess *Session) SetOptionValue(key string, val happy.Value) happy.Error {
	return happyx.ErrNotImplemented
}
