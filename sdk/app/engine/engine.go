// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/networking/address"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/services"
	"github.com/happy-sdk/happy/sdk/services/service"
	"github.com/happy-sdk/happy/sdk/session"
	"github.com/happy-sdk/happy/sdk/stats"
)

var (
	Error                = fmt.Errorf("engine")
	ErrServiceTerminated = fmt.Errorf("%w: service terminated", Error)
	ErrEngineStopped     = fmt.Errorf("%w stopped", Error)
)

type engineState int

const (
	engineInit engineState = iota
	engineStarting
	engineRunning
	engineStopping
	engineStopped
	engineFailed
)

func (es engineState) String() string {
	switch es {
	case engineInit:
		return "init"
	case engineStarting:
		return "starting"
	case engineRunning:
		return "running"
	case engineStopping:
		return "stopping"
	case engineStopped:
		return "stopped"
	case engineFailed:
		return "failed"
	}
	return "unknown"
}

type Engine struct {
	mu            sync.RWMutex
	readyCallback sync.Once
	state         engineState
	engineOK      bool

	tick action.Tick
	tock action.Tock

	engineLoopCancel context.CancelFunc
	engineLoopCtx    context.Context

	eventLoopCancel      context.CancelFunc
	eventLoopCtx         context.Context
	eventLoopShutdownCtx context.Context
	evch                 <-chan events.Event
	events               map[string]bool
	gsd                  *gracefulShutdown

	registry map[string]*services.Container

	stats *stats.Profiler
	errs  []error
}

func New(evch <-chan events.Event, tick action.Tick, tock action.Tock) *Engine {
	e := &Engine{
		tick:     tick,
		tock:     tock,
		evch:     evch,
		events:   make(map[string]bool),
		registry: make(map[string]*services.Container),
		gsd:      newGracefulShutdown(),
		stats:    stats.New("app-stats"),
	}

	var sysevs = []events.Event{
		services.StartEvent,
		services.StopEvent,
		service.StartedEvent,
		service.StoppedEvent,
	}

	for _, sev := range sysevs {
		if err := e.listenEvent(sev.Scope(), sev.Key()); err != nil {
			e.errs = append(e.errs, err)
		}
	}

	return e
}

func (e *Engine) Start(sess *session.Context) error {
	e.mu.RLock()
	state := e.state
	e.mu.RUnlock()

	if state != engineInit {
		return fmt.Errorf("%w: can not start engine %s", Error, state.String())
	}
	internal.Log(sess.Log(), "starting engine ...")

	e.mu.Lock()
	e.state = engineStarting
	tick := e.tick
	tock := e.tock
	e.mu.Unlock()

	if tick == nil && tock != nil {
		return fmt.Errorf("%w: register tick action or move tock logic into tick action", Error)
	}
	if sess.Get("app.stats.enabled").Bool() {
		e.mu.Lock()
		statsSvc := stats.AsService(e.stats)
		e.mu.Unlock()
		if err := sess.AttachAPI("app.stats", e.stats); err != nil {
			return err
		}
		if err := e.RegisterService(sess, statsSvc); err != nil {
			return err
		}
	}

	var init sync.WaitGroup

	e.loopStart(sess, &init)
	e.servicesInit(sess, &init)

	init.Wait()

	e.mu.Lock()
	var failed bool
	if len(e.errs) > 0 {
		for _, err := range e.errs {
			if err != nil {
				failed = true
				sess.Log().Error(err.Error())
			}
		}
	}
	if failed {
		state = engineFailed
	} else {
		state = engineRunning
	}
	e.state = state
	e.stats.Update()
	e.mu.Unlock()

	if state == engineRunning {
		e.startEventDispatcher(sess)
	} else {
		sess.Destroy(fmt.Errorf("%w: starting engine failed: state %s", Error, state.String()))
	}

	if sess.Get("app.stats.enabled").Bool() {
		loader := services.NewLoader(sess, "app-runtime-stats")
		<-loader.Load()
		if err := loader.Err(); err != nil {
			return err
		}
	}

	internal.Log(sess.Log(), "engine started", slog.String("state", state.String()))
	return nil
}

