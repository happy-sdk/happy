// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"context"
	"fmt"
	"sync"
	"time"

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

	appLoopReadyCallback sync.Once
	appLoopOK            bool
	appLoopCancel        context.CancelFunc
	appLoopContext       context.Context

	evCancel  context.CancelFunc
	evContext context.Context

	services []Service
}

func (e *Engine) start(sess *Session) error {
	e.started = time.Now()
	if e.tickAction == nil && e.tockAction != nil {
		return fmt.Errorf("%w: register tick action or move tock logic into tick action", ErrEngine)
	}
	var init sync.WaitGroup

	e.startApploop(&init, sess)

	e.initServices(&init, sess)

	init.Wait()

	if e.appLoopOK {
		sess.setReady()
	} else {
		sess.Destroy(fmt.Errorf("%w: starting engine failed", ErrEngine))
	}

	e.startEventDispatcher(sess)
	e.running = true
	return nil
}

func (e *Engine) uptime() time.Duration {
	return time.Since(e.started)
}

func (e *Engine) onTick(action ActionTick) {
	return
}
func (e *Engine) onTock(action ActionTick) {
	return
}

func (e *Engine) startEventDispatcher(sess *Session) {
	e.evContext, e.evCancel = context.WithCancel(sess)

	go func(sess *Session) {
		events := sess.events()
	evLoop:
		for {
			select {
			case <-e.evContext.Done():
				break evLoop
			case ev, ok := <-events:
				if !ok {
					continue
				}
				switch ev.Scope() {
				case "session":
					switch ev.Key() {
					case "require.services":
						payload := ev.Payload()
						if payload != nil {
							payload.Range(func(v vars.Variable) bool {
								e.startService(sess, v.String())
								return true
							})
							continue
						}
					}
				}
				sess.Log().SystemDebug("event", slog.String("scope", ev.Scope()), slog.String("key", ev.Key()))
			}
		}
	}(sess)
}

func (e *Engine) startService(sess *Session, svcurl string) {
	sess.Log().NotImplemented("startService")
}

func (e *Engine) startApploop(init *sync.WaitGroup, sess *Session) {
	sess.Log().SystemDebug("starting engine ...")

	e.appLoopContext, e.appLoopCancel = context.WithCancel(sess)

	if e.tickAction == nil && e.tockAction == nil {
		sess.Log().SystemDebug("engine loop skipped")
		e.appLoopOK = true
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
			case <-e.appLoopContext.Done():
				sess.Log().SystemDebug("engineLoop appLoopContext Done")
				break engineLoop
			case now := <-ttick.C:
				// mark engine running only if first tick tock are successful
				e.appLoopReadyCallback.Do(func() {
					sess.Log().SystemDebug("engine started")

					e.mu.Lock()
					e.appLoopOK = true
					e.mu.Unlock()

					init.Done()
				})

				delta := time.Since(lastTick)
				lastTick = now
				if err := e.tickAction(sess, lastTick, delta); err != nil {
					sess.Log().Error("tick error", err)
					sess.Dispatch(event("app.tick.err", err, nil))
					break engineLoop
				}
				tickDelta := time.Since(lastTick)
				if err := e.tockAction(sess, lastTick, tickDelta); err != nil {
					sess.Log().Error("tock error", err)
					sess.Dispatch(event("app.tock.err", err, nil))
					break engineLoop
				}

			}
		}
		sess.Log().SystemDebug("engine loop stopped")
	}()
}

func (e *Engine) initServices(init *sync.WaitGroup, sess *Session) {
	if e.services == nil {
		sess.Log().SystemDebug("no services to initialize ...")
		return
	}
	sess.Log().SystemDebug("initialize services ...")
}

func (e *Engine) stop(sess *Session) error {
	if !e.running {
		return nil
	}
	sess.Log().SystemDebug("stopping engine")

	e.appLoopCancel()
	<-e.appLoopContext.Done()

	e.evCancel()
	<-e.evContext.Done()

	sess.Log().SystemDebug("engine stopped")
	return nil
}

func event(key string, err error, payload *vars.Map) Event {
	return &EngineEvent{
		ts:      time.Now(),
		key:     key,
		err:     err,
		payload: payload,
	}
}

type EngineEvent struct {
	ts      time.Time
	key     string
	err     error
	payload *vars.Map
}

func (ev *EngineEvent) Time() time.Time {
	return ev.ts
}

func (ev *EngineEvent) Scope() string {
	return "engine"
}

func (ev *EngineEvent) Key() string {
	return ev.key
}
func (ev *EngineEvent) Err() error {
	return ev.err
}

func (ev *EngineEvent) Payload() *vars.Map {
	return ev.payload
}
