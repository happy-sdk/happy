// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mkungla/happy/pkg/address"
	"github.com/mkungla/happy/pkg/vars"
	"golang.org/x/exp/slog"
)

// noop for tick | tock
var ttnoop = func(sess *Session, ts time.Time, delta time.Duration) error { return nil }

type Engine struct {
	mu      sync.RWMutex
	running bool
	started time.Time

	tickAction, tockAction ActionTick

	readyCallback sync.Once
	engineOK      bool
	ctxCancel     context.CancelFunc
	ctx           context.Context

	evCancel  context.CancelFunc
	evContext context.Context

	registry map[string]*serviceContainer
}

func newEngine() *Engine {
	return &Engine{
		registry: make(map[string]*serviceContainer),
	}
}

func (e *Engine) start(sess *Session) error {
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
	return nil
}

func (e *Engine) uptime() time.Duration {
	return time.Since(e.started)
}

func (e *Engine) onTick(action ActionTick) {
	e.tickAction = action
	return
}
func (e *Engine) onTock(action ActionTick) {
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

func (e *Engine) handleEvent(sess *Session, ev Event) {
	sess.Log().SystemDebug("event", slog.String("scope", ev.Scope()), slog.String("key", ev.Key()))
	switch ev.Scope() {
	case "service.loader":
		switch ev.Key() {
		case "require.services":
			payload := ev.Payload()
			payload.Range(func(v vars.Variable) bool {
				e.serviceStart(sess, v.String())
				return true
			})
		}
	}
	e.mu.RLock()
	registry := e.registry
	e.mu.RUnlock()
	for _, svcc := range registry {
		svcc.handleEvent(sess, ev)
	}
}

func (e *Engine) loopStart(sess *Session, init *sync.WaitGroup) {
	sess.Log().SystemDebug("starting engine ...")

	e.ctx, e.ctxCancel = context.WithCancel(sess)

	if e.tickAction == nil && e.tockAction == nil {
		sess.Log().SystemDebug("engine loop skipped")
		e.engineOK = true
		return
	}
	// we should have enured that tick action is set.
	// so only set noop tock if needed
	if e.tockAction == nil {
		e.tockAction = ttnoop
	}

	init.Add(2)
	defer init.Done()

	go func() {
		lastTick := time.Now()

		ttick := time.NewTicker(time.Microsecond * 100)
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

				delta := time.Since(lastTick)
				lastTick = now
				if err := e.tickAction(sess, lastTick, delta); err != nil {
					sess.Log().Error("tick error", err)
					sess.Dispatch(NewEvent("engine", "app.tick.err", nil, err))
					break engineLoop
				}
				tickDelta := time.Since(lastTick)
				if err := e.tockAction(sess, lastTick, tickDelta); err != nil {
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
			sarg := slog.String("service", url)
			// wait for engine context is canceled which triggers
			// r.ctx also to be cancelled, however lets wait for the
			// context done since r.ctx is cancelled after last tickk completes.
			// so e.xtc is not parent of r.ctx.
			<-svcc.ctx.Done()
			// lets call stop now we know that tick loop has exited.
			if err := svcc.stop(sess); err != nil {
				sess.Log().Error("failed to stop service", err, sarg)
			}
			sess.Log().SystemDebug("stopping service", sarg)
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
		return fmt.Errorf("%w: can not register services engine is already running - %s", ErrEngine, svc.slug)
	}

	hostaddr, err := address.Parse(sess.Get("app.host.addr").String())
	if err != nil {
		return errors.Join(ErrEngine, err)
	}
	addr, err := hostaddr.ResolveService(svc.slug)
	if err != nil {
		return err
	}

	addrstr := addr.String()
	if _, ok := e.registry[addrstr]; ok {
		return fmt.Errorf("%w: services is already registered %s", ErrEngine, addr)
	}

	container := svc.container(sess, addr)
	e.registry[addrstr] = container
	sess.setServiceInfo(container.info)
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
			sess.Log().Debug("initializing service", slog.String("service", c.info.addr.String()))
			if err := c.initialize(sess); err != nil {
				sess.Log().Error("failed to initialize service", err, slog.String("service", c.info.addr.String()))
				return
			}
		}(svcaddrstr, svcc)
	}
	sess.Log().SystemDebug("initialize services ...")
}

func (e *Engine) serviceStart(sess *Session, svcurl string) {
	sess.Log().SystemDebug("starting service", slog.String("service", svcurl))
	e.mu.RLock()
	svcc, ok := e.registry[svcurl]
	e.mu.RUnlock()

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
	sess.Log().Debug("starting service", sarg)

	if err := svcc.start(sess); err != nil {
		sess.Log().Error(
			"failed to start service",
			err,
			sarg,
		)
		return
	}

	svcc.ctx, svcc.cancel = context.WithCancelCause(sess)
	go func(svcc *serviceContainer, sarg slog.Attr) {
		svcc.info.start()
		lastTick := time.Now()
		kill := func(err error) {
			sess.Log().Error("service error", err, sarg)
			svcc.info.stop()
			svcc.cancel(err)
			if err := svcc.stop(sess); err != nil {
				sess.Log().Error("error when stopping service", err, sarg)
			}
		}
		if svcc.svc.tickAction == nil {
			<-e.ctx.Done()
			svcc.cancel(nil)
			return
		}

		ttick := time.NewTicker(time.Microsecond * 100)
		defer ttick.Stop()

	ticker:
		for {
			select {
			case <-e.ctx.Done():
				svcc.cancel(nil)
				break ticker
			case now := <-ttick.C:
				delta := time.Since(lastTick)
				lastTick = now
				if err := svcc.tick(sess, lastTick, delta); err != nil {
					kill(err)
					break ticker
				}
				tickDelta := time.Since(lastTick)
				if err := svcc.tock(sess, lastTick, tickDelta); err != nil {
					kill(err)
					break ticker
				}
			}
		}
	}(svcc, sarg)
}