func (e *Engine) Stop(sess *session.Context) error {
	e.mu.RLock()
	state := e.state
	e.mu.RUnlock()
	if state != engineRunning {
		return nil
	}
	e.mu.Lock()
	e.state = engineStopping
	gsd := e.gsd
	e.mu.Unlock()

	internal.Log(sess.Log(), "stopping engine ...")

	e.engineLoopCancel()

	internal.Log(sess.Log(), "stopping engineLoopCancel ...")

	var (
		quarantine           map[string]*services.Container
		totalRunningServices atomic.Int64
	)

	for u, rsvc := range e.registry {
		if rsvc.IsLocked() {
			time.Sleep(time.Second)
			if rsvc.IsLocked() {
				gsd.Add(1)
				totalRunningServices.Add(1)
				sess.Log().Warn(fmt.Sprintf("%s service quarantined, still busy, or possibly deadlocked.", u))
				if quarantine == nil {
					quarantine = make(map[string]*services.Container)
				}
				quarantine[u] = rsvc
				continue
			}
		}
		if !rsvc.Info().Running() {
			internal.Log(sess.Log(), fmt.Sprintf("service already stopped %s", rsvc.Info().Slug()))
			continue
		}
		totalRunningServices.Add(1)

		internal.Log(sess.Log(), fmt.Sprintf("shutdown %s", rsvc.Info().Slug()))
		gsd.Add(1)
		go func(url string, svcc *services.Container) {
			defer gsd.Done()
			// wait for iengine context is canceled which triggers
			// r.ctx also to be cancelled, however lets wait for the
			// context done since r.ctx is cancelled after last tickk completes.
			// so e.xtc is not parent of r.ctx.
			internal.Log(sess.Log(), fmt.Sprintf("waiting for %s to prepare shutdown", svcc.Info().Slug()))
			<-svcc.Done()
			// lets call stop now we know that tick loop has exited.
			e.serviceStop(sess, url, ErrServiceTerminated)
			totalRunningServices.Add(-1)
		}(u, rsvc)
	}

	trs := totalRunningServices.Load()
	if trs > 0 {
		internal.Log(sess.Log(), fmt.Sprintf("waiting for %d services to stop", trs))
	}

	errChan := make(chan error, len(quarantine))
	if len(quarantine) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Overall deadline
		defer cancel()

		for u, container := range quarantine {
			go func(url string, svcc *services.Container) {
				defer gsd.Done()
				defer totalRunningServices.Add(-1)
				const maxAttempts = 4
				for attempt := range maxAttempts {
					if !svcc.IsLocked() {
						sess.Log().Info(fmt.Sprintf("service %s released lock after %d attempts", svcc.Info().Slug(), attempt))
						e.serviceStop(sess, url, ErrServiceTerminated)
						return
					}
					if attempt == maxAttempts {
						break
					}
					backoff := time.Duration(10*math.Pow(10, float64(attempt))) * time.Millisecond
					sess.Log().Notice(fmt.Sprintf("service %s still locked, backing off %v (attempt %d/%d)",
						url, backoff, attempt+1, maxAttempts))

					select {
					case <-time.After(backoff):
						continue // Retry
					case <-ctx.Done():
						errChan <- fmt.Errorf("service %s: termination deadline reached", url)
						return
					}
				}

				// Still locked after all attempts - force shutdown
				if container.IsLocked() {
					sess.Log().Warn(fmt.Sprintf("service %s still locked after %d attempts, forcing shutdown", url, maxAttempts))
					if err := container.ForceShutdown(sess, ErrServiceTerminated); err != nil {
						errChan <- err
					} else {
						sess.Log().Ok(fmt.Sprintf("service %s force shutdown completed", url))
					}
				} else {
					sess.Log().Info(fmt.Sprintf("service %s released lock right before force shutdown", svcc.Info().Slug()))
					e.serviceStop(sess, url, ErrServiceTerminated)
				}
			}(u, container)
		}
	}
	internal.Log(sess.Log(), "waiting for engine to stop")
	// Wait for completion and handle any timeout errors
	go func() {
		gsd.Wait()
		close(errChan)
	}()
	for err := range errChan {
		sess.Log().Error(err.Error())
	}

	e.mu.Lock()
	e.state = engineStopped
	e.mu.Unlock()

	// Consumes all events from the event channel after all services are stopped.
	// This is to ensure that no events are lost.
	if e.evch != nil {
		e.eventLoopCancel()
		<-e.eventLoopShutdownCtx.Done()
	}
	internal.Log(sess.Log(), "engine stopped")
	return nil
}

