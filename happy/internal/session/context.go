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

	"container/list"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/happy/internal/jsonlog"
	"github.com/mkungla/happy/internal/settings"
	"github.com/mkungla/varflag/v6"
	"github.com/mkungla/vars/v6"
)

type Context struct {
	mu  sync.RWMutex
	ctx context.Context
	sig context.Context

	tasks int64
	// cancelFromExit bool
	settings happy.Settings
	done     chan struct{}
	data     *vars.Collection
	err      error
	logger   happy.Logger

	events *list.List
	ready  bool

	args  []vars.Value
	flags varflag.Flags
	sm    happy.ServiceManager
}

func New(logger happy.Logger, sm happy.ServiceManager) *Context {
	return &Context{
		ctx:      context.Background(),
		data:     new(vars.Collection),
		events:   list.New(),
		settings: settings.New(),
		logger:   logger,
		sm:       sm,
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
	if ctx.flags == nil {
		return nil
	}

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

	ctx.Log().SystemDebug("service load deadline is 5 seconds")
	deadline, deadlineCancel := context.WithTimeout(ctx.ctx, time.Second*5)
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

func (ctx *Context) ServiceCall(serviceUrl, fnName string, args ...vars.Variable) (any, error) {
	if ctx.sm == nil {
		return nil, errors.New("can not access services from session")
	}
	return ctx.sm.ServiceCall(serviceUrl, fnName, args...)
}

func (ctx *Context) Dispatch(key string, payload ...vars.Variable) error {
	ev := happy.Event{
		Time:    time.Now().UTC(),
		Key:     key,
		Payload: new(vars.Collection),
	}

	for _, v := range payload {
		ev.Payload.Store(v.Key(), v.Value())
	}

	ctx.Log().SystemDebugf("store event: %s - payload(%d)", ev.Key, ev.Payload.Len())

	ctx.mu.Lock()
	ctx.events.PushBack(ev)
	ctx.mu.Unlock()
	return nil
}

func (ctx *Context) Events() []happy.Event {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	var events []happy.Event
	for e := ctx.events.Front(); e != nil; e = e.Next() {
		ev, ok := e.Value.(happy.Event)
		if !ok {
			ctx.Log().Error("failed to read event, not type of event: ", e.Value)
		}
		events = append(events, ev)
	}
	_ = ctx.events.Init()
	return events
}

// EventsByTypeLoadAndDelete (internal use)
// It returns slice of events by event name and removes
// these events from queue.
func (ctx *Context) EventsByType(ev string) []happy.Event {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	var events []happy.Event

	for e := ctx.events.Front(); e != nil; e = e.Next() {
		ev, ok := ctx.events.Remove(e).(happy.Event)
		if !ok {
			ctx.Log().Error("failed to read event by type, not type of event")
		}
		events = append(events, ev)
	}
	return events
}
