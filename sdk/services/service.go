// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package services

import (
	"github.com/happy-sdk/happy/pkg/strings/slug"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/services/service"
)

type Service struct {
	settings       service.Settings
	slug           string
	name           string
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
func New(s service.Settings) *Service {
	svc := &Service{}

	_, err := s.Blueprint()
	if err != nil {
		svc.errs = append(svc.errs, err)
	}

	svc.name = s.Name.String()
	svc.slug = slug.Create(s.Name.String())
	return svc
}

func (s *Service) Name() string {
	if s.name == "" {
		return "anonymous-service"
	}
	return s.name
}
func (s *Service) Slug() string {
	return s.slug
}

// OnRegister is called when app is preparing runtime and attaching services,
// This does not mean that service will be used or started.
func (s *Service) OnRegister(action action.Action) {
	s.registerAction = action
}

// OnStart is called when service is requested to be started.
// For instace when command is requiring this service or whenever
// service is required on runtime via sess.RequireService call.
//
// Start can be called multiple times in case of service restarts.
// If you do not want to allow service restarts you should implement
// your logic in OnStop when it's called first time and check that
// state OnStart.
func (s *Service) OnStart(action action.Action) {
	s.startAction = action
}

// OnStop is called when runtime request to stop the service is recieved.
func (s *Service) OnStop(action action.WithPrevErr) {
	s.stopAction = action
}

// OnTick when set will be called every application tick when service is in running state.
func (s *Service) Tick(action action.Tick) {
	s.tickAction = action
}

// OnTock is called after every tick.
func (s *Service) Tock(action action.Tock) {
	s.tockAction = action
}

// OnEvent is called when a specific event is received.
func (s *Service) OnEvent(scope, key string, cb events.ActionWithEvent[*session.Context]) {
	if s.listeners == nil {
		s.listeners = make(map[string][]events.ActionWithEvent[*session.Context])
	}

	lid := scope + "." + key
	s.listeners[lid] = append(s.listeners[lid], cb)
}

// OnAnyEvent called when any event is received.
func (s *Service) OnAnyEvent(cb events.ActionWithEvent[*session.Context]) {
	if s.listeners == nil {
		s.listeners = make(map[string][]events.ActionWithEvent[*session.Context])
	}
	s.listeners["any"] = append(s.listeners["any"], cb)
}

// Cron scheduled cron jobs to run when the service is running.
func (s *Service) Cron(setupFunc func(schedule CronScheduler)) {
	s.cronsetup = setupFunc
}