func (e *Engine) Stats() *stats.Profiler {
	e.mu.RLock()
	defer e.mu.RUnlock()
	stats := e.stats
	return stats
}

func (e *Engine) loopStart(sess *session.Context, init *sync.WaitGroup) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.engineLoopCtx, e.engineLoopCancel = context.WithCancel(sess)

	if e.tick == nil && e.tock == nil {
		internal.Log(sess.Log(), "engine loop skipped")
		return
	}
	if e.tock == nil {
		e.tock = nooptock
	}

	init.Add(2)
	defer init.Done()

	e.gsd.Add(1)
	go func() {

		defer func() {
			e.gsd.Done()

			if r := recover(); r != nil {
				// Log the panic message
				var errMessage string
				if err, ok := r.(error); ok {
					errMessage = err.Error()
				} else {
					errMessage = fmt.Sprintf("%v", r)
				}

				stack := debug.Stack()
				// Obtain and log the stack trace
				stackTrace := string(stack)

				sess.Log().LogDepth(2, logging.LevelBUG, "panic: engine loop (recovered)",
					slog.String("msg", errMessage),
				)
				sess.Log().LogDepth(2, logging.LevelAlways, stackTrace)
				sess.Destroy(fmt.Errorf("%w: engine loop panic", Error))
			}

		}()
		e.readyCallback.Do(func() {
			e.mu.Lock()
			e.engineOK = true
			e.mu.Unlock()
			init.Done()
			internal.Log(sess.Log(), "engine loop initialized")
		})

		// start when session is ready
	waitStart:
		for {
			select {
			case <-sess.Ready():
				break waitStart
			case <-e.engineLoopCtx.Done():
				return
			case <-sess.Done():
				return
			}
		}

		throttle := time.Duration(sess.Get("app.engine.throttle_ticks").Int64())
		lastTick := sess.Time(time.Now())
		ttick := time.NewTicker(throttle)
		defer ttick.Stop()

		tps := 0
		tpsEnabled := throttle < time.Second
		const tpsSize = 120 // size of the tick delta array
		var tickDeltas [tpsSize]time.Duration
		var tdi int           // tick delta index
		var tds time.Duration // tick delta sum
		if tpsEnabled {
			initialDelta := throttle
			for i := 0; i < tpsSize; i++ {
				tickDeltas[i] = initialDelta
				tds += initialDelta
			}
		}

		internal.Log(sess.Log(), "engine loop started")

	engineLoop:
		for {
			select {
			case <-e.engineLoopCtx.Done():
				break engineLoop
			case now := <-ttick.C:
				now = sess.Time(now)
				delta := now.Sub(lastTick)
				lastTick = now
				if err := e.tick(sess, lastTick, delta); err != nil {
					sess.Log().Error("engine tick error", slog.String("err", err.Error()))
					sess.Dispatch(events.New("engine", "tick.error").Create(err, nil))
					break engineLoop
				}

				if tpsEnabled {
					// Update the sliding window of frame times
					otd := tickDeltas[tdi] // oldest tick delta
					tickDeltas[tdi] = delta
					tds += delta - otd
					tdi = (tdi + 1) % tpsSize
					atd := tds / tpsSize // average tick delta
					tps = int(math.Round(float64(time.Second) / float64(atd)))
				}

				tickDelta := time.Since(lastTick)
				if err := e.tock(sess, tickDelta, tps); err != nil {
					sess.Log().Error("tock error", slog.String("err", err.Error()))
					sess.Dispatch(events.New("engine", "tock.error").Create(err, nil))
					break engineLoop
				}
			}
		}
		internal.Log(sess.Log(), "engine loop stopped")
	}()
}

