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
	"github.com/mkungla/vars/v5"
	// "github.com/mkungla/vars/v5"
)

type Manager struct {
	mu       sync.Mutex
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
	sm.running = false
	sm.cancel()
	sm.exitwg.Wait()
	return nil
}

func (sm *Manager) Len() int {
	return len(sm.services)
}

func (sm *Manager) Register(ns string, services ...happy.Service) error {
	if sm.running {
		return errors.New("service manager already running")
	}
	var err error
	for _, service := range services {
		id := fmt.Sprintf("%p", service)
		if _, registered := sm.registry.Load(id); registered {
			err = fmt.Errorf("%w: service %q already registered", ErrServiceRegister, service.Name())
			continue
		}
		if !config.ValidSlug(service.Slug()) {
			err = fmt.Errorf("%w: invalid service slug %q", ErrServiceRegister, service.Slug())
			continue
		}
		sm.registry.Store(id, &Status{
			Registered: time.Now().UnixNano(),
			URL:        fmt.Sprintf("happy://%s/services/%s", ns, service.Slug()),
		})
		sm.services = append(sm.services, service)
	}
	return err
}

func (sm *Manager) Initialize(ctx happy.Session, keepAlive bool) error {
	if sm.running {
		return errors.New("service manager already running")
	}
	for _, service := range sm.services {
		ctx.Log().Experimentalf("inititialize service: %s - %s", service.Slug(), service.Version())

		id := fmt.Sprintf("%p", service)
		if status, registered := sm.registry.Load(id); registered {
			if err := service.Initialize(ctx); err != nil {
				return err
			}
			s, _ := status.(*Status)
			s.Initialized = time.Now().UnixNano()
			ctx.Set(fmt.Sprintf("service.%s", s.URL), false)
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
		go func() {
			defer ctx.TaskDone()
			ctx.RequireService(serviceURL)
		}()
	}
}

func (sm *Manager) Tick(ctx happy.Session) {
	if !sm.running {
		return
	}

	evks := ctx.Events()
	if len(evks) == 0 {
		return
	}

	sm.mu.Lock()

	events := make(map[string][]vars.Variable)
	for _, ev := range evks {
		payload, err := ctx.GetEventPayload(ev)
		if err != nil {
			ctx.Log().Errorf(err.Error())
			continue
		}
		events[ev] = payload
		ctx.Log().SystemDebugf("event: %s - payload (%d)", ev, len(payload))
	}

	// events what service manager is listening
	for ev, pl := range events {
		go sm.OnEvent(ctx, ev, pl)
	}

}

func (sm *Manager) OnEvent(ctx happy.Session, ev string, payload []vars.Variable) {
	// switch ev {
	// case "services.enable":
	// }
	if ev != "happy.services.enable" {
		return
	}

	for _, urivar := range payload {
		u, err := url.Parse(urivar.Key())
		if err != nil {
			ctx.Log().Error(err)
			continue
		}
		// opts := u.Query()
		u.RawQuery = ""

		for _, service := range sm.services {
			id := fmt.Sprintf("%p", service)
			if status, registered := sm.registry.Load(id); registered {
				s, _ := status.(*Status)
				if s.URL == u.String() {
					sm.StartService(ctx, id)
				}
			}
		}
	}
}

func (sm *Manager) StartService(ctx happy.Session, id string) {
	status, registered := sm.registry.Load(id)
	if !registered {
		ctx.Log().Errorf("no service registered with id %s", id)
		return
	}

	var service happy.Service

	for _, srv := range sm.services {
		sid := fmt.Sprintf("%p", srv)
		if sid == id {
			service = srv
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
	s.Started = time.Now().UnixNano()
	s.Running = true
	if err := ctx.Store(fmt.Sprintf("service.%s", s.URL), true); err != nil {
		ctx.Log().Error(err)
		return
	}

	sm.exitwg.Add(1)
	go func(ctx happy.Session, srv happy.Service) {
		defer sm.exitwg.Done()

		ctx.Log().Debugf("starting service %q", service.Slug())

		var prevts time.Time
	bgticker:
		for {
			select {
			case <-sm.ctx.Done():
				break bgticker
			default:
				ts := time.Now()
				delta := ts.Sub(prevts)
				service.Tick(ctx, ts, delta)
				prevts = ts
				time.Sleep(time.Microsecond * 100)
			}
		}

		ctx.Log().Debugf("stopping service %q ", service.Slug())
		if err := service.Stop(ctx); err != nil {
			ctx.Log().Error(err.Error())
		}
	}(ctx, service)
}
