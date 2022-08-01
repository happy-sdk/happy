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
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/happy/internal/jsonlog"
	"github.com/mkungla/happy/internal/settings"
	"github.com/mkungla/varflag/v5"
	"github.com/mkungla/vars/v5"
)

type Context struct {
	ctx            context.Context
	mu             sync.RWMutex
	tasks          int64
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

	args  []vars.Value
	flags varflag.Flags
}

func New(logger happy.Logger) *Context {
	return &Context{
		ctx:      context.Background(),
		data:     new(vars.Collection),
		events:   &sync.Map{},
		settings: settings.New(),
		logger:   logger,
	}
}

func (c *Context) Set(key string, val any) error {
	if c.ready && (strings.HasPrefix(key, "app.") || strings.HasPrefix(key, "service.")) {
		return config.ErrSetConfigOpt
	}
	c.data.Set(key, val)
	return nil
}

func (c *Context) Store(key string, val any) error {
	if c.ready && (strings.HasPrefix(key, "app.") || strings.HasPrefix(key, "service.")) {
		return config.ErrSetConfigOpt
	}
	c.data.Store(key, val)
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
	c.mu.RLock()
	err := c.err
	c.mu.RUnlock()
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

func (ctx *Context) Args() []vars.Value {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.args
}
func (ctx *Context) Arg(pos int) vars.Value {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	if len(ctx.args) == 0 || len(ctx.args) < pos {
		return vars.NewValue("")
	}
	return ctx.args[pos]
}

func (ctx *Context) Flags() varflag.Flags {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.flags
}

func (ctx *Context) Flag(name string) varflag.Flag {
	f, err := ctx.flags.Get(name)
	if err != nil && !errors.Is(err, varflag.ErrNoNamedFlag) {
		ctx.Log().Error(err)
	}

	if err == nil {
		return f
	}

	// thes could be predefined
	f, err = varflag.Bool(config.CreateSlug(name), false, "")
	if err != nil {
		ctx.Log().Error(err)
	}
	return f
}

func (ctx *Context) Start(cmd string, args []vars.Value, flags varflag.Flags) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.args = args
	ctx.flags = flags
	return nil
}

func (ctx *Context) Out(response any) {
	if f, err := ctx.flags.Get("json"); err == nil && f.Present() {
		if log, ok := ctx.logger.(*jsonlog.Logger); ok {
			log.SetResponse(response)
		}
	} else {
		fmt.Println(response)
	}
}

func (ctx *Context) Ready() {
ticker:
	for {
		select {
		case <-ctx.Done():
			break ticker
		default:
			if ctx.tasks == 0 {
				break ticker
			}
		}
	}
}

func (ctx *Context) TaskDone() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.tasks--
}

func (ctx *Context) TaskAddf(format string, args ...any) {
	ctx.TaskAdd(fmt.Sprintf(format, args...))
}
func (ctx *Context) TaskAdd(msg string) {
	ctx.Log().SystemDebugf("session: %s", msg)
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.tasks++
}

func (ctx *Context) RequireService(serviceURL string) {
	err := ctx.Dispatch("happy.services.enable", vars.New(serviceURL, true))
	if err != nil {
		ctx.Log().Error(err)
		ctx.done <- struct{}{}
		return
	}

	u, err := url.Parse(serviceURL)
	if err != nil {
		ctx.Log().Error(err)
		ctx.done <- struct{}{}
		return
	}
	u.RawQuery = ""

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	ctx.Log().Experimental("service load deadline is 30 seconds")
	deadline, deadlineCancel := context.WithTimeout(ctx.ctx, time.Second*30)
	defer deadlineCancel()

	key := "service." + u.String()
servicedep:
	for {
		select {
		case <-ticker.C:
			if !ctx.Has(key) {
				ctx.mu.Lock()
				ctx.err = fmt.Errorf("require.service: no such service - %s", serviceURL)
				ctx.mu.Unlock()
				return
			}

			if ctx.Get(key).Bool() {
				return
			}
		case <-deadline.Done():
			ctx.mu.Lock()
			ctx.err = fmt.Errorf("require.service: ctx cancelled or deadline reached - %s", serviceURL)
			ctx.mu.Unlock()
			break servicedep
		}
	}
}

func (ctx *Context) Dispatch(ev string, val ...vars.Variable) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if _, ok := ctx.events.Load(ev); ok {
		return fmt.Errorf("event %q already queued", ev) //nolint: goerr113
	}

	ctx.Log().SystemDebugf("store event: %s", ev)
	ctx.events.Store(ev, val)
	return nil
}

func (ctx *Context) Events() []string {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	var events []string
	ctx.events.Range(func(k, v any) bool {
		events = append(events, fmt.Sprint(k))
		return true
	})
	return events
}

func (ctx *Context) GetEventPayload(ev string) ([]vars.Variable, error) {
	payload, loaded := ctx.events.LoadAndDelete(ev)
	if !loaded {
		return nil, fmt.Errorf("failed to load event %q", ev)
	}
	args, ok := payload.([]vars.Variable)
	if !ok {
		return nil, fmt.Errorf("invalid event payload for %q", ev)
	}
	return args, nil
}