func (e *Engine) servicesInit(sess *session.Context, init *sync.WaitGroup) {
	e.mu.Lock()
	svccount := len(e.registry)
	e.mu.Unlock()
	if svccount == 0 {
		internal.Log(sess.Log(), "no services to initialize ...")
		return
	}

	internal.Log(sess.Log(), "initialize services", slog.Int("count", svccount))

	init.Add(svccount)
	for svcaddrstr, svcc := range e.registry {
		go func(addr string, c *services.Container) {
			defer init.Done()
			if err := c.Register(sess); err != nil {
				sess.Log().Error(
					err.Error(),
					slog.String("service", c.Info().Addr().String()),
					slog.String("err", "failed to initialize service"))
				return
			}
			// register events what service listens for
			for _, ev := range c.Listeners() {
				scope, key, _ := strings.Cut(ev, ".")
				// we can ignore error because this error is handled
				// when emitter registers this event. Listening for unregistered event is not an error.
				_ = e.listenEvent(scope, key)
			}
		}(svcaddrstr, svcc)
		e.stats.Update()
	}
}

func (e *Engine) startEventDispatcher(sess *session.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()
	internal.Log(sess.Log(), "starting engine event dispatcher")

	if e.evch == nil {
		sess.Log().Warn("event channel is nil, skipping event dispatcher")
		return
	}

	e.eventLoopCtx, e.eventLoopCancel = context.WithCancel(context.Background())
	var eventLoopShutdownComplete context.CancelFunc
	e.eventLoopShutdownCtx, eventLoopShutdownComplete = context.WithCancel(context.Background())

	go func(sess *session.Context) {
		defer eventLoopShutdownComplete()
	evLoop:
		for {
			select {
			case <-e.eventLoopCtx.Done():
				for {
					select {
					case ev, ok := <-e.evch:
						if !ok {
							break evLoop
						}
						e.handleEvent(sess, ev)
					default:
						break evLoop
					}
				}
			case ev, ok := <-e.evch:
				if !ok {
					continue
				}
				e.handleEvent(sess, ev)
			}
		}
		internal.Log(sess.Log(), "engine event dispatcher stopped")
	}(sess)
}

