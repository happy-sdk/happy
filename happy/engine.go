// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mkungla/happy/pkg/address"
	"github.com/mkungla/happy/pkg/vars"
	"golang.org/x/exp/slog"
)

// noop for tick | tock
var nooptock = func(sess *Session, delta time.Duration, tps int) error { return nil }

type Engine struct {
	mu      sync.RWMutex
	running bool
	started time.Time

	tickAction ActionTick
	tockAction ActionTock

	readyCallback sync.Once
	engineOK      bool
	ctxCancel     context.CancelFunc
	ctx           context.Context

	evCancel  context.CancelFunc
	evContext context.Context

	registry map[string]*serviceContainer
	events   map[string]Event
}

func newEngine() *Engine {
	engine := &Engine{
		registry: make(map[string]*serviceContainer),
		events:   make(map[string]Event),
	}

	return engine
}

func (e *Engine) start(sess *Session) error {
	sess.Log().SystemDebug("starting engine ...")
	e.started = time.Now()
	if e.tickAction == nil && e.tockAction != nil {
		return fmt.Errorf("%w: register tick action or move tock logic into tick action", ErrEngine)
	}

	var init sync.WaitGroup

	e.loopStart(sess, &init)

	e.servicesInit(sess, &init)

	init.Wait()

	if e.engineOK {
		e.startEventDispatcher(sess)
		sess.setReady()
	} else {
		sess.Destroy(fmt.Errorf("%w: starting engine failed", ErrEngine))
	}

	e.running = true
	sess.Log().SystemDebug("engine started")
	return nil
}

func (e *Engine) uptime() time.Duration {
	return time.Since(e.started)
}

func (e *Engine) onTick(action ActionTick) {
	e.tickAction = action
	return
}
func (e *Engine) onTock(action ActionTock) {
	e.tockAction = action
	return
}

func (e *Engine) startEventDispatcher(sess *Session) {
	e.evContext, e.evCancel = context.WithCancel(sess)

	go func(sess *Session) {
	evLoop:
		for {
			select {
			case <-e.evContext.Done():
				break evLoop
			case ev, ok := <-sess.evch:
				if !ok {
					continue
				}
				e.handleEvent(sess, ev)
			}
		}
	}(sess)
}

func (e *Engine) registerEvent(ev Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	skey := ev.Scope() + "." + ev.Key()
	if _, ok := e.events[skey]; ok {
		return fmt.Errorf("%w: event already registered %s", ErrEngine, skey)
	}
	e.events[skey] = ev
	return nil
}

func (e *Engine) handleEvent(sess *Session, ev Event) {
	skey := ev.Scope() + "." + ev.Key()

	e.mu.RLock()
	_, rev := e.events[skey]
	registry := e.registry
	e.mu.RUnlock()

	if len(skey) == 1 || !rev {
		sess.Log().NotImplemented("event not registered, ignoring", slog.String("scope", ev.Scope()), slog.String("key", ev.Key()))
		return
	}
	switch ev.Scope() {
	case "services":
		switch ev.Key() {
		case "start.services":
			payload := ev.Payload()
			payload.Range(func(v vars.Variable) bool {
				go e.serviceStart(sess, v.String())
				return true
			})
		case "stop.services":
			payload := ev.Payload()
			payload.Range(func(v vars.Variable) bool {
				go e.serviceStop(sess, v.String(), nil)
				return true
			})
		}
	}
	for _, svcc := range registry {
		go svcc.handleEvent(sess, ev)
	}
	sess.Log().SystemDebug("event", slog.String("scope", ev.Scope()), slog.String("key", ev.Key()))
}

func (e *Engine) loopStart(sess *Session, init *sync.WaitGroup) {
	e.ctx, e.ctxCancel = context.WithCancel(sess)

	if e.tickAction == nil && e.tockAction == nil {
		sess.Log().SystemDebug("engine loop skipped")
		e.engineOK = true
		return
	}
	// we should have enured that tick action is set.
	// so only set noop tock if needed
	if e.tockAction == nil {
		e.tockAction = nooptock
	}

	init.Add(2)
	defer init.Done()

	go func() {
		lastTick := time.Now()

		ttick := time.NewTicker(time.Duration(sess.Get("app.throttle.ticks").Int64()))
		defer ttick.Stop()

	engineLoop:
		for {
			select {
			case <-e.ctx.Done():
				sess.Log().SystemDebug("engineLoop ctx Done")
				break engineLoop
			case now := <-ttick.C:
				// mark engine running only if first tick tock are successful
				e.readyCallback.Do(func() {
					sess.Log().SystemDebug("engine started")

					e.mu.Lock()
					e.engineOK = true
					e.mu.Unlock()

					init.Done()
				})

				delta := now.Sub(lastTick)
				lastTick = now
				if err := e.tickAction(sess, lastTick, delta); err != nil {
					sess.Log().Error("tick error", err)
					sess.Dispatch(NewEvent("engine", "app.tick.err", nil, err))
					break engineLoop
				}
				tickDelta := time.Since(lastTick)
				if err := e.tockAction(sess, tickDelta, 0); err != nil {
					sess.Log().Error("tock error", err)
					sess.Dispatch(NewEvent("engine", "app.tock.err", nil, err))
					break engineLoop
				}

			}
		}
		sess.Log().SystemDebug("engine loop stopped")
	}()
}

