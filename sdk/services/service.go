// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package services

import (
	"sync"

	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/services/service"
	"github.com/happy-sdk/happy/sdk/session"
)

type Service struct {
	mu             sync.Mutex
	settings       service.Config
	registerAction action.Action
	startAction    action.Action
	stopAction     action.WithPrevErr
	tickAction     action.Tick
	tockAction     action.Tock
	listeners      map[string][]events.ActionWithEvent[*session.Context]

	cronsetup func(schedule CronScheduler)
	errs      []error
}

type CronScheduler interface {
	Job(name, expr string, cb action.Action)
}

// New cretes new draft service which you can compose
// before passing it to applciation or providing it from addon.
func New(s service.Config) *Service {
	svc := &Service{}

	_, err := s.Blueprint()
	if err != nil {
		svc.errs = append(svc.errs, err)
	}

	svc.settings = s

	return svc
}

func (s *Service) Name() string {
	return s.settings.Name.String()
}

func (s *Service) Slug() string {
	return s.settings.Slug.String()
}

// OnRegister is called when app is preparing runtime and attaching services,
// This does not mean that service will be used or started.
func (s *Service) OnRegister(action action.Action) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registerAction = action
	return s
}

// OnStart is called when service is requested to be started.
// For instace when command is requiring this service or whenever
// service is required on runtime via sess.RequireService call.
//
// Start can be called multiple times in case of service restarts.
// If you do not want to allow service restarts you should implement
// your logic in OnStop when it's called first time and check that
// state OnStart.
func (s *Service) OnStart(action action.Action) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startAction = action
	return s
}

// OnStop is called when runtime request to stop the service is recieved.
func (s *Service) OnStop(action action.WithPrevErr) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopAction = action
	return s
}

// OnTick when set will be called every application tick when service is in running state.
func (s *Service) Tick(action action.Tick) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tickAction = action
	return s
}

// OnTock is called after every tick.
func (s *Service) Tock(action action.Tock) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tockAction = action
	return s
}

// OnEvent is called when a specific event is received.
func (s *Service) OnEvent(scope, key string, cb events.ActionWithEvent[*session.Context]) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listeners == nil {
		s.listeners = make(map[string][]events.ActionWithEvent[*session.Context])
	}

	lid := scope + "." + key
	s.listeners[lid] = append(s.listeners[lid], cb)
	return s
}

// OnAnyEvent called when any event is received.
func (s *Service) OnAnyEvent(cb events.ActionWithEvent[*session.Context]) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listeners == nil {
		s.listeners = make(map[string][]events.ActionWithEvent[*session.Context])
	}
	s.listeners["any"] = append(s.listeners["any"], cb)
	return s
}

// Cron scheduled cron jobs to run when the service is running.
func (s *Service) Cron(setupFunc func(schedule CronScheduler)) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cronsetup = setupFunc
	return s
}

// StopEvent returns the event to stop the service.
func (s *Service) StopEvent() events.Event {
	return StopEvent.Create(s.settings.Slug.String(), nil)
}

// StartEvent returns the event to start the service.
func (s *Service) StartEvent() events.Event {
	return StartEvent.Create(s.settings.Slug.String(), nil)
}