func (e *Engine) handleEvent(sess *session.Context, ev events.Event) {
	skey := ev.Scope() + "." + ev.Key()

	e.mu.RLock()
	_, ok := e.events[skey]
	registry := e.registry
	e.mu.RUnlock()

	if len(skey) == 1 || !ok {
		sess.Log().NotImplemented("event not registered, ignoring", slog.String("scope", ev.Scope()), slog.String("key", ev.Key()))
		return
	}

	if ev.Value() == vars.NilValue {
		sess.Log().Warn(fmt.Sprintf("event(%s.%s)", ev.Scope(), ev.Key()), slog.String("value", ev.Value().String()))
	} else {
		internal.Log(sess.Log(), fmt.Sprintf("event(%s.%s)", ev.Scope(), ev.Key()), slog.String("value", ev.Value().String()))
	}

	switch ev.Scope() {
	case "services":
		switch ev.Key() {
		case services.StartEvent.Key():
			payload := ev.Payload()
			if payload != nil {
				payload.Range(func(v vars.Variable) bool {
					go e.serviceStart(sess, v.String())
					return true
				})
			}
			if ev.Value().Kind() != vars.KindString {
				sess.Log().Warn(fmt.Sprintf("start.services event is not addressable, ignoring: %s", ev.Value().String()))
				return
			}
			if ev.Value() == vars.NilValue {
				sess.Log().Warn("start.services event has no payload, ignoring")
				return
			}
			hostaddr, err := address.Parse(sess.Get("app.address").String())
			if err != nil {
				sess.Log().Error("failed to parse app address", slog.String("err", err.Error()))
				return
			}
			if ev.Value().String() != "bundle" {
				addr, err := hostaddr.ResolveService(ev.Value().String())
				if err != nil {
					sess.Log().Error("failed to resolve service address", slog.String("err", err.Error()))
					return
				}
				sess.Log().Debug("starting service", slog.String("service", addr.String()))
				go e.serviceStart(sess, addr.String())
			}

		case services.StopEvent.Key():
			payload := ev.Payload()
			if payload != nil {
				payload.Range(func(v vars.Variable) bool {
					go e.serviceStop(sess, v.String(), ErrServiceTerminated) // prevents restart
					return true
				})
			}
			if ev.Value() == vars.NilValue {
				sess.Log().Warn("stop.services event has no payload, ignoring")
				return
			}
			hostaddr, err := address.Parse(sess.Get("app.address").String())
			if err != nil {
				sess.Log().Error("failed to parse app address", slog.String("err", err.Error()))
				return
			}
			addr, err := hostaddr.ResolveService(ev.Value().String())
			if err != nil {
				sess.Log().Error("failed to resolve service address", slog.String("err", err.Error()))
				return
			}
			go e.serviceStop(sess, addr.String(), ErrServiceTerminated) // prevents restart
		}
	}
	for _, svcc := range registry {
		go svcc.HandleEvent(sess, ev)
	}

}

func (e *Engine) RegisterService(sess *session.Context, svc *services.Service) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if svc == nil {
		return fmt.Errorf("%w: attempt to register <nil> service", Error)
	}

	if e.state == engineRunning {
		return fmt.Errorf("%w: can not register services engine is already running - %s", Error, svc.Slug())
	}

	hostaddr, err := address.Parse(sess.Get("app.address").String())
	if err != nil {
		return fmt.Errorf("%w:%s", Error, err.Error())
	}
	addr, err := hostaddr.ResolveService(svc.Slug())
	if err != nil {
		return err
	}

	addrstr := addr.String()
	if _, ok := e.registry[addrstr]; ok {
		return fmt.Errorf("%w: services is already registered %s", Error, addr)
	}

	container, err := services.NewContainer(sess, addr, svc)
	if err != nil {
		return fmt.Errorf("%w: %s", Error, err.Error())
	}
	e.registry[addrstr] = container

	internal.Log(sess.Log(), "service registered", slog.String("service", svc.Slug()))
	return nil
}

func (e *Engine) RegisterEvent(ev events.Event) error {
	return e.listenEvent(ev.Scope(), ev.Key())
}

func (e *Engine) listenEvent(scope, key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	skey := scope + "." + key
	if _, ok := e.events[skey]; ok {
		return fmt.Errorf("%w: event already registered %s", Error, skey)
	}
	e.events[skey] = true
	return nil
}

