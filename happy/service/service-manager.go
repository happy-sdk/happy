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

package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/vars/v6"
)

type Manager struct {
	mu       sync.RWMutex
	exitwg   sync.WaitGroup
	services []happy.Service
	registry sync.Map
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewManager() *Manager {
	return &Manager{}
}

func (sm *Manager) Stop() error {
	if !sm.running {
		return nil
	}
	sm.running = false
	sm.cancel()
	sm.exitwg.Wait()
	return nil
}

func (sm *Manager) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.services)
}

func (sm *Manager) Register(ns string, services ...happy.Service) error {
	if sm.running {
		return errors.New("service manager already running")
	}
	var err error
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, service := range services {

		if !config.ValidSlug(service.Slug()) {
			err = fmt.Errorf("%w: invalid service slug %q", ErrServiceRegister, service.Slug())
			break
		}

		if _, registered := sm.registry.Load(service.Slug()); registered {
			err = fmt.Errorf("%w: service %q already registered", ErrServiceRegister, service.Slug())
			continue
		}

		sm.services = append(sm.services, service)

		sm.registry.Store(service.Slug(), &Status{
			Registered: time.Now().UnixNano(),
			URL:        fmt.Sprintf("happy://%s/services/%s", ns, service.Slug()),
		})
	}
	return err
}

func (sm *Manager) Initialize(ctx happy.Session, keepAlive bool) error {
	if sm.running {
		return errors.New("service manager already running")
	}

	for _, service := range sm.services {
		ctx.Log().SystemDebugf("inititialize service: %s - %s", service.Slug(), service.Version())

		if status, registered := sm.registry.Load(service.Slug()); registered {
			if err := service.Initialize(ctx); err != nil {
				return err
			}
			s, _ := status.(*Status)
			if err := ctx.Set(fmt.Sprintf("service.%s", s.URL), false); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sm *Manager) Start(ctx happy.Session, srvurls ...string) {
	if sm.running {
		ctx.Log().Error("service manager already running")
		return
	}
	if len(srvurls) == 0 {
		return
	}

	ctx.TaskAdd("starting service manager")
	sm.running = true
	sm.ctx, sm.cancel = context.WithCancel(ctx)
	ctx.TaskDone()

	for _, serviceURL := range srvurls {
		ctx.TaskAddf("require service: %s", serviceURL)
		go func(u string) {
			defer ctx.TaskDone()
			ctx.RequireService(u)
		}(serviceURL)
	}
}

func (sm *Manager) Tick(ctx happy.Session) {
	if !sm.running {
		return
	}

	evs := ctx.Events()
	if len(evs) == 0 {
		return
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()
	// events what service manager is listening
	for _, ev := range evs {
		sm.OnEvent(ctx, ev)

		for _, service := range sm.services {
			go service.OnEvent(ctx, ev)
		}
	}
}

func (sm *Manager) OnEvent(ctx happy.Session, ev happy.Event) {
	if ev.Key != "happy.services.enable" {
		return
	}

	ev.Payload.Range(func(key string, val vars.Value) bool {
		u, err := url.Parse(key)
		if err != nil {
			ctx.Log().Error(err)
			return false
		}
		// opts := u.Query()
		u.RawQuery = ""
		for _, service := range sm.services {

			if status, registered := sm.registry.Load(service.Slug()); registered {
				s, _ := status.(*Status)
				if s.URL == u.String() {
					ctx.Log().SystemDebug("start service: ", s.URL)
					sm.StartService(ctx, service.Slug())
				}
			}
		}

		return true
	})
}

func (sm *Manager) StartService(ctx happy.Session, id string) {

	status, registered := sm.registry.Load(id)
	if !registered {
		ctx.Log().Errorf("no service registered with id %s", id)
		return
	}

	var service happy.Service

	for i := range sm.services {
		if sm.services[i].Slug() == id {
			service = sm.services[i]
			break
		}
	}
	if service == nil {
		ctx.Log().Errorf("unable to find service registered with id %s", id)
		return
	}

	if err := service.Start(ctx); err != nil {
		ctx.Log().Error(err)
		return
	}

	s, _ := status.(*Status)
	// s.Started = time.Now().UnixNano()
	// s.Running = true
	ctx.Log().SystemDebugf("starting service %q", service.Slug())
	if err := ctx.Store(fmt.Sprintf("service.%s", s.URL), true); err != nil {
		ctx.Log().Error(err)
		return
	}

	sm.exitwg.Add(1)
	go func(ctx happy.Session, srv happy.Service) {
		defer sm.exitwg.Done()

		var prevts time.Time
	bgticker:
		for {
			select {
			case <-sm.ctx.Done():
				break bgticker
			default:
				ts := time.Now()
				delta := ts.Sub(prevts)
				if err := service.Tick(ctx, ts, delta); err != nil {
					ctx.Log().Error(err)
					break bgticker
				}
				prevts = ts
				// tick throttle
				time.Sleep(time.Microsecond * 100)
			}
		}

		ctx.Log().Debugf("stopping service %q ", service.Slug())
		if err := service.Stop(ctx); err != nil {
			ctx.Log().Error(err)
		}
	}(ctx, service)
}

func (sm *Manager) ServiceCall(serviceUrl, fnName string, args ...vars.Variable) (any, error) {
	u, err := url.Parse(serviceUrl)
	if err != nil {
		return nil, err
	}

	u.RawQuery = ""
	for _, service := range sm.services {
		if status, registered := sm.registry.Load(service.Slug()); registered {
			s, _ := status.(*Status)
			if s.URL == u.String() {
				return service.Call(fnName, args...)
			}
		}
	}

	return nil, fmt.Errorf("failed to exec service call func %s (%s)", fnName, serviceUrl)
}
