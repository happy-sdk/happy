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
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/happy/internal/settings"
	"github.com/mkungla/vars/v5"
)

type Context struct {
	ctx            context.Context
	mu             sync.Mutex
	cancelFromExit bool
	settings       happy.Settings
	done           chan struct{}
	data           *vars.Collection
	err            error
	logger         happy.Logger
	sig            context.Context
	// payload        *sync.Map
	tempDir string
	events  *sync.Map
	ready   bool

	deadline time.Time
}

func New() *Context {
	return &Context{
		ctx:      context.Background(),
		data:     new(vars.Collection),
		events:   &sync.Map{},
		settings: settings.New(),
	}
}

func (c *Context) Set(key string, val any) error {
	if c.ready && strings.HasPrefix(key, "app.") {
		return config.ErrSetConfigOpt
	}
	c.data.Set(key, val)
	return nil
}

// Get returns the value stored in the map for a key, or nil if no
// The ok result indicates whether value was found in the map.
func (c *Context) Get(key string) vars.Value {
	return c.data.Get(key)
}

// Has result indicates whether this setting is set.
func (c *Context) Has(key string) bool {
	return c.data.Has(key)
}

// Delete deletes the value for a key.
func (c *Context) Delete(key string) {
	c.data.Delete(key)
}

func (m *Context) Range(f func(key string, value vars.Value) bool) {
	m.data.Range(f)
}

func (c *Context) Log() happy.Logger {
	return c.logger
}

func (c *Context) Settings() happy.Settings {
	return c.settings
}

func (c *Context) NotifyContext(signals ...os.Signal) {
	if c.sig != nil {
		c.Log().Warn("session NotifyContext called multiple times")
		return
	}
	c.sig, _ = signal.NotifyContext(c, signals...)
}

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return
}

// Done enables you to hook into chan to know when application exits
// however DO NOT use that for graceful shutdown actions.
// Use Application.AddExitFunc instead.
func (c *Context) Done() <-chan struct{} {
	c.mu.Lock()
	if c.done == nil {
		c.done = make(chan struct{})
	}
	d := c.done
	c.mu.Unlock()
	return d
}

func (c *Context) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

func (c *Context) Value(key any) any {
	switch k := key.(type) {
	case string:
		if v, ok := c.data.Load(k); ok {
			return v
		}
	case *int:
		if c.sig != nil && c.sig.Err() != nil {
			c.Destroy(c.sig.Err())
		}
		return nil
	}
	return c.ctx.Value(key)
}

func (ctx *Context) Destroy(err error) {
	if err == nil {
		err = ErrContextFinished
	}
	ctx.mu.Lock()
	if ctx.err != nil {
		ctx.mu.Unlock()
		ctx.Log().SystemDebugf("%s: already canceled", ctx.String())
		return
	}

	ctx.err = err
	// ensure chan
	if ctx.done != nil {
		close(ctx.done)
	}

	ctx.data.Range(func(key string, value vars.Value) bool {
		ctx.data.Delete(key)
		return true
	})

	ctx.Log().SystemDebugf("%s: %s", ctx.String(), ctx.err)
	ctx.mu.Unlock()
}

func (ctx *Context) String() string {
	return "happy.Session"
}
