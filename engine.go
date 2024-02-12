// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/happy-sdk/happy/internal/fsutils"
	"github.com/happy-sdk/happy/pkg/strings/humanize"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/networking/address"
)

// noop for tick | tock
var nooptock = func(sess *Session, delta time.Duration, tps int) error { return nil }
var ErrEngine = fmt.Errorf("%w:engine", Error)

type engine struct {
	mu      sync.RWMutex
	running bool
	started time.Time

	tick ActionTick
	tock ActionTock

	readyCallback sync.Once
	engineOK      bool
	ctxCancel     context.CancelFunc
	ctx           context.Context

	evCancel  context.CancelFunc
	evContext context.Context

	registry map[string]*serviceContainer
	events   map[string]Event
	// address  string
}

func newEngine() *engine {
	engine := &engine{
		registry: make(map[string]*serviceContainer),
		events:   make(map[string]Event),
	}
	return engine
}

// func (e *engine) setAddress(addr string) {
// 	e.mu.Lock()
// 	defer e.mu.Unlock()
// 	e.address = addr
// }

func (e *engine) start(sess *Session) error {
	e.mu.Lock()

	sess.Log().SystemDebug("starting engine ...")
	e.started = time.Now()
	if e.tick == nil && e.tock != nil {
		return fmt.Errorf("%w: register tick action or move tock logic into tick action", ErrEngine)
	}

	var init sync.WaitGroup
	e.mu.Unlock()

	if sess.Get("app.stats.enabled").Bool() {
		if err := e.registerService(sess, statsService()); err != nil {
			return err
		}
	}
	e.loopStart(sess, &init)

	e.servicesInit(sess, &init)

	init.Wait()

	if e.engineOK {
		e.startEventDispatcher(sess)
	} else {
		sess.Destroy(fmt.Errorf("%w: starting engine failed", ErrEngine))
	}

	if sess.Get("app.stats.enabled").Bool() {
		loader := NewServiceLoader(sess, "app.stats")
		<-loader.Load()
		if err := loader.Err(); err != nil {
			return err
		}
	}

	e.running = true
	sess.stats.Update()
	sess.Log().SystemDebug("engine started")
	return nil
}

func (e *engine) uptime() time.Duration {
	if e.started.IsZero() {
		return 0
	}
	return time.Since(e.started)
}

func (e *engine) onTick(action ActionTick) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tick = action
}

func (e *engine) onTock(action ActionTock) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tock = action
}

func (e *engine) startEventDispatcher(sess *Session) {
	e.mu.Lock()
	defer e.mu.Unlock()
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

func (e *engine) registerEvent(ev Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	skey := ev.Scope() + "." + ev.Key()
	if _, ok := e.events[skey]; ok {
		return fmt.Errorf("%w: event already registered %s", ErrEngine, skey)
	}
	e.events[skey] = ev
	return nil
}

func (e *engine) handleEvent(sess *Session, ev Event) {
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

func (e *engine) loopStart(sess *Session, init *sync.WaitGroup) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.ctx, e.ctxCancel = context.WithCancel(sess)

	if e.tick == nil && e.tock == nil {
		sess.Log().SystemDebug("engine loop skipped")
		e.engineOK = true
		return
	}
	// we should have enured that tick action is set.
	// so only set noop tock if needed
	if e.tock == nil {
		e.tock = nooptock
	}

	init.Add(2)
	defer init.Done()

	go func() {
		// mark engine running only if first tick tock are successful
		e.readyCallback.Do(func() {
			sess.Log().SystemDebug("engine loop ready")
			e.mu.Lock()
			e.engineOK = true
			e.mu.Unlock()

			init.Done()
		})
		// start when session is ready
		<-sess.Ready()

		lastTick := time.Now()
		ttick := time.NewTicker(time.Duration(sess.Get("app.engine.throttle_ticks").Int64()))
		defer ttick.Stop()

	engineLoop:
		for {
			select {
			case <-e.ctx.Done():
				sess.Log().SystemDebug("engineLoop ctx Done")
				break engineLoop
			case now := <-ttick.C:
				delta := now.Sub(lastTick)
				lastTick = now
				if err := e.tick(sess, lastTick, delta); err != nil {
					sess.Log().Error("tick error", slog.String("err", err.Error()))
					sess.Dispatch(NewEvent("engine", "app.tick.err", nil, err))
					break engineLoop
				}
				tickDelta := time.Since(lastTick)
				if err := e.tock(sess, tickDelta, 0); err != nil {
					sess.Log().Error("tock error", slog.String("err", err.Error()))
					sess.Dispatch(NewEvent("engine", "app.tock.err", nil, err))
					break engineLoop
				}

			}
		}
		sess.Log().SystemDebug("engine loop stopped")
	}()
}

func (e *engine) stop(sess *Session) error {
	if !e.running {
		return nil
	}
	e.running = false

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
		graceful.Add(1)
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
	}
	if len(e.registry) > 0 {
		sess.Log().SystemDebug(fmt.Sprintf("waiting for %d services to stop", len(e.registry)))
	}
	graceful.Wait()
	sess.Log().SystemDebug("engine stopped")
	return nil
}