func (e *Engine) stop(sess *Session) error {
	if !e.running {
		return nil
	}
	sess.Log().SystemDebug("stopping engine")

	e.ctxCancel()
	<-e.ctx.Done()

	e.evCancel()
	<-e.evContext.Done()

	var graceful sync.WaitGroup
	for u, rsvc := range e.registry {
		if !rsvc.info.Running() {
			continue
		}
		go func(url string, svcc *serviceContainer) {
			defer graceful.Done()
			// wait for engine context is canceled which triggers
			// r.ctx also to be cancelled, however lets wait for the
			// context done since r.ctx is cancelled after last tickk completes.
			// so e.xtc is not parent of r.ctx.
			<-svcc.Done()
			// lets call stop now we know that tick loop has exited.
			e.serviceStop(sess, url, nil)
		}(u, rsvc)
		graceful.Add(1)
	}
	graceful.Wait()
	sess.Log().SystemDebug("engine stopped")
	return nil
}

func (e *Engine) serviceRegister(sess *Session, svc *Service) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if svc == nil {
		return fmt.Errorf("%w: attempt to register <nil> service", ErrEngine)
	}

	if e.running {
		return fmt.Errorf("%w: can not register services engine is already running - %s", ErrEngine, svc.name)
	}

	hostaddr, err := address.Parse(sess.Get("happy.host.addr").String())
	if err != nil {
		return errors.Join(ErrEngine, err)
	}
	addr, err := hostaddr.ResolveService(svc.name)
	if err != nil {
		return err
	}

	addrstr := addr.String()
	if _, ok := e.registry[addrstr]; ok {
		return fmt.Errorf("%w: services is already registered %s", ErrEngine, addr)
	}

	container := svc.container(sess, addr)
	e.registry[addrstr] = container
	sess.setServiceInfo(&container.info)
	sess.Log().Debug("registered service", slog.String("service", addrstr))
	return nil
}

func (e *Engine) servicesInit(sess *Session, init *sync.WaitGroup) {
	e.mu.Lock()
	svccount := len(e.registry)
	e.mu.Unlock()
	if svccount == 0 {
		sess.Log().SystemDebug("no services to initialize ...")
		return
	}
	init.Add(len(e.registry))
	for svcaddrstr, svcc := range e.registry {
		go func(addr string, c *serviceContainer) {
			defer init.Done()
			if err := c.initialize(sess); err != nil {
				sess.Log().Error("failed to initialize service", err, slog.String("service", c.info.Addr().String()))
				return
			}
			// register events what service listens for
			for ev, _ := range c.svc.listeners {
				scope, key, _ := strings.Cut(ev, ".")
				// we can ignore error because this error is handled
				// when emitter registers this event. Listening
				// for unregistered event is not an error.
				_ = e.registerEvent(registerEvent(scope, key, "has listener", nil))
			}
		}(svcaddrstr, svcc)
	}
	sess.Log().SystemDebug("initialize services ...")
}

func (e *Engine) serviceStart(sess *Session, svcurl string) {
	e.mu.RLock()
	svcc, ok := e.registry[svcurl]
	e.mu.RUnlock()
	if !ok {
		sess.Log().Warn("no such service to start", slog.String("service", svcurl))
		return
	}
	if svcc.info.Failed() {
		sess.Log().SystemDebug("skip starting service due previous errors", slog.String("service", svcurl))
		return
	}

	sarg := slog.String("service", svcurl)
	if !ok {
		sess.Log().Warn(
			"requested unknown service",
			sarg,
		)
		return
	}

	if svcc.info.Running() {
		sess.Log().Warn(
			"failed to start service, service already running",
			sarg,
		)
		return
	}

	if err := svcc.start(e.ctx, sess); err != nil {
		sess.Log().Error(
			"failed to start service",
			err,
			sarg,
		)
		return
	}

	go func(svcc *serviceContainer, svcurl string, sarg slog.Attr) {

		if svcc.svc.tickAction == nil {
			<-e.ctx.Done()
			svcc.cancel(nil)
			return
		}

		ttick := time.NewTicker(time.Duration(sess.Get("app.throttle.ticks").Int64()))
		defer ttick.Stop()

		lastTick := time.Now()
		tis := 0
		tps := 0
	ticker:
		for {
			select {
			case <-svcc.ctx.Done():
				svcc.cancel(nil)
				break ticker
			case now := <-ttick.C:
				if lastTick.Truncate(time.Second) == now.Truncate(time.Second) {
					tis++
				} else {
					tps = tis
					tis = 0
				}
				delta := now.Sub(lastTick)
				lastTick = now
				if err := svcc.tick(sess, lastTick, delta); err != nil {
					e.serviceStop(sess, svcurl, err)
					break ticker
				}
				tickDelta := time.Since(lastTick)
				if err := svcc.tock(sess, tickDelta, tps); err != nil {
					e.serviceStop(sess, svcurl, err)
					break ticker
				}
			}
		}
	}(svcc, svcurl, sarg)
}

func (e *Engine) serviceStop(sess *Session, svcurl string, err error) {
	sarg := slog.String("service", svcurl)

	e.mu.RLock()
	svcc, ok := e.registry[svcurl]
	e.mu.RUnlock()
	if !ok {
		sess.Log().Warn("no such service to start", sarg)
		return
	}
	sess.Log().SystemDebug("stopping service", sarg)
	if e := svcc.stop(sess, err); e != nil {
		sess.Log().Error("failed to stop service", e, sarg)
	}
}
