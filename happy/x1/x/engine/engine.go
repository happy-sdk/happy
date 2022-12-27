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

package engine

import (
	"context"
	"sync"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
)

// var ErrEngine = happyx.NewError("engine error")

// // noop for tick | tock
// var ttnoop = func(sess happy.Session, ts time.Time, delta time.Duration) error { return nil }

// type EngineEvent struct {
// 	ts      time.Time
// 	key     string
// 	err     happy.Error
// 	payload happy.Variables
// }

// func (ev *EngineEvent) Time() time.Time {
// 	return ev.ts
// }

// func (ev *EngineEvent) Scope() string {
// 	return "engine"
// }

// func (ev *EngineEvent) Key() string {
// 	return ev.key
// }
// func (ev *EngineEvent) Err() happy.Error {
// 	return ev.err
// }

// func (ev *EngineEvent) Payload() happy.Variables {
// 	return ev.payload
// }

type registeredService struct {
	svc        happy.BackgroundService
	running    bool
	jobCancel  context.CancelFunc
	jobContext context.Context
}

type Engine struct {
	mu                     sync.RWMutex
	tickAction, tockAction happy.ActionTickFunc
	errs                   []happy.Error
	running                bool
	config                 []happy.OptionSetFunc
	opts                   happy.Variables

	appLoopReadyCallback sync.Once
	appLoopOK            bool
	appLoopCancel        context.CancelFunc
	appLoopContext       context.Context

	services []happy.Service
	registry map[string]*registeredService

	// evch      <-chan happy.Event
	// evCancel  context.CancelFunc
	// evContext context.Context

	monitor happy.Monitor
}

func New(opts ...happy.OptionSetFunc) *Engine {
	return &Engine{
		config:   opts,
		registry: make(map[string]*registeredService),
	}
}

func (e *Engine) AttachMonitor(monitor happy.Monitor) happy.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.monitor != nil {
		return ErrEngine.WithText("monitor already attached")
	}
	e.monitor = monitor
	return nil
}

func (e *Engine) Monitor() happy.Monitor {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.monitor
}

func (e *Engine) Register(svcs ...happy.Service) happy.Error {
	if e.running {
		return ErrEngine.WithText("can not register services engine is already running")
	}
	for _, svc := range svcs {
		if svc == nil {
			return ErrEngine.WithText("attempt to register <nil> service")
		}

		e.services = append(e.services, svc)
	}
	return nil
}

var settings = map[string]bool{
	"services.discovery.timeout": true,
}

func (e *Engine) Start(sess happy.Session) happy.Error {

	if e.running {
		return ErrEngine.WithText("engine already started")
	}

	if err := e.monitor.Start(sess); err != nil {
		return err
	}

	opts, err := happyx.OptionsToVariables(e.config...)
	if err != nil {
		return ErrEngine.Wrap(err)
	}
	e.opts = opts

	for _, o := range opts.All() {
		allowed, exists := settings[o.Key()]
		if !exists {
			return ErrEngine.WithTextf("engine does not have option: %s", o.Key())
		}
		if !allowed {
			sess.Log().Deprecatedf("engine setting [%s] is deprecated", o.Key())
		} else {
			key := "engine." + o.Key()
			sess.Settings().LoadOrStore(key, o.Value())
		}
	}

	if len(e.errs) > 0 {
		err := ErrEngine
		for _, e := range e.errs {
			err = err.Wrap(e)
		}
		return err
	}

	// if e.tickAction == nil && e.tockAction != nil {
	// 	return ErrEngine.WithText("register tick action or move tock logic into tick action")
	// }

	// var init sync.WaitGroup

	// e.startApploop(&init, sess)

	// e.initServices(&init, sess)

	// init.Wait()

	// if e.appLoopOK {
	// 	sess.Dispatch(event("ready", nil, nil))
	// } else {
	// 	sess.Destroy(ErrEngine.WithText("starting engine failed"))
	// }

	// e.evch = sess.Events()

	// if e.evch != nil {
	// 	e.startEventDispatcher(sess)
	// }
	// e.running = true
	return nil
}