func (e *Engine) serviceStart(sess *session.Context, svcurl string) {
	svcaddr, err := address.Parse(svcurl)
	if err != nil {
		sess.Log().Error(err.Error())
		return
	}
	e.mu.RLock()

	svcc, ok := e.registry[svcaddr.Path()]
	e.mu.RUnlock()
	if !ok {
		sess.Log().Warn("no such service to start", slog.String("service", svcaddr.String()))
		return
	}
	if svcc.Info().Failed() {
		sess.Log().NotImplemented("skip starting service due previous errors", slog.String("service", svcaddr.String()))
		return
	}

	sarg := slog.String("service", svcaddr.String())
	if !ok {
		sess.Log().Warn(
			"requested unknown service",
			sarg,
		)
		return
	}
	if svcc.Info().Running() {

		sess.Log().Warn(
			"failed to start service, service already running",
			sarg,
		)
		return
	}

	// Update service address with query parameters when start requetsed
	service.SetFullStartAddr(svcc.Info(), svcaddr)

	if err := svcc.Start(e.engineLoopCtx, sess); err != nil {
		sess.Log().Error(
			"failed to start service",
			slog.String("err", err.Error()),
			sarg,
		)
		if e.state == engineRunning && svcc.CanRetry() && sess.CanRecover(nil) {
			sess.Log().Notice("retrying to start the service 1", sarg, slog.Int("retry", svcc.Retries()))
			e.serviceStart(sess, svcaddr.String())
		}
		return
	}

	go func(svcc *services.Container, svcaddr *address.Address, sarg slog.Attr) {

		if !svcc.HasTick() {
			<-e.engineLoopCtx.Done()
			svcc.Cancel(ErrEngineStopped)
			return
		}

		throttle := time.Duration(sess.Get("app.engine.throttle_ticks").Int64())
		if svcc.Settings().ThrottleTicks > 0 {
			throttle = time.Duration(svcc.Settings().ThrottleTicks)
		}
		lastTick := sess.Time(time.Now())
		ttick := time.NewTicker(throttle)
		defer ttick.Stop()

		tps := 0
		tpsEnabled := throttle < time.Second
		const tpsSize = 120 // size of the tick delta array
		var tickDeltas [tpsSize]time.Duration
		var tdi int           // tick delta index
		var tds time.Duration // tick delta sum
		if tpsEnabled {
			initialDelta := throttle
			for i := range tpsSize {
				tickDeltas[i] = initialDelta
				tds += initialDelta
			}
		}

	ticker:
		for {
			select {
			case <-svcc.Done():
				break ticker
			case now := <-ttick.C:
				now = sess.Time(now)
				delta := now.Sub(lastTick)
				lastTick = now

				if err := svcc.Tick(sess, lastTick, delta); err != nil {
					e.serviceStop(sess, svcaddr.Path(), err)
					break ticker
				}

				if tpsEnabled {
					// Update the sliding window of frame times
					otd := tickDeltas[tdi] // oldest tick delta
					tickDeltas[tdi] = delta
					tds += delta - otd
					tdi = (tdi + 1) % tpsSize
					atd := tds / tpsSize // average tick delta
					tps = int(math.Round(float64(time.Second) / float64(atd)))
				}

				tickDelta := time.Since(lastTick)
				if err := svcc.Tock(sess, tickDelta, tps); err != nil {
					e.serviceStop(sess, svcaddr.Path(), err)
					break ticker
				}
			}
		}
	}(svcc, svcaddr, sarg)
}

func (e *Engine) serviceStop(sess *session.Context, svcurl string, err error) {
	sarg := slog.String("service", svcurl)

	e.mu.RLock()
	svcc, ok := e.registry[svcurl]
	e.mu.RUnlock()
	if !ok {
		sess.Log().Warn("no such service to stop", sarg)
		return
	}
	internal.Log(sess.Log(), "stopping service", sarg)
	serr := err
	// When ErrServiceTerminated is encountered, set the error to nil
	// This is to prevent the service from being restarted unnecessarily
	if errors.Is(serr, ErrServiceTerminated) {
		serr = nil
	}
	if stoperr := svcc.Stop(sess, serr); stoperr != nil {
		sess.Log().Error("failed to stop service", slog.String("err", stoperr.Error()), sarg)
	} else {
		if err == nil && e.state == engineRunning && svcc.CanRetry() && sess.CanRecover(nil) {
			sess.Log().Notice("retrying to start the service 2", sarg, slog.Int("retry", svcc.Retries()))
			go e.serviceStart(sess, svcurl)
		}
	}

}

var nooptock = func(*session.Context, time.Duration, int) error { return nil }

type gracefulShutdown struct {
	wg sync.WaitGroup
}

func newGracefulShutdown() *gracefulShutdown {
	return &gracefulShutdown{}
}

func (gsd *gracefulShutdown) Add(delta int) {
	gsd.wg.Add(delta)
}

func (gsd *gracefulShutdown) Done() {
	gsd.wg.Done()
}

func (gsd *gracefulShutdown) Wait() {
	gsd.wg.Wait()
}