func (e *engine) registerService(sess *Session, svc *Service) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if svc == nil {
		return fmt.Errorf("%w: attempt to register <nil> service", ErrEngine)
	}

	if e.running {
		return fmt.Errorf("%w: can not register services engine is already running - %s", ErrEngine, svc.name)
	}

	hostaddr, err := address.Parse(sess.Get("app.address").String())
	if err != nil {
		return fmt.Errorf("%w:%s", ErrEngine, err.Error())
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
	sess.Log().LogDepth(3, logging.LevelSystemDebug, "registered service", slog.String("service", svc.name))
	return nil
}

func (e *engine) servicesInit(sess *Session, init *sync.WaitGroup) {
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
				sess.Log().Error(
					"failed to initialize service",
					slog.String("service", c.info.Addr().String()),
					slog.String("err", err.Error()))
				return
			}
			// register events what service listens for
			for ev := range c.svc.listeners {
				scope, key, _ := strings.Cut(ev, ".")
				// we can ignore error because this error is handled
				// when emitter registers this event. Listening for unregistered event is not an error.
				_ = e.registerEvent(registrableEvent(scope, key, "has listener", nil))
			}
		}(svcaddrstr, svcc)
		sess.stats.Update()
	}
	sess.Log().SystemDebug("initialize services ...")
}

func (e *engine) serviceStart(sess *Session, svcurl string) {
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
			slog.String("err", err.Error()),
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

		ttick := time.NewTicker(time.Duration(sess.Get("app.engine.throttle_ticks").Int64()))
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

func (e *engine) serviceStop(sess *Session, svcurl string, err error) {
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
		sess.Log().Error("failed to stop service", slog.String("err", err.Error()), sarg)
	}
}

func statsService() *Service {
	stats := NewService("app.stats")

	var (
		tt      = 0
		initial = true
		started time.Time
	)
	stats.Tick(func(sess *Session, ts time.Time, delta time.Duration) (err error) {
		sess.stats.Update()
		tt++
		if initial {
			startedAt := sess.stats.Get("app.started.at").String()
			if startedAt == "" {
				return nil
			}
			started, err = time.Parse(time.RFC3339, startedAt)
			if err != nil {
				return err
			}
			initial = false
		}
		uptime := time.Since(started)
		return sess.stats.Set("app.uptime", uptime.String())
	})

	stats.Tock(func(sess *Session, delta time.Duration, tps int) error {
		// every 10 ticks
		if tt < 10 && !initial {
			return nil
		}
		tt = 0

		cachePath := sess.Get("app.fs.path.cache").String()
		tmpPath := sess.Get("app.fs.path.tmp").String()

		if cacheSize, err := fsutils.DirSize(cachePath); err != nil {
			sess.Log().Error("failed to get cache size", slog.String("err", err.Error()))
		} else {
			_ = sess.stats.Set("app.fs.cache.size", humanize.Bytes(uint64(cacheSize)))
		}

		if tmpSize, err := fsutils.DirSize(tmpPath); err != nil {
			sess.Log().Error("failed to get tmp size", slog.String("err", err.Error()))
		} else {
			_ = sess.stats.Set("app.fs.tmp.size", humanize.Bytes(uint64(tmpSize)))
		}

		if availableSpace, err := fsutils.AvailableSpace(cachePath); err != nil {
			sess.Log().Error("failed to get available space", slog.String("err", err.Error()))
		} else {
			_ = sess.stats.Set("app.fs.available", humanize.Bytes(uint64(availableSpace)))
		}

		return nil
	})

	return stats
}