func (e *Engine) startEventDispatcher(sess happy.Session) {
	e.evContext, e.evCancel = context.WithCancel(sess)
	go func() {
	evLoop:
		for {
			select {
			case <-e.evContext.Done():
				break evLoop
			case ev, ok := <-e.evch:
				if !ok {
					continue
				}
				switch ev.Scope() {
				// case "session":
				// 	switch ev.Key() {
				// 	case "require.services":
				// 		ev.Payload().Range(func(v happy.Variable) bool {
				// 			e.startService(sess, v.String())
				// 			return true
				// 		})
				// 		continue
				// 	}
				// }

				// sess.Log().SystemDebugf("got ev: %s(%s)", ev.Scope(), ev.Key())
				e.mu.RLock()
				registry := e.registry
				e.mu.RUnlock()
				for u, s := range registry {
					if err := s.svc.HandleEvent(sess, ev); err != nil {
						sess.Log().Warnf("%s: %s", u, err)
					}
				}
			}
		}
	}()
}

func (e *Engine) startService(sess happy.Session, svcurl string) {
	u, err := happyx.ParseURL(svcurl)
	if err != nil {
		sess.Log().Errorf("failed to parse service url: %s", err)
		return
	}
	svcurl = u.PeerService()

	e.mu.RLock()
	reg, ok := e.registry[svcurl]
	e.mu.RUnlock()

	if !ok {
		sess.Log().SystemDebugf("requested unknown service: %s", svcurl)
		return
	}

	sess.Log().SystemDebugf("starting service: %s", svcurl)

	if err := reg.svc.Start(sess, u.Args()); err != nil {
		sess.Log().Errorf("failed to start service: %s", err)
		if err := reg.svc.Stop(sess); err != nil {
			sess.Log().Error(err)
		}
		return
	}

	reg.jobContext, reg.jobCancel = context.WithCancel(sess)
	go func(r *registeredService, svcurl string) {
		defer func() {
			e.Monitor().SetServiceStatus(svcurl, "running", false)
			e.Monitor().SetServiceStatus(svcurl, "stopped.at", time.Now())
		}()
		r.running = true
		e.Monitor().SetServiceStatus(svcurl, "running", true)
		e.Monitor().SetServiceStatus(svcurl, "started.at", time.Now())

		lastTick := time.Now()
		kill := func(err happy.Error) {
			sess.Log().Error(err)
			e.Monitor().SetServiceStatus(svcurl, "failed", true)
			e.Monitor().SetServiceStatus(svcurl, "err", err)
			r.jobCancel()
			if err := r.svc.Stop(sess); err != nil {
				sess.Log().Error(err)
			}
		}
		ttick := time.NewTicker(time.Microsecond * 100)
		defer ttick.Stop()

	ticker:
		for {
			select {
			case <-e.appLoopContext.Done():
				r.jobCancel()
				break ticker
			case now := <-ttick.C:
				delta := time.Since(lastTick)
				lastTick = now
				if err := r.svc.Tick(sess, lastTick, delta); err != nil {
					kill(err)
					break ticker
				}
				tickDelta := time.Since(lastTick)
				if err := r.svc.Tock(sess, lastTick, tickDelta); err != nil {
					kill(err)
					break ticker
				}
			}
		}
	}(reg, svcurl)
}

func (e *Engine) initServices(init *sync.WaitGroup, sess happy.Session) {
	// if e.services == nil {
	// 	sess.Log().SystemDebug("no services to initialize ...")
	// 	return
	// }
	// sess.Log().SystemDebug("initialize services ...")
	// init.Add(len(e.services))
	for _, svc := range e.services {

		go func(svc happy.Service) {
			defer init.Done()
			bgsvc, err := svc.Register(sess)
			if err != nil {
				sess.Log().Alertf("failed to register service(%s): %s", svc.Slug(), err)
				return
			}
			u := svc.URL().String()
			if len(u) == 0 {
				sess.Log().Alert("service url empty after registration: ", svc.Slug(), svc.URL())
				return
			}

			e.mu.RLock()
			_, ok := e.registry[u]
			e.mu.RUnlock()

			if ok {
				sess.Log().Alert("service url already in use: ", svc.URL())
				return
			}
			if err := bgsvc.Initialize(sess, e.monitor.Status()); err != nil {
				sess.Log().Alertf("failed to initialize service(%s): %s", svc.Slug(), err)
				return
			}
			e.mu.Lock()

			e.registry[u] = &registeredService{
				svc: bgsvc,
			}

			e.monitor.SetServiceStatus(u, "registered", true)
			e.monitor.SetServiceStatus(u, "running", false)

			sess.Log().SystemDebugf("initialized service: %s(%s)", svc.Slug(), svc.URL())
			e.mu.Unlock()

		}(svc)
	}
}

// func (e *Engine) startApploop(init *sync.WaitGroup, sess happy.Session) {
// 	sess.Log().SystemDebug("starting engine ...")

// 	e.appLoopContext, e.appLoopCancel = context.WithCancel(sess)

// 	if e.tickAction == nil && e.tockAction == nil {
// 		sess.Log().SystemDebug("engine loop skipped")
// 		e.appLoopOK = true
// 		return
// 	}
// 	// we should have enured that tick action is set.
// 	// so only set noop tock if needed
// 	if e.tockAction == nil {
// 		e.tockAction = ttnoop
// 	}

// 	init.Add(2)
// 	defer init.Done()

// 	go func() {
// 		lastTick := time.Now()

// 		ttick := time.NewTicker(time.Microsecond * 100)
// 		defer ttick.Stop()

// 	engineLoop:
// 		for {
// 			select {
// 			case <-e.appLoopContext.Done():
// 				sess.Log().SystemDebug("engineLoop appLoopContext Done")
// 				break engineLoop
// 			case now := <-ttick.C:
// 				// mark engine running only if first tick tock are successful
// 				e.appLoopReadyCallback.Do(func() {
// 					sess.Log().SystemDebug("engine started")

// 					e.mu.Lock()
// 					e.appLoopOK = true
// 					e.mu.Unlock()

// 					init.Done()
// 				})

// 				delta := time.Since(lastTick)
// 				lastTick = now
// 				if err := e.tickAction(sess, lastTick, delta); err != nil {
// 					sess.Log().Alert(err)
// 					sess.Dispatch(event("app.tick.err", ErrEngine.Wrap(err), nil))
// 					break engineLoop
// 				}
// 				tickDelta := time.Since(lastTick)
// 				if err := e.tockAction(sess, lastTick, tickDelta); err != nil {
// 					sess.Log().Alert(err)
// 					sess.Dispatch(event("app.tock.err", ErrEngine.Wrap(err), nil))
// 					break engineLoop
// 				}

// 			}
// 		}
// 		sess.Log().SystemDebug("engine loop stopped")
// 	}()
// }

func (e *Engine) Stop(sess happy.Session) happy.Error {
	// sess.Log().SystemDebug("stopping engine")

	// if !e.running {
	// 	return nil
	// }

	// e.appLoopCancel()
	// <-e.appLoopContext.Done()

	// e.evCancel()
	// <-e.evContext.Done()

	var graceful sync.WaitGroup
	for u, reg := range e.registry {
		if reg.running {
			graceful.Add(1)
			go func(url string, r *registeredService) {
				defer graceful.Done()
				sess.Log().SystemDebugf("stopping service: %s", url)
				<-r.jobContext.Done()
				if err := r.svc.Stop(sess); err != nil {
					sess.Log().Error(err)
				}
			}(u, reg)
		}
	}
	graceful.Wait()

	sess.Log().SystemDebug("engine stopped")
	return nil
}

func (e *Engine) ResolvePeerTo(ns, ipport string) {}

func (e *Engine) OnTick(action happy.ActionTickFunc) {
	if e.tickAction != nil {
		e.errs = append(e.errs, ErrEngine.WithText("attempt to override engine.OnTick action"))
		return
	}
	if action == nil {
		e.errs = append(e.errs, ErrEngine.WithText("provided <nil> action to engine.OnTick"))
		return
	}
	e.tickAction = action
}

func (e *Engine) OnTock(action happy.ActionTickFunc) {
	if e.tockAction != nil {
		e.errs = append(e.errs, ErrEngine.WithText("attempt to override engine.OnTock action"))
		return
	}
	if action == nil {
		e.errs = append(e.errs, ErrEngine.WithText("provided <nil> action to engine.OnTock"))
		return
	}
	e.tockAction = action
}

func (e *Engine) ListenEvents(evch <-chan happy.Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.evch = evch
}

func event(key string, err happy.Error, payload happy.Variables) happy.Event {
	return &EngineEvent{
		ts:      time.Now(),
		key:     key,
		err:     err,
		payload: payload,
	}
}
